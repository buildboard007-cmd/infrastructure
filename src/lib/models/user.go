package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// User represents a user in the system based on iam.users table
type User struct {
	UserID                 int64          `json:"user_id"`                             // Primary key from iam.users.id
	CognitoID              string         `json:"cognito_id"`                          // AWS Cognito sub UUID
	Email                  string         `json:"email"`                               // User's email (must match Cognito email)
	FirstName              string         `json:"first_name"`                          // User's first name
	LastName               string         `json:"last_name"`                           // User's last name
	Phone                  sql.NullString `json:"phone,omitempty"`                     // Optional contact phone number
	Mobile                 sql.NullString `json:"mobile,omitempty"`                    // Optional mobile phone number
	JobTitle               sql.NullString `json:"job_title,omitempty"`                 // Optional professional title
	EmployeeID             sql.NullString `json:"employee_id,omitempty"`               // Optional employee ID
	AvatarURL              sql.NullString `json:"avatar_url,omitempty"`                // Optional profile photo URL
	LastSelectedLocationID sql.NullInt64  `json:"last_selected_location_id,omitempty"` // User's last selected location for UI
	Status                 string         `json:"status"`                              // Account status: 'pending', 'active', 'inactive', 'suspended', 'pending_org_setup'
	IsSuperAdmin           bool           `json:"is_super_admin"`                      // SuperAdmin role flag
	OrgID                  int64          `json:"org_id"`                              // Organization this user belongs to
	CreatedAt              time.Time      `json:"created_at"`                          // Creation timestamp
	UpdatedAt              time.Time      `json:"updated_at"`                          // Last update timestamp
}

// UserWithLocationsAndRoles represents a user with their assigned locations and roles
type UserWithLocationsAndRoles struct {
	User
	LocationRoleAssignments []UserLocationRoleAssignment `json:"location_role_assignments"`
}

// UserLocationRoleAssignment represents a user's role assignment at a specific location
type UserLocationRoleAssignment struct {
	LocationID   int64  `json:"location_id"`   // Location ID
	LocationName string `json:"location_name"` // Location name for display
	RoleID       int64  `json:"role_id"`       // Role ID
	RoleName     string `json:"role_name"`     // Role name for display
}

// CreateUserRequest represents the request payload for creating a new user
type CreateUserRequest struct {
	Email                  string `json:"email" binding:"required,email"`
	FirstName              string `json:"first_name" binding:"required,min=2,max=50"`
	LastName               string `json:"last_name" binding:"required,min=2,max=50"`
	Phone                  string `json:"phone,omitempty"`
	Mobile                 string `json:"mobile,omitempty"`
	JobTitle               string `json:"job_title,omitempty"`
	EmployeeID             string `json:"employee_id,omitempty"`
	AvatarURL              string `json:"avatar_url,omitempty"`
	LastSelectedLocationID int64  `json:"last_selected_location_id,omitempty"`
	// Location and role assignments (optional for initial user creation)
	LocationRoleAssignments []LocationRoleAssignmentRequest `json:"location_role_assignments,omitempty"`
	// Note: Status is automatically set to "pending" by backend
}

// LocationRoleAssignmentRequest represents a location-role assignment in the create user request
type LocationRoleAssignmentRequest struct {
	LocationID int64 `json:"location_id" binding:"required"`
	RoleID     int64 `json:"role_id" binding:"required"`
}

// UpdateUserRequest represents the request payload for updating an existing user
type UpdateUserRequest struct {
	FirstName              string `json:"first_name,omitempty" binding:"omitempty,min=2,max=50"`
	LastName               string `json:"last_name,omitempty" binding:"omitempty,min=2,max=50"`
	Phone                  string `json:"phone,omitempty"`
	Mobile                 string `json:"mobile,omitempty"`
	JobTitle               string `json:"job_title,omitempty"`
	EmployeeID             string `json:"employee_id,omitempty"`
	AvatarURL              string `json:"avatar_url,omitempty"`
	LastSelectedLocationID int64  `json:"last_selected_location_id,omitempty"`
	Status                 string `json:"status,omitempty" binding:"omitempty,oneof=pending active inactive suspended"`
	// Location and role assignments (required - will replace ALL existing assignments)
	LocationRoleAssignments []LocationRoleAssignmentRequest `json:"location_role_assignments" binding:"required"`
}

// UpdateUserStatusRequest represents the request payload for updating user status
type UpdateUserStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active inactive suspended"`
}

// UserListResponse represents the response for listing users
type UserListResponse struct {
	Users []UserWithLocationsAndRoles `json:"users"`
	Total int                         `json:"total"`
}

// CreateUserResponse represents the response after creating a user
type CreateUserResponse struct {
	UserWithLocationsAndRoles
	TemporaryPassword string `json:"temporary_password"` // Temporary password sent via email
	Message           string `json:"message"`
}

// User status constants
const (
	UserStatusPending   = "pending"
	UserStatusActive    = "active"
	UserStatusInactive  = "inactive"
	UserStatusSuspended = "suspended"
)

// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	return u.FirstName + " " + u.LastName
}

// IsActive returns true if user status is active
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// CanLogin returns true if user can login (active or pending status)
func (u *User) CanLogin() bool {
	return u.Status == UserStatusActive || u.Status == UserStatusPending
}

// MarshalJSON provides custom JSON serialization for User struct
func (u *User) MarshalJSON() ([]byte, error) {
	type UserJSON struct {
		UserID                 int64     `json:"user_id"`
		CognitoID              string    `json:"cognito_id"`
		Email                  string    `json:"email"`
		FirstName              string    `json:"first_name"`
		LastName               string    `json:"last_name"`
		Phone                  *string   `json:"phone"`
		Mobile                 *string   `json:"mobile"`
		JobTitle               *string   `json:"job_title"`
		EmployeeID             *string   `json:"employee_id"`
		AvatarURL              *string   `json:"avatar_url"`
		LastSelectedLocationID *int64    `json:"last_selected_location_id"`
		Status                 string    `json:"status"`
		IsSuperAdmin           bool      `json:"is_super_admin"`
		OrgID                  int64     `json:"org_id"`
		CreatedAt              time.Time `json:"created_at"`
		UpdatedAt              time.Time `json:"updated_at"`
	}

	userJSON := UserJSON{
		UserID:       u.UserID,
		CognitoID:    u.CognitoID,
		Email:        u.Email,
		FirstName:    u.FirstName,
		LastName:     u.LastName,
		Status:       u.Status,
		IsSuperAdmin: u.IsSuperAdmin,
		OrgID:        u.OrgID,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}

	if u.Phone.Valid {
		userJSON.Phone = &u.Phone.String
	}
	if u.Mobile.Valid {
		userJSON.Mobile = &u.Mobile.String
	}
	if u.JobTitle.Valid {
		userJSON.JobTitle = &u.JobTitle.String
	}
	if u.EmployeeID.Valid {
		userJSON.EmployeeID = &u.EmployeeID.String
	}
	if u.AvatarURL.Valid {
		userJSON.AvatarURL = &u.AvatarURL.String
	}
	if u.LastSelectedLocationID.Valid {
		userJSON.LastSelectedLocationID = &u.LastSelectedLocationID.Int64
	}

	return json.Marshal(userJSON)
}
