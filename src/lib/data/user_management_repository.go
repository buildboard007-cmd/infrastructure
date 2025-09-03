package data

import (
	"context"
	"database/sql"
	"fmt"
	"infrastructure/lib/models"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/sirupsen/logrus"
)

// UserManagementRepository defines the interface for user management operations
type UserManagementRepository interface {
	// CreateUser creates a new user in the organization (legacy method)
	CreateUser(ctx context.Context, orgID int64, user *models.User) (*models.User, error)

	// CreateNormalUser creates a normal user (non-super admin) with Cognito integration
	CreateNormalUser(ctx context.Context, orgID int64, request *models.CreateUserRequest, createdBy int64) (*models.CreateUserResponse, error)

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

	// SendPasswordResetEmail sends a password reset email to a user
	SendPasswordResetEmail(ctx context.Context, userEmail string) error
}

// UserManagementDao implements UserManagementRepository interface using PostgreSQL
type UserManagementDao struct {
	DB            *sql.DB
	Logger        *logrus.Logger
	CognitoClient *cognitoidentityprovider.Client
	UserPoolID    string
	ClientID      string
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
		INSERT INTO iam.users (cognito_id, email, first_name, last_name, phone, mobile, job_title, employee_id, avatar_url, last_selected_location_id, is_super_admin, status, org_id, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`, user.CognitoID, user.Email, user.FirstName, user.LastName, phone, mobile, jobTitle, employeeID, avatarURL, user.LastSelectedLocationID, user.IsSuperAdmin, user.Status, orgID, 1, 1).Scan(
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
		"user_id": userID,
		"org_id":  orgID,
		"email":   user.Email,
	}).Info("Successfully created user")

	return user, nil
}

// CreateNormalUser creates a normal user (non-super admin) with Cognito integration
func (dao *UserManagementDao) CreateNormalUser(ctx context.Context, orgID int64, request *models.CreateUserRequest, createdBy int64) (*models.CreateUserResponse, error) {
	// Generate temporary password
	tempPassword := generateTemporaryPassword()

	// Create user in Cognito first - default behavior sends welcome email
	cognitoInput := &cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId: aws.String(dao.UserPoolID),
		Username:   aws.String(request.Email),
		UserAttributes: []types.AttributeType{
			{
				Name:  aws.String("email"),
				Value: aws.String(request.Email),
			},
			{
				Name:  aws.String("email_verified"),
				Value: aws.String("true"),
			},
			{
				Name:  aws.String("custom:isSuperAdmin"),
				Value: aws.String("false"),
			},
		},
		TemporaryPassword: aws.String(tempPassword),
		// No MessageAction specified - uses default behavior to send invite email
	}

	cognitoResult, err := dao.CognitoClient.AdminCreateUser(ctx, cognitoInput)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"email": request.Email,
			"error": err.Error(),
		}).Error("Failed to create user in Cognito")
		return nil, fmt.Errorf("failed to create user in Cognito: %w", err)
	}

	cognitoUserID := *cognitoResult.User.Username

	// Create user record in database
	var userID int64
	var createdAt, updatedAt time.Time

	phone := sql.NullString{String: request.Phone, Valid: request.Phone != ""}
	mobile := sql.NullString{String: request.Mobile, Valid: request.Mobile != ""}
	jobTitle := sql.NullString{String: request.JobTitle, Valid: request.JobTitle != ""}
	employeeID := sql.NullString{String: request.EmployeeID, Valid: request.EmployeeID != ""}
	avatarURL := sql.NullString{String: request.AvatarURL, Valid: request.AvatarURL != ""}
	lastSelectedLocationID := sql.NullInt64{Int64: request.LastSelectedLocationID, Valid: request.LastSelectedLocationID != 0}

	err = dao.DB.QueryRowContext(ctx, `
		INSERT INTO iam.users (cognito_id, email, first_name, last_name, phone, mobile, job_title, employee_id, avatar_url, last_selected_location_id, is_super_admin, status, org_id, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`, cognitoUserID, request.Email, request.FirstName, request.LastName, phone, mobile, jobTitle, employeeID, avatarURL, lastSelectedLocationID, false, "pending", orgID, createdBy, createdBy).Scan(
		&userID, &createdAt, &updatedAt)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"org_id":     orgID,
			"email":      request.Email,
			"cognito_id": cognitoUserID,
			"error":      err.Error(),
		}).Error("Failed to create user in database")

		// If database creation fails, clean up Cognito user
		_, deleteErr := dao.CognitoClient.AdminDeleteUser(ctx, &cognitoidentityprovider.AdminDeleteUserInput{
			UserPoolId: aws.String(dao.UserPoolID),
			Username:   aws.String(cognitoUserID),
		})
		if deleteErr != nil {
			dao.Logger.WithError(deleteErr).Error("Failed to cleanup Cognito user after database error")
		}

		return nil, fmt.Errorf("failed to create user in database: %w", err)
	}

	// Email with temporary password is automatically sent via MessageAction: RESEND

	dao.Logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"org_id":     orgID,
		"email":      request.Email,
		"cognito_id": cognitoUserID,
	}).Info("Successfully created normal user with password reset")

	// Get the created user with location-role assignments
	userWithAssignments, err := dao.GetUserByID(ctx, userID, orgID)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to get created user")
		return nil, fmt.Errorf("user created but failed to retrieve details: %w", err)
	}

	return &models.CreateUserResponse{
		UserWithLocationsAndRoles: *userWithAssignments,
		Message:                   "User created successfully. Welcome email with temporary password sent.",
		TemporaryPassword:         tempPassword,
	}, nil
}

// generateTemporaryPassword generates a secure temporary password
func generateTemporaryPassword() string {
	// Generate a random 12-character password with mixed case, numbers, and symbols
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, 12)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// GetUsersByOrg retrieves all users for a specific organization with their location-role assignments
func (dao *UserManagementDao) GetUsersByOrg(ctx context.Context, orgID int64) ([]models.UserWithLocationsAndRoles, error) {
	query := `
		SELECT u.id, u.cognito_id, u.email, u.first_name, u.last_name, 
		       u.phone, u.mobile, u.job_title, u.employee_id, u.avatar_url, u.last_selected_location_id, u.is_super_admin, u.status, u.org_id, u.created_at, u.updated_at
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
			&user.Phone, &user.Mobile, &user.JobTitle, &user.EmployeeID, &user.AvatarURL, &user.LastSelectedLocationID, &user.IsSuperAdmin, &user.Status, &user.OrgID,
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
		       avatar_url, last_selected_location_id, is_super_admin, status, org_id, created_at, updated_at
		FROM iam.users
		WHERE id = $1 AND org_id = $2 AND is_deleted = FALSE
	`

	err := dao.DB.QueryRowContext(ctx, query, userID, orgID).Scan(
		&user.UserID, &user.CognitoID, &user.Email, &user.FirstName, &user.LastName,
		&user.Phone, &user.Mobile, &user.JobTitle, &user.EmployeeID, &user.AvatarURL, &user.LastSelectedLocationID, &user.IsSuperAdmin, &user.Status, &user.OrgID,
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
		       avatar_url, last_selected_location_id, is_super_admin, status, org_id, created_at, updated_at
		FROM iam.users
		WHERE cognito_id = $1 AND org_id = $2 AND is_deleted = FALSE
	`

	err := dao.DB.QueryRowContext(ctx, query, cognitoID, orgID).Scan(
		&user.UserID, &user.CognitoID, &user.Email, &user.FirstName, &user.LastName,
		&user.Phone, &user.Mobile, &user.JobTitle, &user.EmployeeID, &user.AvatarURL, &user.LastSelectedLocationID, &user.IsSuperAdmin, &user.Status, &user.OrgID,
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
		SET first_name = $1, last_name = $2, phone = $3, mobile = $4, job_title = $5, employee_id = $6, avatar_url = $7, last_selected_location_id = $8, status = $9, updated_by = $10
		WHERE id = $11 AND org_id = $12 AND is_deleted = FALSE
		RETURNING id, cognito_id, email, first_name, last_name, phone, mobile, job_title, employee_id, 
		          avatar_url, last_selected_location_id, is_super_admin, status, org_id, created_at, updated_at
	`

	var updatedUser models.User
	err := dao.DB.QueryRowContext(ctx, query,
		user.FirstName, user.LastName, user.Phone, user.Mobile, user.JobTitle, user.EmployeeID, user.AvatarURL, user.LastSelectedLocationID, user.Status, userID,
		userID, orgID,
	).Scan(
		&updatedUser.UserID, &updatedUser.CognitoID, &updatedUser.Email, &updatedUser.FirstName,
		&updatedUser.LastName, &updatedUser.Phone, &updatedUser.Mobile, &updatedUser.JobTitle, &updatedUser.EmployeeID, &updatedUser.AvatarURL, &updatedUser.LastSelectedLocationID, &updatedUser.IsSuperAdmin,
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

// SendPasswordResetEmail sends a password reset email to a user
func (dao *UserManagementDao) SendPasswordResetEmail(ctx context.Context, userEmail string) error {
	// Use Cognito's AdminInitiateAuth to trigger password reset
	input := &cognitoidentityprovider.AdminResetUserPasswordInput{
		UserPoolId: aws.String(dao.UserPoolID),
		Username:   aws.String(userEmail),
	}

	_, err := dao.CognitoClient.AdminResetUserPassword(ctx, input)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"email": userEmail,
			"error": err.Error(),
		}).Error("Failed to send password reset email")
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	dao.Logger.WithField("email", userEmail).Info("Successfully sent password reset email")
	return nil
}
