package observability

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestLoggerRedactsSensitiveFieldsAndValues(t *testing.T) {
	var output bytes.Buffer
	logger := NewLogger(&output, slog.LevelInfo)
	logger.ErrorContext(context.Background(), "provider failed for user@example.com",
		"authorization", "Bearer top-secret",
		"order_id", "SF20260001",
		"error", errors.New("connect postgres://supportflow:password@database/supportflow"),
		"token_count", 42,
	)
	logged := output.String()
	for _, secret := range []string{"top-secret", "SF20260001", "user@example.com", ":password@"} {
		if strings.Contains(logged, secret) {
			t.Fatalf("log leaked %q: %s", secret, logged)
		}
	}
	if !strings.Contains(logged, `"token_count":42`) {
		t.Fatalf("non-secret metric was removed: %s", logged)
	}
}

func TestLoggerRedactsNestedGroups(t *testing.T) {
	var output bytes.Buffer
	logger := NewLogger(&output, slog.LevelInfo).With("component", "test")
	logger.Info("request", slog.Group("customer", "email", "private@example.com", "locale", "zh-CN"))
	logged := output.String()
	if strings.Contains(logged, "private@example.com") || !strings.Contains(logged, "zh-CN") {
		t.Fatalf("unexpected grouped redaction: %s", logged)
	}
}
