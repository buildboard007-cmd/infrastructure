package models

import (
	"time"
)

// Organization represents an organization in the system
type Organization struct {
	OrgID     int64     `json:"org_id"`
	OrgName   string    `json:"org_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdateOrganizationRequest represents the request payload for updating an organization
type UpdateOrganizationRequest struct {
	OrgName string `json:"org_name" validate:"required,min=3,max=150"`
}