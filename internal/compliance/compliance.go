package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ComplianceStandard defines supported compliance standards
type ComplianceStandard string

const (
	StandardSOC2       ComplianceStandard = "SOC2"
	StandardHIPAA      ComplianceStandard = "HIPAA"
	StandardGDPR       ComplianceStandard = "GDPR"
	StandardPCI        ComplianceStandard = "PCI-DSS"
	StandardISO27001   ComplianceStandard = "ISO27001"
	StandardFedRAMP    ComplianceStandard = "FedRAMP"
	StandardHIPAABusinessAssociate ComplianceStandard = "HIPAA_BAA"
)

// AuditEventType defines types of audit events
type AuditEventType string

const (
	AuditEventDataAccess     AuditEventType = "data.access"
	AuditEventDataModification AuditEventType = "data.modification"
	AuditEventDataDeletion   AuditEventType = "data.deletion"
	AuditEventDataTransfer   AuditEventType = "data.transfer"
	AuditEventDataExport     AuditEventType = "data.export"
	AuditEventAuthSuccess    AuditEventType = "auth.success"
	AuditEventAuthFailure    AuditEventType = "auth.failure"
	AuditEventConfigChange   AuditEventType = "config.change"
	AuditEventAPICall        AuditEventType = "api.call"
	AuditEventGuardrailBlock AuditEventType = "guardrail.block"
)

