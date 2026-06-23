package circuitbreaker

import (
	"sync"
	"testing"
	"time"
)

// TestConfig tests configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.FailureThreshold != 5 {
		t.Errorf("Expected FailureThreshold 5, got %d", config.FailureThreshold)
	}
	
	if config.SuccessThreshold != 3 {
		t.Errorf("Expected SuccessThreshold 3, got %d", config.SuccessThreshold)
	}
	
	if config.OpenDuration != 30*time.Second {
		t.Errorf("Expected OpenDuration 30s, got %v", config.OpenDuration)
	}
	
	if config.HalfOpenMaxCalls != 3 {
		t.Errorf("Expected HalfOpenMaxCalls 3, got %d", config.HalfOpenMaxCalls)
	}
}

// TestCircuitInitialState tests the initial state of a circuit
func TestCircuitInitialState(t *testing.T) {
	circuit := NewCircuit("test-provider", DefaultConfig())
	
	if circuit.State() != StateClosed {
		t.Errorf("Expected initial state CLOSED, got %s", circuit.State())
	}
	
	if circuit.Provider() != "test-provider" {
		t.Errorf("Expected provider 'test-provider', got '%s'", circuit.Provider())
	}
}

// TestCircuitAllow tests Allow() in closed state
func TestCircuitAllow(t *testing.T) {
	circuit := NewCircuit("test", DefaultConfig())
	
	// Should allow requests in closed state
	if !circuit.Allow() {
		t.Error("Expected Allow() to return true in CLOSED state")
	}
}

// TestCircuitRecordsSuccess tests success recording
func TestCircuitRecordsSuccess(t *testing.T) {
	circuit := NewCircuit("test", DefaultConfig())
	
	// Record some successes
	for i := 0; i < 3; i++ {
		circuit.RecordSuccess()
	}
	
	stats := circuit.Stats()
	if stats.Successes != 3 {
		t.Errorf("Expected 3 successes, got %d", stats.Successes)
	}
	
	if stats.Failures != 0 {
		t.Errorf("Expected 0 failures, got %d", stats.Failures)
	}
	
	// Should still be closed
	if circuit.State() != StateClosed {
		t.Errorf("Expected state CLOSED, got %s", circuit.State())
	}
}

// TestCircuitOpensOnFailures tests circuit opening on consecutive failures
func TestCircuitOpensOnFailures(t *testing.T) {
	config := DefaultConfig()
	circuit := NewCircuit("test", config)
	
	// Record failures up to threshold
	for i := 0; i < config.FailureThreshold; i++ {
		circuit.RecordFailure()
	}
	
	stats := circuit.Stats()
	if stats.State != "OPEN" {
		t.Errorf("Expected state OPEN after %d failures, got %s", config.FailureThreshold, stats.State)
	}
	
	// Should now block requests
	if circuit.Allow() {
		t.Error("Expected Allow() to return false in OPEN state")
	}
}

// TestCircuitHalfOpenAfterTimeout tests transition to half-open
func TestCircuitHalfOpenAfterTimeout(t *testing.T) {
	config := DefaultConfig()
	config.OpenDuration = 100 * time.Millisecond
	circuit := NewCircuit("test", config)
	
	// Open the circuit
	for i := 0; i < config.FailureThreshold; i++ {
		circuit.RecordFailure()
	}
	
	if circuit.State() != StateOpen {
		t.Fatalf("Expected state OPEN, got %s", circuit.State())
	}
	
	// Wait for timeout
	time.Sleep(150 * time.Millisecond)
	
	// Should now transition to half-open and allow request
	if !circuit.Allow() {
		t.Error("Expected Allow() to return true after timeout")
	}
	
	if circuit.State() != StateHalfOpen {
		t.Errorf("Expected state HALF_OPEN after timeout, got %s", circuit.State())
	}
}

// TestCircuitClosesOnSuccessesInHalfOpen tests closing after successes in half-open
func TestCircuitClosesOnSuccessesInHalfOpen(t *testing.T) {
	config := DefaultConfig()
	config.OpenDuration = 1 * time.Millisecond // Immediate transition
	circuit := NewCircuit("test", config)
	
	// Open the circuit
	for i := 0; i < config.FailureThreshold; i++ {
		circuit.RecordFailure()
	}
	
	// Wait and trigger half-open
	time.Sleep(10 * time.Millisecond)
	circuit.Allow() // This triggers transition to half-open
	
	if circuit.State() != StateHalfOpen {
		t.Fatalf("Expected state HALF_OPEN, got %s", circuit.State())
	}
	
	// Record successes in half-open
	for i := 0; i < config.SuccessThreshold; i++ {
		circuit.RecordSuccess()
	}
	
	if circuit.State() != StateClosed {
		t.Errorf("Expected state CLOSED after %d successes in half-open, got %s", 
			config.SuccessThreshold, circuit.State())
	}
}

// TestCircuitReopensOnFailureInHalfOpen tests reopening on failure in half-open
func TestCircuitReopensOnFailureInHalfOpen(t *testing.T) {
	config := DefaultConfig()
	config.OpenDuration = 1 * time.Millisecond
	circuit := NewCircuit("test", config)
	
	// Open the circuit
	for i := 0; i < config.FailureThreshold; i++ {
		circuit.RecordFailure()
	}
	
	// Wait and trigger half-open
	time.Sleep(10 * time.Millisecond)
	circuit.Allow()
	
	if circuit.State() != StateHalfOpen {
		t.Fatalf("Expected state HALF_OPEN, got %s", circuit.State())
	}
	
	// Record failure in half-open
	circuit.RecordFailure()
	
	if circuit.State() != StateOpen {
		t.Errorf("Expected state OPEN after failure in half-open, got %s", circuit.State())
	}
}

