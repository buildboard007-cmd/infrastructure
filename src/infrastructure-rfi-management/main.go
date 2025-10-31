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
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"
)

// Global variables for Lambda cold start optimization
var (
	logger        *logrus.Logger
	isLocal       bool
	ssmRepository data.SSMRepository
	ssmParams     map[string]string
	sqlDB         *sql.DB
	rfiRepository data.RFIRepository
)

// Handler processes API Gateway requests for RFI management operations
//
// SIMPLIFIED API ENDPOINTS (matching Issue Management pattern):
//
// Core CRUD Operations:
//   GET    /rfis/{rfiId}                    - Get RFI with all data (attachments, comments)
//   POST   /rfis                             - Create RFI
//   PUT    /rfis/{rfiId}                     - Update RFI
//
// List Query:
//   GET    /projects/{projectId}/rfis       - Get RFIs for project (with filters)
//
// Sub-resources:
//   POST   /rfis/{rfiId}/comments           - Add comment
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.WithFields(logrus.Fields{
		"method":      request.HTTPMethod,
		"path":        request.Path,
		"resource":    request.Resource,
		"path_params": request.PathParameters,
		"operation":   "Handler",
	}).Info("Processing RFI management request")

	// Extract claims from JWT token via API Gateway authorizer
	claims, err := auth.ExtractClaimsFromRequest(request)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
			"operation":  "Handler",
			"path":       request.Path,
			"method":     request.HTTPMethod,
		}).Error("Failed to extract claims from request - authentication failed")
		return api.ErrorResponse(http.StatusUnauthorized, fmt.Sprintf("Authentication failed: %v", err), logger), nil
	}

	// Validate required claims
	if claims.UserID == 0 {
		logger.WithFields(logrus.Fields{
			"operation": "Handler",
			"path":      request.Path,
		}).Error("Invalid claims: user_id is 0")
		return api.ErrorResponse(http.StatusUnauthorized, "Invalid authentication: missing user ID", logger), nil
	}

	if claims.OrgID == 0 {
		logger.WithFields(logrus.Fields{
			"operation": "Handler",
			"user_id":   claims.UserID,
			"path":      request.Path,
		}).Error("Invalid claims: org_id is 0")
		return api.ErrorResponse(http.StatusUnauthorized, "Invalid authentication: missing organization ID", logger), nil
	}

	logger.WithFields(logrus.Fields{
		"user_id":   claims.UserID,
		"org_id":    claims.OrgID,
		"email":     claims.Email,
		"operation": "Handler",
	}).Info("User authenticated successfully")

	// Route the request based on path and method
	switch {
	// GET /projects/{projectId}/rfis - List RFIs for project (simple, consistent with Issue API)
	case request.Resource == "/projects/{projectId}/rfis" && request.HTTPMethod == "GET":
		return handleGetProjectRFIs(ctx, request, claims)

	// GET /rfis/{rfiId} - Get single RFI
	case request.Resource == "/rfis/{rfiId}" && request.HTTPMethod == "GET":
		return handleGetRFI(ctx, request, claims)

	// POST /rfis - Create RFI
	case request.Resource == "/rfis" && request.HTTPMethod == "POST":
		return handleCreateRFI(ctx, request, claims)

	// PUT /rfis/{rfiId} - Update RFI
	case request.Resource == "/rfis/{rfiId}" && request.HTTPMethod == "PUT":
		return handleUpdateRFI(ctx, request, claims)

	// POST /rfis/{rfiId}/comments - Add comment
	case request.Resource == "/rfis/{rfiId}/comments" && request.HTTPMethod == "POST":
		return handleAddRFIComment(ctx, request, claims)

	// DEPRECATED: Context-based query (kept for backwards compatibility, will be removed)
	case request.Resource == "/contexts/{contextType}/{contextId}/rfis" && request.HTTPMethod == "GET":
		return handleGetContextRFIs(ctx, request, claims)

	default:
		logger.WithFields(logrus.Fields{
			"method":    request.HTTPMethod,
			"resource":  request.Resource,
			"path":      request.Path,
			"operation": "Handler",
		}).Warn("Endpoint not found - no matching route")
		return api.ErrorResponse(http.StatusNotFound, fmt.Sprintf("Endpoint not found: %s %s", request.HTTPMethod, request.Resource), logger), nil
	}
}

