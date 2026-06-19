package billing

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Service provides billing business logic
type Service struct {
	repo *Repository
}

// NewService creates a new billing service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetModelPrice retrieves pricing for a model, falling back to predefined prices
func (s *Service) GetModelPrice(ctx context.Context, modelName string) (*ModelPricing, error) {
	// First try database
	price, err := s.repo.GetModelPricing(ctx, modelName)
	if err != nil {
		return nil, err
	}

	if price != nil && price.IsActive {
		return price, nil
	}

	// Fall back to predefined prices
	if input, output, found := GetPredefinedPrice(modelName); found {
		return &ModelPricing{
			ModelName:        modelName,
			InputPricePer1K:  input,
			OutputPricePer1K: output,
			IsActive:         true,
			UpdatedAt:        time.Now(),
		}, nil
	}

	// Return default pricing if no match
	return &ModelPricing{
		ModelName:        modelName,
		InputPricePer1K:  0.001, // Default $1 per 1M tokens
		OutputPricePer1K: 0.002,
		IsActive:         true,
		UpdatedAt:        time.Now(),
	}, nil
}

// CalculateCost calculates the cost for a request
func (s *Service) CalculateCost(ctx context.Context, modelName string, inputTokens, outputTokens int64) (float64, error) {
	price, err := s.GetModelPrice(ctx, modelName)
	if err != nil {
		return 0, err
	}

	cost := price.CalculateCost(inputTokens, outputTokens)
	slog.Debug("Calculated cost",
		slog.String("model", modelName),
		slog.Int64("input_tokens", inputTokens),
		slog.Int64("output_tokens", outputTokens),
		slog.Float64("cost", cost),
	)

	return cost, nil
}

// RecordBilling records a billing transaction
func (s *Service) RecordBilling(ctx context.Context, rec *BillingRecord) error {
	// Calculate costs if not provided
	if rec.InputCost == 0 && rec.OutputCost == 0 {
		price, err := s.GetModelPrice(ctx, rec.Model)
		if err != nil {
			return fmt.Errorf("failed to get pricing: %w", err)
		}

		rec.InputCost = float64(rec.InputTokens) / 1000 * price.InputPricePer1K
		rec.OutputCost = float64(rec.OutputTokens) / 1000 * price.OutputPricePer1K
		rec.TotalCost = rec.InputCost + rec.OutputCost
	}

	if rec.Currency == "" {
		rec.Currency = "USD"
	}

	slog.Debug("Recording billing",
		slog.String("model", rec.Model),
		slog.Int64("input_tokens", rec.InputTokens),
		slog.Int64("output_tokens", rec.OutputTokens),
		slog.Float64("total_cost", rec.TotalCost),
	)

	return s.repo.CreateBillingRecord(ctx, rec)
}

// GetBillingSummary retrieves billing summary for a period
func (s *Service) GetBillingSummary(ctx context.Context, orgID, userID *int64, period string) (*BillingSummary, error) {
	now := time.Now()
	var startDate, endDate time.Time

	switch period {
	case "daily":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDate = startDate.Add(24 * time.Hour)
	case "weekly":
		startDate = now.AddDate(0, 0, -int(now.Weekday()))
		startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, now.Location())
		endDate = startDate.Add(7 * 24 * time.Hour)
	case "monthly":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, 0)
	case "yearly":
		startDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		endDate = time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, now.Location())
	default:
		// Default to last 30 days
		startDate = now.AddDate(0, 0, -30)
		endDate = now
	}

	return s.repo.GetBillingSummary(ctx, orgID, userID, startDate, endDate)
}

// UpdateModelPricing updates pricing for a model
func (s *Service) UpdateModelPricing(ctx context.Context, p *ModelPricing) error {
	existing, err := s.repo.GetModelPricing(ctx, p.ModelName)
	if err != nil {
		return err
	}

	if existing == nil {
		// Create new pricing
		if p.ID == 0 {
			return s.repo.CreateModelPricing(ctx, p)
		}
		// Use existing ID if not provided
		p.ID = existing.ID
	}

	return s.repo.UpdateModelPricing(ctx, p)
}

// ListModelPricing retrieves all model pricing
func (s *Service) ListModelPricing(ctx context.Context) ([]*ModelPricing, error) {
	return s.repo.ListModelPricing(ctx)
}

// GetBalance retrieves balance for an entity
func (s *Service) GetBalance(ctx context.Context, orgID, userID *int64) (*Balance, error) {
	return s.repo.GetBalance(ctx, orgID, userID)
}

// AddCredit adds credit to a balance
func (s *Service) AddCredit(ctx context.Context, orgID, userID *int64, amount float64, description string) error {
	if amount <= 0 {
		return fmt.Errorf("credit amount must be positive")
	}

	balance, err := s.repo.GetBalance(ctx, orgID, userID)
	if err != nil {
		return err
	}

	newBalance := balance.Balance + amount

	// Update balance
	if err := s.repo.UpdateBalance(ctx, orgID, userID, newBalance); err != nil {
		return err
	}

	// Record transaction
	tx := &Transaction{
		OrganizationID: orgID,
		UserID:         userID,
		Amount:         amount,
		BalanceBefore:  balance.Balance,
		BalanceAfter:   newBalance,
		Type:           "credit",
		Description:    description,
	}

	return s.repo.CreateTransaction(ctx, tx)
}

// DeductBalance deducts from a balance
func (s *Service) DeductBalance(ctx context.Context, orgID, userID *int64, amount float64, description string, referenceID string) error {
	if amount <= 0 {
		return fmt.Errorf("deduction amount must be positive")
	}

	balance, err := s.repo.GetBalance(ctx, orgID, userID)
	if err != nil {
		return err
	}

	if balance.Balance < amount {
		return fmt.Errorf("insufficient balance: have %.2f, need %.2f", balance.Balance, amount)
	}

	newBalance := balance.Balance - amount

	// Update balance
	if err := s.repo.UpdateBalance(ctx, orgID, userID, newBalance); err != nil {
		return err
	}

	// Record transaction
	tx := &Transaction{
		OrganizationID: orgID,
		UserID:         userID,
		Amount:         -amount,
		BalanceBefore:  balance.Balance,
		BalanceAfter:   newBalance,
		Type:           "debit",
		Description:    description,
		ReferenceID:    referenceID,
	}

	return s.repo.CreateTransaction(ctx, tx)
}

// CheckBalance checks if there's sufficient balance
func (s *Service) CheckBalance(ctx context.Context, orgID, userID *int64, requiredAmount float64) (bool, error) {
	balance, err := s.repo.GetBalance(ctx, orgID, userID)
	if err != nil {
		return false, err
	}

	return balance.Balance >= requiredAmount, nil
}

// InitializeDefaultPricing initializes default pricing for common models
func (s *Service) InitializeDefaultPricing(ctx context.Context) error {
	for modelName, price := range PredefinedModelPrices {
		existing, err := s.repo.GetModelPricing(ctx, modelName)
		if err != nil {
			return err
		}

		if existing != nil {
			continue // Skip if already exists
		}

		p := &ModelPricing{
			ModelName:        modelName,
			InputPricePer1K:  price.Input,
			OutputPricePer1K: price.Output,
			IsActive:         true,
		}

		if err := s.repo.CreateModelPricing(ctx, p); err != nil {
			slog.Warn("Failed to initialize pricing for model",
				slog.String("model", modelName),
				slog.Any("error", err),
			)
		} else {
			slog.Info("Initialized pricing for model",
				slog.String("model", modelName),
				slog.Float64("input_price", price.Input),
				slog.Float64("output_price", price.Output),
			)
		}
	}

	return nil
}
