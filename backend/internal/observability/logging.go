package observability

import (
	"context"
	"io"
	"log/slog"
	"regexp"
	"strings"
)

const redactedValue = "[REDACTED]"

var (
	bearerPattern        = regexp.MustCompile(`(?i)bearer\s+[a-z0-9._~+/=-]+`)
	connectionPattern    = regexp.MustCompile(`(?i)(postgres(?:ql)?://[^:\s/]+:)[^@\s]+@`)
	emailPattern         = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	mainlandPhonePattern = regexp.MustCompile(`\b1[3-9][0-9]{9}\b`)
)

type RedactingHandler struct {
	next slog.Handler
}

func NewLogger(writer io.Writer, level slog.Leveler) *slog.Logger {
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: level})
	return slog.New(&RedactingHandler{next: handler})
}

func (h *RedactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *RedactingHandler) Handle(ctx context.Context, record slog.Record) error {
	redacted := slog.NewRecord(record.Time, record.Level, sanitizeText(record.Message), record.PC)
	record.Attrs(func(attribute slog.Attr) bool {
		redacted.AddAttrs(redactAttr(attribute))
		return true
	})
	return h.next.Handle(ctx, redacted)
}

func (h *RedactingHandler) WithAttrs(attributes []slog.Attr) slog.Handler {
	redacted := make([]slog.Attr, 0, len(attributes))
	for _, attribute := range attributes {
		redacted = append(redacted, redactAttr(attribute))
	}
	return &RedactingHandler{next: h.next.WithAttrs(redacted)}
}

func (h *RedactingHandler) WithGroup(name string) slog.Handler {
	return &RedactingHandler{next: h.next.WithGroup(name)}
}

func redactAttr(attribute slog.Attr) slog.Attr {
	attribute.Value = attribute.Value.Resolve()
	key := normalizeKey(attribute.Key)
	if isSecretKey(key) || isPrivateContentKey(key) {
		return slog.String(attribute.Key, redactedValue)
	}
	if attribute.Value.Kind() == slog.KindGroup {
		children := attribute.Value.Group()
		redacted := make([]slog.Attr, 0, len(children))
		for _, child := range children {
			redacted = append(redacted, redactAttr(child))
		}
		return slog.Group(attribute.Key, attrsToAny(redacted)...)
	}
	if attribute.Value.Kind() == slog.KindString {
		return slog.String(attribute.Key, sanitizeText(attribute.Value.String()))
	}
	if attribute.Value.Kind() == slog.KindAny {
		if err, ok := attribute.Value.Any().(error); ok {
			return slog.String(attribute.Key, sanitizeText(err.Error()))
		}
	}
	return attribute
}

func attrsToAny(attributes []slog.Attr) []any {
	values := make([]any, len(attributes))
	for index, attribute := range attributes {
		values[index] = attribute
	}
	return values
}

func normalizeKey(key string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "-", "_"))
}

func isSecretKey(key string) bool {
	if key == "authorization" || key == "cookie" || key == "set_cookie" || key == "database_url" || key == "dsn" {
		return true
	}
	return key == "token" || strings.HasSuffix(key, "_token") || key == "password" || strings.HasSuffix(key, "_password") || key == "secret" || strings.HasSuffix(key, "_secret") || key == "api_key" || strings.HasSuffix(key, "_api_key")
}

func isPrivateContentKey(key string) bool {
	switch key {
	case "email", "phone", "address", "order_id", "order_number", "prompt", "raw_input", "customer_input", "content_text", "message_text":
		return true
	default:
		return false
	}
}

func sanitizeText(value string) string {
	value = bearerPattern.ReplaceAllString(value, "Bearer "+redactedValue)
	value = connectionPattern.ReplaceAllString(value, `${1}`+redactedValue+"@")
	value = emailPattern.ReplaceAllString(value, redactedValue)
	return mainlandPhonePattern.ReplaceAllString(value, redactedValue)
}
