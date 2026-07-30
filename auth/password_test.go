package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("sufficiently-long-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Fatalf("HashPassword() = %q, want encoded Argon2id hash", hash)
	}
	if !VerifyPassword(hash, "sufficiently-long-password") {
		t.Fatal("VerifyPassword() = false, want true")
	}
	if VerifyPassword(hash, "different-long-password") {
		t.Fatal("VerifyPassword() = true for incorrect password")
	}
	if VerifyPassword("not-a-hash", "sufficiently-long-password") {
		t.Fatal("VerifyPassword() = true for malformed hash")
	}
}

func TestNormalizeCredentials(t *testing.T) {
	email, err := normalizeCredentials("  Player@Example.COM ", "sufficiently-long-password")
	if err != nil {
		t.Fatalf("normalizeCredentials() error = %v", err)
	}
	if email != "player@example.com" {
		t.Errorf("normalized email = %q, want player@example.com", email)
	}

	for _, test := range []struct {
		email    string
		password string
		caseName string
	}{
		{email: "not-an-email", password: "sufficiently-long-password", caseName: "invalid email"},
		{email: "player@example.com", password: "short", caseName: "short password"},
		{email: "player@example.com", password: strings.Repeat("a", maxPasswordLength+1), caseName: "long password"},
	} {
		t.Run(test.caseName, func(t *testing.T) {
			if _, err := normalizeCredentials(test.email, test.password); err != ErrInvalidInput {
				t.Fatalf("normalizeCredentials() error = %v, want ErrInvalidInput", err)
			}
		})
	}
}
