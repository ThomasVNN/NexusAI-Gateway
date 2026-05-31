package billing

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Repository handles billing data persistence
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new billing repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateModelPricing creates a new model pricing entry
func (r *Repository) CreateModelPricing(ctx context.Context, p *ModelPricing) error {
	query := `
		INSERT INTO model_pricing (model_name, input_price_per_1k, output_price_per_1k, batch_input_price_per_1k, is_active, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	p.UpdatedAt = time.Now()
	err := r.db.QueryRowContext(ctx, query,
		p.ModelName, p.InputPricePer1K, p.OutputPricePer1K, p.BatchInputPricePer1K, p.IsActive, p.UpdatedAt,
	).Scan(&p.ID)

	if err != nil {
		return fmt.Errorf("failed to create model pricing: %w", err)
	}

	return nil
}

// UpdateModelPricing updates an existing model pricing entry
func (r *Repository) UpdateModelPricing(ctx context.Context, p *ModelPricing) error {
	query := `
		UPDATE model_pricing SET
			input_price_per_1k = $1, output_price_per_1k = $2,
			batch_input_price_per_1k = $3, is_active = $4, updated_at = $5
		WHERE id = $6`

	p.UpdatedAt = time.Now()
	result, err := r.db.ExecContext(ctx, query,
		p.InputPricePer1K, p.OutputPricePer1K, p.BatchInputPricePer1K, p.IsActive, p.UpdatedAt, p.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update model pricing: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("model pricing not found")
	}

	return nil
}

// GetModelPricing retrieves pricing for a specific model
func (r *Repository) GetModelPricing(ctx context.Context, modelName string) (*ModelPricing, error) {
	query := `
		SELECT id, model_name, input_price_per_1k, output_price_per_1k, batch_input_price_per_1k, is_active, updated_at
		FROM model_pricing WHERE model_name = $1`

	p := &ModelPricing{}
	err := r.db.QueryRowContext(ctx, query, modelName).Scan(
		&p.ID, &p.ModelName, &p.InputPricePer1K, &p.OutputPricePer1K, &p.BatchInputPricePer1K, &p.IsActive, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model pricing: %w", err)
	}

	return p, nil
}

// ListModelPricing retrieves all model pricing entries
func (r *Repository) ListModelPricing(ctx context.Context) ([]*ModelPricing, error) {
	query := `
		SELECT id, model_name, input_price_per_1k, output_price_per_1k, batch_input_price_per_1k, is_active, updated_at
		FROM model_pricing ORDER BY model_name ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list model pricing: %w", err)
	}
	defer rows.Close()

	var prices []*ModelPricing
	for rows.Next() {
		p := &ModelPricing{}
		if err := rows.Scan(
			&p.ID, &p.ModelName, &p.InputPricePer1K, &p.OutputPricePer1K, &p.BatchInputPricePer1K, &p.IsActive, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan model pricing: %w", err)
		}
		prices = append(prices, p)
	}

	return prices, nil
}

// CreateBillingRecord creates a new billing record
func (r *Repository) CreateBillingRecord(ctx context.Context, rec *BillingRecord) error {
	query := `
		INSERT INTO billing_records (organization_id, user_id, api_key_id, model, input_tokens, output_tokens,
		                           input_cost, output_cost, total_cost, currency, channel_id, token_group_id, request_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id`

	rec.CreatedAt = time.Now()
	err := r.db.QueryRowContext(ctx, query,
		rec.OrganizationID, rec.UserID, rec.APIKeyID, rec.Model, rec.InputTokens, rec.OutputTokens,
		rec.InputCost, rec.OutputCost, rec.TotalCost, rec.Currency, rec.ChannelID, rec.TokenGroupID, rec.RequestID, rec.CreatedAt,
	).Scan(&rec.ID)

	if err != nil {
		return fmt.Errorf("failed to create billing record: %w", err)
	}

	return nil
}