// AuditEntry represents a compliance audit entry
type AuditEntry struct {
	ID            string                 `json:"id"`
	OrgID         string                 `json:"org_id"`
	Timestamp     time.Time              `json:"timestamp"`
	EventType     AuditEventType         `json:"event_type"`
	Standard      ComplianceStandard     `json:"standard,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	ResourceType  string                 `json:"resource_type"`
	ResourceID    string                 `json:"resource_id"`
	Action        string                 `json:"action"`
	Outcome       string                 `json:"outcome"` // "success", "failure", "blocked"
	IPAddress     string                 `json:"ip_address,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	DataCategory  string                 `json:"data_category,omitempty"` // "pii", "phi", "financial", etc.
	DataHash      string                 `json:"data_hash,omitempty"`
	RetentionDays int                    `json:"retention_days"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ComplianceCheck represents a compliance check configuration
type ComplianceCheck struct {
	ID           string             `json:"id"`
	Standard     ComplianceStandard `json:"standard"`
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Rule         string             `json:"rule"`
	Enabled      bool               `json:"enabled"`
	Severity     string             `json:"severity"` // "critical", "high", "medium", "low"
	Frequency    string             `json:"frequency"` // "realtime", "hourly", "daily"
	LastChecked  time.Time          `json:"last_checked,omitempty"`
	NextCheck    time.Time          `json:"next_check,omitempty"`
}

// ComplianceViolation represents a detected violation
type ComplianceViolation struct {
	ID          string                 `json:"id"`
	CheckID     string                 `json:"check_id"`
	Standard    ComplianceStandard     `json:"standard"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	ResourceID  string                 `json:"resource_id"`
	DetectedAt  time.Time              `json:"detected_at"`
	Status      string                 `json:"status"` // "open", "acknowledged", "remediated", "waived"
	RemediatedAt time.Time             `json:"remediated_at,omitempty"`
	Remediation string                 `json:"remediation,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DataRetentionPolicy defines data retention rules
type DataRetentionPolicy struct {
	ID           string             `json:"id"`
	OrgID        string             `json:"org_id"`
	DataType     string             `json:"data_type"` // "audit_logs", "requests", "responses", etc.
	RetentionDays int               `json:"retention_days"`
	Standard     ComplianceStandard `json:"standard"`
	Encrypted    bool               `json:"encrypted"`
	Anonymized   bool               `json:"anonymized"`
}

// ComplianceReport represents a compliance report
type ComplianceReport struct {
	ID             string                  `json:"id"`
	OrgID          string                  `json:"org_id"`
	Standard       ComplianceStandard     `json:"standard"`
	PeriodStart    time.Time               `json:"period_start"`
	PeriodEnd      time.Time               `json:"period_end"`
	GeneratedAt    time.Time               `json:"generated_at"`
	Status         string                  `json:"status"` // "draft", "review", "approved"
	Summary        *ReportSummary          `json:"summary"`
	Violations     []*ComplianceViolation  `json:"violations"`
	Exceptions     []*ComplianceException  `json:"exceptions"`
	Controls       []*ControlAssessment    `json:"controls"`
	Findings       []*ComplianceFinding   `json:"findings"`
}

// ReportSummary provides high-level summary
type ReportSummary struct {
	TotalChecks        int     `json:"total_checks"`
	PassedChecks       int     `json:"passed_checks"`
	FailedChecks       int     `json:"failed_checks"`
	ComplianceScore    float64 `json:"compliance_score"` // 0-100
	CriticalViolations int     `json:"critical_violations"`
	HighViolations     int     `json:"high_violations"`
	MediumViolations   int     `json:"medium_violations"`
	LowViolations      int     `json:"low_violations"`
}

// ComplianceException represents an approved exception
type ComplianceException struct {
	ID          string    `json:"id"`
	ViolationID string    `json:"violation_id"`
	Reason      string    `json:"reason"`
	ApprovedBy  string    `json:"approved_by"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// ControlAssessment represents assessment of a control
type ControlAssessment struct {
	ControlID    string `json:"control_id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	Status       string `json:"status"` // "compliant", "non_compliant", "not_applicable"
	Evidence     string `json:"evidence,omitempty"`
	LastTested   time.Time `json:"last_tested,omitempty"`
	TestResult   string `json:"test_result,omitempty"`
}

// ComplianceFinding represents a detailed finding
type ComplianceFinding struct {
	ID           string    `json:"id"`
	CheckID     string    `json:"check_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	Impact      string    `json:"impact"`
	Recommendation string `json:"recommendation"`
	Evidence    string    `json:"evidence,omitempty"`
	DetectedAt  time.Time `json:"detected_at"`
}

// AuditLogger handles compliance audit logging
type AuditLogger struct {
	mu         sync.RWMutex
	entries    map[string]*AuditEntry
	retention  map[string]*DataRetentionPolicy
	maxEntries int
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(maxEntries int) *AuditLogger {
	return &AuditLogger{
		entries:   make(map[string]*AuditEntry),
		retention: make(map[string]*DataRetentionPolicy),
		maxEntries: maxEntries,
	}
}

// Log creates a new audit entry
func (al *AuditLogger) Log(ctx context.Context, entry *AuditEntry) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Set retention based on policy
	if policy, ok := al.retention[entry.DataCategory]; ok {
		entry.RetentionDays = policy.RetentionDays
	} else {
		entry.RetentionDays = 365 // Default 1 year
	}

	al.entries[entry.ID] = entry

	slog.DebugContext(ctx, "Audit entry logged",
		slog.String("id", entry.ID),
		slog.String("event_type", string(entry.EventType)),
		slog.String("org_id", entry.OrgID),
	)

	return nil
}

// Query queries audit entries
func (al *AuditLogger) Query(ctx context.Context, req *AuditQuery) ([]*AuditEntry, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	var results []*AuditEntry

	for _, entry := range al.entries {
		// Apply filters
		if req.OrgID != "" && entry.OrgID != req.OrgID {
			continue
		}
		if req.EventType != "" && entry.EventType != req.EventType {
			continue
		}
		if req.UserID != "" && entry.UserID != req.UserID {
			continue
		}
		if req.ResourceType != "" && entry.ResourceType != req.ResourceType {
			continue
		}
		if !req.StartTime.IsZero() && entry.Timestamp.Before(req.StartTime) {
			continue
		}
		if !req.EndTime.IsZero() && entry.Timestamp.After(req.EndTime) {
			continue
		}
		if req.DataCategory != "" && entry.DataCategory != req.DataCategory {
			continue
		}

		results = append(results, entry)

		// Apply limit
		if req.Limit > 0 && len(results) >= req.Limit {
			break
		}
	}

	return results, nil
}

// AuditQuery represents a query for audit entries
type AuditQuery struct {
	OrgID        string             `json:"org_id"`
	EventType    AuditEventType     `json:"event_type,omitempty"`
	UserID       string             `json:"user_id,omitempty"`
	ResourceType string             `json:"resource_type,omitempty"`
	ResourceID   string             `json:"resource_id,omitempty"`
	StartTime    time.Time          `json:"start_time,omitempty"`
	EndTime      time.Time          `json:"end_time,omitempty"`
	DataCategory string             `json:"data_category,omitempty"`
	Limit        int                `json:"limit,omitempty"`
}

// ComplianceManager handles compliance checks and reporting
type ComplianceManager struct {
	mu      sync.RWMutex
	checks  map[string]*ComplianceCheck
	violations map[string]*ComplianceViolation
	auditLog *AuditLogger
}

// NewComplianceManager creates a new compliance manager
func NewComplianceManager(auditLog *AuditLogger) *ComplianceManager {
	cm := &ComplianceManager{
		checks:     make(map[string]*ComplianceCheck),
		violations: make(map[string]*ComplianceViolation),
		auditLog:  auditLog,
	}

	// Initialize default checks for common standards
	cm.initializeDefaultChecks()

	return cm
}

// initializeDefaultChecks sets up default compliance checks
func (cm *ComplianceManager) initializeDefaultChecks() {
	checks := []*ComplianceCheck{
		// SOC2 controls
		{ID: "SOC2-CC1.1", Standard: StandardSOC2, Name: "Access Control", Description: "User access is restricted to authorized users", Severity: "high", Frequency: "realtime"},
		{ID: "SOC2-CC2.1", Standard: StandardSOC2, Name: "Logical Access", Description: "Logical access controls are in place", Severity: "high", Frequency: "realtime"},
		{ID: "SOC2-CC3.1", Standard: StandardSOC2, Name: "Data Encryption", Description: "Data is encrypted at rest and in transit", Severity: "critical", Frequency: "daily"},
		{ID: "SOC2-CC6.1", Standard: StandardSOC2, Name: "Audit Logging", Description: "Audit logs are generated and retained", Severity: "high", Frequency: "realtime"},

		// HIPAA controls
		{ID: "HIPAA-164.308", Standard: StandardHIPAA, Name: "Access Management", Description: "Access to PHI is restricted", Severity: "critical", Frequency: "realtime"},
		{ID: "HIPAA-164.312", Standard: StandardHIPAA, Name: "Encryption", Description: "PHI is encrypted", Severity: "critical", Frequency: "daily"},
		{ID: "HIPAA-164.316", Standard: StandardHIPAA, Name: "Audit Controls", Description: "Audit controls are implemented", Severity: "high", Frequency: "realtime"},

		// GDPR controls
		{ID: "GDPR-Art5", Standard: StandardGDPR, Name: "Data Minimization", Description: "Only necessary data is collected", Severity: "medium", Frequency: "daily"},
		{ID: "GDPR-Art6", Standard: StandardGDPR, Name: "Lawful Basis", Description: "Processing has lawful basis", Severity: "high", Frequency: "realtime"},
		{ID: "GDPR-Art32", Standard: StandardGDPR, Name: "Security Measures", Description: "Appropriate security measures in place", Severity: "high", Frequency: "daily"},
		{ID: "GDPR-Art33", Standard: StandardGDPR, Name: "Breach Notification", Description: "Breaches reported within 72 hours", Severity: "critical", Frequency: "realtime"},

		// PCI-DSS controls
		{ID: "PCI-3.4", Standard: StandardPCI, Name: "Data Encryption", Description: "Cardholder data is encrypted", Severity: "critical", Frequency: "daily"},
		{ID: "PCI-8.3", Standard: StandardPCI, Name: "Authentication", Description: "Strong authentication is enforced", Severity: "high", Frequency: "realtime"},
		{ID: "PCI-10.1", Standard: StandardPCI, Name: "Audit Logging", Description: "Audit trails are in place", Severity: "high", Frequency: "realtime"},
	}

	for _, check := range checks {
		check.Enabled = true
		cm.checks[check.ID] = check
	}
}

// CreateViolation creates a new violation record
func (cm *ComplianceManager) CreateViolation(ctx context.Context, violation *ComplianceViolation) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if violation.ID == "" {
		violation.ID = uuid.New().String()
	}
	if violation.DetectedAt.IsZero() {
		violation.DetectedAt = time.Now()
	}
	violation.Status = "open"

	cm.violations[violation.ID] = violation

	slog.WarnContext(ctx, "Compliance violation detected",
		slog.String("id", violation.ID),
		slog.String("check_id", violation.CheckID),
		slog.String("standard", string(violation.Standard)),
		slog.String("severity", violation.Severity),
	)

	return nil
}

