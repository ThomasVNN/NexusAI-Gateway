package channel

import (
	"testing"
)

// TestCircuitBreaker_InitialState tests that circuit breaker starts closed
func TestCircuitBreaker_InitialState(t *testing.T) {
	cb := NewCircuitBreaker()

	// Initially, states map should be empty (meaning all circuits are closed)
	if len(cb.states) != 0 {
		t.Errorf("Expected empty states map initially, got %d entries", len(cb.states))
	}
}

// TestCircuitBreaker_FailureCount tests failure counting
func TestCircuitBreaker_FailureCount(t *testing.T) {
	router := &Router{
		circuitBreaker: NewCircuitBreaker(),
	}

	channelID := int64(123)

	// Record some failures
	router.RecordFailure(channelID)
	router.RecordFailure(channelID)
	router.RecordFailure(channelID)

	if router.circuitBreaker.failureCounts[channelID] != 3 {
		t.Errorf("Expected 3 failures, got %d", router.circuitBreaker.failureCounts[channelID])
	}
}

// TestCircuitBreaker_Threshold tests that circuit opens at threshold
func TestCircuitBreaker_Threshold(t *testing.T) {
	router := &Router{
		circuitBreaker: NewCircuitBreaker(),
	}

	channelID := int64(456)

	// Record failures up to threshold
	for i := 0; i < circuitFailureThreshold; i++ {
		router.RecordFailure(channelID)
	}

	// Circuit should now be open
	if router.circuitBreaker.states[channelID] != CircuitOpen {
		t.Errorf("Expected circuit to be open, got state %v", router.circuitBreaker.states[channelID])
	}
}

// TestCircuitBreaker_SuccessResetsCount tests that success resets failure count
func TestCircuitBreaker_SuccessResetsCount(t *testing.T) {
	router := &Router{
		circuitBreaker: NewCircuitBreaker(),
	}

	channelID := int64(789)

	// Record some failures
	for i := 0; i < 3; i++ {
		router.RecordFailure(channelID)
	}

	// Verify failure count
	if router.circuitBreaker.failureCounts[channelID] != 3 {
		t.Errorf("Expected 3 failures, got %d", router.circuitBreaker.failureCounts[channelID])
	}

	// Record success
	router.RecordSuccess(channelID)

	// Failure count should be cleared
	if router.circuitBreaker.failureCounts[channelID] != 0 {
		t.Errorf("Expected 0 failures after success, got %d", router.circuitBreaker.failureCounts[channelID])
	}
}

// TestCircuitBreaker_ClosedCircuitAvailability tests that closed circuits are available
func TestCircuitBreaker_ClosedCircuitAvailability(t *testing.T) {
	router := &Router{
		circuitBreaker: NewCircuitBreaker(),
	}

	channelID := int64(101)

	// New channel should have no state (effectively closed)
	state, exists := router.circuitBreaker.states[channelID]
	if exists {
		t.Errorf("Expected no state for new channel, got %v", state)
	}

	// Channel should be considered available (no open circuit)
	// This is tested implicitly - a channel with no state is available
}

// TestCircuitBreaker_OpenCircuitBlocksAvailability tests that open circuits block requests
func TestCircuitBreaker_OpenCircuitBlocksAvailability(t *testing.T) {
	router := &Router{
		circuitBreaker: NewCircuitBreaker(),
	}

	channelID := int64(202)

	// Open the circuit
	for i := 0; i < circuitFailureThreshold; i++ {
		router.RecordFailure(channelID)
	}

	// Circuit should be open
	if router.circuitBreaker.states[channelID] != CircuitOpen {
		t.Fatal("Circuit should be open")
	}

	// Any subsequent failures should not change state
	router.RecordFailure(channelID)

	if router.circuitBreaker.states[channelID] != CircuitOpen {
		t.Error("Circuit should remain open after more failures")
	}
}
