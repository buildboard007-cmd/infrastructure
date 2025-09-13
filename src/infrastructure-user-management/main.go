package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"infrastructure/lib/api"
	"infrastructure/lib/auth"
	"infrastructure/lib/clients"
	"infrastructure/lib/constants"
	"infrastructure/lib/data"
	"infrastructure/lib/models"
	"infrastructure/lib/util"
	"net/http"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/sirupsen/logrus"
)

// Global variables for Lambda cold start optimization
var (
	logger              *logrus.Logger
	isLocal             bool
	ssmRepository       data.SSMRepository
	ssmParams           map[string]string
	sqlDB               *sql.DB
	userRepository      data.UserManagementRepository
	cognitoClient       *cognitoidentityprovider.Client
	userPoolID          string
	clientID            string
)

func LambdaHandler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.WithFields(logrus.Fields{
		"operation": "LambdaHandler",
		"method":    request.HTTPMethod,
		"path":      request.Path,
		"resource":  request.Resource,
	}).Info("User management request received")

	// Extract claims from JWT token via API Gateway authorizer
	claims, err := auth.ExtractClaimsFromRequest(request)
	if err != nil {
		logger.WithError(err).Error("Authentication failed")
		return api.ErrorResponse(http.StatusUnauthorized, "Authentication failed", logger), nil
	}

	// Check authorization based on the endpoint being accessed
	if request.Resource != "/users/{userId}/location" && !claims.IsSuperAdmin {
		logger.WithField("user_id", claims.UserID).Warn("User is not a super admin")
		return api.ErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can manage users", logger), nil
	}

	// Route based on HTTP method
	switch request.HTTPMethod {
	case http.MethodPost:
		return handleCreateUser(ctx, request, claims), nil
	case http.MethodGet:
		if userID := request.PathParameters["userId"]; userID != "" {
			return handleGetUser(ctx, request, claims), nil
		}
		return handleGetUsers(ctx, request, claims), nil
	case http.MethodPut:
		return handleUpdateUser(ctx, request, claims), nil
	case http.MethodDelete:
		return handleDeleteUser(ctx, request, claims), nil
	case http.MethodPatch:
		// Handle password reset requests via PATCH /users/{userId}/reset-password
		if request.PathParameters["userId"] != "" && request.Resource == "/users/{userId}/reset-password" {
			return handlePasswordReset(ctx, request, claims), nil
		}
		// Handle location update requests via PATCH /users/{userId}/location
		if request.PathParameters["userId"] != "" && request.Resource == "/users/{userId}/location" {
			return handleLocationUpdate(ctx, request, claims), nil
		}
		return api.ErrorResponse(http.StatusNotFound, "Endpoint not found", logger), nil
	default:
		return api.ErrorResponse(http.StatusMethodNotAllowed, "Method not allowed", logger), nil
	}
}

