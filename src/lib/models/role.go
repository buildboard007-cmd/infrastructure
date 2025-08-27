package models

import (
	"time"
)

// Role represents a role within an organization based on iam.role table
type Role struct {
	RoleID      int64     `json:"role_id"`      // Primary key from iam.role.role_id
	RoleName    string    `json:"role_name"`    // Role name (max 100 characters)
	Description string    `json:"description,omitempty"` // Optional role description
	OrgID       int64     `json:"org_id"`       // Organization this role belongs to
	CreatedAt   time.Time `json:"created_at"`   // Creation timestamp
	UpdatedAt   time.Time `json:"updated_at"`   // Last update timestamp
}

// CreateRoleRequest represents the request payload for creating a new role
type CreateRoleRequest struct {
	RoleName    string `json:"role_name" binding:"required,min=2,max=100"`
	Description string `json:"description,omitempty"`
}

// UpdateRoleRequest represents the request payload for updating an existing role
type UpdateRoleRequest struct {
	RoleName    string `json:"role_name,omitempty" binding:"omitempty,min=2,max=100"`
	Description string `json:"description,omitempty"`
}

// RoleListResponse represents the response for listing roles
type RoleListResponse struct {
	Roles []Role `json:"roles"`
	Total int    `json:"total"`
}

// RoleWithPermissions represents a role with its associated permissions
type RoleWithPermissions struct {
	Role
	Permissions []Permission `json:"permissions"`
}