// handleCreateRFI handles POST /rfis
func handleCreateRFI(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	logger.WithFields(logrus.Fields{
		"body":      request.Body,
		"user_id":   claims.UserID,
		"org_id":    claims.OrgID,
		"operation": "handleCreateRFI",
	}).Info("Received create RFI request")

	// Validate request body is not empty
	if strings.TrimSpace(request.Body) == "" {
		logger.WithFields(logrus.Fields{
			"operation": "handleCreateRFI",
			"user_id":   claims.UserID,
		}).Error("Request body is empty")
		return api.ErrorResponse(http.StatusBadRequest, "Request body cannot be empty", logger), nil
	}

	// Parse request body
	var createReq models.CreateRFIRequest
	if err := api.ParseJSONBody(request.Body, &createReq); err != nil {
		logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
			"body":       request.Body,
			"operation":  "handleCreateRFI",
			"user_id":    claims.UserID,
		}).Error("Failed to parse JSON request body")
		return api.ErrorResponse(http.StatusBadRequest, fmt.Sprintf("Invalid JSON in request body: %v", err), logger), nil
	}

	// Validate required fields
	if createReq.ProjectID == 0 {
		logger.WithFields(logrus.Fields{
			"operation": "handleCreateRFI",
			"user_id":   claims.UserID,
		}).Error("Missing required field: project_id")
		return api.ErrorResponse(http.StatusBadRequest, "project_id is required and must be greater than 0", logger), nil
	}

	if createReq.LocationID == 0 {
		logger.WithFields(logrus.Fields{
			"operation":  "handleCreateRFI",
			"user_id":    claims.UserID,
			"project_id": createReq.ProjectID,
		}).Error("Missing required field: location_id")
		return api.ErrorResponse(http.StatusBadRequest, "location_id is required and must be greater than 0", logger), nil
	}

	if strings.TrimSpace(createReq.Subject) == "" {
		logger.WithFields(logrus.Fields{
			"operation":  "handleCreateRFI",
			"user_id":    claims.UserID,
			"project_id": createReq.ProjectID,
		}).Error("Missing required field: subject")
		return api.ErrorResponse(http.StatusBadRequest, "subject is required and cannot be empty", logger), nil
	}

	if strings.TrimSpace(createReq.Description) == "" {
		logger.WithFields(logrus.Fields{
			"operation":  "handleCreateRFI",
			"user_id":    claims.UserID,
			"project_id": createReq.ProjectID,
		}).Error("Missing required field: description")
		return api.ErrorResponse(http.StatusBadRequest, "description is required and cannot be empty", logger), nil
	}

	if strings.TrimSpace(createReq.Category) == "" {
		logger.WithFields(logrus.Fields{
			"operation":  "handleCreateRFI",
			"user_id":    claims.UserID,
			"project_id": createReq.ProjectID,
		}).Error("Missing required field: category")
		return api.ErrorResponse(http.StatusBadRequest, "category is required and cannot be empty", logger), nil
	}

	if strings.TrimSpace(createReq.Priority) == "" {
		logger.WithFields(logrus.Fields{
			"operation":  "handleCreateRFI",
			"user_id":    claims.UserID,
			"project_id": createReq.ProjectID,
		}).Error("Missing required field: priority")
		return api.ErrorResponse(http.StatusBadRequest, "priority is required and cannot be empty", logger), nil
	}

	logger.WithFields(logrus.Fields{
		"project_id":  createReq.ProjectID,
		"location_id": createReq.LocationID,
		"subject":     createReq.Subject,
		"category":    createReq.Category,
		"priority":    createReq.Priority,
		"status":      createReq.Status,
		"operation":   "handleCreateRFI",
		"user_id":     claims.UserID,
		"org_id":      claims.OrgID,
	}).Info("Request validation passed, creating RFI")

	// Create RFI via repository
	userID := claims.UserID
	createdRFI, err := rfiRepository.CreateRFI(ctx, createReq.ProjectID, userID, claims.OrgID, &createReq)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
			"project_id": createReq.ProjectID,
			"user_id":    userID,
			"org_id":     claims.OrgID,
			"operation":  "handleCreateRFI",
		}).Error("Repository failed to create RFI")

		// Return detailed error message for better debugging
		errorMsg := fmt.Sprintf("Failed to create RFI: %v", err)
		return api.ErrorResponse(http.StatusInternalServerError, errorMsg, logger), nil
	}

	// Validate created RFI is not nil
	if createdRFI == nil {
		logger.WithFields(logrus.Fields{
			"project_id": createReq.ProjectID,
			"user_id":    userID,
			"org_id":     claims.OrgID,
			"operation":  "handleCreateRFI",
		}).Error("Repository returned nil RFI after creation")
		return api.ErrorResponse(http.StatusInternalServerError, "RFI creation failed: repository returned nil", logger), nil
	}

	logger.WithFields(logrus.Fields{
		"rfi_id":     createdRFI.ID,
		"rfi_number": createdRFI.RFINumber,
		"status":     createdRFI.Status,
		"operation":  "handleCreateRFI",
		"user_id":    userID,
	}).Info("RFI created successfully")

	return api.SuccessResponse(http.StatusCreated, createdRFI, logger), nil
}