// GetBillingSummary retrieves billing summary for a period
func (r *Repository) GetBillingSummary(ctx context.Context, orgID, userID *int64, startDate, endDate time.Time) (*BillingSummary, error) {
	query := `
		SELECT
			COALESCE(SUM(total_cost), 0) as total_cost,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COUNT(*) as total_requests
		FROM billing_records
		WHERE created_at >= $1 AND created_at < $2`

	args := []interface{}{startDate, endDate}
	argIdx := 3

	if orgID != nil {
		query += fmt.Sprintf(" AND organization_id = $%d", argIdx)
		args = append(args, *orgID)
		argIdx++
	}
	if userID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, *userID)
	}

	summary := &BillingSummary{
		OrganizationID: orgID,
		UserID:         userID,
		StartDate:      startDate,
		EndDate:        endDate,
		Currency:       "USD",
		CostByModel:    make(map[string]float64),
		CostByDay:      make(map[string]float64),
	}

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&summary.TotalCost, &summary.TotalInputTokens, &summary.TotalOutputTokens, &summary.TotalRequests,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get billing summary: %w", err)
	}

	// Get cost by model
	modelQuery := `
		SELECT model, COALESCE(SUM(total_cost), 0)
		FROM billing_records
		WHERE created_at >= $1 AND created_at < $2`
	modelArgs := []interface{}{startDate, endDate}
	modelArgIdx := 3

	if orgID != nil {
		modelQuery += fmt.Sprintf(" AND organization_id = $%d", modelArgIdx)
		modelArgs = append(modelArgs, *orgID)
		modelArgIdx++
	}
	if userID != nil {
		modelQuery += fmt.Sprintf(" AND user_id = $%d", modelArgIdx)
		modelArgs = append(modelArgs, *userID)
	}

	modelQuery += " GROUP BY model"
	rows, err := r.db.QueryContext(ctx, modelQuery, modelArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost by model: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var model string
		var cost float64
		if err := rows.Scan(&model, &cost); err != nil {
			return nil, fmt.Errorf("failed to scan model cost: %w", err)
		}
		summary.CostByModel[model] = cost
	}

	// Get cost by day
	dayQuery := `
		SELECT DATE(created_at) as day, COALESCE(SUM(total_cost), 0)
		FROM billing_records
		WHERE created_at >= $1 AND created_at < $2`
	dayArgs := []interface{}{startDate, endDate}
	dayArgIdx := 3

	if orgID != nil {
		dayQuery += fmt.Sprintf(" AND organization_id = $%d", dayArgIdx)
		dayArgs = append(dayArgs, *orgID)
		dayArgIdx++
	}
	if userID != nil {
		dayQuery += fmt.Sprintf(" AND user_id = $%d", dayArgIdx)
		dayArgs = append(dayArgs, *userID)
	}

	dayQuery += " GROUP BY DATE(created_at) ORDER BY day DESC"
	rows2, err := r.db.QueryContext(ctx, dayQuery, dayArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost by day: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var day time.Time
		var cost float64
		if err := rows2.Scan(&day, &cost); err != nil {
			return nil, fmt.Errorf("failed to scan daily cost: %w", err)
		}
		summary.CostByDay[day.Format("2006-01-02")] = cost
	}

	return summary, nil
}

// GetBalance retrieves the balance for an organization or user
func (r *Repository) GetBalance(ctx context.Context, orgID, userID *int64) (*Balance, error) {
	query := `
		SELECT id, organization_id, user_id, balance, currency, updated_at
		FROM balances WHERE 1=1`

	args := []interface{}{}
	argIdx := 1

	if orgID != nil {
		query += fmt.Sprintf(" AND organization_id = $%d", argIdx)
		args = append(args, *orgID)
		argIdx++
	}
	if userID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, *userID)
	}

	b := &Balance{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&b.ID, &b.OrganizationID, &b.UserID, &b.Balance, &b.Currency, &b.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return &Balance{
			OrganizationID: orgID,
			UserID:         userID,
			Balance:        0,
			Currency:       "USD",
			UpdatedAt:      time.Now(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return b, nil
}

// UpdateBalance updates the balance for an organization or user
func (r *Repository) UpdateBalance(ctx context.Context, orgID, userID *int64, newBalance float64) error {
	query := `
		INSERT INTO balances (organization_id, user_id, balance, currency, updated_at)
		VALUES ($1, $2, $3, 'USD', $4)
		ON CONFLICT (organization_id, user_id) DO UPDATE SET
			balance = $3, updated_at = $4`

	_, err := r.db.ExecContext(ctx, query, orgID, userID, newBalance, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	return nil
}

// CreateTransaction creates a new transaction record
func (r *Repository) CreateTransaction(ctx context.Context, tx *Transaction) error {
	query := `
		INSERT INTO billing_transactions (organization_id, user_id, amount, balance_before, balance_after, type, description, reference_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	tx.CreatedAt = time.Now()
	err := r.db.QueryRowContext(ctx, query,
		tx.OrganizationID, tx.UserID, tx.Amount, tx.BalanceBefore, tx.BalanceAfter,
		tx.Type, tx.Description, tx.ReferenceID, tx.CreatedAt,
	).Scan(&tx.ID)

	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}
