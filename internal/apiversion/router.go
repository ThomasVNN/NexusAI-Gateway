package apiversion

import (
	"fmt"
	"net/http"
	"strings"
)

// Version represents an API version
type Version struct {
	Major int
	Minor int
	Str   string
}

// String returns the version string
func (v Version) String() string {
	return v.Str
}

// IsCompatible checks if this version is compatible with another
func (v Version) IsCompatible(other Version) bool {
	return v.Major == other.Major
}

// Compare compares two versions
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major > other.Major {
			return 1
		}
		return -1
	}
	if v.Minor != other.Minor {
		if v.Minor > other.Minor {
			return 1
		}
		return -1
	}
	return 0
}

// Latest returns the latest version from a list
func Latest(versions ...Version) Version {
	if len(versions) == 0 {
		return Version{}
	}

	latest := versions[0]
	for _, v := range versions[1:] {
		if v.Compare(latest) > 0 {
			latest = v
		}
	}
	return latest
}

// Parse parses a version string (e.g., "v1", "v2.0", "v1.2.3")
func Parse(s string) (Version, error) {
	s = strings.TrimPrefix(s, "v")

	parts := strings.Split(s, ".")
	major := 0
	minor := 0

	if len(parts) >= 1 {
		fmt.Sscanf(parts[0], "%d", &major)
	}
	if len(parts) >= 2 {
		fmt.Sscanf(parts[1], "%d", &minor)
	}

	return Version{
		Major: major,
		Minor: minor,
		Str:   "v" + s,
	}, nil
}

// Router handles API versioning
type Router struct {
	routes         map[string]map[Version]http.Handler
	versions       []Version
	defaultVersion Version
}

// NewRouter creates a new versioned router
func NewRouter(defaultV Version) *Router {
	return &Router{
		routes:         make(map[string]map[Version]http.Handler),
		versions:       []Version{},
		defaultVersion: defaultV,
	}
}

// Register registers a handler for a specific version
func (r *Router) Register(path string, version Version, handler http.Handler) {
	if r.routes[path] == nil {
		r.routes[path] = make(map[Version]http.Handler)
		r.versions = append(r.versions, version)
	}
	r.routes[path][version] = handler
}

// ServeHTTP routes requests based on version
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	version := r.getVersion(req)

	if handlers, ok := r.routes[path]; ok {
		if handler, ok := handlers[version]; ok {
			handler.ServeHTTP(w, req)
			return
		}

		// Try default version
		if handler, ok := handlers[r.defaultVersion]; ok {
			handler.ServeHTTP(w, req)
			return
		}
	}

	http.NotFound(w, req)
}

// getVersion extracts the version from the request
func (r *Router) getVersion(req *http.Request) Version {
	// Check header first
	if version := req.Header.Get("API-Version"); version != "" {
		if v, err := Parse(version); err == nil {
			return v
		}
	}

	// Check URL path
	path := req.URL.Path
	parts := strings.Split(path, "/")

	for _, part := range parts {
		if strings.HasPrefix(part, "v") && len(part) > 1 {
			if v, err := Parse(part); err == nil {
				return v
			}
		}
	}

	return r.defaultVersion
}

// SupportedVersions returns all supported versions
func (r *Router) SupportedVersions() []Version {
	return r.versions
}

// Middleware adds version headers to responses
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("API-Version", "v1")
		w.Header().Set("X-API-Version", "1.0")
		next.ServeHTTP(w, r)
	})
}

// DeprecationMiddleware adds deprecation headers
func DeprecationMiddleware(next http.Handler, sunsetDate string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Deprecation", "true")
		w.Header().Set("Sunset", sunsetDate)
		next.ServeHTTP(w, r)
	})
}

// VersionHeaderMiddleware validates version requirements
func VersionHeaderMiddleware(next http.Handler, required Version) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		versionStr := r.Header.Get("API-Version")
		if versionStr != "" {
			version, err := Parse(versionStr)
			if err == nil && version.IsCompatible(required) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Version header missing or incompatible
		w.Header().Set("API-Version", required.String())
		next.ServeHTTP(w, r)
	})
}
