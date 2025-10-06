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
	logger        *logrus.Logger
	isLocal       bool
	ssmRepository data.SSMRepository
	ssmParams     map[string]string
	sqlDB         *sql.DB
	rfiRepository data.RFIRepository
)

// Handler processes API Gateway requests for RFI management operations
//
// CONSOLIDATED API ENDPOINTS (6 total):
//
// Core CRUD Operations:
//   GET    /rfis/{id}                                    - Get RFI with all data (attachments, comments)
//   POST   /rfis                                         - Create RFI
//   PUT    /rfis/{id}                                    - Update RFI (including status changes via action field)
//
// Context Query:
//   GET    /contexts/{contextType}/{contextId}/rfis     - Get RFIs for project/location
//
// Sub-resources:
//   POST   /rfis/{id}/attachments                       - Add attachment
//   POST   /rfis/{id}/comments                          - Add comment
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.WithFields(logrus.Fields{
		"method":      request.HTTPMethod,
		"path":        request.Path,
		"resource":    request.Resource,
		"path_params": request.PathParameters,
		"operation":   "Handler",
	}).Debug("Processing RFI management request")

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
	// Core RFI CRUD operations
	case request.Resource == "/rfis/{rfiId}" && request.HTTPMethod == "GET":
		return handleGetRFI(ctx, request, claims)
	case request.Resource == "/rfis" && request.HTTPMethod == "POST":
		return handleCreateRFI(ctx, request, claims)
	case request.Resource == "/rfis/{rfiId}" && request.HTTPMethod == "PUT":
		return handleUpdateRFI(ctx, request, claims)

	// Context-based RFI queries
	case request.Resource == "/contexts/{contextType}/{contextId}/rfis" && request.HTTPMethod == "GET":
		return handleGetContextRFIs(ctx, request, claims)

	// Sub-resource operations
	// RFI attachments now handled by centralized attachment management service
	case request.Resource == "/rfis/{rfiId}/comments" && request.HTTPMethod == "POST":
		return handleAddRFIComment(ctx, request, claims)

	default:
		logger.WithFields(logrus.Fields{
			"method":    request.HTTPMethod,
			"resource":  request.Resource,
			"operation": "Handler",
		}).Warn("Endpoint not found")
		return api.ErrorResponse(http.StatusNotFound, "Endpoint not found", logger), nil
	}
}