// handleGetRFI handles GET /rfis/{rfiId} - returns RFI with all attachments and comments
func handleGetRFI(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	// Extract and validate RFI ID
	rfiIDStr, exists := request.PathParameters["rfiId"]
	if !exists || strings.TrimSpace(rfiIDStr) == "" {
		logger.WithFields(logrus.Fields{
			"operation": "handleGetRFI",
			"user_id":   claims.UserID,
		}).Error("Missing rfiId in path parameters")
		return api.ErrorResponse(http.StatusBadRequest, "rfiId is required in path", logger), nil
	}

	rfiID, err := strconv.ParseInt(rfiIDStr, 10, 64)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
			"rfi_id_str": rfiIDStr,
			"operation":  "handleGetRFI",
			"user_id":    claims.UserID,
		}).Error("Failed to parse rfiId as integer")
		return api.ErrorResponse(http.StatusBadRequest, fmt.Sprintf("Invalid RFI ID format: %v", err), logger), nil
	}

	if rfiID <= 0 {
		logger.WithFields(logrus.Fields{
			"rfi_id":    rfiID,
			"operation": "handleGetRFI",
			"user_id":   claims.UserID,
		}).Error("RFI ID must be positive")
		return api.ErrorResponse(http.StatusBadRequest, "RFI ID must be greater than 0", logger), nil
	}

	logger.WithFields(logrus.Fields{
		"rfi_id":    rfiID,
		"operation": "handleGetRFI",
		"user_id":   claims.UserID,
		"org_id":    claims.OrgID,
	}).Info("Fetching RFI")

	// Fetch RFI from repository
	rfi, err := rfiRepository.GetRFI(ctx, rfiID)
	if err != nil {
		if err.Error() == "RFI not found" {
			logger.WithFields(logrus.Fields{
				"rfi_id":    rfiID,
				"operation": "handleGetRFI",
				"user_id":   claims.UserID,
			}).Warn("RFI not found in database")
			return api.ErrorResponse(http.StatusNotFound, "RFI not found", logger), nil
		}
		logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
			"rfi_id":     rfiID,
			"operation":  "handleGetRFI",
			"user_id":    claims.UserID,
		}).Error("Repository failed to fetch RFI")
		return api.ErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to get RFI: %v", err), logger), nil
	}

	// Validate fetched RFI
	if rfi == nil {
		logger.WithFields(logrus.Fields{
			"rfi_id":    rfiID,
			"operation": "handleGetRFI",
			"user_id":   claims.UserID,
		}).Error("Repository returned nil RFI without error")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get RFI: repository returned nil", logger), nil
	}

	// Verify organization access
	if rfi.OrgID != claims.OrgID {
		logger.WithFields(logrus.Fields{
			"rfi_id":      rfiID,
			"rfi_org_id":  rfi.OrgID,
			"user_org_id": claims.OrgID,
			"operation":   "handleGetRFI",
			"user_id":     claims.UserID,
		}).Warn("User attempted to access RFI from different organization")
		return api.ErrorResponse(http.StatusForbidden, "Access denied: RFI belongs to a different organization", logger), nil
	}

	// Fetch comments for RFI
	comments, err := rfiRepository.GetRFIComments(ctx, rfiID)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
			"rfi_id":     rfiID,
			"operation":  "handleGetRFI",
			"user_id":    claims.UserID,
		}).Warn("Failed to fetch RFI comments, continuing with empty comments")
		rfi.Comments = []models.RFIComment{}
	} else if comments == nil {
		rfi.Comments = []models.RFIComment{}
	} else {
		rfi.Comments = comments
	}

	// Fetch attachments for RFI
	attachments, err := rfiRepository.GetRFIAttachments(ctx, rfiID)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
			"rfi_id":     rfiID,
			"operation":  "handleGetRFI",
			"user_id":    claims.UserID,
		}).Warn("Failed to fetch RFI attachments, continuing with empty attachments")
		rfi.Attachments = []models.RFIAttachment{}
	} else if attachments == nil {
		rfi.Attachments = []models.RFIAttachment{}
	} else {
		rfi.Attachments = attachments
	}

	logger.WithFields(logrus.Fields{
		"rfi_id":           rfiID,
		"rfi_number":       rfi.RFINumber,
		"status":           rfi.Status,
		"comments_count":   len(rfi.Comments),
		"attachments_count": len(rfi.Attachments),
		"operation":        "handleGetRFI",
		"user_id":          claims.UserID,
	}).Info("RFI fetched successfully")

	return api.SuccessResponse(http.StatusOK, rfi, logger), nil
}

