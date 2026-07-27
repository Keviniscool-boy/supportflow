package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Keviniscool-boy/supportflow/backend/internal/conversation"
	"github.com/Keviniscool-boy/supportflow/backend/internal/identity"
)

var errInvalidCursor = errors.New("invalid cursor")

type conversationCursorPayload struct {
	TenantID   string `json:"tenant_id"`
	CustomerID string `json:"customer_id"`
	SortTime   string `json:"sort_time"`
	ID         string `json:"id"`
}

func encodeConversationCursor(secret []byte, customer identity.CustomerContext, cursor *conversation.ConversationCursor) (string, error) {
	if cursor == nil {
		return "", nil
	}
	payload, err := json.Marshal(conversationCursorPayload{
		TenantID: customer.TenantID, CustomerID: customer.CustomerID, SortTime: cursor.SortTime.UTC().Format(time.RFC3339Nano), ID: cursor.ID,
	})
	if err != nil {
		return "", err
	}
	signature := signCursor(secret, payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func decodeConversationCursor(secret []byte, customer identity.CustomerContext, encoded string) (*conversation.ConversationCursor, error) {
	parts := strings.Split(encoded, ".")
	if len(parts) != 2 {
		return nil, errInvalidCursor
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errInvalidCursor
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(signature, signCursor(secret, payload)) {
		return nil, errInvalidCursor
	}
	var value conversationCursorPayload
	if err := json.Unmarshal(payload, &value); err != nil || value.TenantID != customer.TenantID || value.CustomerID != customer.CustomerID || value.ID == "" {
		return nil, errInvalidCursor
	}
	sortTime, err := time.Parse(time.RFC3339Nano, value.SortTime)
	if err != nil {
		return nil, errInvalidCursor
	}
	return &conversation.ConversationCursor{SortTime: sortTime.UTC(), ID: value.ID}, nil
}

func signCursor(secret, payload []byte) []byte {
	hash := hmac.New(sha256.New, secret)
	_, _ = hash.Write([]byte("supportflow:conversation-cursor:v1:"))
	_, _ = hash.Write(payload)
	return hash.Sum(nil)
}
