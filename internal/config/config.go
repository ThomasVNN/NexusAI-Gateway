package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	API      APIConfig
	Logging  LoggingConfig
}

// ServerConfig contains server configuration
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	MaxConns int
}

// RedisConfig contains Redis configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

// APIConfig contains API configuration
type APIConfig struct {
	Key            string
	RateLimit      int
	Timeout        time.Duration
	AllowedOrigins []string
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:     getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     getEnvDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Database: getEnv("DB_NAME", "nexusai"),
			MaxConns: getEnvInt("DB_MAX_CONNS", 25),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			PoolSize: getEnvInt("REDIS_POOL_SIZE", 10),
		},
		API: APIConfig{
			Key:            getEnv("API_KEY", ""),
			RateLimit:      getEnvInt("API_RATE_LIMIT", 100),
			Timeout:        getEnvDuration("API_TIMEOUT", 60*time.Second),
			AllowedOrigins: getEnvSlice("API_ALLOWED_ORIGINS", ","),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	var errors []string

	// Validate server config
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errors = append(errors, "Server port must be between 1 and 65535")
	}
	if c.Server.ReadTimeout < 0 {
		errors = append(errors, "Server read timeout must be non-negative")
	}
	if c.Server.WriteTimeout < 0 {
		errors = append(errors, "Server write timeout must be non-negative")
	}

	// Validate database config
	if c.Database.Host == "" {
		errors = append(errors, "Database host is required")
	}
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		errors = append(errors, "Database port must be between 1 and 65535")
	}
	if c.Database.MaxConns < 1 {
		errors = append(errors, "Database max connections must be at least 1")
	}

	// Validate Redis config
	if c.Redis.Host == "" {
		errors = append(errors, "Redis host is required")
	}
	if c.Redis.Port < 1 || c.Redis.Port > 65535 {
		errors = append(errors, "Redis port must be between 1 and 65535")
	}

	// Validate API config
	if c.API.Timeout < 0 {
		errors = append(errors, "API timeout must be non-negative")
	}

	// Validate logging config
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		errors = append(errors, "Log level must be one of: debug, info, warn, error")
	}
	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[c.Logging.Format] {
		errors = append(errors, "Log format must be one of: json, text")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// GetEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt gets an environment variable as an integer or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvDuration gets an environment variable as a duration or returns a default value
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// GetEnvSlice gets an environment variable as a slice or returns a default value
func getEnvSlice(key, separator string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, separator)
	}
	return []string{}
}

// String returns a string representation of the config
func (c *Config) String() string {
	return fmt.Sprintf("Config{Server:%s:%d, Database:%s:%d, Redis:%s:%d}",
		c.Server.Host, c.Server.Port,
		c.Database.Host, c.Database.Port,
		c.Redis.Host, c.Redis.Port)
}
