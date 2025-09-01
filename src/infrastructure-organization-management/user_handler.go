package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"infrastructure/lib/api"
	"infrastructure/lib/auth"
	"infrastructure/lib/data"
	"infrastructure/lib/models"
	"net/http"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/sirupsen/logrus"
)

func (h *Handler) handleUserRoutes(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	method := request.HTTPMethod
	pathParams := request.PathParameters
	
	switch {
	case method == http.MethodPost && request.Resource == "/users":
		return h.createUser(ctx, request)
	case method == http.MethodGet && request.Resource == "/users":
		return h.listUsers(ctx, request)
	case method == http.MethodGet && request.Resource == "/users/{id}":
		return h.getUserByID(ctx, request, pathParams["id"])
	case method == http.MethodPut && request.Resource == "/users/{id}":
		return h.updateUser(ctx, request, pathParams["id"])
	case method == http.MethodDelete && request.Resource == "/users/{id}":
		return h.deleteUser(ctx, request, pathParams["id"])
	case method == http.MethodPut && request.Resource == "/users/{id}/status":
		return h.updateUserStatus(ctx, request, pathParams["id"])
	default:
		return api.ErrorResponse(http.StatusMethodNotAllowed, "Method not allowed", h.Logger), nil
	}
}

func (h *Handler) createUser(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract JWT claims and validate super admin
	claims, err := auth.ExtractClaimsFromRequest(request)
	if err != nil {
		h.Logger.WithError(err).Error("Failed to extract JWT claims")
		return api.ErrorResponse(http.StatusUnauthorized, "Unauthorized", h.Logger), nil
	}

	if !claims.IsSuperAdmin {
		h.Logger.WithField("user_id", claims.UserID).Warn("Non-super admin attempted to create user")
		return api.ErrorResponse(http.StatusForbidden, "Super admin access required", h.Logger), nil
	}

	// Parse request body
	var createRequest models.CreateUserRequest
	if err := json.Unmarshal([]byte(request.Body), &createRequest); err != nil {
		h.Logger.WithError(err).Error("Failed to parse create user request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", h.Logger), nil
	}

	// Validate required fields
	if createRequest.Email == "" || createRequest.FirstName == "" || createRequest.LastName == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Email, first name, and last name are required", h.Logger), nil
	}

	if len(createRequest.LocationRoleAssignments) == 0 {
		return api.ErrorResponse(http.StatusBadRequest, "At least one location-role assignment is required", h.Logger), nil
	}

	// Start transaction for user creation
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		h.Logger.WithError(err).Error("Failed to start transaction for user creation")
		return api.ErrorResponse(http.StatusInternalServerError, "Internal server error", h.Logger), nil
	}
	defer tx.Rollback()

	// Create user in Cognito first
	tempPassword, cognitoID, err := h.createCognitoUser(ctx, createRequest.Email, createRequest.FirstName, createRequest.LastName)
	if err != nil {
		h.Logger.WithError(err).Error("Failed to create user in Cognito")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create user account", h.Logger), nil
	}

	// Create user object for database
	user := &models.User{
		CognitoID:    cognitoID,
		Email:        createRequest.Email,
		FirstName:    createRequest.FirstName,
		LastName:     createRequest.LastName,
		Status:       models.UserStatusPending,
		IsSuperAdmin: false,
	}

	// Set optional fields
	if createRequest.Phone != "" {
		user.Phone = sql.NullString{String: createRequest.Phone, Valid: true}
	}
	if createRequest.JobTitle != "" {
		user.JobTitle = sql.NullString{String: createRequest.JobTitle, Valid: true}
	}
	if createRequest.AvatarURL != "" {
		user.AvatarURL = sql.NullString{String: createRequest.AvatarURL, Valid: true}
	}

	// Create user in database
	userRepo := &data.UserManagementDao{DB: h.DB, Logger: h.Logger}
	createdUser, err := userRepo.CreateUser(ctx, claims.OrgID, user)
	if err != nil {
		// If database creation fails, clean up Cognito user
		h.deleteCognitoUser(ctx, cognitoID)
		h.Logger.WithError(err).Error("Failed to create user in database")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create user", h.Logger), nil
	}

	// Assign location-role assignments
	locationRoleRepo := &data.UserLocationRoleDao{DB: h.DB, Logger: h.Logger}
	err = locationRoleRepo.UpdateUserLocationRoleAssignments(ctx, createdUser.UserID, claims.OrgID, createRequest.LocationRoleAssignments)
	if err != nil {
		// Clean up on failure
		h.deleteCognitoUser(ctx, cognitoID)
		h.Logger.WithError(err).Error("Failed to create user location-role assignments")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to assign user roles", h.Logger), nil
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		h.deleteCognitoUser(ctx, cognitoID)
		h.Logger.WithError(err).Error("Failed to commit user creation transaction")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create user", h.Logger), nil
	}

	// Get user assignments for response
	assignments, err := locationRoleRepo.GetUserAssignments(ctx, createdUser.UserID)
	if err != nil {
		h.Logger.WithError(err).Error("Failed to get user assignments for response")
		assignments = []models.UserLocationRoleAssignment{} // Continue with empty assignments
	}

	// Build response
	response := models.CreateUserResponse{
		UserWithLocationsAndRoles: models.UserWithLocationsAndRoles{
			User:                    *createdUser,
			LocationRoleAssignments: assignments,
		},
		TemporaryPassword: tempPassword,
		Message:          "User created successfully. Temporary password sent via email.",
	}

	h.Logger.WithFields(logrus.Fields{
		"user_id": createdUser.UserID,
		"email":   createdUser.Email,
		"org_id":  claims.OrgID,
	}).Info("Successfully created user")

	return api.SuccessResponse(http.StatusCreated, response, h.Logger), nil
}