// handleCreateRFI handles POST /rfis
func handleCreateRFI(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	var createReq models.CreateRFIRequest
	if err := api.ParseJSONBody(request.Body, &createReq); err != nil {
		logger.WithError(err).Error("Invalid request body for create RFI")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	userID := claims.UserID
	createdRFI, err := rfiRepository.CreateRFI(ctx, createReq.ProjectID, userID, claims.OrgID, &createReq)
	if err != nil {
		logger.WithError(err).Error("Failed to create RFI")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create RFI", logger), nil
	}

	return api.SuccessResponse(http.StatusCreated, createdRFI, logger), nil
}

// handleGetRFI handles GET /rfis/{rfiId} - returns RFI with all attachments and comments
func handleGetRFI(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	rfiID, err := strconv.ParseInt(request.PathParameters["rfiId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid RFI ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid RFI ID", logger), nil
	}

	rfi, err := rfiRepository.GetRFI(ctx, rfiID)
	if err != nil {
		if err.Error() == "RFI not found" {
			return api.ErrorResponse(http.StatusNotFound, "RFI not found", logger), nil
		}
		logger.WithError(err).Error("Failed to get RFI")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get RFI", logger), nil
	}

	// Enrich with comments and attachments
	comments, _ := rfiRepository.GetRFIComments(ctx, rfiID)
	attachments, _ := rfiRepository.GetRFIAttachments(ctx, rfiID)

	rfi.Comments = comments
	rfi.Attachments = attachments

	return api.SuccessResponse(http.StatusOK, rfi, logger), nil
}

// handleUpdateRFI handles PUT /rfis/{rfiId} - supports action field for status changes
func handleUpdateRFI(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	rfiID, err := strconv.ParseInt(request.PathParameters["rfiId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid RFI ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid RFI ID", logger), nil
	}

	var updateReq models.UpdateRFIRequest
	if err := api.ParseJSONBody(request.Body, &updateReq); err != nil {
		logger.WithError(err).Error("Invalid request body for update RFI")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	userID := claims.UserID

	// Handle action field for status changes (consolidates submit, approve, reject, respond)
	if updateReq.Action != nil {
		switch *updateReq.Action {
		case "submit":
			err = rfiRepository.SubmitRFI(ctx, rfiID, nil, userID) // assignedTo can be nil for now
		case "approve":
			err = rfiRepository.ApproveRFI(ctx, rfiID, userID, updateReq.Notes)
		case "reject":
			if updateReq.Notes == "" {
				return api.ErrorResponse(http.StatusBadRequest, "Rejection reason is required", logger), nil
			}
			err = rfiRepository.RejectRFI(ctx, rfiID, userID, updateReq.Notes)
		case "respond":
			if updateReq.ResponseText == "" {
				return api.ErrorResponse(http.StatusBadRequest, "Response text is required", logger), nil
			}
			err = rfiRepository.RespondToRFI(ctx, rfiID, updateReq.ResponseText, userID)
		default:
			return api.ErrorResponse(http.StatusBadRequest, "Invalid action", logger), nil
		}

		if err != nil {
			logger.WithError(err).Error("Failed to perform RFI action")
			return api.ErrorResponse(http.StatusInternalServerError, "Failed to update RFI", logger), nil
		}

		// Add comment for action
		if updateReq.Notes != "" {
			comment := &models.RFIComment{
				RFIID:       rfiID,
				Comment:     updateReq.Notes,
				CommentType: models.RFICommentTypeStatusChange,
				CreatedBy:   userID,
				UpdatedBy:   userID,
			}
			rfiRepository.AddRFIComment(ctx, comment)
		}

		return api.SuccessResponse(http.StatusOK, map[string]string{"message": "RFI updated successfully"}, logger), nil
	}

	// Regular update
	updatedRFI, err := rfiRepository.UpdateRFI(ctx, rfiID, userID, claims.OrgID, &updateReq)
	if err != nil {
		if err.Error() == "RFI not found" {
			return api.ErrorResponse(http.StatusNotFound, "RFI not found", logger), nil
		}
		logger.WithError(err).Error("Failed to update RFI")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update RFI", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, updatedRFI, logger), nil
}

// handleGetContextRFIs handles GET /contexts/{contextType}/{contextId}/rfis
func handleGetContextRFIs(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
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

	rfis, err := rfiRepository.GetRFIsByProject(ctx, contextID, filters)
	if err != nil {
		logger.WithError(err).Error("Failed to get context RFIs")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get RFIs", logger), nil
	}

	response := map[string]interface{}{
		"context_type": contextType,
		"context_id":   contextID,
		"rfis":         rfis,
	}

	return api.SuccessResponse(http.StatusOK, response, logger), nil
}

// RFI attachment handler removed - now handled by centralized attachment management service
// Removed function: handleAddRFIAttachment

// handleAddRFIComment handles POST /rfis/{rfiId}/comments
func handleAddRFIComment(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	rfiID, err := strconv.ParseInt(request.PathParameters["rfiId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid RFI ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid RFI ID", logger), nil
	}

	var req struct {
		Comment string `json:"comment"`
	}
	if err := api.ParseJSONBody(request.Body, &req); err != nil {
		logger.WithError(err).Error("Invalid request body for add comment")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	if req.Comment == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Comment is required", logger), nil
	}

	userID := claims.UserID
	comment := &models.RFIComment{
		RFIID:       rfiID,
		Comment:     req.Comment,
		CommentType: models.RFICommentTypeComment,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}

	if err := rfiRepository.AddRFIComment(ctx, comment); err != nil {
		logger.WithError(err).Error("Failed to add RFI comment")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to add comment", logger), nil
	}

	return api.SuccessResponse(http.StatusCreated, comment, logger), nil
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

	// Initialize RFI repository
	rfiRepository = &data.RFIDao{
		DB:     sqlDB,
		Logger: logger,
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.WithField("operation", "setupPostgresSQLClient").Debug("PostgreSQL client initialized successfully")
	}

	return nil
}