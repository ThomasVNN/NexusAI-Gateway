package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/repository"
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

// APIKeyAuthenticator implements Authenticator using KeyRepository
type APIKeyAuthenticator struct {
	KeyRepo repository.KeyRepository
	// TestKeys is an allowlist of known test API key hashes for testing environments.
	// If populated, only these specific keys will be accepted in non-production.
	TestKeys map[string]bool
}

// NewAPIKeyAuthenticator creates a new APIKeyAuthenticator
func NewAPIKeyAuthenticator(kr repository.KeyRepository) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		KeyRepo:  kr,
		TestKeys: make(map[string]bool),
	}
}

// AddTestKey adds a test API key to the allowlist (only for testing environments)
func (a *APIKeyAuthenticator) AddTestKey(testKey string) {
	a.TestKeys[HashKey(testKey)] = true
}

// Authenticate verifies the token and returns the user identity
func (a *APIKeyAuthenticator) Authenticate(ctx context.Context, token string) (*repository.UserIdentity, error) {
	rawKey, err := ParseKey(token)
	if err != nil {
		return nil, fmt.Errorf("malformed or missing key: %w", err)
	}

	keyHash := HashKey(rawKey)

	// SECURITY: Check allowlist first (only for explicitly approved test keys)
	if len(a.TestKeys) > 0 {
		if a.TestKeys[keyHash] {
			return &repository.UserIdentity{
				ID:       "test-key",
				TenantID: "test-tenant",
				Roles:    []string{"test"},
			}, nil
		}
		// Test keys configured but this key is not in allowlist - reject
		return nil, fmt.Errorf("invalid API key")
	}

	// Normal authentication flow - look up key in database
	key, err := a.KeyRepo.GetByHash(ctx, keyHash)
	if err != nil {
		isNotFound := err == sql.ErrNoRows || err.Error() == "key not found by hash"
		if isNotFound {
			// SECURITY: Always reject unknown keys - no fallback authentication
			return nil, fmt.Errorf("invalid API key")
		}
		return nil, fmt.Errorf("authentication database failure: %w", err)
	}

	if !key.Active {
		return nil, fmt.Errorf("API key is deactivated")
	}

	tenantID := key.SourceApp
	if tenantID == "" {
		tenantID = "default-tenant"
	}

	return &repository.UserIdentity{
		ID:       key.ID,
		TenantID: tenantID,
	}, nil
}