func (h *Handler) listUsers(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract JWT claims
	claims, err := auth.ExtractClaimsFromRequest(request)
	if err != nil {
		h.Logger.WithError(err).Error("Failed to extract JWT claims")
		return api.ErrorResponse(http.StatusUnauthorized, "Unauthorized", h.Logger), nil
	}

	if !claims.IsSuperAdmin {
		h.Logger.WithField("user_id", claims.UserID).Warn("Non-super admin attempted to list users")
		return api.ErrorResponse(http.StatusForbidden, "Super admin access required", h.Logger), nil
	}

	// Get users from database
	userRepo := &data.UserManagementDao{DB: h.DB, Logger: h.Logger}
	users, err := userRepo.GetUsersByOrg(ctx, claims.OrgID)
	if err != nil {
		h.Logger.WithError(err).Error("Failed to get users")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to retrieve users", h.Logger), nil
	}

	response := models.UserListResponse{
		Users: users,
		Total: len(users),
	}

	return api.SuccessResponse(http.StatusOK, response, h.Logger), nil
}

func (h *Handler) getUserByID(ctx context.Context, request events.APIGatewayProxyRequest, userIDStr string) (events.APIGatewayProxyResponse, error) {
	// Extract JWT claims
	claims, err := auth.ExtractClaimsFromRequest(request)
	if err != nil {
		h.Logger.WithError(err).Error("Failed to extract JWT claims")
		return api.ErrorResponse(http.StatusUnauthorized, "Unauthorized", h.Logger), nil
	}

	if !claims.IsSuperAdmin {
		h.Logger.WithField("user_id", claims.UserID).Warn("Non-super admin attempted to get user")
		return api.ErrorResponse(http.StatusForbidden, "Super admin access required", h.Logger), nil
	}

	// Parse user ID
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return api.ErrorResponse(http.StatusBadRequest, "Invalid user ID", h.Logger), nil
	}

	// Get user from database
	userRepo := &data.UserManagementDao{DB: h.DB, Logger: h.Logger}
	user, err := userRepo.GetUserByID(ctx, userID, claims.OrgID)
	if err != nil {
		if err.Error() == "user not found" {
			return api.ErrorResponse(http.StatusNotFound, "User not found", h.Logger), nil
		}
		h.Logger.WithError(err).Error("Failed to get user")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to retrieve user", h.Logger), nil
	}

	return api.SuccessResponse(http.StatusOK, user, h.Logger), nil
}

