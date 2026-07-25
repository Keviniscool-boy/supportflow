package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment    string
	HTTPAddress    string
	DatabaseURL    string
	MaxBodyBytes   int64
	RequestTimeout time.Duration
	ModelMode      string
	AllowedOrigin  string
	SessionTTL     time.Duration
	SecureCookies  bool
}

func Load() (Config, error) {
	maxBodyBytes, err := int64Env("SUPPORTFLOW_MAX_BODY_BYTES", 1<<20)
	if err != nil {
		return Config{}, err
	}
	requestTimeout, err := durationEnv("SUPPORTFLOW_REQUEST_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, err
	}
	sessionTTL, err := durationEnv("SUPPORTFLOW_SESSION_TTL", 30*time.Minute)
	if err != nil {
		return Config{}, err
	}
	config := Config{
		Environment:    stringEnv("SUPPORTFLOW_ENV", "development"),
		HTTPAddress:    stringEnv("SUPPORTFLOW_HTTP_ADDR", ":8080"),
		DatabaseURL:    os.Getenv("SUPPORTFLOW_DATABASE_URL"),
		MaxBodyBytes:   maxBodyBytes,
		RequestTimeout: requestTimeout,
		ModelMode:      stringEnv("SUPPORTFLOW_MODEL_MODE", "mock"),
		AllowedOrigin:  stringEnv("SUPPORTFLOW_ALLOWED_ORIGIN", ""),
		SessionTTL:     sessionTTL,
		SecureCookies:  secureCookies(stringEnv("SUPPORTFLOW_ENV", "development")),
	}
	if value := os.Getenv("SUPPORTFLOW_COOKIE_SECURE"); value != "" {
		parsed, parseErr := strconv.ParseBool(value)
		if parseErr != nil {
			return Config{}, fmt.Errorf("SUPPORTFLOW_COOKIE_SECURE must be true or false")
		}
		config.SecureCookies = parsed
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c Config) Validate() error {
	switch c.Environment {
	case "development", "lite", "production-like":
	default:
		return fmt.Errorf("SUPPORTFLOW_ENV must be development, lite, or production-like")
	}
	if c.HTTPAddress == "" {
		return fmt.Errorf("SUPPORTFLOW_HTTP_ADDR must not be empty")
	}
	if c.MaxBodyBytes < 1 || c.MaxBodyBytes > 32<<20 {
		return fmt.Errorf("SUPPORTFLOW_MAX_BODY_BYTES must be between 1 and 33554432")
	}
	if c.RequestTimeout <= 0 || c.RequestTimeout > 2*time.Minute {
		return fmt.Errorf("SUPPORTFLOW_REQUEST_TIMEOUT must be between 1ns and 2m")
	}
	if c.SessionTTL <= 0 {
		return fmt.Errorf("SUPPORTFLOW_SESSION_TTL must be positive")
	}
	if c.ModelMode != "mock" && c.ModelMode != "openai-compatible" {
		return fmt.Errorf("SUPPORTFLOW_MODEL_MODE must be mock or openai-compatible")
	}
	return nil
}

func stringEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func int64Env(key string, fallback int64) (int64, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	return parsed, nil
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration", key)
	}
	return parsed, nil
}

func secureCookies(environment string) bool {
	return environment == "production-like"
}
