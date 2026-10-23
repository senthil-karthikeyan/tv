package store_test

import (
	"sync"
	"testing"

	"tv/internal/model"
	"tv/internal/store"
)

func TestSettingsStore_Defaults(t *testing.T) {
	s := store.NewSettingsStore()
	st := s.Get()

	if st.Theme != "dark" {
		t.Errorf("expected default theme 'dark', got %q", st.Theme)
	}
	if st.Language != "en" {
		t.Errorf("expected default language 'en', got %q", st.Language)
	}
	if st.AutoRefreshInterval != 30 {
		t.Errorf("expected default interval 30, got %d", st.AutoRefreshInterval)
	}
	if st.MaxStreamQuality != "1080p" {
		t.Errorf("expected default quality '1080p', got %q", st.MaxStreamQuality)
	}
	if !st.Notifications {
		t.Error("expected default notifications true")
	}
	if !st.TelemetryEnabled {
		t.Error("expected default telemetry true")
	}
	if st.MaintenanceMode {
		t.Error("expected default maintenance mode false")
	}
	if st.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
}

func TestSettingsStore_Update_Partial(t *testing.T) {
	s := store.NewSettingsStore()
	initial := s.Get()

	newTheme := "light"
	newInterval := 60
	maintenance := true

	updated := s.Update(&model.UpdateSettingsRequest{
		Theme:               &newTheme,
		AutoRefreshInterval: &newInterval,
		MaintenanceMode:     &maintenance,
	})

	if updated.Theme != "light" {
		t.Errorf("expected theme 'light', got %q", updated.Theme)
	}
	if updated.AutoRefreshInterval != 60 {
		t.Errorf("expected interval 60, got %d", updated.AutoRefreshInterval)
	}
	if !updated.MaintenanceMode {
		t.Error("expected maintenance mode true")
	}

	// Verify untouched fields retained
	if updated.Language != initial.Language {
		t.Errorf("language modified unexpectedly: %q", updated.Language)
	}
	if updated.MaxStreamQuality != initial.MaxStreamQuality {
		t.Errorf("quality modified unexpectedly: %q", updated.MaxStreamQuality)
	}
	if updated.Notifications != initial.Notifications {
		t.Errorf("notifications modified unexpectedly: %v", updated.Notifications)
	}
}

func TestSettingsStore_Reset(t *testing.T) {
	s := store.NewSettingsStore()

	newTheme := "system"
	maintenance := true
	s.Update(&model.UpdateSettingsRequest{
		Theme:           &newTheme,
		MaintenanceMode: &maintenance,
	})

	reset := s.Reset()

	if reset.Theme != "dark" {
		t.Errorf("expected reset theme 'dark', got %q", reset.Theme)
	}
	if reset.MaintenanceMode {
		t.Error("expected reset maintenance mode false")
	}
}

func TestSettingsStore_ConcurrentAccess(t *testing.T) {
	s := store.NewSettingsStore()
	var wg sync.WaitGroup

	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				_ = s.Get()
			} else {
				interval := 10 + idx
				_ = s.Update(&model.UpdateSettingsRequest{
					AutoRefreshInterval: &interval,
				})
			}
		}(i)
	}

	wg.Wait()
}