func (h *Handler) updateUser(ctx context.Context, request events.APIGatewayProxyRequest, userIDStr string) (events.APIGatewayProxyResponse, error) {
	// Extract JWT claims
	claims, err := auth.ExtractClaimsFromRequest(request)
	if err != nil {
		h.Logger.WithError(err).Error("Failed to extract JWT claims")
		return api.ErrorResponse(http.StatusUnauthorized, "Unauthorized", h.Logger), nil
	}

	if !claims.IsSuperAdmin {
		h.Logger.WithField("user_id", claims.UserID).Warn("Non-super admin attempted to update user")
		return api.ErrorResponse(http.StatusForbidden, "Super admin access required", h.Logger), nil
	}

	// Parse user ID
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return api.ErrorResponse(http.StatusBadRequest, "Invalid user ID", h.Logger), nil
	}

	// Parse request body
	var updateRequest models.UpdateUserRequest
	if err := json.Unmarshal([]byte(request.Body), &updateRequest); err != nil {
		h.Logger.WithError(err).Error("Failed to parse update user request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", h.Logger), nil
	}

	// Create user object for update
	user := &models.User{}
	if updateRequest.FirstName != "" {
		user.FirstName = updateRequest.FirstName
	}
	if updateRequest.LastName != "" {
		user.LastName = updateRequest.LastName
	}
	if updateRequest.Phone != "" {
		user.Phone = sql.NullString{String: updateRequest.Phone, Valid: true}
	}
	if updateRequest.JobTitle != "" {
		user.JobTitle = sql.NullString{String: updateRequest.JobTitle, Valid: true}
	}
	if updateRequest.AvatarURL != "" {
		user.AvatarURL = sql.NullString{String: updateRequest.AvatarURL, Valid: true}
	}
	if updateRequest.Status != "" {
		user.Status = updateRequest.Status
	}

	// Update user in database
	userRepo := &data.UserManagementDao{DB: h.DB, Logger: h.Logger}
	_, err = userRepo.UpdateUser(ctx, userID, claims.OrgID, user)
	if err != nil {
		if err.Error() == "user not found" {
			return api.ErrorResponse(http.StatusNotFound, "User not found", h.Logger), nil
		}
		h.Logger.WithError(err).Error("Failed to update user")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update user", h.Logger), nil
	}

	// Update location-role assignments (required in update request)
	locationRoleRepo := &data.UserLocationRoleDao{DB: h.DB, Logger: h.Logger}
	err = locationRoleRepo.UpdateUserLocationRoleAssignments(ctx, userID, claims.OrgID, updateRequest.LocationRoleAssignments)
	if err != nil {
		h.Logger.WithError(err).Error("Failed to update user location-role assignments")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update user assignments", h.Logger), nil
	}

	// Get updated user with assignments
	userWithAssignments, err := userRepo.GetUserByID(ctx, userID, claims.OrgID)
	if err != nil {
		h.Logger.WithError(err).Error("Failed to get updated user")
		return api.ErrorResponse(http.StatusInternalServerError, "User updated but failed to retrieve updated data", h.Logger), nil
	}

	return api.SuccessResponse(http.StatusOK, userWithAssignments, h.Logger), nil
}

func (h *Handler) updateUserStatus(ctx context.Context, request events.APIGatewayProxyRequest, userIDStr string) (events.APIGatewayProxyResponse, error) {
	// Extract JWT claims
	claims, err := auth.ExtractClaimsFromRequest(request)
	if err != nil {
		h.Logger.WithError(err).Error("Failed to extract JWT claims")
		return api.ErrorResponse(http.StatusUnauthorized, "Unauthorized", h.Logger), nil
	}

	if !claims.IsSuperAdmin {
		h.Logger.WithField("user_id", claims.UserID).Warn("Non-super admin attempted to update user status")
		return api.ErrorResponse(http.StatusForbidden, "Super admin access required", h.Logger), nil
	}

	// Parse user ID
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return api.ErrorResponse(http.StatusBadRequest, "Invalid user ID", h.Logger), nil
	}

	// Parse request body
	var statusRequest models.UpdateUserStatusRequest
	if err := json.Unmarshal([]byte(request.Body), &statusRequest); err != nil {
		h.Logger.WithError(err).Error("Failed to parse update user status request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", h.Logger), nil
	}

	// Update user status
	userRepo := &data.UserManagementDao{DB: h.DB, Logger: h.Logger}
	err = userRepo.UpdateUserStatus(ctx, userID, claims.OrgID, statusRequest.Status)
	if err != nil {
		if err.Error() == "user not found" {
			return api.ErrorResponse(http.StatusNotFound, "User not found", h.Logger), nil
		}
		h.Logger.WithError(err).Error("Failed to update user status")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update user status", h.Logger), nil
	}

	response := map[string]interface{}{
		"message": "User status updated successfully",
		"status":  statusRequest.Status,
	}

	return api.SuccessResponse(http.StatusOK, response, h.Logger), nil
}


