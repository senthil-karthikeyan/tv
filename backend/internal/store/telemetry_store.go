package store

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"tv/internal/model"
)

const defaultTelemetryCapacity = 1000

// TelemetryStore is a concurrency-safe bounded in-memory store for telemetry events.
// It stores events in a circular ring buffer to prevent unbounded memory growth.
type TelemetryStore struct {
	mu            sync.RWMutex
	capacity      int
	events        []*model.TelemetryEvent
	head          int // index of the oldest item
	count         int // current number of items stored
	totalRecorded int
	byType        map[string]int
	bySource      map[string]int
	lastEventAt   *time.Time
}

// NewTelemetryStore constructs a TelemetryStore with default capacity (1,000 events).
func NewTelemetryStore() *TelemetryStore {
	return NewTelemetryStoreWithCapacity(defaultTelemetryCapacity)
}

// NewTelemetryStoreWithCapacity constructs a TelemetryStore with a custom capacity.
func NewTelemetryStoreWithCapacity(capacity int) *TelemetryStore {
	if capacity <= 0 {
		capacity = defaultTelemetryCapacity
	}
	return &TelemetryStore{
		capacity: capacity,
		events:   make([]*model.TelemetryEvent, capacity),
		byType:   make(map[string]int),
		bySource: make(map[string]int),
	}
}

// Record ingests a single telemetry event.
func (s *TelemetryStore) Record(req *model.IngestEventRequest) (*model.TelemetryEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.insertLocked(req)
}

// RecordBatch ingests a slice of telemetry events in a single locked transaction.
func (s *TelemetryStore) RecordBatch(reqs []model.IngestEventRequest) ([]model.TelemetryEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]model.TelemetryEvent, 0, len(reqs))
	for i := range reqs {
		evt, err := s.insertLocked(&reqs[i])
		if err != nil {
			return nil, err
		}
		result = append(result, *evt)
	}
	return result, nil
}

func (s *TelemetryStore) insertLocked(req *model.IngestEventRequest) (*model.TelemetryEvent, error) {
	rawID, err := generateTelemetryID()
	if err != nil {
		return nil, err
	}
	id := "tel_" + rawID

	ts := time.Now().UTC()
	if req.Timestamp != nil && !req.Timestamp.IsZero() {
		ts = req.Timestamp.UTC()
	}

	evt := &model.TelemetryEvent{
		ID:        id,
		Type:      strings.TrimSpace(req.Type),
		Source:    strings.TrimSpace(req.Source),
		UserID:    strings.TrimSpace(req.UserID),
		SessionID: strings.TrimSpace(req.SessionID),
		Payload:   req.Payload,
		Timestamp: ts,
	}

	// Insert into ring buffer
	if s.count < s.capacity {
		idx := (s.head + s.count) % s.capacity
		s.events[idx] = evt
		s.count++
	} else {
		// Overwrite oldest item at head and advance head
		s.events[s.head] = evt
		s.head = (s.head + 1) % s.capacity
	}

	// Update running stats
	s.totalRecorded++
	s.byType[evt.Type]++
	s.bySource[evt.Source]++
	last := ts
	s.lastEventAt = &last

	cp := *evt
	return &cp, nil
}

// List returns stored telemetry events, newest first, with optional type and source filters.
func (s *TelemetryStore) List(eventType, source string, limit int) []model.TelemetryEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	eventType = strings.TrimSpace(eventType)
	source = strings.TrimSpace(source)

	result := make([]model.TelemetryEvent, 0, limit)

	// Iterate from newest to oldest:
	// Newest item is at index: (s.head + s.count - 1 - i) % s.capacity
	for i := 0; i < s.count && len(result) < limit; i++ {
		idx := (s.head + s.count - 1 - i) % s.capacity
		evt := s.events[idx]
		if evt == nil {
			continue
		}

		if eventType != "" && !strings.EqualFold(evt.Type, eventType) {
			continue
		}
		if source != "" && !strings.EqualFold(evt.Source, source) {
			continue
		}

		result = append(result, *evt)
	}

	return result
}

// Summary returns current aggregate telemetry metrics.
func (s *TelemetryStore) Summary() model.TelemetrySummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	byType := make(map[string]int, len(s.byType))
	for k, v := range s.byType {
		byType[k] = v
	}

	bySource := make(map[string]int, len(s.bySource))
	for k, v := range s.bySource {
		bySource[k] = v
	}

	var lastCopy *time.Time
	if s.lastEventAt != nil {
		t := *s.lastEventAt
		lastCopy = &t
	}

	return model.TelemetrySummary{
		TotalRecorded: s.totalRecorded,
		ByType:        byType,
		BySource:      bySource,
		LastEventAt:   lastCopy,
	}
}

func generateTelemetryID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
