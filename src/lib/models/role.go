package models

import (
	"time"
)

// Role represents a role within an organization based on iam.roles table
type Role struct {
	ID                       int64     `json:"id"`                                  // Primary key from iam.roles.id
	OrgID                    *int64    `json:"org_id,omitempty"`                    // Organization ID (NULL for standard/system roles)
	Name                     string    `json:"name"`                                // Role name (max 100 characters)
	Description              *string   `json:"description,omitempty"`               // Optional role description
	RoleType                 string    `json:"role_type"`                           // 'system' (across all orgs) or 'custom' (specific to org)
	Category                 string    `json:"category"`                            // 'management', 'field', 'office', 'external', 'admin'
	AccessLevel              string    `json:"access_level"`                        // 'organization', 'location', 'project'
	CreatedAt                time.Time `json:"created_at"`                          // Creation timestamp
	CreatedBy                int64     `json:"created_by"`                          // User who created
	UpdatedAt                time.Time `json:"updated_at"`                          // Last update timestamp
	UpdatedBy                int64     `json:"updated_by"`                          // User who updated
	IsDeleted                bool      `json:"is_deleted"`                          // Soft delete flag
}

// RoleRequest represents the unified request payload for creating/updating roles
type RoleRequest struct {
	Name                     string `json:"name" binding:"required,min=2,max=100"`
	Description              string `json:"description,omitempty"`
	RoleType                 string `json:"role_type,omitempty"`                         // 'standard' or 'custom'
	Category                 string `json:"category" binding:"required"`                   // 'management', 'field', 'office', 'external', 'admin'
	AccessLevel              string `json:"access_level,omitempty"`                      // 'organization', 'location', 'project'
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