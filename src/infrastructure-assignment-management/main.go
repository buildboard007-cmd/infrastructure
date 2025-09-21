package main

import (
	"context"
	"database/sql"
	"fmt"
	"infrastructure/lib/api"
	"infrastructure/lib/auth"
	"infrastructure/lib/clients"
	"infrastructure/lib/constants"
	"infrastructure/lib/data"
	"infrastructure/lib/models"
	"net/http"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"
)

var (
	logger               *logrus.Logger
	isLocal              bool
	ssmRepository        data.SSMRepository
	ssmParams            map[string]string
	sqlDB                *sql.DB
	assignmentRepository data.AssignmentRepository
)

// Handler processes API Gateway requests for assignment management operations
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.WithFields(logrus.Fields{
		"method":      request.HTTPMethod,
		"path":        request.Path,
		"resource":    request.Resource,
		"path_params": request.PathParameters,
		"operation":   "Handler",
	}).Debug("Processing assignment management request")

	// Extract claims from JWT token via API Gateway authorizer
	claims, err := auth.ExtractClaimsFromRequest(request)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":     err.Error(),
			"operation": "Handler",
		}).Error("Authentication failed")
		return api.ErrorResponse(http.StatusUnauthorized, "Authentication failed", logger), nil
	}

	logger.WithFields(logrus.Fields{
		"user_id":   claims.UserID,
		"org_id":    claims.OrgID,
		"email":     claims.Email,
		"operation": "Handler",
	}).Debug("User authenticated successfully")

	// Route the request based on path and method
	switch {
	// Basic assignment CRUD operations
	case request.Resource == "/assignments" && request.HTTPMethod == "POST":
		return handleCreateAssignment(ctx, request, claims)
	case request.Resource == "/assignments" && request.HTTPMethod == "GET":
		return handleGetAssignments(ctx, request, claims)
	case request.Resource == "/assignments/{assignmentId}" && request.HTTPMethod == "GET":
		return handleGetAssignment(ctx, request, claims)
	case request.Resource == "/assignments/{assignmentId}" && request.HTTPMethod == "PUT":
		return handleUpdateAssignment(ctx, request, claims)
	case request.Resource == "/assignments/{assignmentId}" && request.HTTPMethod == "DELETE":
		return handleDeleteAssignment(ctx, request, claims)

	// Bulk operations
	case request.Resource == "/assignments/bulk" && request.HTTPMethod == "POST":
		return handleCreateBulkAssignments(ctx, request, claims)
	case request.Resource == "/assignments/transfer" && request.HTTPMethod == "POST":
		return handleTransferAssignments(ctx, request, claims)

	// User-specific endpoints
	case request.Resource == "/users/{userId}/assignments" && request.HTTPMethod == "GET":
		return handleGetUserAssignments(ctx, request, claims)
	case request.Resource == "/users/{userId}/assignments/active" && request.HTTPMethod == "GET":
		return handleGetUserActiveAssignments(ctx, request, claims)
	case request.Resource == "/users/{userId}/contexts/{contextType}" && request.HTTPMethod == "GET":
		return handleGetUserContexts(ctx, request, claims)

	// Context-specific endpoints
	case request.Resource == "/contexts/{contextType}/{contextId}/assignments" && request.HTTPMethod == "GET":
		return handleGetContextAssignments(ctx, request, claims)

	// Permission checking
	case request.Resource == "/permissions/check" && request.HTTPMethod == "POST":
		return handleCheckPermission(ctx, request, claims)

	default:
		logger.WithFields(logrus.Fields{
			"method":    request.HTTPMethod,
			"resource":  request.Resource,
			"operation": "Handler",
		}).Warn("Endpoint not found")
		return api.ErrorResponse(http.StatusNotFound, "Endpoint not found", logger), nil
	}
}

