package models

import (
	"database/sql"
	"time"
)

// UserAssignment represents a user's assignment to any context (organization, project, location, etc.)
// This is the unified model that replaces separate assignment tables
type UserAssignment struct {
	ID          int64          `json:"id"`
	UserID      int64          `json:"user_id"`
	RoleID      int64          `json:"role_id"`
	ContextType string         `json:"context_type"` // "organization", "project", "location", "department", etc.
	ContextID   int64          `json:"context_id"`   // ID of the context (project ID, location ID, etc.)
	TradeType   sql.NullString `json:"trade_type,omitempty"`
	IsPrimary   bool           `json:"is_primary"`
	StartDate   sql.NullTime   `json:"start_date,omitempty"`
	EndDate     sql.NullTime   `json:"end_date,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	CreatedBy   int64          `json:"created_by"`
	UpdatedAt   time.Time      `json:"updated_at"`
	UpdatedBy   int64          `json:"updated_by"`
	IsDeleted   bool           `json:"is_deleted"`
}

// CreateAssignmentRequest represents the request to create a new assignment
type CreateAssignmentRequest struct {
	UserID      int64  `json:"user_id" binding:"required"`
	RoleID      int64  `json:"role_id" binding:"required"`
	ContextType string `json:"context_type" binding:"required,oneof=organization project location department equipment phase"`
	ContextID   int64  `json:"context_id" binding:"required"`
	TradeType   string `json:"trade_type,omitempty"`
	IsPrimary   bool   `json:"is_primary,omitempty"`
	StartDate   string `json:"start_date,omitempty"` // YYYY-MM-DD format
	EndDate     string `json:"end_date,omitempty"`   // YYYY-MM-DD format
}

// UpdateAssignmentRequest represents the request to update an existing assignment
type UpdateAssignmentRequest struct {
	RoleID      *int64 `json:"role_id,omitempty"`
	TradeType   string `json:"trade_type,omitempty"`
	IsPrimary   *bool  `json:"is_primary,omitempty"`
	StartDate   string `json:"start_date,omitempty"`
	EndDate     string `json:"end_date,omitempty"`
}

// BulkAssignmentRequest represents the request to create multiple assignments at once
type BulkAssignmentRequest struct {
	UserIDs     []int64 `json:"user_ids" binding:"required,min=1"`
	RoleID      int64   `json:"role_id" binding:"required"`
	ContextType string  `json:"context_type" binding:"required,oneof=organization project location department equipment phase"`
	ContextID   int64   `json:"context_id" binding:"required"`
	TradeType   string  `json:"trade_type,omitempty"`
	IsPrimary   bool    `json:"is_primary,omitempty"`
	StartDate   string  `json:"start_date,omitempty"`
	EndDate     string  `json:"end_date,omitempty"`
}

// AssignmentResponse represents the clean assignment response without sql.Null* types
type AssignmentResponse struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	RoleID      int64     `json:"role_id"`
	ContextType string    `json:"context_type"`
	ContextID   int64     `json:"context_id"`
	TradeType   *string   `json:"trade_type,omitempty"`
	IsPrimary   bool      `json:"is_primary"`
	StartDate   *string   `json:"start_date,omitempty"`   // YYYY-MM-DD format
	EndDate     *string   `json:"end_date,omitempty"`     // YYYY-MM-DD format
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   int64     `json:"created_by"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   int64     `json:"updated_by"`
	IsDeleted   bool      `json:"is_deleted"`

	// Enriched fields
	UserName    string `json:"user_name,omitempty"`
	UserEmail   string `json:"user_email,omitempty"`
	RoleName    string `json:"role_name,omitempty"`
	ContextName string `json:"context_name,omitempty"`
}

// AssignmentListResponse represents the response for listing assignments
type AssignmentListResponse struct {
	Assignments []AssignmentResponse `json:"assignments"`
	Total       int                  `json:"total"`
	Page        int                  `json:"page,omitempty"`
	PageSize    int                  `json:"page_size,omitempty"`
}

// UserAssignmentSummary represents a summary of all assignments for a user
type UserAssignmentSummary struct {
	UserID           int64                    `json:"user_id"`
	UserName         string                   `json:"user_name"`
	UserEmail        string                   `json:"user_email"`
	OrgID            int64                    `json:"org_id"`
	OrgName          string                   `json:"org_name"`
	TotalAssignments int                      `json:"total_assignments"`
	ActiveAssignments int                     `json:"active_assignments"`
	AssignmentsByType map[string]int          `json:"assignments_by_type"` // {"organization": 1, "project": 3, "location": 2}
	Assignments      []AssignmentResponse     `json:"assignments"`
}

// ContextAssignmentSummary represents assignments for a context (project, location, etc.)
type ContextAssignmentSummary struct {
	ContextType string               `json:"context_type"`
	ContextID   int64                `json:"context_id"`
	ContextName string               `json:"context_name"`
	OrgID       int64                `json:"org_id"`
	Assignments []AssignmentResponse `json:"assignments"`
}

// AssignmentTransferRequest represents the request to transfer assignments from one user to another
type AssignmentTransferRequest struct {
	FromUserID      int64   `json:"from_user_id" binding:"required"`
	ToUserID        int64   `json:"to_user_id" binding:"required"`
	AssignmentIDs   []int64 `json:"assignment_ids,omitempty"` // If empty, transfer all active assignments
	PreservePrimary bool    `json:"preserve_primary,omitempty"` // Whether to keep primary flags during transfer
}

// Assignment Context Types Constants
const (
	ContextTypeOrganization = "organization"
	ContextTypeProject      = "project"
	ContextTypeLocation     = "location"
	ContextTypeDepartment   = "department"
	ContextTypeEquipment    = "equipment"
	ContextTypePhase        = "phase"
)

// Assignment Query Filters
type AssignmentFilters struct {
	UserID          *int64    `json:"user_id,omitempty"`
	RoleID          *int64    `json:"role_id,omitempty"`
	ContextType     string    `json:"context_type,omitempty"`
	ContextID       *int64    `json:"context_id,omitempty"`
	OrganizationID  *int64    `json:"organization_id,omitempty"`
	IsPrimary       *bool     `json:"is_primary,omitempty"`
	IsActive        *bool     `json:"is_active,omitempty"` // Based on start/end dates
	TradeType       string    `json:"trade_type,omitempty"`
	StartDateFrom   *time.Time `json:"start_date_from,omitempty"`
	StartDateTo     *time.Time `json:"start_date_to,omitempty"`
	Page            int       `json:"page,omitempty"`
	PageSize        int       `json:"page_size,omitempty"`
	IncludeDeleted  bool      `json:"include_deleted,omitempty"`
}

// Permission Check Request - for authorization validation
type PermissionCheckRequest struct {
	UserID      int64  `json:"user_id" binding:"required"`
	ContextType string `json:"context_type" binding:"required"`
	ContextID   int64  `json:"context_id" binding:"required"`
	Permission  string `json:"permission" binding:"required"` // "read", "write", "admin", etc.
}

// Permission Check Response
type PermissionCheckResponse struct {
	HasPermission bool                 `json:"has_permission"`
	Reason        string               `json:"reason,omitempty"`
	UserRoles     []string             `json:"user_roles,omitempty"`
	InheritedFrom *AssignmentResponse  `json:"inherited_from,omitempty"` // If permission is inherited
}