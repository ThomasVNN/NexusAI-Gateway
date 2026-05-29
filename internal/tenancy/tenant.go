package tenancy

import (
	"context"
	"errors"
)

// Tenant represents an isolated business unit
type Tenant struct {
	ID       string
	Name     string
	Plan     string // e.g. "free", "standard", "enterprise"
	IsActive bool
	Settings map[string]string
}

// TenantResolver defines the contract for identifying a tenant from a request
type TenantResolver interface {
	Resolve(ctx context.Context, identifier string) (*Tenant, error)
}

type contextKey string

const TenantKey contextKey = "tenant_context"

// WithTenant injects tenant information into the context
func WithTenant(ctx context.Context, tenant *Tenant) context.Context {
	return context.WithValue(ctx, TenantKey, tenant)
}

// GetTenant retrieves the tenant information from the context
func GetTenant(ctx context.Context) (*Tenant, error) {
	if ctx == nil {
		return nil, errors.New("nil context")
	}
	t, ok := ctx.Value(TenantKey).(*Tenant)
	if !ok {
		return nil, errors.New("tenant context not found")
	}
	return t, nil
}

// DefaultTenantResolver implements TenantResolver
type DefaultTenantResolver struct{}

// NewDefaultTenantResolver creates a new DefaultTenantResolver
func NewDefaultTenantResolver() *DefaultTenantResolver {
	return &DefaultTenantResolver{}
}

// Resolve identifies and returns the tenant context
func (r *DefaultTenantResolver) Resolve(ctx context.Context, identifier string) (*Tenant, error) {
	if identifier == "" {
		identifier = "default-tenant"
	}
	return &Tenant{
		ID:       identifier,
		Name:     "Tenant " + identifier,
		Plan:     "enterprise",
		IsActive: true,
		Settings: map[string]string{},
	}, nil
}
