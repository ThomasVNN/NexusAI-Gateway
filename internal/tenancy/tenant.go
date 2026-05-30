package tenancy

import (
	"context"
	"errors"
)

// Tenant represents an isolated business unit with organization membership.
type Tenant struct {
	ID                 string
	Slug               string
	Name               string
	Environment        string // "local", "development", "staging", "production"
	Status             string // "active", "suspended", "archived"
	Plan               string // e.g. "free", "standard", "enterprise"
	IsActive           bool
	Settings           map[string]string
	OrganizationID     string
	OrganizationSlug   string
	OrganizationName   string
}

// Validate checks if the tenant is in a valid state for processing requests.
func (t *Tenant) Validate() error {
	if t == nil {
		return errors.New("nil tenant")
	}
	if t.ID == "" {
		return errors.New("tenant ID is required")
	}
	if t.Slug == "" {
		return errors.New("tenant slug is required")
	}
	if t.Status != "active" {
		return errors.New("tenant is not active")
	}
	if !t.IsActive {
		return errors.New("tenant is deactivated")
	}
	return nil
}

// IsEnterprise returns true if the tenant is on an enterprise plan.
func (t *Tenant) IsEnterprise() bool {
	return t.Plan == "enterprise"
}

// IsProduction returns true if the tenant is running in production environment.
func (t *Tenant) IsProduction() bool {
	return t.Environment == "production"
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

// DefaultTenantResolver implements TenantResolver with in-memory default tenant.
type DefaultTenantResolver struct{}

// NewDefaultTenantResolver creates a new DefaultTenantResolver.
func NewDefaultTenantResolver() *DefaultTenantResolver {
	return &DefaultTenantResolver{}
}

// Resolve identifies and returns the default tenant context.
// This resolver should only be used when database is unavailable.
func (r *DefaultTenantResolver) Resolve(ctx context.Context, identifier string) (*Tenant, error) {
	if identifier == "" {
		identifier = "default"
	}
	tenant := &Tenant{
		ID:               identifier,
		Slug:             identifier,
		Name:             "Tenant " + identifier,
		Environment:      "development",
		Status:           "active",
		Plan:             "enterprise",
		IsActive:         true,
		OrganizationSlug: "default-org",
		OrganizationName: "Default Organization",
		Settings:         map[string]string{},
	}
	return tenant, tenant.Validate()
}
