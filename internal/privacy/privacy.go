package privacy

import (
	"context"
)

// PrivacyLevel defines the privacy enforcement strength
type PrivacyLevel int

const (
	PrivacyLevelLow PrivacyLevel = iota
	PrivacyLevelMedium
	PrivacyLevelHigh
	PrivacyLevelStrict
)

// PrivacyFilter defines the contract for scanning and altering data for privacy safety
type PrivacyFilter interface {
	Process(ctx context.Context, input string, level PrivacyLevel) (string, error)
}

// AuditLogger defines how privacy violations and operations are logged for compliance
type AuditLogger interface {
	LogAccess(ctx context.Context, tenantID string, userID string, resource string, action string, allowed bool) error
	LogRedaction(ctx context.Context, tenantID string, userID string, fieldCount int) error
}

// Pipeline coordinates multiple privacy operations on model requests/responses
type Pipeline struct {
	Engine      *Engine
	AuditLogger AuditLogger
}

func NewPipeline(engine *Engine, logger AuditLogger) *Pipeline {
	return &Pipeline{
		Engine:      engine,
		AuditLogger: logger,
	}
}

// ProcessPrompt applies PII redaction and logs audit records for compliance
func (p *Pipeline) ProcessPrompt(ctx context.Context, tenantID, userID, prompt string, level PrivacyLevel) (string, error) {
	redacted := p.Engine.Redact(prompt)
	if redacted != prompt && p.AuditLogger != nil {
		_ = p.AuditLogger.LogRedaction(ctx, tenantID, userID, 1)
	}
	return redacted, nil
}
