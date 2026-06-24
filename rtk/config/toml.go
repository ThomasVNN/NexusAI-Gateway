package config

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

// FilterDefinition represents a TOML filter configuration
type FilterDefinition struct {
	Name        string         `toml:"name"`
	Enabled     bool           `toml:"enabled"`
	Priority    int            `toml:"priority"`
	CommandType string         `toml:"command_type"`
	Patterns    []string       `toml:"patterns"`
	Actions     []FilterAction `toml:"actions"`
	Conditions  []Condition    `toml:"conditions"`
}

// FilterAction represents an action to apply
type FilterAction struct {
	Type  string      `toml:"type"` // "replace", "remove", "add", "optimize"
	Key   string      `toml:"key"`
	Value interface{} `toml:"value"`
}

// Condition represents a filter condition
type Condition struct {
	Field    string      `toml:"field"`    // "exit_code", "output_size", "duration", "pattern"
	Operator string      `toml:"operator"` // "eq", "gt", "lt", "contains", "regex"
	Value    interface{} `toml:"value"`
}

// TOMLConfig represents the complete filter configuration
type TOMLConfig struct {
	Version   string             `toml:"version"`
	Filters   []FilterDefinition `toml:"filter"`
	Global    GlobalSettings     `toml:"global"`
	HotReload bool               `toml:"hot_reload"`
	ConfigPath string
}

// GlobalSettings contains global configuration
type GlobalSettings struct {
	Enabled       bool   `toml:"enabled"`
	DefaultAction string `toml:"default_action"`
	LogLevel      string `toml:"log_level"`
	MaxFileSize   int64  `toml:"max_file_size"`
}

// TOMLParser parses and manages TOML configurations
type TOMLParser struct {
	config     *TOMLConfig
	configPath string
	watchers   []func(*TOMLConfig)
	hotReload  bool
	mu         sync.RWMutex
}

// NewTOMLParser creates a new TOML parser
func NewTOMLParser(configPath string, hotReload bool) (*TOMLParser, error) {
	if configPath == "" {
		return nil, fmt.Errorf("config path cannot be empty")
	}
	return &TOMLParser{
		configPath: configPath,
		hotReload:  hotReload,
		watchers:   make([]func(*TOMLConfig), 0),
	}, nil
}

// Load loads configuration from file
func (p *TOMLParser) Load() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	content, err := os.ReadFile(p.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	config, err := p.Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	config.ConfigPath = p.configPath
	p.config = config
	return nil
}

// Parse parses TOML content
func (p *TOMLParser) Parse(content []byte) (*TOMLConfig, error) {
	var config TOMLConfig
	if err := toml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal TOML: %w", err)
	}
	return &config, nil
}

// Save saves configuration to file
func (p *TOMLParser) Save(config *TOMLConfig) error {
	p.mu.Lock()

	file, err := os.Create(p.configPath)
	if err != nil {
		p.mu.Unlock()
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	if err := toml.NewEncoder(file).Encode(config); err != nil {
		p.mu.Unlock()
		return fmt.Errorf("failed to encode TOML: %w", err)
	}

	// Copy watchers before modifying state
	watchers := make([]func(*TOMLConfig), len(p.watchers))
	copy(watchers, p.watchers)

	p.config = config
	p.mu.Unlock()

	// Notify outside of lock
	for _, watcher := range watchers {
		watcher(p.config)
	}

	return nil
}

// Watch starts watching for file changes
func (p *TOMLParser) Watch(ctx context.Context) error {
	if !p.hotReload {
		return nil
	}

	watcher, err := os.Open(p.configPath)
	if err != nil {
		return fmt.Errorf("failed to watch file: %w", err)
	}

	var lastModTime time.Time
	stat, _ := os.Stat(p.configPath)
	if stat != nil {
		lastModTime = stat.ModTime()
	}
	watcher.Close()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			stat, err := os.Stat(p.configPath)
			if err != nil {
				continue
			}
			if stat.ModTime().After(lastModTime) {
				lastModTime = stat.ModTime()
				if err := p.Load(); err != nil {
					continue
				}
				// Copy config and watchers before releasing lock
				config := p.GetConfig()
				p.mu.RLock()
				watchers := make([]func(*TOMLConfig), len(p.watchers))
				copy(watchers, p.watchers)
				p.mu.RUnlock()
				for _, watcher := range watchers {
					watcher(config)
				}
			}
		}
	}
}

// RegisterWatcher registers a change callback
func (p *TOMLParser) RegisterWatcher(watcher func(*TOMLConfig)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.watchers = append(p.watchers, watcher)
}

// notifyWatchers notifies all registered watchers
func (p *TOMLParser) notifyWatchers() {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, watcher := range p.watchers {
		watcher(p.config)
	}
}

// GetConfig returns current configuration
func (p *TOMLParser) GetConfig() *TOMLConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}

// GetFilters returns enabled filters
func (p *TOMLParser) GetFilters() []FilterDefinition {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.config == nil {
		return nil
	}

	enabled := make([]FilterDefinition, 0)
	for _, filter := range p.config.Filters {
		if filter.Enabled {
			enabled = append(enabled, filter)
		}
	}
	return enabled
}

// GetFilterByName returns a filter by name
func (p *TOMLParser) GetFilterByName(name string) *FilterDefinition {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.config == nil {
		return nil
	}

	for i := range p.config.Filters {
		if p.config.Filters[i].Name == name {
			return &p.config.Filters[i]
		}
	}
	return nil
}

// Validate validates the configuration
func (p *TOMLParser) Validate() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.config == nil {
		return fmt.Errorf("config is nil")
	}

	for _, filter := range p.config.Filters {
		if filter.Name == "" {
			return fmt.Errorf("filter name cannot be empty")
		}
		if filter.Priority < 0 {
			return fmt.Errorf("filter %s has invalid priority: %d", filter.Name, filter.Priority)
		}
		for _, action := range filter.Actions {
			if action.Type == "" {
				return fmt.Errorf("filter %s has action without type", filter.Name)
			}
		}
		for _, condition := range filter.Conditions {
			if condition.Field == "" {
				return fmt.Errorf("filter %s has condition without field", filter.Name)
			}
			if condition.Operator == "" {
				return fmt.Errorf("filter %s has condition without operator", filter.Name)
			}
		}
	}
	return nil
}

// Merge merges another config into current
func (p *TOMLParser) Merge(other *TOMLConfig) *TOMLConfig {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.config == nil {
		return other
	}
	if other == nil {
		return p.config
	}

	merged := &TOMLConfig{
		Version:    p.config.Version,
		HotReload:  p.config.HotReload,
		ConfigPath: p.config.ConfigPath,
		Global:     p.config.Global,
		Filters:    make([]FilterDefinition, len(p.config.Filters)),
	}
	copy(merged.Filters, p.config.Filters)

	for _, filter := range other.Filters {
		found := false
		for i, existing := range merged.Filters {
			if existing.Name == filter.Name {
				merged.Filters[i] = filter
				found = true
				break
			}
		}
		if !found {
			merged.Filters = append(merged.Filters, filter)
		}
	}

	if other.Global.Enabled {
		merged.Global = other.Global
	}

	return merged
}
