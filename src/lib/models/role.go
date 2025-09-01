package models

import (
	"time"
)

// Role represents a role within an organization based on iam.roles table
type Role struct {
	RoleID                     int64     `json:"role_id"`                              // Primary key from iam.roles.id
	RoleName                   string    `json:"role_name"`                            // Role name (max 100 characters)
	Description                string    `json:"description,omitempty"`                // Optional role description
	OrgID                      int64     `json:"org_id"`                               // Organization this role belongs to (NULL for system roles)
	RoleType                   string    `json:"role_type"`                            // 'system' or 'custom'
	ConstructionRoleCategory   string    `json:"construction_role_category"`           // 'management', 'field', 'office', 'external', 'admin'
	AccessLevel                string    `json:"access_level"`                         // 'organization', 'location', 'project'
	CreatedAt                  time.Time `json:"created_at"`                           // Creation timestamp
	UpdatedAt                  time.Time `json:"updated_at"`                           // Last update timestamp
}

// CreateRoleRequest represents the request payload for creating a new role
type CreateRoleRequest struct {
	RoleName                   string `json:"role_name" binding:"required,min=2,max=100"`
	Description                string `json:"description,omitempty"`
	RoleType                   string `json:"role_type,omitempty"`
	ConstructionRoleCategory   string `json:"construction_role_category" binding:"required"`
	AccessLevel                string `json:"access_level,omitempty"`
}

// UpdateRoleRequest represents the request payload for updating an existing role
type UpdateRoleRequest struct {
	RoleName                   string `json:"role_name,omitempty" binding:"omitempty,min=2,max=100"`
	Description                string `json:"description,omitempty"`
	RoleType                   string `json:"role_type,omitempty"`
	ConstructionRoleCategory   string `json:"construction_role_category,omitempty"`
	AccessLevel                string `json:"access_level,omitempty"`
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