package guardrails

import (
	"context"
	"regexp"
	"strings"
)

// PIIType represents the type of PII being detected
type PIIType string

const (
	PIIEmail      PIIType = "email"
	PIIPhone      PIIType = "phone"
	PIISSN        PIIType = "ssn"
	PIICreditCard PIIType = "credit_card"
	PIIAddress    PIIType = "address"
	PIIName       PIIType = "name"
	PIIIPAddress  PIIType = "ip_address"
	PIIPassword   PIIType = "password"
	PIIAPIKey     PIIType = "api_key"
	PIIIBAN       PIIType = "iban"
	PIIVIN        PIIType = "vin"
	PIIDOB        PIIType = "date_of_birth"
	PIIPassport   PIIType = "passport"
	PIIDriverLicense PIIType = "driver_license"
	PIIMedicalRecord PIIType = "medical_record"
	PIIHealthPlan  PIIType = "health_plan"
	PIITaxID       PIIType = "tax_id"
	PIIAWSKey      PIIType = "aws_key"
	PIIPrivateKey  PIIType = "private_key"
	PIIDatabase    PIIType = "database_conn"
)

// PIIPattern defines a pattern for detecting specific PII types
type PIIPattern struct {
	Type      PIIType
	Regex     *regexp.Regexp
	Marker    string
	Enabled   bool
	Context   []string // Contextual keywords that indicate this PII
}

// PIIGuardrail implements PII detection and masking
type PIIGuardrail struct {
	mode     GuardrailMode
	patterns []*PIIPattern
	strict   bool
}

// DefaultMarkers for redacted PII
var DefaultMarkers = map[PIIType]string{
	PIIEmail:          "[REDACTED_EMAIL]",
	PIIPhone:          "[REDACTED_PHONE]",
	PIISSN:            "[REDACTED_SSN]",
	PIICreditCard:     "[REDACTED_CARD]",
	PIIAddress:        "[REDACTED_ADDRESS]",
	PIIName:           "[REDACTED_NAME]",
	PIIIPAddress:      "[REDACTED_IP]",
	PIIPassword:       "[REDACTED_PASSWORD]",
	PIIAPIKey:         "[REDACTED_API_KEY]",
	PIIIBAN:           "[REDACTED_IBAN]",
	PIIVIN:            "[REDACTED_VIN]",
	PIIDOB:            "[REDACTED_DOB]",
	PIIPassport:       "[REDACTED_PASSPORT]",
	PIIDriverLicense:  "[REDACTED_LICENSE]",
	PIIMedicalRecord:  "[REDACTED_MRN]",
	PIIHealthPlan:     "[REDACTED_HEALTH_ID]",
	PIITaxID:          "[REDACTED_TAX_ID]",
	PIIAWSKey:         "[REDACTED_AWS_KEY]",
	PIIPrivateKey:     "[REDACTED_PRIVATE_KEY]",
	PIIDatabase:       "[REDACTED_DB_CONN]",
}