// handleUpdateRFI handles PUT /rfis/{rfiId} - supports action field for status changes
func handleUpdateRFI(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	// Extract and validate RFI ID
	rfiIDStr, exists := request.PathParameters["rfiId"]
	if !exists || strings.TrimSpace(rfiIDStr) == "" {
		logger.WithFields(logrus.Fields{
			"operation": "handleUpdateRFI",
			"user_id":   claims.UserID,
		}).Error("Missing rfiId in path parameters")
		return api.ErrorResponse(http.StatusBadRequest, "rfiId is required in path", logger), nil
	}

	rfiID, err := strconv.ParseInt(rfiIDStr, 10, 64)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
			"rfi_id_str": rfiIDStr,
			"operation":  "handleUpdateRFI",
			"user_id":    claims.UserID,
		}).Error("Failed to parse rfiId as integer")
		return api.ErrorResponse(http.StatusBadRequest, fmt.Sprintf("Invalid RFI ID format: %v", err), logger), nil
	}

	if rfiID <= 0 {
		logger.WithFields(logrus.Fields{
			"rfi_id":    rfiID,
			"operation": "handleUpdateRFI",
			"user_id":   claims.UserID,
		}).Error("RFI ID must be positive")
		return api.ErrorResponse(http.StatusBadRequest, "RFI ID must be greater than 0", logger), nil
	}

	// Validate request body is not empty
	if strings.TrimSpace(request.Body) == "" {
		logger.WithFields(logrus.Fields{
			"operation": "handleUpdateRFI",
			"rfi_id":    rfiID,
			"user_id":   claims.UserID,
		}).Error("Request body is empty")
		return api.ErrorResponse(http.StatusBadRequest, "Request body cannot be empty", logger), nil
	}

	// Parse request body
	var updateReq models.UpdateRFIRequest
	if err := api.ParseJSONBody(request.Body, &updateReq); err != nil {
		logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
			"body":       request.Body,
			"rfi_id":     rfiID,
			"operation":  "handleUpdateRFI",
			"user_id":    claims.UserID,
		}).Error("Failed to parse JSON request body")
		return api.ErrorResponse(http.StatusBadRequest, fmt.Sprintf("Invalid JSON in request body: %v", err), logger), nil
	}

	logger.WithFields(logrus.Fields{
		"rfi_id":    rfiID,
		"status":    updateReq.Status,
		"operation": "handleUpdateRFI",
		"user_id":   claims.UserID,
		"org_id":    claims.OrgID,
	}).Info("Updating RFI")

	// Update RFI via repository
	userID := claims.UserID
	updatedRFI, err := rfiRepository.UpdateRFI(ctx, rfiID, userID, claims.OrgID, &updateReq)
	if err != nil {
		if strings.Contains(err.Error(), "RFI not found") || strings.Contains(err.Error(), "not found") {
			logger.WithFields(logrus.Fields{
				"error":     err.Error(),
				"rfi_id":    rfiID,
				"operation": "handleUpdateRFI",
				"user_id":   userID,
			}).Warn("RFI not found during update")
			return api.ErrorResponse(http.StatusNotFound, "RFI not found", logger), nil
		}
		if strings.Contains(err.Error(), "does not belong") || strings.Contains(err.Error(), "organization") {
			logger.WithFields(logrus.Fields{
				"error":     err.Error(),
				"rfi_id":    rfiID,
				"operation": "handleUpdateRFI",
				"user_id":   userID,
				"org_id":    claims.OrgID,
			}).Warn("User attempted to update RFI from different organization")
			return api.ErrorResponse(http.StatusForbidden, "Access denied: RFI belongs to a different organization", logger), nil
		}
		logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
			"rfi_id":     rfiID,
			"user_id":    userID,
			"org_id":     claims.OrgID,
			"operation":  "handleUpdateRFI",
		}).Error("Repository failed to update RFI")

		// Return detailed error message for better debugging
		errorMsg := fmt.Sprintf("Failed to update RFI: %v", err)
		return api.ErrorResponse(http.StatusInternalServerError, errorMsg, logger), nil
	}

	// Validate updated RFI
	if updatedRFI == nil {
		logger.WithFields(logrus.Fields{
			"rfi_id":    rfiID,
			"user_id":   userID,
			"org_id":    claims.OrgID,
			"operation": "handleUpdateRFI",
		}).Error("Repository returned nil RFI after update")
		return api.ErrorResponse(http.StatusInternalServerError, "RFI update failed: repository returned nil", logger), nil
	}

	logger.WithFields(logrus.Fields{
		"rfi_id":     rfiID,
		"rfi_number": updatedRFI.RFINumber,
		"status":     updatedRFI.Status,
		"operation":  "handleUpdateRFI",
		"user_id":    userID,
	}).Info("RFI updated successfully")

	return api.SuccessResponse(http.StatusOK, updatedRFI, logger), nil
}

