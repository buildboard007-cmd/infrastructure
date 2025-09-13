package data

import (
	"context"
	"database/sql"
	"fmt"
	"infrastructure/lib/models"
	"strings"

	"github.com/sirupsen/logrus"
)

// OrgRepository defines the interface for organization data operations
type OrgRepository interface {
	CreateOrganization(ctx context.Context, userID int64, org *models.Organization) (*models.Organization, error)
	UpdateOrganization(ctx context.Context, userID int64, orgID int64, updateReq *models.UpdateOrganizationRequest) (*models.Organization, error)
	GetOrganizationByUserID(ctx context.Context, userID int64) (*models.Organization, error)
	GetOrganizationByID(ctx context.Context, orgID int64) (*models.Organization, error)
	DeleteOrganization(ctx context.Context, orgID int64, userID int64) error
}

// OrgDao implements the OrgRepository interface for PostgreSQL
type OrgDao struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

// CreateOrganization creates a new organization
func (dao *OrgDao) CreateOrganization(ctx context.Context, userID int64, org *models.Organization) (*models.Organization, error) {
	// Set defaults
	orgType := org.OrgType
	if orgType == "" {
		orgType = "general_contractor"
	}
	
	status := org.Status
	if status == "" {
		status = "pending_setup"
	}

	// Create the organization
	var orgID int64
	err := dao.DB.QueryRowContext(ctx, `
		INSERT INTO iam.organizations (
			name, org_type, license_number, address, phone, email, website, status,
			created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`, org.Name, orgType, org.LicenseNumber, org.Address, org.Phone, org.Email, 
		org.Website, status, userID, userID).Scan(
		&orgID, &org.CreatedAt, &org.UpdatedAt)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"user_id": userID,
			"name":    org.Name,
			"error":   err.Error(),
		}).Error("Failed to create organization")
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	// Populate the response
	org.ID = orgID
	org.CreatedBy = userID
	org.UpdatedBy = userID

	dao.Logger.WithFields(logrus.Fields{
		"org_id":   orgID,
		"user_id":  userID,
		"name":     org.Name,
	}).Info("Successfully created organization")

	return org, nil
}

// UpdateOrganization updates an existing organization
func (dao *OrgDao) UpdateOrganization(ctx context.Context, userID int64, orgID int64, updateReq *models.UpdateOrganizationRequest) (*models.Organization, error) {
	// Build dynamic update query based on provided fields
	setParts := []string{"updated_by = $1", "updated_at = CURRENT_TIMESTAMP"}
	args := []interface{}{userID}
	argIndex := 2
	
	if updateReq.Name != "" {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, updateReq.Name)
		argIndex++
	}
	if updateReq.OrgType != "" {
		setParts = append(setParts, fmt.Sprintf("org_type = $%d", argIndex))
		args = append(args, updateReq.OrgType)
		argIndex++
	}
	if updateReq.LicenseNumber != "" {
		setParts = append(setParts, fmt.Sprintf("license_number = $%d", argIndex))
		args = append(args, updateReq.LicenseNumber)
		argIndex++
	}
	if updateReq.Address != "" {
		setParts = append(setParts, fmt.Sprintf("address = $%d", argIndex))
		args = append(args, updateReq.Address)
		argIndex++
	}
	if updateReq.Phone != "" {
		setParts = append(setParts, fmt.Sprintf("phone = $%d", argIndex))
		args = append(args, updateReq.Phone)
		argIndex++
	}
	if updateReq.Email != "" {
		setParts = append(setParts, fmt.Sprintf("email = $%d", argIndex))
		args = append(args, updateReq.Email)
		argIndex++
	}
	if updateReq.Website != "" {
		setParts = append(setParts, fmt.Sprintf("website = $%d", argIndex))
		args = append(args, updateReq.Website)
		argIndex++
	}
	if updateReq.Status != "" {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, updateReq.Status)
		argIndex++
	}
	
	// Add WHERE conditions
	args = append(args, orgID)
	
	query := fmt.Sprintf(`
		UPDATE iam.organizations 
		SET %s
		WHERE id = $%d AND is_deleted = FALSE
		RETURNING id, name, org_type, license_number, address, phone, email, website, 
		          status, created_at, created_by, updated_at, updated_by
	`, strings.Join(setParts, ", "), argIndex)

	var updatedOrg models.Organization
	err := dao.DB.QueryRowContext(ctx, query, args...).Scan(
		&updatedOrg.ID,
		&updatedOrg.Name,
		&updatedOrg.OrgType,
		&updatedOrg.LicenseNumber,
		&updatedOrg.Address,
		&updatedOrg.Phone,
		&updatedOrg.Email,
		&updatedOrg.Website,
		&updatedOrg.Status,
		&updatedOrg.CreatedAt,
		&updatedOrg.CreatedBy,
		&updatedOrg.UpdatedAt,
		&updatedOrg.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		dao.Logger.WithFields(logrus.Fields{
			"org_id":  orgID,
			"user_id": userID,
		}).Warn("Organization not found for update")
		return nil, fmt.Errorf("organization not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"org_id":  orgID,
			"user_id": userID,
			"error":   err.Error(),
		}).Error("Failed to update organization")
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	// Check if this is the first organization update and if user has at least one location
	// If so, update user status from pending_org_setup to active
	err = dao.checkAndUpdateUserStatus(ctx, userID, orgID)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"org_id":  orgID,
			"user_id": userID,
			"warning": err.Error(),
		}).Warn("Failed to check/update user status after org update")
		// Don't return error as org update was successful
	}

	dao.Logger.WithFields(logrus.Fields{
		"org_id":   orgID,
		"org_name": updatedOrg.Name,
		"user_id":  userID,
	}).Info("Successfully updated organization")

	return &updatedOrg, nil
}

