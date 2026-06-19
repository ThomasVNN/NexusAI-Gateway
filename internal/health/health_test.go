package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestManagerCreation(t *testing.T) {
	m := NewManager(5 * time.Second)
	if m == nil {
		t.Error("Expected non-nil manager")
	}
}

func TestManagerRegister(t *testing.T) {
	m := NewManager(time.Second)

	checker := NewSimpleChecker("test", func(ctx context.Context) error {
		return nil
	})

	m.Register(checker)

	if len(m.checks) != 1 {
		t.Errorf("Expected 1 check, got %d", len(m.checks))
	}
}

func TestManagerUnregister(t *testing.T) {
	m := NewManager(time.Second)

	checker := NewSimpleChecker("test", func(ctx context.Context) error {
		return nil
	})

	m.Register(checker)
	m.Unregister("test")

	if len(m.checks) != 0 {
		t.Errorf("Expected 0 checks, got %d", len(m.checks))
	}
}

func TestManagerCheck_AllHealthy(t *testing.T) {
	m := NewManager(time.Second)

	m.Register(NewSimpleChecker("check1", func(ctx context.Context) error {
		return nil
	}))
	m.Register(NewSimpleChecker("check2", func(ctx context.Context) error {
		return nil
	}))

	status := m.Check(context.Background())

	if status.Status != StatusHealthy {
		t.Errorf("Expected %s, got %s", StatusHealthy, status.Status)
	}

	if len(status.Checks) != 2 {
		t.Errorf("Expected 2 checks, got %d", len(status.Checks))
	}
}

func TestManagerCheck_OneUnhealthy(t *testing.T) {
	m := NewManager(time.Second)

	m.Register(NewSimpleChecker("healthy", func(ctx context.Context) error {
		return nil
	}))
	m.Register(NewSimpleChecker("unhealthy", func(ctx context.Context) error {
		return errors.New("test error")
	}))

	status := m.Check(context.Background())

	if status.Status != StatusUnhealthy {
		t.Errorf("Expected %s, got %s", StatusUnhealthy, status.Status)
	}
}

func TestManagerCheck_Latency(t *testing.T) {
	m := NewManager(time.Second)

	m.Register(NewSimpleChecker("slow", func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}))

	status := m.Check(context.Background())

	result, ok := status.Checks["slow"]
	if !ok {
		t.Error("Expected check result for 'slow'")
	}

	if result.LatencyMs < 5 {
		t.Errorf("Expected latency >= 5ms, got %dms", result.LatencyMs)
	}
}

func TestSimpleChecker_Name(t *testing.T) {
	checker := NewSimpleChecker("my-check", func(ctx context.Context) error {
		return nil
	})

	if checker.Name() != "my-check" {
		t.Errorf("Expected 'my-check', got '%s'", checker.Name())
	}
}

func TestSimpleChecker_Check_Success(t *testing.T) {
	checker := NewSimpleChecker("test", func(ctx context.Context) error {
		return nil
	})

	err := checker.Check(context.Background())
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func TestSimpleChecker_Check_Error(t *testing.T) {
	expectedErr := errors.New("test error")
	checker := NewSimpleChecker("test", func(ctx context.Context) error {
		return expectedErr
	})

	err := checker.Check(context.Background())
	if err != expectedErr {
		t.Errorf("Expected %v, got %v", expectedErr, err)
	}
}

func TestHandler(t *testing.T) {
	m := NewManager(time.Second)

	m.Register(NewSimpleChecker("test", func(ctx context.Context) error {
		return nil
	}))

	handler := m.Handler()

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestHandler_Unhealthy(t *testing.T) {
	m := NewManager(time.Second)

	m.Register(NewSimpleChecker("test", func(ctx context.Context) error {
		return errors.New("failing")
	}))

	handler := m.Handler()

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", rr.Code)
	}
}

func TestReadyHandler(t *testing.T) {
	m := NewManager(time.Second)

	m.Register(NewSimpleChecker("test", func(ctx context.Context) error {
		return nil
	}))

	handler := m.ReadyHandler()

	req := httptest.NewRequest("GET", "/ready", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestReadyHandler_Degraded(t *testing.T) {
	m := NewManager(time.Second)

	m.Register(NewSimpleChecker("healthy", func(ctx context.Context) error {
		return nil
	}))
	m.Register(NewSimpleChecker("unhealthy", func(ctx context.Context) error {
		return errors.New("unhealthy")
	}))

	handler := m.ReadyHandler()

	req := httptest.NewRequest("GET", "/ready", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// With one unhealthy check, should return 503
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", rr.Code)
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusHealthy != "healthy" {
		t.Errorf("Expected 'healthy', got '%s'", StatusHealthy)
	}
	if StatusDegraded != "degraded" {
		t.Errorf("Expected 'degraded', got '%s'", StatusDegraded)
	}
	if StatusUnhealthy != "unhealthy" {
		t.Errorf("Expected 'unhealthy', got '%s'", StatusUnhealthy)
	}
}