// handleCreateAssignment handles POST /assignments
func handleCreateAssignment(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	var createRequest models.CreateAssignmentRequest
	if err := api.ParseJSONBody(request.Body, &createRequest); err != nil {
		logger.WithError(err).Error("Invalid request body for create assignment")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	userID := claims.UserID
	assignment, err := assignmentRepository.CreateAssignment(ctx, &createRequest, userID)
	if err != nil {
		logger.WithError(err).Error("Failed to create assignment")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create assignment", logger), nil
	}

	return api.SuccessResponse(http.StatusCreated, assignment, logger), nil
}

// handleGetAssignments handles GET /assignments with query filters
func handleGetAssignments(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	filters := &models.AssignmentFilters{}

	// Parse query parameters into filters
	if userID := request.QueryStringParameters["user_id"]; userID != "" {
		if id, err := strconv.ParseInt(userID, 10, 64); err == nil {
			filters.UserID = &id
		}
	}

	if roleID := request.QueryStringParameters["role_id"]; roleID != "" {
		if id, err := strconv.ParseInt(roleID, 10, 64); err == nil {
			filters.RoleID = &id
		}
	}

	if contextType := request.QueryStringParameters["context_type"]; contextType != "" {
		filters.ContextType = contextType
	}

	if contextID := request.QueryStringParameters["context_id"]; contextID != "" {
		if id, err := strconv.ParseInt(contextID, 10, 64); err == nil {
			filters.ContextID = &id
		}
	}

	if isPrimary := request.QueryStringParameters["is_primary"]; isPrimary != "" {
		if primary, err := strconv.ParseBool(isPrimary); err == nil {
			filters.IsPrimary = &primary
		}
	}

	if isActive := request.QueryStringParameters["is_active"]; isActive != "" {
		if active, err := strconv.ParseBool(isActive); err == nil {
			filters.IsActive = &active
		}
	}

	if tradeType := request.QueryStringParameters["trade_type"]; tradeType != "" {
		filters.TradeType = tradeType
	}

	if page := request.QueryStringParameters["page"]; page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			filters.Page = p
		}
	}

	if pageSize := request.QueryStringParameters["page_size"]; pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			filters.PageSize = ps
		}
	}

	// Set organization filter from JWT
	filters.OrganizationID = &claims.OrgID

	assignments, err := assignmentRepository.GetAssignments(ctx, filters, claims.OrgID)
	if err != nil {
		logger.WithError(err).Error("Failed to get assignments")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get assignments", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, assignments, logger), nil
}

