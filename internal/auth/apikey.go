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
	KeyRepo               repository.KeyRepository
	EnableSandboxFallback bool
}

// NewAPIKeyAuthenticator creates a new APIKeyAuthenticator
func NewAPIKeyAuthenticator(kr repository.KeyRepository, enableSandboxFallback bool) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		KeyRepo:               kr,
		EnableSandboxFallback: enableSandboxFallback,
	}
}

// Authenticate verifies the token and returns the user identity
func (a *APIKeyAuthenticator) Authenticate(ctx context.Context, token string) (*UserIdentity, error) {
	rawKey, err := ParseKey(token)
	if err != nil {
		return nil, fmt.Errorf("malformed or missing key: %w", err)
	}

	keyHash := HashKey(rawKey)
	key, err := a.KeyRepo.GetByHash(ctx, keyHash)
	if err != nil {
		isNotFound := err == sql.ErrNoRows || err.Error() == "key not found by hash"
		if isNotFound {
			if a.EnableSandboxFallback {
				return &UserIdentity{
					ID:       "mock-local-key",
					TenantID: "default-sandbox-tenant",
					Roles:    []string{"sandbox"},
				}, nil
			}
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

	return &UserIdentity{
		ID:       key.ID,
		TenantID: tenantID,
	}, nil
}
