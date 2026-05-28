package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

// HashKey computes the SHA-256 hash of an API key
func HashKey(key string) string {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

// ParseKey extracts the token part from a bearer header or direct key
func ParseKey(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("missing API key")
	}

	parts := strings.Split(authHeader, " ")
	key := authHeader
	if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
		key = parts[1]
	}

	if !strings.HasPrefix(key, "ork_") {
		return "", errors.New("invalid API key format, must start with 'ork_'")
	}

	return key, nil
}
