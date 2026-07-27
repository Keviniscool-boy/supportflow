package session

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	appclock "github.com/Keviniscool-boy/supportflow/backend/internal/clock"
)

const DefaultTenantID = "00000000-0000-0000-0000-000000000001"

var (
	ErrNotFound  = errors.New("session not found")
	ErrExpired   = errors.New("session expired")
	ErrRevoked   = errors.New("session revoked")
	ErrResetting = errors.New("session resetting")
)

type Session struct {
	TenantID       string
	ID             string
	CustomerID     string
	DisplayName    string
	Locale         string
	Status         string
	SessionType    string
	DataGeneration int
	RequestLimit   int
	TokenLimit     int64
	UploadLimit    int
	RequestCount   int
	TokenCount     int64
	UploadCount    int
	ExpiresAt      time.Time
}

type Store interface {
	CreateOrGet(context.Context, string, string) (Session, bool, error)
	Get(context.Context, string) (Session, error)
	Revoke(context.Context, string) error
}

type MemoryStore struct {
	mu       sync.Mutex
	sessions map[string]Session
	ttl      time.Duration
	clock    appclock.Clock
}

func NewMemoryStore(ttl time.Duration) *MemoryStore {
	return NewMemoryStoreWithClock(ttl, appclock.System{})
}

func NewMemoryStoreWithClock(ttl time.Duration, clock appclock.Clock) *MemoryStore {
	return &MemoryStore{sessions: make(map[string]Session), ttl: ttl, clock: clock}
}

func (s *MemoryStore) CreateOrGet(_ context.Context, rawToken, locale string) (Session, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	hash := TokenHash(rawToken)
	if existing, ok := s.sessions[hash]; ok {
		if err := active(existing, s.clock.Now()); err != nil {
			return Session{}, false, err
		}
		return existing, true, nil
	}
	now := s.clock.Now()
	created := Session{
		TenantID:       DefaultTenantID,
		ID:             newID(),
		CustomerID:     newID(),
		DisplayName:    "演示客户",
		Locale:         normalizeLocale(locale),
		Status:         "ACTIVE",
		SessionType:    "VISITOR",
		DataGeneration: 1,
		RequestLimit:   100,
		TokenLimit:     100000,
		UploadLimit:    5,
		ExpiresAt:      now.Add(s.ttl),
	}
	s.sessions[hash] = created
	return created, false, nil
}

func (s *MemoryStore) Get(_ context.Context, rawToken string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.sessions[TokenHash(rawToken)]
	if !ok {
		return Session{}, ErrNotFound
	}
	if err := active(existing, s.clock.Now()); err != nil {
		return Session{}, err
	}
	return existing, nil
}

func (s *MemoryStore) Revoke(_ context.Context, rawToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	hash := TokenHash(rawToken)
	existing, ok := s.sessions[hash]
	if !ok {
		return ErrNotFound
	}
	if err := active(existing, s.clock.Now()); err != nil {
		return err
	}
	existing.Status = "REVOKED"
	s.sessions[hash] = existing
	return nil
}

type SQLStore struct {
	db    *sql.DB
	ttl   time.Duration
	clock appclock.Clock
}

func NewSQLStore(db *sql.DB, ttl time.Duration) *SQLStore {
	return NewSQLStoreWithClock(db, ttl, appclock.System{})
}

func NewSQLStoreWithClock(db *sql.DB, ttl time.Duration, clock appclock.Clock) *SQLStore {
	return &SQLStore{db: db, ttl: ttl, clock: clock}
}

func (s *SQLStore) CreateOrGet(ctx context.Context, rawToken, locale string) (Session, bool, error) {
	existing, err := s.query(ctx, TokenHash(rawToken))
	if err == nil {
		if err := active(existing, s.clock.Now()); err != nil {
			return Session{}, false, err
		}
		return existing, true, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Session{}, false, err
	}
	now := s.clock.Now()
	expiresAt := now.Add(s.ttl)
	sessionID := newID()
	customerID := newID()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Session{}, false, fmt.Errorf("begin session transaction: %w", err)
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO demo_sessions
		(tenant_id, id, token_hash, session_type, data_generation, request_limit, token_limit, upload_file_limit, expires_at, created_at, updated_at)
		SELECT id, $1, $2, 'VISITOR', data_generation, 100, 100000, 5, $3, $4, $4 FROM tenants WHERE id = $5`,
		sessionID, TokenHash(rawToken), expiresAt, now, DefaultTenantID)
	if err != nil {
		return Session{}, false, fmt.Errorf("insert demo session: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO customers (tenant_id, id, demo_session_id, display_name, locale, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)`, DefaultTenantID, customerID, sessionID, "演示客户", normalizeLocale(locale), now)
	if err != nil {
		return Session{}, false, fmt.Errorf("insert demo customer: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Session{}, false, fmt.Errorf("commit session transaction: %w", err)
	}
	created, err := s.query(ctx, TokenHash(rawToken))
	if err != nil {
		return Session{}, false, err
	}
	return created, false, nil
}

