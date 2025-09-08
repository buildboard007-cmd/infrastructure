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
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"
)

// Global variables for Lambda cold start optimization
var (
	logger          *logrus.Logger
	isLocal         bool
	ssmRepository   data.SSMRepository
	ssmParams       map[string]string
	sqlDB           *sql.DB
	issueRepository data.IssueRepository
)

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.WithFields(logrus.Fields{
		"operation": "Handler",
		"method":    request.HTTPMethod,
		"path":      request.Path,
		"resource":  request.Resource,
	}).Info("Issue management request received")

	// Extract claims from JWT token via API Gateway authorizer
	claims, err := auth.ExtractClaimsFromRequest(request)
	if err != nil {
		logger.WithError(err).Error("Authentication failed")
		return api.ErrorResponse(http.StatusUnauthorized, "Authentication failed", logger), nil
	}

	// Handle different routes
	switch request.HTTPMethod {
	case http.MethodPost:
		// POST /issues - Create new issue (project info in request body)
		if request.Resource == "/issues" {
			return handleCreateIssue(ctx, claims.UserID, claims.OrgID, request.Body), nil
		}
		return api.ErrorResponse(http.StatusNotFound, "Endpoint not found", logger), nil
		
	case http.MethodGet:
		// GET /projects/{projectId}/issues - List issues for project
		if strings.Contains(request.Resource, "/projects/{projectId}/issues") && request.PathParameters["issueId"] == "" {
			projectID, err := strconv.ParseInt(request.PathParameters["projectId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid project ID", logger), nil
			}
			return handleGetProjectIssues(ctx, projectID, claims.OrgID, request.QueryStringParameters), nil
		}
		
		// GET /issues/{issueId} - Get specific issue
		if strings.Contains(request.Resource, "/issues/{issueId}") {
			issueID, err := strconv.ParseInt(request.PathParameters["issueId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid issue ID", logger), nil
			}
			return handleGetIssue(ctx, issueID, claims.OrgID), nil
		}
		
		return api.ErrorResponse(http.StatusNotFound, "Endpoint not found", logger), nil
		
	case http.MethodPut:
		// PUT /issues/{issueId} - Update issue
		if strings.Contains(request.Resource, "/issues/{issueId}") {
			issueID, err := strconv.ParseInt(request.PathParameters["issueId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid issue ID", logger), nil
			}
			return handleUpdateIssue(ctx, issueID, claims.UserID, claims.OrgID, request.Body), nil
		}
		return api.ErrorResponse(http.StatusNotFound, "Endpoint not found", logger), nil
		
	case http.MethodPatch:
		// PATCH /issues/{issueId}/status - Update issue status
		if strings.Contains(request.Resource, "/issues/{issueId}/status") {
			issueID, err := strconv.ParseInt(request.PathParameters["issueId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid issue ID", logger), nil
			}
			return handleUpdateIssueStatus(ctx, issueID, claims.UserID, claims.OrgID, request.Body), nil
		}
		return api.ErrorResponse(http.StatusNotFound, "Endpoint not found", logger), nil
		
	case http.MethodDelete:
		// DELETE /issues/{issueId} - Delete issue
		if strings.Contains(request.Resource, "/issues/{issueId}") {
			issueID, err := strconv.ParseInt(request.PathParameters["issueId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid issue ID", logger), nil
			}
			return handleDeleteIssue(ctx, issueID, claims.UserID, claims.OrgID), nil
		}
		return api.ErrorResponse(http.StatusNotFound, "Endpoint not found", logger), nil
		
	default:
		return api.ErrorResponse(http.StatusMethodNotAllowed, "Method not allowed", logger), nil
	}
}

