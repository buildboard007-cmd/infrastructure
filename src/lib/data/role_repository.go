package data

import (
	"context"
	"database/sql"
	"fmt"
	"infrastructure/lib/models"

	"github.com/sirupsen/logrus"
)

// RoleRepository defines the interface for role data operations
type RoleRepository interface {
	// CreateRole creates a new role in the organization
	CreateRole(ctx context.Context, orgID int64, role *models.Role) (*models.Role, error)
	
	// GetRolesByOrg retrieves all roles for a specific organization
	GetRolesByOrg(ctx context.Context, orgID int64) ([]models.Role, error)
	
	// GetRoleByID retrieves a specific role by ID (with org validation)
	GetRoleByID(ctx context.Context, roleID, orgID int64) (*models.Role, error)
	
	// UpdateRole updates an existing role
	UpdateRole(ctx context.Context, roleID, orgID int64, role *models.Role) (*models.Role, error)
	
	// DeleteRole deletes a role (removes role-permission assignments but keeps role record)
	DeleteRole(ctx context.Context, roleID, orgID int64) error
	
	// GetRoleWithPermissions retrieves a role with its associated permissions
	GetRoleWithPermissions(ctx context.Context, roleID, orgID int64) (*models.RoleWithPermissions, error)
}

// RoleDao implements RoleRepository interface using PostgreSQL
type RoleDao struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

// CreateRole creates a new role in the organization
func (dao *RoleDao) CreateRole(ctx context.Context, orgID int64, role *models.Role) (*models.Role, error) {
	var roleID int64
	err := dao.DB.QueryRowContext(ctx, `
		INSERT INTO iam.roles (name, description, org_id, role_type, category, access_level, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id, created_at, updated_at
	`, role.Name, role.Description, role.OrgID, role.RoleType, role.Category, role.AccessLevel, orgID).Scan(
		&roleID, &role.CreatedAt, &role.UpdatedAt)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"org_id":    orgID,
			"role_name": role.Name,
			"error":     err.Error(),
		}).Error("Failed to create role")
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	// Populate the response
	role.ID = roleID
	if role.RoleType == "custom" {
		role.OrgID = &orgID
	}

	dao.Logger.WithFields(logrus.Fields{
		"role_id":   roleID,
		"org_id":    orgID,
		"role_name": role.Name,
	}).Info("Successfully created role")

	return role, nil
}

// GetRolesByOrg retrieves all roles for a specific organization
func (dao *RoleDao) GetRolesByOrg(ctx context.Context, orgID int64) ([]models.Role, error) {
	query := `
		SELECT id, name, description, org_id, role_type, category, access_level, created_at, updated_at
		FROM iam.roles
		WHERE (org_id = $1 OR role_type = 'standard') AND is_deleted = FALSE
		ORDER BY name ASC
	`

	rows, err := dao.DB.QueryContext(ctx, query, orgID)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"org_id": orgID,
			"error":  err.Error(),
		}).Error("Failed to query roles")
		return nil, fmt.Errorf("failed to query roles: %w", err)
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
			err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.Description,
			&role.OrgID,
			&role.RoleType,
			&role.Category,
			&role.AccessLevel,
			&role.CreatedAt,
			&role.UpdatedAt,
		)
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan role row")
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	if err = rows.Err(); err != nil {
		dao.Logger.WithError(err).Error("Error iterating role rows")
		return nil, fmt.Errorf("error iterating roles: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"org_id": orgID,
		"count":  len(roles),
	}).Debug("Successfully retrieved roles for organization")

	return roles, nil
}

// GetRoleByID retrieves a specific role by ID with organization validation
func (dao *RoleDao) GetRoleByID(ctx context.Context, roleID, orgID int64) (*models.Role, error) {
	var role models.Role
	query := `
		SELECT id, name, description, org_id, role_type, category, access_level, created_at, updated_at
		FROM iam.roles
		WHERE id = $1 AND org_id = $2 AND is_deleted = FALSE
	`

	err := dao.DB.QueryRowContext(ctx, query, roleID, orgID).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&role.OrgID,
		&role.RoleType,
		&role.Category,
		&role.AccessLevel,
		&role.CreatedAt,
		&role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		dao.Logger.WithFields(logrus.Fields{
			"role_id": roleID,
			"org_id":  orgID,
		}).Warn("Role not found")
		return nil, fmt.Errorf("role not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"role_id": roleID,
			"org_id":  orgID,
			"error":   err.Error(),
		}).Error("Failed to get role")
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return &role, nil
}

