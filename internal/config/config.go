package config

import (
	"os"
)

// Config holds all environmental configurations for the application
type Config struct {
	Port        string
	PostgresURL string
	RedisURL    string
	JWKSPrivate string
	OIDCIssuer  string
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

	return &Config{
		Port:        port,
		PostgresURL: postgresURL,
		RedisURL:    redisURL,
		OIDCIssuer:  oidcIssuer,
	}
}
