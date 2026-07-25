package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Keviniscool-boy/supportflow/backend/internal/config"
	"github.com/gin-gonic/gin"
)

func TestHealthIncludesRequestID(t *testing.T) {
	server := NewServer(config.Config{Environment: "development", MaxBodyBytes: 1024})
	recording := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("X-Request-ID", "test-request")
	server.Handler().ServeHTTP(recording, request)
	if recording.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recording.Code)
	}
	if recording.Header().Get("X-Request-ID") != "test-request" {
		t.Fatalf("expected request id header, got %q", recording.Header().Get("X-Request-ID"))
	}
	if !strings.Contains(recording.Body.String(), "test-request") {
		t.Fatalf("expected request id in response, got %s", recording.Body.String())
	}
}

func TestUnknownRouteUsesErrorEnvelope(t *testing.T) {
	server := NewServer(config.Config{Environment: "development", MaxBodyBytes: 1024})
	recording := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	server.Handler().ServeHTTP(recording, request)
	if recording.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recording.Code)
	}
	if !strings.Contains(recording.Body.String(), "RESOURCE_NOT_FOUND") {
		t.Fatalf("expected error code, got %s", recording.Body.String())
	}
}

func TestInvalidRequestIDIsReplaced(t *testing.T) {
	server := NewServer(config.Config{Environment: "development", MaxBodyBytes: 1024})
	recording := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("X-Request-ID", "invalid request id")
	server.Handler().ServeHTTP(recording, request)
	if recording.Header().Get("X-Request-ID") == "invalid request id" {
		t.Fatal("expected invalid request id to be replaced")
	}
}

func TestDemoSessionCookieAndCSRF(t *testing.T) {
	server := NewServer(config.Config{Environment: "development", MaxBodyBytes: 1024, SessionTTL: time.Hour})
	createRecorder := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/demo/sessions", nil)
	server.Handler().ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	cookies := createRecorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != customerCookieName || !cookies[0].HttpOnly {
		t.Fatalf("expected HttpOnly customer cookie, got %#v", cookies)
	}
	var response map[string]any
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	data := response["data"].(map[string]any)
	csrfToken := data["csrf_token"].(string)

	getRecorder := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/api/v1/demo/session", nil)
	getRequest.AddCookie(cookies[0])
	server.Handler().ServeHTTP(getRecorder, getRequest)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected session GET 200, got %d: %s", getRecorder.Code, getRecorder.Body.String())
	}

	deniedRecorder := httptest.NewRecorder()
	deniedRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/demo/session", nil)
	deniedRequest.AddCookie(cookies[0])
	server.Handler().ServeHTTP(deniedRecorder, deniedRequest)
	if deniedRecorder.Code != http.StatusForbidden || !strings.Contains(deniedRecorder.Body.String(), "CSRF_VALIDATION_FAILED") {
		t.Fatalf("expected CSRF rejection, got %d: %s", deniedRecorder.Code, deniedRecorder.Body.String())
	}

	revokeRecorder := httptest.NewRecorder()
	revokeRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/demo/session", nil)
	revokeRequest.AddCookie(cookies[0])
	revokeRequest.Header.Set("X-CSRF-Token", csrfToken)
	server.Handler().ServeHTTP(revokeRecorder, revokeRequest)
	if revokeRecorder.Code != http.StatusAccepted {
		t.Fatalf("expected revoke 202, got %d: %s", revokeRecorder.Code, revokeRecorder.Body.String())
	}

	afterRecorder := httptest.NewRecorder()
	afterRequest := httptest.NewRequest(http.MethodGet, "/api/v1/demo/session", nil)
	afterRequest.AddCookie(cookies[0])
	server.Handler().ServeHTTP(afterRecorder, afterRequest)
	if afterRecorder.Code != http.StatusUnauthorized || !strings.Contains(afterRecorder.Body.String(), "SESSION_EXPIRED") {
		t.Fatalf("expected revoked session rejection, got %d: %s", afterRecorder.Code, afterRecorder.Body.String())
	}
}

func TestAllowedOriginCORS(t *testing.T) {
	server := NewServer(config.Config{Environment: "development", MaxBodyBytes: 1024, AllowedOrigin: "http://localhost:3000"})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/demo/sessions", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected preflight 204, got %d", recorder.Code)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("expected allowed origin, got %q", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestBodyLimitReturnsErrorEnvelope(t *testing.T) {
	server := NewServer(config.Config{Environment: "development", MaxBodyBytes: 4})
	recording := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/unknown", strings.NewReader("12345"))
	server.Handler().ServeHTTP(recording, request)
	if recording.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", recording.Code)
	}
	if !strings.Contains(recording.Body.String(), "PAYLOAD_TOO_LARGE") {
		t.Fatalf("expected payload error code, got %s", recording.Body.String())
	}
}

func TestRecoveryUsesErrorEnvelope(t *testing.T) {
	server := NewServer(config.Config{Environment: "development", MaxBodyBytes: 1024})
	server.engine.GET("/panic", func(_ *gin.Context) {
		panic("sensitive internal value")
	})
	recording := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	server.Handler().ServeHTTP(recording, request)
	if recording.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recording.Code)
	}
	if !strings.Contains(recording.Body.String(), "INTERNAL_ERROR") {
		t.Fatalf("expected internal error code, got %s", recording.Body.String())
	}
	if strings.Contains(recording.Body.String(), "sensitive internal value") {
		t.Fatalf("response leaked panic value: %s", recording.Body.String())
	}
}
