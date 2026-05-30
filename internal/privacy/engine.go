package privacy

import (
	"regexp"
	"strings"
)

// PIIType represents types of personally identifiable information
type PIIType string

const (
	PIIEmail       PIIType = "email"
	PIIPhone       PIIType = "phone"
	PIICreditCard  PIIType = "credit_card"
	PIISSN         PIIType = "ssn"
	PIIIPAddress   PIIType = "ip_address"
	PIIPassport    PIIType = "passport"
	PIIDriverLicense PIIType = "driver_license"
	PIIDateOfBirth PIIType = "date_of_birth"
	PIIAddress     PIIType = "address"
	PIIName        PIIType = "name"
	PIIPassword    PIIType = "password"
	PIIToken       PIIType = "token"
)

// PIIDetection represents a detected PII entity
type PIIDetection struct {
	Type      PIIType `json:"type"`
	Value     string  `json:"value"`
	Start     int     `json:"start"`
	End       int     `json:"end"`
	Masked    string  `json:"masked"`
	Confirmed bool    `json:"confirmed"`
}

// PrivacyConfig defines privacy filtering configuration
type PrivacyConfig struct {
	// EnablePIIRedaction enables automatic PII redaction
	EnablePIIRedaction bool
	// EnableResponseFiltering enables filtering of responses
	EnableResponseFiltering bool
	// EnableAuditLogging enables compliance audit logging
	EnableAuditLogging bool
	// EnabledTypes specifies which PII types to detect
	EnabledTypes []PIIType
	// CustomPatterns additional detection patterns
	CustomPatterns map[string]*regexp.Regexp
}

// DefaultPrivacyConfig returns the default privacy configuration
func DefaultPrivacyConfig() *PrivacyConfig {
	return &PrivacyConfig{
		EnablePIIRedaction:       true,
		EnableResponseFiltering:  true,
		EnableAuditLogging:       true,
		EnabledTypes: []PIIType{
			PIIEmail, PIIPhone, PIICreditCard, PIISSN,
			PIIIPAddress, PIIPassport, PIIDriverLicense,
			PIIPassword, PIIToken,
		},
		CustomPatterns: make(map[string]*regexp.Regexp),
	}
}

// EnhancedEngine provides advanced PII detection and redaction
type EnhancedEngine struct {
	patterns  map[PIIType]*regexp.Regexp
	config    *PrivacyConfig
	maskFuncs map[PIIType]func(string) string
}

// NewEnhancedEngine creates a new enhanced privacy engine
func NewEnhancedEngine(config *PrivacyConfig) *EnhancedEngine {
	if config == nil {
		config = DefaultPrivacyConfig()
	}

	engine := &EnhancedEngine{
		patterns:  make(map[PIIType]*regexp.Regexp),
		config:    config,
		maskFuncs: make(map[PIIType]func(string) string),
	}

	// Initialize default patterns
	engine.patterns[PIIEmail] = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	engine.patterns[PIIPhone] = regexp.MustCompile(`\+?\d{1,4}?[-.\s]?\(?\d{1,3}?\)?[-.\s]?\d{1,4}[-.\s]?\d{1,4}[-.\s]?\d{1,9}`)
	engine.patterns[PIICreditCard] = regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)
	engine.patterns[PIISSN] = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	engine.patterns[PIIIPAddress] = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	engine.patterns[PIIPassport] = regexp.MustCompile(`\b[A-Z]{1,2}\d{6,9}\b`)
	engine.patterns[PIIDriverLicense] = regexp.MustCompile(`\b[A-Z]{1,2}\d{5,8}\b`)
	engine.patterns[PIIPassword] = regexp.MustCompile(`(?i)(?:password|pwd|pass)\s*[:=]\s*\S+`)
	engine.patterns[PIIToken] = regexp.MustCompile(`(?i)(?:api_key|apikey|token|auth)\s*[:=]\s*['"]?[\w-]{16,}['"]?`)
	
	// Date of birth patterns
	engine.patterns[PIIDateOfBirth] = regexp.MustCompile(`\b(?:DOB|dob|Date\s*of\s*birth)\s*[:=]\s*\d{1,2}[-\/]\d{1,2}[-\/]\d{2,4}\b`)

	// Initialize mask functions
	engine.maskFuncs[PIIEmail] = func(s string) string {
		parts := strings.Split(s, "@")
		if len(parts) == 2 {
			local := parts[0]
			domain := parts[1]
			masked := strings.Repeat("*", len(local)-2) + local[len(local)-2:]
			return masked + "@" + domain
		}
		return "[REDACTED_EMAIL]"
	}
	engine.maskFuncs[PIIPhone] = func(s string) string {
		return "[REDACTED_PHONE]"
	}
	engine.maskFuncs[PIICreditCard] = func(s string) string {
		digits := strings.ReplaceAll(s, " ", "")
		if len(digits) >= 4 {
			return "****-****-****-" + digits[len(digits)-4:]
		}
		return "[REDACTED_CARD]"
	}
	engine.maskFuncs[PIISSN] = func(s string) string {
		return "***-**-****"
	}
	engine.maskFuncs[PIIIPAddress] = func(s string) string {
		return "[REDACTED_IP]"
	}
	engine.maskFuncs[PIIPassport] = func(s string) string {
		return "[REDACTED_PASSPORT]"
	}
	engine.maskFuncs[PIIDriverLicense] = func(s string) string {
		return "[REDACTED_LICENSE]"
	}
	engine.maskFuncs[PIIPassword] = func(s string) string {
		return "[REDACTED_PASSWORD]"
	}
	engine.maskFuncs[PIIToken] = func(s string) string {
		return "[REDACTED_TOKEN]"
	}
	engine.maskFuncs[PIIDateOfBirth] = func(s string) string {
		return "[REDACTED_DOB]"
	}

	// Add custom patterns
	for name, pattern := range config.CustomPatterns {
		piType := PIIType(name)
		engine.patterns[piType] = pattern
	}

	return engine
}

