package config

import (
	"os"
)

// Config holds all environmental configurations for the application
type Config struct {
	Port        string
	PostgresURL string
	JWKSPrivate string
	OIDCIssuer  string
}

// Load reads all configurations from environment variables and sets defaults
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "20129" // Default to LocalAgent OmniRoute port
	}

	postgresURL := os.Getenv("DATABASE_URL")
	if postgresURL == "" {
		postgresURL = "postgres://postgres:postgres@localhost:5432/vectors?sslmode=disable"
	}

	oidcIssuer := os.Getenv("OIDC_ISSUER")
	if oidcIssuer == "" {
		oidcIssuer = "http://localhost:20129"
	}

	return &Config{
		Port:        port,
		PostgresURL: postgresURL,
		OIDCIssuer:  oidcIssuer,
	}
}
