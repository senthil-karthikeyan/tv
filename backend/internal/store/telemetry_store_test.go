package store_test

import (
	"fmt"
	"sync"
	"testing"

	"tv/internal/model"
	"tv/internal/store"
)

func TestTelemetryStore_Record(t *testing.T) {
	s := store.NewTelemetryStore()

	req := model.IngestEventRequest{
		Type:   "video_start",
		Source: "player",
		UserID: "usr_123",
		Payload: map[string]any{
			"content_id": "movie_456",
		},
	}

	evt, err := s.Record(&req)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	if evt.ID == "" || len(evt.ID) <= 4 {
		t.Errorf("expected valid tel_ prefix ID, got %q", evt.ID)
	}
	if evt.Type != "video_start" || evt.Source != "player" {
		t.Errorf("unexpected event data: %+v", evt)
	}
	if evt.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	events := s.List("", "", 10)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != evt.ID {
		t.Errorf("expected event ID %q, got %q", evt.ID, events[0].ID)
	}
}

func TestTelemetryStore_RecordBatch(t *testing.T) {
	s := store.NewTelemetryStore()

	reqs := []model.IngestEventRequest{
		{Type: "page_view", Source: "launcher"},
		{Type: "scene_selected", Source: "launcher"},
		{Type: "video_start", Source: "player"},
	}

	evts, err := s.RecordBatch(reqs)
	if err != nil {
		t.Fatalf("RecordBatch failed: %v", err)
	}
	if len(evts) != 3 {
		t.Fatalf("expected 3 events, got %d", len(evts))
	}

	list := s.List("", "", 10)
	if len(list) != 3 {
		t.Fatalf("expected 3 stored events, got %d", len(list))
	}

	// Newest first order
	if list[0].Type != "video_start" {
		t.Errorf("expected newest event 'video_start' at index 0, got %q", list[0].Type)
	}
}

func TestTelemetryStore_RingBufferEviction(t *testing.T) {
	// Create store with tiny capacity of 3
	s := store.NewTelemetryStoreWithCapacity(3)

	for i := 1; i <= 5; i++ {
		_, err := s.Record(&model.IngestEventRequest{
			Type:   fmt.Sprintf("event_%d", i),
			Source: "test",
		})
		if err != nil {
			t.Fatalf("Record %d failed: %v", i, err)
		}
	}

	list := s.List("", "", 10)
	if len(list) != 3 {
		t.Fatalf("expected bounded count of 3, got %d", len(list))
	}

	// Should contain event_5, event_4, event_3 (event_1 and event_2 evicted)
	if list[0].Type != "event_5" || list[1].Type != "event_4" || list[2].Type != "event_3" {
		t.Errorf("unexpected buffer contents: %v, %v, %v", list[0].Type, list[1].Type, list[2].Type)
	}

	summary := s.Summary()
	if summary.TotalRecorded != 5 {
		t.Errorf("expected lifetime total 5, got %d", summary.TotalRecorded)
	}
}

func TestTelemetryStore_List_Filtering(t *testing.T) {
	s := store.NewTelemetryStore()

	s.Record(&model.IngestEventRequest{Type: "click", Source: "launcher"})
	s.Record(&model.IngestEventRequest{Type: "click", Source: "player"})
	s.Record(&model.IngestEventRequest{Type: "error", Source: "player"})
	s.Record(&model.IngestEventRequest{Type: "heartbeat", Source: "launcher"})

	// Filter by type
	clicks := s.List("click", "", 10)
	if len(clicks) != 2 {
		t.Errorf("expected 2 click events, got %d", len(clicks))
	}

	// Filter by source
	playerEvents := s.List("", "player", 10)
	if len(playerEvents) != 2 {
		t.Errorf("expected 2 player events, got %d", len(playerEvents))
	}

	// Filter by both
	playerErrors := s.List("error", "player", 10)
	if len(playerErrors) != 1 {
		t.Errorf("expected 1 player error event, got %d", len(playerErrors))
	}

	// Limit
	limited := s.List("", "", 2)
	if len(limited) != 2 {
		t.Errorf("expected 2 limited events, got %d", len(limited))
	}
}

func TestTelemetryStore_Summary(t *testing.T) {
	s := store.NewTelemetryStore()

	s.Record(&model.IngestEventRequest{Type: "video_start", Source: "player"})
	s.Record(&model.IngestEventRequest{Type: "video_start", Source: "player"})
	s.Record(&model.IngestEventRequest{Type: "page_view", Source: "web"})

	summary := s.Summary()
	if summary.TotalRecorded != 3 {
		t.Errorf("expected 3 total recorded, got %d", summary.TotalRecorded)
	}
	if summary.ByType["video_start"] != 2 || summary.ByType["page_view"] != 1 {
		t.Errorf("unexpected ByType map: %+v", summary.ByType)
	}
	if summary.BySource["player"] != 2 || summary.BySource["web"] != 1 {
		t.Errorf("unexpected BySource map: %+v", summary.BySource)
	}
	if summary.LastEventAt == nil || summary.LastEventAt.IsZero() {
		t.Error("expected non-nil LastEventAt")
	}
}

func TestTelemetryStore_ConcurrentAccess(t *testing.T) {
	s := store.NewTelemetryStoreWithCapacity(50)
	var wg sync.WaitGroup

	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = s.Record(&model.IngestEventRequest{
				Type:   fmt.Sprintf("type_%d", idx%5),
				Source: "concurrent_test",
			})
			_ = s.List("", "", 10)
			_ = s.Summary()
		}(i)
	}

	wg.Wait()

	if s.Summary().TotalRecorded != 40 {
		t.Errorf("expected 40 recorded events, got %d", s.Summary().TotalRecorded)
	}
}
