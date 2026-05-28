package privacy

import (
	"regexp"
)

type Engine struct {
	patterns []*regexp.Regexp
}

func NewEngine() *Engine {
	// Compile default PII detection regex patterns
	emailPattern := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	phonePattern := regexp.MustCompile(`\+?\d{1,4}?[-.\s]?\(?\d{1,3}?\)?[-.\s]?\d{1,4}[-.\s]?\d{1,4}[-.\s]?\d{1,9}`)

	return &Engine{
		patterns: []*regexp.Regexp{emailPattern, phonePattern},
	}
}

// Redact sanitizes prompt input by redacting matched PII fields
func (e *Engine) Redact(text string) string {
	result := text
	for _, p := range e.patterns {
		result = p.ReplaceAllString(result, "[REDACTED]")
	}
	return result
}
