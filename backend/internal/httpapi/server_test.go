package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Keviniscool-boy/supportflow/backend/internal/config"
	"github.com/Keviniscool-boy/supportflow/backend/internal/identity"
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

func TestDemoSessionUsesValidatedLocaleBody(t *testing.T) {
	server := NewServer(config.Config{Environment: "development", MaxBodyBytes: 1024, SessionTTL: time.Hour})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/demo/sessions", strings.NewReader(`{"locale":"en-US"}`))
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated || !strings.Contains(recorder.Body.String(), `"locale":"en-US"`) {
		t.Fatalf("expected validated locale, got %d: %s", recorder.Code, recorder.Body.String())
	}

	invalidRecorder := httptest.NewRecorder()
	invalidRequest := httptest.NewRequest(http.MethodPost, "/api/v1/demo/sessions", strings.NewReader(`{"locale":"fr-FR"}`))
	server.Handler().ServeHTTP(invalidRecorder, invalidRequest)
	if invalidRecorder.Code != http.StatusUnprocessableEntity || !strings.Contains(invalidRecorder.Body.String(), "UNSUPPORTED_LOCALE") {
		t.Fatalf("expected locale validation error, got %d: %s", invalidRecorder.Code, invalidRecorder.Body.String())
	}
}

func TestCustomerContextComesOnlyFromValidatedSession(t *testing.T) {
	server := NewServer(config.Config{Environment: "development", MaxBodyBytes: 1024, SessionTTL: time.Hour})
	server.engine.GET("/customer-context-test", server.customerContextMiddleware(), func(c *gin.Context) {
		customer, ok := identity.CustomerFromContext(c.Request.Context())
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, gin.H{"tenant_id": customer.TenantID, "customer_id": customer.CustomerID, "session_id": customer.SessionID})
	})

	unauthenticated := httptest.NewRecorder()
	server.Handler().ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/customer-context-test", nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing session rejection, got %d", unauthenticated.Code)
	}

	createRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRecorder, httptest.NewRequest(http.MethodPost, "/api/v1/demo/sessions", nil))
	cookie := createRecorder.Result().Cookies()[0]

	first := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodGet, "/customer-context-test?customer_id=attacker-controlled", nil)
	firstRequest.AddCookie(cookie)
	server.Handler().ServeHTTP(first, firstRequest)
	if first.Code != http.StatusOK {
		t.Fatalf("expected authenticated context, got %d: %s", first.Code, first.Body.String())
	}
	var firstContext map[string]string
	if err := json.Unmarshal(first.Body.Bytes(), &firstContext); err != nil {
		t.Fatal(err)
	}
	if firstContext["customer_id"] == "attacker-controlled" || firstContext["customer_id"] == "" || firstContext["tenant_id"] == "" {
		t.Fatalf("customer context accepted untrusted identity: %#v", firstContext)
	}

	second := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodGet, "/customer-context-test", nil)
	secondRequest.AddCookie(cookie)
	server.Handler().ServeHTTP(second, secondRequest)
	if second.Body.String() != first.Body.String() {
		t.Fatalf("same session must inject same customer context: %s != %s", first.Body.String(), second.Body.String())
	}
}

func TestTraceparentIsAcceptedWithoutCollector(t *testing.T) {
	server := NewServer(config.Config{Environment: "development", MaxBodyBytes: 1024})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected health response, got %d", recorder.Code)
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
