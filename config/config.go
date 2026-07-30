// Package config loads and validates API configuration from the environment.
package config

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Supported application environments.
const (
	EnvironmentLocal       = "local"
	EnvironmentTest        = "test"
	EnvironmentStaging     = "staging"
	EnvironmentProduction  = "production"
	defaultMaxHeaderBytes  = 1 << 20
	defaultSessionCookie   = "tidekeepers_session"
	defaultSessionTTL      = 30 * 24 * time.Hour
	defaultDatabaseMaxConn = 20
)

// Config is the startup-validated API configuration.
type Config struct {
	Environment       string
	HTTP              HTTP
	Database          Database
	Session           Session
	RateLimit         RateLimit
	TrustedProxyCIDRs []netip.Prefix
	LogLevel          string
	ServiceVersion    string
	TestSupport       bool
}

// HTTP contains listener and lifecycle limits.
type HTTP struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	MaxHeaderBytes    int
}

// Database contains PostgreSQL pool and operation limits.
type Database struct {
	URL            string
	MaxConnections int32
	MinConnections int32
	AcquireTimeout time.Duration
	QueryTimeout   time.Duration
	HealthTimeout  time.Duration
}

// Session contains opaque cookie settings. Raw session values are never configuration.
type Session struct {
	CookieName   string
	CookieSecure bool
	CookieDomain string
	TTL          time.Duration
}

// RateLimit contains the process-local HTTP request limit.
type RateLimit struct {
	Requests int
	Window   time.Duration
}

// Load reads and validates configuration from the process environment.
func Load() (Config, error) {
	return load(os.LookupEnv)
}

type lookupFunc func(string) (string, bool)

type reader struct {
	lookup lookupFunc
	errs   []error
}

func load(lookup lookupFunc) (Config, error) {
	r := &reader{lookup: lookup}

	cfg := Config{
		Environment: r.required("APP_ENV"),
		HTTP: HTTP{
			Address:           r.value("HTTP_ADDR", ":8080"),
			ReadHeaderTimeout: r.duration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
			ReadTimeout:       r.duration("HTTP_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:      r.duration("HTTP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:       r.duration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout:   r.duration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
			MaxHeaderBytes:    r.integer("HTTP_MAX_HEADER_BYTES", defaultMaxHeaderBytes),
		},
		Database: Database{
			URL:            r.required("DATABASE_URL"),
			MaxConnections: int32(r.integer("DATABASE_MAX_CONNECTIONS", defaultDatabaseMaxConn)),
			MinConnections: int32(r.integer("DATABASE_MIN_CONNECTIONS", 2)),
			AcquireTimeout: r.duration("DATABASE_ACQUIRE_TIMEOUT", 2*time.Second),
			QueryTimeout:   r.duration("DATABASE_QUERY_TIMEOUT", 3*time.Second),
			HealthTimeout:  r.duration("DATABASE_HEALTH_TIMEOUT", time.Second),
		},
		Session: Session{
			CookieName:   r.value("SESSION_COOKIE_NAME", defaultSessionCookie),
			CookieSecure: r.boolean("SESSION_COOKIE_SECURE", true),
			CookieDomain: strings.TrimSpace(r.value("SESSION_COOKIE_DOMAIN", "")),
			TTL:          r.duration("SESSION_TTL", defaultSessionTTL),
		},
		RateLimit: RateLimit{
			Requests: r.integer("RATE_LIMIT_REQUESTS", 120),
			Window:   r.duration("RATE_LIMIT_WINDOW", time.Minute),
		},
		LogLevel:       strings.ToLower(r.value("LOG_LEVEL", "info")),
		ServiceVersion: r.value("SERVICE_VERSION", "development"),
		TestSupport:    r.boolean("TEST_SUPPORT_ENABLED", false),
	}

	cfg.Environment = strings.ToLower(strings.TrimSpace(cfg.Environment))
	cfg.TrustedProxyCIDRs = r.cidrs("TRUSTED_PROXY_CIDRS")
	r.validate(cfg)

	if len(r.errs) > 0 {
		return Config{}, errors.Join(r.errs...)
	}
	return cfg, nil
}

func (r *reader) required(name string) string {
	value, ok := r.lookup(name)
	if !ok || strings.TrimSpace(value) == "" {
		r.errs = append(r.errs, fmt.Errorf("%s is required", name))
		return ""
	}
	return strings.TrimSpace(value)
}

func (r *reader) value(name, fallback string) string {
	value, ok := r.lookup(name)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func (r *reader) integer(name string, fallback int) int {
	value, ok := r.lookup(name)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s must be an integer", name))
		return fallback
	}
	return parsed
}

func (r *reader) duration(name string, fallback time.Duration) time.Duration {
	value, ok := r.lookup(name)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s must be a duration", name))
		return fallback
	}
	return parsed
}

func (r *reader) boolean(name string, fallback bool) bool {
	value, ok := r.lookup(name)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s must be a boolean", name))
		return fallback
	}
	return parsed
}

