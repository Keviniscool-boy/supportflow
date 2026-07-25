package httpapi

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Keviniscool-boy/supportflow/backend/internal/config"
	"github.com/Keviniscool-boy/supportflow/backend/internal/session"
	"github.com/gin-gonic/gin"
)

const requestIDKey = "supportflow.request_id"
const customerCookieName = "sf_demo_customer"

type Server struct {
	engine     *gin.Engine
	config     config.Config
	sessions   session.Store
	csrfSecret []byte
}

func NewServer(config config.Config) *Server {
	return NewServerWithSessionStore(config, session.NewMemoryStore(config.SessionTTL))
}

func NewServerWithSessionStore(config config.Config, store session.Store) *Server {
	secret, err := session.NewSecret()
	if err != nil {
		panic(err)
	}
	if config.Environment == "production-like" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(requestIDMiddleware(), corsMiddleware(config.AllowedOrigin), csrfMiddleware(secret), recoveryMiddleware(), bodyLimitMiddleware(config.MaxBodyBytes), contentTypeMiddleware())
	server := &Server{engine: engine, config: config, sessions: store, csrfSecret: secret}
	server.registerRoutes()
	return server
}

func (s *Server) Handler() http.Handler {
	return s.engine
}

func (s *Server) registerRoutes() {
	s.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "ok"}, "meta": gin.H{"request_id": requestID(c)}})
	})
	s.engine.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "ready"}, "meta": gin.H{"request_id": requestID(c)}})
	})
	s.engine.POST("/api/v1/demo/sessions", s.createSession)
	s.engine.GET("/api/v1/demo/session", s.getSession)
	s.engine.DELETE("/api/v1/demo/session", s.revokeSession)
	s.engine.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, "RESOURCE_NOT_FOUND", "errors.resource_not_found", false, nil)
	})
}

func (s *Server) createSession(c *gin.Context) {
	rawToken, cookieErr := c.Cookie(customerCookieName)
	if cookieErr == nil && rawToken != "" {
		current, err := s.sessions.Get(c.Request.Context(), rawToken)
		if err == nil {
			setCustomerCookie(c, rawToken, current.ExpiresAt, s.config.SecureCookies)
			writeSessionResponse(c, http.StatusOK, current, session.CSRFToken(s.csrfSecret, rawToken))
			return
		}
		if !isSessionLifecycleError(err) {
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", false, nil)
			return
		}
		clearCustomerCookie(c, s.config.SecureCookies)
	}
	rawToken, err := session.NewToken()
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", false, nil)
		return
	}
	locale := c.GetHeader("Accept-Language")
	created, _, err := s.sessions.CreateOrGet(c.Request.Context(), rawToken, locale)
	if err != nil {
		writeSessionError(c, err)
		return
	}
	setCustomerCookie(c, rawToken, created.ExpiresAt, s.config.SecureCookies)
	writeSessionResponse(c, http.StatusCreated, created, session.CSRFToken(s.csrfSecret, rawToken))
}

func (s *Server) getSession(c *gin.Context) {
	rawToken, current, ok := s.requireSession(c)
	if !ok {
		return
	}
	writeSessionResponse(c, http.StatusOK, current, session.CSRFToken(s.csrfSecret, rawToken))
}

func (s *Server) revokeSession(c *gin.Context) {
	rawToken, _, ok := s.requireSession(c)
	if !ok {
		return
	}
	if err := s.sessions.Revoke(c.Request.Context(), rawToken); err != nil {
		writeSessionError(c, err)
		return
	}
	clearCustomerCookie(c, s.config.SecureCookies)
	c.JSON(http.StatusAccepted, gin.H{"data": gin.H{"status": "REVOKED"}, "meta": gin.H{"request_id": requestID(c)}})
}

func (s *Server) requireSession(c *gin.Context) (string, session.Session, bool) {
	rawToken, err := c.Cookie(customerCookieName)
	if err != nil || rawToken == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "errors.unauthenticated", false, nil)
		return "", session.Session{}, false
	}
	current, err := s.sessions.Get(c.Request.Context(), rawToken)
	if err != nil {
		if errors.Is(err, session.ErrExpired) || errors.Is(err, session.ErrRevoked) {
			clearCustomerCookie(c, s.config.SecureCookies)
		}
		writeSessionError(c, err)
		return "", session.Session{}, false
	}
	return rawToken, current, true
}

