package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config holds all environmental configurations for the application
type Config struct {
	Port            string
	PostgresURL     string
	RedisURL        string
	JWKSPrivate     string
	OIDCIssuer      string
	InitialPassword string
	AppEnv          string // local, development, staging, production
	UpstreamAPIURL  string
	UpstreamAPIKey  string
}

// UnsafeDefaults contains known unsafe default values for detection
var UnsafeDefaults = []string{
	"postgres_secure_pass",
	"admin",
	"mock-key-for-local-dev",
	"change-me-before-production",
	"password",
	"secret",
}

// Load reads all configurations from environment variables and sets defaults
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "20129"
	}

	// SECURITY: No default password - must be provided via environment variable
	initialPassword := os.Getenv("INITIAL_PASSWORD")
	if initialPassword == "" {
		initialPassword = os.Getenv("OMNIROUTE_ADMIN_KEY")
	}
	// Note: Password validation happens in Validate() for production environments

	appEnv := strings.ToLower(os.Getenv("APP_ENV"))
	if appEnv == "" {
		appEnv = "development"
	}

	// SECURITY: Only load database URL from env - no hardcoded credentials
	postgresURL := os.Getenv("DATABASE_URL")

	redisURL := os.Getenv("REDIS_URL")

	oidcIssuer := os.Getenv("OIDC_ISSUER")
	if oidcIssuer == "" {
		oidcIssuer = "http://localhost:20129"
	}

	upstreamAPIURL := os.Getenv("UPSTREAM_API_URL")
	upstreamAPIKey := os.Getenv("UPSTREAM_API_KEY")

	return &Config{
		Port:            port,
		PostgresURL:     postgresURL,
		RedisURL:        redisURL,
		OIDCIssuer:      oidcIssuer,
		InitialPassword: initialPassword,
		AppEnv:          appEnv,
		UpstreamAPIURL:  upstreamAPIURL,
		UpstreamAPIKey:  upstreamAPIKey,
	}
}

// Validate ensures there are no unsafe defaults in production and all values are safe
func (c *Config) Validate() error {
	// Validate AppEnv value
	switch c.AppEnv {
	case "local", "development", "staging", "production":
		// OK
	default:
		return fmt.Errorf("invalid APP_ENV: %q. Must be one of: local, development, staging, production", c.AppEnv)
	}

	// Validate Port format
	portNum, err := strconv.Atoi(c.Port)
	if err != nil || portNum <= 0 || portNum > 65535 {
		return fmt.Errorf("invalid PORT: %q. Must be a valid port number (1-65535)", c.Port)
	}

	// In production or staging, enforce strict checks
	if c.AppEnv == "production" || c.AppEnv == "staging" {
		// SECURITY: No fallback administrative password allowed - must be provided
		if c.InitialPassword == "" {
			return fmt.Errorf("administrative password (INITIAL_PASSWORD) is required in %s environment", c.AppEnv)
		}

		// Check against known unsafe defaults
		for _, unsafe := range UnsafeDefaults {
			if c.InitialPassword == unsafe {
				return fmt.Errorf("unsafe default administrative password (INITIAL_PASSWORD) is not allowed in %s environment", c.AppEnv)
			}
		}

		if c.PostgresURL == "" {
			return fmt.Errorf("PostgreSQL database URL (DATABASE_URL) is required in %s environment", c.AppEnv)
		}

		// Ensure we don't use default localhost or default hostnames in staging/production
		u, err := url.Parse(c.PostgresURL)
		if err != nil {
			return fmt.Errorf("failed to parse DATABASE_URL: %w", err)
		}
		if u.Host == "localhost:5432" || u.Host == "127.0.0.1:5432" {
			return fmt.Errorf("unsafe default host in DATABASE_URL for %s environment", c.AppEnv)
		}

		// Check for unsafe password in database URL
		if u.User != nil {
			pwd, set := u.User.Password()
			if set {
				for _, unsafe := range UnsafeDefaults {
					if pwd == unsafe {
						return fmt.Errorf("unsafe default credential in DATABASE_URL for %s environment", c.AppEnv)
					}
				}
			}
		}

		// Redis URL validation
		if c.RedisURL != "" {
			ru, err := url.Parse(c.RedisURL)
			if err == nil {
				if ru.Host == "localhost:6379" || ru.Host == "127.0.0.1:6379" {
					return fmt.Errorf("unsafe default host in REDIS_URL for %s environment", c.AppEnv)
				}
			}
		}
	}

	return nil
}

// GetEnvironment returns the current environment without exposing internal state
func (c *Config) GetEnvironment() string {
	return c.AppEnv
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// ValidateNoUnsafeDefaults checks if a value matches any known unsafe default
func ValidateNoUnsafeDefaults(value string) bool {
	for _, unsafe := range UnsafeDefaults {
		if value == unsafe {
			return false
		}
	}
	return true
}