// GetViolation returns a violation by ID
func (cm *ComplianceManager) GetViolation(id string) (*ComplianceViolation, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	v, exists := cm.violations[id]
	return v, exists
}

// GetViolations returns violations matching criteria
func (cm *ComplianceManager) GetViolations(ctx context.Context, req *ViolationQuery) []*ComplianceViolation {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var results []*ComplianceViolation
	for _, v := range cm.violations {
		if req.OrgID != "" && v.Standard == StandardSOC2 { // Filter by org
		}
		if req.Standard != "" && v.Standard != req.Standard {
			continue
		}
		if req.Severity != "" && v.Severity != req.Severity {
			continue
		}
		if req.Status != "" && v.Status != req.Status {
			continue
		}
		results = append(results, v)
	}
	return results
}

// ViolationQuery represents a query for violations
type ViolationQuery struct {
	OrgID    string             `json:"org_id,omitempty"`
	Standard ComplianceStandard `json:"standard,omitempty"`
	Severity string             `json:"severity,omitempty"`
	Status   string             `json:"status,omitempty"`
}

// AcknowledgeViolation acknowledges a violation
func (cm *ComplianceManager) AcknowledgeViolation(ctx context.Context, id, acknowledgedBy string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	v, exists := cm.violations[id]
	if !exists {
		return fmt.Errorf("violation not found: %s", id)
	}

	v.Status = "acknowledged"

	slog.InfoContext(ctx, "Violation acknowledged",
		slog.String("id", id),
		slog.String("acknowledged_by", acknowledgedBy),
	)

	return nil
}

