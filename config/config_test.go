package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadValidConfiguration(t *testing.T) {
	values := validEnvironment()
	values["TRUSTED_PROXY_CIDRS"] = "10.0.0.7/8, 2001:db8::1/32"

	cfg, err := load(mapLookup(values))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}

	if cfg.Environment != EnvironmentTest {
		t.Errorf("Environment = %q, want %q", cfg.Environment, EnvironmentTest)
	}
	if cfg.Database.MaxConnections != 12 || cfg.Database.MinConnections != 2 {
		t.Errorf("database pool = %d/%d, want 2/12", cfg.Database.MinConnections, cfg.Database.MaxConnections)
	}
	if cfg.Database.QueryTimeout != 4*time.Second {
		t.Errorf("query timeout = %s, want 4s", cfg.Database.QueryTimeout)
	}
	if len(cfg.TrustedProxyCIDRs) != 2 {
		t.Fatalf("trusted proxy count = %d, want 2", len(cfg.TrustedProxyCIDRs))
	}
	if got := cfg.TrustedProxyCIDRs[0].String(); got != "10.0.0.0/8" {
		t.Errorf("masked proxy CIDR = %q, want 10.0.0.0/8", got)
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := load(mapLookup(map[string]string{
		"APP_ENV":      EnvironmentLocal,
		"DATABASE_URL": "postgres://localhost/tidekeepers",
	}))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}

	if cfg.HTTP.Address != ":8080" {
		t.Errorf("HTTP address = %q, want :8080", cfg.HTTP.Address)
	}
	if cfg.Session.CookieName != defaultSessionCookie {
		t.Errorf("cookie name = %q, want %q", cfg.Session.CookieName, defaultSessionCookie)
	}
	if !cfg.Session.CookieSecure {
		t.Error("cookie secure = false, want true by default")
	}
	if cfg.Session.TTL != defaultSessionTTL {
		t.Errorf("session TTL = %s, want %s", cfg.Session.TTL, defaultSessionTTL)
	}
	if cfg.HTTP.MaxHeaderBytes != defaultMaxHeaderBytes {
		t.Errorf("max headers = %d, want %d", cfg.HTTP.MaxHeaderBytes, defaultMaxHeaderBytes)
	}
}

func TestLoadRejectsMissingRequiredValues(t *testing.T) {
	_, err := load(mapLookup(nil))
	assertErrorContains(t, err, "APP_ENV is required")
	assertErrorContains(t, err, "DATABASE_URL is required")
}

func TestLoadRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		message string
	}{
		{name: "environment", key: "APP_ENV", value: "demo", message: "APP_ENV must be one of"},
		{name: "HTTP address", key: "HTTP_ADDR", value: "not-an-address", message: "HTTP_ADDR must be a valid TCP address"},
		{name: "duration syntax", key: "DATABASE_QUERY_TIMEOUT", value: "soon", message: "DATABASE_QUERY_TIMEOUT must be a duration"},
		{name: "duration bound", key: "HTTP_IDLE_TIMEOUT", value: "0s", message: "HTTP_IDLE_TIMEOUT must be positive"},
		{name: "integer syntax", key: "RATE_LIMIT_REQUESTS", value: "many", message: "RATE_LIMIT_REQUESTS must be an integer"},
		{name: "integer bound", key: "DATABASE_MAX_CONNECTIONS", value: "0", message: "DATABASE_MAX_CONNECTIONS must be positive"},
		{name: "pool order", key: "DATABASE_MIN_CONNECTIONS", value: "13", message: "DATABASE_MIN_CONNECTIONS must not exceed"},
		{name: "database URL", key: "DATABASE_URL", value: "https://example.com/database", message: "DATABASE_URL must be a valid PostgreSQL URL"},
		{name: "cookie name", key: "SESSION_COOKIE_NAME", value: "bad cookie", message: "SESSION_COOKIE_NAME must be a valid cookie name"},
		{name: "cookie domain", key: "SESSION_COOKIE_DOMAIN", value: "https://example.com", message: "SESSION_COOKIE_DOMAIN must be a hostname"},
		{name: "session TTL", key: "SESSION_TTL", value: "0s", message: "SESSION_TTL must be positive"},
		{name: "CIDR", key: "TRUSTED_PROXY_CIDRS", value: "10.0.0.0/8,anything", message: "TRUSTED_PROXY_CIDRS must contain only valid CIDRs"},
		{name: "log level", key: "LOG_LEVEL", value: "verbose", message: "LOG_LEVEL must be one of"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := validEnvironment()
			values[tt.key] = tt.value
			_, err := load(mapLookup(values))
			assertErrorContains(t, err, tt.message)
		})
	}
}

func TestLoadRejectsUnsafeProductionSettings(t *testing.T) {
	values := validEnvironment()
	values["APP_ENV"] = EnvironmentProduction
	values["SESSION_COOKIE_SECURE"] = "false"
	values["TEST_SUPPORT_ENABLED"] = "true"

	_, err := load(mapLookup(values))
	assertErrorContains(t, err, "SESSION_COOKIE_SECURE must be true in production")
	assertErrorContains(t, err, "TEST_SUPPORT_ENABLED must be false in production")
}

func TestLoadErrorDoesNotExposeDatabaseURL(t *testing.T) {
	secret := "not-a-postgres-url-with-secret-password"
	values := validEnvironment()
	values["DATABASE_URL"] = secret

	_, err := load(mapLookup(values))
	if err == nil {
		t.Fatal("load() error = nil, want validation error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error exposed DATABASE_URL: %v", err)
	}
}

func validEnvironment() map[string]string {
	return map[string]string{
		"APP_ENV":                  EnvironmentTest,
		"HTTP_ADDR":                "127.0.0.1:8080",
		"DATABASE_URL":             "postgres://tidekeepers:secret@localhost:5432/tidekeepers_test?sslmode=disable",
		"DATABASE_MAX_CONNECTIONS": "12",
		"DATABASE_MIN_CONNECTIONS": "2",
		"DATABASE_ACQUIRE_TIMEOUT": "2s",
		"DATABASE_QUERY_TIMEOUT":   "4s",
		"DATABASE_HEALTH_TIMEOUT":  "1s",
		"SESSION_COOKIE_NAME":      "tidekeepers_session",
		"SESSION_COOKIE_SECURE":    "false",
		"SESSION_TTL":              "720h",
		"RATE_LIMIT_REQUESTS":      "100",
		"RATE_LIMIT_WINDOW":        "1m",
		"HTTP_READ_HEADER_TIMEOUT": "5s",
		"HTTP_READ_TIMEOUT":        "10s",
		"HTTP_WRITE_TIMEOUT":       "15s",
		"HTTP_IDLE_TIMEOUT":        "1m",
		"HTTP_SHUTDOWN_TIMEOUT":    "15s",
		"HTTP_MAX_HEADER_BYTES":    "1048576",
		"LOG_LEVEL":                "info",
		"SERVICE_VERSION":          "test",
		"TEST_SUPPORT_ENABLED":     "true",
	}
}

func mapLookup(values map[string]string) lookupFunc {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("load() error = nil, want %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("load() error = %q, want it to contain %q", err, want)
	}
}
