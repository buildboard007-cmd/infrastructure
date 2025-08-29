package data

import (
	"context"
	"database/sql"
	"fmt"
	"infrastructure/lib/models"

	"github.com/sirupsen/logrus"
)

// UserLocationRoleRepository defines the interface for user-location-role assignment operations
type UserLocationRoleRepository interface {
	// UpdateUserLocationRoleAssignments replaces all assignments for a user
	UpdateUserLocationRoleAssignments(ctx context.Context, userID, orgID int64, assignments []models.LocationRoleAssignmentRequest) error

	// GetUserAssignments retrieves all location-role assignments for a user
	GetUserAssignments(ctx context.Context, userID int64) ([]models.UserLocationRoleAssignment, error)
}

// UserLocationRoleDao implements UserLocationRoleRepository interface using PostgreSQL
type UserLocationRoleDao struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

// UpdateUserLocationRoleAssignments replaces all assignments for a user
func (dao *UserLocationRoleDao) UpdateUserLocationRoleAssignments(ctx context.Context, userID, orgID int64, assignments []models.LocationRoleAssignmentRequest) error {
	// Start transaction
	tx, err := dao.DB.BeginTx(ctx, nil)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to start transaction for updating user assignments")
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// First, validate that the user belongs to the organization
	var userOrgID int64
	err = tx.QueryRowContext(ctx, `
		SELECT org_id FROM iam.user WHERE user_id = $1
	`, userID).Scan(&userOrgID)

	if err == sql.ErrNoRows {
		dao.Logger.WithField("user_id", userID).Warn("User not found for assignment update")
		return fmt.Errorf("user not found")
	}

	if err != nil {
		dao.Logger.WithError(err).Error("Failed to validate user")
		return fmt.Errorf("failed to validate user: %w", err)
	}

	if userOrgID != orgID {
		dao.Logger.WithFields(logrus.Fields{
			"user_id":      userID,
			"user_org_id":  userOrgID,
			"expected_org": orgID,
		}).Warn("User does not belong to the specified organization")
		return fmt.Errorf("user not found in organization")
	}

	// Validate all new assignments
	for _, assignment := range assignments {
		err = dao.validateLocationRoleInTransaction(ctx, tx, assignment.LocationID, assignment.RoleID, orgID)
		if err != nil {
			return fmt.Errorf("invalid assignment (location_id: %d, role_id: %d): %w",
				assignment.LocationID, assignment.RoleID, err)
		}
	}

	// Remove all existing assignments for this user
	_, err = tx.ExecContext(ctx, `
		DELETE FROM iam.user_location_role WHERE user_id = $1
	`, userID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"user_id": userID,
			"error":   err.Error(),
		}).Error("Failed to remove existing user assignments")
		return fmt.Errorf("failed to remove existing assignments: %w", err)
	}

	// Add new assignments
	for _, assignment := range assignments {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO iam.user_location_role (user_id, location_id, role_id)
			VALUES ($1, $2, $3)
		`, userID, assignment.LocationID, assignment.RoleID)

		if err != nil {
			dao.Logger.WithFields(logrus.Fields{
				"user_id":     userID,
				"location_id": assignment.LocationID,
				"role_id":     assignment.RoleID,
				"error":       err.Error(),
			}).Error("Failed to create new user assignment")
			return fmt.Errorf("failed to create assignment: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		dao.Logger.WithError(err).Error("Failed to commit user assignment update transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"user_id":           userID,
		"org_id":            orgID,
		"assignments_count": len(assignments),
	}).Info("Successfully updated user location-role assignments")

	return nil
}

// GetUserAssignments retrieves all location-role assignments for a user
func (dao *UserLocationRoleDao) GetUserAssignments(ctx context.Context, userID int64) ([]models.UserLocationRoleAssignment, error) {
	query := `
		SELECT ulr.location_id, l.location_name, ulr.role_id, r.role_name
		FROM iam.user_location_role ulr
		JOIN iam.location l ON ulr.location_id = l.location_id
		JOIN iam.role r ON ulr.role_id = r.role_id
		WHERE ulr.user_id = $1
		ORDER BY l.location_name, r.role_name
	`

	rows, err := dao.DB.QueryContext(ctx, query, userID)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"user_id": userID,
			"error":   err.Error(),
		}).Error("Failed to query user assignments")
		return nil, fmt.Errorf("failed to query user assignments: %w", err)
	}
	defer rows.Close()

	var assignments []models.UserLocationRoleAssignment
	for rows.Next() {
		var assignment models.UserLocationRoleAssignment
		err := rows.Scan(
			&assignment.LocationID, &assignment.LocationName,
			&assignment.RoleID, &assignment.RoleName,
		)
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan assignment row")
			return nil, fmt.Errorf("failed to scan assignment: %w", err)
		}
		assignments = append(assignments, assignment)
	}

	if err = rows.Err(); err != nil {
		dao.Logger.WithError(err).Error("Error iterating assignment rows")
		return nil, fmt.Errorf("error iterating assignments: %w", err)
	}

	return assignments, nil
}


// validateLocationRoleInTransaction validates location and role in a transaction context
func (dao *UserLocationRoleDao) validateLocationRoleInTransaction(ctx context.Context, db interface{}, locationID, roleID, orgID int64) error {
	var count int
	var err error

	// Handle both *sql.DB and *sql.Tx
	switch d := db.(type) {
	case *sql.DB:
		err = d.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM iam.location l, iam.role r
			WHERE l.location_id = $1 AND r.role_id = $2
			AND l.org_id = $3 AND r.org_id = $3
		`, locationID, roleID, orgID).Scan(&count)
	case *sql.Tx:
		err = d.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM iam.location l, iam.role r
			WHERE l.location_id = $1 AND r.role_id = $2
			AND l.org_id = $3 AND r.org_id = $3
		`, locationID, roleID, orgID).Scan(&count)
	default:
		return fmt.Errorf("invalid database connection type")
	}

	if err != nil {
		dao.Logger.WithError(err).Error("Failed to validate location and role")
		return fmt.Errorf("failed to validate location and role: %w", err)
	}

	if count == 0 {
		dao.Logger.WithFields(logrus.Fields{
			"location_id": locationID,
			"role_id":     roleID,
			"org_id":      orgID,
		}).Warn("Location or role not found in organization")
		return fmt.Errorf("location or role not found in organization")
	}

	return nil
}
