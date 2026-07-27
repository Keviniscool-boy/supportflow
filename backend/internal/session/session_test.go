package session

import (
	"context"
	"errors"
	"testing"
	"time"

	appclock "github.com/Keviniscool-boy/supportflow/backend/internal/clock"
	"github.com/Keviniscool-boy/supportflow/backend/internal/testfixture"
)

func TestMemoryStoreCreatesAndReusesSession(t *testing.T) {
	store := NewMemoryStore(time.Hour)
	created, reused, err := store.CreateOrGet(context.Background(), "token", "zh-CN")
	if err != nil || reused {
		t.Fatalf("expected new session, reused=%v err=%v", reused, err)
	}
	loaded, reused, err := store.CreateOrGet(context.Background(), "token", "en-US")
	if err != nil || !reused || loaded.ID != created.ID {
		t.Fatalf("expected reused session, reused=%v err=%v", reused, err)
	}
}

func TestMemoryStoreExpiresAndRevokes(t *testing.T) {
	clock := appclock.NewFixed(testfixture.FrozenTime)
	store := NewMemoryStoreWithClock(time.Hour, clock)
	_, _, err := store.CreateOrGet(context.Background(), "token", "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Hour)
	if _, err := store.Get(context.Background(), "token"); !errors.Is(err, ErrExpired) {
		t.Fatalf("expected expired session, got %v", err)
	}
	store = NewMemoryStoreWithClock(time.Hour, appclock.NewFixed(testfixture.FrozenTime))
	_, _, err = store.CreateOrGet(context.Background(), "token", "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Revoke(context.Background(), "token"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(context.Background(), "token"); !errors.Is(err, ErrRevoked) {
		t.Fatalf("expected revoked session, got %v", err)
	}
}

func TestCSRFTokenIsDeterministic(t *testing.T) {
	first := CSRFToken([]byte("secret"), "token")
	second := CSRFToken([]byte("secret"), "token")
	if first != second || first == "" {
		t.Fatalf("expected stable csrf token")
	}
}
