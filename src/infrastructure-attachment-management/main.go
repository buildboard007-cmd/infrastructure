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
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"
)

// Global variables for Lambda cold start optimization
var (
	logger                *logrus.Logger
	isLocal               bool
	ssmRepository         data.SSMRepository
	ssmParams             map[string]string
	sqlDB                 *sql.DB
	attachmentRepository  data.AttachmentRepository
	s3Client              clients.S3ClientInterface
)

// Handler processes API Gateway requests for attachment management operations
//
// CENTRALIZED ATTACHMENT API ENDPOINTS:
//
// Core Operations:
//   POST   /attachments/upload-url                     - Generate presigned upload URL
//   POST   /attachments/confirm                        - Confirm upload completion
//   GET    /attachments/{id}                           - Get attachment metadata
//   GET    /attachments/{id}/download-url              - Generate presigned download URL
//   DELETE /attachments/{id}                           - Soft delete attachment
//
// Entity Queries:
//   GET    /entities/{type}/{id}/attachments           - List attachments for entity
//
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.WithFields(logrus.Fields{
		"method":      request.HTTPMethod,
		"path":        request.Path,
		"resource":    request.Resource,
		"path_params": request.PathParameters,
		"operation":   "Handler",
	}).Debug("Processing attachment management request")

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
	// Upload operations
	case request.Resource == "/attachments/upload-url" && request.HTTPMethod == "POST":
		return handleGenerateUploadURL(ctx, request, claims)
	case request.Resource == "/attachments/confirm" && request.HTTPMethod == "POST":
		return handleConfirmUpload(ctx, request, claims)

	// Download operations
	case request.Resource == "/attachments/{id}" && request.HTTPMethod == "GET":
		return handleGetAttachment(ctx, request, claims)
	case request.Resource == "/attachments/{id}/download-url" && request.HTTPMethod == "GET":
		return handleGenerateDownloadURL(ctx, request, claims)

	// Delete operations
	case request.Resource == "/attachments/{id}" && request.HTTPMethod == "DELETE":
		return handleDeleteAttachment(ctx, request, claims)

	// Entity-based queries
	case request.Resource == "/entities/{type}/{id}/attachments" && request.HTTPMethod == "GET":
		return handleGetEntityAttachments(ctx, request, claims)

	default:
		logger.WithFields(logrus.Fields{
			"method":    request.HTTPMethod,
			"resource":  request.Resource,
			"operation": "Handler",
		}).Warn("Endpoint not found")
		return api.ErrorResponse(http.StatusNotFound, "Endpoint not found", logger), nil
	}
}

