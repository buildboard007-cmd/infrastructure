// Package models defines the data structures used throughout the IAM system.
// These models map directly to the PostgreSQL IAM schema and are used for:
// 1. Database queries and result mapping
// 2. JWT token generation (via Token Customizer Lambda)
// 3. API responses and inter-service communication
//
// All models use JSON tags for serialization and db tags for database mapping.
package models

import "database/sql"

// LocationRole represents a user's role at a specific location within an organization.
// Roles define the level of access and responsibilities a user has.
// Examples: 'admin', 'manager', 'employee'
//
// Database mapping: iam.role table
type LocationRole struct {
	RoleID      int64  `json:"role_id"`                    // Primary key from iam.role.role_id
	RoleName    string `json:"role_name"`                  // Human-readable role name (unique across system)
	Description string `json:"description,omitempty"`     // Optional detailed description of role responsibilities
}

// UserLocation represents a physical or logical location within an organization.
// This structure contains only location information - roles are handled separately per-project.
// Examples: 'New York Office', 'Remote Team', 'Manufacturing Plant'
//
// Database mapping: iam.location table
type UserLocation struct {
	ID           int64  `json:"id"`                        // Primary key from iam.location.id
	Name         string `json:"name"`                      // Human-readable location name
	LocationType string `json:"location_type"`             // Location type (office, warehouse, job_site, yard)
	Address      string `json:"address,omitempty"`         // Optional physical address
}

// UserProfile represents the complete user profile aggregated from the iam.user_summary view.
// This is the primary data structure used throughout the system for user information.
//
// Key relationships:
// - One user belongs to ONE organization (org_id)
// - One user can work at MULTIPLE locations with DIFFERENT roles at each location
// - Cognito integration via cognito_id (maps to Cognito 'sub' claim)
//
// Usage:
// - Token Customizer Lambda: Adds this data to JWT tokens
// - API services: User authentication and authorization
// - Frontend: User profile display and role-based UI
//
// Database mapping: iam.user_summary view (aggregates users, organization, locations, roles)
type UserProfile struct {
	// Core Identity
	UserID    sql.NullString `json:"user_id" db:"user_id"`         // Internal user ID (auto-incrementing)
	CognitoID sql.NullString `json:"cognito_id" db:"cognito_id"`   // AWS Cognito sub UUID (unique identifier)
	Email     sql.NullString `json:"email" db:"email"`             // User's email (must match Cognito email)

	// Personal Information
	FirstName sql.NullString `json:"first_name" db:"first_name"`   // User's first name
	LastName  sql.NullString `json:"last_name" db:"last_name"`     // User's last name
	Phone     sql.NullString `json:"phone" db:"phone"`             // Optional contact phone number
	JobTitle  sql.NullString `json:"job_title" db:"job_title"`     // Optional professional title
	AvatarURL sql.NullString `json:"avatar_url" db:"avatar_url"`   // Optional profile photo URL

	// Account Status
	Status sql.NullString `json:"status" db:"status"`             // Account status: 'active', 'inactive', 'suspended'
	
	// Role Information
	IsSuperAdmin bool `json:"is_super_admin" db:"is_super_admin"` // SuperAdmin role flag
	
	// Organizational Context
	OrgID   sql.NullString `json:"org_id" db:"org_id"`           // Organization ID this user belongs to
	OrgName sql.NullString `json:"org_name" db:"org_name"`       // Organization name for display
	
	// Location Context
	LastSelectedLocationID sql.NullString   `json:"last_selected_location_id" db:"last_selected_location_id"` // User's last selected location for UI
	Locations         []UserLocation `json:"locations" db:"locations"`                      // All locations and roles for this user
}

// GetFullName returns the user's full name as "FirstName LastName"
func (u *UserProfile) GetFullName() string {
	firstName := ""
	if u.FirstName.Valid {
		firstName = u.FirstName.String
	}
	lastName := ""
	if u.LastName.Valid {
		lastName = u.LastName.String
	}
	if firstName != "" && lastName != "" {
		return firstName + " " + lastName
	} else if firstName != "" {
		return firstName
	} else if lastName != "" {
		return lastName
	}
	return ""
}

// GetAccessibleLocationIDs returns all location IDs this user can access
func (u *UserProfile) GetAccessibleLocationIDs() []int64 {
	locationIDs := make([]int64, 0, len(u.Locations))
	for _, location := range u.Locations {
		locationIDs = append(locationIDs, location.ID)
	}
	return locationIDs
}

// HasLocationAccess checks if the user has access to a specific location
func (u *UserProfile) HasLocationAccess(locationID int64) bool {
	for _, location := range u.Locations {
		if location.ID == locationID {
			return true
		}
	}
	return false
}

// GetLocationByID returns the location details if user has access to it
func (u *UserProfile) GetLocationByID(locationID int64) *UserLocation {
	for _, location := range u.Locations {
		if location.ID == locationID {
			return &location
		}
	}
	return nil
}
