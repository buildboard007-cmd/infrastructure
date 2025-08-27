package models

import (
	"time"
)

// Permission represents a permission within an organization based on iam.permission table
type Permission struct {
	PermissionID   int64     `json:"permission_id"`   // Primary key from iam.permission.permission_id
	PermissionName string    `json:"permission_name"` // Permission name (max 100 characters)
	Description    string    `json:"description,omitempty"` // Optional permission description
	OrgID          int64     `json:"org_id"`          // Organization this permission belongs to
	CreatedAt      time.Time `json:"created_at"`      // Creation timestamp
	UpdatedAt      time.Time `json:"updated_at"`      // Last update timestamp
}

// CreatePermissionRequest represents the request payload for creating a new permission
type CreatePermissionRequest struct {
	PermissionName string `json:"permission_name" binding:"required,min=2,max=100"`
	Description    string `json:"description,omitempty"`
}

// UpdatePermissionRequest represents the request payload for updating an existing permission
type UpdatePermissionRequest struct {
	PermissionName string `json:"permission_name,omitempty" binding:"omitempty,min=2,max=100"`
	Description    string `json:"description,omitempty"`
}

// PermissionListResponse represents the response for listing permissions
type PermissionListResponse struct {
	Permissions []Permission `json:"permissions"`
	Total       int          `json:"total"`
}

// AssignPermissionRequest represents the request payload for assigning permission to role
type AssignPermissionRequest struct {
	PermissionID int64 `json:"permission_id" binding:"required"`
}

// UnassignPermissionRequest represents the request payload for unassigning permission from role
type UnassignPermissionRequest struct {
	PermissionID int64 `json:"permission_id" binding:"required"`
}