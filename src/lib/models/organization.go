package models

import (
	"database/sql"
	"time"
)

// Organization represents an organization in the system
type Organization struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Description  sql.NullString `json:"description,omitempty"`
	Status       string         `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	AdminUserID  sql.NullString `json:"admin_user_id,omitempty"`
	AdminEmail   sql.NullString `json:"admin_email,omitempty"`
}

// CreateOrganizationRequest represents the request payload for creating an organization
type CreateOrganizationRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=100"`
	Description string `json:"description,omitempty" validate:"max=500"`
	AdminEmail  string `json:"admin_email" validate:"required,email"`
}

// UpdateAdminProfileRequest represents the request payload for updating admin profile
type UpdateAdminProfileRequest struct {
	FirstName string `json:"first_name,omitempty" validate:"max=50"`
	LastName  string `json:"last_name,omitempty" validate:"max=50"`
	Phone     string `json:"phone,omitempty" validate:"max=20"`
	JobTitle  string `json:"job_title,omitempty" validate:"max=100"`
}