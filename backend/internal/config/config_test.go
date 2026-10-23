package config_test

import (
	"testing"

	"tv/internal/config"
)

// setEnv sets pairs of (key, value) environment variables using t.Setenv,
// which automatically restores their original values after the test.
func setEnv(t *testing.T, pairs ...string) {
	t.Helper()
	for i := 0; i+1 < len(pairs); i += 2 {
		t.Setenv(pairs[i], pairs[i+1])
	}
}

func TestLoad_ValidDefaults(t *testing.T) {
	setEnv(t,
		"FRONTEND_URL", "http://localhost:4200",
		"API_KEY", "test-key",
		"PORT", "",
	)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Port)
	}
	if cfg.FrontendURL != "http://localhost:4200" {
		t.Errorf("unexpected FrontendURL: %s", cfg.FrontendURL)
	}
	if cfg.APIKey != "test-key" {
		t.Errorf("unexpected APIKey")
	}
}

func TestLoad_CustomPort(t *testing.T) {
	setEnv(t,
		"PORT", "9090",
		"FRONTEND_URL", "http://localhost:4200",
		"API_KEY", "test-key",
	)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Port)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	cases := []struct {
		name string
		port string
	}{
		{"not_a_number", "abc"},
		{"zero", "0"},
		{"above_max", "99999"},
		{"negative", "-1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t,
				"PORT", tc.port,
				"FRONTEND_URL", "http://localhost:4200",
				"API_KEY", "test-key",
			)
			_, err := config.Load()
			if err == nil {
				t.Errorf("expected error for PORT=%q, got nil", tc.port)
			}
		})
	}
}

func TestLoad_MissingFrontendURL(t *testing.T) {
	setEnv(t,
		"FRONTEND_URL", "",
		"API_KEY", "test-key",
		"PORT", "",
	)

	_, err := config.Load()
	if err == nil {
		t.Error("expected error when FRONTEND_URL is empty")
	}
}

func TestLoad_MissingAPIKey(t *testing.T) {
	setEnv(t,
		"FRONTEND_URL", "http://localhost:4200",
		"API_KEY", "",
		"PORT", "",
	)

	_, err := config.Load()
	if err == nil {
		t.Error("expected error when API_KEY is empty")
	}
}

func TestLoad_WhitespaceAPIKey(t *testing.T) {
	setEnv(t,
		"FRONTEND_URL", "http://localhost:4200",
		"API_KEY", "  has-spaces  ",
		"PORT", "",
	)

	_, err := config.Load()
	if err == nil {
		t.Error("expected error when API_KEY has leading/trailing whitespace")
	}
}
