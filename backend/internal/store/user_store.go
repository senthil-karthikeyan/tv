package store

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"tv/internal/model"
)

// ErrDuplicateEmail is returned when attempting to create or update a user with an email already taken.
var ErrDuplicateEmail = errors.New("email already in use")

// UserStore is a concurrency-safe in-memory store for User entities.
type UserStore struct {
	mu      sync.RWMutex
	users   map[string]*model.User
	byEmail map[string]string // normalized email -> user ID
}

// NewUserStore creates a UserStore populated with default seed users.
func NewUserStore() *UserStore {
	s := &UserStore{
		users:   make(map[string]*model.User),
		byEmail: make(map[string]string),
	}
	s.seed()
	return s
}

// NewEmptyUserStore creates an empty UserStore for unit testing.
func NewEmptyUserStore() *UserStore {
	return &UserStore{
		users:   make(map[string]*model.User),
		byEmail: make(map[string]string),
	}
}

func (s *UserStore) seed() {
	now := time.Now().UTC()
	defaults := []model.CreateUserRequest{
		{Name: "Admin User", Email: "admin@ott-dashboard.local", Role: "admin", Status: "active", Avatar: "avatar-1"},
		{Name: "Content Editor", Email: "editor@ott-dashboard.local", Role: "editor", Status: "active", Avatar: "avatar-2"},
		{Name: "Family Viewer", Email: "viewer@ott-dashboard.local", Role: "viewer", Status: "active", Avatar: "avatar-3"},
	}

	for _, req := range defaults {
		id := "usr_" + mustGenerateUserID()
		normEmail := strings.ToLower(strings.TrimSpace(req.Email))
		u := &model.User{
			ID:        id,
			Name:      req.Name,
			Email:     normEmail,
			Role:      req.Role,
			Status:    req.Status,
			Avatar:    req.Avatar,
			CreatedAt: now,
			UpdatedAt: now,
		}
		s.users[id] = u
		s.byEmail[normEmail] = id
	}
}

// List returns a copy of all users sorted by CreatedAt ascending.
func (s *UserStore) List() []model.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.User, 0, len(s.users))
	for _, u := range s.users {
		result = append(result, *u)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	return result
}

// Get returns a single user by ID. Returns ErrNotFound if not found.
func (s *UserStore) Get(id string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.users[id]
	if !ok {
		return nil, ErrNotFound
	}

	cp := *u
	return &cp, nil
}

// GetByEmail returns a single user by email address. Returns ErrNotFound if not found.
func (s *UserStore) GetByEmail(email string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	normEmail := strings.ToLower(strings.TrimSpace(email))
	id, ok := s.byEmail[normEmail]
	if !ok {
		return nil, ErrNotFound
	}

	cp := *s.users[id]
	return &cp, nil
}

// Create inserts a new user record. Returns ErrDuplicateEmail if email is already taken.
func (s *UserStore) Create(req *model.CreateUserRequest) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normEmail := strings.ToLower(strings.TrimSpace(req.Email))
	if _, exists := s.byEmail[normEmail]; exists {
		return nil, ErrDuplicateEmail
	}

	rawID, err := generateUserID()
	if err != nil {
		return nil, err
	}
	id := "usr_" + rawID

	role := req.Role
	if role == "" {
		role = "viewer"
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	now := time.Now().UTC()
	u := &model.User{
		ID:        id,
		Name:      strings.TrimSpace(req.Name),
		Email:     normEmail,
		Role:      role,
		Status:    status,
		Avatar:    req.Avatar,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.users[id] = u
	s.byEmail[normEmail] = id

	cp := *u
	return &cp, nil
}

// Update modifies an existing user record. Only non-nil fields in req are applied.
func (s *UserStore) Update(id string, req *model.UpdateUserRequest) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[id]
	if !ok {
		return nil, ErrNotFound
	}

	if req.Email != nil {
		newEmail := strings.ToLower(strings.TrimSpace(*req.Email))
		if existingID, exists := s.byEmail[newEmail]; exists && existingID != id {
			return nil, ErrDuplicateEmail
		}
		delete(s.byEmail, u.Email)
		u.Email = newEmail
		s.byEmail[newEmail] = id
	}

	if req.Name != nil {
		u.Name = strings.TrimSpace(*req.Name)
	}
	if req.Role != nil {
		u.Role = *req.Role
	}
	if req.Status != nil {
		u.Status = *req.Status
	}
	if req.Avatar != nil {
		u.Avatar = *req.Avatar
	}

	u.UpdatedAt = time.Now().UTC()

	cp := *u
	return &cp, nil
}

// Delete removes a user record and its email index. Returns ErrNotFound if not found.
func (s *UserStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[id]
	if !ok {
		return ErrNotFound
	}

	delete(s.byEmail, u.Email)
	delete(s.users, id)
	return nil
}

func generateUserID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func mustGenerateUserID() string {
	id, err := generateUserID()
	if err != nil {
		panic("user store seed: failed to generate ID: " + err.Error())
	}
	return id
}