// DefaultPIIPatterns includes 20+ PII patterns
var DefaultPIIPatterns = []*PIIPattern{
	{
		Type:    PIIEmail,
		Regex:   regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
		Marker:  DefaultMarkers[PIIEmail],
		Enabled: true,
	},
	{
		Type:    PIIPhone,
		Regex:   regexp.MustCompile(`(?:\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}`),
		Marker:  DefaultMarkers[PIIPhone],
		Enabled: true,
	},
	{
		Type:    PIISSN,
		Regex:   regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		Marker:  DefaultMarkers[PIISSN],
		Enabled: true,
	},
	{
		Type:    PIICreditCard,
		Regex:   regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`),
		Marker:  DefaultMarkers[PIICreditCard],
		Enabled: true,
	},
	{
		Type:    PIIIPAddress,
		Regex:   regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`),
		Marker:  DefaultMarkers[PIIIPAddress],
		Enabled: true,
	},
	{
		Type:    PIIAddress,
		Regex:   regexp.MustCompile(`\d+\s+[\w\s]+\s+(?:Street|St|Avenue|Ave|Road|Rd|Boulevard|Blvd|Drive|Dr|Lane|Ln|Court|Ct|Way|Place|Pl)[\s,]*[\w\s]*,?\s*[A-Z]{2}\s*\d{5}(-\d{4})?`),
		Marker:  DefaultMarkers[PIIAddress],
		Enabled: true,
	},
	{
		Type:    PIIPassword,
		Regex:   regexp.MustCompile(`(?i)(?:password|pwd|pass)\s*[:=]\s*\S+`),
		Marker:  DefaultMarkers[PIIPassword],
		Context: []string{"password", "pwd", "pass"},
		Enabled: true,
	},
	{
		Type:    PIIAPIKey,
		Regex:   regexp.MustCompile(`(?i)(?:api[_-]?key|apikey|secret[_-]?key)\s*[:=]\s*['"]?[\w\-]+['"]?`),
		Marker:  DefaultMarkers[PIIAPIKey],
		Context: []string{"api_key", "apikey", "secret_key"},
		Enabled: true,
	},
	{
		Type:    PIIIBAN,
		Regex:   regexp.MustCompile(`\b[A-Z]{2}\d{2}[A-Z0-9]{11,30}\b`),
		Marker:  DefaultMarkers[PIIIBAN],
		Enabled: true,
	},
	{
		Type:    PIIVIN,
		Regex:   regexp.MustCompile(`\b[A-HJ-NPR-Z0-9]{17}\b`),
		Marker:  DefaultMarkers[PIIVIN],
		Enabled: true,
	},
	{
		Type:    PIIDOB,
		Regex:   regexp.MustCompile(`\b(?:0?[1-9]|1[0-2])[/\-](0?[1-9]|[12]\d|3[01])[/\-](?:19|20)\d{2}\b`),
		Marker:  DefaultMarkers[PIIDOB],
		Enabled: true,
	},
	{
		Type:    PIIPassport,
		Regex:   regexp.MustCompile(`\b[A-Z]{1,2}\d{6,9}\b`),
		Marker:  DefaultMarkers[PIIPassport],
		Enabled: true,
	},
	{
		Type:    PIIDriverLicense,
		Regex:   regexp.MustCompile(`\b[A-Z]\d{7,14}\b`),
		Marker:  DefaultMarkers[PIIDriverLicense],
		Enabled: true,
	},
	{
		Type:    PIIMedicalRecord,
		Regex:   regexp.MustCompile(`(?i)(?:mrn|medical[_-]?record[_-]?number)\s*[:=]?\s*\d+`),
		Marker:  DefaultMarkers[PIIMedicalRecord],
		Context: []string{"mrn", "medical record"},
		Enabled: true,
	},
	{
		Type:    PIIHealthPlan,
		Regex:   regexp.MustCompile(`(?i)(?:health[_-]?plan|member[_-]?id|insurance[_-]?id)\s*[:=]?\s*[\w\-]+`),
		Marker:  DefaultMarkers[PIIHealthPlan],
		Context: []string{"health plan", "member id", "insurance"},
		Enabled: true,
	},
	{
		Type:    PIITaxID,
		Regex:   regexp.MustCompile(`\b\d{2}-\d{7}\b`),
		Marker:  DefaultMarkers[PIITaxID],
		Enabled: true,
	},
	{
		Type:    PIIAWSKey,
		Regex:   regexp.MustCompile(`(?i)(?:AKIA|ABIA|ACCA|ASIA)[0-9A-Z]{16}`),
		Marker:  DefaultMarkers[PIIAWSKey],
		Enabled: true,
	},
	{
		Type:    PIIPrivateKey,
		Regex:   regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`),
		Marker:  DefaultMarkers[PIIPrivateKey],
		Enabled: true,
	},
	{
		Type:    PIIDatabase,
		Regex:   regexp.MustCompile(`(?i)(?:mongodb|postgres|mysql|redis):\/\/[^\s]+`),
		Marker:  DefaultMarkers[PIIDatabase],
		Enabled: true,
	},
	{
		Type:    PIIIPAddress,
		Regex:   regexp.MustCompile(`\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`),
		Marker:  DefaultMarkers[PIIIPAddress],
		Enabled: true,
	},
}

// NewPIIGuardrail creates a new PII guardrail with default patterns
func NewPIIGuardrail(mode GuardrailMode) *PIIGuardrail {
	patterns := make([]*PIIPattern, len(DefaultPIIPatterns))
	copy(patterns, DefaultPIIPatterns)
	return &PIIGuardrail{
		mode:     mode,
		patterns: patterns,
		strict:   false,
	}
}

// NewPIIGuardrailWithPatterns creates a PII guardrail with custom patterns
func NewPIIGuardrailWithPatterns(mode GuardrailMode, patterns []*PIIPattern) *PIIGuardrail {
	if patterns == nil {
		patterns = DefaultPIIPatterns
	}
	return &PIIGuardrail{
		mode:     mode,
		patterns: patterns,
		strict:   false,
	}
}

// Name returns the guardrail name
func (g *PIIGuardrail) Name() string {
	return "pii-guardrail"
}

// Priority returns the priority (lower = earlier)
func (g *PIIGuardrail) Priority() int {
	return 10 // Run early for PII
}

// Stage returns the stage this guardrail runs at
func (g *PIIGuardrail) Stage() GuardrailStage {
	return StagePreCall
}

// Check performs PII detection and redaction
func (g *PIIGuardrail) Check(ctx context.Context, gc *GuardrailContext) (*GuardrailResult, error) {
	var prompt string
	
	// Extract prompt from request
	if gc.Request != nil {
		if gc.Request.Prompt != "" {
			prompt = gc.Request.Prompt
		} else if len(gc.Request.Messages) > 0 {
			// Concatenate all message contents
			var parts []string
			for _, msg := range gc.Request.Messages {
				parts = append(parts, msg.Content)
			}
			prompt = strings.Join(parts, "\n")
		}
	}

	if prompt == "" {
		return &GuardrailResult{
			Passed:  true,
			Action:  ActionAllow,
			Message: "No content to scan for PII",
		}, nil
	}

	// Detect PII
	detections := g.detect(prompt)
	
	if len(detections) == 0 {
		return &GuardrailResult{
			Passed:  true,
			Action:  ActionAllow,
			Message: "No PII detected",
		}, nil
	}

	// Redact PII
	redacted, count := g.redact(prompt, detections)

	result := &GuardrailResult{
		Passed:   true,
		Action:   g.modeToAction(),
		Message:  "PII detected and redacted",
		Detections: detections,
		Metadata: map[string]interface{}{
			"total_detections": len(detections),
			"redaction_count":  count,
		},
	}

	// Add redaction info if we redacted something
	if count > 0 {
		result.Redacted = &RedactedContent{
			Original: prompt,
			Result:   redacted,
			Type:     "pii",
			Count:    count,
		}
		// Update the request with redacted content
		if gc.Request != nil {
			if gc.Request.Prompt != "" {
				gc.Request.Prompt = redacted
			}
			// Update messages
			for i := range gc.Request.Messages {
				gc.Request.Messages[i].Content = redacted
				break // Only update first message for now
			}
		}
	}

	return result, nil
}

// detect scans text for PII and returns all detections
func (g *PIIGuardrail) detect(text string) []*Detection {
	var detections []*Detection

	for _, pattern := range g.patterns {
		if !pattern.Enabled {
			continue
		}

		matches := pattern.Regex.FindAllStringIndex(text, -1)
		for _, match := range matches {
			value := text[match[0]:match[1]]

			// For credit cards, validate with Luhn
			if pattern.Type == PIICreditCard && !validateLuhn(value) {
				continue
			}

			// For SSN, validate format
			if pattern.Type == PIISSN && !validateSSN(value) {
				continue
			}

			detections = append(detections, &Detection{
				Type:      string(pattern.Type),
				Value:     value,
				Start:     match[0],
				End:       match[1],
				Pattern:   pattern.Regex.String(),
				Confidence: 0.9,
			})
		}
	}

	return detections
}

// redact replaces detected PII with markers
func (g *PIIGuardrail) redact(text string, detections []*Detection) (string, int) {
	result := text
	count := 0

	// Process in reverse order to maintain indices
	for i := len(detections) - 1; i >= 0; i-- {
		d := detections[i]
		
		// Find the marker for this PII type
		marker := DefaultMarkers[PIIType(d.Type)]
		if marker == "" {
			marker = "[REDACTED]"
		}

		// Replace the matched text
		if d.Start < len(result) && d.End <= len(result) && d.Start < d.End {
			result = result[:d.Start] + marker + result[d.End:]
			count++
		}
	}

	return result, count
}

// modeToAction converts the guardrail mode to an action
func (g *PIIGuardrail) modeToAction() GuardrailAction {
	switch g.mode {
	case ModeBlock:
		return ActionBlock
	case ModeWarn:
		return ActionWarn
	case ModeLog:
		return ActionLog
	default:
		return ActionWarn
	}
}

// EnablePattern enables a specific PII pattern
func (g *PIIGuardrail) EnablePattern(piiType PIIType) {
	for _, p := range g.patterns {
		if p.Type == piiType {
			p.Enabled = true
			return
		}
	}
}

// DisablePattern disables a specific PII pattern
func (g *PIIGuardrail) DisablePattern(piiType PIIType) {
	for _, p := range g.patterns {
		if p.Type == piiType {
			p.Enabled = false
			return
		}
	}
}

// SetMode updates the guardrail mode
func (g *PIIGuardrail) SetMode(mode GuardrailMode) {
	g.mode = mode
}

// validateLuhn performs Luhn algorithm validation for credit cards
func validateLuhn(card string) bool {
	var digits []int
	for _, c := range card {
		if c >= '0' && c <= '9' {
			digits = append(digits, int(c-'0'))
		}
	}

	if len(digits) < 13 || len(digits) > 19 {
		return false
	}

	sum := 0
	alternate := false

	for i := len(digits) - 1; i >= 0; i-- {
		n := digits[i]
		if alternate {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alternate = !alternate
	}

	return sum%10 == 0
}

// validateSSN performs basic SSN validation
func validateSSN(ssn string) bool {
	digits := strings.ReplaceAll(ssn, "-", "")
	if len(digits) != 9 {
		return false
	}

	area := digits[:3]
	if area == "000" || area == "666" || digits[0] == '9' {
		return false
	}

	if digits[3:5] == "00" {
		return false
	}

	if digits[5:9] == "0000" {
		return false
	}

	return true
}

// GetEnabledTypes returns the list of enabled PII types
func (g *PIIGuardrail) GetEnabledTypes() []PIIType {
	var enabled []PIIType
	for _, p := range g.patterns {
		if p.Enabled {
			enabled = append(enabled, p.Type)
		}
	}
	return enabled
}
