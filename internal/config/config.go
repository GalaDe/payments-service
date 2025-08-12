package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application.
type Config struct {
	Port             string
	DatabaseURL      string
	StripeAPIKey     string
	PlaidClientID    string
	PlaidSecret      string
	PlaidEnv         string
	TemporalHostPort string
	LogLevel         string
}

// Load loads environment variables into the Config struct.
func Load() (*Config, error) {
	// Load from .env file if present (optional)
	_ = godotenv.Load()

	cfg := &Config{
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      mustEnv("DATABASE_URL"),
		StripeAPIKey:     mustEnv("STRIPE_API_KEY"),
		PlaidClientID:    mustEnv("PLAID_CLIENT_ID"),
		PlaidSecret:      mustEnv("PLAID_SECRET"),
		PlaidEnv:         getEnv("PLAID_ENV", "sandbox"), // sandbox | development | production
		TemporalHostPort: getEnv("TEMPORAL_HOST_PORT", "localhost:7233"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
	}

	return cfg, nil
}

// mustEnv returns the value of the env var or errors if missing.
func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("missing required environment variable: %s", key))
	}
	return val
}

// getEnv returns the env var value or default if unset.
func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
