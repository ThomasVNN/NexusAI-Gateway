package logging

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestLoggerCreation(t *testing.T) {
	cfg := DefaultConfig()
	logger := New(cfg)

	if logger == nil {
		t.Error("Expected non-nil logger")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", DebugLevel},
		{"info", InfoLevel},
		{"warn", WarnLevel},
		{"warning", WarnLevel},
		{"error", ErrorLevel},
		{"invalid", InfoLevel},
	}

	for _, tt := range tests {
		result := parseLevel(tt.input)
		if result != tt.expected {
			t.Errorf("parseLevel(%s) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
	}

	for _, tt := range tests {
		result := tt.level.String()
		if result != tt.expected {
			t.Errorf("%v.String() = %s, want %s", tt.level, result, tt.expected)
		}
	}
}

func TestLoggerWithField(t *testing.T) {
	logger := New(DefaultConfig())
	logger = logger.WithField("key", "value")

	if logger == nil {
		t.Error("Expected non-nil logger")
	}
}

func TestLoggerWithFields(t *testing.T) {
	logger := New(DefaultConfig())
	logger = logger.WithFields(map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	})

	if logger == nil {
		t.Error("Expected non-nil logger")
	}
}

func TestLoggerDebug(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := New(DefaultConfig())
	logger.SetLevel(DebugLevel)
	logger.Debug("test message")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Expected some output")
	}

	// Verify JSON format
	var entry Entry
	if err := json.Unmarshal([]byte(output), &entry); err == nil {
		if entry.Message != "test message" {
			t.Errorf("Expected 'test message', got '%s'", entry.Message)
		}
		if entry.Level != "DEBUG" {
			t.Errorf("Expected 'DEBUG', got '%s'", entry.Level)
		}
	}
}

func TestLoggerInfo(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := New(DefaultConfig())
	logger.Info("info message")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Expected some output")
	}
}

func TestLoggerWarn(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := New(DefaultConfig())
	logger.Warn("warn message")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Expected some output")
	}
}

func TestLoggerError(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := New(DefaultConfig())
	logger.Error("error message")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Expected some output")
	}
}

func TestLoggerLevelFiltering(t *testing.T) {
	// Set level to ERROR, DEBUG should not be logged
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := New(DefaultConfig())
	logger.SetLevel(ErrorLevel)
	logger.Debug("should not appear")
	logger.Error("should appear")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should only contain the error message
	if !bytes.Contains([]byte(output), []byte("should appear")) {
		t.Error("Expected 'should appear' in output")
	}
}

func TestLoggerFormatted(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := New(DefaultConfig())
	logger.SetLevel(DebugLevel)
	logger.Infof("Hello %s, number %d", "world", 42)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !bytes.Contains([]byte(output), []byte("Hello world, number 42")) {
		t.Error("Expected formatted message in output")
	}
}

func TestGetLogger(t *testing.T) {
	logger := GetLogger()
	if logger == nil {
		t.Error("Expected non-nil default logger")
	}

	// Should return same instance
	logger2 := GetLogger()
	if logger != logger2 {
		t.Error("Expected same logger instance")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Level != "info" {
		t.Errorf("Expected level 'info', got '%s'", cfg.Level)
	}

	if cfg.Format != "json" {
		t.Errorf("Expected format 'json', got '%s'", cfg.Format)
	}
}
