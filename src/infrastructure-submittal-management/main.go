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
	"infrastructure/lib/util"
	"net/http"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"
)

// Global variables for Lambda cold start optimization
var (
	logger               *logrus.Logger
	isLocal              bool
	ssmRepository        data.SSMRepository
	ssmParams            map[string]string
	sqlDB                *sql.DB
	submittalRepository  data.SubmittalRepository
)

// Handler processes API Gateway requests for Submittal management operations
//
// CONSOLIDATED API ENDPOINTS (10 total):
//
// Core CRUD Operations:
//   GET    /submittals/{id}                                    - Get submittal with all data (attachments, reviews)
//   POST   /submittals                                         - Create submittal
//   PUT    /submittals/{id}                                    - Update submittal (including soft delete)
//
// Context Query:
//   GET    /contexts/{contextType}/{contextId}/submittals     - Get submittals for project
//
// Workflow Operations:
//   POST   /submittals/{id}/workflow                          - Execute workflow action
//
// Statistics & Export:
//   GET    /contexts/{contextType}/{contextId}/submittals/stats - Get submittal statistics
//   GET    /contexts/{contextType}/{contextId}/submittals/export - Export submittals
//
// File Management:
//   POST   /submittals/{id}/attachments                       - Add attachment
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.WithFields(logrus.Fields{
		"method":      request.HTTPMethod,
		"path":        request.Path,
		"resource":    request.Resource,
		"path_params": request.PathParameters,
		"operation":   "Handler",
	}).Debug("Processing submittal management request")

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
	// Core submittal CRUD operations
	case request.Resource == "/submittals/{submittalId}" && request.HTTPMethod == "GET":
		return handleGetSubmittal(ctx, request, claims)
	case request.Resource == "/submittals" && request.HTTPMethod == "POST":
		return handleCreateSubmittal(ctx, request, claims)
	case request.Resource == "/submittals/{submittalId}" && request.HTTPMethod == "PUT":
		return handleUpdateSubmittal(ctx, request, claims)

	// Context-based submittal queries
	case request.Resource == "/contexts/{contextType}/{contextId}/submittals" && request.HTTPMethod == "GET":
		return handleGetContextSubmittals(ctx, request, claims)

	// Workflow operations
	case request.Resource == "/submittals/{submittalId}/workflow" && request.HTTPMethod == "POST":
		return handleWorkflowAction(ctx, request, claims)

	// Statistics and export
	case request.Resource == "/contexts/{contextType}/{contextId}/submittals/stats" && request.HTTPMethod == "GET":
		return handleGetSubmittalStats(ctx, request, claims)
	case request.Resource == "/contexts/{contextType}/{contextId}/submittals/export" && request.HTTPMethod == "GET":
		return handleExportSubmittals(ctx, request, claims)

	// Submittal attachments now handled by centralized attachment management service

	default:
		logger.WithFields(logrus.Fields{
			"method":    request.HTTPMethod,
			"resource":  request.Resource,
			"operation": "Handler",
		}).Warn("Endpoint not found")
		return api.ErrorResponse(http.StatusNotFound, "Endpoint not found", logger), nil
	}
}

// handleCreateSubmittal handles POST /submittals
func handleCreateSubmittal(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	var createReq models.CreateSubmittalRequest
	if err := api.ParseJSONBody(request.Body, &createReq); err != nil {
		logger.WithError(err).Error("Invalid request body for create submittal")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	// Validate required fields
	if createReq.Title == "" || createReq.SubmittalType == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Missing required fields: title and submittal_type are required", logger), nil
	}

	userID := claims.UserID
	createdSubmittal, err := submittalRepository.CreateSubmittal(ctx, createReq.ProjectID, userID, claims.OrgID, &createReq)
	if err != nil {
		logger.WithError(err).Error("Failed to create submittal")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create submittal", logger), nil
	}

	return api.SuccessResponse(http.StatusCreated, createdSubmittal, logger), nil
}

// handleGetSubmittal handles GET /submittals/{submittalId} - returns submittal with all attachments
func handleGetSubmittal(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	submittalID, err := strconv.ParseInt(request.PathParameters["submittalId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid submittal ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid submittal ID", logger), nil
	}

	submittal, err := submittalRepository.GetSubmittal(ctx, submittalID)
	if err != nil {
		if err.Error() == "submittal not found" {
			return api.ErrorResponse(http.StatusNotFound, "Submittal not found", logger), nil
		}
		logger.WithError(err).Error("Failed to get submittal")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get submittal", logger), nil
	}

	// Enrich with attachments
	attachments, _ := submittalRepository.GetSubmittalAttachments(ctx, submittalID)
	if attachments == nil {
		submittal.Attachments = []models.SubmittalAttachment{}
	} else {
		submittal.Attachments = attachments
	}

	return api.SuccessResponse(http.StatusOK, submittal, logger), nil
}

