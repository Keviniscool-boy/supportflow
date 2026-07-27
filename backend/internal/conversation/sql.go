package conversation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Keviniscool-boy/supportflow/backend/internal/identity"
)

type SQLRepository struct {
	database *sql.DB
}

func NewSQLRepository(database *sql.DB) *SQLRepository {
	return &SQLRepository{database: database}
}

func (r *SQLRepository) Create(ctx context.Context, value Conversation) (Conversation, error) {
	row := r.database.QueryRowContext(ctx, `INSERT INTO conversations
		(tenant_id, id, conversation_number, customer_id, status, response_owner, subject, created_at, updated_at, row_version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8, $9)
		RETURNING tenant_id::text, id::text, conversation_number, customer_id::text, status::text, response_owner::text,
			subject, last_message_at, created_at, updated_at, row_version`,
		value.TenantID, value.ID, value.Number, value.CustomerID, value.Status, value.ResponseOwner, value.Subject, value.CreatedAt, value.RowVersion)
	created, err := scanConversation(row)
	if err != nil {
		return Conversation{}, fmt.Errorf("create conversation: %w", err)
	}
	return created, nil
}

func (r *SQLRepository) List(ctx context.Context, customer identity.CustomerContext, options ConversationListOptions) (ConversationPage, error) {
	var rows *sql.Rows
	var err error
	if options.Before == nil {
		rows, err = r.database.QueryContext(ctx, `SELECT tenant_id::text, id::text, conversation_number, customer_id::text, status::text,
			response_owner::text, subject, last_message_at, created_at, updated_at, row_version
			FROM conversations WHERE tenant_id = $1 AND customer_id = $2
			ORDER BY COALESCE(last_message_at, created_at) DESC, id DESC LIMIT $3`, customer.TenantID, customer.CustomerID, options.Limit+1)
	} else {
		rows, err = r.database.QueryContext(ctx, `SELECT tenant_id::text, id::text, conversation_number, customer_id::text, status::text,
			response_owner::text, subject, last_message_at, created_at, updated_at, row_version
			FROM conversations WHERE tenant_id = $1 AND customer_id = $2
			AND (COALESCE(last_message_at, created_at), id) < ($3, $4::uuid)
			ORDER BY COALESCE(last_message_at, created_at) DESC, id DESC LIMIT $5`,
			customer.TenantID, customer.CustomerID, options.Before.SortTime, options.Before.ID, options.Limit+1)
	}
	if err != nil {
		return ConversationPage{}, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()
	page := ConversationPage{Items: make([]Conversation, 0, options.Limit)}
	for rows.Next() {
		value, scanErr := scanConversation(rows)
		if scanErr != nil {
			return ConversationPage{}, fmt.Errorf("scan conversation list: %w", scanErr)
		}
		page.Items = append(page.Items, value)
	}
	if err := rows.Err(); err != nil {
		return ConversationPage{}, fmt.Errorf("iterate conversation list: %w", err)
	}
	if len(page.Items) > options.Limit {
		page.Items = page.Items[:options.Limit]
		last := page.Items[len(page.Items)-1]
		page.Next = &ConversationCursor{SortTime: last.SortTime(), ID: last.ID}
	}
	return page, nil
}

func (r *SQLRepository) Get(ctx context.Context, customer identity.CustomerContext, conversationID string) (Conversation, error) {
	row := r.database.QueryRowContext(ctx, `SELECT tenant_id::text, id::text, conversation_number, customer_id::text, status::text,
		response_owner::text, subject, last_message_at, created_at, updated_at, row_version
		FROM conversations WHERE tenant_id = $1 AND customer_id = $2 AND id = $3`, customer.TenantID, customer.CustomerID, conversationID)
	value, err := scanConversation(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Conversation{}, ErrNotFound
	}
	if err != nil {
		return Conversation{}, fmt.Errorf("get conversation: %w", err)
	}
	return value, nil
}

func (r *SQLRepository) AppendMessage(ctx context.Context, customer identity.CustomerContext, conversationID string, message Message) (Message, error) {
	transaction, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return Message{}, fmt.Errorf("begin message transaction: %w", err)
	}
	defer transaction.Rollback()
	var sequence int64
	err = transaction.QueryRowContext(ctx, `UPDATE conversations
		SET next_message_sequence = next_message_sequence + 1, last_message_at = $4, updated_at = $4, row_version = row_version + 1
		WHERE tenant_id = $1 AND customer_id = $2 AND id = $3 AND status = 'OPEN'
		RETURNING next_message_sequence - 1`, customer.TenantID, customer.CustomerID, conversationID, message.CreatedAt).Scan(&sequence)
	if errors.Is(err, sql.ErrNoRows) {
		return Message{}, ErrNotFound
	}
	if err != nil {
		return Message{}, fmt.Errorf("reserve message sequence: %w", err)
	}
	message.SequenceNo = sequence
	_, err = transaction.ExecContext(ctx, `INSERT INTO messages
		(tenant_id, id, conversation_id, sequence_no, actor_type, customer_id, member_id, agent_run_id,
		 content_type, content_text, locale, redaction_state, content_sha256, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		message.TenantID, message.ID, message.ConversationID, message.SequenceNo, message.ActorType,
		message.CustomerID, message.MemberID, message.AgentRunID, message.ContentType, message.ContentText,
		message.Locale, message.RedactionState, message.ContentSHA256, message.CreatedAt)
	if err != nil {
		return Message{}, fmt.Errorf("insert message: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return Message{}, fmt.Errorf("commit message transaction: %w", err)
	}
	return message, nil
}

func (r *SQLRepository) ListMessages(ctx context.Context, customer identity.CustomerContext, conversationID string, afterSequence int64, limit int) (MessagePage, error) {
	rows, err := r.database.QueryContext(ctx, `SELECT m.tenant_id::text, m.id::text, m.conversation_id::text, m.sequence_no,
		m.actor_type::text, m.customer_id::text, m.member_id::text, m.agent_run_id::text, m.content_type::text,
		m.content_text, m.locale, m.redaction_state, m.content_sha256, m.created_at
		FROM messages m JOIN conversations c ON c.tenant_id = m.tenant_id AND c.id = m.conversation_id
		WHERE m.tenant_id = $1 AND m.conversation_id = $2 AND c.customer_id = $3 AND m.sequence_no > $4
		ORDER BY m.sequence_no ASC LIMIT $5`, customer.TenantID, conversationID, customer.CustomerID, afterSequence, limit+1)
	if err != nil {
		return MessagePage{}, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()
	page := MessagePage{Items: make([]Message, 0, limit), NextAfterSequence: afterSequence}
	for rows.Next() {
		message, scanErr := scanMessage(rows)
		if scanErr != nil {
			return MessagePage{}, fmt.Errorf("scan message list: %w", scanErr)
		}
		page.Items = append(page.Items, message)
	}
	if err := rows.Err(); err != nil {
		return MessagePage{}, fmt.Errorf("iterate message list: %w", err)
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		page.HasMore = true
	}
	if len(page.Items) > 0 {
		page.NextAfterSequence = page.Items[len(page.Items)-1].SequenceNo
	}
	if len(page.Items) == 0 {
		if _, err := r.Get(ctx, customer, conversationID); err != nil {
			return MessagePage{}, err
		}
	}
	return page, nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanConversation(scanner rowScanner) (Conversation, error) {
	var value Conversation
	var subject sql.NullString
	var lastMessageAt sql.NullTime
	err := scanner.Scan(&value.TenantID, &value.ID, &value.Number, &value.CustomerID, &value.Status, &value.ResponseOwner,
		&subject, &lastMessageAt, &value.CreatedAt, &value.UpdatedAt, &value.RowVersion)
	if err != nil {
		return Conversation{}, err
	}
	if subject.Valid {
		value.Subject = &subject.String
	}
	if lastMessageAt.Valid {
		last := lastMessageAt.Time.UTC()
		value.LastMessageAt = &last
	}
	value.CreatedAt = value.CreatedAt.UTC()
	value.UpdatedAt = value.UpdatedAt.UTC()
	return value, nil
}

func scanMessage(scanner rowScanner) (Message, error) {
	var value Message
	var customerID, memberID, agentRunID sql.NullString
	err := scanner.Scan(&value.TenantID, &value.ID, &value.ConversationID, &value.SequenceNo, &value.ActorType,
		&customerID, &memberID, &agentRunID, &value.ContentType, &value.ContentText, &value.Locale,
		&value.RedactionState, &value.ContentSHA256, &value.CreatedAt)
	if err != nil {
		return Message{}, err
	}
	if customerID.Valid {
		value.CustomerID = &customerID.String
	}
	if memberID.Valid {
		value.MemberID = &memberID.String
	}
	if agentRunID.Valid {
		value.AgentRunID = &agentRunID.String
	}
	value.CreatedAt = value.CreatedAt.UTC()
	return value, nil
}
