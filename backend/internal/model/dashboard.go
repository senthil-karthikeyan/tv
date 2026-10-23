package model

import "time"

// DashboardStatus provides high-level application status and health indicators.
type DashboardStatus struct {
	Status          string    `json:"status"`           // "healthy", "maintenance"
	Version         string    `json:"version"`          // application version, e.g. "1.0.0"
	Environment     string    `json:"environment"`      // "development", "production", etc.
	MaintenanceMode bool      `json:"maintenance_mode"` // true when maintenance mode is active
	StartedAt       time.Time `json:"started_at"`       // server boot time
	UptimeSeconds   int64     `json:"uptime_seconds"`   // elapsed seconds since boot
}

// ResourceCounts aggregates entity counts across all backend subsystems.
type ResourceCounts struct {
	TotalScenes       int `json:"total_scenes"`
	VisibleScenes     int `json:"visible_scenes"`
	TotalUsers        int `json:"total_users"`
	ActiveUsers       int `json:"active_users"`
	ActiveConnections int `json:"active_connections"`
}

// SystemMetrics captures live Go runtime execution statistics.
type SystemMetrics struct {
	Goroutines    int     `json:"goroutines"`
	MemoryAllocMB float64 `json:"memory_alloc_mb"`
	MemorySysMB   float64 `json:"memory_sys_mb"`
	CPUCores      int     `json:"cpu_cores"`
}

// DashboardStats provides comprehensive operational telemetry and metrics.
type DashboardStats struct {
	Status    DashboardStatus `json:"status"`
	Resources ResourceCounts  `json:"resources"`
	Metrics   SystemMetrics   `json:"metrics"`
	UpdatedAt time.Time       `json:"updated_at"`
}