// handleGetProjectRFIs handles GET /projects/{projectId}/rfis - Simple, consistent with Issue API
func handleGetProjectRFIs(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	// Extract and validate project ID
	projectIDStr, exists := request.PathParameters["projectId"]
	if !exists || strings.TrimSpace(projectIDStr) == "" {
		logger.WithFields(logrus.Fields{
			"operation": "handleGetProjectRFIs",
			"user_id":   claims.UserID,
		}).Error("Missing projectId in path parameters")
		return api.ErrorResponse(http.StatusBadRequest, "projectId is required in path", logger), nil
	}

	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":          err.Error(),
			"error_type":     fmt.Sprintf("%T", err),
			"project_id_str": projectIDStr,
			"operation":      "handleGetProjectRFIs",
			"user_id":        claims.UserID,
		}).Error("Failed to parse projectId as integer")
		return api.ErrorResponse(http.StatusBadRequest, fmt.Sprintf("Invalid project ID format: %v", err), logger), nil
	}

	if projectID <= 0 {
		logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"operation":  "handleGetProjectRFIs",
			"user_id":    claims.UserID,
		}).Error("Project ID must be positive")
		return api.ErrorResponse(http.StatusBadRequest, "Project ID must be greater than 0", logger), nil
	}

	// Get query string filters
	filters := request.QueryStringParameters
	if filters == nil {
		filters = make(map[string]string)
	}

	logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"filters":    filters,
		"operation":  "handleGetProjectRFIs",
		"user_id":    claims.UserID,
		"org_id":     claims.OrgID,
	}).Info("Fetching project RFIs")

	// Fetch RFIs from repository
	rfis, err := rfiRepository.GetRFIsByProject(ctx, projectID, filters)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
			"project_id": projectID,
			"filters":    filters,
			"operation":  "handleGetProjectRFIs",
			"user_id":    claims.UserID,
		}).Error("Repository failed to fetch project RFIs")
		return api.ErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to get RFIs: %v", err), logger), nil
	}

	// Ensure we return an empty array instead of null
	if rfis == nil {
		rfis = []models.RFIResponse{}
	}

	logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"count":      len(rfis),
		"operation":  "handleGetProjectRFIs",
		"user_id":    claims.UserID,
	}).Info("Project RFIs fetched successfully")

	return api.SuccessResponse(http.StatusOK, rfis, logger), nil
}

