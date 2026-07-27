package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Keviniscool-boy/supportflow/backend/internal/config"
	"github.com/Keviniscool-boy/supportflow/backend/internal/conversation"
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

func TestCustomerConversationLifecycleAndIsolation(t *testing.T) {
	server := NewServer(config.Config{Environment: "development", MaxBodyBytes: 4096, SessionTTL: time.Hour})
	firstCookie, firstCSRF, firstSession := createTestSession(t, server)

	missingCSRF := httptest.NewRecorder()
	missingCSRFRequest := httptest.NewRequest(http.MethodPost, "/api/v1/customer/conversations", strings.NewReader(`{"subject":"耳机问题"}`))
	missingCSRFRequest.AddCookie(firstCookie)
	server.Handler().ServeHTTP(missingCSRF, missingCSRFRequest)
	if missingCSRF.Code != http.StatusForbidden {
		t.Fatalf("expected CSRF rejection, got %d: %s", missingCSRF.Code, missingCSRF.Body.String())
	}

	conversationIDs := make([]string, 0, 3)
	for index := 0; index < 3; index++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/customer/conversations", strings.NewReader(`{"subject":"耳机问题 test@example.com"}`))
		request.AddCookie(firstCookie)
		request.Header.Set("X-CSRF-Token", firstCSRF)
		server.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusCreated || recorder.Header().Get("ETag") == "" || strings.Contains(recorder.Body.String(), "test@example.com") {
			t.Fatalf("unexpected create response %d: %s", recorder.Code, recorder.Body.String())
		}
		var response struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		conversationIDs = append(conversationIDs, response.Data.ID)
	}

	current, err := server.sessions.Get(context.Background(), firstCookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	customer := identity.CustomerContext{
		TenantID: current.TenantID, SessionID: current.ID, CustomerID: current.CustomerID, DisplayName: current.DisplayName, Locale: current.Locale, DataGeneration: current.DataGeneration,
	}
	for _, text := range []string{"手机号 13800138000", "订单 SF20260001", "仍然没有声音"} {
		if _, err := server.conversations.AppendMessage(context.Background(), customer, conversationIDs[0], conversation.NewMessage{ActorType: conversation.ActorCustomer, ContentType: conversation.ContentText, Text: text, Locale: "zh-CN"}); err != nil {
			t.Fatal(err)
		}
	}

	listRecorder := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/customer/conversations?limit=2", nil)
	listRequest.AddCookie(firstCookie)
	server.Handler().ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected list response, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}
	var listResponse struct {
		Data []map[string]any `json:"data"`
		Meta struct {
			NextCursor string `json:"next_cursor"`
			HasMore    bool   `json:"has_more"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &listResponse); err != nil {
		t.Fatal(err)
	}
	if len(listResponse.Data) != 2 || !listResponse.Meta.HasMore || listResponse.Meta.NextCursor == "" {
		t.Fatalf("unexpected conversation page: %s", listRecorder.Body.String())
	}

	tamperedRecorder := httptest.NewRecorder()
	tamperedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/customer/conversations?cursor="+listResponse.Meta.NextCursor+"x", nil)
	tamperedRequest.AddCookie(firstCookie)
	server.Handler().ServeHTTP(tamperedRecorder, tamperedRequest)
	if tamperedRecorder.Code != http.StatusBadRequest || !strings.Contains(tamperedRecorder.Body.String(), "INVALID_CURSOR") {
		t.Fatalf("expected cursor rejection, got %d: %s", tamperedRecorder.Code, tamperedRecorder.Body.String())
	}

	messagesRecorder := httptest.NewRecorder()
	messagesRequest := httptest.NewRequest(http.MethodGet, "/api/v1/customer/conversations/"+conversationIDs[0]+"/messages?limit=2", nil)
	messagesRequest.AddCookie(firstCookie)
	server.Handler().ServeHTTP(messagesRecorder, messagesRequest)
	if messagesRecorder.Code != http.StatusOK || strings.Contains(messagesRecorder.Body.String(), "13800138000") || strings.Contains(messagesRecorder.Body.String(), "SF20260001") {
		t.Fatalf("unexpected message response %d: %s", messagesRecorder.Code, messagesRecorder.Body.String())
	}
	if !strings.Contains(messagesRecorder.Body.String(), `"next_after_sequence":2`) || !strings.Contains(messagesRecorder.Body.String(), `"has_more":true`) {
		t.Fatalf("message resume metadata missing: %s", messagesRecorder.Body.String())
	}

	secondCookie, _, _ := createTestSession(t, server)
	foreignRecorder := httptest.NewRecorder()
	foreignRequest := httptest.NewRequest(http.MethodGet, "/api/v1/customer/conversations/"+conversationIDs[0], nil)
	foreignRequest.AddCookie(secondCookie)
	server.Handler().ServeHTTP(foreignRecorder, foreignRequest)
	if foreignRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected cross-customer 404, got %d: %s", foreignRecorder.Code, foreignRecorder.Body.String())
	}

	if firstSession == "" {
		t.Fatal("test session id was empty")
	}
}

func TestCreateConversationRejectsUnknownIdentityFields(t *testing.T) {
	server := NewServer(config.Config{Environment: "development", MaxBodyBytes: 4096, SessionTTL: time.Hour})
	cookie, csrfToken, _ := createTestSession(t, server)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/customer/conversations", strings.NewReader(`{"subject":"耳机问题","customer_id":"attacker"}`))
	request.AddCookie(cookie)
	request.Header.Set("X-CSRF-Token", csrfToken)
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "UNKNOWN_FIELD") {
		t.Fatalf("expected unknown identity field rejection, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func createTestSession(t *testing.T, server *Server) (*http.Cookie, string, string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/demo/sessions", nil))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected session creation, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			CSRFToken string `json:"csrf_token"`
			Session   struct {
				ID string `json:"id"`
			} `json:"session"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return recorder.Result().Cookies()[0], response.Data.CSRFToken, response.Data.Session.ID
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
