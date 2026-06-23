package guardrails

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Middleware provides HTTP middleware for guardrail enforcement
type Middleware struct {
	manager            *GuardrailManager
	mode               GuardrailMode
	disabledByHeader   string
	disabledByOrg      map[string]bool // Orgs that have disabled guardrails
	logger             *slog.Logger
	config             *MiddlewareConfig
	configWatcher      *ConfigWatcher
	mu                 sync.RWMutex
}

// MiddlewareConfig holds middleware configuration
type MiddlewareConfig struct {
	Enabled          bool     `json:"enabled"`
	DisabledHeader   string   `json:"disabled_header"`
	ExcludePaths     []string `json:"exclude_paths"`
	ExcludePrefixes  []string `json:"exclude_prefixes"`
	Timeout          time.Duration `json:"timeout"`
	HotReload        bool     `json:"hot_reload"`
	ConfigPath       string   `json:"config_path"`
}

// DefaultMiddlewareConfig returns default middleware configuration
func DefaultMiddlewareConfig() *MiddlewareConfig {
	return &MiddlewareConfig{
		Enabled:        true,
		DisabledHeader: "X-Omniroute-Disabled-Guardrails",
		ExcludePaths:   []string{"/health", "/metrics", "/ready", "/status"},
		ExcludePrefixes: []string{"/api/v1/health"},
		Timeout:        5 * time.Second,
		HotReload:      false,
		ConfigPath:     "",
	}
}

// NewMiddleware creates a new guardrail middleware
func NewMiddleware(mode GuardrailMode, manager *GuardrailManager) *Middleware {
	return &Middleware{
		manager:          manager,
		mode:             mode,
		disabledByHeader: "X-Omniroute-Disabled-Guardrails",
		disabledByOrg:    make(map[string]bool),
		logger:           slog.Default(),
		config:           DefaultMiddlewareConfig(),
	}
}

// NewMiddlewareWithConfig creates middleware with custom config
func NewMiddlewareWithConfig(config *MiddlewareConfig, manager *GuardrailManager) (*Middleware, error) {
	m := &Middleware{
		manager:          manager,
		disabledByHeader: config.DisabledHeader,
		disabledByOrg:    make(map[string]bool),
		logger:           slog.Default(),
		config:           config,
	}

	// Start config watcher if hot reload is enabled
	if config.HotReload && config.ConfigPath != "" {
		watcher, err := NewConfigWatcher(config.ConfigPath, m.onConfigUpdate)
		if err != nil {
			m.logger.Warn("Failed to start config watcher",
				slog.String("path", config.ConfigPath),
				slog.String("error", err.Error()),
			)
		} else {
			m.configWatcher = watcher
		}
	}

	return m, nil
}

// Handler returns the HTTP middleware handler
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip if disabled
		if !m.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Check if guardrails are disabled for this request
		if m.isDisabled(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Check exclude paths
		if m.shouldExclude(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Create context with timeout
		ctx, cancel := context.WithTimeout(r.Context(), m.config.Timeout)
		defer cancel()

		// Build guardrail context from request
		gc := m.buildContext(r)

		// Run pre-call guardrails
		results, err := m.manager.RunStage(ctx, gc, StagePreCall)
		if err != nil {
			m.logger.Error("Guardrail check failed",
				slog.String("path", r.URL.Path),
				slog.String("error", err.Error()),
			)
			// Continue on error, don't block
		}

		// Check if blocked
		for _, result := range results {
			if result.Action == ActionBlock && !result.Passed {
				m.handleBlock(w, r, result)
				return
			}
		}

		// Store results in context for post-call checks
		ctx = context.WithValue(ctx, contextKey("guardrail_results"), results)
		ctx = context.WithValue(ctx, contextKey("guardrail_context"), gc)
		r = r.WithContext(ctx)

		// Wrap response writer to capture response for post-call guardrails
		wrapper := &responseWriter{
			ResponseWriter: w,
			recordedBody:   nil,
		}

		next.ServeHTTP(wrapper, r)

		// Run post-call guardrails
		m.runPostCallGuardrails(ctx, gc, wrapper)
	})
}

// HandlerFunc returns the middleware as an http.HandlerFunc
func (m *Middleware) HandlerFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.Handler(next).ServeHTTP(w, r)
	}
}

// buildContext creates a GuardrailContext from HTTP request
func (m *Middleware) buildContext(r *http.Request) *GuardrailContext {
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	gc := &GuardrailContext{
		Headers:        headers,
		Timestamp:      time.Now(),
		OrganizationID: m.extractOrgID(r),
		UserID:         m.extractUserID(r),
		Metadata:       make(map[string]interface{}),
	}

	// Extract request body if present
	if r.Body != nil {
		// Body will be parsed by the handler
		gc.Metadata["content_type"] = r.Header.Get("Content-Type")
	}

	// Extract trace info
	gc.TraceID = r.Header.Get("X-Trace-ID")
	gc.SpanID = r.Header.Get("X-Span-ID")

	return gc
}

// extractOrgID extracts organization ID from request
func (m *Middleware) extractOrgID(r *http.Request) string {
	// Check header first
	if org := r.Header.Get("X-Organization-ID"); org != "" {
		return org
	}
	if org := r.Header.Get("X-Org-ID"); org != "" {
		return org
	}
	if org := r.Header.Get("X-Tenant-ID"); org != "" {
		return org
	}
	return "unknown"
}