// handleCreateUser handles POST /users
func handleCreateUser(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) events.APIGatewayProxyResponse {
	var createRequest models.CreateUserRequest
	if err := json.Unmarshal([]byte(request.Body), &createRequest); err != nil {
		logger.WithError(err).Error("Invalid request body for create user")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Create user with Cognito integration
	response, err := userRepository.CreateNormalUser(ctx, claims.OrgID, &createRequest, claims.UserID)
	if err != nil {
		logger.WithError(err).Error("Failed to create user")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create user", logger)
	}

	return api.SuccessResponse(http.StatusCreated, response, logger)
}

// handleGetUsers handles GET /users
func handleGetUsers(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) events.APIGatewayProxyResponse {
	users, err := userRepository.GetUsersByOrg(ctx, claims.OrgID)
	if err != nil {
		logger.WithError(err).Error("Failed to get users")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get users", logger)
	}

	response := models.UserListResponse{
		Users: users,
		Total: len(users),
	}

	return api.SuccessResponse(http.StatusOK, response, logger)
}

// handleGetUser handles GET /users/{userId}
func handleGetUser(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) events.APIGatewayProxyResponse {
	userID, err := strconv.ParseInt(request.PathParameters["userId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid user ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid user ID", logger)
	}

	user, err := userRepository.GetUserByID(ctx, userID, claims.OrgID)
	if err != nil {
		logger.WithError(err).Error("Failed to get user")
		return api.ErrorResponse(http.StatusNotFound, "User not found", logger)
	}

	return api.SuccessResponse(http.StatusOK, user, logger)
}

// handleUpdateUser handles PUT /users/{userId}
func handleUpdateUser(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) events.APIGatewayProxyResponse {
	userID, err := strconv.ParseInt(request.PathParameters["userId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid user ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid user ID", logger)
	}

	var updateRequest models.UpdateUserRequest
	if err := json.Unmarshal([]byte(request.Body), &updateRequest); err != nil {
		logger.WithError(err).Error("Invalid request body for update user")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Convert to User model for repository
	user := &models.User{
		Email:             updateRequest.Email,
		FirstName:         updateRequest.FirstName,
		LastName:          updateRequest.LastName,
		Phone:             sql.NullString{String: updateRequest.Phone, Valid: updateRequest.Phone != ""},
		Mobile:            sql.NullString{String: updateRequest.Mobile, Valid: updateRequest.Mobile != ""},
		JobTitle:          sql.NullString{String: updateRequest.JobTitle, Valid: updateRequest.JobTitle != ""},
		EmployeeID:        sql.NullString{String: updateRequest.EmployeeID, Valid: updateRequest.EmployeeID != ""},
		AvatarURL:         sql.NullString{String: updateRequest.AvatarURL, Valid: updateRequest.AvatarURL != ""},
		LastSelectedLocationID: sql.NullInt64{Int64: updateRequest.LastSelectedLocationID, Valid: updateRequest.LastSelectedLocationID != 0},
		Status:            updateRequest.Status,
	}

	updatedUser, err := userRepository.UpdateUser(ctx, userID, claims.OrgID, user, claims.UserID)
	if err != nil {
		logger.WithError(err).Error("Failed to update user")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update user", logger)
	}

	return api.SuccessResponse(http.StatusOK, updatedUser, logger)
}

// handleDeleteUser handles DELETE /users/{userId}
func handleDeleteUser(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) events.APIGatewayProxyResponse {
	userID, err := strconv.ParseInt(request.PathParameters["userId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid user ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid user ID", logger)
	}

	err = userRepository.DeleteUser(ctx, userID, claims.OrgID)
	if err != nil {
		logger.WithError(err).Error("Failed to delete user")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to delete user", logger)
	}

	return api.SuccessResponse(http.StatusOK, map[string]string{"message": "User deleted successfully"}, logger)
}

// handlePasswordReset handles PATCH /users/{userId}/reset-password
func handlePasswordReset(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) events.APIGatewayProxyResponse {
	userID, err := strconv.ParseInt(request.PathParameters["userId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid user ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid user ID", logger)
	}

	// Get the user to retrieve their email
	user, err := userRepository.GetUserByID(ctx, userID, claims.OrgID)
	if err != nil {
		logger.WithError(err).Error("Failed to get user for password reset")
		return api.ErrorResponse(http.StatusNotFound, "User not found", logger)
	}

	// Send password reset email
	err = userRepository.SendPasswordResetEmail(ctx, user.Email)
	if err != nil {
		logger.WithError(err).Error("Failed to send password reset email")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to send password reset email", logger)
	}

	return api.SuccessResponse(http.StatusOK, map[string]string{"message": "Password reset email sent successfully"}, logger)
}

// handleLocationUpdate handles PATCH /users/{userId}/location
func handleLocationUpdate(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) events.APIGatewayProxyResponse {
	userID, err := strconv.ParseInt(request.PathParameters["userId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid user ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid user ID", logger)
	}

	var locationRequest struct {
		LocationID int64 `json:"location_id" binding:"required"`
	}

	if err := json.Unmarshal([]byte(request.Body), &locationRequest); err != nil {
		logger.WithError(err).Error("Invalid request body for location update")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// For location updates, users can update their own location or super admins can update any user's location
	if !claims.IsSuperAdmin && claims.UserID != userID {
		logger.WithField("user_id", claims.UserID).Warn("User attempting to update another user's location")
		return api.ErrorResponse(http.StatusForbidden, "Forbidden: You can only update your own location", logger)
	}

	// Verify the user exists and belongs to the same organization
	_, err = userRepository.GetUserByID(ctx, userID, claims.OrgID)
	if err != nil {
		logger.WithError(err).Error("Failed to get user for location update")
		return api.ErrorResponse(http.StatusNotFound, "User not found", logger)
	}

	// Create partial user update with only location change
	userUpdate := &models.User{
		LastSelectedLocationID: sql.NullInt64{Int64: locationRequest.LocationID, Valid: true},
	}

	// Update user with location change
	updatedUser, err := userRepository.UpdateUser(ctx, userID, claims.OrgID, userUpdate, claims.UserID)
	if err != nil {
		logger.WithError(err).Error("Failed to update user location")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update location", logger)
	}

	logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"location_id": locationRequest.LocationID,
		"updated_by":  claims.UserID,
	}).Info("User location updated successfully")

	return api.SuccessResponse(http.StatusOK, map[string]interface{}{
		"message":     "Location updated successfully",
		"user_id":     updatedUser.UserID,
		"location_id": updatedUser.LastSelectedLocationID.Int64,
	}, logger)
}

func main() {
	lambda.Start(LambdaHandler)
}

func init() {
	var err error

	isLocal = parseIsLocal()

	// Logger setup
	logger = setupLogger(isLocal)

	// Initialize AWS SSM Parameter Store client for configuration management
	ssmClient := clients.NewSSMClient(isLocal)
	ssmRepository = &data.SSMDao{
		SSM:    ssmClient,
		Logger: logger,
	}

	// Retrieve configuration parameters from SSM Parameter Store
	ssmParams, err = ssmRepository.GetParameters()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"operation": "init",
			"error":     err.Error(),
		}).Fatal("Error while getting SSM params from parameter store")
	}

	// Initialize PostgreSQL database connection
	err = setupPostgresSQLClient(ssmParams)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"operation": "init",
			"error":     err.Error(),
		}).Fatal("Error setting up PostgreSQL client")
	}

	// Initialize Cognito client
	cognitoClient = clients.NewCognitoIdentityProviderClient(isLocal)
	
	// Get User Pool ID from SSM parameters
	userPoolID = ssmParams[constants.COGNITO_USER_POOL_ID]
	if userPoolID == "" {
		logger.Fatal("COGNITO_USER_POOL_ID not found in SSM parameters")
	}

	// Get Client ID from SSM parameters
	clientID = ssmParams[constants.COGNITO_CLIENT_ID]
	if clientID == "" {
		logger.Fatal("COGNITO_CLIENT_ID not found in SSM parameters")
	}

	// Initialize user repository with Cognito integration
	userRepository = &data.UserManagementDao{
		DB:            sqlDB,
		Logger:        logger,
		CognitoClient: cognitoClient,
		UserPoolID:    userPoolID,
		ClientID:      clientID,
	}

	logger.WithField("operation", "init").Info("User Management Lambda initialization completed successfully")
}

func parseIsLocal() bool {
	isLocal, _ := strconv.ParseBool(os.Getenv("IS_LOCAL"))
	return isLocal
}

func setupLogger(isLocal bool) *logrus.Logger {
	logger := logrus.New()
	util.SetLogLevel(logger, os.Getenv("LOG_LEVEL"))
	logger.SetFormatter(&logrus.JSONFormatter{PrettyPrint: isLocal})
	return logger
}

func setupPostgresSQLClient(ssmParams map[string]string) error {
	var err error

	// Create PostgreSQL client using RDS connection parameters from SSM
	sqlDB, err = clients.NewPostgresSQLClient(
		ssmParams[constants.DATABASE_RDS_ENDPOINT],
		ssmParams[constants.DATABASE_PORT],
		ssmParams[constants.DATABASE_NAME],
		ssmParams[constants.DATABASE_USERNAME],
		ssmParams[constants.DATABASE_PASSWORD],
		ssmParams[constants.SSL_MODE],
	)
	if err != nil {
		return fmt.Errorf("error creating PostgreSQL client: %w", err)
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.WithField("operation", "setupPostgresSQLClient").Debug("PostgreSQL client initialized successfully")
	}
	return nil
}