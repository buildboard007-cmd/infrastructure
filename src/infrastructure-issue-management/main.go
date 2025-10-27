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
		// POST /issues/{issueId}/comments - Add comment to issue
		if strings.Contains(request.Resource, "/issues/{issueId}/comments") {
			issueID, err := strconv.ParseInt(request.PathParameters["issueId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid issue ID", logger), nil
			}
			return handleCreateComment(ctx, issueID, claims.UserID, claims.OrgID, request.Body), nil
		}

		// POST /issues - Create new issue (unified structure, orgID from JWT)
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
			// Ensure filters map is not nil
			filters := request.QueryStringParameters
			if filters == nil {
				filters = make(map[string]string)
			}
			return handleGetProjectIssues(ctx, projectID, claims.OrgID, filters), nil
		}

		// GET /issues/{issueId}/comments - Get comments for issue
		if strings.Contains(request.Resource, "/issues/{issueId}/comments") {
			issueID, err := strconv.ParseInt(request.PathParameters["issueId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid issue ID", logger), nil
			}
			return handleGetIssueComments(ctx, issueID, claims.OrgID), nil
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
		// PUT /issues/{issueId} - Update issue (unified structure, orgID from JWT)
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

// handleCreateIssue handles POST /issues with unified structure and JWT-based orgID
func handleCreateIssue(ctx context.Context, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	// Parse unified request structure
	var createReq models.CreateIssueRequest
	if err := json.Unmarshal([]byte(body), &createReq); err != nil {
		logger.WithError(err).Error("Failed to parse create issue request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Extract project_id from request (should be in request body)
	projectID := createReq.ProjectID
	if projectID == 0 {
		return api.ErrorResponse(http.StatusBadRequest, "Project ID is required", logger)
	}

	// Validate required fields from flatter structure
	if createReq.Title == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Title is required", logger)
	}
	if createReq.Description == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Description is required", logger)
	}
	if createReq.Priority == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Priority is required", logger)
	}
	if createReq.AssignedTo == 0 {
		return api.ErrorResponse(http.StatusBadRequest, "Assigned to is required", logger)
	}
	if createReq.DueDate == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Due date is required", logger)
	}

	// Validate assigned_to user exists and belongs to organization
	var assignedUserOrgID int64
	err := sqlDB.QueryRowContext(ctx, `
		SELECT org_id FROM iam.users
		WHERE id = $1 AND is_deleted = FALSE
	`, createReq.AssignedTo).Scan(&assignedUserOrgID)

	if err == sql.ErrNoRows {
		return api.ErrorResponse(http.StatusBadRequest, fmt.Sprintf("Invalid assigned_to user ID. User %d does not exist.", createReq.AssignedTo), logger)
	}
	if err != nil {
		logger.WithError(err).Error("Failed to validate assigned user")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to validate assigned user", logger)
	}
	if assignedUserOrgID != orgID {
		return api.ErrorResponse(http.StatusBadRequest, fmt.Sprintf("Invalid assigned_to user ID. User %d does not belong to your organization.", createReq.AssignedTo), logger)
	}

	// Create issue using repository with orgID from JWT (validation happens in repository)
	issue, err := issueRepository.CreateIssue(ctx, projectID, userID, orgID, &createReq)
	if err != nil {
		logger.WithError(err).Error("Failed to create issue")
		// Check for specific database errors to provide better error messages
		if strings.Contains(err.Error(), "project does not belong to your organization") {
			return api.ErrorResponse(http.StatusBadRequest, "Invalid project ID. Project does not belong to your organization.", logger)
		}
		if strings.Contains(err.Error(), "foreign key constraint") {
			return api.ErrorResponse(http.StatusBadRequest, "Invalid reference data provided", logger)
		}
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

	// Fetch attachments for the issue from issue_attachments table
	attachments, _ := issueRepository.GetIssueAttachments(ctx, issueID)
	if attachments == nil {
		issue.Attachments = []models.IssueAttachment{}
	} else {
		issue.Attachments = attachments
	}

	// Fetch comments and activity log for the issue
	comments, err := issueRepository.GetIssueComments(ctx, issueID)
	if err != nil {
		logger.WithError(err).Warn("Failed to fetch comments for issue")
		issue.Comments = []models.IssueComment{}
	} else {
		issue.Comments = comments
	}

	return api.SuccessResponse(http.StatusOK, issue, logger)
}

