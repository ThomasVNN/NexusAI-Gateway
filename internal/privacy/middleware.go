package privacy

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// FilterMiddleware provides HTTP middleware for privacy filtering
type FilterMiddleware struct {
	engine       *Engine
	auditLogger  AuditLogger
	enabled      bool
	includeBody  bool
	includeQuery bool
}

// FilterMiddlewareConfig holds configuration for the filter middleware
type FilterMiddlewareConfig struct {
	Engine         *Engine
	AuditLogger    AuditLogger
	Enabled        bool
	IncludeBody    bool   // Whether to filter request/response body
	IncludeQuery   bool   // Whether to filter query parameters
	IncludeHeaders bool   // Whether to filter headers
	ExcludePaths   []string // Paths to exclude from filtering
}

// DefaultFilterMiddlewareConfig returns default middleware configuration
func DefaultFilterMiddlewareConfig() *FilterMiddlewareConfig {
	return &FilterMiddlewareConfig{
		Engine:         NewEngine(),
		Enabled:        true,
		IncludeBody:    true,
		IncludeQuery:   false,
		IncludeHeaders: false,
		ExcludePaths:   []string{"/health", "/metrics", "/ready"},
	}
}

// NewFilterMiddleware creates a new filter middleware instance
func NewFilterMiddleware(config *FilterMiddlewareConfig) *FilterMiddleware {
	engine := config.Engine
	if engine == nil {
		engine = NewEngine()
	}

	return &FilterMiddleware{
		engine:       engine,
		auditLogger:  config.AuditLogger,
		enabled:      config.Enabled,
		includeBody:  config.IncludeBody,
		includeQuery: config.IncludeQuery,
	}
}

// Middleware returns an HTTP handler that wraps the given handler with privacy filtering
func (m *FilterMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled || m.shouldExclude(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract tenant and user info from context or headers
		tenantID := m.extractTenantID(r)
		userID := m.extractUserID(r)

		// Filter request body if enabled
		if m.includeBody && r.Body != nil {
			r = m.filterRequestBody(r, tenantID, userID)
		}

		// Filter query parameters if enabled
		if m.includeQuery {
			m.filterQueryParams(r)
		}

		// Create response wrapper for response body filtering
		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			recordedBody:   nil,
		}

		next.ServeHTTP(wrapper, r)
	})
}

// shouldExclude checks if the path should be excluded from filtering
func (m *FilterMiddleware) shouldExclude(path string) bool {
	for _, excluded := range []string{"/health", "/metrics", "/ready", "/status"} {
		if strings.HasPrefix(path, excluded) {
			return true
		}
	}
	return false
}

// filterRequestBody reads, filters, and replaces the request body
func (m *FilterMiddleware) filterRequestBody(r *http.Request, tenantID, userID string) *http.Request {
	// Read the body
	bodyBytes, err := readBody(r.Body)
	if err != nil {
		return r
	}

	// Parse as JSON if possible
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyMap); err == nil {
		// Filter the body map recursively
		filtered := m.filterValue(bodyMap)
		filteredBytes, _ := json.Marshal(filtered)
		bodyBytes = filteredBytes
	} else {
		// Filter as plain text
		bodyBytes = []byte(m.engine.Redact(string(bodyBytes)))
	}

	// Log the redaction if audit logger is configured
	if m.auditLogger != nil {
		counts := m.engine.GetRedactionCount(string(bodyBytes))
		if len(counts) > 0 {
			total := 0
			for _, c := range counts {
				total += c
			}
			_ = m.auditLogger.LogRedaction(r.Context(), tenantID, userID, total)
		}
	}

	// Replace the body
	r.Body = newReadCloser(bodyBytes)
	r.ContentLength = int64(len(bodyBytes))

	return r
}