// TestCircuitReset tests manual reset
func TestCircuitReset(t *testing.T) {
	circuit := NewCircuit("test", DefaultConfig())
	
	// Open the circuit
	for i := 0; i < 5; i++ {
		circuit.RecordFailure()
	}
	
	if circuit.State() != StateOpen {
		t.Fatalf("Expected state OPEN, got %s", circuit.State())
	}
	
	// Reset
	circuit.Reset()
	
	if circuit.State() != StateClosed {
		t.Errorf("Expected state CLOSED after reset, got %s", circuit.State())
	}
}

// TestManagerGetOrCreate tests manager circuit creation
func TestManagerGetOrCreate(t *testing.T) {
	manager := NewManager(DefaultConfig())
	
	// Get non-existent circuit - should create
	circuit1 := manager.GetOrCreate("provider-a")
	if circuit1 == nil {
		t.Fatal("Expected circuit to be created")
	}
	if circuit1.Provider() != "provider-a" {
		t.Errorf("Expected provider 'provider-a', got '%s'", circuit1.Provider())
	}
	
	// Get same circuit again - should return existing
	circuit2 := manager.GetOrCreate("provider-a")
	if circuit1 != circuit2 {
		t.Error("Expected same circuit instance on second call")
	}
	
	// Get different circuit
	circuit3 := manager.GetOrCreate("provider-b")
	if circuit1 == circuit3 {
		t.Error("Expected different circuit for different provider")
	}
}

// TestManagerGet tests manager Get
func TestManagerGet(t *testing.T) {
	manager := NewManager(DefaultConfig())
	
	// Get non-existent should return nil
	if manager.Get("nonexistent") != nil {
		t.Error("Expected nil for non-existent provider")
	}
	
	// Create circuit
	manager.GetOrCreate("test-provider")
	
	// Get existing should return circuit
	if manager.Get("test-provider") == nil {
		t.Error("Expected circuit for existing provider")
	}
}

// TestManagerList tests manager List
func TestManagerList(t *testing.T) {
	manager := NewManager(DefaultConfig())
	
	// Empty manager
	list := manager.List()
	if len(list) != 0 {
		t.Errorf("Expected empty list, got %d circuits", len(list))
	}
	
	// Add circuits
	manager.GetOrCreate("provider-a")
	manager.GetOrCreate("provider-b")
	manager.GetOrCreate("provider-c")
	
	list = manager.List()
	if len(list) != 3 {
		t.Errorf("Expected 3 circuits, got %d", len(list))
	}
}

// TestCircuitStateString tests state string representation
func TestCircuitStateString(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{StateClosed, "CLOSED"},
		{StateHalfOpen, "HALF_OPEN"},
		{StateOpen, "OPEN"},
		{CircuitState(99), "UNKNOWN"},
	}
	
	for _, test := range tests {
		if test.state.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, test.state.String())
		}
	}
}

// TestCircuitStats tests stats collection
func TestCircuitStats(t *testing.T) {
	circuit := NewCircuit("test-stats", DefaultConfig())
	
	// Record some activity
	circuit.RecordSuccess()
	circuit.RecordSuccess()
	
	stats := circuit.Stats()
	
	if stats.Provider != "test-stats" {
		t.Errorf("Expected provider 'test-stats', got '%s'", stats.Provider)
	}
	
	// Successes are counted even in closed state
	if stats.Successes != 2 {
		t.Errorf("Expected 2 successes, got %d", stats.Successes)
	}
	
	if stats.State != "CLOSED" {
		t.Errorf("Expected state CLOSED, got %s", stats.State)
	}
	
	// Now record failures - should open circuit
	for i := 0; i < 5; i++ {
		circuit.RecordFailure()
	}
	
	stats = circuit.Stats()
	
	// After transition to OPEN, failures stay at threshold (transition happens after increment)
	if stats.State != "OPEN" {
		t.Errorf("Expected state OPEN, got %s", stats.State)
	}
	if stats.Failures != 5 {
		t.Errorf("Expected 5 failures after OPEN, got %d", stats.Failures)
	}
}

// TestCircuitConcurrency tests thread safety
func TestCircuitConcurrency(t *testing.T) {
	circuit := NewCircuit("concurrent", DefaultConfig())
	var wg sync.WaitGroup
	
	// Concurrent reads and writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			circuit.Allow()
			circuit.State()
			circuit.Stats()
		}()
	}
	
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			circuit.RecordSuccess()
			circuit.RecordFailure()
		}()
	}
	
	wg.Wait()
	// If we get here without race detector warnings, test passes
}

// TestManagerConcurrency tests manager thread safety
func TestManagerConcurrency(t *testing.T) {
	manager := NewManager(DefaultConfig())
	var wg sync.WaitGroup
	
	// Concurrent gets and creates
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			manager.GetOrCreate("provider")
			manager.Get("provider")
			manager.List()
		}(i)
	}
	
	wg.Wait()
	// If we get here without race detector warnings, test passes
}
