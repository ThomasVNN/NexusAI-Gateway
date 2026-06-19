package middleware

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

// ValidationRule represents a validation rule
type ValidationRule struct {
	Field    string
	Type     string // string, number, email, url, enum
	Required bool
	MinLen   int
	MaxLen   int
	Pattern  *regexp.Regexp
	Enum     []string
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Validator provides request validation middleware
type Validator struct {
	rules map[string][]ValidationRule
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		rules: make(map[string][]ValidationRule),
	}
}

// AddRule adds a validation rule for an endpoint
func (v *Validator) AddRule(endpoint string, rules ...ValidationRule) {
	v.rules[endpoint] = append(v.rules[endpoint], rules...)
}

// Middleware returns an HTTP middleware for validation
func (v *Validator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Find rules for this endpoint
		path := r.URL.Path
		rules, exists := v.rules[path]
		if !exists {
			// Try pattern matching
			for pattern, r := range v.rules {
				if matchPath(pattern, path) {
					rules = r
					break
				}
			}
		}

		if len(rules) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		// Parse body if needed
		var body map[string]interface{}
		if r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				ErrorHandler(w, "Invalid JSON body", http.StatusBadRequest)
				return
			}
		}

		// Validate
		if errs := v.validate(body, rules); len(errs) > 0 {
			ErrorHandler(w, "Validation failed", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// validate validates data against rules
func (v *Validator) validate(data map[string]interface{}, rules []ValidationRule) []ValidationError {
	var errors []ValidationError

	for _, rule := range rules {
		value, exists := data[rule.Field]

		// Check required
		if rule.Required && (!exists || value == nil || (rule.Type == "string" && strings.TrimSpace(toString(value)) == "")) {
			errors = append(errors, ValidationError{
				Field:   rule.Field,
				Message: "Field is required",
			})
			continue
		}

		if !exists || value == nil {
			continue
		}

		// Type-specific validation
		switch rule.Type {
		case "string":
			strValue := toString(value)
			if rule.MinLen > 0 && len(strValue) < rule.MinLen {
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: "Field is too short",
				})
			}
			if rule.MaxLen > 0 && len(strValue) > rule.MaxLen {
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: "Field is too long",
				})
			}
			if rule.Pattern != nil && !rule.Pattern.MatchString(strValue) {
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: "Field format is invalid",
				})
			}
		case "email":
			if !isValidEmail(toString(value)) {
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: "Invalid email format",
				})
			}
		case "url":
			if !isValidURL(toString(value)) {
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: "Invalid URL format",
				})
			}
		case "enum":
			if !contains(rule.Enum, toString(value)) {
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: "Field value is not in allowed list",
				})
			}
		}
	}

	return errors
}

// Helper functions

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		return ""
	}
}

func isValidEmail(email string) bool {
	pattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return pattern.MatchString(email)
}

func isValidURL(url string) bool {
	pattern := regexp.MustCompile(`^https?://[^\s]+$`)
	return pattern.MatchString(url)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func matchPath(pattern, path string) bool {
	// Simple pattern matching with *
	parts := strings.Split(pattern, "*")
	for _, part := range parts {
		if part != "" && !strings.Contains(path, part) {
			return false
		}
	}
	return true
}