// filterValue recursively filters a JSON value
func (m *FilterMiddleware) filterValue(v interface{}) interface{} {
	switch val := v.(type) {
	case string:
		return m.engine.Redact(val)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, v := range val {
			// Skip sensitive fields from filtering by name
			if isSensitiveField(k) {
				result[k] = v
			} else {
				result[k] = m.filterValue(v)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, v := range val {
			result[i] = m.filterValue(v)
		}
		return result
	default:
		return v
	}
}

// filterQueryParams filters query parameters
func (m *FilterMiddleware) filterQueryParams(r *http.Request) {
	values := r.URL.Query()
	changed := false

	for key, vals := range values {
		if isSensitiveField(key) {
			continue
		}
		for i, val := range vals {
			redacted := m.engine.Redact(val)
			if redacted != val {
				values[key][i] = redacted
				changed = true
			}
		}
	}

	if changed {
		r.URL.RawQuery = values.Encode()
	}
}

// extractTenantID extracts tenant ID from request
func (m *FilterMiddleware) extractTenantID(r *http.Request) string {
	// Check header first
	if tenant := r.Header.Get("X-Tenant-ID"); tenant != "" {
		return tenant
	}
	// Check context
	if tenant := r.Context().Value(contextKey("tenant_id")); tenant != nil {
		if s, ok := tenant.(string); ok {
			return s
		}
	}
	return "unknown"
}

// extractUserID extracts user ID from request
func (m *FilterMiddleware) extractUserID(r *http.Request) string {
	// Check header first
	if user := r.Header.Get("X-User-ID"); user != "" {
		return user
	}
	// Check context
	if user := r.Context().Value(contextKey("user_id")); user != nil {
		if s, ok := user.(string); ok {
			return s
		}
	}
	return "unknown"
}

// contextKey is used for context values
type contextKey string

// responseWriterWrapper wraps http.ResponseWriter to capture response body
type responseWriterWrapper struct {
	http.ResponseWriter
	recordedBody []byte
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	w.recordedBody = append(w.recordedBody, b...)
	return w.ResponseWriter.Write(b)
}

// readBody reads the entire body and returns it
func readBody(body io.ReadCloser) ([]byte, error) {
	defer body.Close()
	return io.ReadAll(body)
}

// newReadCloser creates a new ReadCloser from bytes
func newReadCloser(b []byte) *readCloser {
	return &readCloser{data: b}
}

type readCloser struct {
	data []byte
	pos  int
}

func (rc *readCloser) Read(p []byte) (n int, err error) {
	if rc.pos >= len(rc.data) {
		return 0, io.EOF
	}
	n = copy(p, rc.data[rc.pos:])
	rc.pos += n
	return n, nil
}

func (rc *readCloser) Close() error {
	return nil
}

// HTTPFilter provides HTTP-level filtering utilities
type HTTPFilter struct {
	engine *Engine
}

// NewHTTPFilter creates a new HTTP filter
func NewHTTPFilter(engine *Engine) *HTTPFilter {
	if engine == nil {
		engine = NewEngine()
	}
	return &HTTPFilter{engine: engine}
}

// FilterRequest filters an HTTP request's body
func (f *HTTPFilter) FilterRequest(r *http.Request) error {
	if r.Body == nil {
		return nil
	}

	body, err := readBody(r.Body)
	if err != nil {
		return err
	}

	// Try to parse as JSON
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(body, &bodyMap); err == nil {
		filtered := f.filterJSONValue(bodyMap)
		body, _ = json.Marshal(filtered)
	} else {
		body = []byte(f.engine.Redact(string(body)))
	}

	r.Body = newReadCloser(body)
	return nil
}

// FilterResponse filters an HTTP response's body
func (f *HTTPFilter) FilterResponse(body []byte) ([]byte, error) {
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(body, &bodyMap); err == nil {
		filtered := f.filterJSONValue(bodyMap)
		return json.Marshal(filtered)
	}
	return []byte(f.engine.Redact(string(body))), nil
}

// filterJSONValue recursively filters JSON values
func (f *HTTPFilter) filterJSONValue(v interface{}) interface{} {
	switch val := v.(type) {
	case string:
		return f.engine.Redact(val)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, v := range val {
			if isSensitiveField(k) {
				result[k] = v
			} else {
				result[k] = f.filterJSONValue(v)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, v := range val {
			result[i] = f.filterJSONValue(v)
		}
		return result
	default:
		return val
	}
}

// isSensitiveField checks if a field name suggests it should not be filtered
func isSensitiveField(fieldName string) bool {
	sensitiveFields := []string{
		"password", "secret", "token", "api_key", "apikey",
		"authorization", "credential", "private_key", "ssn", "credit_card",
	}
	lower := strings.ToLower(fieldName)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(lower, sensitive) {
			return true
		}
	}
	return false
}

// FilterContext provides context-aware filtering
type FilterContext struct {
	Engine      *Engine
	TenantID    string
	UserID      string
	RequestID   string
	TraceID     string
	Level       PrivacyLevel
	AuditLogger AuditLogger
}

// NewFilterContext creates a new filter context
func NewFilterContext(engine *Engine, tenantID, userID string) *FilterContext {
	return &FilterContext{
		Engine:   engine,
		TenantID: tenantID,
		UserID:   userID,
		Level:    PrivacyLevelMedium,
	}
}

// Filter applies filtering to the input with the current context
func (fc *FilterContext) Filter(ctx context.Context, input string) (string, []DetectResult, error) {
	detections := fc.Engine.Detect(input)
	redacted := fc.Engine.Redact(input)

	// Log audit entry
	if fc.AuditLogger != nil && len(detections) > 0 {
		auditEntry := &RedactionAuditEntry{
			Timestamp:     time.Now().UTC(),
			TenantID:      fc.TenantID,
			UserID:        fc.UserID,
			RequestID:     fc.RequestID,
			TotalRedacted: len(detections),
			Level:         fc.Level,
			Success:       true,
		}

		// Count by type
		counts := make(map[PIIType]int)
		for _, d := range detections {
			counts[d.Type]++
		}
		auditEntry.PIICounts = counts

		_ = fc.AuditLogger.LogRedaction(ctx, fc.TenantID, fc.UserID, len(detections))
		_ = fc.AuditLogger.LogRedactionDetailed(ctx, auditEntry)
	}

	return redacted, detections, nil
}

// ConfigAPI provides REST API endpoints for privacy filter configuration
type ConfigAPI struct {
	engine    *Engine
	recorder  *AuditRecorder
	mu        sync.RWMutex
}

// NewConfigAPI creates a new configuration API
func NewConfigAPI(engine *Engine) *ConfigAPI {
	return &ConfigAPI{
		engine:   engine,
		recorder: NewAuditRecorder(slog.Default(), 24*time.Hour),
	}
}

// ConfigResponse represents the API response for configuration
type ConfigResponse struct {
	EnabledTypes []TypeConfig    `json:"enabled_types"`
	AllTypes     []TypeConfig    `json:"all_types"`
	GlobalConfig *FilterConfig   `json:"global_config"`
	Stats        map[string]interface{} `json:"stats"`
}

// TypeConfig represents configuration for a single PII type
type TypeConfig struct {
	Type    PIIType `json:"type"`
	Marker  string `json:"marker"`
	Enabled bool   `json:"enabled"`
	Strict  bool   `json:"strict"`
}

// GetConfig returns the current configuration
func (api *ConfigAPI) GetConfig() *ConfigResponse {
	config := api.engine.GetConfig()

	var enabledTypes, allTypes []TypeConfig
	for piiType := range api.engine.detector.patterns {
		cfg := TypeConfig{
			Type:    piiType,
			Marker:  api.engine.GetMarker(piiType),
			Enabled: false,
		}

		if rule, ok := config.Rules[piiType]; ok {
			cfg.Enabled = rule.Enabled
		}

		allTypes = append(allTypes, cfg)
		if cfg.Enabled {
			enabledTypes = append(enabledTypes, cfg)
		}
	}

	return &ConfigResponse{
		EnabledTypes: enabledTypes,
		AllTypes:     allTypes,
		GlobalConfig: config,
		Stats:        api.recorder.GetStats(),
	}
}

// UpdateTypeConfig updates the configuration for a specific PII type
func (api *ConfigAPI) UpdateTypeConfig(piiType PIIType, enabled bool, marker string) error {
	api.mu.Lock()
	defer api.mu.Unlock()

	if enabled {
		api.engine.EnableType(piiType)
	} else {
		api.engine.DisableType(piiType)
	}

	if marker != "" {
		api.engine.SetMarker(piiType, marker)
	}

	return nil
}

// SetLevel sets the global redaction level
func (api *ConfigAPI) SetLevel(level RedactionLevel) {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.engine.SetLevel(level)
}

// SetMaxRedactions sets the maximum number of redactions
func (api *ConfigAPI) SetMaxRedactions(max int) {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.engine.SetMaxRedactions(max)
}

// GetAuditLogs returns audit logs with optional filtering
func (api *ConfigAPI) GetAuditLogs(tenantID, userID string) []*RedactionAuditEntry {
	if tenantID != "" {
		return api.recorder.GetEntriesByTenant(tenantID)
	}
	if userID != "" {
		return api.recorder.GetEntriesByUser(userID)
	}
	return api.recorder.GetEntries()
}

// TestRedaction tests redaction on a sample text and returns results
func (api *ConfigAPI) TestRedaction(text string) *RedactionResult {
	return api.engine.RedactWithResult(text)
}

// TestRedactionDetailed tests redaction with full detection details
func (api *ConfigAPI) TestRedactionDetailed(text string) map[string]interface{} {
	detections := api.engine.Detect(text)
	redacted := api.engine.Redact(text)

	counts := make(map[PIIType]int)
	for _, d := range detections {
		counts[d.Type]++
	}

	return map[string]interface{}{
		"original":   text,
		"redacted":   redacted,
		"detections": detections,
		"counts":     counts,
		"total":      len(detections),
	}
}
