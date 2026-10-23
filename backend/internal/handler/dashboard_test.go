package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tv/internal/handler"
	"tv/internal/model"
	"tv/internal/store"
)

type mockClientCounter struct {
	count int
}

func (m *mockClientCounter) ClientCount() int {
	return m.count
}

func TestDashboardHandler_Status_Returns200(t *testing.T) {
	sceneStore := store.NewSceneStore()
	userStore := store.NewUserStore()
	settingsStore := store.NewSettingsStore()
	startedAt := time.Now().UTC().Add(-10 * time.Second)

	h := handler.NewDashboardHandler(sceneStore, userStore, settingsStore, nil, startedAt, "1.0.0", "test")

	r := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/status", nil)
	rr := httptest.NewRecorder()

	h.Status(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Data model.DashboardStatus `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode status: %v", err)
	}

	status := resp.Data
	if status.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", status.Status)
	}
	if status.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", status.Version)
	}
	if status.Environment != "test" {
		t.Errorf("expected environment 'test', got %q", status.Environment)
	}
	if status.MaintenanceMode {
		t.Error("expected maintenance mode false")
	}
	if status.UptimeSeconds < 10 {
		t.Errorf("expected uptime >= 10s, got %d", status.UptimeSeconds)
	}
}

func TestDashboardHandler_Status_MaintenanceMode(t *testing.T) {
	settingsStore := store.NewSettingsStore()
	maintenance := true
	settingsStore.Update(&model.UpdateSettingsRequest{
		MaintenanceMode: &maintenance,
	})

	h := handler.NewDashboardHandler(nil, nil, settingsStore, nil, time.Now().UTC(), "1.0.0", "test")

	r := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/status", nil)
	rr := httptest.NewRecorder()

	h.Status(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Data model.DashboardStatus `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode status: %v", err)
	}

	if resp.Data.Status != "maintenance" {
		t.Errorf("expected status 'maintenance', got %q", resp.Data.Status)
	}
	if !resp.Data.MaintenanceMode {
		t.Error("expected maintenance mode true")
	}
}

func TestDashboardHandler_Stats_Returns200AndCounts(t *testing.T) {
	sceneStore := store.NewSceneStore() // 3 seeded scenes, all visible
	userStore := store.NewUserStore()   // 3 seeded users, all active
	settingsStore := store.NewSettingsStore()
	counter := &mockClientCounter{count: 7}
	startedAt := time.Now().UTC().Add(-5 * time.Second)

	h := handler.NewDashboardHandler(sceneStore, userStore, settingsStore, counter, startedAt, "1.0.0", "production")

	r := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/stats", nil)
	rr := httptest.NewRecorder()

	h.Stats(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Data model.DashboardStats `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode stats: %v", err)
	}

	stats := resp.Data

	// Verify Status block
	if stats.Status.Status != "healthy" || stats.Status.Environment != "production" {
		t.Errorf("unexpected status block: %+v", stats.Status)
	}

	// Verify Resource counts
	if stats.Resources.TotalScenes != 3 {
		t.Errorf("expected 3 total scenes, got %d", stats.Resources.TotalScenes)
	}
	if stats.Resources.VisibleScenes != 3 {
		t.Errorf("expected 3 visible scenes, got %d", stats.Resources.VisibleScenes)
	}
	if stats.Resources.TotalUsers != 3 {
		t.Errorf("expected 3 total users, got %d", stats.Resources.TotalUsers)
	}
	if stats.Resources.ActiveUsers != 3 {
		t.Errorf("expected 3 active users, got %d", stats.Resources.ActiveUsers)
	}
	if stats.Resources.ActiveConnections != 7 {
		t.Errorf("expected 7 active connections, got %d", stats.Resources.ActiveConnections)
	}

	// Verify System metrics
	if stats.Metrics.Goroutines <= 0 {
		t.Errorf("expected positive goroutines, got %d", stats.Metrics.Goroutines)
	}
	if stats.Metrics.CPUCores <= 0 {
		t.Errorf("expected positive cpu cores, got %d", stats.Metrics.CPUCores)
	}
	if stats.Metrics.MemoryAllocMB <= 0 {
		t.Errorf("expected positive memory alloc, got %f", stats.Metrics.MemoryAllocMB)
	}
}
