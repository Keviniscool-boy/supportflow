package conversation

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	appclock "github.com/Keviniscool-boy/supportflow/backend/internal/clock"
	"github.com/Keviniscool-boy/supportflow/backend/internal/identity"
	"github.com/Keviniscool-boy/supportflow/backend/internal/testfixture"
)

func TestServiceScopesAndPaginatesConversations(t *testing.T) {
	clock := appclock.NewFixed(testfixture.FrozenTime)
	service := NewService(NewMemoryRepository(), clock)
	customer := fixtureCustomer()
	for index := 0; index < 3; index++ {
		if _, err := service.Create(context.Background(), customer, "耳机问题 test@example.com"); err != nil {
			t.Fatal(err)
		}
		clock.Advance(time.Second)
	}
	page, err := service.List(context.Background(), customer, ConversationListOptions{Limit: 2})
	if err != nil || len(page.Items) != 2 || page.Next == nil {
		t.Fatalf("unexpected first page: %#v err=%v", page, err)
	}
	if page.Items[0].Subject == nil || strings.Contains(*page.Items[0].Subject, "test@example.com") {
		t.Fatalf("subject was not redacted: %#v", page.Items[0].Subject)
	}
	second, err := service.List(context.Background(), customer, ConversationListOptions{Limit: 2, Before: page.Next})
	if err != nil || len(second.Items) != 1 || second.Next != nil {
		t.Fatalf("unexpected second page: %#v err=%v", second, err)
	}

	other := customer
	other.CustomerID = "00000000-0000-0000-0000-000000000099"
	if _, err := service.Get(context.Background(), other, page.Items[0].ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected cross-customer lookup rejection, got %v", err)
	}
}

func TestServiceAppendsRedactedMessagesInSequence(t *testing.T) {
	clock := appclock.NewFixed(testfixture.FrozenTime)
	service := NewService(NewMemoryRepository(), clock)
	customer := fixtureCustomer()
	conversation, err := service.Create(context.Background(), customer, "耳机问题")
	if err != nil {
		t.Fatal(err)
	}
	for index, text := range []string{"手机号 13800138000", "订单 SF20260001", "仍然没有声音"} {
		if _, err := service.AppendMessage(context.Background(), customer, conversation.ID, NewMessage{ActorType: ActorCustomer, ContentType: ContentText, Text: text, Locale: "zh-CN"}); err != nil {
			t.Fatal(err)
		}
		clock.Advance(time.Second)
		_ = index
	}
	first, err := service.ListMessages(context.Background(), customer, conversation.ID, 0, 2)
	if err != nil || len(first.Items) != 2 || !first.HasMore || first.NextAfterSequence != 2 {
		t.Fatalf("unexpected message page: %#v err=%v", first, err)
	}
	if strings.Contains(first.Items[0].ContentText, "13800138000") || strings.Contains(first.Items[1].ContentText, "SF20260001") {
		t.Fatalf("messages were not redacted: %#v", first.Items)
	}
	second, err := service.ListMessages(context.Background(), customer, conversation.ID, first.NextAfterSequence, 2)
	if err != nil || len(second.Items) != 1 || second.HasMore || second.Items[0].SequenceNo != 3 {
		t.Fatalf("unexpected resumed page: %#v err=%v", second, err)
	}
}

func fixtureCustomer() identity.CustomerContext {
	fixture := testfixture.Standard()
	return identity.CustomerContext{TenantID: fixture.Tenant.ID, SessionID: fixture.Session.ID, CustomerID: fixture.Customer.ID, DisplayName: fixture.Customer.DisplayName, Locale: fixture.Customer.Locale, DataGeneration: 1}
}
