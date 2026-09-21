package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Host                    string
	Port                    int
	RegistryURL             string
	RefreshEnabled          bool
	RefreshInterval         time.Duration
	RefreshTimeout          time.Duration
	MaxPages                int
	MaxBytes                int64
	MaxDecisionRequestBytes int64
}

func FromEnvironment() (Config, error) {
	cfg := Config{
		Host:                    envOr("DECISION_HOST", "127.0.0.1"),
		RegistryURL:             envOr("POLICY_REGISTRY_URL", "http://127.0.0.1:8082"),
		MaxDecisionRequestBytes: 64 * 1024,
	}

	var err error
	if cfg.Port, err = intEnv("DECISION_PORT", 8081, 1); err != nil {
		return Config{}, err
	}
	if cfg.RefreshEnabled, err = boolEnv("POLICY_REFRESH_ENABLED", false); err != nil {
		return Config{}, err
	}
	if cfg.RefreshInterval, err = durationEnv("POLICY_REFRESH_INTERVAL", 30*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.RefreshTimeout, err = durationEnv("POLICY_REFRESH_TIMEOUT", 200*time.Millisecond); err != nil {
		return Config{}, err
	}
	if cfg.MaxPages, err = intEnv("POLICY_MAX_PAGES", 16, 1); err != nil {
		return Config{}, err
	}
	maxBytes, err := intEnv("POLICY_MAX_BYTES", 1024*1024, 1)
	if err != nil {
		return Config{}, err
	}
	cfg.MaxBytes = int64(maxBytes)
	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func intEnv(key string, fallback, minimum int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum {
		return 0, fmt.Errorf("%s must be an integer greater than or equal to %d", key, minimum)
	}
	return value, nil
}

func boolEnv(key string, fallback bool) (bool, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean", key)
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
