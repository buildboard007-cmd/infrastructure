package data

import (
	"context"
	"database/sql"
	"fmt"
	"infrastructure/lib/models"

	"github.com/sirupsen/logrus"
)

// UserManagementRepository defines the interface for user management operations
type UserManagementRepository interface {
	// CreateUser creates a new user in the organization
	CreateUser(ctx context.Context, orgID int64, user *models.User) (*models.User, error)
	
	// GetUsersByOrg retrieves all users for a specific organization
	GetUsersByOrg(ctx context.Context, orgID int64) ([]models.UserWithLocationsAndRoles, error)
	
	// GetUserByID retrieves a specific user by ID (with org validation)
	GetUserByID(ctx context.Context, userID, orgID int64) (*models.UserWithLocationsAndRoles, error)
	
	// GetUserByCognitoID retrieves a user by Cognito ID
	GetUserByCognitoID(ctx context.Context, cognitoID string, orgID int64) (*models.UserWithLocationsAndRoles, error)
	
	// UpdateUser updates an existing user
	UpdateUser(ctx context.Context, userID, orgID int64, user *models.User) (*models.User, error)
	
	// UpdateUserStatus updates user status (activate, deactivate, suspend, etc.)
	UpdateUserStatus(ctx context.Context, userID, orgID int64, status string) error
	
	// DeleteUser deletes a user from the system
	DeleteUser(ctx context.Context, userID, orgID int64) error
	
	// GetUserLocationRoleAssignments retrieves user's location-role assignments
	GetUserLocationRoleAssignments(ctx context.Context, userID int64) ([]models.UserLocationRoleAssignment, error)
}

// UserManagementDao implements UserManagementRepository interface using PostgreSQL
type UserManagementDao struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

// CreateUser creates a new user in the organization
func (dao *UserManagementDao) CreateUser(ctx context.Context, orgID int64, user *models.User) (*models.User, error) {
	var userID int64
	
	// Convert sql.NullString fields for insertion
	phone := sql.NullString{String: "", Valid: false}
	if user.Phone.Valid && user.Phone.String != "" {
		phone = user.Phone
	}
	
	jobTitle := sql.NullString{String: "", Valid: false}
	if user.JobTitle.Valid && user.JobTitle.String != "" {
		jobTitle = user.JobTitle
	}
	
	avatarURL := sql.NullString{String: "", Valid: false}
	if user.AvatarURL.Valid && user.AvatarURL.String != "" {
		avatarURL = user.AvatarURL
	}

	mobile := sql.NullString{String: "", Valid: false}
	if user.Mobile.Valid && user.Mobile.String != "" {
		mobile = user.Mobile
	}

	employeeID := sql.NullString{String: "", Valid: false}
	if user.EmployeeID.Valid && user.EmployeeID.String != "" {
		employeeID = user.EmployeeID
	}

	err := dao.DB.QueryRowContext(ctx, `
		INSERT INTO iam.users (cognito_id, email, first_name, last_name, phone, mobile, job_title, employee_id, avatar_url, current_location_id, is_super_admin, status, org_id, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`, user.CognitoID, user.Email, user.FirstName, user.LastName, phone, mobile, jobTitle, employeeID, avatarURL, user.CurrentLocationID, user.IsSuperAdmin, user.Status, orgID, 1, 1).Scan(
		&userID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"org_id":     orgID,
			"email":      user.Email,
			"cognito_id": user.CognitoID,
			"error":      err.Error(),
		}).Error("Failed to create user")
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Populate the response
	user.UserID = userID
	user.OrgID = orgID

	dao.Logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"org_id":   orgID,
		"email":    user.Email,
	}).Info("Successfully created user")

	return user, nil
}

// GetUsersByOrg retrieves all users for a specific organization with their location-role assignments
func (dao *UserManagementDao) GetUsersByOrg(ctx context.Context, orgID int64) ([]models.UserWithLocationsAndRoles, error) {
	query := `
		SELECT u.id, u.cognito_id, u.email, u.first_name, u.last_name, 
		       u.phone, u.mobile, u.job_title, u.employee_id, u.avatar_url, u.current_location_id, u.is_super_admin, u.status, u.org_id, u.created_at, u.updated_at
		FROM iam.users u
		WHERE u.org_id = $1 AND u.is_deleted = FALSE
		ORDER BY u.created_at DESC
	`

	rows, err := dao.DB.QueryContext(ctx, query, orgID)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"org_id": orgID,
			"error":  err.Error(),
		}).Error("Failed to query users")
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []models.UserWithLocationsAndRoles
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.UserID, &user.CognitoID, &user.Email, &user.FirstName, &user.LastName,
			&user.Phone, &user.Mobile, &user.JobTitle, &user.EmployeeID, &user.AvatarURL, &user.CurrentLocationID, &user.IsSuperAdmin, &user.Status, &user.OrgID,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan user row")
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		// Get location-role assignments for this user
		assignments, err := dao.GetUserLocationRoleAssignments(ctx, user.UserID)
		if err != nil {
			dao.Logger.WithError(err).WithField("user_id", user.UserID).Error("Failed to get user assignments")
			return nil, fmt.Errorf("failed to get user assignments: %w", err)
		}

		users = append(users, models.UserWithLocationsAndRoles{
			User:                    user,
			LocationRoleAssignments: assignments,
		})
	}

	if err = rows.Err(); err != nil {
		dao.Logger.WithError(err).Error("Error iterating user rows")
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"org_id": orgID,
		"count":  len(users),
	}).Debug("Successfully retrieved users for organization")

	return users, nil
}