// handleGetContextRFIs handles GET /contexts/{contextType}/{contextId}/rfis
// DEPRECATED: This endpoint is kept for backwards compatibility only
// Use GET /projects/{projectId}/rfis instead
func handleGetContextRFIs(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	// Extract context type
	contextType, exists := request.PathParameters["contextType"]
	if !exists || strings.TrimSpace(contextType) == "" {
		logger.WithFields(logrus.Fields{
			"operation": "handleGetContextRFIs",
			"user_id":   claims.UserID,
		}).Error("Missing contextType in path parameters")
		return api.ErrorResponse(http.StatusBadRequest, "contextType is required in path", logger), nil
	}

	// Extract and validate context ID
	contextIDStr, exists := request.PathParameters["contextId"]
	if !exists || strings.TrimSpace(contextIDStr) == "" {
		logger.WithFields(logrus.Fields{
			"operation":    "handleGetContextRFIs",
			"context_type": contextType,
			"user_id":      claims.UserID,
		}).Error("Missing contextId in path parameters")
		return api.ErrorResponse(http.StatusBadRequest, "contextId is required in path", logger), nil
	}

	contextID, err := strconv.ParseInt(contextIDStr, 10, 64)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":          err.Error(),
			"error_type":     fmt.Sprintf("%T", err),
			"context_id_str": contextIDStr,
			"context_type":   contextType,
			"operation":      "handleGetContextRFIs",
			"user_id":        claims.UserID,
		}).Error("Failed to parse contextId as integer")
		return api.ErrorResponse(http.StatusBadRequest, fmt.Sprintf("Invalid context ID format: %v", err), logger), nil
	}

	if contextID <= 0 {
		logger.WithFields(logrus.Fields{
			"context_id":   contextID,
			"context_type": contextType,
			"operation":    "handleGetContextRFIs",
			"user_id":      claims.UserID,
		}).Error("Context ID must be positive")
		return api.ErrorResponse(http.StatusBadRequest, "Context ID must be greater than 0", logger), nil
	}

	// For now, only support project context
	if contextType != "project" {
		logger.WithFields(logrus.Fields{
			"context_type": contextType,
			"context_id":   contextID,
			"operation":    "handleGetContextRFIs",
			"user_id":      claims.UserID,
		}).Warn("Unsupported context type requested")
		return api.ErrorResponse(http.StatusBadRequest, fmt.Sprintf("Only project context is supported, received: %s", contextType), logger), nil
	}

	// Get query string filters
	filters := request.QueryStringParameters
	if filters == nil {
		filters = make(map[string]string)
	}

	logger.WithFields(logrus.Fields{
		"context_type": contextType,
		"context_id":   contextID,
		"filters":      filters,
		"operation":    "handleGetContextRFIs",
		"user_id":      claims.UserID,
		"org_id":       claims.OrgID,
	}).Info("Fetching context RFIs (deprecated endpoint)")

	// Fetch RFIs from repository
	rfis, err := rfiRepository.GetRFIsByProject(ctx, contextID, filters)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":        err.Error(),
			"error_type":   fmt.Sprintf("%T", err),
			"context_type": contextType,
			"context_id":   contextID,
			"filters":      filters,
			"operation":    "handleGetContextRFIs",
			"user_id":      claims.UserID,
		}).Error("Repository failed to fetch context RFIs")
		return api.ErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to get RFIs: %v", err), logger), nil
	}

	// Ensure we return an empty array instead of null
	if rfis == nil {
		rfis = []models.RFIResponse{}
	}

	response := map[string]interface{}{
		"context_type": contextType,
		"context_id":   contextID,
		"rfis":         rfis,
	}

	logger.WithFields(logrus.Fields{
		"context_type": contextType,
		"context_id":   contextID,
		"count":        len(rfis),
		"operation":    "handleGetContextRFIs",
		"user_id":      claims.UserID,
	}).Info("Context RFIs fetched successfully")

	return api.SuccessResponse(http.StatusOK, response, logger), nil
}

