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
	logger        *logrus.Logger
	isLocal       bool
	ssmRepository data.SSMRepository
	ssmParams     map[string]string
	sqlDB         *sql.DB
	rfiRepository data.RFIRepository
)

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.WithFields(logrus.Fields{
		"operation": "Handler",
		"method":    request.HTTPMethod,
		"path":      request.Path,
		"resource":  request.Resource,
	}).Info("RFI management request received")

	// Extract claims from JWT token via API Gateway authorizer
	claims, err := auth.ExtractClaimsFromRequest(request)
	if err != nil {
		logger.WithError(err).Error("Authentication failed")
		return api.ErrorResponse(http.StatusUnauthorized, "Authentication failed", logger), nil
	}

	// Handle different routes
	switch request.HTTPMethod {
	case http.MethodPost:
		// POST /rfis - Create new RFI
		if request.Resource == "/rfis" {
			return handleCreateRFI(ctx, claims.UserID, claims.OrgID, request.Body), nil
		}
		// POST /rfis/{rfiId}/submit - Submit RFI for review
		if strings.Contains(request.Resource, "/rfis/{rfiId}/submit") {
			rfiID, err := strconv.ParseInt(request.PathParameters["rfiId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid RFI ID", logger), nil
			}
			return handleSubmitRFI(ctx, rfiID, claims.UserID, request.Body), nil
		}
		// POST /rfis/{rfiId}/respond - Respond to RFI
		if strings.Contains(request.Resource, "/rfis/{rfiId}/respond") {
			rfiID, err := strconv.ParseInt(request.PathParameters["rfiId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid RFI ID", logger), nil
			}
			return handleRespondToRFI(ctx, rfiID, claims.UserID, request.Body), nil
		}
		// POST /rfis/{rfiId}/approve - Approve RFI
		if strings.Contains(request.Resource, "/rfis/{rfiId}/approve") {
			rfiID, err := strconv.ParseInt(request.PathParameters["rfiId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid RFI ID", logger), nil
			}
			return handleApproveRFI(ctx, rfiID, claims.UserID, request.Body), nil
		}
		// POST /rfis/{rfiId}/reject - Reject RFI
		if strings.Contains(request.Resource, "/rfis/{rfiId}/reject") {
			rfiID, err := strconv.ParseInt(request.PathParameters["rfiId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid RFI ID", logger), nil
			}
			return handleRejectRFI(ctx, rfiID, claims.UserID, request.Body), nil
		}
		// POST /rfis/{rfiId}/comments - Add comment to RFI
		if strings.Contains(request.Resource, "/rfis/{rfiId}/comments") {
			rfiID, err := strconv.ParseInt(request.PathParameters["rfiId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid RFI ID", logger), nil
			}
			return handleAddRFIComment(ctx, rfiID, claims.UserID, request.Body), nil
		}
		// POST /rfis/{rfiId}/attachments - Add attachment to RFI
		if strings.Contains(request.Resource, "/rfis/{rfiId}/attachments") {
			rfiID, err := strconv.ParseInt(request.PathParameters["rfiId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid RFI ID", logger), nil
			}
			return handleAddRFIAttachment(ctx, rfiID, claims.UserID, request.Body), nil
		}
		return api.ErrorResponse(http.StatusNotFound, "Endpoint not found", logger), nil

	case http.MethodGet:
		// GET /projects/{projectId}/rfis - List RFIs for project
		if strings.Contains(request.Resource, "/projects/{projectId}/rfis") {
			projectID, err := strconv.ParseInt(request.PathParameters["projectId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid project ID", logger), nil
			}
			// Ensure filters map is not nil
			filters := request.QueryStringParameters
			if filters == nil {
				filters = make(map[string]string)
			}
			return handleGetProjectRFIs(ctx, projectID, claims.OrgID, filters), nil
		}

		// GET /rfis/{rfiId} - Get specific RFI
		if strings.Contains(request.Resource, "/rfis/{rfiId}") && !strings.Contains(request.Resource, "/comments") && !strings.Contains(request.Resource, "/attachments") {
			rfiID, err := strconv.ParseInt(request.PathParameters["rfiId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid RFI ID", logger), nil
			}
			return handleGetRFI(ctx, rfiID, claims.OrgID), nil
		}

		// GET /rfis/{rfiId}/comments - Get RFI comments
		if strings.Contains(request.Resource, "/rfis/{rfiId}/comments") {
			rfiID, err := strconv.ParseInt(request.PathParameters["rfiId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid RFI ID", logger), nil
			}
			return handleGetRFIComments(ctx, rfiID), nil
		}

		// GET /rfis/{rfiId}/attachments - Get RFI attachments
		if strings.Contains(request.Resource, "/rfis/{rfiId}/attachments") {
			rfiID, err := strconv.ParseInt(request.PathParameters["rfiId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid RFI ID", logger), nil
			}
			return handleGetRFIAttachments(ctx, rfiID), nil
		}

		return api.ErrorResponse(http.StatusNotFound, "Endpoint not found", logger), nil

	case http.MethodPut:
		// PUT /rfis/{rfiId} - Update RFI
		if strings.Contains(request.Resource, "/rfis/{rfiId}") {
			rfiID, err := strconv.ParseInt(request.PathParameters["rfiId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid RFI ID", logger), nil
			}
			return handleUpdateRFI(ctx, rfiID, claims.UserID, claims.OrgID, request.Body), nil
		}
		return api.ErrorResponse(http.StatusNotFound, "Endpoint not found", logger), nil

	case http.MethodPatch:
		// PATCH /rfis/{rfiId}/status - Update RFI status
		if strings.Contains(request.Resource, "/rfis/{rfiId}/status") {
			rfiID, err := strconv.ParseInt(request.PathParameters["rfiId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid RFI ID", logger), nil
			}
			return handleUpdateRFIStatus(ctx, rfiID, claims.UserID, claims.OrgID, request.Body), nil
		}
		return api.ErrorResponse(http.StatusNotFound, "Endpoint not found", logger), nil

	case http.MethodDelete:
		// DELETE /rfis/{rfiId} - Delete RFI
		if strings.Contains(request.Resource, "/rfis/{rfiId}") {
			rfiID, err := strconv.ParseInt(request.PathParameters["rfiId"], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid RFI ID", logger), nil
			}
			return handleDeleteRFI(ctx, rfiID, claims.UserID, claims.OrgID), nil
		}
		return api.ErrorResponse(http.StatusNotFound, "Endpoint not found", logger), nil

	default:
		return api.ErrorResponse(http.StatusMethodNotAllowed, "Method not allowed", logger), nil
	}
}

