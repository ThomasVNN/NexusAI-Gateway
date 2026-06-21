package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTracerMiddleware(t *testing.T) {
	t.Run("creates middleware", func(t *testing.T) {
		m := NewTracerMiddleware("test-service")
		if m == nil {
			t.Error("NewTracerMiddleware returned nil")
		}
		if m.tracer == nil {
			t.Error("tracer is nil")
		}
	})
}

func TestTracerMiddleware_Middleware(t *testing.T) {
	t.Run("wraps handler with tracing", func(t *testing.T) {
		m := NewTracerMiddleware("test-service")

		// Create a simple handler
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Wrap with middleware
		wrapped := m.Middleware(handler)

		// Create request
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		// Execute
		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestStartSpanWithAttrs(t *testing.T) {
	t.Run("creates span with attributes", func(t *testing.T) {
		ctx := context.Background()
		attrs := map[string]interface{}{
			"test.key":  "test-value",
			"test.int":  42,
			"test.bool": true,
		}

		ctx, span := StartSpanWithAttrs(ctx, "test-span", attrs)
		defer span.End()

		if span == nil {
			t.Error("span is nil")
		}

		// Verify context is returned
		if ctx == nil {
			t.Error("context is nil")
		}
	})
}

func TestStartProviderSpan(t *testing.T) {
	t.Run("creates provider span", func(t *testing.T) {
		ctx := context.Background()
		ctx, span := StartProviderSpan(ctx, "gpt-4", "openai")
		defer span.End()

		if span == nil {
			t.Error("span is nil")
		}
	})
}

func TestStartCacheSpan(t *testing.T) {
	t.Run("creates cache span", func(t *testing.T) {
		ctx := context.Background()
		ctx, span := StartCacheSpan(ctx, "cache-key", "response")
		defer span.End()

		if span == nil {
			t.Error("span is nil")
		}
	})
}

func TestStartDBSpan(t *testing.T) {
	t.Run("creates DB span", func(t *testing.T) {
		ctx := context.Background()
		ctx, span := StartDBSpan(ctx, "SELECT")
		defer span.End()

		if span == nil {
			t.Error("span is nil")
		}
	})
}

func TestRecordTokens(t *testing.T) {
	t.Run("records tokens on current span", func(t *testing.T) {
		ctx := context.Background()
		// No span in context - should not panic
		RecordTokens(ctx, 100, 50)
	})
}

func TestRecordError(t *testing.T) {
	t.Run("records error on current span", func(t *testing.T) {
		ctx := context.Background()
		err := context.DeadlineExceeded
		// No span in context - should not panic
		RecordError(ctx, err, "test error")
	})
}

func TestRecordCacheHit(t *testing.T) {
	t.Run("records cache hit", func(t *testing.T) {
		ctx := context.Background()
		// No span in context - should not panic
		RecordCacheHit(ctx, true)
	})
}

func TestToAttribute(t *testing.T) {
	t.Run("converts string", func(t *testing.T) {
		attr := toAttribute("key", "value")
		if attr.Key != "key" {
			t.Errorf("Expected key 'key', got %s", attr.Key)
		}
	})

	t.Run("converts int", func(t *testing.T) {
		attr := toAttribute("key", 42)
		if attr.Value.AsInt64() != 42 {
			t.Errorf("Expected 42, got %d", attr.Value.AsInt64())
		}
	})

	t.Run("converts int64", func(t *testing.T) {
		attr := toAttribute("key", int64(42))
		if attr.Value.AsInt64() != 42 {
			t.Errorf("Expected 42, got %d", attr.Value.AsInt64())
		}
	})

	t.Run("converts float64", func(t *testing.T) {
		attr := toAttribute("key", 3.14)
		if attr.Value.AsFloat64() != 3.14 {
			t.Errorf("Expected 3.14, got %f", attr.Value.AsFloat64())
		}
	})

	t.Run("converts bool", func(t *testing.T) {
		attr := toAttribute("key", true)
		if !attr.Value.AsBool() {
			t.Error("Expected true")
		}
	})
}

func TestGetClientIP(t *testing.T) {
	t.Run("extracts from X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.1")
		ip := getClientIP(req)
		if ip != "192.168.1.1" {
			t.Errorf("Expected 192.168.1.1, got %s", ip)
		}
	})

	t.Run("extracts from X-Real-IP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Real-IP", "10.0.0.1")
		ip := getClientIP(req)
		if ip != "10.0.0.1" {
			t.Errorf("Expected 10.0.0.1, got %s", ip)
		}
	})

	t.Run("falls back to RemoteAddr", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		ip := getClientIP(req)
		if ip != "127.0.0.1:12345" {
			t.Errorf("Expected 127.0.0.1:12345, got %s", ip)
		}
	})
}

func TestResponseWriterWrapper(t *testing.T) {
	t.Run("captures status code", func(t *testing.T) {
		rec := httptest.NewRecorder()
		wrapper := &responseWriter{ResponseWriter: rec, statusCode: 0}

		wrapper.WriteHeader(http.StatusNotFound)

		if wrapper.statusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", wrapper.statusCode)
		}
	})

	t.Run("sets default status on Write", func(t *testing.T) {
		rec := httptest.NewRecorder()
		wrapper := &responseWriter{ResponseWriter: rec, statusCode: 0}

		wrapper.Write([]byte("test"))

		if wrapper.statusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", wrapper.statusCode)
		}
	})
}
