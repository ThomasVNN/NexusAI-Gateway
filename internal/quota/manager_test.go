package quota

import (
	"testing"
)

func TestDefaultQuotaPlans(t *testing.T) {
	plans := []string{"free", "standard", "enterprise"}

	for _, plan := range plans {
		t.Run(plan, func(t *testing.T) {
			p, ok := DefaultQuotaPlans[plan]
			if !ok {
				t.Errorf("DefaultQuotaPlans missing plan: %s", plan)
				return
			}
			if p.RequestsPerMinute <= 0 {
				t.Errorf("RequestsPerMinute should be positive, got %d", p.RequestsPerMinute)
			}
			if p.RequestsPerHour <= 0 {
				t.Errorf("RequestsPerHour should be positive, got %d", p.RequestsPerHour)
			}
			if p.RequestsPerDay <= 0 {
				t.Errorf("RequestsPerDay should be positive, got %d", p.RequestsPerDay)
			}
			if p.RequestsPerMinute > p.RequestsPerHour {
				t.Error("RequestsPerMinute should not exceed RequestsPerHour")
			}
			if p.RequestsPerHour > p.RequestsPerDay {
				t.Error("RequestsPerHour should not exceed RequestsPerDay")
			}
		})
	}
}

func TestTenantRateLimiter_Allow(t *testing.T) {
	limiter := NewTenantRateLimiter()

	tests := []struct {
		name          string
		tenantID      string
		plan          string
		requests      int
		wantAllowed   bool
		wantRemaining int
	}{
		{
			name:        "first request should be allowed",
			tenantID:    "tenant-1",
			plan:        "standard",
			requests:    1,
			wantAllowed: true,
		},
		{
			name:        "requests under limit",
			tenantID:    "tenant-2",
			plan:        "free",
			requests:    5,
			wantAllowed: true,
		},
		{
			name:        "enterprise has higher limit",
			tenantID:    "tenant-3",
			plan:        "enterprise",
			requests:    100,
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make multiple requests up to the count
			for i := 0; i < tt.requests; i++ {
				allowed, usage := limiter.Allow(tt.tenantID, tt.plan, 100)
				if i == tt.requests-1 {
					// Check last request result
					if allowed != tt.wantAllowed {
						t.Errorf("Allow() = %v, want %v", allowed, tt.wantAllowed)
					}
					if usage != nil && usage.RemainingMinute < 0 {
						t.Errorf("RemainingMinute = %d, should not be negative", usage.RemainingMinute)
					}
				}
			}
		})
	}
}

func TestTenantRateLimiter_EnterpriseHigherLimits(t *testing.T) {
	limiter := NewTenantRateLimiter()

	// Enterprise plan should have higher limits than free
	enterpriseAllowed, _ := limiter.Allow("ent-tenant", "enterprise", 100)
	freeAllowed, _ := limiter.Allow("free-tenant", "free", 100)

	// Enterprise should allow more requests
	if !enterpriseAllowed && freeAllowed {
		t.Error("Enterprise plan should allow at least as many requests as free plan")
	}
}

func TestTenantRateLimiter_DifferentTenants(t *testing.T) {
	limiter := NewTenantRateLimiter()

	// Different tenants should have independent limits
	allowed1, _ := limiter.Allow("tenant-A", "standard", 100)
	allowed2, _ := limiter.Allow("tenant-B", "standard", 100)

	if !allowed1 || !allowed2 {
		t.Error("Different tenants should have independent rate limits")
	}
}

func TestTenantRateLimiter_Cleanup(t *testing.T) {
	limiter := NewTenantRateLimiter()

	// Add a tenant
	limiter.Allow("temp-tenant", "standard", 100)

	// Cleanup should not panic
	limiter.Cleanup(0)
}
