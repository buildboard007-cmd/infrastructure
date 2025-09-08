package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"infrastructure/lib/api"
	"infrastructure/lib/auth"
	"infrastructure/lib/clients"
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
			return handleGetProjectRFIs(ctx, projectID, claims.OrgID, request.QueryStringParameters), nil
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

// handleCreateRFI handles the creation of a new RFI
func handleCreateRFI(ctx context.Context, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	var req models.CreateRFIRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		logger.WithError(err).Error("Failed to parse create RFI request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Validate required fields
	if req.ProjectID == 0 || req.Subject == "" || req.Question == "" {
		return api.ErrorResponse(http.StatusBadRequest, "Missing required fields", logger)
	}

	// Generate RFI number if not provided
	if req.RFINumber == "" {
		rfiNumber, err := rfiRepository.GenerateRFINumber(ctx, req.ProjectID)
		if err != nil {
			logger.WithError(err).Error("Failed to generate RFI number")
			return api.ErrorResponse(http.StatusInternalServerError, "Failed to generate RFI number", logger)
		}
		req.RFINumber = rfiNumber
	}

	// Convert request to RFI model
	rfi := &models.RFI{
		ProjectID:               req.ProjectID,
		OrgID:                   req.OrgID,
		LocationID:              req.LocationID,
		RFINumber:               req.RFINumber,
		Subject:                 req.Subject,
		Question:                req.Question,
		Description:             req.Description,
		Category:                req.Category,
		Discipline:              req.Discipline,
		TradeType:               req.TradeType,
		ProjectPhase:            req.ProjectPhase,
		Priority:                req.Priority,
		Status:                  models.RFIStatusDraft,
		SubmittedBy:             userID,
		ReviewerEmail:           req.ReviewerEmail,
		ApproverEmail:           req.ApproverEmail,
		CCList:                  req.CCList,
		DistributionList:        req.DistributionList,
		DueDate:                 req.DueDate,
		CostImpact:              req.CostImpact == "Yes",
		ScheduleImpact:          req.ScheduleImpact == "Yes",
		CostImpactAmount:        req.CostImpactAmount,
		ScheduleImpactDays:      req.ScheduleImpactDays,
		CostImpactDetails:       req.CostImpactDetails,
		ScheduleImpactDetails:   req.ScheduleImpactDetails,
		LocationDescription:     req.Location,
		DrawingReferences:       req.DrawingReferences,
		SpecificationReferences: req.SpecificationReferences,
		RelatedSubmittals:       req.RelatedSubmittals,
		RelatedChangeEvents:     req.RelatedChangeEvents,
		WorkflowType:            req.WorkflowType,
		RequiresApproval:        req.RequiresApproval,
		UrgencyJustification:    req.UrgencyJustification,
		BusinessJustification:   req.BusinessJustification,
		CreatedBy:               userID,
		UpdatedBy:               userID,
	}

	// Set default values if not provided
	if rfi.Priority == "" {
		rfi.Priority = models.RFIPriorityMedium
	}
	if rfi.WorkflowType == "" {
		rfi.WorkflowType = models.RFIWorkflowStandard
	}

	// Create RFI
	createdRFI, err := rfiRepository.CreateRFI(ctx, rfi)
	if err != nil {
		logger.WithError(err).Error("Failed to create RFI")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create RFI", logger)
	}

	// Process attachments if any
	for _, attachment := range req.Attachments {
		attachment.RFIID = createdRFI.ID
		attachment.UploadedBy = userID
		attachment.CreatedBy = userID
		attachment.UpdatedBy = userID

		if attachment.AttachmentType == "" {
			attachment.AttachmentType = "document"
		}

		_, err := rfiRepository.AddRFIAttachment(ctx, &attachment)
		if err != nil {
			logger.WithError(err).Warn("Failed to add attachment to RFI")
		}
	}

	logger.WithFields(logrus.Fields{
		"rfi_id":     createdRFI.ID,
		"project_id": createdRFI.ProjectID,
		"user_id":    userID,
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

// handleUpdateRFI handles updating an RFI
func handleUpdateRFI(ctx context.Context, rfiID, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	var req models.UpdateRFIRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		logger.WithError(err).Error("Failed to parse update RFI request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Check if RFI exists
	existingRFI, err := rfiRepository.GetRFI(ctx, rfiID)
	if err != nil {
		if err.Error() == "RFI not found" {
			return api.ErrorResponse(http.StatusNotFound, "RFI not found", logger)
		}
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to retrieve RFI", logger)
	}

	// Check if user can update (must be in draft or user is submitter)
	if existingRFI.Status != models.RFIStatusDraft && existingRFI.SubmittedBy != userID {
		return api.ErrorResponse(http.StatusForbidden, "Cannot update RFI in current status", logger)
	}

	// Update RFI
	if err := rfiRepository.UpdateRFI(ctx, rfiID, &req, userID); err != nil {
		logger.WithError(err).Error("Failed to update RFI")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update RFI", logger)
	}

	logger.WithFields(logrus.Fields{
		"rfi_id":  rfiID,
		"user_id": userID,
	}).Info("RFI updated successfully")

	return api.SuccessResponse(http.StatusOK, "RFI updated successfully", logger)
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
		models.RFIStatusInReview,
		models.RFIStatusResponded,
		models.RFIStatusClosed,
		models.RFIStatusCancelled,
		models.RFIStatusOnHold,
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
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Check if running locally
	if os.Getenv("AWS_EXECUTION_ENV") == "" {
		isLocal = true
		logger.Info("Running in local mode")
	}

	// Initialize AWS SSM Parameter Store client
	ssmClient := clients.NewSSMClient(isLocal)
	ssmRepository = &data.SSMDao{
		SSM:    ssmClient,
		Logger: logger,
	}

	// Retrieve all required configuration parameters from SSM
	var err error
	ssmParams, err = ssmRepository.GetParameters()
	if err != nil {
		logger.WithError(err).Fatal("Failed to fetch SSM parameters")
	}

	// Create database connection string
	dbHost := ssmParams["db_host"]
	dbUsername := ssmParams["db_username"] 
	dbPassword := ssmParams["db_password"]
	dbName := ssmParams["db_name"]

	connectionString := fmt.Sprintf(
		"host=%s port=5432 user=%s password=%s dbname=%s sslmode=require",
		dbHost, dbUsername, dbPassword, dbName,
	)
	logger.Info("connection string: ", connectionString)

	// Initialize database connection
	sqlDB, err = sql.Open("postgres", connectionString)
	if err != nil {
		logger.WithError(err).Fatal("Failed to open database connection")
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Test database connection
	if err = sqlDB.Ping(); err != nil {
		logger.WithError(err).Fatal("Failed to ping database")
	}

	// Initialize RFI repository
	rfiRepository = data.NewRFIDao(sqlDB, logger)

	logger.Info("RFI management service initialized successfully")
}

func main() {
	lambda.Start(Handler)
}