// handleGenerateUploadURL handles POST /attachments/upload-url
func handleGenerateUploadURL(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	var uploadReq models.AttachmentUploadRequest
	if err := api.ParseJSONBody(request.Body, &uploadReq); err != nil {
		logger.WithError(err).Error("Invalid request body for upload URL")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	// Set org ID from claims
	uploadReq.OrgID = claims.OrgID

	// Validate required fields
	// For issue_comment and rfi_comment, entity_id can be 0 (will be updated after comment creation)
	if uploadReq.EntityType == "" || uploadReq.ProjectID == 0 || uploadReq.LocationID == 0 || uploadReq.FileName == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Missing required fields", logger), nil
	}

	// For non-comment entity types, entity_id must be > 0
	if uploadReq.EntityType != models.EntityTypeIssueComment && uploadReq.EntityType != models.EntityTypeRFIComment && uploadReq.EntityID == 0 {
		return api.ErrorResponse(http.StatusBadRequest, "entity_id is required for this entity type", logger), nil
	}

	// Validate file type
	if !models.ValidateFileType(uploadReq.FileName) {
		return api.ErrorResponse(http.StatusBadRequest, "File type not allowed", logger), nil
	}

	// Validate entity type is supported
	if !isValidEntityType(uploadReq.EntityType) {
		return api.ErrorResponse(http.StatusBadRequest, "Invalid entity type", logger), nil
	}

	// Validate entity access (entity exists, belongs to project, project belongs to org and location)
	if uploadReq.EntityType != models.EntityTypeIssueComment && uploadReq.EntityType != models.EntityTypeRFIComment {
		statusCode, errMsg := validateEntityAccess(ctx, uploadReq.EntityType, uploadReq.EntityID, uploadReq.ProjectID, uploadReq.LocationID, uploadReq.OrgID)
		if errMsg != "" {
			return api.ErrorResponse(statusCode, errMsg, logger), nil
		}
	} else {
		// For issue_comment/rfi_comment without entity_id, just validate project
		statusCode, errMsg := validateProjectAccess(ctx, uploadReq.ProjectID, uploadReq.LocationID, uploadReq.OrgID)
		if errMsg != "" {
			return api.ErrorResponse(statusCode, errMsg, logger), nil
		}
	}

	// Generate S3 key
	s3Key := uploadReq.GenerateS3Key()
	if s3Key == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Failed to generate S3 key", logger), nil
	}

	// Create attachment record in database
	attachment := &models.Attachment{
		EntityType:     uploadReq.EntityType,
		EntityID:       uploadReq.EntityID,
		ProjectID:      uploadReq.ProjectID,
		LocationID:     uploadReq.LocationID,
		OrgID:          uploadReq.OrgID,
		FileName:       uploadReq.FileName,
		FilePath:       s3Key,
		FileSize:       &uploadReq.FileSize,
		AttachmentType: uploadReq.AttachmentType,
		UploadedBy:     claims.UserID,
		CreatedBy:      claims.UserID,
		UpdatedBy:      claims.UserID,
	}

	// Set file type and MIME type
	fileType := models.GetMimeType(uploadReq.FileName)
	attachment.FileType = &fileType
	attachment.MimeType = &fileType

	createdAttachment, err := attachmentRepository.CreateAttachment(ctx, attachment)
	if err != nil {
		logger.WithError(err).Error("Failed to create attachment record")
		// Parse specific database errors
		if strings.Contains(err.Error(), "violates foreign key constraint") {
			return api.ErrorResponse(http.StatusBadRequest, "Invalid reference: Entity or project does not exist", logger), nil
		}
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create attachment", logger), nil
	}

	// Generate presigned upload URL (15 minutes expiry)
	uploadURL, err := s3Client.GenerateUploadURL(s3Key, 15*time.Minute)
	if err != nil {
		logger.WithError(err).Error("Failed to generate upload URL")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to generate upload URL", logger), nil
	}

	response := models.AttachmentUploadResponse{
		AttachmentID: createdAttachment.ID,
		UploadURL:    uploadURL,
		S3Key:        s3Key,
		ExpiresAt:    time.Now().Add(15 * time.Minute).Format(time.RFC3339),
	}

	return api.SuccessResponse(http.StatusOK, response, logger), nil
}

