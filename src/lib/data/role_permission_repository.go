package data

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sirupsen/logrus"
)

// RolePermissionRepository defines the interface for role-permission relationship operations
type RolePermissionRepository interface {
	// AssignPermissionToRole assigns a permission to a role
	AssignPermissionToRole(ctx context.Context, roleID, permissionID, orgID int64) error
	
	// UnassignPermissionFromRole removes a permission from a role
	UnassignPermissionFromRole(ctx context.Context, roleID, permissionID, orgID int64) error
	
	// IsPermissionAssignedToRole checks if a permission is assigned to a role
	IsPermissionAssignedToRole(ctx context.Context, roleID, permissionID int64) (bool, error)
}

// RolePermissionDao implements RolePermissionRepository interface using PostgreSQL
type RolePermissionDao struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

// AssignPermissionToRole assigns a permission to a role with organization validation
func (dao *RolePermissionDao) AssignPermissionToRole(ctx context.Context, roleID, permissionID, orgID int64) error {
	// Start transaction for validation and assignment
	tx, err := dao.DB.BeginTx(ctx, nil)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to start transaction for permission assignment")
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Validate that both role and permission belong to the same organization
	var roleOrgID, permissionOrgID int64
	
	// Check role exists and belongs to org
	err = tx.QueryRowContext(ctx, `
		SELECT org_id FROM iam.roles WHERE id = $1 AND org_id = $2 AND is_deleted = FALSE
	`, roleID, orgID).Scan(&roleOrgID)
	
	if err == sql.ErrNoRows {
		dao.Logger.WithFields(logrus.Fields{
			"role_id": roleID,
			"org_id":  orgID,
		}).Warn("Role not found or doesn't belong to organization")
		return fmt.Errorf("role not found")
	}
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to validate role")
		return fmt.Errorf("failed to validate role: %w", err)
	}

	// Check permission exists and belongs to org
	err = tx.QueryRowContext(ctx, `
		SELECT org_id FROM iam.permission WHERE permission_id = $1 AND org_id = $2
	`, permissionID, orgID).Scan(&permissionOrgID)
	
	if err == sql.ErrNoRows {
		dao.Logger.WithFields(logrus.Fields{
			"permission_id": permissionID,
			"org_id":        orgID,
		}).Warn("Permission not found or doesn't belong to organization")
		return fmt.Errorf("permission not found")
	}
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to validate permission")
		return fmt.Errorf("failed to validate permission: %w", err)
	}

	// Insert the role-permission assignment (ON CONFLICT DO NOTHING to handle duplicates)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO iam.role_permission (role_id, permission_id)
		VALUES ($1, $2)
		ON CONFLICT (role_id, permission_id) DO NOTHING
	`, roleID, permissionID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"role_id":       roleID,
			"permission_id": permissionID,
			"error":         err.Error(),
		}).Error("Failed to assign permission to role")
		return fmt.Errorf("failed to assign permission to role: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		dao.Logger.WithError(err).Error("Failed to commit permission assignment transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"role_id":       roleID,
		"permission_id": permissionID,
		"org_id":        orgID,
	}).Info("Successfully assigned permission to role")

	return nil
}

// UnassignPermissionFromRole removes a permission from a role with organization validation
func (dao *RolePermissionDao) UnassignPermissionFromRole(ctx context.Context, roleID, permissionID, orgID int64) error {
	// Start transaction for validation and removal
	tx, err := dao.DB.BeginTx(ctx, nil)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to start transaction for permission unassignment")
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Validate that both role and permission belong to the organization
	var count int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM iam.roles r
		JOIN iam.permission p ON p.org_id = r.org_id
		WHERE r.id = $1 AND p.permission_id = $2 AND r.org_id = $3 AND r.is_deleted = FALSE
	`, roleID, permissionID, orgID).Scan(&count)

	if err != nil {
		dao.Logger.WithError(err).Error("Failed to validate role and permission")
		return fmt.Errorf("failed to validate role and permission: %w", err)
	}

	if count == 0 {
		dao.Logger.WithFields(logrus.Fields{
			"role_id":       roleID,
			"permission_id": permissionID,
			"org_id":        orgID,
		}).Warn("Role or permission not found in organization")
		return fmt.Errorf("role or permission not found")
	}

	// Remove the role-permission assignment
	result, err := tx.ExecContext(ctx, `
		DELETE FROM iam.role_permission 
		WHERE role_id = $1 AND permission_id = $2
	`, roleID, permissionID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"role_id":       roleID,
			"permission_id": permissionID,
			"error":         err.Error(),
		}).Error("Failed to unassign permission from role")
		return fmt.Errorf("failed to unassign permission from role: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		dao.Logger.WithFields(logrus.Fields{
			"role_id":       roleID,
			"permission_id": permissionID,
		}).Warn("Permission was not assigned to role")
		return fmt.Errorf("permission not assigned to role")
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		dao.Logger.WithError(err).Error("Failed to commit permission unassignment transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"role_id":       roleID,
		"permission_id": permissionID,
		"org_id":        orgID,
	}).Info("Successfully unassigned permission from role")

	return nil
}

// IsPermissionAssignedToRole checks if a permission is assigned to a role
func (dao *RolePermissionDao) IsPermissionAssignedToRole(ctx context.Context, roleID, permissionID int64) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM iam.role_permission 
		WHERE role_id = $1 AND permission_id = $2
	`

	err := dao.DB.QueryRowContext(ctx, query, roleID, permissionID).Scan(&count)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"role_id":       roleID,
			"permission_id": permissionID,
			"error":         err.Error(),
		}).Error("Failed to check permission assignment")
		return false, fmt.Errorf("failed to check permission assignment: %w", err)
	}

	return count > 0, nil
}