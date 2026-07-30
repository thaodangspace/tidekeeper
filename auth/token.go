package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

const tokenBytes = 32

var errInvalidToken = errors.New("invalid opaque session token")

// Digest is the SHA-256 digest persisted and used for session lookup.
type Digest [sha256.Size]byte

// NewToken generates a 256-bit unpadded base64url token and its storage digest.
// The raw token is suitable only for setting in the session cookie.
func NewToken() (string, Digest, error) {
	bytes := make([]byte, tokenBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", Digest{}, fmt.Errorf("generate session token: %w", err)
	}
	defer clear(bytes)

	return base64.RawURLEncoding.EncodeToString(bytes), sha256.Sum256(bytes), nil
}

// DigestToken validates and hashes a 256-bit unpadded base64url token.
func DigestToken(raw string) (Digest, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || len(decoded) != tokenBytes {
		clear(decoded)
		return Digest{}, errInvalidToken
	}
	defer clear(decoded)

	return sha256.Sum256(decoded), nil
}

// Bytes returns an independent digest byte slice suitable for a query argument.
func (d Digest) Bytes() []byte {
	result := make([]byte, len(d))
	copy(result, d[:])
	return result
}
