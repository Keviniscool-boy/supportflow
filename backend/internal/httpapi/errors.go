package httpapi

import "github.com/gin-gonic/gin"

type ErrorDetail struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

type APIError struct {
	Code       string         `json:"code"`
	MessageKey string         `json:"message_key"`
	Params     map[string]any `json:"params,omitempty"`
	Retryable  bool           `json:"retryable"`
	Details    []ErrorDetail  `json:"details,omitempty"`
	RequestID  string         `json:"request_id"`
}

func writeError(c *gin.Context, status int, code, messageKey string, retryable bool, details []ErrorDetail) {
	requestID, _ := c.Get(requestIDKey)
	requestIDValue, _ := requestID.(string)
	c.AbortWithStatusJSON(status, gin.H{"error": APIError{
		Code:       code,
		MessageKey: messageKey,
		Retryable:  retryable,
		Details:    details,
		RequestID:  requestIDValue,
	}})
}