// handleAddRFIComment handles POST /rfis/{rfiId}/comments
func handleAddRFIComment(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	// Extract and validate RFI ID
	rfiIDStr, exists := request.PathParameters["rfiId"]
	if !exists || strings.TrimSpace(rfiIDStr) == "" {
		logger.WithFields(logrus.Fields{
			"operation": "handleAddRFIComment",
			"user_id":   claims.UserID,
		}).Error("Missing rfiId in path parameters")
		return api.ErrorResponse(http.StatusBadRequest, "rfiId is required in path", logger), nil
	}

	rfiID, err := strconv.ParseInt(rfiIDStr, 10, 64)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
			"rfi_id_str": rfiIDStr,
			"operation":  "handleAddRFIComment",
			"user_id":    claims.UserID,
		}).Error("Failed to parse rfiId as integer")
		return api.ErrorResponse(http.StatusBadRequest, fmt.Sprintf("Invalid RFI ID format: %v", err), logger), nil
	}

	if rfiID <= 0 {
		logger.WithFields(logrus.Fields{
			"rfi_id":    rfiID,
			"operation": "handleAddRFIComment",
			"user_id":   claims.UserID,
		}).Error("RFI ID must be positive")
		return api.ErrorResponse(http.StatusBadRequest, "RFI ID must be greater than 0", logger), nil
	}

	// Validate that RFI exists and belongs to user's organization
	rfi, err := rfiRepository.GetRFI(ctx, rfiID)
	if err != nil {
		if strings.Contains(err.Error(), "RFI not found") || strings.Contains(err.Error(), "not found") {
			logger.WithFields(logrus.Fields{
				"error":     err.Error(),
				"rfi_id":    rfiID,
				"operation": "handleAddRFIComment",
				"user_id":   claims.UserID,
			}).Warn("RFI not found when adding comment")
			return api.ErrorResponse(http.StatusNotFound, "RFI not found", logger), nil
		}
		logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
			"rfi_id":     rfiID,
			"operation":  "handleAddRFIComment",
			"user_id":    claims.UserID,
		}).Error("Repository failed to fetch RFI for comment validation")
		return api.ErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to validate RFI: %v", err), logger), nil
	}

	// Validate RFI is not nil
	if rfi == nil {
		logger.WithFields(logrus.Fields{
			"rfi_id":    rfiID,
			"operation": "handleAddRFIComment",
			"user_id":   claims.UserID,
		}).Error("Repository returned nil RFI without error")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to validate RFI: repository returned nil", logger), nil
	}

	// Verify organization access
	if rfi.OrgID != claims.OrgID {
		logger.WithFields(logrus.Fields{
			"rfi_id":      rfiID,
			"rfi_org_id":  rfi.OrgID,
			"user_org_id": claims.OrgID,
			"operation":   "handleAddRFIComment",
			"user_id":     claims.UserID,
		}).Warn("User attempted to add comment to RFI from different organization")
		return api.ErrorResponse(http.StatusForbidden, "Access denied: RFI belongs to a different organization", logger), nil
	}

	// Validate request body is not empty
	if strings.TrimSpace(request.Body) == "" {
		logger.WithFields(logrus.Fields{
			"operation": "handleAddRFIComment",
			"rfi_id":    rfiID,
			"user_id":   claims.UserID,
		}).Error("Request body is empty")
		return api.ErrorResponse(http.StatusBadRequest, "Request body cannot be empty", logger), nil
	}

	// Parse request body
	var req models.CreateRFICommentRequest
	if err := api.ParseJSONBody(request.Body, &req); err != nil {
		logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
			"body":       request.Body,
			"rfi_id":     rfiID,
			"operation":  "handleAddRFIComment",
			"user_id":    claims.UserID,
		}).Error("Failed to parse JSON request body")
		return api.ErrorResponse(http.StatusBadRequest, fmt.Sprintf("Invalid JSON in request body: %v", err), logger), nil
	}

	// Validate required fields
	if strings.TrimSpace(req.Comment) == "" {
		logger.WithFields(logrus.Fields{
			"operation": "handleAddRFIComment",
			"rfi_id":    rfiID,
			"user_id":   claims.UserID,
		}).Error("Missing required field: comment")
		return api.ErrorResponse(http.StatusBadRequest, "comment is required and cannot be empty", logger), nil
	}

	logger.WithFields(logrus.Fields{
		"rfi_id":           rfiID,
		"comment_length":   len(req.Comment),
		"attachment_count": len(req.AttachmentIDs),
		"operation":        "handleAddRFIComment",
		"user_id":          claims.UserID,
		"org_id":           claims.OrgID,
	}).Info("Adding comment to RFI")

	// Add comment via repository
	userID := claims.UserID
	comment, err := rfiRepository.AddRFIComment(ctx, rfiID, userID, &req)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
			"rfi_id":     rfiID,
			"user_id":    userID,
			"operation":  "handleAddRFIComment",
		}).Error("Repository failed to add RFI comment")
		return api.ErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to add comment: %v", err), logger), nil
	}

	// Validate comment is not nil
	if comment == nil {
		logger.WithFields(logrus.Fields{
			"rfi_id":    rfiID,
			"user_id":   userID,
			"operation": "handleAddRFIComment",
		}).Error("Repository returned nil comment after creation")
		return api.ErrorResponse(http.StatusInternalServerError, "Comment creation failed: repository returned nil", logger), nil
	}

	logger.WithFields(logrus.Fields{
		"rfi_id":     rfiID,
		"comment_id": comment.ID,
		"operation":  "handleAddRFIComment",
		"user_id":    userID,
	}).Info("RFI comment added successfully")

	return api.SuccessResponse(http.StatusCreated, comment, logger), nil
}

