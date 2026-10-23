package store_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"tv/internal/model"
	"tv/internal/store"
)

func TestUserStore_SeedUsers(t *testing.T) {
	s := store.NewUserStore()
	users := s.List()

	if len(users) != 3 {
		t.Fatalf("expected 3 seed users, got %d", len(users))
	}

	admin, err := s.GetByEmail("admin@ott-dashboard.local")
	if err != nil {
		t.Fatalf("expected admin user to exist: %v", err)
	}
	if admin.Role != "admin" || admin.Status != "active" {
		t.Errorf("unexpected admin role/status: role=%s, status=%s", admin.Role, admin.Status)
	}
}

func TestUserStore_Create_Success(t *testing.T) {
	s := store.NewEmptyUserStore()

	req := model.CreateUserRequest{
		Name:   "Jane Doe",
		Email:  "Jane.Doe@Example.com",
		Role:   "editor",
		Status: "active",
		Avatar: "avatar-jane",
	}

	u, err := s.Create(&req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if u.ID == "" || len(u.ID) <= 4 {
		t.Errorf("expected valid ID with usr_ prefix, got %q", u.ID)
	}
	if u.Name != "Jane Doe" {
		t.Errorf("expected name 'Jane Doe', got %q", u.Name)
	}
	if u.Email != "jane.doe@example.com" {
		t.Errorf("expected lowercase normalized email, got %q", u.Email)
	}
	if u.Role != "editor" {
		t.Errorf("expected role 'editor', got %q", u.Role)
	}
	if u.CreatedAt.IsZero() || u.UpdatedAt.IsZero() {
		t.Error("expected non-zero timestamps")
	}

	// Verify retrieval by ID and by Email
	byID, err := s.Get(u.ID)
	if err != nil {
		t.Fatalf("Get by ID failed: %v", err)
	}
	if byID.Email != u.Email {
		t.Errorf("retrieved email mismatch: got %q, want %q", byID.Email, u.Email)
	}

	byEmail, err := s.GetByEmail("JANE.DOE@example.com")
	if err != nil {
		t.Fatalf("Get by Email failed: %v", err)
	}
	if byEmail.ID != u.ID {
		t.Errorf("retrieved ID mismatch: got %q, want %q", byEmail.ID, u.ID)
	}
}

func TestUserStore_Create_DuplicateEmail(t *testing.T) {
	s := store.NewEmptyUserStore()

	req1 := model.CreateUserRequest{
		Name:  "User One",
		Email: "same@example.com",
	}
	if _, err := s.Create(&req1); err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	req2 := model.CreateUserRequest{
		Name:  "User Two",
		Email: "SAME@example.com",
	}
	_, err := s.Create(&req2)
	if !errors.Is(err, store.ErrDuplicateEmail) {
		t.Fatalf("expected ErrDuplicateEmail, got %v", err)
	}
}

func TestUserStore_Update(t *testing.T) {
	s := store.NewEmptyUserStore()

	u, err := s.Create(&model.CreateUserRequest{
		Name:  "Original Name",
		Email: "orig@example.com",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	newName := "Updated Name"
	newRole := "admin"
	updated, err := s.Update(u.ID, &model.UpdateUserRequest{
		Name: &newName,
		Role: &newRole,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Name != newName || updated.Role != newRole {
		t.Errorf("fields not updated: %+v", updated)
	}
	if updated.Email != "orig@example.com" {
		t.Errorf("omitted field email modified: %q", updated.Email)
	}
	if !updated.UpdatedAt.After(u.UpdatedAt) && !updated.UpdatedAt.Equal(u.UpdatedAt) {
		t.Error("expected UpdatedAt to advance or match")
	}

	// Test duplicate email on update
	_, _ = s.Create(&model.CreateUserRequest{
		Name:  "Other",
		Email: "other@example.com",
	})

	conflictEmail := "other@example.com"
	_, err = s.Update(u.ID, &model.UpdateUserRequest{
		Email: &conflictEmail,
	})
	if !errors.Is(err, store.ErrDuplicateEmail) {
		t.Fatalf("expected ErrDuplicateEmail on conflict, got %v", err)
	}
}

func TestUserStore_Delete(t *testing.T) {
	s := store.NewEmptyUserStore()

	u, err := s.Create(&model.CreateUserRequest{
		Name:  "To Delete",
		Email: "delete@example.com",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := s.Delete(u.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify user is gone
	if _, err := s.Get(u.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}

	// Verify email index was cleared so the same email can be registered again
	recreated, err := s.Create(&model.CreateUserRequest{
		Name:  "Recreated",
		Email: "delete@example.com",
	})
	if err != nil {
		t.Fatalf("re-creating with deleted email failed: %v", err)
	}
	if recreated.ID == u.ID {
		t.Error("expected new user to have a different ID")
	}
}

func TestUserStore_ConcurrentAccess(t *testing.T) {
	s := store.NewEmptyUserStore()
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			email := fmt.Sprintf("user%d@example.com", idx)
			u, err := s.Create(&model.CreateUserRequest{
				Name:  fmt.Sprintf("User %d", idx),
				Email: email,
			})
			if err != nil {
				t.Errorf("concurrent create %d failed: %v", idx, err)
				return
			}
			_, _ = s.Get(u.ID)
			_, _ = s.GetByEmail(email)
			_ = s.List()
		}(i)
	}

	wg.Wait()
	if len(s.List()) != 20 {
		t.Errorf("expected 20 users, got %d", len(s.List()))
	}
}
