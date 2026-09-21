package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Host             string
	Port             int
	UpstreamURL      string
	UpstreamTimeout  time.Duration
	MaxResponseBytes int64
	MaxRequestBytes  int64
}

func FromEnvironment() (Config, error) {
	cfg := Config{
		Host:            envOr("GATEWAY_HOST", "127.0.0.1"),
		UpstreamURL:     envOr("DECISION_SERVICE_URL", "http://127.0.0.1:8081"),
		MaxRequestBytes: 64 * 1024,
	}
	var err error
	if cfg.Port, err = intEnv("GATEWAY_PORT", 8080); err != nil {
		return Config{}, err
	}
	if cfg.UpstreamTimeout, err = durationEnv("GATEWAY_UPSTREAM_TIMEOUT", 2*time.Second); err != nil {
		return Config{}, err
	}
	maxResponse, err := intEnv("GATEWAY_MAX_RESPONSE_BYTES", 1024*1024)
	if err != nil {
		return Config{}, err
	}
	cfg.MaxResponseBytes = int64(maxResponse)
	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func intEnv(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return value, nil
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return value, nil
}
