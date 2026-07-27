package redaction

import (
	"strings"
	"testing"
)

func TestTextRemovesCommonSensitiveValues(t *testing.T) {
	result := Text(" 联系 test@example.com 或 13800138000，订单 SF20260001 ")
	for _, secret := range []string{"test@example.com", "13800138000", "SF20260001"} {
		if strings.Contains(result, secret) {
			t.Fatalf("redaction leaked %q: %s", secret, result)
		}
	}
	if !strings.Contains(result, "0001") {
		t.Fatalf("expected safe order suffix, got %s", result)
	}
}
