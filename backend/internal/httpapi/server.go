package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/Keviniscool-boy/supportflow/backend/internal/config"
	"github.com/gin-gonic/gin"
)

const requestIDKey = "supportflow.request_id"

type Server struct {
	engine *gin.Engine
}

func NewServer(config config.Config) *Server {
	if config.Environment == "production-like" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(requestIDMiddleware(), recoveryMiddleware(), bodyLimitMiddleware(config.MaxBodyBytes), contentTypeMiddleware())
	server := &Server{engine: engine}
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
	s.engine.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, "RESOURCE_NOT_FOUND", "errors.resource_not_found", false, nil)
	})
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