func (h *Handler) deleteUser(ctx context.Context, request events.APIGatewayProxyRequest, userIDStr string) (events.APIGatewayProxyResponse, error) {
	// Extract JWT claims
	claims, err := auth.ExtractClaimsFromRequest(request)
	if err != nil {
		h.Logger.WithError(err).Error("Failed to extract JWT claims")
		return api.ErrorResponse(http.StatusUnauthorized, "Unauthorized", h.Logger), nil
	}

	if !claims.IsSuperAdmin {
		h.Logger.WithField("user_id", claims.UserID).Warn("Non-super admin attempted to delete user")
		return api.ErrorResponse(http.StatusForbidden, "Super admin access required", h.Logger), nil
	}

	// Parse user ID
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return api.ErrorResponse(http.StatusBadRequest, "Invalid user ID", h.Logger), nil
	}

	// Get user details before deletion for Cognito cleanup
	userRepo := &data.UserManagementDao{DB: h.DB, Logger: h.Logger}
	user, err := userRepo.GetUserByID(ctx, userID, claims.OrgID)
	if err != nil {
		if err.Error() == "user not found" {
			return api.ErrorResponse(http.StatusNotFound, "User not found", h.Logger), nil
		}
		h.Logger.WithError(err).Error("Failed to get user for deletion")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to delete user", h.Logger), nil
	}

	// Delete user from database (this also removes assignments via transaction)
	err = userRepo.DeleteUser(ctx, userID, claims.OrgID)
	if err != nil {
		h.Logger.WithError(err).Error("Failed to delete user from database")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to delete user", h.Logger), nil
	}

	// Delete user from Cognito
	err = h.deleteCognitoUser(ctx, user.CognitoID)
	if err != nil {
		h.Logger.WithError(err).Warn("Failed to delete user from Cognito, but database deletion succeeded")
	}

	response := map[string]string{
		"message": "User deleted successfully",
	}

	return api.SuccessResponse(http.StatusOK, response, h.Logger), nil
}

// Helper function to create user in Cognito
func (h *Handler) createCognitoUser(ctx context.Context, email, firstName, lastName string) (tempPassword, cognitoID string, err error) {
	// Generate temporary password (8 characters with mixed case, numbers, and special chars)
	tempPassword = generateTempPassword()

	// Create user in Cognito
	input := &cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId:        aws.String(h.UserPoolID),
		Username:          aws.String(email),
		TemporaryPassword: aws.String(tempPassword),
		MessageAction:     types.MessageActionType("SEND"), // Send welcome email
		UserAttributes: []types.AttributeType{
			{
				Name:  aws.String("email"),
				Value: aws.String(email),
			},
			{
				Name:  aws.String("given_name"),
				Value: aws.String(firstName),
			},
			{
				Name:  aws.String("family_name"),
				Value: aws.String(lastName),
			},
			{
				Name:  aws.String("email_verified"),
				Value: aws.String("true"),
			},
		},
	}

	result, err := h.CognitoClient.AdminCreateUser(ctx, input)
	if err != nil {
		return "", "", fmt.Errorf("failed to create user in Cognito: %w", err)
	}

	// Extract the Cognito user ID (sub) from the response
	for _, attr := range result.User.Attributes {
		if *attr.Name == "sub" {
			cognitoID = *attr.Value
			break
		}
	}

	if cognitoID == "" {
		return "", "", fmt.Errorf("failed to get Cognito user ID from response")
	}

	return tempPassword, cognitoID, nil
}

// Helper function to delete user from Cognito
func (h *Handler) deleteCognitoUser(ctx context.Context, cognitoID string) error {
	input := &cognitoidentityprovider.AdminDeleteUserInput{
		UserPoolId: aws.String(h.UserPoolID),
		Username:   aws.String(cognitoID),
	}

	_, err := h.CognitoClient.AdminDeleteUser(ctx, input)
	return err
}

// Helper function to generate temporary password
func generateTempPassword() string {
	// Simple temporary password generation - in production, use crypto/rand
	return "TempPass1!"
}