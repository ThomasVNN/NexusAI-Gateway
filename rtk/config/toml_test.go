package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTOMLParser(t *testing.T) {
	parser, err := NewTOMLParser("/path/to/config.toml", true)
	if err != nil {
		t.Fatalf("NewTOMLParser failed: %v", err)
	}
	if parser == nil {
		t.Fatal("NewTOMLParser returned nil")
	}
	if parser.configPath != "/path/to/config.toml" {
		t.Errorf("expected configPath '/path/to/config.toml', got '%s'", parser.configPath)
	}
	if !parser.hotReload {
		t.Error("expected hotReload to be true")
	}
}

func TestNewTOMLParserEmptyPath(t *testing.T) {
	_, err := NewTOMLParser("", true)
	if err == nil {
		t.Error("expected error for empty config path")
	}
}

func TestParse(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	content := []byte(`
version = "1.0"

[global]
enabled = true
default_action = "pass"
log_level = "debug"
max_file_size = 1024

hot_reload = true

[[filter]]
name = "test-filter"
enabled = true
priority = 100
command_type = "shell"
patterns = ["test"]

[[filter.actions]]
type = "replace"
key = "output"
value = "filtered"

[[filter.conditions]]
field = "exit_code"
operator = "eq"
value = 0
`)

	config, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if config.Version != "1.0" {
		t.Errorf("expected version '1.0', got '%s'", config.Version)
	}
	if !config.Global.Enabled {
		t.Error("expected global.enabled to be true")
	}
	if config.Global.LogLevel != "debug" {
		t.Errorf("expected log_level 'debug', got '%s'", config.Global.LogLevel)
	}
	if config.Global.MaxFileSize != 1024 {
		t.Errorf("expected max_file_size 1024, got %d", config.Global.MaxFileSize)
	}
	if !config.HotReload {
		t.Error("expected hot_reload to be true")
	}
	if len(config.Filters) != 1 {
		t.Fatalf("expected 1 filter, got %d", len(config.Filters))
	}

	filter := config.Filters[0]
	if filter.Name != "test-filter" {
		t.Errorf("expected filter name 'test-filter', got '%s'", filter.Name)
	}
	if !filter.Enabled {
		t.Error("expected filter.enabled to be true")
	}
	if filter.Priority != 100 {
		t.Errorf("expected priority 100, got %d", filter.Priority)
	}
	if filter.CommandType != "shell" {
		t.Errorf("expected command_type 'shell', got '%s'", filter.CommandType)
	}
	if len(filter.Patterns) != 1 || filter.Patterns[0] != "test" {
		t.Errorf("expected patterns ['test'], got %v", filter.Patterns)
	}
	if len(filter.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(filter.Actions))
	}
	if filter.Actions[0].Type != "replace" {
		t.Errorf("expected action type 'replace', got '%s'", filter.Actions[0].Type)
	}
	if len(filter.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(filter.Conditions))
	}
	if filter.Conditions[0].Field != "exit_code" {
		t.Errorf("expected condition field 'exit_code', got '%s'", filter.Conditions[0].Field)
	}
}

