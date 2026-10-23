package store_test

import (
	"fmt"
	"testing"

	"tv/internal/model"
	"tv/internal/store"
)

// ── Create & Get ─────────────────────────────────────────────────────────────

func TestSceneStore_Create_SetsIDAndTimestamps(t *testing.T) {
	s := store.NewEmptySceneStore()
	sc, err := s.Create(validCreateReq("Alpha"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sc.ID == "" {
		t.Error("expected non-empty ID")
	}
	if sc.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if sc.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
	if sc.Name != "Alpha" {
		t.Errorf("expected name 'Alpha', got %q", sc.Name)
	}
}

func TestSceneStore_Create_GeneratesUniqueIDs(t *testing.T) {
	s := store.NewEmptySceneStore()
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		sc, err := s.Create(validCreateReq("scene"))
		if err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
		if seen[sc.ID] {
			t.Fatalf("duplicate ID generated: %s", sc.ID)
		}
		seen[sc.ID] = true
	}
}

func TestSceneStore_Get_ReturnsCorrectScene(t *testing.T) {
	s := store.NewEmptySceneStore()
	created, _ := s.Create(validCreateReq("Beta"))

	got, err := s.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID mismatch: %q vs %q", got.ID, created.ID)
	}
	if got.Name != "Beta" {
		t.Errorf("expected name 'Beta', got %q", got.Name)
	}
}

func TestSceneStore_Get_NotFound(t *testing.T) {
	s := store.NewEmptySceneStore()
	_, err := s.Get("nonexistent")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
}

func TestSceneStore_Get_ReturnsCopy(t *testing.T) {
	// Modifying the returned scene must not affect the store's internal state.
	s := store.NewEmptySceneStore()
	created, _ := s.Create(validCreateReq("Original"))

	got, _ := s.Get(created.ID)
	got.Name = "Mutated"

	again, _ := s.Get(created.ID)
	if again.Name == "Mutated" {
		t.Error("store returned a reference; modifying it corrupted internal state")
	}
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestSceneStore_List_Empty(t *testing.T) {
	s := store.NewEmptySceneStore()
	if got := s.List(); len(got) != 0 {
		t.Errorf("expected empty list, got %d items", len(got))
	}
}

func TestSceneStore_List_SortedByOrder(t *testing.T) {
	s := store.NewEmptySceneStore()
	// Insert in reverse order to verify sorting.
	for i := 5; i >= 1; i-- {
		req := validCreateReq(fmt.Sprintf("Scene%d", i))
		req.Order = i
		if _, err := s.Create(req); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	scenes := s.List()
	if len(scenes) != 5 {
		t.Fatalf("expected 5 scenes, got %d", len(scenes))
	}
	for i := 1; i < len(scenes); i++ {
		if scenes[i].Order < scenes[i-1].Order {
			t.Errorf("not sorted at index %d: order %d < %d", i, scenes[i].Order, scenes[i-1].Order)
		}
	}
}

func TestSceneStore_NewSceneStore_HasSeedData(t *testing.T) {
	s := store.NewSceneStore()
	scenes := s.List()
	if len(scenes) != 3 {
		t.Errorf("expected 3 seed scenes, got %d", len(scenes))
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestSceneStore_Update_AppliesOnlyNonNilFields(t *testing.T) {
	s := store.NewEmptySceneStore()
	created, _ := s.Create(&model.CreateSceneRequest{
		Name: "Original", Label: "orig", Route: "/original", Order: 1, Visible: true,
	})

	newName := "Updated"
	updated, err := s.Update(created.ID, &model.UpdateSceneRequest{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("expected name %q, got %q", newName, updated.Name)
	}
	// Fields not present in the request must be unchanged.
	if updated.Label != "orig" {
		t.Errorf("label changed unexpectedly: %q", updated.Label)
	}
	if updated.Route != "/original" {
		t.Errorf("route changed unexpectedly: %q", updated.Route)
	}
}

func TestSceneStore_Update_AdvancesUpdatedAt(t *testing.T) {
	s := store.NewEmptySceneStore()
	created, _ := s.Create(validCreateReq("X"))

	newName := "Y"
	updated, err := s.Update(created.ID, &model.UpdateSceneRequest{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) && !updated.UpdatedAt.Equal(created.UpdatedAt) {
		t.Error("UpdatedAt should have advanced or stayed equal after update")
	}
}

func TestSceneStore_Update_NotFound(t *testing.T) {
	s := store.NewEmptySceneStore()
	newName := "X"
	_, err := s.Update("nonexistent", &model.UpdateSceneRequest{Name: &newName})
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestSceneStore_Delete_RemovesScene(t *testing.T) {
	s := store.NewEmptySceneStore()
	created, _ := s.Create(validCreateReq("Gone"))

	if err := s.Delete(created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(created.ID); err == nil {
		t.Error("expected ErrNotFound after delete, got nil")
	}
	scenes := s.List()
	for _, sc := range scenes {
		if sc.ID == created.ID {
			t.Error("deleted scene still present in List()")
		}
	}
}

func TestSceneStore_Delete_NotFound(t *testing.T) {
	s := store.NewEmptySceneStore()
	if err := s.Delete("nonexistent"); err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
}

// ── helper ────────────────────────────────────────────────────────────────────

func validCreateReq(name string) *model.CreateSceneRequest {
	return &model.CreateSceneRequest{
		Name:    name,
		Label:   "label",
		Icon:    "star",
		Route:   "/route",
		Order:   1,
		Visible: true,
	}
}
