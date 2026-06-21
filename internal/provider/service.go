package provider

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Service provides provider business logic
type Service struct {
	repo       *Repository
	httpClient *http.Client
}

// NewService creates a new provider service
func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Create creates a new provider after validation
func (s *Service) Create(ctx context.Context, p *Provider) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Generate ID if not provided
	if p.ID == "" {
		p.ID = generateProviderID(p.Type, p.Name)
	}

	// Set defaults
	if p.Status == "" {
		p.Status = "active"
	}
	if p.Priority == 0 {
		p.Priority = 100 // Default priority
	}
	if !p.Enabled {
		p.Enabled = true
	}

	slog.Info("Creating provider",
		slog.String("id", p.ID),
		slog.String("name", p.Name),
		slog.String("type", string(p.Type)),
	)

	return s.repo.Create(ctx, p)
}

// Update updates an existing provider
func (s *Service) Update(ctx context.Context, p *Provider) error {
	existing, err := s.repo.GetByID(ctx, p.ID)
	if err != nil {
		return err
	}

	// Preserve credentials if not provided
	if p.Credentials == "" {
		p.Credentials = existing.Credentials
	}

	if err := p.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	slog.Info("Updating provider",
		slog.String("id", p.ID),
		slog.String("name", p.Name),
	)

	return s.repo.Update(ctx, p)
}

// Delete removes a provider
func (s *Service) Delete(ctx context.Context, id string) error {
	slog.Info("Deleting provider", slog.String("id", id))
	return s.repo.Delete(ctx, id)
}

// GetByID retrieves a provider by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Provider, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves all providers with optional filtering
func (s *Service) List(ctx context.Context, filter *ProviderFilter) ([]*Provider, error) {
	return s.repo.List(ctx, filter)
}

// ListEnabled retrieves all enabled providers
func (s *Service) ListEnabled(ctx context.Context) ([]*Provider, error) {
	return s.repo.ListEnabled(ctx)
}

// GetWithHealth retrieves a provider with its health metrics
func (s *Service) GetWithHealth(ctx context.Context, id string) (*ProviderWithHealth, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	health, err := s.repo.GetHealth(ctx, id)
	if err != nil {
		slog.Warn("Failed to get health for provider", slog.String("id", id), slog.Any("error", err))
		health = &ProviderHealth{ProviderID: id, IsHealthy: true}
	}

	return &ProviderWithHealth{
		Provider: *p,
		Health:   *health,
	}, nil
}

// ListWithHealth retrieves all providers with their health metrics
func (s *Service) ListWithHealth(ctx context.Context, filter *ProviderFilter) ([]*ProviderWithHealth, error) {
	providers, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	result := make([]*ProviderWithHealth, len(providers))
	for i, p := range providers {
		health, err := s.repo.GetHealth(ctx, p.ID)
		if err != nil {
			slog.Warn("Failed to get health for provider", slog.String("id", p.ID), slog.Any("error", err))
			health = &ProviderHealth{ProviderID: p.ID, IsHealthy: true}
		}
		result[i] = &ProviderWithHealth{
			Provider: *p,
			Health:   *health,
		}
	}

	return result, nil
}

// SelectProvider selects the best available provider using priority and health
func (s *Service) SelectProvider(ctx context.Context, providerType ProviderType) (*Provider, error) {
	providers, err := s.repo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	if len(providers) == 0 {
		return nil, ErrNoHealthyProvider
	}

	// Filter by type if specified
	var filtered []*Provider
	if providerType != "" {
		for _, p := range providers {
			if p.Type == providerType {
				filtered = append(filtered, p)
			}
		}
	} else {
		filtered = providers
	}

	if len(filtered) == 0 {
		return nil, ErrNoHealthyProvider
	}

	// Get health for each provider and filter by health status
	var healthy []*Provider
	for _, p := range filtered {
		health, err := s.repo.GetHealth(ctx, p.ID)
		if err != nil {
			continue
		}
		if health.IsHealthy {
			healthy = append(healthy, p)
		}
	}

	if len(healthy) == 0 {
		// Fall back to enabled providers even if unhealthy
		slog.Warn("No healthy providers found, falling back to enabled providers")
		return filtered[0], nil
	}

	// Already sorted by priority from database query
	return healthy[0], nil
}

// CheckHealth performs a health check on a provider
func (s *Service) CheckHealth(ctx context.Context, id string) (*ProviderHealth, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	health := &ProviderHealth{
		ProviderID:      id,
		LastHealthCheck: time.Now(),
		ErrorCount:      0,
		LatencyP99:      0,
		SuccessRate:     100.0,
		IsHealthy:       true,
	}

	// Determine health check URL
	healthURL := p.HealthCheckURL
	if healthURL == "" {
		healthURL = p.Endpoint + "/health"
	}

	// Perform HTTP health check
	start := time.Now()
	resp, err := s.httpClient.Get(healthURL)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		health.IsHealthy = false
		health.ErrorCount = 1
		s.logHealthCheck(id, false, latency)

		// Update health in DB
		_ = s.repo.UpdateHealth(ctx, health)

		// Increment error count
		_ = s.repo.IncrementErrorCount(ctx, id)

		return health, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		health.IsHealthy = false
		health.ErrorCount = 1
		s.logHealthCheck(id, false, latency)

		_ = s.repo.UpdateHealth(ctx, health)
		_ = s.repo.IncrementErrorCount(ctx, id)

		return health, nil
	}

	if resp.StatusCode >= 400 {
		// Client errors don't count as provider failure
		s.logHealthCheck(id, true, latency)
	} else {
		s.logHealthCheck(id, true, latency)
	}

	health.LatencyP99 = float64(latency)
	health.SuccessRate = 100.0

	_ = s.repo.UpdateHealth(ctx, health)
	_ = s.repo.ResetErrorCount(ctx, id)

	return health, nil
}

// RecordSuccess records a successful request for a provider
func (s *Service) RecordSuccess(ctx context.Context, id string) error {
	_ = s.repo.ResetErrorCount(ctx, id)
	return nil
}

// RecordFailure records a failed request for a provider
func (s *Service) RecordFailure(ctx context.Context, id string) error {
	return s.repo.IncrementErrorCount(ctx, id)
}

// Enable enables a provider
func (s *Service) Enable(ctx context.Context, id string) error {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	p.Enabled = true
	p.Status = "active"
	return s.repo.Update(ctx, p)
}

// Disable disables a provider
func (s *Service) Disable(ctx context.Context, id string) error {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	p.Enabled = false
	p.Status = "inactive"
	return s.repo.Update(ctx, p)
}

// generateProviderID generates a unique ID for a provider
func generateProviderID(pType ProviderType, name string) string {
	timestamp := time.Now().UnixNano() / 1000000 // milliseconds
	return fmt.Sprintf("%s-%s-%d", pType, sanitizeName(name), timestamp)
}

// sanitizeName removes special characters from a name
func sanitizeName(name string) string {
	result := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' {
			result += string(c)
		}
	}
	if result == "" {
		result = "provider"
	}
	return result
}

// logHealthCheck logs a health check result
func (s *Service) logHealthCheck(providerID string, success bool, latencyMs int64) {
	if success {
		slog.Debug("Health check passed",
			slog.String("provider_id", providerID),
			slog.Int64("latency_ms", latencyMs),
		)
	} else {
		slog.Warn("Health check failed",
			slog.String("provider_id", providerID),
			slog.Int64("latency_ms", latencyMs),
		)
	}
}
