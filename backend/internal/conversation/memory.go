package conversation

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/Keviniscool-boy/supportflow/backend/internal/identity"
)

type MemoryRepository struct {
	mu            sync.RWMutex
	conversations map[string]Conversation
	messages      map[string][]Message
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{conversations: make(map[string]Conversation), messages: make(map[string][]Message)}
}

func (r *MemoryRepository) Create(_ context.Context, value Conversation) (Conversation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conversations[objectKey(value.TenantID, value.ID)] = value
	return value, nil
}

func (r *MemoryRepository) List(_ context.Context, customer identity.CustomerContext, options ConversationListOptions) (ConversationPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]Conversation, 0)
	for _, value := range r.conversations {
		if value.TenantID != customer.TenantID || value.CustomerID != customer.CustomerID {
			continue
		}
		if options.Before != nil && !isBefore(value.SortTime(), value.ID, options.Before.SortTime, options.Before.ID) {
			continue
		}
		items = append(items, value)
	}
	sort.Slice(items, func(first, second int) bool {
		firstTime, secondTime := items[first].SortTime(), items[second].SortTime()
		if firstTime.Equal(secondTime) {
			return items[first].ID > items[second].ID
		}
		return firstTime.After(secondTime)
	})
	page := ConversationPage{Items: items}
	if len(page.Items) > options.Limit {
		page.Items = page.Items[:options.Limit]
		last := page.Items[len(page.Items)-1]
		page.Next = &ConversationCursor{SortTime: last.SortTime(), ID: last.ID}
	}
	return page, nil
}

func (r *MemoryRepository) Get(_ context.Context, customer identity.CustomerContext, conversationID string) (Conversation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.conversations[objectKey(customer.TenantID, conversationID)]
	if !ok || value.CustomerID != customer.CustomerID {
		return Conversation{}, ErrNotFound
	}
	return value, nil
}

func (r *MemoryRepository) AppendMessage(_ context.Context, customer identity.CustomerContext, conversationID string, message Message) (Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := objectKey(customer.TenantID, conversationID)
	value, ok := r.conversations[key]
	if !ok || value.CustomerID != customer.CustomerID {
		return Message{}, ErrNotFound
	}
	if value.Status != "OPEN" {
		return Message{}, ErrClosed
	}
	messages := r.messages[key]
	message.SequenceNo = int64(len(messages) + 1)
	r.messages[key] = append(messages, message)
	lastMessageAt := message.CreatedAt
	value.LastMessageAt = &lastMessageAt
	value.UpdatedAt = message.CreatedAt
	value.RowVersion++
	r.conversations[key] = value
	return message, nil
}

func (r *MemoryRepository) ListMessages(_ context.Context, customer identity.CustomerContext, conversationID string, afterSequence int64, limit int) (MessagePage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := objectKey(customer.TenantID, conversationID)
	value, ok := r.conversations[key]
	if !ok || value.CustomerID != customer.CustomerID {
		return MessagePage{}, ErrNotFound
	}
	filtered := make([]Message, 0, limit+1)
	for _, message := range r.messages[key] {
		if message.SequenceNo > afterSequence {
			filtered = append(filtered, message)
		}
	}
	page := MessagePage{Items: filtered}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		page.HasMore = true
	}
	if len(page.Items) > 0 {
		page.NextAfterSequence = page.Items[len(page.Items)-1].SequenceNo
	} else {
		page.NextAfterSequence = afterSequence
	}
	return page, nil
}

func objectKey(tenantID, id string) string {
	return tenantID + ":" + id
}

func isBefore(valueTime time.Time, valueID string, cursorTime time.Time, cursorID string) bool {
	return valueTime.Before(cursorTime) || valueTime.Equal(cursorTime) && valueID < cursorID
}