func (s *SQLStore) Get(ctx context.Context, rawToken string) (Session, error) {
	existing, err := s.query(ctx, TokenHash(rawToken))
	if err != nil {
		return Session{}, err
	}
	if err := active(existing, s.clock.Now()); err != nil {
		if errors.Is(err, ErrExpired) {
			_, _ = s.db.ExecContext(ctx, "UPDATE demo_sessions SET status = 'EXPIRED', updated_at = $3, row_version = row_version + 1 WHERE tenant_id = $1 AND id = $2 AND status = 'ACTIVE'", existing.TenantID, existing.ID, s.clock.Now())
		}
		return Session{}, err
	}
	return existing, nil
}

func (s *SQLStore) Revoke(ctx context.Context, rawToken string) error {
	existing, err := s.query(ctx, TokenHash(rawToken))
	if err != nil {
		return err
	}
	if err := active(existing, s.clock.Now()); err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, "UPDATE demo_sessions SET status = 'REVOKED', updated_at = $3, row_version = row_version + 1 WHERE tenant_id = $1 AND id = $2 AND status = 'ACTIVE'", existing.TenantID, existing.ID, s.clock.Now())
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return ErrRevoked
	}
	return nil
}

func (s *SQLStore) query(ctx context.Context, tokenHash string) (Session, error) {
	var result Session
	var expiresAt time.Time
	err := s.db.QueryRowContext(ctx, `SELECT s.tenant_id::text, s.id::text, c.id::text, c.display_name, c.locale,
		s.status::text, s.session_type, s.data_generation, s.request_limit, s.token_limit, s.upload_file_limit,
		s.request_count, s.token_count, s.upload_file_count, s.expires_at
		FROM demo_sessions s JOIN customers c ON c.tenant_id = s.tenant_id AND c.demo_session_id = s.id
		WHERE s.tenant_id = $1 AND s.token_hash = $2`, DefaultTenantID, tokenHash).Scan(
		&result.TenantID, &result.ID, &result.CustomerID, &result.DisplayName, &result.Locale,
		&result.Status, &result.SessionType, &result.DataGeneration, &result.RequestLimit, &result.TokenLimit, &result.UploadLimit,
		&result.RequestCount, &result.TokenCount, &result.UploadCount, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("query demo session: %w", err)
	}
	result.ExpiresAt = expiresAt.UTC()
	return result, nil
}

func active(value Session, now time.Time) error {
	if value.Status == "REVOKED" {
		return ErrRevoked
	}
	if value.Status == "RESETTING" {
		return ErrResetting
	}
	if value.Status != "ACTIVE" || !now.Before(value.ExpiresAt) {
		return ErrExpired
	}
	return nil
}

func TokenHash(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}

func CSRFToken(secret []byte, rawToken string) string {
	hash := hmac.New(sha256.New, secret)
	_, _ = hash.Write([]byte(rawToken))
	return hex.EncodeToString(hash.Sum(nil))
}

func NewToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return hex.EncodeToString(value), nil
}

func NewSecret() ([]byte, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return nil, fmt.Errorf("generate csrf secret: %w", err)
	}
	return value, nil
}

func normalizeLocale(value string) string {
	value = strings.TrimSpace(value)
	if value == "zh-CN" || value == "en-US" {
		return value
	}
	return "zh-CN"
}

func newID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		panic(err)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s", hex.EncodeToString(value[0:4]), hex.EncodeToString(value[4:6]), hex.EncodeToString(value[6:8]), hex.EncodeToString(value[8:10]), hex.EncodeToString(value[10:16]))
}
