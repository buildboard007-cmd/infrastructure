package models

import (
	"time"
)

// Location represents a physical or virtual location within an organization
// Examples: offices, stores, warehouses, regions, departments
type Location struct {
	LocationID  int64     `json:"location_id"`  // Unique location identifier
	OrgID       int64     `json:"org_id"`       // Organization this location belongs to
	LocationName string   `json:"location_name"` // Display name of the location
	Address     string    `json:"address,omitempty"` // Optional physical address
	CreatedAt   time.Time `json:"created_at"`   // Creation timestamp
	UpdatedAt   time.Time `json:"updated_at"`   // Last update timestamp
}

// CreateLocationRequest represents the request payload for creating a new location
type CreateLocationRequest struct {
	LocationName string `json:"location_name" binding:"required,min=2,max=100"`
	Address      string `json:"address,omitempty"`
}

// UpdateLocationRequest represents the request payload for updating an existing location
type UpdateLocationRequest struct {
	LocationName string `json:"location_name,omitempty" binding:"omitempty,min=2,max=100"`
	Address      string `json:"address,omitempty"`
}

// LocationListResponse represents the response for listing locations
type LocationListResponse struct {
	Locations []Location `json:"locations"`
	Total     int        `json:"total"`
}