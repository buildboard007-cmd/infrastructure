package models

import (
	"time"
)

// Location represents a physical or virtual location within an organization based on iam.locations table
// Examples: offices, warehouses, job sites, yards
type Location struct {
	LocationID   int64     `json:"location_id"`    // Unique location identifier
	OrgID        int64     `json:"org_id"`         // Organization this location belongs to
	LocationName string    `json:"location_name"`  // Display name of the location
	LocationType string    `json:"location_type"`  // 'office', 'warehouse', 'job_site', 'yard'
	Address      string    `json:"address,omitempty"` // Optional physical address
	City         string    `json:"city,omitempty"`
	State        string    `json:"state,omitempty"`
	ZipCode      string    `json:"zip_code,omitempty"`
	Country      string    `json:"country,omitempty"`
	Status       string    `json:"status"`         // 'active', 'inactive', 'under_construction', 'closed'
	CreatedAt    time.Time `json:"created_at"`     // Creation timestamp
	UpdatedAt    time.Time `json:"updated_at"`     // Last update timestamp
}

// CreateLocationRequest represents the request payload for creating a new location
type CreateLocationRequest struct {
	LocationName string `json:"location_name" binding:"required,min=2,max=100"`
	LocationType string `json:"location_type,omitempty"`
	Address      string `json:"address,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
	ZipCode      string `json:"zip_code,omitempty"`
	Country      string `json:"country,omitempty"`
	Status       string `json:"status,omitempty"`
}

// UpdateLocationRequest represents the request payload for updating an existing location
type UpdateLocationRequest struct {
	LocationName string `json:"location_name,omitempty" binding:"omitempty,min=2,max=100"`
	LocationType string `json:"location_type,omitempty"`
	Address      string `json:"address,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
	ZipCode      string `json:"zip_code,omitempty"`
	Country      string `json:"country,omitempty"`
	Status       string `json:"status,omitempty"`
}

// LocationListResponse represents the response for listing locations
type LocationListResponse struct {
	Locations []Location `json:"locations"`
	Total     int        `json:"total"`
}