// UpdateRole updates an existing role
func (dao *RoleDao) UpdateRole(ctx context.Context, roleID, orgID int64, role *models.Role) (*models.Role, error) {
	query := `
		UPDATE iam.roles
		SET name = $1, description = $2, updated_at = NOW(), updated_by = 1
		WHERE id = $3 AND org_id = $4 AND is_deleted = FALSE
		RETURNING id, name, description, org_id, role_type, category, access_level, created_at, updated_at
	`

	var updatedRole models.Role
	err := dao.DB.QueryRowContext(ctx, query,
		role.Name,
		role.Description,
		roleID,
		orgID,
	).Scan(
		&updatedRole.ID,
		&updatedRole.Name,
		&updatedRole.Description,
		&updatedRole.OrgID,
		&updatedRole.RoleType,
		&updatedRole.Category,
		&updatedRole.AccessLevel,
		&updatedRole.CreatedAt,
		&updatedRole.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		dao.Logger.WithFields(logrus.Fields{
			"role_id": roleID,
			"org_id":  orgID,
		}).Warn("Role not found for update")
		return nil, fmt.Errorf("role not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"role_id": roleID,
			"org_id":  orgID,
			"error":   err.Error(),
		}).Error("Failed to update role")
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"role_id":   roleID,
		"org_id":    orgID,
		"role_name": updatedRole.Name,
	}).Info("Successfully updated role")

	return &updatedRole, nil
}

// DeleteRole removes a role and all its permission assignments
func (dao *RoleDao) DeleteRole(ctx context.Context, roleID, orgID int64) error {
	// Start transaction
	tx, err := dao.DB.BeginTx(ctx, nil)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to start transaction for role deletion")
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// First, remove all role-permission assignments
	_, err = tx.ExecContext(ctx, `
		DELETE FROM iam.role_permission WHERE role_id = $1
	`, roleID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"role_id": roleID,
			"error":   err.Error(),
		}).Error("Failed to remove role-permission assignments")
		return fmt.Errorf("failed to remove role assignments: %w", err)
	}

	// Then delete the role (with org validation)
	result, err := tx.ExecContext(ctx, `
		UPDATE iam.roles SET is_deleted = TRUE, updated_at = NOW(), updated_by = 1 WHERE id = $1 AND org_id = $2 AND is_deleted = FALSE
	`, roleID, orgID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"role_id": roleID,
			"org_id":  orgID,
			"error":   err.Error(),
		}).Error("Failed to delete role")
		return fmt.Errorf("failed to delete role: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		dao.Logger.WithFields(logrus.Fields{
			"role_id": roleID,
			"org_id":  orgID,
		}).Warn("Role not found for deletion")
		return fmt.Errorf("role not found")
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		dao.Logger.WithError(err).Error("Failed to commit role deletion transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"role_id": roleID,
		"org_id":  orgID,
	}).Info("Successfully deleted role and all assignments")

	return nil
}

// GetRoleWithPermissions retrieves a role with its associated permissions
func (dao *RoleDao) GetRoleWithPermissions(ctx context.Context, roleID, orgID int64) (*models.RoleWithPermissions, error) {
	// First get the role
	role, err := dao.GetRoleByID(ctx, roleID, orgID)
	if err != nil {
		return nil, err
	}

	// Then get its permissions
	query := `
		SELECT p.permission_id, p.permission_name, p.description, p.org_id, p.created_at, p.updated_at
		FROM iam.permission p
		JOIN iam.role_permission rp ON p.permission_id = rp.permission_id
		WHERE rp.role_id = $1 AND p.org_id = $2
		ORDER BY p.permission_name ASC
	`

	rows, err := dao.DB.QueryContext(ctx, query, roleID, orgID)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"role_id": roleID,
			"org_id":  orgID,
			"error":   err.Error(),
		}).Error("Failed to query role permissions")
		return nil, fmt.Errorf("failed to query role permissions: %w", err)
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

	return &models.RoleWithPermissions{
		Role:        *role,
		Permissions: permissions,
	}, nil
}