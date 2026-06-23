package guardrails

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// GuardrailModeHandler handles different guardrail modes
type GuardrailModeHandler struct {
	mode         GuardrailMode
	logger       *slog.Logger
	onBlock      func(ctx context.Context, result *GuardrailResult) error
	onWarn       func(ctx context.Context, result *GuardrailResult) error
	onLog        func(ctx context.Context, result *GuardrailResult) error
}

// NewModeHandler creates a new mode handler
func NewModeHandler(mode GuardrailMode, logger *slog.Logger) *GuardrailModeHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &GuardrailModeHandler{
		mode:   mode,
		logger: logger,
	}
}

// Handle processes the guardrail result based on mode
func (h *GuardrailModeHandler) Handle(ctx context.Context, result *GuardrailResult) error {
	switch result.Action {
	case ActionBlock:
		return h.handleBlock(ctx, result)
	case ActionWarn:
		return h.handleWarn(ctx, result)
	case ActionLog:
		return h.handleLog(ctx, result)
	case ActionAllow:
		return nil
	default:
		h.logger.Warn("Unknown guardrail action", "action", result.Action)
		return nil
	}
}

// handleBlock handles blocking actions
func (h *GuardrailModeHandler) handleBlock(ctx context.Context, result *GuardrailResult) error {
	h.logger.Warn("Guardrail blocked request",
		slog.String("message", result.Message),
		slog.Int("detections", len(result.Detections)),
	)

	if h.onBlock != nil {
		return h.onBlock(ctx, result)
	}
	return nil
}

// handleWarn handles warning actions
func (h *GuardrailModeHandler) handleWarn(ctx context.Context, result *GuardrailResult) error {
	h.logger.Warn("Guardrail warning",
		slog.String("message", result.Message),
		slog.Int("detections", len(result.Detections)),
	)

	if h.onWarn != nil {
		return h.onWarn(ctx, result)
	}
	return nil
}

// handleLog handles logging-only actions
func (h *GuardrailModeHandler) handleLog(ctx context.Context, result *GuardrailResult) error {
	h.logger.Info("Guardrail log",
		slog.String("message", result.Message),
		slog.Int("detections", len(result.Detections)),
	)

	if h.onLog != nil {
		return h.onLog(ctx, result)
	}
	return nil
}

// SetMode updates the handler mode
func (h *GuardrailModeHandler) SetMode(mode GuardrailMode) {
	h.mode = mode
}

// SetOnBlock sets the block callback
func (h *GuardrailModeHandler) SetOnBlock(fn func(ctx context.Context, result *GuardrailResult) error) {
	h.onBlock = fn
}

// SetOnWarn sets the warn callback
func (h *GuardrailModeHandler) SetOnWarn(fn func(ctx context.Context, result *GuardrailResult) error) {
	h.onWarn = fn
}

// SetOnLog sets the log callback
func (h *GuardrailModeHandler) SetOnLog(fn func(ctx context.Context, result *GuardrailResult) error) {
	h.onLog = fn
}

// ModeConfig holds configuration for each mode
type ModeConfig struct {
	BlockOnSeverity Severity   `json:"block_on_severity"`
	WarnOnSeverity  Severity   `json:"warn_on_severity"`
	LogAll          bool       `json:"log_all"`
	AuditAll        bool       `json:"audit_all"`
	Timeout         time.Duration `json:"timeout"`
}

// DefaultModeConfig returns default mode configuration
func DefaultModeConfig() *ModeConfig {
	return &ModeConfig{
		BlockOnSeverity: SeverityHigh,
		WarnOnSeverity:  SeverityMedium,
		LogAll:          true,
		AuditAll:        true,
		Timeout:         30 * time.Second,
	}
}

// GuardrailManager manages all guardrails and their execution
type GuardrailManager struct {
	guardrails []Guardrail
	mode       GuardrailMode
	config     *GuardrailConfig
	stats      *GuardrailStats
	logger     *slog.Logger
	mu         sync.RWMutex
}

// NewManager creates a new guardrail manager
func NewManager(mode GuardrailMode) *GuardrailManager {
	return &GuardrailManager{
		guardrails: make([]Guardrail, 0),
		mode:       mode,
		config:     DefaultGuardrailConfig(),
		stats:     &GuardrailStats{},
		logger:    slog.Default(),
	}
}

// NewManagerWithConfig creates a manager with custom config
func NewManagerWithConfig(mode GuardrailMode, config *GuardrailConfig) *GuardrailManager {
	m := NewManager(mode)
	if config != nil {
		m.config = config
		m.mode = config.Mode
	}
	return m
}

// AddGuardrail adds a guardrail to the manager
func (m *GuardrailManager) AddGuardrail(g Guardrail) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.guardrails = append(m.guardrails, g)
}

// RemoveGuardrail removes a guardrail by name
func (m *GuardrailManager) RemoveGuardrail(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var filtered []Guardrail
	for _, g := range m.guardrails {
		if g.Name() != name {
			filtered = append(filtered, g)
		}
	}
	m.guardrails = filtered
}

