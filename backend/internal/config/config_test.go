package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadAllowsLocalDefaults(t *testing.T) {
	unsetEnv(t, "APP_ENV")
	unsetEnv(t, "ALLOWED_ORIGIN")
	unsetEnv(t, "TRUSTED_PROXY_CIDRS")
	unsetEnv(t, "RATE_LIMIT_ENABLED")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error for local defaults: %v", err)
	}

	if cfg.AppEnv != defaultAppEnv {
		t.Fatalf("expected default APP_ENV %q, got %q", defaultAppEnv, cfg.AppEnv)
	}

	if cfg.AllowedOrigin != defaultAllowedOrigin {
		t.Fatalf("expected default ALLOWED_ORIGIN %q, got %q", defaultAllowedOrigin, cfg.AllowedOrigin)
	}
}

func TestLoadRejectsProductionWithoutExplicitAllowedOrigin(t *testing.T) {
	setValidProductionBaseEnv(t)
	unsetEnv(t, "ALLOWED_ORIGIN")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "ALLOWED_ORIGIN") {
		t.Fatalf("expected ALLOWED_ORIGIN validation error, got %v", err)
	}
}

func TestLoadRejectsProductionUnsafeAllowedOrigin(t *testing.T) {
	setValidProductionBaseEnv(t)
	t.Setenv("ALLOWED_ORIGIN", "http://localhost:5173")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "ALLOWED_ORIGIN") {
		t.Fatalf("expected ALLOWED_ORIGIN validation error, got %v", err)
	}
}

func TestLoadRejectsProductionWithoutExplicitTrustedProxyCIDRs(t *testing.T) {
	setValidProductionBaseEnv(t)
	unsetEnv(t, "TRUSTED_PROXY_CIDRS")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "TRUSTED_PROXY_CIDRS") {
		t.Fatalf("expected TRUSTED_PROXY_CIDRS validation error, got %v", err)
	}
}

func TestLoadRejectsProductionInvalidTrustedProxyCIDRs(t *testing.T) {
	setValidProductionBaseEnv(t)
	t.Setenv("TRUSTED_PROXY_CIDRS", "not-a-cidr")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "TRUSTED_PROXY_CIDRS") {
		t.Fatalf("expected TRUSTED_PROXY_CIDRS validation error, got %v", err)
	}
}

func TestLoadRejectsProductionWhenRateLimitingIsDisabled(t *testing.T) {
	setValidProductionBaseEnv(t)
	t.Setenv("RATE_LIMIT_ENABLED", "false")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "RATE_LIMIT_ENABLED") {
		t.Fatalf("expected RATE_LIMIT_ENABLED validation error, got %v", err)
	}
}

func TestLoadAcceptsValidProductionConfig(t *testing.T) {
	setValidProductionBaseEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.AllowedOrigin != "https://secret.quorix.io.vn" {
		t.Fatalf("expected production ALLOWED_ORIGIN to round-trip, got %q", cfg.AllowedOrigin)
	}

	if cfg.TrustedProxyCIDRs != "172.16.0.0/12" {
		t.Fatalf("expected production TRUSTED_PROXY_CIDRS to round-trip, got %q", cfg.TrustedProxyCIDRs)
	}
}

func setValidProductionBaseEnv(t *testing.T) {
	t.Helper()

	t.Setenv("APP_ENV", "production")
	t.Setenv("ALLOWED_ORIGIN", "https://secret.quorix.io.vn")
	t.Setenv("TRUSTED_PROXY_CIDRS", "172.16.0.0/12")
	t.Setenv("RATE_LIMIT_ENABLED", "true")
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()

	previousValue, existed := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("failed to unset %s: %v", key, err)
	}

	t.Cleanup(func() {
		var err error
		if existed {
			err = os.Setenv(key, previousValue)
		} else {
			err = os.Unsetenv(key)
		}

		if err != nil {
			t.Fatalf("failed to restore %s: %v", key, err)
		}
	})
}
