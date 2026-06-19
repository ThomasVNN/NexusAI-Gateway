package circuitbreaker

import (
	"testing"
	"time"
)

func TestCircuitBreaker_InitialState(t *testing.T) {
	cb := New(CircuitBreakerConfig{
		Name:         "test-service",
		FailureLimit: 3,
		SuccessLimit: 2,
		Timeout:      10 * time.Second,
	})

	if cb.GetState() != StateClosed {
		t.Error("Expected initial state to be closed")
	}
}

func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	cb := New(CircuitBreakerConfig{
		Name:         "test-service",
		FailureLimit: 3,
		SuccessLimit: 2,
		Timeout:      10 * time.Second,
	})

	for i := 0; i < 5; i++ {
		cb.RecordSuccess()
	}

	if cb.state != StateClosed {
		t.Errorf("Expected closed state, got %v", cb.state)
	}
}

func TestCircuitBreaker_RecordFailure(t *testing.T) {
	cb := New(CircuitBreakerConfig{
		Name:         "test-service",
		FailureLimit: 3,
		SuccessLimit: 2,
		Timeout:      10 * time.Second,
	})

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	if cb.GetState() != StateOpen {
		t.Errorf("Expected open state, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_RecordTimeout(t *testing.T) {
	cb := New(CircuitBreakerConfig{
		Name:         "test-service",
		FailureLimit: 3,
		SuccessLimit: 2,
		Timeout:      10 * time.Second,
	})

	for i := 0; i < 3; i++ {
		cb.RecordTimeout()
	}

	if cb.GetState() != StateOpen {
		t.Errorf("Expected open state, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_Allow(t *testing.T) {
	cb := New(CircuitBreakerConfig{
		Name:         "test-service",
		FailureLimit: 3,
		SuccessLimit: 2,
		Timeout:      10 * time.Second,
	})

	// Should allow in closed state
	if !cb.Allow() {
		t.Error("Expected allow in closed state")
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := New(CircuitBreakerConfig{
		Name:         "test-service",
		FailureLimit: 3,
		SuccessLimit: 2,
		Timeout:      10 * time.Second,
	})

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.GetState() != StateOpen {
		t.Errorf("Expected open state, got %v", cb.GetState())
	}

	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Errorf("Expected closed state after reset, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := New(CircuitBreakerConfig{
		Name:         "test-service",
		FailureLimit: 10,
		SuccessLimit: 5,
		Timeout:      10 * time.Second,
	})

	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			cb.RecordSuccess()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			cb.RecordFailure()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			cb.Allow()
		}
		done <- true
	}()

	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify no panic
	_ = cb.state
}

func TestManager_GetOrCreate(t *testing.T) {
	m := NewManager()

	cb1 := m.GetOrCreate("service-1")
	if cb1 == nil {
		t.Error("Expected non-nil breaker")
	}

	cb2 := m.GetOrCreate("service-1")
	if cb1 != cb2 {
		t.Error("Expected same breaker instance")
	}

	cb3 := m.GetOrCreate("service-2")
	if cb1 == cb3 {
		t.Error("Expected different breaker")
	}
}

func TestManager_List(t *testing.T) {
	m := NewManager()

	m.GetOrCreate("service-1")
	m.GetOrCreate("service-2")
	m.GetOrCreate("service-3")

	breakers := m.List()
	if len(breakers) != 3 {
		t.Errorf("Expected 3 breakers, got %d", len(breakers))
	}
}

func TestManager_Reset(t *testing.T) {
	m := NewManager()

	cb := m.GetOrCreate("service-1")
	// Manager default FailureLimit is 5
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	if cb.GetState() != StateOpen {
		t.Errorf("Expected open state, got %v", cb.GetState())
	}

	m.ResetAll()

	if cb.GetState() != StateClosed {
		t.Errorf("Expected closed state, got %v", cb.GetState())
	}
}

func TestManager_Delete(t *testing.T) {
	m := NewManager()

	m.GetOrCreate("service-1")
	if len(m.breakers) != 1 {
		t.Error("Expected 1 breaker")
	}

	m.Delete("service-1")
	if len(m.breakers) != 0 {
		t.Error("Expected 0 breakers")
	}
}