// handleUpdateSubmittal handles PUT /submittals/{submittalId}
func handleUpdateSubmittal(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	submittalID, err := strconv.ParseInt(request.PathParameters["submittalId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid submittal ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid submittal ID", logger), nil
	}

	var updateReq models.UpdateSubmittalRequest
	if err := api.ParseJSONBody(request.Body, &updateReq); err != nil {
		logger.WithError(err).Error("Invalid request body for update submittal")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	userID := claims.UserID
	updatedSubmittal, err := submittalRepository.UpdateSubmittal(ctx, submittalID, userID, claims.OrgID, &updateReq)
	if err != nil {
		if err.Error() == "submittal not found" {
			return api.ErrorResponse(http.StatusNotFound, "Submittal not found", logger), nil
		}
		logger.WithError(err).Error("Failed to update submittal")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update submittal", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, updatedSubmittal, logger), nil
}


// handleGetContextSubmittals handles GET /contexts/{contextType}/{contextId}/submittals
func handleGetContextSubmittals(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	contextType := request.PathParameters["contextType"]
	contextID, err := strconv.ParseInt(request.PathParameters["contextId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid context ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid context ID", logger), nil
	}

	// For now, only support project context
	if contextType != "project" {
		return api.ErrorResponse(http.StatusBadRequest, "Only project context is supported", logger), nil
	}

	filters := request.QueryStringParameters
	if filters == nil {
		filters = make(map[string]string)
	}

	submittals, err := submittalRepository.GetSubmittalsByProject(ctx, contextID, filters)
	if err != nil {
		logger.WithError(err).Error("Failed to get context submittals")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get submittals", logger), nil
	}

	// Build paginated response
	page := 1
	if pageStr := filters["page"]; pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	pageSize := 20
	if limitStr := filters["limit"]; limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			pageSize = parsedLimit
		}
	}

	response := models.SubmittalListResponse{
		Submittals: submittals,
		TotalCount: len(submittals),
		Page:       page,
		PageSize:   pageSize,
		HasNext:    len(submittals) == pageSize, // Simplified logic
		HasPrev:    page > 1,
	}

	return api.SuccessResponse(http.StatusOK, response, logger), nil
}

// handleWorkflowAction handles POST /submittals/{submittalId}/workflow
func handleWorkflowAction(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	submittalID, err := strconv.ParseInt(request.PathParameters["submittalId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid submittal ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid submittal ID", logger), nil
	}

	var action models.SubmittalWorkflowAction
	if err := api.ParseJSONBody(request.Body, &action); err != nil {
		logger.WithError(err).Error("Invalid request body for workflow action")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	// Validate action
	validActions := []string{
		models.WorkflowActionSubmitForReview,
		models.WorkflowActionApprove,
		models.WorkflowActionApproveAsNoted,
		models.WorkflowActionReviseResubmit,
		models.WorkflowActionReject,
		models.WorkflowActionMarkForInformation,
	}

	valid := false
	for _, validAction := range validActions {
		if action.Action == validAction {
			valid = true
			break
		}
	}

	if !valid {
		return api.ErrorResponse(http.StatusBadRequest, "Invalid workflow action", logger), nil
	}

	userID := claims.UserID
	updatedSubmittal, err := submittalRepository.ExecuteWorkflowAction(ctx, submittalID, userID, &action)
	if err != nil {
		logger.WithError(err).Error("Failed to execute workflow action")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to execute workflow action", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, updatedSubmittal, logger), nil
}

// handleGetSubmittalStats handles GET /contexts/{contextType}/{contextId}/submittals/stats
func handleGetSubmittalStats(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	contextType := request.PathParameters["contextType"]
	contextID, err := strconv.ParseInt(request.PathParameters["contextId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid context ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid context ID", logger), nil
	}

	// For now, only support project context
	if contextType != "project" {
		return api.ErrorResponse(http.StatusBadRequest, "Only project context is supported", logger), nil
	}

	stats, err := submittalRepository.GetSubmittalStats(ctx, contextID)
	if err != nil {
		logger.WithError(err).Error("Failed to get submittal stats")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get stats", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, stats, logger), nil
}

// handleExportSubmittals handles GET /contexts/{contextType}/{contextId}/submittals/export
func handleExportSubmittals(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	// This is a placeholder for export functionality
	// In a real implementation, you would generate CSV/Excel/PDF exports
	return api.ErrorResponse(http.StatusNotImplemented, "Export functionality not implemented", logger), nil
}

// Submittal attachment handler removed - now handled by centralized attachment management service
// Removed function: handleAddSubmittalAttachment


func init() {
	var err error

	isLocal = parseIsLocal()

	// Logger Setup
	logger = setupLogger(isLocal)

	// Initialize AWS SSM Parameter Store client
	ssmClient := clients.NewSSMClient(isLocal)
	ssmRepository = &data.SSMDao{
		SSM:    ssmClient,
		Logger: logger,
	}

	// Retrieve all required configuration parameters from SSM
	ssmParams, err = ssmRepository.GetParameters()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"operation": "init",
			"error":     err.Error(),
		}).Fatal("Error while getting SSM params from parameter store")
	}

	logger.WithFields(logrus.Fields{
		"operation":    "init",
		"params_count": len(ssmParams),
	}).Debug("Retrieved SSM parameters")

	// Initialize PostgreSQL database connection
	err = setupPostgresSQLClient(ssmParams)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"operation": "init",
			"error":     err.Error(),
		}).Fatal("Error setting up PostgreSQL client")
	}

	logger.Info("Submittal management service initialized successfully")
}

func main() {
	lambda.Start(Handler)
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

	// Initialize submittal repository
	submittalRepository = &data.SubmittalDao{
		DB:     sqlDB,
		Logger: logger,
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.WithField("operation", "setupPostgresSQLClient").Debug("PostgreSQL client initialized successfully")
	}

	return nil
}