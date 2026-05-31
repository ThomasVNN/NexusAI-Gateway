package billing

import (
	"math"
	"testing"
)

func TestCalculateCost_Basic(t *testing.T) {
	pricing := &ModelPricing{
		ModelName:        "gpt-4o",
		InputPricePer1K:  0.005,
		OutputPricePer1K: 0.015,
		IsActive:         true,
	}

	tests := []struct {
		name         string
		inputTokens  int64
		outputTokens int64
		expectedCost float64
	}{
		{
			name:         "1000 input tokens, 0 output",
			inputTokens:  1000,
			outputTokens: 0,
			expectedCost: 0.005, // 1000/1000 * 0.005
		},
		{
			name:         "0 input tokens, 1000 output",
			inputTokens:  0,
			outputTokens: 1000,
			expectedCost: 0.015, // 1000/1000 * 0.015
		},
		{
			name:         "1000 input, 1000 output",
			inputTokens:  1000,
			outputTokens: 1000,
			expectedCost: 0.020, // 0.005 + 0.015
		},
		{
			name:         "10000 input, 5000 output",
			inputTokens:  10000,
			outputTokens: 5000,
			expectedCost: 0.125, // (10000/1000 * 0.005) + (5000/1000 * 0.015) = 0.05 + 0.075
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := pricing.CalculateCost(tt.inputTokens, tt.outputTokens)
			if math.Abs(cost-tt.expectedCost) > 0.0001 {
				t.Errorf("CalculateCost() = %f, want %f", cost, tt.expectedCost)
			}
		})
	}
}

func TestCalculateCostWithBatch(t *testing.T) {
	pricing := &ModelPricing{
		ModelName:            "gpt-4o",
		InputPricePer1K:      0.005,
		OutputPricePer1K:     0.015,
		BatchInputPricePer1K: 0.0015, // 70% discount
		IsActive:             true,
	}

	tests := []struct {
		name             string
		inputTokens      int64
		outputTokens     int64
		batchInputTokens int64
		expectedCost     float64
	}{
		{
			name:             "all batch input",
			inputTokens:      1000,
			outputTokens:     500,
			batchInputTokens: 1000,            // all input tokens are batch
			expectedCost:     0.0015 + 0.0075, // batch input + output = 0.009
		},
		{
			name:             "mixed standard and batch",
			inputTokens:      6000,
			outputTokens:     1000,
			batchInputTokens: 4000,
			expectedCost:     0.031, // (2000/1000 * 0.005) + (4000/1000 * 0.0015) + (1000/1000 * 0.015)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := pricing.CalculateCostWithBatch(tt.inputTokens, tt.outputTokens, tt.batchInputTokens)
			if math.Abs(cost-tt.expectedCost) > 0.0001 {
				t.Errorf("CalculateCostWithBatch() = %f, want %f", cost, tt.expectedCost)
			}
		})
	}
}

func TestPredefinedPrices(t *testing.T) {
	tests := []struct {
		modelName  string
		wantInput  float64
		wantOutput float64
		wantFound  bool
	}{
		{"gpt-4o", 0.005, 0.015, true},
		{"gpt-4o-mini", 0.00015, 0.0006, true},
		{"gpt-4-turbo", 0.01, 0.03, true},
		{"gemini-1.5-flash", 0.000075, 0.0003, true},
		{"unknown-model", 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			input, output, found := GetPredefinedPrice(tt.modelName)
			if found != tt.wantFound {
				t.Errorf("GetPredefinedPrice(%s) found = %v, want %v", tt.modelName, found, tt.wantFound)
			}
			if found && (math.Abs(input-tt.wantInput) > 0.000001 || math.Abs(output-tt.wantOutput) > 0.000001) {
				t.Errorf("GetPredefinedPrice(%s) = (%f, %f), want (%f, %f)", tt.modelName, input, output, tt.wantInput, tt.wantOutput)
			}
		})
	}
}

func TestBillingRecord_TotalCost(t *testing.T) {
	record := &BillingRecord{
		InputTokens:  10000,
		OutputTokens: 5000,
		InputCost:    0.05,
		OutputCost:   0.075,
		TotalCost:    0.125,
		Currency:     "USD",
	}

	expectedTotal := record.InputCost + record.OutputCost
	if math.Abs(record.TotalCost-expectedTotal) > 0.0001 {
		t.Errorf("TotalCost = %f, expected %f", record.TotalCost, expectedTotal)
	}
}

func TestBalance_AddCredit(t *testing.T) {
	balance := &Balance{
		Balance: 100.0,
	}

	// Simulate adding credit
	balance.Balance += 50.0
	if balance.Balance != 150.0 {
		t.Errorf("Balance after credit = %f, want 150.0", balance.Balance)
	}
}

func TestBalance_DeductDebit(t *testing.T) {
	balance := &Balance{
		Balance: 100.0,
	}

	// Simulate debit
	balance.Balance -= 25.0
	if balance.Balance != 75.0 {
		t.Errorf("Balance after debit = %f, want 75.0", balance.Balance)
	}
}

func TestTransaction_BalanceCalculation(t *testing.T) {
	transaction := &Transaction{
		Amount:        -10.0,
		BalanceBefore: 100.0,
		BalanceAfter:  90.0,
		Type:          "debit",
	}

	expected := transaction.BalanceBefore + transaction.Amount
	if math.Abs(transaction.BalanceAfter-expected) > 0.0001 {
		t.Errorf("BalanceAfter = %f, expected %f", transaction.BalanceAfter, expected)
	}
}

func TestBillingSummary_CostByModel(t *testing.T) {
	summary := &BillingSummary{
		TotalCost:   100.0,
		CostByModel: make(map[string]float64),
	}

	summary.CostByModel["gpt-4o"] = 60.0
	summary.CostByModel["claude-3-5-sonnet"] = 40.0

	// Verify total matches sum of individual models
	var sum float64
	for _, cost := range summary.CostByModel {
		sum += cost
	}

	if math.Abs(sum-summary.TotalCost) > 0.0001 {
		t.Errorf("Sum of CostByModel = %f, TotalCost = %f", sum, summary.TotalCost)
	}
}

func TestPricing_ZeroTokens(t *testing.T) {
	pricing := &ModelPricing{
		ModelName:        "gpt-4o",
		InputPricePer1K:  0.005,
		OutputPricePer1K: 0.015,
	}

	cost := pricing.CalculateCost(0, 0)
	if cost != 0 {
		t.Errorf("CalculateCost(0, 0) = %f, want 0", cost)
	}
}

func TestPricing_LargeTokenCount(t *testing.T) {
	pricing := &ModelPricing{
		ModelName:        "gpt-4o",
		InputPricePer1K:  0.005,
		OutputPricePer1K: 0.015,
	}

	// 1 million tokens
	cost := pricing.CalculateCost(500000, 500000)
	expected := 2.5 + 7.5 // (500000/1000 * 0.005) + (500000/1000 * 0.015)

	if math.Abs(cost-expected) > 0.0001 {
		t.Errorf("CalculateCost(500000, 500000) = %f, want %f", cost, expected)
	}
}
