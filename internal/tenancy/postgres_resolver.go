package tenancy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/db/postgres"
)

// PostgresTenantResolver implements TenantResolver using the database.
type PostgresTenantResolver struct {
	db *postgres.DB
}

// NewPostgresTenantResolver creates a new PostgresTenantResolver.
func NewPostgresTenantResolver(db *postgres.DB) *PostgresTenantResolver {
	return &PostgresTenantResolver{db: db}
}

// Resolve looks up a tenant by slug and returns tenant information.
// Falls back to default tenant if no identifier is provided.
func (r *PostgresTenantResolver) Resolve(ctx context.Context, identifier string) (*Tenant, error) {
	if r.db == nil {
		return nil, errors.New("database connection unavailable")
	}

	// Default tenant fallback for empty identifier
	if identifier == "" {
		return r.resolveByDefault(ctx)
	}

	// Try to resolve by tenant slug first
	tenant, err := r.resolveBySlug(ctx, identifier)
	if err == nil {
		return tenant, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to resolve tenant by slug: %w", err)
	}

	// Try to resolve by tenant ID (UUID)
	tenant, err = r.resolveByID(ctx, identifier)
	if err == nil {
		return tenant, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to resolve tenant by ID: %w", err)
	}

	// Try to resolve by organization slug
	tenant, err = r.resolveByOrganizationSlug(ctx, identifier)
	if err == nil {
		return tenant, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to resolve tenant by org slug: %w", err)
	}

	return nil, fmt.Errorf("tenant not found: %s", identifier)
}

// resolveBySlug resolves a tenant by its slug field.
func (r *PostgresTenantResolver) resolveBySlug(ctx context.Context, slug string) (*Tenant, error) {
	query := `
		SELECT t.id, t.slug, t.display_name, t.environment, t.status, o.slug as org_slug, o.display_name as org_name
		FROM tenants t
		JOIN organizations o ON t.organization_id = o.id
		WHERE t.slug = $1`

	var tenant Tenant
	var orgSlug, orgName string
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&tenant.ID,
		&tenant.Slug,
		&tenant.Name,
		&tenant.Environment,
		&tenant.Status,
		&orgSlug,
		&orgName,
	)
	if err != nil {
		return nil, err
	}
	tenant.OrganizationSlug = orgSlug
	tenant.OrganizationName = orgName
	return &tenant, nil
}

// resolveByID resolves a tenant by its UUID.
func (r *PostgresTenantResolver) resolveByID(ctx context.Context, id string) (*Tenant, error) {
	query := `
		SELECT t.id, t.slug, t.display_name, t.environment, t.status, o.slug as org_slug, o.display_name as org_name
		FROM tenants t
		JOIN organizations o ON t.organization_id = o.id
		WHERE t.id = $1`

	var tenant Tenant
	var orgSlug, orgName string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tenant.ID,
		&tenant.Slug,
		&tenant.Name,
		&tenant.Environment,
		&tenant.Status,
		&orgSlug,
		&orgName,
	)
	if err != nil {
		return nil, err
	}
	tenant.OrganizationSlug = orgSlug
	tenant.OrganizationName = orgName
	return &tenant, nil
}

// resolveByOrganizationSlug resolves a tenant by organization slug (returns first active tenant).
func (r *PostgresTenantResolver) resolveByOrganizationSlug(ctx context.Context, orgSlug string) (*Tenant, error) {
	query := `
		SELECT t.id, t.slug, t.display_name, t.environment, t.status, o.slug as org_slug, o.display_name as org_name
		FROM tenants t
		JOIN organizations o ON t.organization_id = o.id
		WHERE o.slug = $1 AND t.status = 'active'
		ORDER BY t.created_at ASC
		LIMIT 1`

	var tenant Tenant
	var tenantOrgSlug, orgName string
	err := r.db.QueryRowContext(ctx, query, orgSlug).Scan(
		&tenant.ID,
		&tenant.Slug,
		&tenant.Name,
		&tenant.Environment,
		&tenant.Status,
		&tenantOrgSlug,
		&orgName,
	)
	if err != nil {
		return nil, err
	}
	tenant.OrganizationSlug = tenantOrgSlug
	tenant.OrganizationName = orgName
	return &tenant, nil
}

// resolveByDefault resolves the default tenant.
func (r *PostgresTenantResolver) resolveByDefault(ctx context.Context) (*Tenant, error) {
	query := `
		SELECT t.id, t.slug, t.display_name, t.environment, t.status, o.slug as org_slug, o.display_name as org_name
		FROM tenants t
		JOIN organizations o ON t.organization_id = o.id
		WHERE t.slug = 'default' AND t.status = 'active'
		ORDER BY t.created_at ASC
		LIMIT 1`

	var tenant Tenant
	var orgSlug, orgName string
	err := r.db.QueryRowContext(ctx, query).Scan(
		&tenant.ID,
		&tenant.Slug,
		&tenant.Name,
		&tenant.Environment,
		&tenant.Status,
		&orgSlug,
		&orgName,
	)
	if err != nil {
		return nil, err
	}
	tenant.OrganizationSlug = orgSlug
	tenant.OrganizationName = orgName
	return &tenant, nil
}
