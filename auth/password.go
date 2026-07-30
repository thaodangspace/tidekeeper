package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	minPasswordLength = 8
	maxPasswordLength = 128
	argonTime         = 3
	argonMemory       = 64 * 1024
	argonThreads      = 4
	argonKeyLength    = 32
	saltLength        = 16
)

var errInvalidPasswordHash = errors.New("invalid password hash")

// HashPassword derives an encoded Argon2id credential hash. The caller must
// discard the plaintext password as soon as this method returns.
func HashPassword(password string) (string, error) {
	if !validPassword(password) {
		return "", ErrInvalidInput
	}

	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	defer clear(salt)

	derived := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLength)
	defer clear(derived)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(derived)), nil
}

// VerifyPassword compares a plaintext password with an encoded Argon2id hash.
// Malformed stored hashes deliberately compare as invalid credentials.
func VerifyPassword(encoded, password string) bool {
	memory, timeCost, threads, salt, expected, err := parsePasswordHash(encoded)
	if err != nil || !validPassword(password) {
		return false
	}
	defer clear(salt)
	defer clear(expected)

	actual := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, uint32(len(expected)))
	defer clear(actual)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func validPassword(password string) bool {
	length := len([]rune(password))
	return length >= minPasswordLength && length <= maxPasswordLength
}

func parsePasswordHash(encoded string) (uint32, uint32, uint8, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		return 0, 0, 0, nil, nil, errInvalidPasswordHash
	}

	var memory, timeCost uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeCost, &threads); err != nil ||
		memory != argonMemory || timeCost != argonTime || threads != argonThreads {
		return 0, 0, 0, nil, nil, errInvalidPasswordHash
	}
	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(salt) != saltLength {
		clear(salt)
		return 0, 0, 0, nil, nil, errInvalidPasswordHash
	}
	expected, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(expected) != argonKeyLength {
		clear(salt)
		clear(expected)
		return 0, 0, 0, nil, nil, errInvalidPasswordHash
	}
	return memory, timeCost, threads, salt, expected, nil
}
