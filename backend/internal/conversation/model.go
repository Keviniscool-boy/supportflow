package conversation

import "time"

type ActorType string
type ContentType string

const (
	ActorCustomer ActorType = "CUSTOMER"
	ActorAgent    ActorType = "AGENT"
	ActorMember   ActorType = "MEMBER"
	ActorSystem   ActorType = "SYSTEM"

	ContentText          ContentType = "TEXT"
	ContentOrderCard     ContentType = "ORDER_CARD"
	ContentTicketCard    ContentType = "TICKET_CARD"
	ContentHandoffStatus ContentType = "HANDOFF_STATUS"
	ContentSystemStatus  ContentType = "SYSTEM_STATUS"
)

type Conversation struct {
	TenantID      string
	ID            string
	Number        string
	CustomerID    string
	Status        string
	ResponseOwner string
	Subject       *string
	LastMessageAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	RowVersion    int64
}

func (c Conversation) SortTime() time.Time {
	if c.LastMessageAt != nil {
		return *c.LastMessageAt
	}
	return c.CreatedAt
}

type Message struct {
	TenantID       string
	ID             string
	ConversationID string
	SequenceNo     int64
	ActorType      ActorType
	CustomerID     *string
	MemberID       *string
	AgentRunID     *string
	ContentType    ContentType
	ContentText    string
	Locale         string
	RedactionState string
	ContentSHA256  string
	CreatedAt      time.Time
}

type ConversationCursor struct {
	SortTime time.Time
	ID       string
}

type ConversationListOptions struct {
	Limit  int
	Before *ConversationCursor
}

type ConversationPage struct {
	Items []Conversation
	Next  *ConversationCursor
}

type MessagePage struct {
	Items             []Message
	NextAfterSequence int64
	HasMore           bool
}

type NewMessage struct {
	ActorType   ActorType
	ContentType ContentType
	Text        string
	Locale      string
	MemberID    string
	AgentRunID  string
}
