package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddlewareCreation(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           86400,
	}

	middleware := NewCORSMiddleware(config)
	if middleware == nil {
		t.Error("Expected non-nil middleware")
	}
}

func TestCORSMiddlewarePreflightRequest(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           86400,
	}

	middleware := NewCORSMiddleware(config)

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Simulate preflight request
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Preflight should return 204 No Content
	if rr.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rr.Code)
	}

	// Check CORS headers
	if rr.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("Expected Access-Control-Allow-Origin header")
	}
}

func TestCORSMiddlewareActualRequest(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:     []string{"http://example.com"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           86400,
	}

	middleware := NewCORSMiddleware(config)

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://example.com")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if rr.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin 'http://example.com', got '%s'", rr.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddlewareDisallowedOrigin(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:     []string{"http://allowed.com"},
		AllowMethods:     []string{"GET"},
		AllowHeaders:     []string{},
		AllowCredentials: false,
		MaxAge:           0,
	}

	middleware := NewCORSMiddleware(config)

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://disallowed.com")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Request should still succeed (CORS doesn't block, just omits headers)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// CORS header should be missing
	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("Expected no Access-Control-Allow-Origin for disallowed origin")
	}
}

func TestCORSMiddlewareCredentials(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET"},
		AllowHeaders:     []string{},
		AllowCredentials: true,
		MaxAge:           0,
	}

	middleware := NewCORSMiddleware(config)

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://example.com")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("Expected Access-Control-Allow-Credentials 'true'")
	}
}

func TestCORSMiddlewareMaxAge(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET"},
		AllowHeaders:     []string{},
		AllowCredentials: false,
		MaxAge:           3600,
	}

	middleware := NewCORSMiddleware(config)

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Max-Age") != "3600" {
		t.Errorf("Expected Access-Control-Max-Age '3600', got '%s'", rr.Header().Get("Access-Control-Max-Age"))
	}
}

func TestCORSMiddlewareExposeHeaders(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET"},
		AllowHeaders:     []string{},
		ExposeHeaders:    []string{"X-Custom-Header", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           0,
	}

	middleware := NewCORSMiddleware(config)

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://example.com")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Expose-Headers") == "" {
		t.Error("Expected Access-Control-Expose-Headers header")
	}
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// CORSMiddleware handles Cross-Origin Resource Sharing
type CORSMiddleware struct {
	config CORSConfig
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(config CORSConfig) *CORSMiddleware {
	return &CORSMiddleware{config: config}
}

// Middleware returns the CORS middleware handler
func (m *CORSMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, o := range m.config.AllowOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", join(m.config.AllowMethods, ", "))

			if len(m.config.AllowHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", join(m.config.AllowHeaders, ", "))
			}

			if len(m.config.ExposeHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", join(m.config.ExposeHeaders, ", "))
			}

			if m.config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if m.config.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", string(rune(m.config.MaxAge)))
			}
		}

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// join concatenates strings with separator
func join(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
