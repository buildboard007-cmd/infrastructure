package models

import (
	"time"
)

// Organization represents an organization in the system based on iam.organizations table
type Organization struct {
	OrgID         int64     `json:"org_id"`
	OrgName       string    `json:"org_name"`
	OrgType       string    `json:"org_type"`          // 'general_contractor', 'subcontractor', 'architect', 'owner', 'consultant'
	LicenseNumber string    `json:"license_number,omitempty"`
	Address       string    `json:"address,omitempty"`
	Phone         string    `json:"phone,omitempty"`
	Email         string    `json:"email,omitempty"`
	Website       string    `json:"website,omitempty"`
	Status        string    `json:"status"`            // 'active', 'inactive', 'pending_setup', 'suspended'
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// UpdateOrganizationRequest represents the request payload for updating an organization
type UpdateOrganizationRequest struct {
	OrgName       string `json:"org_name" validate:"required,min=3,max=150"`
	OrgType       string `json:"org_type,omitempty"`
	LicenseNumber string `json:"license_number,omitempty"`
	Address       string `json:"address,omitempty"`
	Phone         string `json:"phone,omitempty"`
	Email         string `json:"email,omitempty"`
	Website       string `json:"website,omitempty"`
	Status        string `json:"status,omitempty"`
}