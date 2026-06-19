package graceful

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServerCreation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	server := NewServer(":8080", handler, 30*time.Second)
	if server == nil {
		t.Error("Expected non-nil server")
	}
}

func TestWithTimeout(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	wrapped := WithTimeout(handler, 5*time.Second)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestWithTimeout_Timeout(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Write([]byte("OK"))
	})

	wrapped := WithTimeout(handler, 50*time.Millisecond)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestTimeout {
		t.Errorf("Expected status 408, got %d", rr.Code)
	}
}

func TestPoolCreation(t *testing.T) {
	pool := NewPool()
	if pool == nil {
		t.Error("Expected non-nil pool")
	}

	if len(pool.servers) != 0 {
		t.Errorf("Expected 0 servers, got %d", len(pool.servers))
	}
}

func TestPoolAdd(t *testing.T) {
	pool := NewPool()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	server := NewServer(":8080", handler, 30*time.Second)
	pool.Add(server)

	if len(pool.servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(pool.servers))
	}
}

func TestHealthCheck(t *testing.T) {
	handler := HealthCheck(func() bool {
		return true
	})

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestHealthCheck_Unhealthy(t *testing.T) {
	handler := HealthCheck(func() bool {
		return false
	})

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", rr.Code)
	}
}

func TestReadinessCheck(t *testing.T) {
	handler := ReadinessCheck(func() bool {
		return true
	})

	req := httptest.NewRequest("GET", "/ready", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestReadinessCheck_NotReady(t *testing.T) {
	handler := ReadinessCheck(func() bool {
		return false
	})

	req := httptest.NewRequest("GET", "/ready", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", rr.Code)
	}
}

func TestTimeoutWriter_WriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	tw := &timeoutWriter{ResponseWriter: rr, done: make(chan struct{})}

	tw.WriteHeader(http.StatusOK)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestTimeoutWriter_Write(t *testing.T) {
	rr := httptest.NewRecorder()
	tw := &timeoutWriter{ResponseWriter: rr, done: make(chan struct{})}

	n, err := tw.Write([]byte("Hello"))

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if n != 5 {
		t.Errorf("Expected 5 bytes written, got %d", n)
	}
}
