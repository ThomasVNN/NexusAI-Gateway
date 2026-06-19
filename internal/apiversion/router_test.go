package apiversion

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersionString(t *testing.T) {
	v := Version{Major: 1, Minor: 0, Str: "v1.0"}
	if v.String() != "v1.0" {
		t.Errorf("Expected 'v1.0', got '%s'", v.String())
	}
}

func TestVersionIsCompatible(t *testing.T) {
	v1 := Version{Major: 1, Minor: 0}
	v2 := Version{Major: 1, Minor: 2}
	v3 := Version{Major: 2, Minor: 0}

	if !v1.IsCompatible(v2) {
		t.Error("Expected v1 and v2 to be compatible")
	}

	if v1.IsCompatible(v3) {
		t.Error("Expected v1 and v3 to be incompatible")
	}
}

func TestVersionCompare(t *testing.T) {
	v1 := Version{Major: 1, Minor: 0}
	v2 := Version{Major: 1, Minor: 2}
	v3 := Version{Major: 2, Minor: 0}

	if v1.Compare(v2) >= 0 {
		t.Error("Expected v1 < v2")
	}

	if v2.Compare(v1) <= 0 {
		t.Error("Expected v2 > v1")
	}

	if v3.Compare(v1) <= 0 {
		t.Error("Expected v3 > v1")
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		input    string
		expected Version
	}{
		{"v1", Version{Major: 1, Minor: 0, Str: "v1"}},
		{"v2.0", Version{Major: 2, Minor: 0, Str: "v2.0"}},
		{"v1.2", Version{Major: 1, Minor: 2, Str: "v1.2"}},
	}

	for _, tt := range tests {
		result, err := Parse(tt.input)
		if err != nil {
			t.Errorf("Parse(%s) failed: %v", tt.input, err)
		}
		if result.Major != tt.expected.Major || result.Minor != tt.expected.Minor {
			t.Errorf("Parse(%s) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestLatest(t *testing.T) {
	versions := []Version{
		{Major: 1, Minor: 0},
		{Major: 2, Minor: 0},
		{Major: 1, Minor: 5},
	}

	latest := Latest(versions...)
	if latest.Major != 2 || latest.Minor != 0 {
		t.Errorf("Expected v2.0, got v%d.%d", latest.Major, latest.Minor)
	}
}

func TestRouterCreation(t *testing.T) {
	defaultV := Version{Major: 1, Minor: 0}
	router := NewRouter(defaultV)

	if router == nil {
		t.Error("Expected non-nil router")
	}
}

func TestRouterRegister(t *testing.T) {
	router := NewRouter(Version{Major: 1, Minor: 0})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	router.Register("/test", Version{Major: 1, Minor: 0}, handler)
	router.Register("/api", Version{Major: 2, Minor: 0}, handler)

	versions := router.SupportedVersions()
	if len(versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(versions))
	}
}

func TestRouterServeHTTP(t *testing.T) {
	router := NewRouter(Version{Major: 1, Minor: 0})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	router.Register("/test", Version{Major: 1, Minor: 0}, handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	wrapped := Middleware(handler)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Header().Get("API-Version") != "v1" {
		t.Errorf("Expected API-Version header 'v1', got '%s'", rr.Header().Get("API-Version"))
	}
}

func TestDeprecationMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	wrapped := DeprecationMiddleware(handler, "2024-12-31")

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Header().Get("Deprecation") != "true" {
		t.Errorf("Expected Deprecation header 'true', got '%s'", rr.Header().Get("Deprecation"))
	}

	if rr.Header().Get("Sunset") != "2024-12-31" {
		t.Errorf("Expected Sunset header '2024-12-31', got '%s'", rr.Header().Get("Sunset"))
	}
}