// GetUserByID retrieves a specific user by ID with organization validation
func (dao *UserManagementDao) GetUserByID(ctx context.Context, userID, orgID int64) (*models.UserWithLocationsAndRoles, error) {
	var user models.User
	query := `
		SELECT id, cognito_id, email, first_name, last_name, phone, mobile, job_title, employee_id, 
		       avatar_url, current_location_id, is_super_admin, status, org_id, created_at, updated_at
		FROM iam.users
		WHERE id = $1 AND org_id = $2 AND is_deleted = FALSE
	`

	err := dao.DB.QueryRowContext(ctx, query, userID, orgID).Scan(
		&user.UserID, &user.CognitoID, &user.Email, &user.FirstName, &user.LastName,
		&user.Phone, &user.Mobile, &user.JobTitle, &user.EmployeeID, &user.AvatarURL, &user.CurrentLocationID, &user.IsSuperAdmin, &user.Status, &user.OrgID,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		dao.Logger.WithFields(logrus.Fields{
			"user_id": userID,
			"org_id":  orgID,
		}).Warn("User not found")
		return nil, fmt.Errorf("user not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"user_id": userID,
			"org_id":  orgID,
			"error":   err.Error(),
		}).Error("Failed to get user")
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get location-role assignments
	assignments, err := dao.GetUserLocationRoleAssignments(ctx, user.UserID)
	if err != nil {
		dao.Logger.WithError(err).WithField("user_id", user.UserID).Error("Failed to get user assignments")
		return nil, fmt.Errorf("failed to get user assignments: %w", err)
	}

	return &models.UserWithLocationsAndRoles{
		User:                    user,
		LocationRoleAssignments: assignments,
	}, nil
}

// GetUserByCognitoID retrieves a user by Cognito ID
func (dao *UserManagementDao) GetUserByCognitoID(ctx context.Context, cognitoID string, orgID int64) (*models.UserWithLocationsAndRoles, error) {
	var user models.User
	query := `
		SELECT id, cognito_id, email, first_name, last_name, phone, mobile, job_title, employee_id, 
		       avatar_url, current_location_id, is_super_admin, status, org_id, created_at, updated_at
		FROM iam.users
		WHERE cognito_id = $1 AND org_id = $2 AND is_deleted = FALSE
	`

	err := dao.DB.QueryRowContext(ctx, query, cognitoID, orgID).Scan(
		&user.UserID, &user.CognitoID, &user.Email, &user.FirstName, &user.LastName,
		&user.Phone, &user.Mobile, &user.JobTitle, &user.EmployeeID, &user.AvatarURL, &user.CurrentLocationID, &user.IsSuperAdmin, &user.Status, &user.OrgID,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		dao.Logger.WithFields(logrus.Fields{
			"cognito_id": cognitoID,
			"org_id":     orgID,
		}).Warn("User not found")
		return nil, fmt.Errorf("user not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"cognito_id": cognitoID,
			"org_id":     orgID,
			"error":      err.Error(),
		}).Error("Failed to get user")
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get location-role assignments
	assignments, err := dao.GetUserLocationRoleAssignments(ctx, user.UserID)
	if err != nil {
		dao.Logger.WithError(err).WithField("user_id", user.UserID).Error("Failed to get user assignments")
		return nil, fmt.Errorf("failed to get user assignments: %w", err)
	}

	return &models.UserWithLocationsAndRoles{
		User:                    user,
		LocationRoleAssignments: assignments,
	}, nil
}

