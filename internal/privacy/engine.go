package privacy

import (
	"regexp"
)

type Engine struct {
	patterns map[string]*regexp.Regexp
}

func NewEngine() *Engine {
	return &Engine{
		patterns: map[string]*regexp.Regexp{
			"email":       regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
			"phone":       regexp.MustCompile(`\+?\d{1,4}?[-.\s]?\(?\d{1,3}?\)?[-.\s]?\d{1,4}[-.\s]?\d{1,4}[-.\s]?\d{1,9}`),
			"credit_card": regexp.MustCompile(`\b(?:\d[ -]*?){13,16}\b`),
			"ssn":         regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		},
	}
}

// Redact sanitizes prompt input by redacting matched PII fields with descriptive tokens
func (e *Engine) Redact(text string) string {
	result := text
	for label, p := range e.patterns {
		switch label {
		case "email":
			result = p.ReplaceAllString(result, "[REDACTED_EMAIL]")
		case "phone":
			result = p.ReplaceAllString(result, "[REDACTED_PHONE]")
		case "credit_card":
			result = p.ReplaceAllString(result, "[REDACTED_CARD]")
		case "ssn":
			result = p.ReplaceAllString(result, "[REDACTED_SSN]")
		default:
			result = p.ReplaceAllString(result, "[REDACTED]")
		}
	}
	return result
}