// DetectPII finds all PII entities in text
func (e *EnhancedEngine) DetectPII(text string) []PIIDetection {
	var detections []PIIDetection

	for _, piiType := range e.config.EnabledTypes {
		pattern, ok := e.patterns[piiType]
		if !ok {
			continue
		}

		indices := pattern.FindAllStringIndex(text, -1)
		for _, idx := range indices {
			match := text[idx[0]:idx[1]]
			detections = append(detections, PIIDetection{
				Type:      piiType,
				Value:     match,
				Start:     idx[0],
				End:       idx[1],
				Masked:    e.maskFuncs[piiType](match),
				Confirmed: e.confirmMatch(piiType, match),
			})
		}
	}

	return detections
}

// confirmMatch provides additional validation for detected PII
func (e *EnhancedEngine) confirmMatch(piiType PIIType, value string) bool {
	switch piiType {
	case PIICreditCard:
		// Luhn algorithm validation
		return e.luhnCheck(value)
	case PIIIPAddress:
		// Basic IP format check
		return e.isValidIP(value)
	default:
		return true
	}
}

// luhnCheck validates credit card numbers using Luhn algorithm
func (e *EnhancedEngine) luhnCheck(card string) bool {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, card)

	if len(digits) < 13 || len(digits) > 19 {
		return false
	}

	sum := 0
	alternate := false
	for i := len(digits) - 1; i >= 0; i-- {
		n := int(digits[i] - '0')
		if alternate {
			n *= 2
			if n > 9 {
				n = (n % 10) + 1
			}
		}
		sum += n
		alternate = !alternate
	}

	return sum%10 == 0
}

// isValidIP checks if string is a valid IP address
func (e *EnhancedEngine) isValidIP(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		val := 0
		for _, c := range part {
			if c >= '0' && c <= '9' {
				val = val*10 + int(c-'0')
			} else {
				return false
			}
		}
		if val < 0 || val > 255 {
			return false
		}
	}
	return true
}

// Redact replaces all detected PII with masked values
func (e *EnhancedEngine) Redact(text string) string {
	if !e.config.EnablePIIRedaction {
		return text
	}

	result := text
	detections := e.DetectPII(text)

	// Process detections from end to start to preserve indices
	for i := len(detections) - 1; i >= 0; i-- {
		d := detections[i]
		if d.Confirmed || e.config.EnablePIIRedaction {
			result = result[:d.Start] + d.Masked + result[d.End:]
		}
	}

	return result
}

// RedactStructured replaces PII in structured data (JSON-like maps)
func (e *EnhancedEngine) RedactStructured(data map[string]interface{}) map[string]interface{} {
	if !e.config.EnablePIIRedaction {
		return data
	}

	result := make(map[string]interface{})
	for key, value := range data {
		switch v := value.(type) {
		case string:
			result[key] = e.Redact(v)
		case map[string]interface{}:
			result[key] = e.RedactStructured(v)
		case []interface{}:
			result[key] = e.RedactSlice(v)
		default:
			result[key] = v
		}
	}
	return result
}

// RedactSlice redacts PII in slices
func (e *EnhancedEngine) RedactSlice(data []interface{}) []interface{} {
	result := make([]interface{}, len(data))
	for i, item := range data {
		switch v := item.(type) {
		case string:
			result[i] = e.Redact(v)
		case map[string]interface{}:
			result[i] = e.RedactStructured(v)
		default:
			result[i] = v
		}
	}
	return result
}

// FilterResponse filters potentially sensitive content from LLM responses
func (e *EnhancedEngine) FilterResponse(response string) string {
	if !e.config.EnableResponseFiltering {
		return response
	}

	result := response

	// Remove potential code injection attempts
	injectionPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:system|exec|eval|os\.system|subprocess)\s*\(`),
		regexp.MustCompile(`(?i)import\s+os|from\s+os\s+import`),
		regexp.MustCompile(`(?i)rm\s+-rf|rm\s+/|del\s+/[a-z]`),
		regexp.MustCompile(`(?i)(?:password|secret|key)\s*=\s*['"][^'"]+['"]`),
	}
	for _, pattern := range injectionPatterns {
		result = pattern.ReplaceAllString(result, "[FILTERED]")
	}

	return result
}

// GetEnabledTypes returns list of enabled PII types
func (e *EnhancedEngine) GetEnabledTypes() []PIIType {
	return e.config.EnabledTypes
}

// AddCustomPattern adds a custom detection pattern
func (e *EnhancedEngine) AddCustomPattern(name string, pattern *regexp.Regexp) {
	piType := PIIType(name)
	e.patterns[piType] = pattern
	e.maskFuncs[piType] = func(s string) string {
		return "[REDACTED_" + strings.ToUpper(name) + "]"
	}
}

// PrivacyReport provides a summary of privacy scanning results
type PrivacyReport struct {
	TotalDetections int                `json:"total_detections"`
	ByType          map[PIIType]int    `json:"by_type"`
	Redactions      int                `json:"redactions"`
	Filters         int                `json:"filters"`
	Timestamp       string             `json:"timestamp"`
}

// GenerateReport creates a privacy report from detections
func GenerateReport(detections []PIIDetection, originalLen, resultLen int) PrivacyReport {
	byType := make(map[PIIType]int)
	for _, d := range detections {
		byType[d.Type]++
	}

	return PrivacyReport{
		TotalDetections: len(detections),
		ByType:          byType,
		Redactions:      originalLen - resultLen,
		Filters:         0,
		Timestamp:       "2026-05-30T00:00:00Z",
	}
}