// handleCreateIssue handles POST /issues
func handleCreateIssue(ctx context.Context, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	var createReq models.CreateIssueRequest
	if err := json.Unmarshal([]byte(body), &createReq); err != nil {
		logger.WithError(err).Error("Failed to parse create issue request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Validate organization ID matches auth
	if createReq.OrganizationID != orgID {
		return api.ErrorResponse(http.StatusForbidden, "Organization ID does not match your organization", logger)
	}
	
	// Validate project belongs to org
	var projectOrgID int64
	err := sqlDB.QueryRowContext(ctx, `
		SELECT org_id FROM project.projects 
		WHERE id = $1 AND is_deleted = FALSE
	`, createReq.ProjectID).Scan(&projectOrgID)
	
	if err == sql.ErrNoRows {
		return api.ErrorResponse(http.StatusNotFound, "Project not found", logger)
	}
	if err != nil {
		logger.WithError(err).Error("Failed to validate project")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to validate project", logger)
	}
	if projectOrgID != orgID {
		return api.ErrorResponse(http.StatusForbidden, "Project does not belong to your organization", logger)
	}
	
	// Get location ID from project if not provided
	if createReq.LocationID == 0 {
		var locationID int64
		err = sqlDB.QueryRowContext(ctx, `
			SELECT location_id FROM project.projects 
			WHERE id = $1
		`, createReq.ProjectID).Scan(&locationID)
		if err != nil {
			logger.WithError(err).Warn("Failed to get location ID from project")
		} else {
			createReq.LocationID = locationID
		}
	}

	// Create issue
	issue, err := issueRepository.CreateIssue(ctx, createReq.ProjectID, userID, &createReq)
	if err != nil {
		logger.WithError(err).Error("Failed to create issue")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create issue", logger)
	}

	return api.SuccessResponse(http.StatusCreated, issue, logger)
}

// handleGetProjectIssues handles GET /projects/{projectId}/issues
func handleGetProjectIssues(ctx context.Context, projectID, orgID int64, filters map[string]string) events.APIGatewayProxyResponse {
	// Validate project belongs to org
	var projectOrgID int64
	err := sqlDB.QueryRowContext(ctx, `
		SELECT org_id FROM project.projects 
		WHERE id = $1 AND is_deleted = FALSE
	`, projectID).Scan(&projectOrgID)
	
	if err == sql.ErrNoRows {
		return api.ErrorResponse(http.StatusNotFound, "Project not found", logger)
	}
	if err != nil {
		logger.WithError(err).Error("Failed to validate project")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to validate project", logger)
	}
	if projectOrgID != orgID {
		return api.ErrorResponse(http.StatusForbidden, "Project does not belong to your organization", logger)
	}
	
	// Get issues
	issues, err := issueRepository.GetIssuesByProject(ctx, projectID, filters)
	if err != nil {
		logger.WithError(err).Error("Failed to get issues")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get issues", logger)
	}

	// Parse pagination params
	page := 1
	pageSize := 50
	if pageStr := filters["page"]; pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr := filters["page_size"]; pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	response := models.IssueListResponse{
		Issues:   issues,
		Total:    len(issues),
		Page:     page,
		PageSize: pageSize,
	}

	return api.SuccessResponse(http.StatusOK, response, logger)
}

// handleGetIssue handles GET /issues/{issueId}
func handleGetIssue(ctx context.Context, issueID, orgID int64) events.APIGatewayProxyResponse {
	issue, err := issueRepository.GetIssueByID(ctx, issueID)
	if err != nil {
		if err.Error() == "issue not found" {
			return api.ErrorResponse(http.StatusNotFound, "Issue not found", logger)
		}
		logger.WithError(err).Error("Failed to get issue")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get issue", logger)
	}

	// Validate issue belongs to org
	var projectOrgID int64
	err = sqlDB.QueryRowContext(ctx, `
		SELECT org_id FROM project.projects 
		WHERE id = $1 AND is_deleted = FALSE
	`, issue.ProjectID).Scan(&projectOrgID)
	
	if err != nil || projectOrgID != orgID {
		return api.ErrorResponse(http.StatusForbidden, "Issue does not belong to your organization", logger)
	}

	return api.SuccessResponse(http.StatusOK, issue, logger)
}

// handleUpdateIssue handles PUT /issues/{issueId}
func handleUpdateIssue(ctx context.Context, issueID, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	// First check if issue exists and belongs to org
	issue, err := issueRepository.GetIssueByID(ctx, issueID)
	if err != nil {
		if err.Error() == "issue not found" {
			return api.ErrorResponse(http.StatusNotFound, "Issue not found", logger)
		}
		logger.WithError(err).Error("Failed to get issue")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get issue", logger)
	}

	// Validate issue belongs to org
	var projectOrgID int64
	err = sqlDB.QueryRowContext(ctx, `
		SELECT org_id FROM project.projects 
		WHERE id = $1 AND is_deleted = FALSE
	`, issue.ProjectID).Scan(&projectOrgID)
	
	if err != nil || projectOrgID != orgID {
		return api.ErrorResponse(http.StatusForbidden, "Issue does not belong to your organization", logger)
	}

	var updateReq models.UpdateIssueRequest
	if err := json.Unmarshal([]byte(body), &updateReq); err != nil {
		logger.WithError(err).Error("Failed to parse update issue request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Update issue
	updatedIssue, err := issueRepository.UpdateIssue(ctx, issueID, userID, &updateReq)
	if err != nil {
		if err.Error() == "issue not found" {
			return api.ErrorResponse(http.StatusNotFound, "Issue not found", logger)
		}
		logger.WithError(err).Error("Failed to update issue")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update issue", logger)
	}

	return api.SuccessResponse(http.StatusOK, updatedIssue, logger)
}

// handleUpdateIssueStatus handles PATCH /issues/{issueId}/status
func handleUpdateIssueStatus(ctx context.Context, issueID, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	// First check if issue exists and belongs to org
	issue, err := issueRepository.GetIssueByID(ctx, issueID)
	if err != nil {
		if err.Error() == "issue not found" {
			return api.ErrorResponse(http.StatusNotFound, "Issue not found", logger)
		}
		logger.WithError(err).Error("Failed to get issue")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get issue", logger)
	}

	// Validate issue belongs to org
	var projectOrgID int64
	err = sqlDB.QueryRowContext(ctx, `
		SELECT org_id FROM project.projects 
		WHERE id = $1 AND is_deleted = FALSE
	`, issue.ProjectID).Scan(&projectOrgID)
	
	if err != nil || projectOrgID != orgID {
		return api.ErrorResponse(http.StatusForbidden, "Issue does not belong to your organization", logger)
	}

	// Parse status update request
	var statusReq struct {
		Status string `json:"status" binding:"required"`
	}
	if err := json.Unmarshal([]byte(body), &statusReq); err != nil {
		logger.WithError(err).Error("Failed to parse status update request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Validate status
	validStatuses := []string{
		models.IssueStatusOpen,
		models.IssueStatusInProgress,
		models.IssueStatusReadyForReview,
		models.IssueStatusClosed,
		models.IssueStatusRejected,
		models.IssueStatusOnHold,
	}
	
	isValid := false
	for _, s := range validStatuses {
		if statusReq.Status == s {
			isValid = true
			break
		}
	}
	
	if !isValid {
		return api.ErrorResponse(http.StatusBadRequest, "Invalid status value", logger)
	}

	// Update status
	err = issueRepository.UpdateIssueStatus(ctx, issueID, userID, statusReq.Status)
	if err != nil {
		if err.Error() == "issue not found" {
			return api.ErrorResponse(http.StatusNotFound, "Issue not found", logger)
		}
		logger.WithError(err).Error("Failed to update issue status")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update issue status", logger)
	}

	return api.SuccessResponse(http.StatusOK, map[string]string{
		"message": "Issue status updated successfully",
		"status":  statusReq.Status,
	}, logger)
}

// handleDeleteIssue handles DELETE /issues/{issueId}
func handleDeleteIssue(ctx context.Context, issueID, userID, orgID int64) events.APIGatewayProxyResponse {
	// First check if issue exists and belongs to org
	issue, err := issueRepository.GetIssueByID(ctx, issueID)
	if err != nil {
		if err.Error() == "issue not found" {
			return api.ErrorResponse(http.StatusNotFound, "Issue not found", logger)
		}
		logger.WithError(err).Error("Failed to get issue")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get issue", logger)
	}

	// Validate issue belongs to org
	var projectOrgID int64
	err = sqlDB.QueryRowContext(ctx, `
		SELECT org_id FROM project.projects 
		WHERE id = $1 AND is_deleted = FALSE
	`, issue.ProjectID).Scan(&projectOrgID)
	
	if err != nil || projectOrgID != orgID {
		return api.ErrorResponse(http.StatusForbidden, "Issue does not belong to your organization", logger)
	}

	// Delete issue
	err = issueRepository.DeleteIssue(ctx, issueID, userID)
	if err != nil {
		if err.Error() == "issue not found" {
			return api.ErrorResponse(http.StatusNotFound, "Issue not found", logger)
		}
		logger.WithError(err).Error("Failed to delete issue")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to delete issue", logger)
	}

	return api.SuccessResponse(http.StatusOK, map[string]string{"message": "Issue deleted successfully"}, logger)
}

// main is the Lambda function entry point
func main() {
	lambda.Start(Handler)
}

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

	logger.WithField("operation", "init").Info("Issue Management Lambda initialization completed successfully")
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

	// Initialize issue repository
	issueRepository = &data.IssueDao{
		DB:     sqlDB,
		Logger: logger,
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.WithField("operation", "setupPostgresSQLClient").Debug("PostgreSQL client initialized successfully")
	}
	return nil
}