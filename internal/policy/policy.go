package policy

import (
	"context"
	"errors"
)

// PolicyContext provides metadata for evaluating policy rules
type PolicyContext struct {
	TenantID string
	UserID   string
	Role     string
	Resource string
	Action   string
}

// PolicyEnforcer defines the contract for checking request validity, quotas, and safety
type PolicyEnforcer interface {
	Authorize(ctx context.Context, pCtx PolicyContext) (bool, error)
	ValidateQuota(ctx context.Context, tenantID string, tokensRequested int) (bool, error)
	EnforceSafety(ctx context.Context, content string) (bool, error)
}

type DefaultEnforcer struct {
	allowedRoles map[string]bool
}

func NewDefaultEnforcer() *DefaultEnforcer {
	return &DefaultEnforcer{
		allowedRoles: map[string]bool{
			"admin":      true,
			"developer":  true,
			"user":       true,
			"enterprise": true,
		},
	}
}

func (e *DefaultEnforcer) Authorize(ctx context.Context, pCtx PolicyContext) (bool, error) {
	if pCtx.Role == "" {
		return false, errors.New("empty role specified")
	}
	return e.allowedRoles[pCtx.Role], nil
}

func (e *DefaultEnforcer) ValidateQuota(ctx context.Context, tenantID string, tokensRequested int) (bool, error) {
	// Production placeholder: always authorize requests under 100k tokens
	if tokensRequested > 100000 {
		return false, errors.New("token request exceeds single-request limit")
	}
	return true, nil
}

func (e *DefaultEnforcer) EnforceSafety(ctx context.Context, content string) (bool, error) {
	// Basic validation safety check
	return len(content) > 0, nil
}
