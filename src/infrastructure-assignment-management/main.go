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
//
// SIMPLIFIED API ENDPOINTS:
//
// Core CRUD Operations:
//   GET    /assignments/{id}                                 - Get single assignment
//   POST   /assignments                                      - Create assignment
//   PUT    /assignments/{id}                                 - Update assignment
//   DELETE /assignments/{id}                                 - Delete assignment
//
// Project Team Query:
//   GET    /contexts/{contextType}/{contextId}/assignments  - Get team for project/location
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
	// Core assignment CRUD operations
	case request.Resource == "/assignments/{assignmentId}" && request.HTTPMethod == "GET":
		return handleGetAssignment(ctx, request, claims)
	case request.Resource == "/assignments" && request.HTTPMethod == "POST":
		return handleCreateAssignment(ctx, request, claims)
	case request.Resource == "/assignments/{assignmentId}" && request.HTTPMethod == "PUT":
		return handleUpdateAssignment(ctx, request, claims)
	case request.Resource == "/assignments/{assignmentId}" && request.HTTPMethod == "DELETE":
		return handleDeleteAssignment(ctx, request, claims)

	// Project team endpoint
	case request.Resource == "/contexts/{contextType}/{contextId}/assignments" && request.HTTPMethod == "GET":
		return handleGetContextAssignments(ctx, request, claims)

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