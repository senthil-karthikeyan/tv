package handler

import (
	"math"
	"net/http"
	"runtime"
	"time"

	"tv/internal/model"
	"tv/internal/store"
)

// ClientCounter is an interface for querying the current count of connected WebSocket clients.
type ClientCounter interface {
	ClientCount() int
}

// DashboardHandler provides endpoints for application health, status, and aggregated system metrics.
type DashboardHandler struct {
	sceneStore    *store.SceneStore
	userStore     *store.UserStore
	settingsStore *store.SettingsStore
	counter       ClientCounter
	startedAt     time.Time
	version       string
	environment   string
}

// NewDashboardHandler constructs a DashboardHandler with references to data stores and server metadata.
func NewDashboardHandler(
	ss *store.SceneStore,
	us *store.UserStore,
	st *store.SettingsStore,
	c ClientCounter,
	startedAt time.Time,
	version string,
	env string,
) *DashboardHandler {
	if version == "" {
		version = "1.0.0"
	}
	if env == "" {
		env = "development"
	}
	return &DashboardHandler{
		sceneStore:    ss,
		userStore:     us,
		settingsStore: st,
		counter:       c,
		startedAt:     startedAt,
		version:       version,
		environment:   env,
	}
}

// Status handles GET /api/v1/dashboard/status.
// Returns high-level application health, uptime, version, and maintenance status.
func (h *DashboardHandler) Status(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	uptime := int64(now.Sub(h.startedAt).Seconds())
	if uptime < 0 {
		uptime = 0
	}

	maintenance := false
	if h.settingsStore != nil {
		maintenance = h.settingsStore.Get().MaintenanceMode
	}

	systemStatus := "healthy"
	if maintenance {
		systemStatus = "maintenance"
	}

	status := model.DashboardStatus{
		Status:          systemStatus,
		Version:         h.version,
		Environment:     h.environment,
		MaintenanceMode: maintenance,
		StartedAt:       h.startedAt,
		UptimeSeconds:   uptime,
	}

	WriteData(w, http.StatusOK, status)
}

// Stats handles GET /api/v1/dashboard/stats.
// Returns comprehensive operational metrics including resource counts and runtime stats.
func (h *DashboardHandler) Stats(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	uptime := int64(now.Sub(h.startedAt).Seconds())
	if uptime < 0 {
		uptime = 0
	}

	maintenance := false
	if h.settingsStore != nil {
		maintenance = h.settingsStore.Get().MaintenanceMode
	}

	systemStatus := "healthy"
	if maintenance {
		systemStatus = "maintenance"
	}

	status := model.DashboardStatus{
		Status:          systemStatus,
		Version:         h.version,
		Environment:     h.environment,
		MaintenanceMode: maintenance,
		StartedAt:       h.startedAt,
		UptimeSeconds:   uptime,
	}

	// Resource counts
	var totalScenes, visibleScenes int
	if h.sceneStore != nil {
		scenes := h.sceneStore.List()
		totalScenes = len(scenes)
		for _, s := range scenes {
			if s.Visible {
				visibleScenes++
			}
		}
	}

	var totalUsers, activeUsers int
	if h.userStore != nil {
		users := h.userStore.List()
		totalUsers = len(users)
		for _, u := range users {
			if u.Status == "active" {
				activeUsers++
			}
		}
	}

	activeConn := 0
	if h.counter != nil {
		activeConn = h.counter.ClientCount()
	}

	resources := model.ResourceCounts{
		TotalScenes:       totalScenes,
		VisibleScenes:     visibleScenes,
		TotalUsers:        totalUsers,
		ActiveUsers:       activeUsers,
		ActiveConnections: activeConn,
	}

	// Runtime system metrics
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	metrics := model.SystemMetrics{
		Goroutines:    runtime.NumGoroutine(),
		MemoryAllocMB: roundToTwoDecimals(float64(mem.Alloc) / (1024 * 1024)),
		MemorySysMB:   roundToTwoDecimals(float64(mem.Sys) / (1024 * 1024)),
		CPUCores:      runtime.NumCPU(),
	}

	stats := model.DashboardStats{
		Status:    status,
		Resources: resources,
		Metrics:   metrics,
		UpdatedAt: now,
	}

	WriteData(w, http.StatusOK, stats)
}

func roundToTwoDecimals(val float64) float64 {
	return math.Round(val*100) / 100
}