// UpdateUser updates an existing user
func (dao *UserManagementDao) UpdateUser(ctx context.Context, userID, orgID int64, user *models.User) (*models.User, error) {
	query := `
		UPDATE iam.users 
		SET first_name = $1, last_name = $2, phone = $3, mobile = $4, job_title = $5, employee_id = $6, avatar_url = $7, current_location_id = $8, status = $9, updated_by = $10
		WHERE id = $11 AND org_id = $12 AND is_deleted = FALSE
		RETURNING id, cognito_id, email, first_name, last_name, phone, mobile, job_title, employee_id, 
		          avatar_url, current_location_id, is_super_admin, status, org_id, created_at, updated_at
	`

	var updatedUser models.User
	err := dao.DB.QueryRowContext(ctx, query,
		user.FirstName, user.LastName, user.Phone, user.Mobile, user.JobTitle, user.EmployeeID, user.AvatarURL, user.CurrentLocationID, user.Status, userID,
		userID, orgID,
	).Scan(
		&updatedUser.UserID, &updatedUser.CognitoID, &updatedUser.Email, &updatedUser.FirstName,
		&updatedUser.LastName, &updatedUser.Phone, &updatedUser.Mobile, &updatedUser.JobTitle, &updatedUser.EmployeeID, &updatedUser.AvatarURL, &updatedUser.CurrentLocationID, &updatedUser.IsSuperAdmin,
		&updatedUser.Status, &updatedUser.OrgID, &updatedUser.CreatedAt, &updatedUser.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		dao.Logger.WithFields(logrus.Fields{
			"user_id": userID,
			"org_id":  orgID,
		}).Warn("User not found for update")
		return nil, fmt.Errorf("user not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"user_id": userID,
			"org_id":  orgID,
			"error":   err.Error(),
		}).Error("Failed to update user")
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"user_id": userID,
		"org_id":  orgID,
		"email":   updatedUser.Email,
	}).Info("Successfully updated user")

	return &updatedUser, nil
}

// UpdateUserStatus updates user status
func (dao *UserManagementDao) UpdateUserStatus(ctx context.Context, userID, orgID int64, status string) error {
	result, err := dao.DB.ExecContext(ctx, `
		UPDATE iam.users 
		SET status = $1, updated_by = $2
		WHERE id = $3 AND org_id = $4 AND is_deleted = FALSE
	`, status, userID, userID, orgID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"user_id": userID,
			"org_id":  orgID,
			"status":  status,
			"error":   err.Error(),
		}).Error("Failed to update user status")
		return fmt.Errorf("failed to update user status: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		dao.Logger.WithFields(logrus.Fields{
			"user_id": userID,
			"org_id":  orgID,
		}).Warn("User not found for status update")
		return fmt.Errorf("user not found")
	}

	dao.Logger.WithFields(logrus.Fields{
		"user_id": userID,
		"org_id":  orgID,
		"status":  status,
	}).Info("Successfully updated user status")

	return nil
}

// DeleteUser deletes a user and all associated assignments
func (dao *UserManagementDao) DeleteUser(ctx context.Context, userID, orgID int64) error {
	// Start transaction
	tx, err := dao.DB.BeginTx(ctx, nil)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to start transaction for user deletion")
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// First, remove all user-location-role assignments
	_, err = tx.ExecContext(ctx, `
		UPDATE iam.user_location_access SET is_deleted = TRUE, updated_by = $1 WHERE user_id = $2
	`, userID, userID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"user_id": userID,
			"error":   err.Error(),
		}).Error("Failed to remove user location-role assignments")
		return fmt.Errorf("failed to remove user assignments: %w", err)
	}

	// Then delete the user (with org validation) - using soft delete
	result, err := tx.ExecContext(ctx, `
		UPDATE iam.users SET is_deleted = TRUE, updated_by = $1 WHERE id = $2 AND org_id = $3
	`, userID, userID, orgID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"user_id": userID,
			"org_id":  orgID,
			"error":   err.Error(),
		}).Error("Failed to delete user")
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		dao.Logger.WithFields(logrus.Fields{
			"user_id": userID,
			"org_id":  orgID,
		}).Warn("User not found for deletion")
		return fmt.Errorf("user not found")
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		dao.Logger.WithError(err).Error("Failed to commit user deletion transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"user_id": userID,
		"org_id":  orgID,
	}).Info("Successfully deleted user and all assignments")

	return nil
}

// GetUserLocationRoleAssignments retrieves user's location-role assignments
// Based on new schema: user_location_access + org_user_roles + location_user_roles
func (dao *UserManagementDao) GetUserLocationRoleAssignments(ctx context.Context, userID int64) ([]models.UserLocationRoleAssignment, error) {
	// For now, return empty assignments as the schema has changed significantly
	// This will need to be reimplemented with the new user_location_access, org_user_roles, and location_user_roles tables
	dao.Logger.WithField("user_id", userID).Debug("GetUserLocationRoleAssignments called - returning empty due to schema changes")
	return []models.UserLocationRoleAssignment{}, nil
}