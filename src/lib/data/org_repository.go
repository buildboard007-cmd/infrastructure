package data

import (
	"context"
	"database/sql"
	"fmt"
	"infrastructure/lib/models"

	"github.com/sirupsen/logrus"
)

// OrgRepository defines the interface for organization data operations
type OrgRepository interface {
	UpdateOrganization(ctx context.Context, userID int64, org *models.Organization) (*models.Organization, error)
	GetOrganizationByUserID(ctx context.Context, userID int64) (*models.Organization, error)
}

// OrgDao implements the OrgRepository interface for PostgreSQL
type OrgDao struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

// UpdateOrganization updates the organization info for a user
// This is used when a super admin updates their org info from the default "system" name
func (dao *OrgDao) UpdateOrganization(ctx context.Context, userID int64, org *models.Organization) (*models.Organization, error) {
	// First, verify the user is a super admin and get their org_id
	var orgID int64
	var isSuperAdmin bool
	
	checkQuery := `
		SELECT org_id, is_super_admin
		FROM iam.users
		WHERE id = $1
	`
	
	err := dao.DB.QueryRowContext(ctx, checkQuery, userID).Scan(&orgID, &isSuperAdmin)
	if err != nil {
		if err == sql.ErrNoRows {
			dao.Logger.WithFields(logrus.Fields{
				"operation": "UpdateOrganization",
				"user_id":   userID,
			}).Warn("User not found")
			return nil, fmt.Errorf("user not found")
		}
		dao.Logger.WithFields(logrus.Fields{
			"operation": "UpdateOrganization",
			"user_id":   userID,
			"error":     err.Error(),
		}).Error("Failed to check user")
		return nil, fmt.Errorf("failed to check user: %w", err)
	}
	
	// Verify user is a super admin
	if !isSuperAdmin {
		dao.Logger.WithFields(logrus.Fields{
			"operation": "UpdateOrganization",
			"user_id":   userID,
		}).Warn("User is not a super admin")
		return nil, fmt.Errorf("unauthorized: user is not a super admin")
	}
	
	// Update the organization
	updateQuery := `
		UPDATE iam.organizations
		SET name = $1, updated_by = $2
		WHERE id = $3
		RETURNING id, name, created_at, updated_at
	`
	
	var updatedOrg models.Organization
	err = dao.DB.QueryRowContext(
		ctx,
		updateQuery,
		org.OrgName,
		userID,
		orgID,
	).Scan(
		&updatedOrg.OrgID,
		&updatedOrg.OrgName,
		&updatedOrg.CreatedAt,
		&updatedOrg.UpdatedAt,
	)
	
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"operation": "UpdateOrganization",
			"org_id":    orgID,
			"error":     err.Error(),
		}).Error("Failed to update organization")
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}
	
	// Also update user status from pending_org_setup to active if it was pending
	statusUpdateQuery := `
		UPDATE iam.users
		SET status = 'active', updated_by = $1
		WHERE id = $2 AND status = 'pending_org_setup'
	`
	
	_, err = dao.DB.ExecContext(ctx, statusUpdateQuery, userID, userID)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"operation": "UpdateOrganization",
			"user_id":   userID,
			"warning":   err.Error(),
		}).Warn("Failed to update user status")
		// Don't return error as org update was successful
	}
	
	dao.Logger.WithFields(logrus.Fields{
		"operation": "UpdateOrganization",
		"org_id":    updatedOrg.OrgID,
		"org_name":  updatedOrg.OrgName,
		"user_id":   userID,
	}).Info("Organization updated successfully")
	
	return &updatedOrg, nil
}

// GetOrganizationByUserID retrieves an organization by user ID
func (dao *OrgDao) GetOrganizationByUserID(ctx context.Context, userID int64) (*models.Organization, error) {
	query := `
		SELECT o.id, o.name, o.created_at, o.updated_at
		FROM iam.organizations o
		INNER JOIN iam.users u ON u.org_id = o.id
		WHERE u.id = $1
	`
	
	var org models.Organization
	err := dao.DB.QueryRowContext(ctx, query, userID).Scan(
		&org.OrgID,
		&org.OrgName,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			dao.Logger.WithFields(logrus.Fields{
				"operation": "GetOrganizationByUserID",
				"user_id":   userID,
			}).Warn("Organization not found for user")
			return nil, fmt.Errorf("organization not found for user")
		}
		dao.Logger.WithFields(logrus.Fields{
			"operation": "GetOrganizationByUserID",
			"user_id":   userID,
			"error":     err.Error(),
		}).Error("Failed to get organization by user ID")
		return nil, fmt.Errorf("failed to get organization by user ID: %w", err)
	}
	
	return &org, nil
}