func init() {
	var err error

	isLocal = parseIsLocal()

	// Logger Setup
	logger = setupLogger(isLocal)

	logger.WithField("operation", "init").Info("Initializing RFI management service")

	// Initialize AWS SSM Parameter Store client
	ssmClient := clients.NewSSMClient(isLocal)
	if ssmClient == nil {
		logger.WithField("operation", "init").Fatal("Failed to create SSM client: client is nil")
	}

	ssmRepository = &data.SSMDao{
		SSM:    ssmClient,
		Logger: logger,
	}

	// Retrieve all required configuration parameters from SSM
	ssmParams, err = ssmRepository.GetParameters()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"operation":  "init",
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
		}).Fatal("Failed to retrieve SSM parameters from parameter store")
	}

	if ssmParams == nil {
		logger.WithField("operation", "init").Fatal("SSM parameters map is nil")
	}

	logger.WithFields(logrus.Fields{
		"operation":    "init",
		"params_count": len(ssmParams),
	}).Info("Retrieved SSM parameters successfully")

	// Initialize PostgreSQL database connection
	err = setupPostgresSQLClient(ssmParams)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"operation":  "init",
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
		}).Fatal("Failed to setup PostgreSQL client")
	}

	// Validate database connection
	if sqlDB == nil {
		logger.WithField("operation", "init").Fatal("PostgreSQL DB connection is nil after setup")
	}

	// Validate repository
	if rfiRepository == nil {
		logger.WithField("operation", "init").Fatal("RFI repository is nil after initialization")
	}

	logger.Info("RFI management service initialized successfully")
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

	// Validate required parameters exist
	requiredParams := []string{
		constants.DATABASE_RDS_ENDPOINT,
		constants.DATABASE_PORT,
		constants.DATABASE_NAME,
		constants.DATABASE_USERNAME,
		constants.DATABASE_PASSWORD,
		constants.SSL_MODE,
	}

	for _, param := range requiredParams {
		if _, exists := ssmParams[param]; !exists {
			return fmt.Errorf("missing required SSM parameter: %s", param)
		}
		if strings.TrimSpace(ssmParams[param]) == "" {
			return fmt.Errorf("SSM parameter %s is empty", param)
		}
	}

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

	if sqlDB == nil {
		return fmt.Errorf("PostgreSQL client creation returned nil without error")
	}

	// Test database connection
	if err = sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	// Initialize RFI repository
	rfiRepository = &data.RFIDao{
		DB:     sqlDB,
		Logger: logger,
	}

	if rfiRepository == nil {
		return fmt.Errorf("failed to initialize RFI repository: repository is nil")
	}

	logger.WithField("operation", "setupPostgresSQLClient").Info("PostgreSQL client and RFI repository initialized successfully")

	return nil
}
