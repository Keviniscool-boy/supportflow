package conversation

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	appclock "github.com/Keviniscool-boy/supportflow/backend/internal/clock"
	"github.com/Keviniscool-boy/supportflow/backend/internal/identity"
	"github.com/Keviniscool-boy/supportflow/backend/internal/redaction"
)

var (
	ErrNotFound       = errors.New("conversation not found")
	ErrClosed         = errors.New("conversation closed")
	ErrInvalidSubject = errors.New("invalid conversation subject")
	ErrInvalidMessage = errors.New("invalid message")
)

var objectIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type Repository interface {
	Create(context.Context, Conversation) (Conversation, error)
	List(context.Context, identity.CustomerContext, ConversationListOptions) (ConversationPage, error)
	Get(context.Context, identity.CustomerContext, string) (Conversation, error)
	AppendMessage(context.Context, identity.CustomerContext, string, Message) (Message, error)
	ListMessages(context.Context, identity.CustomerContext, string, int64, int) (MessagePage, error)
}

type Service struct {
	repository Repository
	clock      appclock.Clock
}

func NewService(repository Repository, clock appclock.Clock) *Service {
	return &Service{repository: repository, clock: clock}
}

func (s *Service) Create(ctx context.Context, customer identity.CustomerContext, subject string) (Conversation, error) {
	if !validCustomer(customer) {
		return Conversation{}, ErrNotFound
	}
	redactedSubject := redaction.Text(subject)
	if utf8.RuneCountInString(redactedSubject) > 240 {
		return Conversation{}, ErrInvalidSubject
	}
	var subjectValue *string
	if redactedSubject != "" {
		subjectValue = &redactedSubject
	}
	id, err := newID()
	if err != nil {
		return Conversation{}, err
	}
	now := s.clock.Now()
	created := Conversation{
		TenantID: customer.TenantID, ID: id, Number: displayNumber("CV", id), CustomerID: customer.CustomerID,
		Status: "OPEN", ResponseOwner: "AGENT", Subject: subjectValue, CreatedAt: now, UpdatedAt: now, RowVersion: 1,
	}
	return s.repository.Create(ctx, created)
}

func (s *Service) List(ctx context.Context, customer identity.CustomerContext, options ConversationListOptions) (ConversationPage, error) {
	if !validCustomer(customer) {
		return ConversationPage{}, ErrNotFound
	}
	if options.Limit < 1 || options.Limit > 100 {
		return ConversationPage{}, ErrInvalidMessage
	}
	return s.repository.List(ctx, customer, options)
}

func (s *Service) Get(ctx context.Context, customer identity.CustomerContext, conversationID string) (Conversation, error) {
	if !validCustomer(customer) || !objectIDPattern.MatchString(conversationID) {
		return Conversation{}, ErrNotFound
	}
	return s.repository.Get(ctx, customer, conversationID)
}

func (s *Service) AppendMessage(ctx context.Context, customer identity.CustomerContext, conversationID string, input NewMessage) (Message, error) {
	if !validCustomer(customer) || !objectIDPattern.MatchString(conversationID) || !validActor(input.ActorType) || !validContentType(input.ContentType) {
		return Message{}, ErrInvalidMessage
	}
	text := redaction.Text(input.Text)
	if text == "" || utf8.RuneCountInString(text) > 50000 {
		return Message{}, ErrInvalidMessage
	}
	if input.ActorType == ActorCustomer && utf8.RuneCountInString(text) > 8000 {
		return Message{}, ErrInvalidMessage
	}
	if input.ActorType == ActorMember && input.MemberID == "" || input.ActorType != ActorMember && input.MemberID != "" {
		return Message{}, ErrInvalidMessage
	}
	id, err := newID()
	if err != nil {
		return Message{}, err
	}
	message := Message{
		TenantID: customer.TenantID, ID: id, ConversationID: conversationID, ActorType: input.ActorType,
		ContentType: input.ContentType, ContentText: text, Locale: normalizeLocale(input.Locale, customer.Locale),
		RedactionState: "APPLIED", ContentSHA256: hashText(text), CreatedAt: s.clock.Now(),
	}
	if input.ActorType == ActorCustomer {
		customerID := customer.CustomerID
		message.CustomerID = &customerID
	}
	if input.ActorType == ActorMember {
		message.MemberID = &input.MemberID
	}
	if input.AgentRunID != "" {
		message.AgentRunID = &input.AgentRunID
	}
	return s.repository.AppendMessage(ctx, customer, conversationID, message)
}

func (s *Service) ListMessages(ctx context.Context, customer identity.CustomerContext, conversationID string, afterSequence int64, limit int) (MessagePage, error) {
	if !validCustomer(customer) || !objectIDPattern.MatchString(conversationID) || afterSequence < 0 || limit < 1 || limit > 200 {
		return MessagePage{}, ErrInvalidMessage
	}
	return s.repository.ListMessages(ctx, customer, conversationID, afterSequence, limit)
}

func validCustomer(customer identity.CustomerContext) bool {
	return customer.TenantID != "" && customer.CustomerID != "" && customer.SessionID != ""
}

func validActor(actor ActorType) bool {
	return actor == ActorCustomer || actor == ActorAgent || actor == ActorMember || actor == ActorSystem
}

func validContentType(contentType ContentType) bool {
	return contentType == ContentText || contentType == ContentOrderCard || contentType == ContentTicketCard || contentType == ContentHandoffStatus || contentType == ContentSystemStatus
}

func normalizeLocale(locale, fallback string) string {
	if locale == "zh-CN" || locale == "en-US" {
		return locale
	}
	if fallback == "en-US" {
		return fallback
	}
	return "zh-CN"
}

func hashText(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func newID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate conversation id: %w", err)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s", hex.EncodeToString(value[0:4]), hex.EncodeToString(value[4:6]), hex.EncodeToString(value[6:8]), hex.EncodeToString(value[8:10]), hex.EncodeToString(value[10:16])), nil
}

func displayNumber(prefix, id string) string {
	return prefix + strings.ToUpper(strings.ReplaceAll(id, "-", "")[:12])
}
