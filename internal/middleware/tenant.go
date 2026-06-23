// Package middleware provides HTTP middleware for the NexusAI Gateway.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/tenancy"
)

// TenantConfig configures tenant extraction behavior.
type TenantConfig struct {
	Header        string // HTTP header to check (default: X-Tenant-ID)
	URLParam      string // URL query param (default: tenant_id)
	Subdomain     bool   // Extract from subdomain
	AllowDefault  bool   // Use default tenant if none found
}

// DefaultTenantConfig returns sensible defaults for tenant extraction.
func DefaultTenantConfig() TenantConfig {
	return TenantConfig{
		Header:       "X-Tenant-ID",
		URLParam:     "tenant_id",
		Subdomain:    true,
		AllowDefault: true,
	}
}

// TenantExtractor creates a middleware that extracts tenant context from requests.
func TenantExtractor(resolver tenancy.TenantResolver, cfg TenantConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			identifier := ""

			// Extract from header
			if identifier == "" {
				identifier = r.Header.Get(cfg.Header)
			}

			// Extract from URL param
			if identifier == "" {
				identifier = r.URL.Query().Get(cfg.URLParam)
			}

			// Extract from subdomain
			if identifier == "" && cfg.Subdomain {
				identifier = extractSubdomain(r.Host)
			}

			// Resolve tenant
			tenant, err := resolver.Resolve(ctx, identifier)
			if err != nil {
				if !cfg.AllowDefault {
					http.Error(w, `{"error":"tenant required"}`, http.StatusUnauthorized)
					return
				}
				// Fall back to default
				defaultResolver := tenancy.NewDefaultTenantResolver()
				tenant, _ = defaultResolver.Resolve(ctx, "")
			}

			// Validate
			if err := tenant.Validate(); err != nil {
				http.Error(w, `{"error":"tenant inactive"}`, http.StatusForbidden)
				return
			}

			// Inject into context
			ctx = tenancy.WithTenant(ctx, tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractSubdomain pulls the first label from a host.
func extractSubdomain(host string) string {
	host = strings.Split(host, ":")[0]
	parts := strings.Split(host, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// GetTenantContext extracts tenant info from context, if present.
func GetTenantContext(ctx context.Context) (tenantID, orgID string, ok bool) {
	t, err := tenancy.GetTenant(ctx)
	if err != nil {
		return "", "", false
	}
	return t.ID, t.OrganizationID, true
}
