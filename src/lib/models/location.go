package models

import (
	"time"
)

// Location represents a physical or virtual location within an organization based on iam.locations table
// Examples: offices, warehouses, job sites, yards
type Location struct {
	ID           int64     `json:"id"`             // Unique location identifier (matches schema: id)
	OrgID        int64     `json:"org_id"`         // Organization this location belongs to
	Name         string    `json:"name"`           // Display name of the location (matches schema: name)
	LocationType string    `json:"location_type"`  // 'office', 'warehouse', 'job_site', 'yard'
	Address      string    `json:"address,omitempty"` // Optional physical address
	City         string    `json:"city,omitempty"`
	State        string    `json:"state,omitempty"`
	ZipCode      string    `json:"zip_code,omitempty"`
	Country      string    `json:"country,omitempty"`
	Status       string    `json:"status"`         // 'active', 'inactive', 'under_construction', 'closed'
	CreatedAt    time.Time `json:"created_at"`     // Creation timestamp
	CreatedBy    int64     `json:"created_by"`     // User who created this location
	UpdatedAt    time.Time `json:"updated_at"`     // Last update timestamp
	UpdatedBy    int64     `json:"updated_by"`     // User who last updated this location
}

// CreateLocationRequest represents the request payload for creating a new location
type CreateLocationRequest struct {
	Name         string `json:"name" binding:"required,min=2,max=255"`
	LocationType string `json:"location_type,omitempty" binding:"omitempty,oneof=office warehouse job_site yard"`
	Address      string `json:"address,omitempty"`
	City         string `json:"city,omitempty" binding:"omitempty,max=100"`
	State        string `json:"state,omitempty" binding:"omitempty,max=50"`
	ZipCode      string `json:"zip_code,omitempty" binding:"omitempty,max=20"`
	Country      string `json:"country,omitempty" binding:"omitempty,max=100"`
	Status       string `json:"status,omitempty" binding:"omitempty,oneof=active inactive under_construction closed"`
}

// UpdateLocationRequest represents the request payload for updating an existing location
type UpdateLocationRequest struct {
	Name         string `json:"name,omitempty" binding:"omitempty,min=2,max=255"`
	LocationType string `json:"location_type,omitempty" binding:"omitempty,oneof=office warehouse job_site yard"`
	Address      string `json:"address,omitempty"`
	City         string `json:"city,omitempty" binding:"omitempty,max=100"`
	State        string `json:"state,omitempty" binding:"omitempty,max=50"`
	ZipCode      string `json:"zip_code,omitempty" binding:"omitempty,max=20"`
	Country      string `json:"country,omitempty" binding:"omitempty,max=100"`
	Status       string `json:"status,omitempty" binding:"omitempty,oneof=active inactive under_construction closed"`
}

// LocationListResponse represents the response for listing locations
type LocationListResponse struct {
	Locations []Location `json:"locations"`
	Total     int        `json:"total"`
}