// handleCreateRFI handles the creation of a new RFI with unified structure and JWT-based orgID
func handleCreateRFI(ctx context.Context, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	// Parse unified request structure
	var createReq models.CreateRFIRequest
	if err := json.Unmarshal([]byte(body), &createReq); err != nil {
		logger.WithError(err).Error("Failed to parse create RFI request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Extract project_id from request (should be in request body)
	projectID := createReq.ProjectID
	if projectID == 0 {
		return api.ErrorResponse(http.StatusBadRequest, "Project ID is required", logger)
	}

	// Validate required fields from unified structure
	if createReq.Subject == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Subject is required", logger)
	}
	if createReq.Question == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Question is required", logger)
	}
	if createReq.Priority == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Priority is required", logger)
	}
	if createReq.Category == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Category is required", logger)
	}

	// Validate category value
	validCategories := []string{
		"DESIGN", "SPECIFICATION", "SCHEDULE", "COORDINATION",
		"GENERAL", "SUBMITTAL", "CHANGE_EVENT",
	}
	isValidCategory := false
	for _, c := range validCategories {
		if createReq.Category == c {
			isValidCategory = true
			break
		}
	}
	if !isValidCategory {
		return api.ErrorResponse(http.StatusBadRequest, "Invalid category value", logger)
	}

	// Create RFI using repository with orgID from JWT (validation happens in repository)
	createdRFI, err := rfiRepository.CreateRFI(ctx, projectID, userID, orgID, &createReq)
	if err != nil {
		if err.Error() == "project does not belong to your organization" {
			return api.ErrorResponse(http.StatusForbidden, "Project does not belong to your organization", logger)
		}
		logger.WithError(err).Error("Failed to create RFI")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create RFI", logger)
	}

	logger.WithFields(logrus.Fields{
		"rfi_id":     createdRFI.ID,
		"project_id": createdRFI.ProjectID,
		"user_id":    userID,
		"org_id":     orgID,
	}).Info("RFI created successfully")

	return api.SuccessResponse(http.StatusCreated, createdRFI, logger)
}

