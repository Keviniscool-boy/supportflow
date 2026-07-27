package httpapi

import (
	"testing"
	"time"

	"github.com/Keviniscool-boy/supportflow/backend/internal/conversation"
	"github.com/Keviniscool-boy/supportflow/backend/internal/identity"
)

func TestConversationCursorIsSignedAndBoundToCustomer(t *testing.T) {
	secret := []byte("test-secret")
	customer := identity.CustomerContext{TenantID: "tenant", CustomerID: "customer"}
	want := &conversation.ConversationCursor{SortTime: time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC), ID: "00000000-0000-4000-8000-000000000001"}
	encoded, err := encodeConversationCursor(secret, customer, want)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeConversationCursor(secret, customer, encoded)
	if err != nil || decoded.ID != want.ID || !decoded.SortTime.Equal(want.SortTime) {
		t.Fatalf("cursor roundtrip failed: %#v err=%v", decoded, err)
	}
	other := customer
	other.CustomerID = "other"
	if _, err := decodeConversationCursor(secret, other, encoded); err == nil {
		t.Fatal("cursor must not cross customer scope")
	}
	if _, err := decodeConversationCursor(secret, customer, encoded+"x"); err == nil {
		t.Fatal("tampered cursor must be rejected")
	}
}