// extractUserID extracts user ID from request
func (m *Middleware) extractUserID(r *http.Request) string {
	// Check header first
	if user := r.Header.Get("X-User-ID"); user != "" {
		return user
	}
	if user := r.Header.Get("X-User-ID"); user != "" {
		return user
	}
	return "unknown"
}

// isDisabled checks if guardrails are disabled for this request
func (m *Middleware) isDisabled(r *http.Request) bool {
	// Check header
	disabledHeader := r.Header.Get(m.disabledByHeader)
	if disabledHeader != "" {
		return true
	}

	// Check org-level disable
	orgID := m.extractOrgID(r)
	if m.disabledByOrg[orgID] {
		return true
	}

	return false
}

// shouldExclude checks if path should be excluded
func (m *Middleware) shouldExclude(path string) bool {
	// Check exact paths
	for _, excluded := range m.config.ExcludePaths {
		if path == excluded {
			return true
		}
	}

	// Check prefixes
	for _, prefix := range m.config.ExcludePrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// handleBlock handles blocked requests
func (m *Middleware) handleBlock(w http.ResponseWriter, r *http.Request, result *GuardrailResult) {
	m.logger.Warn("Request blocked by guardrail",
		slog.String("path", r.URL.Path),
		slog.String("message", result.Message),
		slog.Int("detections", len(result.Detections)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	
	response := map[string]interface{}{
		"error":   "request_blocked",
		"message":  "Request blocked by security guardrail",
		"details": result.Message,
		"code":    "GUARDRAIL_BLOCK",
	}
	json.NewEncoder(w).Encode(response)
}

// runPostCallGuardrails runs post-call guardrail checks
func (m *Middleware) runPostCallGuardrails(ctx context.Context, gc *GuardrailContext, w *responseWriter) {
	if len(w.recordedBody) == 0 {
		return
	}

	// Build response
	gc.Response = &AIResponse{
		Content: string(w.recordedBody),
	}

	// Run post-call guardrails
	_, _ = m.manager.RunStage(ctx, gc, StagePostCall)
}

// DisableForOrg disables guardrails for a specific organization
func (m *Middleware) DisableForOrg(orgID string) {
	m.disabledByOrg[orgID] = true
}

// EnableForOrg enables guardrails for a specific organization
func (m *Middleware) EnableForOrg(orgID string) {
	delete(m.disabledByOrg, orgID)
}

// SetMode updates the middleware mode
func (m *Middleware) SetMode(mode GuardrailMode) {
	m.mode = mode
	m.manager.SetMode(mode)
}

// SetEnabled enables or disables the middleware
func (m *Middleware) SetEnabled(enabled bool) {
	m.config.Enabled = enabled
}

// onConfigUpdate handles configuration updates
func (m *Middleware) onConfigUpdate(data []byte) error {
	var config MiddlewareConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	m.mu.Lock()
	m.config = &config
	m.mu.Unlock()

	// Update mode if changed
	if config.Enabled {
		m.manager.SetMode(m.mode)
	}

	m.logger.Info("Guardrail middleware config updated",
		slog.Bool("enabled", config.Enabled),
		slog.String("mode", string(m.mode)),
	)

	return nil
}

// Reload reloads the configuration from file
func (m *Middleware) Reload() error {
	if m.configWatcher != nil {
		return m.configWatcher.Reload()
	}
	return nil
}

// Close closes the middleware and stops watching
func (m *Middleware) Close() error {
	if m.configWatcher != nil {
		return m.configWatcher.Close()
	}
	return nil
}

// responseWriter wraps http.ResponseWriter to capture response body
type responseWriter struct {
	http.ResponseWriter
	recordedBody []byte
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.recordedBody = append(w.recordedBody, b...)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

// contextKey type for context keys
type contextKey string

const (
	contextKeyResults contextKey = "guardrail_results"
	contextKeyGC     contextKey = "guardrail_context"
)

// ConfigWatcher watches a config file for changes
type ConfigWatcher struct {
	path      string
	onUpdate  func([]byte) error
	interval  time.Duration
	lastHash  string
	stopCh    chan struct{}
}

// NewConfigWatcher creates a new config watcher
func NewConfigWatcher(path string, onUpdate func([]byte) error) (*ConfigWatcher, error) {
	w := &ConfigWatcher{
		path:     path,
		onUpdate: onUpdate,
		interval: 30 * time.Second,
		stopCh:   make(chan struct{}),
	}

	// Initial load
	if err := w.Reload(); err != nil {
		return nil, err
	}

	// Start watching
	go w.watch()

	return w, nil
}

// watch monitors the config file
func (w *ConfigWatcher) watch() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = w.Reload()
		case <-w.stopCh:
			return
		}
	}
}

// Reload reloads the config file
func (w *ConfigWatcher) Reload() error {
	data, err := readFile(w.path)
	if err != nil {
		return err
	}

	// Check if content changed
	hash := hashBytes(data)
	if hash == w.lastHash {
		return nil
	}

	w.lastHash = hash
	return w.onUpdate(data)
}

// Close stops the watcher
func (w *ConfigWatcher) Close() error {
	close(w.stopCh)
	return nil
}

// Helper to read file
func readFile(path string) ([]byte, error) {
	// Simple implementation - in production use os.ReadFile
	return nil, nil
}

// Simple hash function
func hashBytes(data []byte) string {
	h := 0
	for _, b := range data {
		h = h*31 + int(b)
	}
	return string(rune(h))
}
