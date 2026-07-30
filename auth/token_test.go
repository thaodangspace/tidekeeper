package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"testing"
)

func TestDigestToken(t *testing.T) {
	rawBytes := make([]byte, tokenBytes)
	for index := range rawBytes {
		rawBytes[index] = byte(index)
	}
	raw := base64.RawURLEncoding.EncodeToString(rawBytes)

	got, err := DigestToken(raw)
	if err != nil {
		t.Fatalf("DigestToken() error = %v", err)
	}
	want := sha256.Sum256(rawBytes)
	if got != want {
		t.Fatalf("DigestToken() = %x, want %x", got, want)
	}

	copyOfDigest := got.Bytes()
	copyOfDigest[0] ^= 0xff
	if got != want {
		t.Fatal("Bytes() returned storage alias")
	}
}

func TestNewToken(t *testing.T) {
	raw, digest, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken() error = %v", err)
	}
	if len(raw) != base64.RawURLEncoding.EncodedLen(tokenBytes) {
		t.Errorf("raw token length = %d, want %d", len(raw), base64.RawURLEncoding.EncodedLen(tokenBytes))
	}
	got, err := DigestToken(raw)
	if err != nil {
		t.Fatalf("DigestToken(NewToken()) error = %v", err)
	}
	if got != digest {
		t.Errorf("NewToken digest = %x, want %x", digest, got)
	}
}

func TestDigestTokenRejectsInvalidTokens(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "empty", raw: ""},
		{name: "not base64url", raw: "contains+standard/slashes=="},
		{name: "too short", raw: base64.RawURLEncoding.EncodeToString(make([]byte, tokenBytes-1))},
		{name: "too long", raw: base64.RawURLEncoding.EncodeToString(make([]byte, tokenBytes+1))},
		{name: "padded", raw: base64.URLEncoding.EncodeToString(make([]byte, tokenBytes))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DigestToken(tt.raw)
			if !errors.Is(err, errInvalidToken) {
				t.Fatalf("DigestToken() error = %v, want invalid token", err)
			}
		})
	}
}