// handleConfirmUpload handles POST /attachments/confirm
func handleConfirmUpload(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	var confirmReq models.AttachmentConfirmRequest
	if err := api.ParseJSONBody(request.Body, &confirmReq); err != nil {
		logger.WithError(err).Error("Invalid request body for confirm upload")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	// Note: In a more complete implementation, we would:
	// 1. Verify the file was actually uploaded to S3
	// 2. Update the upload status in the database
	// 3. Possibly trigger virus scanning

	logger.WithFields(logrus.Fields{
		"attachment_id": confirmReq.AttachmentID,
		"user_id":       claims.UserID,
	}).Info("Upload confirmed")

	return api.SuccessResponse(http.StatusOK, map[string]string{"status": "confirmed"}, logger), nil
}

// handleGetAttachment handles GET /attachments/{id}
func handleGetAttachment(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	attachmentIDStr := request.PathParameters["id"]
	attachmentID, err := strconv.ParseInt(attachmentIDStr, 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid attachment ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid attachment ID", logger), nil
	}

	// Get entity type from query parameter (required for dynamic table access)
	entityType := request.QueryStringParameters["entity_type"]
	if entityType == "" {
		return api.ErrorResponse(http.StatusBadRequest, "entity_type query parameter is required", logger), nil
	}

	// Verify access
	hasAccess, err := attachmentRepository.VerifyAttachmentAccess(ctx, attachmentID, entityType, claims.OrgID)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "unsupported entity type") {
			return api.ErrorResponse(http.StatusBadRequest, errMsg, logger), nil
		}
		if strings.Contains(errMsg, "attachment not found") {
			return api.ErrorResponse(http.StatusNotFound, "Attachment not found", logger), nil
		}
		if strings.Contains(errMsg, "access denied") {
			return api.ErrorResponse(http.StatusForbidden, "Access denied to this attachment", logger), nil
		}
		if strings.Contains(errMsg, "database error") {
			logger.WithError(err).Error("Database error while verifying attachment access")
			return api.ErrorResponse(http.StatusInternalServerError, "Database error occurred", logger), nil
		}
		// Fallback for any other unexpected errors
		logger.WithError(err).Error("Unexpected error while verifying attachment access")
		return api.ErrorResponse(http.StatusInternalServerError, "An unexpected error occurred", logger), nil
	}
	if !hasAccess {
		return api.ErrorResponse(http.StatusForbidden, "Access denied to this attachment", logger), nil
	}

	attachment, err := attachmentRepository.GetAttachment(ctx, attachmentID, entityType)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return api.ErrorResponse(http.StatusNotFound, "Attachment not found", logger), nil
		}
		logger.WithError(err).Error("Failed to get attachment")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get attachment", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, attachment, logger), nil
}

// handleGenerateDownloadURL handles GET /attachments/{id}/download-url
func handleGenerateDownloadURL(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	attachmentIDStr := request.PathParameters["id"]
	attachmentID, err := strconv.ParseInt(attachmentIDStr, 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid attachment ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid attachment ID", logger), nil
	}

	// Get entity type from query parameter
	entityType := request.QueryStringParameters["entity_type"]
	if entityType == "" {
		return api.ErrorResponse(http.StatusBadRequest, "entity_type query parameter is required", logger), nil
	}

	// Verify access
	hasAccess, err := attachmentRepository.VerifyAttachmentAccess(ctx, attachmentID, entityType, claims.OrgID)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "unsupported entity type") {
			return api.ErrorResponse(http.StatusBadRequest, errMsg, logger), nil
		}
		if strings.Contains(errMsg, "attachment not found") {
			return api.ErrorResponse(http.StatusNotFound, "Attachment not found", logger), nil
		}
		if strings.Contains(errMsg, "access denied") {
			return api.ErrorResponse(http.StatusForbidden, "Access denied to this attachment", logger), nil
		}
		if strings.Contains(errMsg, "database error") {
			logger.WithError(err).Error("Database error while verifying attachment access")
			return api.ErrorResponse(http.StatusInternalServerError, "Database error occurred", logger), nil
		}
		// Fallback for any other unexpected errors
		logger.WithError(err).Error("Unexpected error while verifying attachment access")
		return api.ErrorResponse(http.StatusInternalServerError, "An unexpected error occurred", logger), nil
	}
	if !hasAccess {
		return api.ErrorResponse(http.StatusForbidden, "Access denied to this attachment", logger), nil
	}

	attachment, err := attachmentRepository.GetAttachment(ctx, attachmentID, entityType)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return api.ErrorResponse(http.StatusNotFound, "Attachment not found", logger), nil
		}
		logger.WithError(err).Error("Failed to get attachment")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get attachment", logger), nil
	}

	// Generate presigned download URL (60 minutes expiry)
	downloadURL, err := s3Client.GenerateDownloadURL(attachment.FilePath, 60*time.Minute)
	if err != nil {
		logger.WithError(err).Error("Failed to generate download URL")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to generate download URL", logger), nil
	}

	response := models.AttachmentDownloadResponse{
		DownloadURL: downloadURL,
		FileName:    attachment.FileName,
		FileSize:    attachment.FileSize,
		ExpiresAt:   time.Now().Add(60 * time.Minute).Format(time.RFC3339),
	}

	return api.SuccessResponse(http.StatusOK, response, logger), nil
}

