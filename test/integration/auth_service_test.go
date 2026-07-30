//go:build integration

package integration_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/auth"
)

func TestAuthenticationService(t *testing.T) {
	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	t.Cleanup(func() { conn.Close(ctx) })

	schema := "auth_service_test_" + authRandomHex(t)
	mustExec(t, ctx, conn, "CREATE SCHEMA "+schema)
	t.Cleanup(func() { _, _ = conn.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE") })
	mustExec(t, ctx, conn, "SET search_path TO "+schema)
	applyAuthMigration(t, ctx, conn, "000001_foundation.up.sql")
	applyAuthMigration(t, ctx, conn, "000003_auth_credentials.up.sql")

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse pool config: %v", err)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)

	service := auth.NewService(pool, time.Hour)
	registered, err := service.Register(ctx, " Player@Example.COM ", "a-long-password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if registered.Token == "" || !registered.ExpiresAt.After(time.Now()) {
		t.Fatalf("Register() session = %#v, want token and future expiry", registered)
	}

	var email string
	if err := pool.QueryRow(ctx, "SELECT email FROM accounts").Scan(&email); err != nil {
		t.Fatalf("query normalized email: %v", err)
	}
	if email != "player@example.com" {
		t.Errorf("stored email = %q, want player@example.com", email)
	}

	if _, err := service.Register(ctx, "player@example.com", "a-long-password"); !errors.Is(err, auth.ErrEmailTaken) {
		t.Fatalf("duplicate Register() error = %v, want ErrEmailTaken", err)
	}
	if _, err := service.Login(ctx, "player@example.com", "wrong-password"); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("invalid Login() error = %v, want ErrInvalidCredentials", err)
	}

	loggedIn, err := service.Login(ctx, "PLAYER@EXAMPLE.COM", "a-long-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if loggedIn.Token == registered.Token {
		t.Error("Login() reissued the registration token")
	}
	digest, err := auth.DigestToken(loggedIn.Token)
	if err != nil {
		t.Fatalf("DigestToken() error = %v", err)
	}
	var sessions int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM sessions WHERE token_digest = $1", digest.Bytes()).Scan(&sessions); err != nil {
		t.Fatalf("query issued session: %v", err)
	}
	if sessions != 1 {
		t.Errorf("issued login sessions = %d, want 1", sessions)
	}
}

func applyAuthMigration(t *testing.T, ctx context.Context, conn *pgx.Conn, filename string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "db", "migrations", filename))
	if err != nil {
		t.Fatalf("read migration %s: %v", filename, err)
	}
	mustExec(t, ctx, conn, string(content))
}

func authRandomHex(t *testing.T) string {
	t.Helper()
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		t.Fatalf("generate random schema suffix: %v", err)
	}
	return hex.EncodeToString(bytes)
}
