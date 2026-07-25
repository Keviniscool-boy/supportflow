package config

import (
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	config := Config{
		Environment:    "lite",
		HTTPAddress:    ":8080",
		MaxBodyBytes:   1 << 20,
		RequestTimeout: 15 * time.Second,
		ModelMode:      "mock",
		SessionTTL:     30 * time.Minute,
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestConfigRejectsUnknownModelMode(t *testing.T) {
	config := Config{
		Environment:    "development",
		HTTPAddress:    ":8080",
		MaxBodyBytes:   1024,
		RequestTimeout: time.Second,
		ModelMode:      "custom",
		SessionTTL:     time.Minute,
	}
	if err := config.Validate(); err == nil {
		t.Fatal("expected unknown model mode to be rejected")
	}
}
