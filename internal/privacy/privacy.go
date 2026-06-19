package privacy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// PrivacyLevel defines the privacy enforcement strength
type PrivacyLevel int

const (
	PrivacyLevelLow PrivacyLevel = iota
	PrivacyLevelMedium
	PrivacyLevelHigh
	PrivacyLevelStrict
)

// String returns the string representation of PrivacyLevel
func (p PrivacyLevel) String() string {
	switch p {
	case PrivacyLevelLow:
		return "low"
	case PrivacyLevelMedium:
		return "medium"
	case PrivacyLevelHigh:
		return "high"
	case PrivacyLevelStrict:
		return "strict"
	default:
		return "unknown"
	}
}

// ParsePrivacyLevel parses a string to PrivacyLevel
func ParsePrivacyLevel(s string) PrivacyLevel {
	switch s {
	case "low":
		return PrivacyLevelLow
	case "medium":
		return PrivacyLevelMedium
	case "high":
		return PrivacyLevelHigh
	case "strict":
		return PrivacyLevelStrict
	default:
		return PrivacyLevelMedium
	}
}

// PrivacyFilter defines the contract for scanning and altering data for privacy safety
type PrivacyFilter interface {
	Process(ctx context.Context, input string, level PrivacyLevel) (string, error)
}

// AuditLogger defines how privacy violations and operations are logged for compliance
type AuditLogger interface {
	LogAccess(ctx context.Context, tenantID, userID, resource, action string, allowed bool) error
	LogRedaction(ctx context.Context, tenantID, userID string, fieldCount int) error
	LogRedactionDetailed(ctx context.Context, entry *RedactionAuditEntry) error
}

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	TenantID    string                 `json:"tenant_id"`
	UserID      string                 `json:"user_id"`
	Resource    string                 `json:"resource"`
	Action      string                 `json:"action"`
	Allowed     bool                   `json:"allowed"`
	Redactions  int                    `json:"redactions,omitempty"`
	PIITypes    []PIIType              `json:"pii_types,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	TraceID     string                 `json:"trace_id,omitempty"`
	Correlation string                 `json:"correlation_id,omitempty"`
}

// RedactionAuditEntry represents an audit entry for redaction operations
type RedactionAuditEntry struct {
	Timestamp     time.Time       `json:"timestamp"`
	TenantID      string          `json:"tenant_id"`
	UserID        string          `json:"user_id"`
	RequestID     string          `json:"request_id"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	OriginalHash  string          `json:"original_hash,omitempty"`
	RedactedHash  string          `json:"redacted_hash,omitempty"`
	PIICounts     map[PIIType]int `json:"pii_counts"`
	TotalRedacted int             `json:"total_redacted"`
	Level         PrivacyLevel    `json:"level"`
	DurationMs    int64           `json:"duration_ms"`
	Success       bool            `json:"success"`
	Error         string          `json:"error,omitempty"`
}

// MarshalJSON implements json.Marshaler for RedactionAuditEntry
func (r *RedactionAuditEntry) MarshalJSON() ([]byte, error) {
	type Alias RedactionAuditEntry
	return json.Marshal(&struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     (*Alias)(r),
		Timestamp: r.Timestamp.Format(time.RFC3339),
	})
}

// StandardAuditLogger provides a standard implementation of AuditLogger using slog
type StandardAuditLogger struct {
	logger   *slog.Logger
	format   string
	redacted bool
}

// AuditLoggerConfig holds configuration for the standard audit logger
type AuditLoggerConfig struct {
	Logger        *slog.Logger
	Format        string // "json" or "text"
	LogRedacted   bool   // Whether to log redacted content hashes
	LogPIIDetails bool   // Whether to log PII type details
	RetentionDays int    // Days to retain audit logs (0 = indefinite)
}

// DefaultAuditLoggerConfig returns default audit logger configuration
func DefaultAuditLoggerConfig() *AuditLoggerConfig {
	return &AuditLoggerConfig{
		Logger:        slog.Default(),
		Format:        "json",
		LogRedacted:   false,
		LogPIIDetails: true,
		RetentionDays: 90,
	}
}

// NewStandardAuditLogger creates a new standard audit logger
func NewStandardAuditLogger(config *AuditLoggerConfig) *StandardAuditLogger {
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &StandardAuditLogger{
		logger:   logger,
		format:   config.Format,
		redacted: config.LogRedacted,
	}
}

// LogAccess logs an access event for compliance
func (l *StandardAuditLogger) LogAccess(ctx context.Context, tenantID, userID, resource, action string, allowed bool) error {
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		TenantID:  tenantID,
		UserID:    userID,
		Resource:  resource,
		Action:    action,
		Allowed:   allowed,
	}

	l.logger.Info("Privacy access audit",
		slog.String("event_type", "access"),
		slog.String("tenant_id", tenantID),
		slog.String("user_id", userID),
		slog.String("resource", resource),
		slog.String("action", action),
		slog.Bool("allowed", allowed),
		slog.Time("timestamp", entry.Timestamp),
	)

	return nil
}

