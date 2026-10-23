package model

import "time"

// Scene is a launcher navigation entry representing one section of the OTT dashboard.
type Scene struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`    // displayed title in the launcher
	Label     string    `json:"label"`   // short tag or subtitle
	Icon      string    `json:"icon"`    // optional icon identifier
	Route     string    `json:"route"`   // Angular router path, e.g. "/dashboard"
	Order     int       `json:"order"`   // display position, ascending
	Visible   bool      `json:"visible"` // whether the scene appears in the launcher
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateSceneRequest holds the client-supplied fields for creating a new scene.
// Server-generated fields (ID, CreatedAt, UpdatedAt) are ignored if present.
type CreateSceneRequest struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Icon    string `json:"icon"`
	Route   string `json:"route"`
	Order   int    `json:"order"`
	Visible bool   `json:"visible"`
}

// UpdateSceneRequest holds the fields accepted when updating an existing scene.
// Only non-nil pointer fields are applied; omitted fields retain their current value.
type UpdateSceneRequest struct {
	Name    *string `json:"name"`
	Label   *string `json:"label"`
	Icon    *string `json:"icon"`
	Route   *string `json:"route"`
	Order   *int    `json:"order"`
	Visible *bool   `json:"visible"`
}
