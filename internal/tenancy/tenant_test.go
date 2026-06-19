package tenancy

import (
	"context"
	"testing"
)

func TestDefaultTenantResolver_Resolve(t *testing.T) {
	resolver := NewDefaultTenantResolver()

	tests := []struct {
		name       string
		identifier string
		wantID     string
		wantErr    bool
	}{
		{
			name:       "resolve with identifier",
			identifier: "acme-corp",
			wantID:     "acme-corp",
			wantErr:    false,
		},
		{
			name:       "resolve with empty identifier uses default",
			identifier: "",
			wantID:     "default",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tenant, err := resolver.Resolve(ctx, tt.identifier)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tenant != nil && tenant.ID != tt.wantID {
				t.Errorf("Resolve() tenant.ID = %v, want %v", tenant.ID, tt.wantID)
			}
		})
	}
}

func TestTenant_Validate(t *testing.T) {
	tests := []struct {
		name    string
		tenant  *Tenant
		wantErr bool
	}{
		{
			name: "valid tenant",
			tenant: &Tenant{
				ID:       "tenant-123",
				Slug:     "acme",
				Status:   "active",
				IsActive: true,
			},
			wantErr: false,
		},
		{
			name:    "nil tenant",
			tenant:  nil,
			wantErr: true,
		},
		{
			name: "missing ID",
			tenant: &Tenant{
				Slug:     "acme",
				Status:   "active",
				IsActive: true,
			},
			wantErr: true,
		},
		{
			name: "missing slug",
			tenant: &Tenant{
				ID:       "tenant-123",
				Status:   "active",
				IsActive: true,
			},
			wantErr: true,
		},
		{
			name: "inactive tenant",
			tenant: &Tenant{
				ID:       "tenant-123",
				Slug:     "acme",
				Status:   "active",
				IsActive: false,
			},
			wantErr: true,
		},
		{
			name: "suspended tenant",
			tenant: &Tenant{
				ID:       "tenant-123",
				Slug:     "acme",
				Status:   "suspended",
				IsActive: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tenant.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Tenant.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTenant_IsEnterprise(t *testing.T) {
	tests := []struct {
		name string
		plan string
		want bool
	}{
		{"enterprise plan", "enterprise", true},
		{"standard plan", "standard", false},
		{"free plan", "free", false},
		{"empty plan", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant := &Tenant{Plan: tt.plan}
			if got := tenant.IsEnterprise(); got != tt.want {
				t.Errorf("Tenant.IsEnterprise() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenant_IsProduction(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		want        bool
	}{
		{"production", "production", true},
		{"staging", "staging", false},
		{"development", "development", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant := &Tenant{Environment: tt.environment}
			if got := tenant.IsProduction(); got != tt.want {
				t.Errorf("Tenant.IsProduction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithTenant(t *testing.T) {
	ctx := context.Background()
	tenant := &Tenant{
		ID:   "test-tenant",
		Slug: "test",
		Name: "Test Tenant",
	}

	ctx = WithTenant(ctx, tenant)
	retrieved, err := GetTenant(ctx)
	if err != nil {
		t.Errorf("GetTenant() error = %v", err)
	}
	if retrieved.ID != tenant.ID {
		t.Errorf("GetTenant() = %v, want %v", retrieved.ID, tenant.ID)
	}
}

func TestGetTenant_NilContext(t *testing.T) {
	_, err := GetTenant(nil)
	if err == nil {
		t.Error("GetTenant(nil) should return error")
	}
}

func TestGetTenant_NotFound(t *testing.T) {
	ctx := context.Background()
	_, err := GetTenant(ctx)
	if err == nil {
		t.Error("GetTenant() without tenant context should return error")
	}
}