// handleDeleteAttachment handles DELETE /attachments/{id}
func handleDeleteAttachment(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	attachmentIDStr := request.PathParameters["id"]
	attachmentID, err := strconv.ParseInt(attachmentIDStr, 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid attachment ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid attachment ID", logger), nil
	}

	// Get entity type from query parameter
	entityType := request.QueryStringParameters["entity_type"]
	if entityType == "" {
		return api.ErrorResponse(http.StatusBadRequest, "entity_type query parameter is required", logger), nil
	}

	// Verify access
	hasAccess, err := attachmentRepository.VerifyAttachmentAccess(ctx, attachmentID, entityType, claims.OrgID)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "unsupported entity type") {
			return api.ErrorResponse(http.StatusBadRequest, errMsg, logger), nil
		}
		if strings.Contains(errMsg, "attachment not found") {
			return api.ErrorResponse(http.StatusNotFound, "Attachment not found", logger), nil
		}
		if strings.Contains(errMsg, "access denied") {
			return api.ErrorResponse(http.StatusForbidden, "Access denied to this attachment", logger), nil
		}
		if strings.Contains(errMsg, "database error") {
			logger.WithError(err).Error("Database error while verifying attachment access")
			return api.ErrorResponse(http.StatusInternalServerError, "Database error occurred", logger), nil
		}
		// Fallback for any other unexpected errors
		logger.WithError(err).Error("Unexpected error while verifying attachment access")
		return api.ErrorResponse(http.StatusInternalServerError, "An unexpected error occurred", logger), nil
	}
	if !hasAccess {
		return api.ErrorResponse(http.StatusForbidden, "Access denied to this attachment", logger), nil
	}

	err = attachmentRepository.SoftDeleteAttachment(ctx, attachmentID, entityType, claims.UserID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return api.ErrorResponse(http.StatusNotFound, "Attachment not found", logger), nil
		}
		logger.WithError(err).Error("Failed to delete attachment")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to delete attachment", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, map[string]string{"status": "deleted"}, logger), nil
}

