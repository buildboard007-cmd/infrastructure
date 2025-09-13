package models

import (
	"database/sql"
	"time"
)

// Organization represents an organization in the system based on iam.organizations table
type Organization struct {
	ID            int64          `json:"id"`       // Unique organization identifier
	Name          string         `json:"name"`     // Organization name (matches schema: name)
	OrgType       string         `json:"org_type"` // 'general_contractor', 'subcontractor', 'architect', 'owner', 'consultant'
	LicenseNumber sql.NullString `json:"license_number,omitempty"`
	Address       sql.NullString `json:"address,omitempty"`
	Phone         sql.NullString `json:"phone,omitempty"`
	Email         sql.NullString `json:"email,omitempty"`
	Website       sql.NullString `json:"website,omitempty"`
	Status        string         `json:"status"` // 'active', 'inactive', 'pending_setup', 'suspended'
	CreatedAt     time.Time      `json:"created_at"`
	CreatedBy     int64          `json:"created_by"` // User who created this organization
	UpdatedAt     time.Time      `json:"updated_at"`
	UpdatedBy     int64          `json:"updated_by"` // User who last updated this organization
}

// CreateOrganizationRequest represents the request payload for creating a new organization
type CreateOrganizationRequest struct {
	Name          string `json:"name" binding:"required,min=3,max=255"`
	OrgType       string `json:"org_type" binding:"required,oneof=general_contractor subcontractor architect owner consultant"`
	LicenseNumber string `json:"license_number,omitempty" binding:"omitempty,max=100"`
	Address       string `json:"address,omitempty"`
	Phone         string `json:"phone,omitempty" binding:"omitempty,max=20"`
	Email         string `json:"email,omitempty" binding:"omitempty,email,max=255"`
	Website       string `json:"website,omitempty" binding:"omitempty,url,max=255"`
	Status        string `json:"status,omitempty" binding:"omitempty,oneof=active inactive pending_setup suspended"`
}

// UpdateOrganizationRequest represents the request payload for updating an organization
type UpdateOrganizationRequest struct {
	Name          string `json:"name,omitempty" binding:"omitempty,min=3,max=255"`
	OrgType       string `json:"org_type,omitempty" binding:"omitempty,oneof=general_contractor subcontractor architect owner consultant"`
	LicenseNumber string `json:"license_number,omitempty" binding:"omitempty,max=100"`
	Address       string `json:"address,omitempty"`
	Phone         string `json:"phone,omitempty" binding:"omitempty,max=20"`
	Email         string `json:"email,omitempty" binding:"omitempty,email,max=255"`
	Website       string `json:"website,omitempty" binding:"omitempty,url,max=255"`
	Status        string `json:"status,omitempty" binding:"omitempty,oneof=active inactive pending_setup suspended"`
}