func TestParseInvalidTOML(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	content := []byte(`
version = "1.0"
invalid syntax here
`)

	_, err := parser.Parse(content)
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	content := []byte(`
version = "2.0"
[global]
enabled = true
`)

	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	parser, _ := NewTOMLParser(configPath, false)
	if err := parser.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	config := parser.GetConfig()
	if config.Version != "2.0" {
		t.Errorf("expected version '2.0', got '%s'", config.Version)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	parser, _ := NewTOMLParser("/nonexistent/config.toml", false)
	if err := parser.Load(); err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	parser, _ := NewTOMLParser(configPath, false)

	config := &TOMLConfig{
		Version:   "3.0",
		HotReload: true,
		Global: GlobalSettings{
			Enabled:       true,
			DefaultAction: "pass",
			LogLevel:      "info",
			MaxFileSize:   2048,
		},
		Filters: []FilterDefinition{
			{
				Name:        "save-test-filter",
				Enabled:     true,
				Priority:    50,
				CommandType: "shell",
				Patterns:    []string{"test"},
				Actions: []FilterAction{
					{Type: "replace", Key: "output", Value: "saved"},
				},
			},
		},
	}

	if err := parser.Save(config); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loadedParser, _ := NewTOMLParser(configPath, false)
	if err := loadedParser.Load(); err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	loaded := loadedParser.GetConfig()
	if loaded.Version != "3.0" {
		t.Errorf("expected version '3.0', got '%s'", loaded.Version)
	}
	if len(loaded.Filters) != 1 {
		t.Fatalf("expected 1 filter, got %d", len(loaded.Filters))
	}
	if loaded.Filters[0].Name != "save-test-filter" {
		t.Errorf("expected filter name 'save-test-filter', got '%s'", loaded.Filters[0].Name)
	}
}

func TestRegisterWatcher(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	called := false
	parser.RegisterWatcher(func(config *TOMLConfig) {
		called = true
	})

	if len(parser.watchers) != 1 {
		t.Errorf("expected 1 watcher, got %d", len(parser.watchers))
	}
}

func TestGetFilters(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	parser.config = &TOMLConfig{
		Filters: []FilterDefinition{
			{Name: "enabled-filter", Enabled: true, Priority: 10},
			{Name: "disabled-filter", Enabled: false, Priority: 20},
			{Name: "another-enabled", Enabled: true, Priority: 5},
		},
	}

	filters := parser.GetFilters()
	if len(filters) != 2 {
		t.Errorf("expected 2 filters, got %d", len(filters))
	}
}

func TestGetFiltersNilConfig(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	filters := parser.GetFilters()
	if filters != nil {
		t.Errorf("expected nil for nil config, got %v", filters)
	}
}

func TestGetFilterByName(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	parser.config = &TOMLConfig{
		Filters: []FilterDefinition{
			{Name: "find-me", Enabled: true},
			{Name: "skip-me", Enabled: false},
		},
	}

	filter := parser.GetFilterByName("find-me")
	if filter == nil {
		t.Fatal("expected to find filter 'find-me'")
	}
	if filter.Name != "find-me" {
		t.Errorf("expected name 'find-me', got '%s'", filter.Name)
	}

	notFound := parser.GetFilterByName("nonexistent")
	if notFound != nil {
		t.Error("expected nil for nonexistent filter")
	}
}

func TestGetFilterByNameNilConfig(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	filter := parser.GetFilterByName("anything")
	if filter != nil {
		t.Error("expected nil for nil config")
	}
}

func TestValidate(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	parser.config = &TOMLConfig{
		Filters: []FilterDefinition{
			{
				Name:        "valid-filter",
				Enabled:     true,
				Priority:    10,
				Actions:     []FilterAction{{Type: "replace", Key: "k"}},
				Conditions:  []Condition{{Field: "exit_code", Operator: "eq"}},
			},
		},
	}

	if err := parser.Validate(); err != nil {
		t.Errorf("Validate failed: %v", err)
	}
}

func TestValidateNilConfig(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	if err := parser.Validate(); err == nil {
		t.Error("expected error for nil config")
	}
}

func TestValidateEmptyFilterName(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	parser.config = &TOMLConfig{
		Filters: []FilterDefinition{
			{Name: "", Enabled: true},
		},
	}

	if err := parser.Validate(); err == nil {
		t.Error("expected error for empty filter name")
	}
}

func TestValidateNegativePriority(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	parser.config = &TOMLConfig{
		Filters: []FilterDefinition{
			{Name: "test", Priority: -1},
		},
	}

	if err := parser.Validate(); err == nil {
		t.Error("expected error for negative priority")
	}
}

func TestValidateEmptyActionType(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	parser.config = &TOMLConfig{
		Filters: []FilterDefinition{
			{
				Name:    "test",
				Actions: []FilterAction{{Type: ""}},
			},
		},
	}

	if err := parser.Validate(); err == nil {
		t.Error("expected error for empty action type")
	}
}

func TestValidateEmptyConditionField(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	parser.config = &TOMLConfig{
		Filters: []FilterDefinition{
			{
				Name:       "test",
				Conditions: []Condition{{Field: "", Operator: "eq"}},
			},
		},
	}

	if err := parser.Validate(); err == nil {
		t.Error("expected error for empty condition field")
	}
}

func TestValidateEmptyConditionOperator(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	parser.config = &TOMLConfig{
		Filters: []FilterDefinition{
			{
				Name:       "test",
				Conditions: []Condition{{Field: "exit_code", Operator: ""}},
			},
		},
	}

	if err := parser.Validate(); err == nil {
		t.Error("expected error for empty condition operator")
	}
}

func TestMerge(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	parser.config = &TOMLConfig{
		Version: "1.0",
		Global: GlobalSettings{
			Enabled: false,
		},
		Filters: []FilterDefinition{
			{Name: "existing-filter", Priority: 10},
		},
	}

	other := &TOMLConfig{
		Version: "2.0",
		Global: GlobalSettings{
			Enabled: true,
		},
		Filters: []FilterDefinition{
			{Name: "new-filter", Priority: 20},
			{Name: "existing-filter", Priority: 100}, // Override
		},
	}

	merged := parser.Merge(other)

	if merged.Version != "1.0" {
		t.Errorf("expected version '1.0', got '%s'", merged.Version)
	}
	if !merged.Global.Enabled {
		t.Error("expected global.enabled to be true after merge")
	}
	if len(merged.Filters) != 2 {
		t.Errorf("expected 2 filters, got %d", len(merged.Filters))
	}

	// Check override worked
	for _, f := range merged.Filters {
		if f.Name == "existing-filter" && f.Priority != 100 {
			t.Errorf("expected existing-filter priority 100, got %d", f.Priority)
		}
	}
}

func TestMergeNilCurrent(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	other := &TOMLConfig{
		Version: "1.0",
		Filters: []FilterDefinition{
			{Name: "only-filter"},
		},
	}

	merged := parser.Merge(other)
	if merged != other {
		t.Error("expected merged to be other when current is nil")
	}
}

func TestMergeNilOther(t *testing.T) {
	parser, _ := NewTOMLParser("/path/to/config.toml", false)

	parser.config = &TOMLConfig{
		Version: "1.0",
		Filters: []FilterDefinition{
			{Name: "only-filter"},
		},
	}

	merged := parser.Merge(nil)
	if merged != parser.config {
		t.Error("expected merged to be current when other is nil")
	}
}