// handleGetEntityAttachments handles GET /entities/{type}/{id}/attachments
func handleGetEntityAttachments(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	entityType := request.PathParameters["type"]
	entityIDStr := request.PathParameters["id"]

	entityID, err := strconv.ParseInt(entityIDStr, 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid entity ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid entity ID", logger), nil
	}

	// Validate entity type
	if !isValidEntityType(entityType) {
		return api.ErrorResponse(http.StatusBadRequest, "Invalid entity type", logger), nil
	}

	filters := request.QueryStringParameters
	if filters == nil {
		filters = make(map[string]string)
	}

	attachments, err := attachmentRepository.GetAttachmentsByEntity(ctx, entityType, entityID, filters)
	if err != nil {
		logger.WithError(err).Error("Failed to get entity attachments")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get attachments", logger), nil
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

	response := models.AttachmentListResponse{
		Attachments: attachments,
		TotalCount:  len(attachments),
		Page:        page,
		PageSize:    pageSize,
		HasNext:     len(attachments) == pageSize, // Simplified logic
		HasPrev:     page > 1,
	}

	return api.SuccessResponse(http.StatusOK, response, logger), nil
}

// Helper function to validate entity type
func isValidEntityType(entityType string) bool {
	validTypes := []string{
		models.EntityTypeProject,
		models.EntityTypeIssue,
		models.EntityTypeRFI,
		models.EntityTypeSubmittal,
		models.EntityTypeIssueComment,
		models.EntityTypeRFIComment,
	}

	for _, validType := range validTypes {
		if entityType == validType {
			return true
		}
	}
	return false
}

// validateProjectAccess validates that project exists, belongs to org, and optionally belongs to location
// Returns (statusCode, errorMessage) - errorMessage is empty string if validation passes
func validateProjectAccess(ctx context.Context, projectID, locationID, orgID int64) (int, string) {
	var projectOrgID, projectLocationID int64

	err := sqlDB.QueryRowContext(ctx, `
		SELECT org_id, location_id FROM project.projects
		WHERE id = $1 AND is_deleted = FALSE
	`, projectID).Scan(&projectOrgID, &projectLocationID)

	if err == sql.ErrNoRows {
		return http.StatusNotFound, "Project not found"
	}
	if err != nil {
		logger.WithError(err).Error("Failed to validate project")
		return http.StatusInternalServerError, "Failed to validate project"
	}

	// Validate project belongs to user's organization
	if projectOrgID != orgID {
		return http.StatusForbidden, "Project does not belong to your organization"
	}

	// Validate project belongs to specified location (if location validation is needed)
	if locationID != 0 && projectLocationID != locationID {
		return http.StatusBadRequest, fmt.Sprintf("Project does not belong to location %d. Expected location: %d", locationID, projectLocationID)
	}

	return 0, ""
}

// validateEntityAccess validates that entity exists, belongs to project, project belongs to org and location
// Returns (statusCode, errorMessage) - errorMessage is empty string if validation passes
func validateEntityAccess(ctx context.Context, entityType string, entityID, projectID, locationID, orgID int64) (int, string) {
	// First validate project access
	statusCode, errMsg := validateProjectAccess(ctx, projectID, locationID, orgID)
	if errMsg != "" {
		return statusCode, errMsg
	}

	// Now validate entity exists and belongs to the specified project
	var entityProjectID int64
	var entityDeleted bool
	var query string

	switch entityType {
	case models.EntityTypeIssue:
		query = "SELECT project_id, is_deleted FROM project.issues WHERE id = $1"
	case models.EntityTypeRFI:
		query = "SELECT project_id, is_deleted FROM project.rfis WHERE id = $1"
	case models.EntityTypeSubmittal:
		query = "SELECT project_id, is_deleted FROM project.submittals WHERE id = $1"
	case models.EntityTypeProject:
		// For project attachments, entity_id = project_id
		if entityID != projectID {
			return http.StatusBadRequest, "For project attachments, entity_id must equal project_id"
		}
		return 0, "" // Already validated project access above
	default:
		return http.StatusBadRequest, "Unsupported entity type for validation"
	}

	err := sqlDB.QueryRowContext(ctx, query, entityID).Scan(&entityProjectID, &entityDeleted)

	if err == sql.ErrNoRows {
		return http.StatusNotFound, fmt.Sprintf("%s not found", strings.Title(entityType))
	}
	if err != nil {
		logger.WithError(err).WithFields(logrus.Fields{
			"entity_type": entityType,
			"entity_id":   entityID,
		}).Error("Failed to validate entity")
		return http.StatusInternalServerError, "Failed to validate entity"
	}

	// Check if entity is deleted
	if entityDeleted {
		return http.StatusNotFound, fmt.Sprintf("%s not found", strings.Title(entityType))
	}

	// Validate entity belongs to the specified project
	if entityProjectID != projectID {
		return http.StatusBadRequest, fmt.Sprintf("%s does not belong to project %d. Expected project: %d",
			strings.Title(entityType), projectID, entityProjectID)
	}

	return 0, ""
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

	// Initialize S3 client
	bucketName := os.Getenv("BUCKET_NAME")
	if bucketName == "" {
		// Fallback to a default or get from SSM
		bucketName = "buildboard-attachments-dev" // This should come from environment
	}

	s3Client = clients.NewS3Client(isLocal, bucketName)

	logger.Info("Attachment management service initialized successfully")
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

	// Initialize attachment repository
	attachmentRepository = &data.AttachmentDao{
		DB:     sqlDB,
		Logger: logger,
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.WithField("operation", "setupPostgresSQLClient").Debug("PostgreSQL client initialized successfully")
	}

	return nil
}