// handleGetProjectRFIs handles getting all RFIs for a project
func handleGetProjectRFIs(ctx context.Context, projectID, orgID int64, filters map[string]string) events.APIGatewayProxyResponse {
	rfis, err := rfiRepository.GetRFIsByProject(ctx, projectID, filters)
	if err != nil {
		logger.WithError(err).Error("Failed to get project RFIs")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to retrieve RFIs", logger)
	}

	response := models.RFIListResponse{
		RFIs:       rfis,
		TotalCount: len(rfis),
	}

	logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"count":      len(rfis),
	}).Info("Project RFIs retrieved successfully")

	return api.SuccessResponse(http.StatusOK, response, logger)
}

// handleGetRFI handles getting a specific RFI
func handleGetRFI(ctx context.Context, rfiID, orgID int64) events.APIGatewayProxyResponse {
	rfi, err := rfiRepository.GetRFI(ctx, rfiID)
	if err != nil {
		if err.Error() == "RFI not found" {
			return api.ErrorResponse(http.StatusNotFound, "RFI not found", logger)
		}
		logger.WithError(err).Error("Failed to get RFI")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to retrieve RFI", logger)
	}

	// Get comments and attachments
	comments, _ := rfiRepository.GetRFIComments(ctx, rfiID)
	attachments, _ := rfiRepository.GetRFIAttachments(ctx, rfiID)

	rfi.Comments = comments
	rfi.Attachments = attachments

	logger.WithFields(logrus.Fields{
		"rfi_id": rfiID,
	}).Info("RFI retrieved successfully")

	return api.SuccessResponse(http.StatusOK, rfi, logger)
}

