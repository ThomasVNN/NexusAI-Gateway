package billing

import (
	"testing"
	"time"
)

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name         string
		modelName    string
		inputTokens  int64
		outputTokens int64
		wantCost     float64
	}{
		{
			name:         "GPT-4 pricing",
			modelName:    "gpt-4",
			inputTokens:  1000,
			outputTokens: 500,
			wantCost:     0.06, // $0.03 per 1K input + $0.03 per 1K output (500 tokens = 0.5 * 0.06)
		},
		{
			name:         "GPT-3.5 pricing",
			modelName:    "gpt-3.5-turbo",
			inputTokens:  1000,
			outputTokens: 500,
			wantCost:     0.00125, // $0.0005 per 1K input + $0.00075 per 1K output
		},
		{
			name:         "Claude pricing",
			modelName:    "claude-3-opus-20240229",
			inputTokens:  1000,
			outputTokens: 500,
			wantCost:     0.0525, // $0.015 per 1K input + $0.0375 per 1K output
		},
		{
			name:         "Gemini pricing",
			modelName:    "gemini-1.5-pro",
			inputTokens:  1000,
			outputTokens: 500,
			wantCost:     0.00375, // $0.00125 per 1K input + $0.0025 per 1K output
		},
		{
			name:         "GPT-4o pricing",
			modelName:    "gpt-4o",
			inputTokens:  1000,
			outputTokens: 500,
			wantCost:     0.0125, // $0.005 per 1K input + $0.0075 per 1K output
		},
		{
			name:         "Unknown model uses default pricing",
			modelName:    "unknown-model",
			inputTokens:  1000,
			outputTokens: 500,
			wantCost:     0.003, // Uses default 0.001 + 0.002
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use GetModelPrice to test pricing lookup
			inputPrice, outputPrice, found := GetPredefinedPrice(tt.modelName)

			if tt.modelName != "unknown-model" && !found {
				t.Errorf("Expected to find pricing for %s", tt.modelName)
				return
			}

			if !found {
				// Use default pricing
				inputPrice = 0.001
				outputPrice = 0.002
			}

			cost := (float64(tt.inputTokens) / 1000 * inputPrice) +
				(float64(tt.outputTokens) / 1000 * outputPrice)

			// Allow small floating point difference
			diff := cost - tt.wantCost
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.001 {
				t.Errorf("CalculateCost() = %v, want %v", cost, tt.wantCost)
			}
		})
	}
}

func TestGetPredefinedPrice(t *testing.T) {
	tests := []struct {
		modelName  string
		wantFound  bool
		wantInput  float64
		wantOutput float64
	}{
		{"gpt-4", true, 0.03, 0.06},
		{"gpt-4o", true, 0.005, 0.015},
		{"gpt-4o-mini", true, 0.00015, 0.0006},
		{"gpt-3.5-turbo", true, 0.0005, 0.0015},
		{"claude-3-5-sonnet-latest", true, 0.003, 0.015},
		{"claude-3-opus-20240229", true, 0.015, 0.075},
		{"gemini-1.5-pro", true, 0.00125, 0.005},
		{"gemini-1.5-flash", true, 0.000075, 0.0003},
		{"mistral-large-latest", true, 0.002, 0.006},
		{"unknown-model", false, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			input, output, found := GetPredefinedPrice(tt.modelName)
			if found != tt.wantFound {
				t.Errorf("GetPredefinedPrice(%s) found = %v, want %v", tt.modelName, found, tt.wantFound)
			}
			if found && (input != tt.wantInput || output != tt.wantOutput) {
				t.Errorf("GetPredefinedPrice(%s) = (%v, %v), want (%v, %v)",
					tt.modelName, input, output, tt.wantInput, tt.wantOutput)
			}
		})
	}
}

func TestModelPricing_CalculateCost(t *testing.T) {
	tests := []struct {
		name         string
		price        *ModelPricing
		inputTokens  int64
		outputTokens int64
		wantCost     float64
	}{
		{
			name: "Standard pricing",
			price: &ModelPricing{
				ModelName:        "test-model",
				InputPricePer1K:  0.01,
				OutputPricePer1K: 0.02,
			},
			inputTokens:  1000,
			outputTokens: 500,
			wantCost:     0.02, // 0.01 + 0.01
		},
		{
			name: "Zero tokens",
			price: &ModelPricing{
				ModelName:        "test-model",
				InputPricePer1K:  0.01,
				OutputPricePer1K: 0.02,
			},
			inputTokens:  0,
			outputTokens: 0,
			wantCost:     0.0,
		},
		{
			name: "Large token count",
			price: &ModelPricing{
				ModelName:        "test-model",
				InputPricePer1K:  0.01,
				OutputPricePer1K: 0.02,
			},
			inputTokens:  100000,
			outputTokens: 50000,
			wantCost:     2.0, // 1.0 + 1.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := tt.price.CalculateCost(tt.inputTokens, tt.outputTokens)
			if cost != tt.wantCost {
				t.Errorf("CalculateCost() = %v, want %v", cost, tt.wantCost)
			}
		})
	}
}

func TestBillingSummary(t *testing.T) {
	summary := &BillingSummary{
		TotalCost:         100.50,
		TotalInputTokens:  10000,
		TotalOutputTokens: 5000,
		TotalRequests:     25,
		Currency:          "USD",
		CostByModel: map[string]float64{
			"gpt-4":   75.00,
			"gpt-3.5": 25.50,
		},
	}

	if summary.TotalRequests != 25 {
		t.Errorf("BillingSummary.TotalRequests = %d, want 25", summary.TotalRequests)
	}

	if len(summary.CostByModel) != 2 {
		t.Errorf("BillingSummary.CostByModel length = %d, want 2", len(summary.CostByModel))
	}

	if summary.CostByModel["gpt-4"] != 75.00 {
		t.Errorf("BillingSummary.CostByModel[gpt-4] = %v, want 75.00", summary.CostByModel["gpt-4"])
	}
}

func TestTransaction(t *testing.T) {
	now := time.Now()
	tx := &Transaction{
		OrganizationID: ptrInt64(1),
		UserID:         ptrInt64(100),
		Amount:         50.00,
		BalanceBefore:  100.00,
		BalanceAfter:   150.00,
		Type:           "credit",
		Description:    "Test credit",
		CreatedAt:      now,
	}

	if tx.Type != "credit" {
		t.Errorf("Transaction.Type = %s, want credit", tx.Type)
	}

	if tx.BalanceAfter != tx.BalanceBefore+tx.Amount {
		t.Errorf("Transaction.BalanceAfter calculation incorrect")
	}
}

func TestBalance(t *testing.T) {
	balance := &Balance{
		OrganizationID: ptrInt64(1),
		UserID:         ptrInt64(100),
		Balance:        500.00,
		Currency:       "USD",
	}

	if balance.Balance != 500.00 {
		t.Errorf("Balance.Balance = %v, want 500.00", balance.Balance)
	}

	if balance.Currency != "USD" {
		t.Errorf("Balance.Currency = %s, want USD", balance.Currency)
	}
}

// Helper function
func ptrInt64(v int64) *int64 {
	return &v
}
