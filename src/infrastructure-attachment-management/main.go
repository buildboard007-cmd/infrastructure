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
	if uploadReq.EntityType == "" || uploadReq.EntityID == 0 || uploadReq.ProjectID == 0 || uploadReq.LocationID == 0 || uploadReq.FileName == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Missing required fields", logger), nil
	}

	// Validate file type
	if !models.ValidateFileType(uploadReq.FileName) {
		return api.ErrorResponse(http.StatusBadRequest, "File type not allowed", logger), nil
	}

	// Generate S3 key
	s3Key := uploadReq.GenerateS3Key()
	if s3Key == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Invalid entity type", logger), nil
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
		logger.WithError(err).Error("Failed to verify attachment access")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to verify access", logger), nil
	}
	if !hasAccess {
		return api.ErrorResponse(http.StatusForbidden, "Access denied", logger), nil
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
		logger.WithError(err).Error("Failed to verify attachment access")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to verify access", logger), nil
	}
	if !hasAccess {
		return api.ErrorResponse(http.StatusForbidden, "Access denied", logger), nil
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
		logger.WithError(err).Error("Failed to verify attachment access")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to verify access", logger), nil
	}
	if !hasAccess {
		return api.ErrorResponse(http.StatusForbidden, "Access denied", logger), nil
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
	}

	for _, validType := range validTypes {
		if entityType == validType {
			return true
		}
	}
	return false
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