// handleGetAssignment handles GET /assignments/{assignmentId}
func handleGetAssignment(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	assignmentID, err := strconv.ParseInt(request.PathParameters["assignmentId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid assignment ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid assignment ID", logger), nil
	}

	assignment, err := assignmentRepository.GetAssignment(ctx, assignmentID, claims.OrgID)
	if err != nil {
		if err.Error() == "assignment not found" {
			return api.ErrorResponse(http.StatusNotFound, "Assignment not found", logger), nil
		}
		logger.WithError(err).Error("Failed to get assignment")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get assignment", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, assignment, logger), nil
}

// handleUpdateAssignment handles PUT /assignments/{assignmentId}
func handleUpdateAssignment(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	assignmentID, err := strconv.ParseInt(request.PathParameters["assignmentId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid assignment ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid assignment ID", logger), nil
	}

	var updateRequest models.UpdateAssignmentRequest
	if err := api.ParseJSONBody(request.Body, &updateRequest); err != nil {
		logger.WithError(err).Error("Invalid request body for update assignment")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	userID := claims.UserID
	assignment, err := assignmentRepository.UpdateAssignment(ctx, assignmentID, &updateRequest, userID)
	if err != nil {
		if err.Error() == "assignment not found" {
			return api.ErrorResponse(http.StatusNotFound, "Assignment not found", logger), nil
		}
		logger.WithError(err).Error("Failed to update assignment")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update assignment", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, assignment, logger), nil
}

// handleDeleteAssignment handles DELETE /assignments/{assignmentId}
func handleDeleteAssignment(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	assignmentID, err := strconv.ParseInt(request.PathParameters["assignmentId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid assignment ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid assignment ID", logger), nil
	}

	userID := claims.UserID
	err = assignmentRepository.DeleteAssignment(ctx, assignmentID, userID)
	if err != nil {
		if err.Error() == "assignment not found" {
			return api.ErrorResponse(http.StatusNotFound, "Assignment not found", logger), nil
		}
		logger.WithError(err).Error("Failed to delete assignment")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to delete assignment", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, map[string]string{"message": "Assignment deleted successfully"}, logger), nil
}

// handleCreateBulkAssignments handles POST /assignments/bulk
func handleCreateBulkAssignments(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	var bulkRequest models.BulkAssignmentRequest
	if err := api.ParseJSONBody(request.Body, &bulkRequest); err != nil {
		logger.WithError(err).Error("Invalid request body for bulk assignment")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	userID := claims.UserID
	assignments, err := assignmentRepository.CreateBulkAssignments(ctx, &bulkRequest, userID)
	if err != nil {
		logger.WithError(err).Error("Failed to create bulk assignments")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create bulk assignments", logger), nil
	}

	response := map[string]interface{}{
		"message":     "Bulk assignments created successfully",
		"count":       len(assignments),
		"assignments": assignments,
	}

	return api.SuccessResponse(http.StatusCreated, response, logger), nil
}

// handleTransferAssignments handles POST /assignments/transfer
func handleTransferAssignments(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	var transferRequest models.AssignmentTransferRequest
	if err := api.ParseJSONBody(request.Body, &transferRequest); err != nil {
		logger.WithError(err).Error("Invalid request body for transfer assignments")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	userID := claims.UserID
	err := assignmentRepository.TransferAssignments(ctx, &transferRequest, userID)
	if err != nil {
		logger.WithError(err).Error("Failed to transfer assignments")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to transfer assignments", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, map[string]string{"message": "Assignments transferred successfully"}, logger), nil
}

// handleGetUserAssignments handles GET /users/{userId}/assignments
func handleGetUserAssignments(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	userID, err := strconv.ParseInt(request.PathParameters["userId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid user ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid user ID", logger), nil
	}

	userAssignments, err := assignmentRepository.GetUserAssignments(ctx, userID, claims.OrgID)
	if err != nil {
		logger.WithError(err).Error("Failed to get user assignments")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get user assignments", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, userAssignments, logger), nil
}

// handleGetUserActiveAssignments handles GET /users/{userId}/assignments/active
func handleGetUserActiveAssignments(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	userID, err := strconv.ParseInt(request.PathParameters["userId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid user ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid user ID", logger), nil
	}

	activeAssignments, err := assignmentRepository.GetActiveAssignments(ctx, userID, claims.OrgID)
	if err != nil {
		logger.WithError(err).Error("Failed to get user active assignments")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get user active assignments", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, activeAssignments, logger), nil
}

// handleGetUserContexts handles GET /users/{userId}/contexts/{contextType}
func handleGetUserContexts(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	userID, err := strconv.ParseInt(request.PathParameters["userId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid user ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid user ID", logger), nil
	}

	contextType := request.PathParameters["contextType"]
	if contextType == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Context type is required", logger), nil
	}

	contextIDs, err := assignmentRepository.GetUserContexts(ctx, userID, contextType, claims.OrgID)
	if err != nil {
		logger.WithError(err).Error("Failed to get user contexts")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get user contexts", logger), nil
	}

	response := map[string]interface{}{
		"user_id":      userID,
		"context_type": contextType,
		"context_ids":  contextIDs,
		"count":        len(contextIDs),
	}

	return api.SuccessResponse(http.StatusOK, response, logger), nil
}

// handleGetContextAssignments handles GET /contexts/{contextType}/{contextId}/assignments
func handleGetContextAssignments(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	contextType := request.PathParameters["contextType"]
	if contextType == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Context type is required", logger), nil
	}

	contextID, err := strconv.ParseInt(request.PathParameters["contextId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid context ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid context ID", logger), nil
	}

	contextAssignments, err := assignmentRepository.GetContextAssignments(ctx, contextType, contextID, claims.OrgID)
	if err != nil {
		logger.WithError(err).Error("Failed to get context assignments")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get context assignments", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, contextAssignments, logger), nil
}

// handleCheckPermission handles POST /permissions/check
func handleCheckPermission(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	var permissionRequest models.PermissionCheckRequest
	if err := api.ParseJSONBody(request.Body, &permissionRequest); err != nil {
		logger.WithError(err).Error("Invalid request body for permission check")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	permissionResponse, err := assignmentRepository.CheckPermission(ctx, &permissionRequest, claims.OrgID)
	if err != nil {
		logger.WithError(err).Error("Failed to check permission")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to check permission", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, permissionResponse, logger), nil
}

// setupPostgresSQLClient initializes the PostgreSQL database connection and repository
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

	// Initialize assignment repository with database connection and logger
	assignmentRepository = &data.AssignmentDao{
		DB:     sqlDB,
		Logger: logger,
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.WithField("operation", "setupPostgresSQLClient").Debug("PostgreSQL client initialized successfully")
	}
	return nil
}

// main is the Lambda function entry point
func main() {
	lambda.Start(Handler)
}

// init initializes the Lambda function during cold start
func init() {
	var err error

	// Parse environment variables for runtime configuration
	isLocal, _ = strconv.ParseBool(os.Getenv("IS_LOCAL"))

	// Initialize structured logging
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Set log level
	if os.Getenv("LOG_LEVEL") == "DEBUG" {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.ErrorLevel)
	}

	logger.WithField("operation", "init").Error("Initializing Assignment Management Lambda")

	// Initialize AWS SSM Parameter Store client for configuration management
	ssmClient := clients.NewSSMClient(isLocal)
	ssmRepository = &data.SSMDao{
		SSM:    ssmClient,
		Logger: logger,
	}

	// Retrieve all required configuration parameters from SSM Parameter Store
	ssmParams, err = ssmRepository.GetParameters()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"operation": "init",
			"error":     err.Error(),
		}).Fatal("Error while getting SSM params from parameter store")
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.WithFields(logrus.Fields{
			"operation":    "init",
			"params_count": len(ssmParams),
		}).Debug("Retrieved SSM parameters")
	}

	// Initialize PostgreSQL database connection using credentials from SSM
	err = setupPostgresSQLClient(ssmParams)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"operation": "init",
			"error":     err.Error(),
		}).Fatal("Error setting up PostgreSQL client")
	}

	logger.WithField("operation", "init").Error("Assignment Management Lambda initialization completed successfully")
}