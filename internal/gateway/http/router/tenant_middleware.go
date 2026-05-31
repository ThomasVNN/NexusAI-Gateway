package router

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/auth"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/gateway/http/handler"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/tenancy"
)

type tenantMiddleware struct {
	resolver      tenancy.TenantResolver
	authenticator auth.Authenticator
}

// WithTenantResolution injects tenant context into the request.
func WithTenantResolution(resolver tenancy.TenantResolver, authenticator auth.Authenticator) func(http.Handler) http.Handler {
	tm := &tenantMiddleware{
		resolver:      resolver,
		authenticator: authenticator,
	}
	return tm.middleware
}

func (tm *tenantMiddleware) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corrID := GetCorrelationID(r.Context())

		// 1. Authenticate the request to get tenant from identity
		var tenantID string
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			identity, err := tm.authenticator.Authenticate(r.Context(), authHeader)
			if err != nil {
				slog.WarnContext(r.Context(), "Tenant middleware: authentication failed",
					slog.String("correlation_id", corrID),
					slog.Any("error", err),
				)
				// Continue without auth - middleware will handle auth errors later
			} else {
				tenantID = identity.TenantID
			}
		}

		// 2. Allow X-Tenant-ID header override for multi-tenant scenarios
		// This should only be allowed for admin tokens
		if headerTenant := r.Header.Get("X-Tenant-ID"); headerTenant != "" {
			// In production, validate that the authenticated user has admin rights
			// For now, allow override if auth succeeded
			if tenantID != "" {
				slog.DebugContext(r.Context(), "Tenant middleware: header override applied",
					slog.String("correlation_id", corrID),
					slog.String("original_tenant", tenantID),
					slog.String("override_tenant", headerTenant),
				)
				tenantID = headerTenant
			}
		}

		// 3. Resolve tenant from identifier
		tenant, err := tm.resolver.Resolve(r.Context(), tenantID)
		if err != nil {
			slog.WarnContext(r.Context(), "Tenant middleware: tenant resolution failed",
				slog.String("correlation_id", corrID),
				slog.String("tenant_identifier", tenantID),
				slog.Any("error", err),
			)
			handler.WriteError(w, http.StatusUnauthorized, "TENANT_NOT_FOUND", "Tenant not found or inactive")
			return
		}

		// 4. Validate tenant state
		if err := tenant.Validate(); err != nil {
			slog.WarnContext(r.Context(), "Tenant middleware: tenant validation failed",
				slog.String("correlation_id", corrID),
				slog.String("tenant_id", tenant.ID),
				slog.Any("error", err),
			)
			handler.WriteError(w, http.StatusForbidden, "TENANT_INACTIVE", "Tenant is not active or suspended")
			return
		}

		// 5. Inject tenant context
		ctx := tenancy.WithTenant(r.Context(), tenant)
		slog.DebugContext(r.Context(), "Tenant middleware: tenant context injected",
			slog.String("correlation_id", corrID),
			slog.String("tenant_id", tenant.ID),
			slog.String("tenant_slug", tenant.Slug),
			slog.String("tenant_org", tenant.OrganizationSlug),
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetTenantFromRequest extracts tenant from request context.
func GetTenantFromRequest(r *http.Request) (*tenancy.Tenant, error) {
	return tenancy.GetTenant(r.Context())
}

// RequireTenant ensures a tenant context is present in the request.
func RequireTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant, err := tenancy.GetTenant(r.Context())
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				handler.WriteError(w, http.StatusGatewayTimeout, "TENANT_TIMEOUT", "Tenant resolution timed out")
				return
			}
			handler.WriteError(w, http.StatusUnauthorized, "TENANT_REQUIRED", "Tenant context is required")
			return
		}
		if err := tenant.Validate(); err != nil {
			handler.WriteError(w, http.StatusForbidden, "TENANT_INVALID", "Tenant validation failed")
			return
		}
		next.ServeHTTP(w, r)
	})
}
