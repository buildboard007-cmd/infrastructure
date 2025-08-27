package data

import (
	"context"
	"database/sql"
	"fmt"
	"infrastructure/lib/models"

	"github.com/sirupsen/logrus"
)

// PermissionRepository defines the interface for permission data operations
type PermissionRepository interface {
	// CreatePermission creates a new permission in the organization
	CreatePermission(ctx context.Context, orgID int64, permission *models.Permission) (*models.Permission, error)
	
	// GetPermissionsByOrg retrieves all permissions for a specific organization
	GetPermissionsByOrg(ctx context.Context, orgID int64) ([]models.Permission, error)
	
	// GetPermissionByID retrieves a specific permission by ID (with org validation)
	GetPermissionByID(ctx context.Context, permissionID, orgID int64) (*models.Permission, error)
	
	// UpdatePermission updates an existing permission
	UpdatePermission(ctx context.Context, permissionID, orgID int64, permission *models.Permission) (*models.Permission, error)
	
	// DeletePermission deletes a permission (removes role-permission assignments but keeps permission record)
	DeletePermission(ctx context.Context, permissionID, orgID int64) error
}

// PermissionDao implements PermissionRepository interface using PostgreSQL
type PermissionDao struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

// CreatePermission creates a new permission in the organization
func (dao *PermissionDao) CreatePermission(ctx context.Context, orgID int64, permission *models.Permission) (*models.Permission, error) {
	var permissionID int64
	err := dao.DB.QueryRowContext(ctx, `
		INSERT INTO iam.permission (permission_name, description, org_id)
		VALUES ($1, $2, $3)
		RETURNING permission_id, created_at, updated_at
	`, permission.PermissionName, permission.Description, orgID).Scan(
		&permissionID, &permission.CreatedAt, &permission.UpdatedAt)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"org_id":          orgID,
			"permission_name": permission.PermissionName,
			"error":           err.Error(),
		}).Error("Failed to create permission")
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	// Populate the response
	permission.PermissionID = permissionID
	permission.OrgID = orgID

	dao.Logger.WithFields(logrus.Fields{
		"permission_id":   permissionID,
		"org_id":          orgID,
		"permission_name": permission.PermissionName,
	}).Info("Successfully created permission")

	return permission, nil
}

// GetPermissionsByOrg retrieves all permissions for a specific organization
func (dao *PermissionDao) GetPermissionsByOrg(ctx context.Context, orgID int64) ([]models.Permission, error) {
	query := `
		SELECT permission_id, permission_name, description, org_id, created_at, updated_at
		FROM iam.permission
		WHERE org_id = $1
		ORDER BY permission_name ASC
	`

	rows, err := dao.DB.QueryContext(ctx, query, orgID)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"org_id": orgID,
			"error":  err.Error(),
		}).Error("Failed to query permissions")
		return nil, fmt.Errorf("failed to query permissions: %w", err)
	}
	defer rows.Close()

	var permissions []models.Permission
	for rows.Next() {
		var permission models.Permission
		err := rows.Scan(
			&permission.PermissionID,
			&permission.PermissionName,
			&permission.Description,
			&permission.OrgID,
			&permission.CreatedAt,
			&permission.UpdatedAt,
		)
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan permission row")
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	if err = rows.Err(); err != nil {
		dao.Logger.WithError(err).Error("Error iterating permission rows")
		return nil, fmt.Errorf("error iterating permissions: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"org_id": orgID,
		"count":  len(permissions),
	}).Debug("Successfully retrieved permissions for organization")

	return permissions, nil
}

// GetPermissionByID retrieves a specific permission by ID with organization validation
func (dao *PermissionDao) GetPermissionByID(ctx context.Context, permissionID, orgID int64) (*models.Permission, error) {
	var permission models.Permission
	query := `
		SELECT permission_id, permission_name, description, org_id, created_at, updated_at
		FROM iam.permission
		WHERE permission_id = $1 AND org_id = $2
	`

	err := dao.DB.QueryRowContext(ctx, query, permissionID, orgID).Scan(
		&permission.PermissionID,
		&permission.PermissionName,
		&permission.Description,
		&permission.OrgID,
		&permission.CreatedAt,
		&permission.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		dao.Logger.WithFields(logrus.Fields{
			"permission_id": permissionID,
			"org_id":        orgID,
		}).Warn("Permission not found")
		return nil, fmt.Errorf("permission not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"permission_id": permissionID,
			"org_id":        orgID,
			"error":         err.Error(),
		}).Error("Failed to get permission")
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return &permission, nil
}

// UpdatePermission updates an existing permission
func (dao *PermissionDao) UpdatePermission(ctx context.Context, permissionID, orgID int64, permission *models.Permission) (*models.Permission, error) {
	query := `
		UPDATE iam.permission 
		SET permission_name = $1, description = $2
		WHERE permission_id = $3 AND org_id = $4
		RETURNING permission_id, permission_name, description, org_id, created_at, updated_at
	`

	var updatedPermission models.Permission
	err := dao.DB.QueryRowContext(ctx, query,
		permission.PermissionName,
		permission.Description,
		permissionID,
		orgID,
	).Scan(
		&updatedPermission.PermissionID,
		&updatedPermission.PermissionName,
		&updatedPermission.Description,
		&updatedPermission.OrgID,
		&updatedPermission.CreatedAt,
		&updatedPermission.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		dao.Logger.WithFields(logrus.Fields{
			"permission_id": permissionID,
			"org_id":        orgID,
		}).Warn("Permission not found for update")
		return nil, fmt.Errorf("permission not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"permission_id": permissionID,
			"org_id":        orgID,
			"error":         err.Error(),
		}).Error("Failed to update permission")
		return nil, fmt.Errorf("failed to update permission: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"permission_id":   permissionID,
		"org_id":          orgID,
		"permission_name": updatedPermission.PermissionName,
	}).Info("Successfully updated permission")

	return &updatedPermission, nil
}

// DeletePermission removes a permission and all its role assignments
func (dao *PermissionDao) DeletePermission(ctx context.Context, permissionID, orgID int64) error {
	// Start transaction
	tx, err := dao.DB.BeginTx(ctx, nil)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to start transaction for permission deletion")
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// First, remove all role-permission assignments
	_, err = tx.ExecContext(ctx, `
		DELETE FROM iam.role_permission WHERE permission_id = $1
	`, permissionID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"permission_id": permissionID,
			"error":         err.Error(),
		}).Error("Failed to remove role-permission assignments")
		return fmt.Errorf("failed to remove permission assignments: %w", err)
	}

	// Then delete the permission (with org validation)
	result, err := tx.ExecContext(ctx, `
		DELETE FROM iam.permission WHERE permission_id = $1 AND org_id = $2
	`, permissionID, orgID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"permission_id": permissionID,
			"org_id":        orgID,
			"error":         err.Error(),
		}).Error("Failed to delete permission")
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		dao.Logger.WithFields(logrus.Fields{
			"permission_id": permissionID,
			"org_id":        orgID,
		}).Warn("Permission not found for deletion")
		return fmt.Errorf("permission not found")
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		dao.Logger.WithError(err).Error("Failed to commit permission deletion transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"permission_id": permissionID,
		"org_id":        orgID,
	}).Info("Successfully deleted permission and all assignments")

	return nil
}