// RemediateViolation marks a violation as remediated
func (cm *ComplianceManager) RemediateViolation(ctx context.Context, id, remediation string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	v, exists := cm.violations[id]
	if !exists {
		return fmt.Errorf("violation not found: %s", id)
	}

	v.Status = "remediated"
	v.Remediation = remediation
	v.RemediatedAt = time.Now()

	slog.InfoContext(ctx, "Violation remediated",
		slog.String("id", id),
	)

	return nil
}

// GenerateReport generates a compliance report
func (cm *ComplianceManager) GenerateReport(ctx context.Context, req *ReportRequest) (*ComplianceReport, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	report := &ComplianceReport{
		ID:           uuid.New().String(),
		OrgID:        req.OrgID,
		Standard:     req.Standard,
		PeriodStart:  req.PeriodStart,
		PeriodEnd:    req.PeriodEnd,
		GeneratedAt:  time.Now(),
		Status:       "draft",
		Summary:      &ReportSummary{},
	}

	// Count violations by severity
	var criticalCount, highCount, mediumCount, lowCount int
	for _, v := range cm.violations {
		if v.Standard != req.Standard {
			continue
		}
		if v.DetectedAt.Before(req.PeriodStart) || v.DetectedAt.After(req.PeriodEnd) {
			continue
		}

		switch v.Severity {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		case "medium":
			mediumCount++
		case "low":
			lowCount++
		}
	}

	report.Summary.CriticalViolations = criticalCount
	report.Summary.HighViolations = highCount
	report.Summary.MediumViolations = mediumCount
	report.Summary.LowViolations = lowCount
	report.Summary.TotalChecks = len(cm.checks)
	report.Summary.FailedChecks = criticalCount + highCount + mediumCount + lowCount
	report.Summary.PassedChecks = report.Summary.TotalChecks - report.Summary.FailedChecks

	// Calculate compliance score
	if report.Summary.TotalChecks > 0 {
		report.Summary.ComplianceScore = float64(report.Summary.PassedChecks) / float64(report.Summary.TotalChecks) * 100
	}

	return report, nil
}

// ReportRequest represents a request for a compliance report
type ReportRequest struct {
	OrgID       string             `json:"org_id"`
	Standard    ComplianceStandard `json:"standard"`
	PeriodStart time.Time         `json:"period_start"`
	PeriodEnd   time.Time         `json:"period_end"`
}

// SetRetentionPolicy sets a data retention policy
func (al *AuditLogger) SetRetentionPolicy(policy *DataRetentionPolicy) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.retention[policy.DataType] = policy
}

// Export exports audit logs for external systems
func (al *AuditLogger) Export(ctx context.Context, req *ExportRequest) ([]*AuditEntry, error) {
	query := &AuditQuery{
		OrgID:        req.OrgID,
		EventType:    req.EventType,
		UserID:       req.UserID,
		ResourceType: req.ResourceType,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		DataCategory: req.DataCategory,
		Limit:        req.Limit,
	}
	entries, err := al.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	// Apply anonymization if required
	if req.Anonymize {
		for _, entry := range entries {
			entry.UserID = fmt.Sprintf("user_%s", uuid.New().String()[:8])
			entry.IPAddress = ""
			entry.UserAgent = ""
		}
	}

	return entries, nil
}

// ExportRequest represents an export request
type ExportRequest struct {
	OrgID        string             `json:"org_id"`
	EventType    AuditEventType     `json:"event_type,omitempty"`
	UserID       string             `json:"user_id,omitempty"`
	ResourceType string             `json:"resource_type,omitempty"`
	StartTime    time.Time          `json:"start_time,omitempty"`
	EndTime      time.Time          `json:"end_time,omitempty"`
	DataCategory string             `json:"data_category,omitempty"`
	Limit        int                `json:"limit,omitempty"`
	Format       string `json:"format"` // "json", "csv", "parquet"
	Anonymize    bool   `json:"anonymize"`
	Compress     bool   `json:"compress"`
}

// Stats returns compliance statistics
type ComplianceStats struct {
	TotalChecks    int `json:"total_checks"`
	EnabledChecks  int `json:"enabled_checks"`
	OpenViolations int `json:"open_violations"`
}

// GetStats returns compliance statistics
func (cm *ComplianceManager) GetStats() ComplianceStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := ComplianceStats{
		TotalChecks: len(cm.checks),
	}

	for _, check := range cm.checks {
		if check.Enabled {
			stats.EnabledChecks++
		}
	}

	for _, v := range cm.violations {
		if v.Status == "open" {
			stats.OpenViolations++
		}
	}

	return stats
}

// Ensure types are used
var _ = json.Marshal
