package privacy

import (
	"regexp"
	"strings"
)

// PIIType represents the type of personally identifiable information
type PIIType string

const (
	PIITypeEmail      PIIType = "email"
	PIITypePhone      PIIType = "phone"
	PIITypeSSN        PIIType = "ssn"
	PIITypeCreditCard PIIType = "credit_card"
	PIITypeIPAddress  PIIType = "ip_address"
	PIITypeCustom     PIIType = "custom"
)

// PIIPattern defines a regex pattern for detecting specific PII types
type PIIPattern struct {
	Type    PIIType
	Pattern *regexp.Regexp
	Marker  string
	Enabled bool
	Strict  bool
}

// Detector provides PII detection capabilities with configurable patterns
type Detector struct {
	patterns map[PIIType]*PIIPattern
}

// DefaultMarker tokens for redacted PII types
var DefaultMarkers = map[PIIType]string{
	PIITypeEmail:      "[REDACTED_EMAIL]",
	PIITypePhone:      "[REDACTED_PHONE]",
	PIITypeSSN:        "[REDACTED_SSN]",
	PIITypeCreditCard: "[REDACTED_CARD]",
	PIITypeIPAddress:  "[REDACTED_IP]",
	PIITypeCustom:     "[REDACTED]",
}

// Precompiled patterns for common PII types
var (
	emailPattern = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

	// Phone patterns: supports international (+1-), US formats, various separators
	phonePattern = regexp.MustCompile(`(?:\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}`)

	// SSN pattern: XXX-XX-XXXX format
	ssnPattern = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)

	// Credit card pattern: supports 13-19 digits with various separators
	creditCardPattern = regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)

	// IPv4 pattern
	ipv4Pattern = regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`)

	// IPv6 pattern (simplified)
	ipv6Pattern = regexp.MustCompile(`\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`)
)

// NewDetector creates a new PII detector with default patterns
func NewDetector() *Detector {
	return &Detector{
		patterns: map[PIIType]*PIIPattern{
			PIITypeEmail: {
				Type:    PIITypeEmail,
				Pattern: emailPattern,
				Marker:  DefaultMarkers[PIITypeEmail],
				Enabled: true,
				Strict:  false,
			},
			PIITypePhone: {
				Type:    PIITypePhone,
				Pattern: phonePattern,
				Marker:  DefaultMarkers[PIITypePhone],
				Enabled: true,
				Strict:  false,
			},
			PIITypeSSN: {
				Type:    PIITypeSSN,
				Pattern: ssnPattern,
				Marker:  DefaultMarkers[PIITypeSSN],
				Enabled: true,
				Strict:  true,
			},
			PIITypeCreditCard: {
				Type:    PIITypeCreditCard,
				Pattern: creditCardPattern,
				Marker:  DefaultMarkers[PIITypeCreditCard],
				Enabled: true,
				Strict:  true,
			},
			PIITypeIPAddress: {
				Type:    PIITypeIPAddress,
				Pattern: ipv4Pattern,
				Marker:  DefaultMarkers[PIITypeIPAddress],
				Enabled: true,
				Strict:  false,
			},
		},
	}
}

// DetectResult represents a single PII detection result
type DetectResult struct {
	Type     PIIType
	Value    string
	Start    int
	End      int
	Verified bool
}

// Detect scans the input text and returns all detected PII matches
func (d *Detector) Detect(text string) []DetectResult {
	var results []DetectResult

	for piiType, pattern := range d.patterns {
		if !pattern.Enabled {
			continue
		}

		matches := pattern.Pattern.FindAllStringIndex(text, -1)
		for _, match := range matches {
			value := text[match[0]:match[1]]

			// For strict patterns, perform additional validation
			verified := true
			if pattern.Strict {
				verified = d.verifyMatch(piiType, value)
			}

			results = append(results, DetectResult{
				Type:     piiType,
				Value:    value,
				Start:    match[0],
				End:      match[1],
				Verified: verified,
			})
		}
	}

	return results
}

// DetectWithFilter returns PII matches filtered by the specified PII types
func (d *Detector) DetectWithFilter(text string, filterTypes []PIIType) []DetectResult {
	var results []DetectResult
	typeSet := make(map[PIIType]bool)
	for _, t := range filterTypes {
		typeSet[t] = true
	}

	for piiType, pattern := range d.patterns {
		if !pattern.Enabled || !typeSet[piiType] {
			continue
		}

		matches := pattern.Pattern.FindAllStringIndex(text, -1)
		for _, match := range matches {
			value := text[match[0]:match[1]]
			results = append(results, DetectResult{
				Type:     piiType,
				Value:    value,
				Start:    match[0],
				End:      match[1],
				Verified: true,
			})
		}
	}

	return results
}

// verifyMatch performs additional validation for strict PII patterns
func (d *Detector) verifyMatch(piiType PIIType, value string) bool {
	switch piiType {
	case PIITypeCreditCard:
		return validateLuhn(value)
	case PIITypeSSN:
		return validateSSN(value)
	default:
		return true
	}
}

// validateLuhn performs the Luhn algorithm check for credit card validation
// Reference: https://en.wikipedia.org/wiki/Luhn_algorithm
func validateLuhn(card string) bool {
	// Remove non-digit characters
	var digits []int
	for _, c := range card {
		if c >= '0' && c <= '9' {
			digits = append(digits, int(c-'0'))
		}
	}

	if len(digits) < 13 || len(digits) > 19 {
		return false
	}

	// Luhn algorithm
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
	// Remove dashes
	digits := strings.ReplaceAll(ssn, "-", "")

	// Must be exactly 9 digits
	if len(digits) != 9 {
		return false
	}

	// Area number (first 3 digits) cannot be 000, 666, or 900-999
	area := digits[:3]
	if area == "000" || area == "666" || digits[0] == '9' {
		return false
	}

	// Group number (middle 2 digits) cannot be 00
	if digits[3:5] == "00" {
		return false
	}

	// Serial number (last 4 digits) cannot be 0000
	if digits[5:9] == "0000" {
		return false
	}

	return true
}

// EnablePattern enables detection for a specific PII type
func (d *Detector) EnablePattern(piiType PIIType) {
	if pattern, ok := d.patterns[piiType]; ok {
		pattern.Enabled = true
	}
}

// DisablePattern disables detection for a specific PII type
func (d *Detector) DisablePattern(piiType PIIType) {
	if pattern, ok := d.patterns[piiType]; ok {
		pattern.Enabled = false
	}
}

// SetMarker sets a custom redaction marker for a specific PII type
func (d *Detector) SetMarker(piiType PIIType, marker string) {
	if pattern, ok := d.patterns[piiType]; ok {
		pattern.Marker = marker
	}
}

// GetMarker returns the current marker for a specific PII type
func (d *Detector) GetMarker(piiType PIIType) string {
	if pattern, ok := d.patterns[piiType]; ok {
		return pattern.Marker
	}
	return DefaultMarkers[PIITypeCustom]
}

// AddCustomPattern adds a custom regex pattern for PII detection
func (d *Detector) AddCustomPattern(name string, pattern *regexp.Regexp, marker string) {
	d.patterns[PIITypeCustom] = &PIIPattern{
		Type:    PIITypeCustom,
		Pattern: pattern,
		Marker:  marker,
		Enabled: true,
		Strict:  false,
	}
}

// GetEnabledTypes returns a list of currently enabled PII types
func (d *Detector) GetEnabledTypes() []PIIType {
	var enabled []PIIType
	for piiType, pattern := range d.patterns {
		if pattern.Enabled {
			enabled = append(enabled, piiType)
		}
	}
	return enabled
}

// GetAllTypes returns all registered PII types
func (d *Detector) GetAllTypes() []PIIType {
	var all []PIIType
	for piiType := range d.patterns {
		all = append(all, piiType)
	}
	return all
}