// LogRedaction logs a redaction event for compliance
func (l *StandardAuditLogger) LogRedaction(ctx context.Context, tenantID, userID string, fieldCount int) error {
	l.logger.Info("Privacy redaction audit",
		slog.String("event_type", "redaction"),
		slog.String("tenant_id", tenantID),
		slog.String("user_id", userID),
		slog.Int("field_count", fieldCount),
		slog.Time("timestamp", time.Now().UTC()),
	)

	return nil
}

// LogRedactionDetailed logs a detailed redaction event with full metadata
func (l *StandardAuditLogger) LogRedactionDetailed(ctx context.Context, entry *RedactionAuditEntry) error {
	attrs := []slog.Attr{
		slog.String("event_type", "redaction_detailed"),
		slog.String("tenant_id", entry.TenantID),
		slog.String("user_id", entry.UserID),
		slog.String("request_id", entry.RequestID),
		slog.Int("total_redacted", entry.TotalRedacted),
		slog.String("level", entry.Level.String()),
		slog.Int64("duration_ms", entry.DurationMs),
		slog.Bool("success", entry.Success),
		slog.Time("timestamp", entry.Timestamp),
	}

	if entry.CorrelationID != "" {
		attrs = append(attrs, slog.String("correlation_id", entry.CorrelationID))
	}

	if entry.Error != "" {
		attrs = append(attrs, slog.String("error", entry.Error))
	}

	if l.redacted && entry.OriginalHash != "" {
		attrs = append(attrs, slog.String("original_hash", entry.OriginalHash))
	}

	l.logger.LogAttrs(ctx, slog.LevelInfo, "Privacy redaction detailed audit", attrs...)

	return nil
}

// LogPolicyViolation logs a privacy policy violation
func (l *StandardAuditLogger) LogPolicyViolation(ctx context.Context, tenantID, userID, violation string, details map[string]interface{}) error {
	attrs := []slog.Attr{
		slog.String("event_type", "policy_violation"),
		slog.String("tenant_id", tenantID),
		slog.String("user_id", userID),
		slog.String("violation", violation),
		slog.Time("timestamp", time.Now().UTC()),
	}

	for k, v := range details {
		attrs = append(attrs, slog.Any(k, v))
	}

	l.logger.LogAttrs(ctx, slog.LevelWarn, "Privacy policy violation", attrs...)

	return nil
}

// Pipeline coordinates multiple privacy operations on model requests/responses
type Pipeline struct {
	Engine      *Engine
	AuditLogger AuditLogger
	Level       PrivacyLevel
}

// NewPipeline creates a new privacy pipeline
func NewPipeline(engine *Engine, logger AuditLogger) *Pipeline {
	return &Pipeline{
		Engine:      engine,
		AuditLogger: logger,
		Level:       PrivacyLevelMedium,
	}
}

// NewPipelineWithLevel creates a new privacy pipeline with a specific level
func NewPipelineWithLevel(engine *Engine, logger AuditLogger, level PrivacyLevel) *Pipeline {
	return &Pipeline{
		Engine:      engine,
		AuditLogger: logger,
		Level:       level,
	}
}

// ProcessPrompt applies PII redaction and logs audit records for compliance
func (p *Pipeline) ProcessPrompt(ctx context.Context, tenantID, userID, prompt string, level PrivacyLevel) (string, error) {
	startTime := time.Now()

	// Determine redaction level
	redactionLevel := p.mapPrivacyToRedaction(level)

	// Detect PII first for audit
	detections := p.Engine.Detect(prompt)

	// Count by type for audit
	counts := make(map[PIIType]int)
	for _, d := range detections {
		counts[d.Type]++
	}

	// Apply redaction based on level
	redacted := p.Engine.RedactWithLevel(prompt, redactionLevel)

	// Log audit entry if logger is configured
	if p.AuditLogger != nil {
		duration := time.Since(startTime).Milliseconds()

		auditEntry := &RedactionAuditEntry{
			Timestamp:     time.Now().UTC(),
			TenantID:      tenantID,
			UserID:        userID,
			RequestID:     generateRequestID(),
			PIICounts:     counts,
			TotalRedacted: len(detections),
			Level:         level,
			DurationMs:    duration,
			Success:       true,
		}

		_ = p.AuditLogger.LogRedactionDetailed(ctx, auditEntry)
	}

	return redacted, nil
}

// ProcessPromptWithResult processes a prompt and returns detailed results
func (p *Pipeline) ProcessPromptWithResult(ctx context.Context, tenantID, userID, prompt string, level PrivacyLevel) (*RedactionResult, error) {
	startTime := time.Now()

	redactionLevel := p.mapPrivacyToRedaction(level)

	// Get detailed result
	result := p.Engine.RedactWithResult(prompt)
	result.Level = redactionLevel

	// Log audit entry
	if p.AuditLogger != nil {
		duration := time.Since(startTime).Milliseconds()

		auditEntry := &RedactionAuditEntry{
			Timestamp:     time.Now().UTC(),
			TenantID:      tenantID,
			UserID:        userID,
			RequestID:     generateRequestID(),
			PIICounts:     result.Counts,
			TotalRedacted: result.TotalCount,
			Level:         level,
			DurationMs:    duration,
			Success:       true,
		}

		_ = p.AuditLogger.LogRedactionDetailed(ctx, auditEntry)
	}

	return result, nil
}

