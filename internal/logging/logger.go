package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// Level represents log level
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging
type Logger struct {
	mu     sync.Mutex
	level  Level
	output *os.File
	fields map[string]interface{}
}

// Entry represents a log entry
type Entry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// Config contains logger configuration
type Config struct {
	Level  string
	Format string
}

// DefaultConfig returns default logger configuration
func DefaultConfig() Config {
	return Config{
		Level:  "info",
		Format: "json",
	}
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// GetLogger returns the default logger
func GetLogger() *Logger {
	once.Do(func() {
		defaultLogger = New(DefaultConfig())
	})
	return defaultLogger
}

// New creates a new logger
func New(cfg Config) *Logger {
	level := parseLevel(cfg.Level)

	return &Logger{
		level:  level,
		output: os.Stdout,
		fields: make(map[string]interface{}),
	}
}

// parseLevel converts string to log level
func parseLevel(s string) Level {
	switch s {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// WithField adds a field to all subsequent log entries
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &Logger{
		level:  l.level,
		output: l.output,
		fields: newFields,
	}
}

// WithFields adds multiple fields to all subsequent log entries
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		level:  l.level,
		output: l.output,
		fields: newFields,
	}
}

// log writes a log entry
func (l *Logger) log(level Level, msg string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := Entry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level.String(),
		Message:   msg,
	}

	// Merge fields
	allFields := make(map[string]interface{})
	for k, v := range l.fields {
		allFields[k] = v
	}
	for k, v := range fields {
		allFields[k] = v
	}
	if len(allFields) > 0 {
		entry.Fields = allFields
	}

	// Write entry
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal log entry: %v", err)
		return
	}

	fmt.Fprintln(l.output, string(data))
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(DebugLevel, msg, f)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(InfoLevel, msg, f)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(WarnLevel, msg, f)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(ErrorLevel, msg, f)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Debug(fmt.Sprintf(format, args...))
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(fmt.Sprintf(format, args...))
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Warn(fmt.Sprintf(format, args...))
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Error(fmt.Sprintf(format, args...))
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	// For JSON logging, nothing to flush
	return nil
}