// handleUpdateRFI handles updating an RFI with unified structure and orgID validation
func handleUpdateRFI(ctx context.Context, rfiID, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	// Parse unified request structure
	var updateReq models.UpdateRFIRequest
	if err := json.Unmarshal([]byte(body), &updateReq); err != nil {
		logger.WithError(err).Error("Failed to parse update RFI request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Update RFI using repository with orgID from JWT (validation happens in repository)
	updatedRFI, err := rfiRepository.UpdateRFI(ctx, rfiID, userID, orgID, &updateReq)
	if err != nil {
		if err.Error() == "RFI not found" {
			return api.ErrorResponse(http.StatusNotFound, "RFI not found", logger)
		}
		if err.Error() == "RFI does not belong to your organization" {
			return api.ErrorResponse(http.StatusForbidden, "RFI does not belong to your organization", logger)
		}
		if err.Error() == "Cannot update RFI in current status" {
			return api.ErrorResponse(http.StatusForbidden, "Cannot update RFI in current status", logger)
		}
		logger.WithError(err).Error("Failed to update RFI")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update RFI", logger)
	}

	logger.WithFields(logrus.Fields{
		"rfi_id":  rfiID,
		"user_id": userID,
		"org_id":  orgID,
	}).Info("RFI updated successfully")

	return api.SuccessResponse(http.StatusOK, updatedRFI, logger)
}

// handleUpdateRFIStatus handles updating RFI status
func handleUpdateRFIStatus(ctx context.Context, rfiID, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	var req models.UpdateRFIStatusRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		logger.WithError(err).Error("Failed to parse update RFI status request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Validate status
	validStatuses := []string{
		models.RFIStatusDraft,
		models.RFIStatusSubmitted,
		models.RFIStatusUnderReview,
		models.RFIStatusAnswered,
		models.RFIStatusClosed,
		models.RFIStatusVoid,
		models.RFIStatusRequiresRevision,
	}

	isValid := false
	for _, s := range validStatuses {
		if s == req.Status {
			isValid = true
			break
		}
	}

	if !isValid {
		return api.ErrorResponse(http.StatusBadRequest, "Invalid status", logger)
	}

	// Update status
	if err := rfiRepository.UpdateRFIStatus(ctx, rfiID, req.Status, userID, req.Comment); err != nil {
		logger.WithError(err).Error("Failed to update RFI status")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update RFI status", logger)
	}

	logger.WithFields(logrus.Fields{
		"rfi_id":     rfiID,
		"new_status": req.Status,
		"user_id":    userID,
	}).Info("RFI status updated successfully")

	return api.SuccessResponse(http.StatusOK, "RFI status updated successfully", logger)
}

// handleDeleteRFI handles deleting an RFI
func handleDeleteRFI(ctx context.Context, rfiID, userID, orgID int64) events.APIGatewayProxyResponse {
	// Check if RFI exists
	existingRFI, err := rfiRepository.GetRFI(ctx, rfiID)
	if err != nil {
		if err.Error() == "RFI not found" {
			return api.ErrorResponse(http.StatusNotFound, "RFI not found", logger)
		}
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to retrieve RFI", logger)
	}

	// Check if user can delete (must be in draft status and user is submitter)
	if existingRFI.Status != models.RFIStatusDraft || existingRFI.SubmittedBy != userID {
		return api.ErrorResponse(http.StatusForbidden, "Cannot delete RFI in current status", logger)
	}

	// Delete RFI
	if err := rfiRepository.DeleteRFI(ctx, rfiID, userID); err != nil {
		logger.WithError(err).Error("Failed to delete RFI")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to delete RFI", logger)
	}

	logger.WithFields(logrus.Fields{
		"rfi_id":  rfiID,
		"user_id": userID,
	}).Info("RFI deleted successfully")

	return api.SuccessResponse(http.StatusOK, "RFI deleted successfully", logger)
}

// handleSubmitRFI handles submitting an RFI for review
func handleSubmitRFI(ctx context.Context, rfiID, userID int64, body string) events.APIGatewayProxyResponse {
	var req models.SubmitRFIRequest
	if body != "" {
		if err := json.Unmarshal([]byte(body), &req); err != nil {
			logger.WithError(err).Error("Failed to parse submit RFI request")
			return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
		}
	}

	// Submit RFI
	if err := rfiRepository.SubmitRFI(ctx, rfiID, req.AssignedTo, userID); err != nil {
		logger.WithError(err).Error("Failed to submit RFI")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to submit RFI", logger)
	}

	// Add comment if provided
	if req.Comment != "" {
		comment := &models.RFIComment{
			RFIID:       rfiID,
			Comment:     req.Comment,
			CommentType: models.RFICommentTypeStatusChange,
			CreatedBy:   userID,
			UpdatedBy:   userID,
		}
		rfiRepository.AddRFIComment(ctx, comment)
	}

	logger.WithFields(logrus.Fields{
		"rfi_id":  rfiID,
		"user_id": userID,
	}).Info("RFI submitted successfully")

	return api.SuccessResponse(http.StatusOK, "RFI submitted successfully", logger)
}

// handleRespondToRFI handles responding to an RFI
func handleRespondToRFI(ctx context.Context, rfiID, userID int64, body string) events.APIGatewayProxyResponse {
	var req models.RespondToRFIRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		logger.WithError(err).Error("Failed to parse respond to RFI request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	if req.Response == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Response is required", logger)
	}

	// Respond to RFI
	if err := rfiRepository.RespondToRFI(ctx, rfiID, req.Response, userID); err != nil {
		logger.WithError(err).Error("Failed to respond to RFI")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to respond to RFI", logger)
	}

	// Add response comment
	comment := &models.RFIComment{
		RFIID:       rfiID,
		Comment:     req.Response,
		CommentType: models.RFICommentTypeResponse,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}
	rfiRepository.AddRFIComment(ctx, comment)

	logger.WithFields(logrus.Fields{
		"rfi_id":  rfiID,
		"user_id": userID,
	}).Info("RFI response added successfully")

	return api.SuccessResponse(http.StatusOK, "RFI response added successfully", logger)
}

// handleApproveRFI handles approving an RFI
func handleApproveRFI(ctx context.Context, rfiID, userID int64, body string) events.APIGatewayProxyResponse {
	var req models.ApproveRFIRequest
	if body != "" {
		if err := json.Unmarshal([]byte(body), &req); err != nil {
			logger.WithError(err).Error("Failed to parse approve RFI request")
			return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
		}
	}

	// Approve RFI
	if err := rfiRepository.ApproveRFI(ctx, rfiID, userID, req.ApprovalComments); err != nil {
		logger.WithError(err).Error("Failed to approve RFI")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to approve RFI", logger)
	}

	// Add approval comment
	comment := &models.RFIComment{
		RFIID:       rfiID,
		Comment:     "RFI approved" + util.ConditionalString(req.ApprovalComments != "", ": "+req.ApprovalComments, ""),
		CommentType: models.RFICommentTypeApproval,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}
	rfiRepository.AddRFIComment(ctx, comment)

	logger.WithFields(logrus.Fields{
		"rfi_id":  rfiID,
		"user_id": userID,
	}).Info("RFI approved successfully")

	return api.SuccessResponse(http.StatusOK, "RFI approved successfully", logger)
}

// handleRejectRFI handles rejecting an RFI
func handleRejectRFI(ctx context.Context, rfiID, userID int64, body string) events.APIGatewayProxyResponse {
	var req models.RejectRFIRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		logger.WithError(err).Error("Failed to parse reject RFI request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	if req.RejectionReason == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Rejection reason is required", logger)
	}

	// Reject RFI
	if err := rfiRepository.RejectRFI(ctx, rfiID, userID, req.RejectionReason); err != nil {
		logger.WithError(err).Error("Failed to reject RFI")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to reject RFI", logger)
	}

	// Add rejection comment
	comment := &models.RFIComment{
		RFIID:       rfiID,
		Comment:     "RFI rejected: " + req.RejectionReason,
		CommentType: models.RFICommentTypeRejection,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}
	rfiRepository.AddRFIComment(ctx, comment)

	logger.WithFields(logrus.Fields{
		"rfi_id":  rfiID,
		"user_id": userID,
	}).Info("RFI rejected successfully")

	return api.SuccessResponse(http.StatusOK, "RFI rejected successfully", logger)
}

// handleAddRFIComment handles adding a comment to an RFI
func handleAddRFIComment(ctx context.Context, rfiID, userID int64, body string) events.APIGatewayProxyResponse {
	var req struct {
		Comment string `json:"comment"`
	}

	if err := json.Unmarshal([]byte(body), &req); err != nil {
		logger.WithError(err).Error("Failed to parse add comment request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	if req.Comment == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Comment is required", logger)
	}

	comment := &models.RFIComment{
		RFIID:       rfiID,
		Comment:     req.Comment,
		CommentType: models.RFICommentTypeComment,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}

	if err := rfiRepository.AddRFIComment(ctx, comment); err != nil {
		logger.WithError(err).Error("Failed to add RFI comment")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to add comment", logger)
	}

	logger.WithFields(logrus.Fields{
		"rfi_id":  rfiID,
		"user_id": userID,
	}).Info("RFI comment added successfully")

	return api.SuccessResponse(http.StatusCreated, comment, logger)
}

// handleGetRFIComments handles getting all comments for an RFI
func handleGetRFIComments(ctx context.Context, rfiID int64) events.APIGatewayProxyResponse {
	comments, err := rfiRepository.GetRFIComments(ctx, rfiID)
	if err != nil {
		logger.WithError(err).Error("Failed to get RFI comments")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to retrieve comments", logger)
	}

	logger.WithFields(logrus.Fields{
		"rfi_id": rfiID,
		"count":  len(comments),
	}).Info("RFI comments retrieved successfully")

	return api.SuccessResponse(http.StatusOK, comments, logger)
}

// handleAddRFIAttachment handles adding an attachment to an RFI
func handleAddRFIAttachment(ctx context.Context, rfiID, userID int64, body string) events.APIGatewayProxyResponse {
	var attachment models.RFIAttachment
	if err := json.Unmarshal([]byte(body), &attachment); err != nil {
		logger.WithError(err).Error("Failed to parse add attachment request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	attachment.RFIID = rfiID
	attachment.UploadedBy = userID
	attachment.CreatedBy = userID
	attachment.UpdatedBy = userID

	if attachment.AttachmentType == "" {
		attachment.AttachmentType = "document"
	}

	createdAttachment, err := rfiRepository.AddRFIAttachment(ctx, &attachment)
	if err != nil {
		logger.WithError(err).Error("Failed to add RFI attachment")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to add attachment", logger)
	}

	logger.WithFields(logrus.Fields{
		"rfi_id":        rfiID,
		"attachment_id": createdAttachment.ID,
		"user_id":       userID,
	}).Info("RFI attachment added successfully")

	return api.SuccessResponse(http.StatusCreated, createdAttachment, logger)
}

// handleGetRFIAttachments handles getting all attachments for an RFI
func handleGetRFIAttachments(ctx context.Context, rfiID int64) events.APIGatewayProxyResponse {
	attachments, err := rfiRepository.GetRFIAttachments(ctx, rfiID)
	if err != nil {
		logger.WithError(err).Error("Failed to get RFI attachments")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to retrieve attachments", logger)
	}

	logger.WithFields(logrus.Fields{
		"rfi_id": rfiID,
		"count":  len(attachments),
	}).Info("RFI attachments retrieved successfully")

	return api.SuccessResponse(http.StatusOK, attachments, logger)
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
