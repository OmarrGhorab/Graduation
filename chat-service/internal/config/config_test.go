package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("PORT")
	os.Unsetenv("ENV")
	os.Unsetenv("POLL_TIMEOUT_SECONDS")
	os.Unsetenv("POLL_INTERVAL_MS")
	os.Unsetenv("RATE_LIMIT_REQUESTS")
	os.Unsetenv("RATE_LIMIT_WINDOW_SECONDS")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "6004" {
		t.Errorf("Port = %v, want 6004", cfg.Port)
	}
	if cfg.Env != "development" {
		t.Errorf("Env = %v, want development", cfg.Env)
	}
	if cfg.PollTimeout != 30*time.Second {
		t.Errorf("PollTimeout = %v, want 30s", cfg.PollTimeout)
	}
	if cfg.PollInterval != 500*time.Millisecond {
		t.Errorf("PollInterval = %v, want 500ms", cfg.PollInterval)
	}
	if cfg.RateLimitRequests != 100 {
		t.Errorf("RateLimitRequests = %v, want 100", cfg.RateLimitRequests)
	}
	if cfg.RateLimitWindow != 60*time.Second {
		t.Errorf("RateLimitWindow = %v, want 60s", cfg.RateLimitWindow)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	// Set custom environment variables
	os.Setenv("PORT", "8080")
	os.Setenv("ENV", "production")
	os.Setenv("POLL_TIMEOUT_SECONDS", "60")
	os.Setenv("POLL_INTERVAL_MS", "1000")
	os.Setenv("RATE_LIMIT_REQUESTS", "200")
	os.Setenv("JWT_ACCESS_SECRET", "my-secret")

	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("ENV")
		os.Unsetenv("POLL_TIMEOUT_SECONDS")
		os.Unsetenv("POLL_INTERVAL_MS")
		os.Unsetenv("RATE_LIMIT_REQUESTS")
		os.Unsetenv("JWT_ACCESS_SECRET")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %v, want 8080", cfg.Port)
	}
	if cfg.Env != "production" {
		t.Errorf("Env = %v, want production", cfg.Env)
	}
	if cfg.PollTimeout != 60*time.Second {
		t.Errorf("PollTimeout = %v, want 60s", cfg.PollTimeout)
	}
	if cfg.PollInterval != 1000*time.Millisecond {
		t.Errorf("PollInterval = %v, want 1s", cfg.PollInterval)
	}
	if cfg.RateLimitRequests != 200 {
		t.Errorf("RateLimitRequests = %v, want 200", cfg.RateLimitRequests)
	}
	if cfg.JWTAccessSecret != "my-secret" {
		t.Errorf("JWTAccessSecret = %v, want my-secret", cfg.JWTAccessSecret)
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	cfg := &Config{Env: "development"}
	if !cfg.IsDevelopment() {
		t.Error("IsDevelopment() = false, want true")
	}

	cfg.Env = "production"
	if cfg.IsDevelopment() {
		t.Error("IsDevelopment() = true, want false")
	}
}

func TestConfig_IsProduction(t *testing.T) {
	cfg := &Config{Env: "production"}
	if !cfg.IsProduction() {
		t.Error("IsProduction() = false, want true")
	}

	cfg.Env = "development"
	if cfg.IsProduction() {
		t.Error("IsProduction() = true, want false")
	}
}

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")

	// Test existing variable
	val := getEnv("TEST_VAR", "default")
	if val != "test-value" {
		t.Errorf("getEnv() = %v, want test-value", val)
	}

	// Test non-existing variable with default
	val = getEnv("NON_EXISTING_VAR", "default-value")
	if val != "default-value" {
		t.Errorf("getEnv() = %v, want default-value", val)
	}
}

func TestGetEnvAsInt(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	os.Setenv("TEST_INVALID", "not-a-number")
	defer func() {
		os.Unsetenv("TEST_INT")
		os.Unsetenv("TEST_INVALID")
	}()

	// Test valid integer
	val := getEnvAsInt("TEST_INT", 0)
	if val != 42 {
		t.Errorf("getEnvAsInt() = %v, want 42", val)
	}

	// Test invalid integer (should return default)
	val = getEnvAsInt("TEST_INVALID", 10)
	if val != 10 {
		t.Errorf("getEnvAsInt() = %v, want 10", val)
	}

	// Test non-existing variable
	val = getEnvAsInt("NON_EXISTING", 99)
	if val != 99 {
		t.Errorf("getEnvAsInt() = %v, want 99", val)
	}
}
