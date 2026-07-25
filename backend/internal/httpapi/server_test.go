package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
