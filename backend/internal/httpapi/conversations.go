package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Keviniscool-boy/supportflow/backend/internal/conversation"
	"github.com/Keviniscool-boy/supportflow/backend/internal/identity"
	"github.com/gin-gonic/gin"
)

type createConversationRequest struct {
	Subject string `json:"subject"`
}

func (s *Server) registerConversationRoutes(routes *gin.RouterGroup) {
	routes.POST("/conversations", s.createConversation)
	routes.GET("/conversations", s.listConversations)
	routes.GET("/conversations/:conversation_id", s.getConversation)
	routes.GET("/conversations/:conversation_id/messages", s.listConversationMessages)
}

func (s *Server) createConversation(c *gin.Context) {
	var request createConversationRequest
	if !decodeOptionalJSON(c, &request) {
		return
	}
	customer, ok := identity.CustomerFromContext(c.Request.Context())
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "errors.unauthenticated", false, nil)
		return
	}
	created, err := s.conversations.Create(c.Request.Context(), customer, request.Subject)
	if err != nil {
		writeConversationError(c, err)
		return
	}
	c.Header("ETag", resourceETag(created.RowVersion))
	c.JSON(http.StatusCreated, gin.H{"data": conversationResponse(created), "meta": gin.H{"request_id": requestID(c)}})
}

func (s *Server) listConversations(c *gin.Context) {
	if !onlyQueryFields(c, "limit", "cursor") {
		return
	}
	limit, ok := queryInteger(c, "limit", 20, 1, 100)
	if !ok {
		return
	}
	customer, authenticated := identity.CustomerFromContext(c.Request.Context())
	if !authenticated {
		writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "errors.unauthenticated", false, nil)
		return
	}
	options := conversation.ConversationListOptions{Limit: limit}
	if encoded := strings.TrimSpace(c.Query("cursor")); encoded != "" {
		cursor, err := decodeConversationCursor(s.csrfSecret, customer, encoded)
		if err != nil {
			writeError(c, http.StatusBadRequest, "INVALID_CURSOR", "errors.invalid_cursor", false, nil)
			return
		}
		options.Before = cursor
	}
	page, err := s.conversations.List(c.Request.Context(), customer, options)
	if err != nil {
		writeConversationError(c, err)
		return
	}
	items := make([]gin.H, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, conversationResponse(item))
	}
	nextCursor, err := encodeConversationCursor(s.csrfSecret, customer, page.Next)
	if err != nil {
		writeConversationError(c, err)
		return
	}
	var cursorValue any
	if nextCursor != "" {
		cursorValue = nextCursor
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"request_id": requestID(c), "next_cursor": cursorValue, "has_more": page.Next != nil}})
}

func (s *Server) getConversation(c *gin.Context) {
	if !onlyQueryFields(c) {
		return
	}
	customer, _ := identity.CustomerFromContext(c.Request.Context())
	value, err := s.conversations.Get(c.Request.Context(), customer, c.Param("conversation_id"))
	if err != nil {
		writeConversationError(c, err)
		return
	}
	c.Header("ETag", resourceETag(value.RowVersion))
	c.JSON(http.StatusOK, gin.H{"data": conversationResponse(value), "meta": gin.H{"request_id": requestID(c)}})
}

func (s *Server) listConversationMessages(c *gin.Context) {
	if !onlyQueryFields(c, "after_sequence", "limit") {
		return
	}
	afterSequence, ok := queryInteger(c, "after_sequence", 0, 0, int(^uint(0)>>1))
	if !ok {
		return
	}
	limit, ok := queryInteger(c, "limit", 100, 1, 200)
	if !ok {
		return
	}
	customer, _ := identity.CustomerFromContext(c.Request.Context())
	page, err := s.conversations.ListMessages(c.Request.Context(), customer, c.Param("conversation_id"), int64(afterSequence), limit)
	if err != nil {
		writeConversationError(c, err)
		return
	}
	items := make([]gin.H, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, messageResponse(item, customer))
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{
		"request_id": requestID(c), "next_after_sequence": page.NextAfterSequence, "has_more": page.HasMore,
	}})
}

func conversationResponse(value conversation.Conversation) gin.H {
	return gin.H{
		"id": value.ID, "conversation_number": value.Number, "status": value.Status, "response_owner": value.ResponseOwner,
		"subject": value.Subject, "active_run": nil, "active_handoff": nil, "last_message_at": optionalTime(value.LastMessageAt),
		"created_at": value.CreatedAt.UTC().Format(time.RFC3339Nano), "updated_at": value.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func messageResponse(value conversation.Message, customer identity.CustomerContext) gin.H {
	displayName := "Nova"
	switch value.ActorType {
	case conversation.ActorCustomer:
		displayName = customer.DisplayName
	case conversation.ActorMember:
		displayName = "Support"
	case conversation.ActorSystem:
		displayName = "SupportFlow"
	}
	return gin.H{
		"id": value.ID, "conversation_id": value.ConversationID, "sequence_no": value.SequenceNo,
		"actor":   gin.H{"type": value.ActorType, "display_name": displayName},
		"content": gin.H{"type": value.ContentType, "text": value.ContentText}, "citations": []any{},
		"created_at": value.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func writeConversationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, conversation.ErrNotFound):
		writeError(c, http.StatusNotFound, "RESOURCE_NOT_FOUND", "errors.resource_not_found", false, nil)
	case errors.Is(err, conversation.ErrClosed):
		writeError(c, http.StatusConflict, "INVALID_STATE_TRANSITION", "errors.invalid_state_transition", false, nil)
	case errors.Is(err, conversation.ErrInvalidSubject):
		writeError(c, http.StatusUnprocessableEntity, "INVALID_ARGUMENT", "errors.invalid_argument", false, []ErrorDetail{{Field: "subject", Code: "INVALID"}})
	case errors.Is(err, conversation.ErrInvalidMessage):
		writeError(c, http.StatusUnprocessableEntity, "INVALID_ARGUMENT", "errors.invalid_argument", false, nil)
	default:
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", false, nil)
	}
}

func decodeOptionalJSON(c *gin.Context, destination any) bool {
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		return true
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		if strings.Contains(err.Error(), "unknown field") {
			writeError(c, http.StatusBadRequest, "UNKNOWN_FIELD", "errors.unknown_field", false, nil)
		} else {
			writeError(c, http.StatusBadRequest, "INVALID_JSON", "errors.invalid_json", false, nil)
		}
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "errors.invalid_json", false, nil)
		return false
	}
	return true
}

func onlyQueryFields(c *gin.Context, allowed ...string) bool {
	allowedFields := make(map[string]struct{}, len(allowed))
	for _, field := range allowed {
		allowedFields[field] = struct{}{}
	}
	for field, values := range c.Request.URL.Query() {
		if _, ok := allowedFields[field]; !ok || len(values) != 1 {
			writeError(c, http.StatusUnprocessableEntity, "INVALID_ARGUMENT", "errors.invalid_argument", false, []ErrorDetail{{Field: field, Code: "UNKNOWN_OR_REPEATED"}})
			return false
		}
	}
	return true
}

func queryInteger(c *gin.Context, field string, fallback, minimum, maximum int) (int, bool) {
	raw := strings.TrimSpace(c.Query(field))
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		writeError(c, http.StatusUnprocessableEntity, "INVALID_ARGUMENT", "errors.invalid_argument", false, []ErrorDetail{{Field: field, Code: "OUT_OF_RANGE"}})
		return 0, false
	}
	return value, true
}

func optionalTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func resourceETag(rowVersion int64) string {
	return `"rv-` + strconv.FormatInt(rowVersion, 10) + `"`
}