// mapPrivacyToRedaction maps PrivacyLevel to RedactionLevel
func (p *Pipeline) mapPrivacyToRedaction(level PrivacyLevel) RedactionLevel {
	switch level {
	case PrivacyLevelLow:
		return RedactionLevelBasic
	case PrivacyLevelMedium:
		return RedactionLevelStandard
	case PrivacyLevelHigh:
		return RedactionLevelStrict
	case PrivacyLevelStrict:
		return RedactionLevelStrict
	default:
		return RedactionLevelStandard
	}
}

// ProcessResponse applies privacy filtering to model responses
func (p *Pipeline) ProcessResponse(ctx context.Context, tenantID, userID, response string) (string, error) {
	// Response filtering is typically lighter than prompt filtering
	// Only apply strict redaction if configured
	return p.Engine.Redact(response), nil
}

// SetLevel updates the privacy level for the pipeline
func (p *Pipeline) SetLevel(level PrivacyLevel) {
	p.Level = level
}

// GetLevel returns the current privacy level
func (p *Pipeline) GetLevel() PrivacyLevel {
	return p.Level
}

// generateRequestID generates a unique request ID for audit tracking
func generateRequestID() string {
	return fmt.Sprintf("redact-%d-%s", time.Now().UnixNano(), randomString(8))
}

// randomString generates a random alphanumeric string
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// AuditRecorder provides programmatic access to audit logging
type AuditRecorder struct {
	logger    *slog.Logger
	entries   []*RedactionAuditEntry
	mu        sync.Mutex
	enabled   bool
	retention time.Duration
}

// NewAuditRecorder creates a new in-memory audit recorder
func NewAuditRecorder(logger *slog.Logger, retention time.Duration) *AuditRecorder {
	if logger == nil {
		logger = slog.Default()
	}
	if retention == 0 {
		retention = 24 * time.Hour
	}

	return &AuditRecorder{
		logger:    logger,
		entries:   make([]*RedactionAuditEntry, 0),
		enabled:   true,
		retention: retention,
	}
}

// Record adds an audit entry to the recorder
func (r *AuditRecorder) Record(entry *RedactionAuditEntry) {
	if !r.enabled {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries = append(r.entries, entry)
	r.cleanup()
}

// GetEntries returns all audit entries
func (r *AuditRecorder) GetEntries() []*RedactionAuditEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]*RedactionAuditEntry, len(r.entries))
	copy(result, r.entries)
	return result
}

// GetEntriesByTenant returns audit entries for a specific tenant
func (r *AuditRecorder) GetEntriesByTenant(tenantID string) []*RedactionAuditEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result []*RedactionAuditEntry
	for _, entry := range r.entries {
		if entry.TenantID == tenantID {
			result = append(result, entry)
		}
	}
	return result
}

// GetEntriesByUser returns audit entries for a specific user
func (r *AuditRecorder) GetEntriesByUser(userID string) []*RedactionAuditEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result []*RedactionAuditEntry
	for _, entry := range r.entries {
		if entry.UserID == userID {
			result = append(result, entry)
		}
	}
	return result
}

// GetStats returns statistics about recorded audit entries
func (r *AuditRecorder) GetStats() map[string]interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	stats := map[string]interface{}{
		"total_entries":   len(r.entries),
		"enabled":         r.enabled,
		"retention_hours": r.retention.Hours(),
	}

	// Count by success/failure
	success := 0
	failure := 0
	totalRedacted := 0

	for _, entry := range r.entries {
		if entry.Success {
			success++
		} else {
			failure++
		}
		totalRedacted += entry.TotalRedacted
	}

	stats["successful_operations"] = success
	stats["failed_operations"] = failure
	stats["total_redactions"] = totalRedacted

	return stats
}

// Enable enables the audit recorder
func (r *AuditRecorder) Enable() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = true
}

// Disable disables the audit recorder
func (r *AuditRecorder) Disable() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = false
}

// Clear removes all recorded entries
func (r *AuditRecorder) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = make([]*RedactionAuditEntry, 0)
}

// cleanup removes entries older than the retention period
func (r *AuditRecorder) cleanup() {
	cutoff := time.Now().Add(-r.retention)
	var valid []*RedactionAuditEntry
	for _, entry := range r.entries {
		if entry.Timestamp.After(cutoff) {
			valid = append(valid, entry)
		}
	}
	r.entries = valid
}

// Flush writes all entries to the logger
func (r *AuditRecorder) Flush() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, entry := range r.entries {
		r.logger.Info("Audit entry",
			slog.Any("entry", entry),
		)
	}
}