func writeSessionResponse(c *gin.Context, status int, value session.Session, csrfToken string) {
	c.JSON(status, gin.H{"data": gin.H{
		"session": gin.H{
			"id":              value.ID,
			"status":          value.Status,
			"expires_at":      value.ExpiresAt.Format(time.RFC3339),
			"data_generation": value.DataGeneration,
			"quota": gin.H{
				"requests_remaining":     maxInt(value.RequestLimit-value.RequestCount, 0),
				"tokens_remaining":       maxInt64(value.TokenLimit-value.TokenCount, 0),
				"upload_files_remaining": maxInt(value.UploadLimit-value.UploadCount, 0),
			},
		},
		"customer":   gin.H{"id": value.CustomerID, "display_name": value.DisplayName, "locale": value.Locale},
		"csrf_token": csrfToken,
	}, "meta": gin.H{"request_id": requestID(c)}})
}

func writeSessionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, session.ErrNotFound):
		writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "errors.unauthenticated", false, nil)
	case errors.Is(err, session.ErrExpired), errors.Is(err, session.ErrRevoked):
		writeError(c, http.StatusUnauthorized, "SESSION_EXPIRED", "errors.session_expired", false, nil)
	case errors.Is(err, session.ErrResetting):
		writeError(c, http.StatusConflict, "DEMO_RESET_IN_PROGRESS", "errors.demo_reset_in_progress", false, nil)
	default:
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", false, nil)
	}
}

func isSessionLifecycleError(err error) bool {
	return errors.Is(err, session.ErrNotFound) || errors.Is(err, session.ErrExpired) || errors.Is(err, session.ErrRevoked) || errors.Is(err, session.ErrResetting)
}

func setCustomerCookie(c *gin.Context, rawToken string, expiresAt time.Time, secure bool) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: customerCookieName, Value: rawToken, Path: "/", MaxAge: maxAge, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, Expires: expiresAt})
}

func clearCustomerCookie(c *gin.Context, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{Name: customerCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
}

func csrfMiddleware(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions || !strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Next()
			return
		}
		rawToken, cookieErr := c.Cookie(customerCookieName)
		if cookieErr != nil || rawToken == "" {
			if c.Request.URL.Path == "/api/v1/demo/sessions" {
				c.Next()
				return
			}
			c.Next()
			return
		}
		expected := session.CSRFToken(secret, rawToken)
		provided := strings.TrimSpace(c.GetHeader("X-CSRF-Token"))
		if len(expected) != len(provided) || subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
			writeError(c, http.StatusForbidden, "CSRF_VALIDATION_FAILED", "errors.csrf_validation_failed", false, nil)
			return
		}
		c.Next()
	}
}

func corsMiddleware(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" && allowedOrigin != "" && origin != allowedOrigin {
			writeError(c, http.StatusForbidden, "PERMISSION_DENIED", "errors.permission_denied", false, nil)
			return
		}
		if origin != "" && allowedOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, X-Request-ID, X-CSRF-Token, Idempotency-Key, If-Match")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

func maxInt(value, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}

func maxInt64(value, minimum int64) int64 {
	if value < minimum {
		return minimum
	}
	return value
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if !validRequestID(requestID) {
			requestID = newRequestID()
		}
		c.Set(requestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func validRequestID(value string) bool {
	if value == "" || len(value) > 100 {
		return false
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' ||
			strings.ContainsRune("._:-", character) {
			continue
		}
		return false
	}
	return true
}

func bodyLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "errors.payload_too_large", false, nil)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

func recoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recover() != nil {
				writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", false, nil)
			}
		}()
		c.Next()
	}
}

func contentTypeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Next()
	}
}

func requestID(c *gin.Context) string {
	value, _ := c.Get(requestIDKey)
	requestIDValue, _ := value.(string)
	return requestIDValue
}

func newRequestID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "request-unknown"
	}
	return hex.EncodeToString(buffer)
}
