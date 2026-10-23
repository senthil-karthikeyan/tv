package model

import "time"

// User represents an OTT dashboard user profile.
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`   // "admin", "editor", "viewer"
	Status    string    `json:"status"` // "active", "inactive", "suspended"
	Avatar    string    `json:"avatar,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUserRequest holds the client-supplied fields for creating a new user.
// Server-generated fields (ID, CreatedAt, UpdatedAt) are set by the store.
type CreateUserRequest struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`   // defaults to "viewer" if empty
	Status string `json:"status"` // defaults to "active" if empty
	Avatar string `json:"avatar"`
}

// UpdateUserRequest holds the fields accepted when updating an existing user.
// Only non-nil pointer fields are applied; omitted fields retain their current value.
type UpdateUserRequest struct {
	Name   *string `json:"name"`
	Email  *string `json:"email"`
	Role   *string `json:"role"`
	Status *string `json:"status"`
	Avatar *string `json:"avatar"`
}
