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
	Port                  string
	PostgresURL           string
	RedisURL              string
	JWKSPrivate           string
	OIDCIssuer            string
	InitialPassword       string
	AppEnv                string // local, development, staging, production
	EnableSandboxFallback bool
	UpstreamAPIURL        string
	UpstreamAPIKey        string
}

// Load reads all configurations from environment variables and sets defaults
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "20129"
	}

	postgresURL := os.Getenv("DATABASE_URL")
	if postgresURL == "" {
		postgresURL = "postgres://postgres:postgres_secure_pass@postgres-nexus:5432/nexusai_gateway?sslmode=disable"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://redis-nexus:6379/1"
	}

	oidcIssuer := os.Getenv("OIDC_ISSUER")
	if oidcIssuer == "" {
		oidcIssuer = "http://localhost:20129"
	}

	initialPassword := os.Getenv("INITIAL_PASSWORD")
	if initialPassword == "" {
		initialPassword = os.Getenv("OMNIROUTE_ADMIN_KEY")
	}
	if initialPassword == "" {
		initialPassword = "postgres_secure_pass" // fallback
	}

	appEnv := strings.ToLower(os.Getenv("APP_ENV"))
	if appEnv == "" {
		appEnv = "development"
	}

	enableSandboxFallbackStr := os.Getenv("ENABLE_SANDBOX_FALLBACK")
	enableSandboxFallback, _ := strconv.ParseBool(enableSandboxFallbackStr)

	upstreamAPIURL := os.Getenv("UPSTREAM_API_URL")
	upstreamAPIKey := os.Getenv("UPSTREAM_API_KEY")

	return &Config{
		Port:                  port,
		PostgresURL:           postgresURL,
		RedisURL:              redisURL,
		OIDCIssuer:            oidcIssuer,
		InitialPassword:       initialPassword,
		AppEnv:                appEnv,
		EnableSandboxFallback: enableSandboxFallback,
		UpstreamAPIURL:        upstreamAPIURL,
		UpstreamAPIKey:        upstreamAPIKey,
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
		// No fallback administrative password or empty password allowed
		if c.InitialPassword == "" ||
			c.InitialPassword == "postgres_secure_pass" ||
			c.InitialPassword == "admin" ||
			c.InitialPassword == "mock-key-for-local-dev" ||
			c.InitialPassword == "change-me-before-production" {
			return fmt.Errorf("unsafe default or empty administrative password (INITIAL_PASSWORD) is not allowed in %s environment", c.AppEnv)
		}

		if c.EnableSandboxFallback {
			return fmt.Errorf("sandbox fallback authentication (ENABLE_SANDBOX_FALLBACK) must be disabled in %s environment", c.AppEnv)
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
		if u.User != nil {
			pwd, set := u.User.Password()
			if set && pwd == "postgres_secure_pass" {
				return fmt.Errorf("unsafe default credential in DATABASE_URL for %s environment", c.AppEnv)
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
