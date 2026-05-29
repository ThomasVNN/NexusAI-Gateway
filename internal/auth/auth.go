package auth

import (
	"context"
	"errors"
)

// UserIdentity represents the authenticated entity
type UserIdentity struct {
	ID          string
	TenantID    string
	Roles       []string
	Permissions []string
}

// Authenticator defines the contract for verifying credentials
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*UserIdentity, error)
}

type contextKey string

const IdentityKey contextKey = "user_identity"

// WithIdentity returns a new context with the user identity injected
func WithIdentity(ctx context.Context, id *UserIdentity) context.Context {
	return context.WithValue(ctx, IdentityKey, id)
}

// GetIdentity retrieves the user identity from context
func GetIdentity(ctx context.Context) (*UserIdentity, error) {
	if ctx == nil {
		return nil, errors.New("nil context")
	}
	id, ok := ctx.Value(IdentityKey).(*UserIdentity)
	if !ok {
		return nil, errors.New("identity not found in context")
	}
	return id, nil
}