func (r *reader) cidrs(name string) []netip.Prefix {
	value, ok := r.lookup(name)
	if !ok || strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	prefixes := make([]netip.Prefix, 0, len(parts))
	for _, part := range parts {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(part))
		if err != nil {
			r.errs = append(r.errs, fmt.Errorf("%s must contain only valid CIDRs", name))
			return nil
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes
}

func (r *reader) validate(cfg Config) {
	switch cfg.Environment {
	case EnvironmentLocal, EnvironmentTest, EnvironmentStaging, EnvironmentProduction:
	default:
		r.errs = append(r.errs, fmt.Errorf("APP_ENV must be one of local, test, staging, production"))
	}

	if _, err := net.ResolveTCPAddr("tcp", cfg.HTTP.Address); err != nil {
		r.errs = append(r.errs, fmt.Errorf("HTTP_ADDR must be a valid TCP address"))
	}
	positiveDuration(&r.errs, "HTTP_READ_HEADER_TIMEOUT", cfg.HTTP.ReadHeaderTimeout)
	positiveDuration(&r.errs, "HTTP_READ_TIMEOUT", cfg.HTTP.ReadTimeout)
	positiveDuration(&r.errs, "HTTP_WRITE_TIMEOUT", cfg.HTTP.WriteTimeout)
	positiveDuration(&r.errs, "HTTP_IDLE_TIMEOUT", cfg.HTTP.IdleTimeout)
	positiveDuration(&r.errs, "HTTP_SHUTDOWN_TIMEOUT", cfg.HTTP.ShutdownTimeout)
	if cfg.HTTP.MaxHeaderBytes < 1024 {
		r.errs = append(r.errs, fmt.Errorf("HTTP_MAX_HEADER_BYTES must be at least 1024"))
	}

	if cfg.Database.URL != "" {
		parsed, err := url.Parse(cfg.Database.URL)
		if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" {
			r.errs = append(r.errs, fmt.Errorf("DATABASE_URL must be a valid PostgreSQL URL"))
		}
	}
	if cfg.Database.MaxConnections < 1 {
		r.errs = append(r.errs, fmt.Errorf("DATABASE_MAX_CONNECTIONS must be positive"))
	}
	if cfg.Database.MinConnections < 0 {
		r.errs = append(r.errs, fmt.Errorf("DATABASE_MIN_CONNECTIONS must not be negative"))
	}
	if cfg.Database.MinConnections > cfg.Database.MaxConnections {
		r.errs = append(r.errs, fmt.Errorf("DATABASE_MIN_CONNECTIONS must not exceed DATABASE_MAX_CONNECTIONS"))
	}
	positiveDuration(&r.errs, "DATABASE_ACQUIRE_TIMEOUT", cfg.Database.AcquireTimeout)
	positiveDuration(&r.errs, "DATABASE_QUERY_TIMEOUT", cfg.Database.QueryTimeout)
	positiveDuration(&r.errs, "DATABASE_HEALTH_TIMEOUT", cfg.Database.HealthTimeout)

	if !validCookieName(cfg.Session.CookieName) {
		r.errs = append(r.errs, fmt.Errorf("SESSION_COOKIE_NAME must be a valid cookie name"))
	}
	if strings.ContainsAny(cfg.Session.CookieDomain, " /\t\r\n") {
		r.errs = append(r.errs, fmt.Errorf("SESSION_COOKIE_DOMAIN must be a hostname"))
	}
	positiveDuration(&r.errs, "SESSION_TTL", cfg.Session.TTL)
	if cfg.Environment == EnvironmentProduction && !cfg.Session.CookieSecure {
		r.errs = append(r.errs, fmt.Errorf("SESSION_COOKIE_SECURE must be true in production"))
	}
	if cfg.Environment == EnvironmentProduction && cfg.TestSupport {
		r.errs = append(r.errs, fmt.Errorf("TEST_SUPPORT_ENABLED must be false in production"))
	}

	if cfg.RateLimit.Requests < 1 {
		r.errs = append(r.errs, fmt.Errorf("RATE_LIMIT_REQUESTS must be positive"))
	}
	positiveDuration(&r.errs, "RATE_LIMIT_WINDOW", cfg.RateLimit.Window)

	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		r.errs = append(r.errs, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error"))
	}
	if strings.TrimSpace(cfg.ServiceVersion) == "" {
		r.errs = append(r.errs, fmt.Errorf("SERVICE_VERSION must not be empty"))
	}
}

func positiveDuration(errs *[]error, name string, value time.Duration) {
	if value <= 0 {
		*errs = append(*errs, fmt.Errorf("%s must be positive", name))
	}
}

func validCookieName(value string) bool {
	if value == "" {
		return false
	}
	const separators = "()<>@,;:\\\"/[]?={} \t\r\n"
	for _, char := range value {
		if char < 0x21 || char > 0x7e || strings.ContainsRune(separators, char) {
			return false
		}
	}
	return true
}
