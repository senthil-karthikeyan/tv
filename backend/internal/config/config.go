// Package config loads and validates application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application-wide configuration values.
// It is loaded once at startup and passed explicitly to the components that need it.
type Config struct {
	// Port is the TCP port the HTTP server listens on. Defaults to 8080.
	Port int
	// FrontendURL is the allowed CORS origin and WebSocket origin. Required.
	FrontendURL string
	// APIKey is the secret clients must supply in X-API-Key on protected routes. Required.
	APIKey string
}

// Load reads configuration from environment variables, applies defaults, and validates every field.
// It returns a non-nil error if any required variable is missing or any value is invalid.
func Load() (*Config, error) {

	godotenv.Load()
	
	port := 8080

	if raw := os.Getenv("PORT"); raw != "" {
		p, err := strconv.Atoi(raw)
		if err != nil || p < 1 || p > 65535 {
			return nil, fmt.Errorf("invalid PORT %q: must be an integer between 1 and 65535", raw)
		}
		port = p
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		return nil, fmt.Errorf("FRONTEND_URL is required")
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API_KEY is required")
	}
	if strings.TrimSpace(apiKey) != apiKey {
		return nil, fmt.Errorf("API_KEY must not have leading or trailing whitespace")
	}

	return &Config{
		Port:        port,
		FrontendURL: frontendURL,
		APIKey:      apiKey,
	}, nil
}
