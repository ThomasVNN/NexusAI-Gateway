// Package tenant provides multi-tenant isolation middleware and context utilities.
package tenant

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/tenancy"
)

// Middleware extracts tenant context from incoming requests and injects it into the context.
type Middleware struct {
	resolver tenancy.TenantResolver
	header   string
	param    string
	allowNil bool
}

// MiddlewareOption configures the tenant middleware.
type MiddlewareOption func(*Middleware)

// WithHeader specifies which request header contains the tenant identifier.
// Default: "X-Tenant-ID"
func WithHeader(name string) MiddlewareOption {
	return func(m *Middleware) { m.header = name }
}

// WithURLParam specifies which URL parameter contains the tenant identifier.
// Default: "tenant_id"
func WithURLParam(name string) MiddlewareOption {
	return func(m *Middleware) { m.param = name }
}

// AllowNil permits requests without tenant context (uses default tenant).
func AllowNil() MiddlewareOption {
	return func(m *Middleware) { m.allowNil = true }
}

// NewMiddleware creates a tenant extraction middleware using the provided resolver.
func NewMiddleware(resolver tenancy.TenantResolver, opts ...MiddlewareOption) *Middleware {
	m := &Middleware{
		resolver: resolver,
		header:   "X-Tenant-ID",
		param:    "tenant_id",
		allowNil: false,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Handler returns an HTTP middleware function that extracts and validates tenant context.
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Try header first
		identifier := r.Header.Get(m.header)

		// Fall back to URL param
		if identifier == "" {
			identifier = r.URL.Query().Get(m.param)
		}

		// Fall back to subdomain
		if identifier == "" {
			identifier = extractSubdomain(r.Host)
		}

		// Resolve tenant
		tenant, err := m.resolver.Resolve(ctx, identifier)
		if err != nil {
			if m.allowNil {
				tenant = &tenancy.Tenant{
					ID:     "default",
					Slug:   "default",
					Status: "active",
					Plan:   "standard",
					IsActive: true,
				}
			} else {
				http.Error(w, `{"error":"tenant not found"}`, http.StatusUnauthorized)
				return
			}
		}

		// Validate tenant state
		if err := tenant.Validate(); err != nil {
			http.Error(w, `{"error":"tenant not active"}`, http.StatusForbidden)
			return
		}

		// Inject tenant into context
		ctx = tenancy.WithTenant(ctx, tenant)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractSubdomain extracts the first segment from a host as tenant identifier.
func extractSubdomain(host string) string {
	host = strings.Split(host, ":")[0] // strip port
	parts := strings.Split(host, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// ScopedContext holds tenant-scoped data for query isolation.
type ScopedContext struct {
	TenantID        string
	OrganizationID   string
	WorkspaceID     string
	UserID          string
	DataIsolation   IsolationLevel
}

type IsolationLevel int

const (
	IsolationPerTenant IsolationLevel = iota
	IsolationPerWorkspace
	IsolationPerUser
)

// Scoper wraps operations with tenant scoping.
type Scoper interface {
	Scope(ctx context.Context) ScopedContext
	Filter(resource, tenantID string) string
}

// QueryFilter adds tenant_id conditions to SQL queries.
type QueryFilter struct {
	mu     sync.RWMutex
	tenant map[string]string // query -> tenant mapping
}

// NewQueryFilter creates a query filter for tenant-scoped queries.
func NewQueryFilter() *QueryFilter {
	return &QueryFilter{
		tenant: make(map[string]string),
	}
}

// FilterQuery applies tenant filtering to a raw query.
func (f *QueryFilter) FilterQuery(query, tenantID string) (string, error) {
	if tenantID == "" {
		return "", errors.New("tenant ID required for filtering")
	}

	// Simple injection guard: only append to SELECT/WHERE if not already present
	lc := strings.ToLower(query)
	if !strings.Contains(lc, "tenant_id") {
		// Add tenant filter
		if strings.Contains(lc, "where") {
			return query + " AND tenant_id = '" + tenantID + "'", nil
		}
		if strings.Contains(lc, "from") {
			return query + " WHERE tenant_id = '" + tenantID + "'", nil
		}
	}
	return query, nil
}

// Isolate creates a scoped context from a standard context.
func Isolate(ctx context.Context) ScopedContext {
	tenant, err := tenancy.GetTenant(ctx)
	if err != nil {
		return ScopedContext{}
	}
	return ScopedContext{
		TenantID:      tenant.ID,
		OrganizationID: tenant.OrganizationID,
		DataIsolation: IsolationPerTenant,
	}
}