// handleUpdateIssue handles PUT /issues/{issueId}
func handleUpdateIssue(ctx context.Context, issueID, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	// Get current issue state for activity logging
	oldIssue, err := issueRepository.GetIssueByID(ctx, issueID)
	if err != nil {
		if err.Error() == "issue not found" {
			return api.ErrorResponse(http.StatusNotFound, "Issue not found", logger)
		}
		logger.WithError(err).Error("Failed to get issue")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get issue", logger)
	}

	// Parse unified request structure
	var updateReq models.UpdateIssueRequest
	if err := json.Unmarshal([]byte(body), &updateReq); err != nil {
		logger.WithError(err).Error("Failed to parse update issue request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Update issue using repository with orgID from JWT (validation happens in repository)
	updatedIssue, err := issueRepository.UpdateIssue(ctx, issueID, userID, orgID, &updateReq)
	if err != nil {
		if err.Error() == "issue not found" {
			return api.ErrorResponse(http.StatusNotFound, "Issue not found", logger)
		}
		if err.Error() == "issue does not belong to your organization" {
			return api.ErrorResponse(http.StatusForbidden, "Issue does not belong to your organization", logger)
		}
		logger.WithError(err).Error("Failed to update issue")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update issue", logger)
	}

	// Log status change activity
	if updateReq.Status != "" && oldIssue.Status != updateReq.Status {
		activityMsg := fmt.Sprintf("Status changed from %s to %s", oldIssue.Status, updateReq.Status)
		err := issueRepository.CreateActivityLog(ctx, issueID, userID, activityMsg, oldIssue.Status, updateReq.Status)
		if err != nil {
			logger.WithError(err).Warn("Failed to log status change activity")
		}
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

	// Store old status for activity logging
	oldStatus := issue.Status

	// Update status
	err = issueRepository.UpdateIssueStatus(ctx, issueID, userID, statusReq.Status)
	if err != nil {
		if err.Error() == "issue not found" {
			return api.ErrorResponse(http.StatusNotFound, "Issue not found", logger)
		}
		logger.WithError(err).Error("Failed to update issue status")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update issue status", logger)
	}

	// Log status change activity
	if oldStatus != statusReq.Status {
		activityMsg := fmt.Sprintf("Status changed from %s to %s", oldStatus, statusReq.Status)
		err := issueRepository.CreateActivityLog(ctx, issueID, userID, activityMsg, oldStatus, statusReq.Status)
		if err != nil {
			logger.WithError(err).Warn("Failed to log status change activity")
		}
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

// handleCreateComment handles POST /issues/{issueId}/comments
func handleCreateComment(ctx context.Context, issueID, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	// First validate that issue exists and belongs to user's organization
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

	// Parse comment request
	var commentReq models.CreateCommentRequest
	if err := json.Unmarshal([]byte(body), &commentReq); err != nil {
		logger.WithError(err).Error("Failed to parse create comment request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Validate required fields
	if commentReq.Comment == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Comment is required", logger)
	}

	// Create comment
	comment, err := issueRepository.CreateComment(ctx, issueID, userID, &commentReq)
	if err != nil {
		logger.WithError(err).Error("Failed to create comment")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create comment", logger)
	}

	return api.SuccessResponse(http.StatusCreated, comment, logger)
}

// handleGetIssueComments handles GET /issues/{issueId}/comments
func handleGetIssueComments(ctx context.Context, issueID, orgID int64) events.APIGatewayProxyResponse {
	// First validate that issue exists and belongs to user's organization
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

	// Get comments
	comments, err := issueRepository.GetIssueComments(ctx, issueID)
	if err != nil {
		logger.WithError(err).Error("Failed to get comments")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get comments", logger)
	}

	// Return comments array directly
	return api.SuccessResponse(http.StatusOK, comments, logger)
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