// RunAll runs all applicable guardrails
func (m *GuardrailManager) RunAll(ctx context.Context, gc *GuardrailContext) ([]*GuardrailResult, error) {
	m.mu.RLock()
	guardrails := make([]Guardrail, len(m.guardrails))
	copy(guardrails, m.guardrails)
	m.mu.RUnlock()

	var results []*GuardrailResult
	var lastError error

	// Sort by priority
	sortGuardrails(guardrails)

	for _, g := range guardrails {
		result, err := g.Check(ctx, gc)
		if err != nil {
			m.logger.Error("Guardrail check failed",
				slog.String("guardrail", g.Name()),
				slog.String("error", err.Error()),
			)
			lastError = err
			continue
		}

		results = append(results, result)

		// Update stats
		m.updateStats(result)

		// Handle result based on mode
		handler := NewModeHandler(m.mode, m.logger)
		if err := handler.Handle(ctx, result); err != nil {
			lastError = err
		}

		// Block immediately if critical and mode is block
		if m.mode == ModeBlock && result.Action == ActionBlock {
			break
		}
	}

	return results, lastError
}

// RunStage runs guardrails for a specific stage
func (m *GuardrailManager) RunStage(ctx context.Context, gc *GuardrailContext, stage GuardrailStage) ([]*GuardrailResult, error) {
	m.mu.RLock()
	guardrails := make([]Guardrail, len(m.guardrails))
	copy(guardrails, m.guardrails)
	m.mu.RUnlock()

	var results []*GuardrailResult

	// Filter by stage and sort by priority
	var stageGuardrails []Guardrail
	for _, g := range guardrails {
		if g.Stage() == stage {
			stageGuardrails = append(stageGuardrails, g)
		}
	}
	sortGuardrails(stageGuardrails)

	handler := NewModeHandler(m.mode, m.logger)

	for _, g := range stageGuardrails {
		result, err := g.Check(ctx, gc)
		if err != nil {
			m.logger.Error("Guardrail check failed",
				slog.String("guardrail", g.Name()),
				slog.String("error", err.Error()),
			)
			continue
		}

		results = append(results, result)
		m.updateStats(result)

		if err := handler.Handle(ctx, result); err != nil {
			m.logger.Error("Handler error",
				slog.String("guardrail", g.Name()),
				slog.String("error", err.Error()),
			)
		}

		if m.mode == ModeBlock && result.Action == ActionBlock {
			break
		}
	}

	return results, nil
}

// updateStats updates guardrail statistics
func (m *GuardrailManager) updateStats(result *GuardrailResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.TotalChecks++
	if result.Passed {
		m.stats.Passed++
	}
	if result.Action == ActionBlock {
		m.stats.Blocked++
	} else if result.Action == ActionWarn {
		m.stats.Warnings++
	}
	m.stats.TotalDetections += int64(len(result.Detections))

	for _, d := range result.Detections {
		if d.Type == "pii" || isPIIType(d.Type) {
			m.stats.PIIDetections++
		} else {
			m.stats.InjectionDetections++
		}
	}
}

// GetStats returns current statistics
func (m *GuardrailManager) GetStats() GuardrailStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.stats
}

// SetMode updates the manager mode
func (m *GuardrailManager) SetMode(mode GuardrailMode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mode = mode
	for _, g := range m.guardrails {
		switch gr := g.(type) {
		case *PIIGuardrail:
			gr.SetMode(mode)
		case *InjectionGuardrail:
			gr.SetMode(mode)
		case *VisionGuardrail:
			gr.SetMode(mode)
		}
	}
}

// ResetStats resets all statistics
func (m *GuardrailManager) ResetStats() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats = &GuardrailStats{}
}

// sortGuardrails sorts guardrails by priority (lower = earlier)
func sortGuardrails(guardrails []Guardrail) {
	for i := 0; i < len(guardrails)-1; i++ {
		for j := i + 1; j < len(guardrails); j++ {
			if guardrails[i].Priority() > guardrails[j].Priority() {
				guardrails[i], guardrails[j] = guardrails[j], guardrails[i]
			}
		}
	}
}

// isPIIType checks if a detection type is PII-related
func isPIIType(t string) bool {
	piiTypes := map[string]bool{
		"email": true, "phone": true, "ssn": true, "credit_card": true,
		"address": true, "name": true, "ip_address": true, "password": true,
		"api_key": true, "iban": true, "vin": true, "date_of_birth": true,
		"passport": true, "driver_license": true, "medical_record": true,
		"health_plan": true, "tax_id": true, "aws_key": true,
		"private_key": true, "database_conn": true,
	}
	return piiTypes[t]
}

// DefaultGuardrailManager creates a manager with all default guardrails
func DefaultGuardrailManager(mode GuardrailMode) *GuardrailManager {
	m := NewManager(mode)
	m.AddGuardrail(NewInjectionGuardrail(mode))
	m.AddGuardrail(NewPIIGuardrail(mode))
	m.AddGuardrail(NewVisionGuardrail(mode))
	return m
}
