package model

import "time"

// Settings holds configurable application preferences and feature flags for the OTT dashboard.
type Settings struct {
	Theme               string    `json:"theme"`                 // "dark", "light", "system"
	Language            string    `json:"language"`              // ISO language code e.g. "en", "es", "fr"
	AutoRefreshInterval int       `json:"auto_refresh_interval"` // seconds: 5..300
	MaxStreamQuality    string    `json:"max_stream_quality"`    // "4K", "1080p", "720p", "480p"
	Notifications       bool      `json:"notifications"`         // push/in-app notification toggle
	TelemetryEnabled    bool      `json:"telemetry_enabled"`     // telemetry tracking toggle
	MaintenanceMode     bool      `json:"maintenance_mode"`      // maintenance mode banner toggle
	UpdatedAt           time.Time `json:"updated_at"`
}

// UpdateSettingsRequest holds the fields that can be partially updated.
// Only non-nil pointer fields are applied; omitted fields retain their current values.
type UpdateSettingsRequest struct {
	Theme               *string `json:"theme"`
	Language            *string `json:"language"`
	AutoRefreshInterval *int    `json:"auto_refresh_interval"`
	MaxStreamQuality    *string `json:"max_stream_quality"`
	Notifications       *bool   `json:"notifications"`
	TelemetryEnabled    *bool   `json:"telemetry_enabled"`
	MaintenanceMode     *bool   `json:"maintenance_mode"`
}
