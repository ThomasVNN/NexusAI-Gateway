package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTimeoutMiddlewareCreation(t *testing.T) {
	config := DefaultTimeoutConfig()
	if config.ReadTimeout != 30*time.Second {
		t.Errorf("Expected 30s read timeout, got %v", config.ReadTimeout)
	}
}

func TestTimeoutMiddlewareSuccess(t *testing.T) {
	middleware := TimeoutMiddleware(DefaultTimeoutConfig())

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestTimeoutMiddlewareWithTimeout(t *testing.T) {
	middleware := WithTimeout(5 * time.Second)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestTimeoutMiddlewareSlowHandler(t *testing.T) {
	middleware := WithTimeout(50 * time.Millisecond)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should timeout
	if rr.Code != http.StatusGatewayTimeout {
		t.Errorf("Expected status 504, got %d", rr.Code)
	}
}

func TestTimeoutHandler(t *testing.T) {
	handler := TimeoutHandler(time.Second, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestReadTimeoutMiddleware(t *testing.T) {
	middleware := ReadTimeoutMiddleware(time.Second)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestWriteTimeoutMiddleware(t *testing.T) {
	middleware := WriteTimeoutMiddleware(time.Second)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestTimeoutWriterWriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	tw := &timeoutWriter{
		ResponseWriter: rr,
		code:           http.StatusOK,
		done:           make(chan struct{}),
	}

	tw.WriteHeader(http.StatusOK)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Second call should not change status
	tw.WriteHeader(http.StatusInternalServerError)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status to remain 200, got %d", rr.Code)
	}
}

func TestTimeoutWriterWrite(t *testing.T) {
	rr := httptest.NewRecorder()
	tw := &timeoutWriter{
		ResponseWriter: rr,
		code:           http.StatusOK,
		done:           make(chan struct{}),
	}

	n, err := tw.Write([]byte("Hello, World!"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if n != 13 {
		t.Errorf("Expected 13 bytes written, got %d", n)
	}
}
