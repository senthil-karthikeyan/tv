package store

import (
	"sync"
	"time"

	"tv/internal/model"
)

// SettingsStore is a concurrency-safe in-memory store for application Settings.
type SettingsStore struct {
	mu       sync.RWMutex
	settings model.Settings
}

// defaultSettings returns a fresh Settings instance initialized with default values.
func defaultSettings() model.Settings {
	return model.Settings{
		Theme:               "dark",
		Language:            "en",
		AutoRefreshInterval: 30,
		MaxStreamQuality:    "1080p",
		Notifications:       true,
		TelemetryEnabled:    true,
		MaintenanceMode:     false,
		UpdatedAt:           time.Now().UTC(),
	}
}

// NewSettingsStore creates a SettingsStore populated with system defaults.
func NewSettingsStore() *SettingsStore {
	return &SettingsStore{
		settings: defaultSettings(),
	}
}

// Get returns a copy of the current application settings.
func (s *SettingsStore) Get() model.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// Update modifies settings with the provided non-nil fields in req and updates UpdatedAt.
// Returns a copy of the updated settings.
func (s *SettingsStore) Update(req *model.UpdateSettingsRequest) model.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.Theme != nil {
		s.settings.Theme = *req.Theme
	}
	if req.Language != nil {
		s.settings.Language = *req.Language
	}
	if req.AutoRefreshInterval != nil {
		s.settings.AutoRefreshInterval = *req.AutoRefreshInterval
	}
	if req.MaxStreamQuality != nil {
		s.settings.MaxStreamQuality = *req.MaxStreamQuality
	}
	if req.Notifications != nil {
		s.settings.Notifications = *req.Notifications
	}
	if req.TelemetryEnabled != nil {
		s.settings.TelemetryEnabled = *req.TelemetryEnabled
	}
	if req.MaintenanceMode != nil {
		s.settings.MaintenanceMode = *req.MaintenanceMode
	}

	s.settings.UpdatedAt = time.Now().UTC()
	return s.settings
}

// Reset restores settings to default values and updates UpdatedAt.
// Returns a copy of the default settings.
func (s *SettingsStore) Reset() model.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.settings = defaultSettings()
	return s.settings
}