// GetOrganizationByUserID retrieves an organization by user ID
func (dao *OrgDao) GetOrganizationByUserID(ctx context.Context, userID int64) (*models.Organization, error) {
	query := `
		SELECT o.id, o.name, o.org_type, o.license_number, o.address, o.phone, o.email, o.website,
		       o.status, o.created_at, o.created_by, o.updated_at, o.updated_by
		FROM iam.organizations o
		INNER JOIN iam.users u ON u.org_id = o.id
		WHERE u.id = $1 AND o.is_deleted = FALSE
	`
	
	var org models.Organization
	err := dao.DB.QueryRowContext(ctx, query, userID).Scan(
		&org.ID,
		&org.Name,
		&org.OrgType,
		&org.LicenseNumber,
		&org.Address,
		&org.Phone,
		&org.Email,
		&org.Website,
		&org.Status,
		&org.CreatedAt,
		&org.CreatedBy,
		&org.UpdatedAt,
		&org.UpdatedBy,
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

// GetOrganizationByID retrieves a specific organization by ID
func (dao *OrgDao) GetOrganizationByID(ctx context.Context, orgID int64) (*models.Organization, error) {
	var org models.Organization
	query := `
		SELECT id, name, org_type, license_number, address, phone, email, website,
		       status, created_at, created_by, updated_at, updated_by
		FROM iam.organizations
		WHERE id = $1 AND is_deleted = FALSE
	`

	err := dao.DB.QueryRowContext(ctx, query, orgID).Scan(
		&org.ID,
		&org.Name,
		&org.OrgType,
		&org.LicenseNumber,
		&org.Address,
		&org.Phone,
		&org.Email,
		&org.Website,
		&org.Status,
		&org.CreatedAt,
		&org.CreatedBy,
		&org.UpdatedAt,
		&org.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		dao.Logger.WithFields(logrus.Fields{
			"org_id": orgID,
		}).Warn("Organization not found")
		return nil, fmt.Errorf("organization not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"org_id": orgID,
			"error":  err.Error(),
		}).Error("Failed to get organization")
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return &org, nil
}

// DeleteOrganization soft deletes an organization
func (dao *OrgDao) DeleteOrganization(ctx context.Context, orgID int64, userID int64) error {
	// Soft delete the organization
	result, err := dao.DB.ExecContext(ctx, `
		UPDATE iam.organizations 
		SET is_deleted = TRUE, updated_by = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND is_deleted = FALSE
	`, userID, orgID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"org_id":  orgID,
			"user_id": userID,
			"error":   err.Error(),
		}).Error("Failed to delete organization")
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		dao.Logger.WithFields(logrus.Fields{
			"org_id":  orgID,
			"user_id": userID,
		}).Warn("Organization not found for deletion")
		return fmt.Errorf("organization not found")
	}

	dao.Logger.WithFields(logrus.Fields{
		"org_id":  orgID,
		"user_id": userID,
	}).Info("Successfully soft deleted organization")

	return nil
}

// checkAndUpdateUserStatus checks if user should be activated after organization setup
// Activates both user and organization immediately upon organization setup completion
func (dao *OrgDao) checkAndUpdateUserStatus(ctx context.Context, userID, orgID int64) error {
	// Check if user is still in pending_org_setup status
	var userStatus string

	query := `
		SELECT u.status
		FROM iam.users u
		WHERE u.id = $1 AND u.org_id = $2
	`

	err := dao.DB.QueryRowContext(ctx, query, userID, orgID).Scan(&userStatus)
	if err != nil {
		return fmt.Errorf("failed to check user status: %w", err)
	}

	// If user is pending_org_setup, activate both user and organization immediately
	// This happens as soon as organization setup is completed
	if userStatus == "pending_org_setup" {
		// Start transaction to update both user and organization atomically
		tx, err := dao.DB.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to start transaction for activation: %w", err)
		}
		defer tx.Rollback()

		// Update user status to active
		_, err = tx.ExecContext(ctx, `
			UPDATE iam.users
			SET status = 'active', updated_by = $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = $2 AND status = 'pending_org_setup'
		`, userID, userID)

		if err != nil {
			return fmt.Errorf("failed to update user status to active: %w", err)
		}

		// Update organization status to active
		_, err = tx.ExecContext(ctx, `
			UPDATE iam.organizations
			SET status = 'active', updated_by = $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = $2 AND status = 'pending'
		`, userID, orgID)

		if err != nil {
			return fmt.Errorf("failed to update organization status to active: %w", err)
		}

		// Commit transaction
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit activation transaction: %w", err)
		}

		dao.Logger.WithFields(logrus.Fields{
			"user_id": userID,
			"org_id":  orgID,
		}).Info("User and organization status updated to active immediately after organization setup completion")
	}

	return nil
}