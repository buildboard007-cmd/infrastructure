package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"infrastructure/lib/models"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// SubmittalRepository defines the interface for submittal data operations
type SubmittalRepository interface {
	CreateSubmittal(ctx context.Context, projectID, userID, orgID int64, req *models.CreateSubmittalRequest) (*models.SubmittalResponse, error)
	GetSubmittal(ctx context.Context, submittalID int64) (*models.SubmittalResponse, error)
	GetSubmittalsByProject(ctx context.Context, projectID int64, filters map[string]string) ([]models.SubmittalResponse, error)
	UpdateSubmittal(ctx context.Context, submittalID, userID, orgID int64, req *models.UpdateSubmittalRequest) (*models.SubmittalResponse, error)
	ExecuteWorkflowAction(ctx context.Context, submittalID, userID int64, action *models.SubmittalWorkflowAction) (*models.SubmittalResponse, error)
	GetSubmittalStats(ctx context.Context, projectID int64) (*models.SubmittalStats, error)
	AddSubmittalAttachment(ctx context.Context, attachment *models.SubmittalAttachment) (*models.SubmittalAttachment, error)
	GetSubmittalAttachments(ctx context.Context, submittalID int64) ([]models.SubmittalAttachment, error)
	AddSubmittalHistory(ctx context.Context, history *models.SubmittalHistory) error
}

// SubmittalDao implements the SubmittalRepository interface
type SubmittalDao struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

// CreateSubmittal creates a new submittal
func (dao *SubmittalDao) CreateSubmittal(ctx context.Context, projectID, userID, orgID int64, req *models.CreateSubmittalRequest) (*models.SubmittalResponse, error) {
	// Generate submittal number if not provided
	submittalNumber := req.SubmittalNumber
	if submittalNumber == "" {
		var err error
		submittalNumber, err = dao.generateSubmittalNumber(ctx, projectID)
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to generate submittal number")
			return nil, fmt.Errorf("failed to generate submittal number: %w", err)
		}
	}

	// Parse dates
	var submissionDate, requiredApprovalDate, fabricationStartDate, installationDate *time.Time
	var err error

	if req.SubmissionDate != nil && *req.SubmissionDate != "" {
		if submissionDate, err = parseDate(*req.SubmissionDate); err != nil {
			return nil, fmt.Errorf("invalid submission_date format: %w", err)
		}
	}

	if req.RequiredApprovalDate != nil && *req.RequiredApprovalDate != "" {
		if requiredApprovalDate, err = parseDate(*req.RequiredApprovalDate); err != nil {
			return nil, fmt.Errorf("invalid required_approval_date format: %w", err)
		}
	}

	if req.FabricationStartDate != nil && *req.FabricationStartDate != "" {
		if fabricationStartDate, err = parseDate(*req.FabricationStartDate); err != nil {
			return nil, fmt.Errorf("invalid fabrication_start_date format: %w", err)
		}
	}

	if req.InstallationDate != nil && *req.InstallationDate != "" {
		if installationDate, err = parseDate(*req.InstallationDate); err != nil {
			return nil, fmt.Errorf("invalid installation_date format: %w", err)
		}
	}

	// Convert JSON fields
	deliveryTrackingJSON, _ := json.Marshal(req.DeliveryTracking)
	teamAssignmentsJSON, _ := json.Marshal(req.TeamAssignments)
	linkedDrawingsJSON, _ := json.Marshal(req.LinkedDrawings)
	referencesJSON, _ := json.Marshal(req.References)
	procurementLogJSON, _ := json.Marshal(req.ProcurementLog)
	approvalActionsJSON, _ := json.Marshal(req.ApprovalActions)
	distributionListJSON, _ := json.Marshal(req.DistributionList)
	notificationSettingsJSON, _ := json.Marshal(req.NotificationSettings)
	tagsJSON, _ := json.Marshal(req.Tags)
	customFieldsJSON, _ := json.Marshal(req.CustomFields)

	// Set defaults
	currentPhase := models.SubmittalPhasePreparation
	if req.CurrentPhase != nil && *req.CurrentPhase != "" {
		currentPhase = *req.CurrentPhase
	}

	ballInCourt := models.BallInCourtContractor
	if req.BallInCourt != nil && *req.BallInCourt != "" {
		ballInCourt = *req.BallInCourt
	}

	workflowStatus := models.SubmittalStatusPendingSubmission
	if req.WorkflowStatus != nil && *req.WorkflowStatus != "" {
		workflowStatus = *req.WorkflowStatus
	}

	query := `
		INSERT INTO project.submittals (
			project_id, org_id, location_id, submittal_number, package_name, csi_division, csi_section,
			title, description, submittal_type, specification_section, priority,
			current_phase, ball_in_court, workflow_status, assigned_to, reviewer, approver,
			submitted_by, submitted_date, required_approval_date, fabrication_start_date, installation_date,
			delivery_tracking, team_assignments, linked_drawings, submittal_references,
			procurement_log, approval_actions, distribution_list, notification_settings,
			tags, custom_fields, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35
		) RETURNING id`

	var submittalID int64
	err = dao.DB.QueryRowContext(ctx, query,
		projectID, orgID, req.LocationID, submittalNumber, req.PackageName, req.CSIDivision, req.CSISection,
		req.Title, req.Description, req.SubmittalType, req.SpecificationSection, req.Priority,
		currentPhase, ballInCourt, workflowStatus, req.AssignedTo, req.Reviewer, req.Approver,
		userID, submissionDate, requiredApprovalDate, fabricationStartDate, installationDate,
		string(deliveryTrackingJSON), string(teamAssignmentsJSON), string(linkedDrawingsJSON), string(referencesJSON),
		string(procurementLogJSON), string(approvalActionsJSON), string(distributionListJSON), string(notificationSettingsJSON),
		string(tagsJSON), string(customFieldsJSON), userID, userID,
	).Scan(&submittalID)

	if err != nil {
		dao.Logger.WithError(err).Error("Failed to create submittal")
		return nil, fmt.Errorf("failed to create submittal: %w", err)
	}

	// Add history entry
	history := &models.SubmittalHistory{
		SubmittalID: submittalID,
		Action:      "created",
		Comment:     &[]string{"Submittal created"}[0],
		CreatedBy:   userID,
	}
	dao.AddSubmittalHistory(ctx, history)

	// Return the created submittal
	return dao.GetSubmittal(ctx, submittalID)
}

// GetSubmittal retrieves a single submittal by ID
func (dao *SubmittalDao) GetSubmittal(ctx context.Context, submittalID int64) (*models.SubmittalResponse, error) {
	query := `
		SELECT s.id, s.project_id, s.org_id, s.location_id, s.submittal_number,
			   s.package_name, s.csi_division, s.csi_section, s.title, s.description,
			   s.submittal_type, s.specification_section, s.drawing_reference, s.trade_type,
			   s.priority, s.status, s.current_phase, s.ball_in_court, s.workflow_status,
			   s.revision_number, s.submitted_by, s.submitted_company_id, s.reviewed_by,
			   s.assigned_to, s.reviewer, s.approver, s.submitted_date, s.due_date,
			   s.required_approval_date, s.reviewed_date, s.approval_date,
			   s.fabrication_start_date, s.installation_date, s.review_comments,
			   s.lead_time_days, s.quantity_submitted, s.unit_of_measure,
			   s.delivery_tracking, s.team_assignments, s.linked_drawings, s.submittal_references,
			   s.procurement_log, s.approval_actions, s.distribution_list, s.notification_settings,
			   s.tags, s.custom_fields, s.created_at, s.created_by, s.updated_at, s.updated_by,
			   s.is_deleted,
			   p.name as project_name,
			   COALESCE(u_submitted.first_name, '') || ' ' || COALESCE(u_submitted.last_name, '') as submitted_by_name,
			   COALESCE(u_assigned.first_name, '') || ' ' || COALESCE(u_assigned.last_name, '') as assigned_to_name,
			   COALESCE(u_reviewer.first_name, '') || ' ' || COALESCE(u_reviewer.last_name, '') as reviewer_name,
			   COALESCE(u_approver.first_name, '') || ' ' || COALESCE(u_approver.last_name, '') as approver_name,
			   COALESCE(u_created.first_name, '') || ' ' || COALESCE(u_created.last_name, '') as created_by_name,
			   COALESCE(u_updated.first_name, '') || ' ' || COALESCE(u_updated.last_name, '') as updated_by_name
		FROM project.submittals s
		LEFT JOIN project.projects p ON s.project_id = p.id
		LEFT JOIN iam.users u_submitted ON s.submitted_by = u_submitted.id
		LEFT JOIN iam.users u_assigned ON s.assigned_to = u_assigned.id
		LEFT JOIN iam.users u_reviewer ON s.reviewer = u_reviewer.id
		LEFT JOIN iam.users u_approver ON s.approver = u_approver.id
		LEFT JOIN iam.users u_created ON s.created_by = u_created.id
		LEFT JOIN iam.users u_updated ON s.updated_by = u_updated.id
		WHERE s.id = $1 AND s.is_deleted = false`

	var submittal models.SubmittalResponse
	var deliveryTrackingJSON, teamAssignmentsJSON, linkedDrawingsJSON, referencesJSON string
	var procurementLogJSON, approvalActionsJSON, distributionListJSON, notificationSettingsJSON string
	var tagsJSON, customFieldsJSON string

	err := dao.DB.QueryRowContext(ctx, query, submittalID).Scan(
		&submittal.ID, &submittal.ProjectID, &submittal.OrgID, &submittal.LocationID, &submittal.SubmittalNumber,
		&submittal.PackageName, &submittal.CSIDivision, &submittal.CSISection, &submittal.Title, &submittal.Description,
		&submittal.SubmittalType, &submittal.SpecificationSection, &submittal.DrawingReference, &submittal.TradeType,
		&submittal.Priority, &submittal.Status, &submittal.CurrentPhase, &submittal.BallInCourt, &submittal.WorkflowStatus,
		&submittal.RevisionNumber, &submittal.SubmittedBy, &submittal.SubmittedCompanyID, &submittal.ReviewedBy,
		&submittal.AssignedTo, &submittal.Reviewer, &submittal.Approver, &submittal.SubmittedDate, &submittal.DueDate,
		&submittal.RequiredApprovalDate, &submittal.ReviewedDate, &submittal.ApprovalDate,
		&submittal.FabricationStartDate, &submittal.InstallationDate, &submittal.ReviewComments,
		&submittal.LeadTimeDays, &submittal.QuantitySubmitted, &submittal.UnitOfMeasure,
		&deliveryTrackingJSON, &teamAssignmentsJSON, &linkedDrawingsJSON, &referencesJSON,
		&procurementLogJSON, &approvalActionsJSON, &distributionListJSON, &notificationSettingsJSON,
		&tagsJSON, &customFieldsJSON, &submittal.CreatedAt, &submittal.CreatedBy, &submittal.UpdatedAt, &submittal.UpdatedBy,
		&submittal.IsDeleted,
		&submittal.ProjectName, &submittal.SubmittedByName, &submittal.AssignedToName,
		&submittal.ReviewerName, &submittal.ApproverName, &submittal.CreatedByName, &submittal.LastModifiedByName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("submittal not found")
		}
		dao.Logger.WithError(err).Error("Failed to get submittal")
		return nil, fmt.Errorf("failed to get submittal: %w", err)
	}

	// Parse JSON fields
	if err := json.Unmarshal([]byte(deliveryTrackingJSON), &submittal.DeliveryTracking); err != nil {
		submittal.DeliveryTracking = make(map[string]interface{})
	}
	if err := json.Unmarshal([]byte(teamAssignmentsJSON), &submittal.TeamAssignments); err != nil {
		submittal.TeamAssignments = make(map[string]interface{})
	}
	if err := json.Unmarshal([]byte(linkedDrawingsJSON), &submittal.LinkedDrawings); err != nil {
		submittal.LinkedDrawings = make(map[string]interface{})
	}
	if err := json.Unmarshal([]byte(referencesJSON), &submittal.References); err != nil {
		submittal.References = make(map[string]interface{})
	}
	if err := json.Unmarshal([]byte(procurementLogJSON), &submittal.ProcurementLog); err != nil {
		submittal.ProcurementLog = make(map[string]interface{})
	}
	if err := json.Unmarshal([]byte(approvalActionsJSON), &submittal.ApprovalActions); err != nil {
		submittal.ApprovalActions = make(map[string]interface{})
	}
	if err := json.Unmarshal([]byte(distributionListJSON), &submittal.DistributionList); err != nil {
		submittal.DistributionList = make([]string, 0)
	}
	if err := json.Unmarshal([]byte(notificationSettingsJSON), &submittal.NotificationSettings); err != nil {
		submittal.NotificationSettings = make(map[string]interface{})
	}
	if err := json.Unmarshal([]byte(tagsJSON), &submittal.Tags); err != nil {
		submittal.Tags = make([]string, 0)
	}
	if err := json.Unmarshal([]byte(customFieldsJSON), &submittal.CustomFields); err != nil {
		submittal.CustomFields = make(map[string]interface{})
	}

	// Calculate derived fields
	submittal.DaysOpen = calculateDaysOpen(submittal.CreatedAt)
	submittal.IsOverdue = isOverdue(submittal.RequiredApprovalDate, submittal.WorkflowStatus)

	// Get attachments
	attachments, _ := dao.GetSubmittalAttachments(ctx, submittalID)
	submittal.Attachments = attachments
	submittal.AttachmentCount = len(attachments)

	return &submittal, nil
}

// GetSubmittalsByProject retrieves submittals for a project with filtering
func (dao *SubmittalDao) GetSubmittalsByProject(ctx context.Context, projectID int64, filters map[string]string) ([]models.SubmittalResponse, error) {
	baseQuery := `
		SELECT s.id, s.project_id, s.org_id, s.location_id, s.submittal_number,
			   s.package_name, s.csi_division, s.csi_section, s.title, s.description,
			   s.submittal_type, s.specification_section, s.drawing_reference, s.trade_type,
			   s.priority, s.status, s.current_phase, s.ball_in_court, s.workflow_status,
			   s.revision_number, s.submitted_by, s.submitted_company_id, s.reviewed_by,
			   s.assigned_to, s.reviewer, s.approver, s.submitted_date, s.due_date,
			   s.required_approval_date, s.reviewed_date, s.approval_date,
			   s.fabrication_start_date, s.installation_date, s.review_comments,
			   s.lead_time_days, s.quantity_submitted, s.unit_of_measure,
			   s.delivery_tracking, s.team_assignments, s.linked_drawings, s.submittal_references,
			   s.procurement_log, s.approval_actions, s.distribution_list, s.notification_settings,
			   s.tags, s.custom_fields, s.created_at, s.created_by, s.updated_at, s.updated_by,
			   s.is_deleted,
			   p.name as project_name,
			   COALESCE(u_submitted.first_name, '') || ' ' || COALESCE(u_submitted.last_name, '') as submitted_by_name,
			   COALESCE(u_assigned.first_name, '') || ' ' || COALESCE(u_assigned.last_name, '') as assigned_to_name,
			   COALESCE(u_reviewer.first_name, '') || ' ' || COALESCE(u_reviewer.last_name, '') as reviewer_name,
			   COALESCE(u_approver.first_name, '') || ' ' || COALESCE(u_approver.last_name, '') as approver_name,
			   COALESCE(u_created.first_name, '') || ' ' || COALESCE(u_created.last_name, '') as created_by_name,
			   COALESCE(u_updated.first_name, '') || ' ' || COALESCE(u_updated.last_name, '') as updated_by_name
		FROM project.submittals s
		LEFT JOIN project.projects p ON s.project_id = p.id
		LEFT JOIN iam.users u_submitted ON s.submitted_by = u_submitted.id
		LEFT JOIN iam.users u_assigned ON s.assigned_to = u_assigned.id
		LEFT JOIN iam.users u_reviewer ON s.reviewer = u_reviewer.id
		LEFT JOIN iam.users u_approver ON s.approver = u_approver.id
		LEFT JOIN iam.users u_created ON s.created_by = u_created.id
		LEFT JOIN iam.users u_updated ON s.updated_by = u_updated.id
		WHERE s.project_id = $1 AND s.is_deleted = false`

	args := []interface{}{projectID}
	argIndex := 2

	// Build WHERE conditions based on filters
	conditions := []string{}

	if status := filters["status"]; status != "" {
		conditions = append(conditions, fmt.Sprintf("s.workflow_status = $%d", argIndex))
		args = append(args, status)
		argIndex++
	}

	if priority := filters["priority"]; priority != "" {
		conditions = append(conditions, fmt.Sprintf("s.priority = $%d", argIndex))
		args = append(args, priority)
		argIndex++
	}

	if csiDivision := filters["csi_division"]; csiDivision != "" {
		conditions = append(conditions, fmt.Sprintf("s.csi_division = $%d", argIndex))
		args = append(args, csiDivision)
		argIndex++
	}

	if ballInCourt := filters["ball_in_court"]; ballInCourt != "" {
		conditions = append(conditions, fmt.Sprintf("s.ball_in_court = $%d", argIndex))
		args = append(args, ballInCourt)
		argIndex++
	}

	if search := filters["search"]; search != "" {
		conditions = append(conditions, fmt.Sprintf(`(
			s.package_name ILIKE $%d OR
			s.title ILIKE $%d OR
			s.description ILIKE $%d OR
			s.submittal_number ILIKE $%d
		)`, argIndex, argIndex, argIndex, argIndex))
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern)
		argIndex++
	}

	// Add conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ordering
	sortField := filters["sort"]
	if sortField == "" {
		sortField = "created_at"
	}

	order := filters["order"]
	if order == "" {
		order = "desc"
	}

	baseQuery += fmt.Sprintf(" ORDER BY s.%s %s", sortField, strings.ToUpper(order))

	// Add pagination
	limit := 20
	if limitStr := filters["limit"]; limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	page := 1
	if pageStr := filters["page"]; pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	offset := (page - 1) * limit
	baseQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	rows, err := dao.DB.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to get submittals by project")
		return nil, fmt.Errorf("failed to get submittals: %w", err)
	}
	defer rows.Close()

	var submittals []models.SubmittalResponse
	for rows.Next() {
		var submittal models.SubmittalResponse
		var deliveryTrackingJSON, teamAssignmentsJSON, linkedDrawingsJSON, referencesJSON string
		var procurementLogJSON, approvalActionsJSON, distributionListJSON, notificationSettingsJSON string
		var tagsJSON, customFieldsJSON string

		err := rows.Scan(
			&submittal.ID, &submittal.ProjectID, &submittal.OrgID, &submittal.LocationID, &submittal.SubmittalNumber,
			&submittal.PackageName, &submittal.CSIDivision, &submittal.CSISection, &submittal.Title, &submittal.Description,
			&submittal.SubmittalType, &submittal.SpecificationSection, &submittal.DrawingReference, &submittal.TradeType,
			&submittal.Priority, &submittal.Status, &submittal.CurrentPhase, &submittal.BallInCourt, &submittal.WorkflowStatus,
			&submittal.RevisionNumber, &submittal.SubmittedBy, &submittal.SubmittedCompanyID, &submittal.ReviewedBy,
			&submittal.AssignedTo, &submittal.Reviewer, &submittal.Approver, &submittal.SubmittedDate, &submittal.DueDate,
			&submittal.RequiredApprovalDate, &submittal.ReviewedDate, &submittal.ApprovalDate,
			&submittal.FabricationStartDate, &submittal.InstallationDate, &submittal.ReviewComments,
			&submittal.LeadTimeDays, &submittal.QuantitySubmitted, &submittal.UnitOfMeasure,
			&deliveryTrackingJSON, &teamAssignmentsJSON, &linkedDrawingsJSON, &referencesJSON,
			&procurementLogJSON, &approvalActionsJSON, &distributionListJSON, &notificationSettingsJSON,
			&tagsJSON, &customFieldsJSON, &submittal.CreatedAt, &submittal.CreatedBy, &submittal.UpdatedAt, &submittal.UpdatedBy,
			&submittal.IsDeleted,
			&submittal.ProjectName, &submittal.SubmittedByName, &submittal.AssignedToName,
			&submittal.ReviewerName, &submittal.ApproverName, &submittal.CreatedByName, &submittal.LastModifiedByName,
		)

		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan submittal row")
			continue
		}

		// Parse JSON fields (simplified for list view)
		json.Unmarshal([]byte(deliveryTrackingJSON), &submittal.DeliveryTracking)
		json.Unmarshal([]byte(teamAssignmentsJSON), &submittal.TeamAssignments)
		json.Unmarshal([]byte(linkedDrawingsJSON), &submittal.LinkedDrawings)
		json.Unmarshal([]byte(referencesJSON), &submittal.References)
		json.Unmarshal([]byte(procurementLogJSON), &submittal.ProcurementLog)
		json.Unmarshal([]byte(approvalActionsJSON), &submittal.ApprovalActions)
		json.Unmarshal([]byte(distributionListJSON), &submittal.DistributionList)
		json.Unmarshal([]byte(notificationSettingsJSON), &submittal.NotificationSettings)
		json.Unmarshal([]byte(tagsJSON), &submittal.Tags)
		json.Unmarshal([]byte(customFieldsJSON), &submittal.CustomFields)

		// Calculate derived fields
		submittal.DaysOpen = calculateDaysOpen(submittal.CreatedAt)
		submittal.IsOverdue = isOverdue(submittal.RequiredApprovalDate, submittal.WorkflowStatus)

		submittals = append(submittals, submittal)
	}

	return submittals, nil
}

// UpdateSubmittal updates an existing submittal
func (dao *SubmittalDao) UpdateSubmittal(ctx context.Context, submittalID, userID, orgID int64, req *models.UpdateSubmittalRequest) (*models.SubmittalResponse, error) {
	// Handle workflow actions
	if req.Action != nil {
		action := &models.SubmittalWorkflowAction{
			Action:       *req.Action,
			Comments:     req.Comments,
			NextReviewer: req.NextReviewer,
		}
		return dao.ExecuteWorkflowAction(ctx, submittalID, userID, action)
	}

	// Build update query dynamically
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	// Basic information fields
	if req.PackageName != nil {
		setParts = append(setParts, fmt.Sprintf("package_name = $%d", argIndex))
		args = append(args, *req.PackageName)
		argIndex++
	}

	if req.CSIDivision != nil {
		setParts = append(setParts, fmt.Sprintf("csi_division = $%d", argIndex))
		args = append(args, *req.CSIDivision)
		argIndex++
	}

	if req.CSISection != nil {
		setParts = append(setParts, fmt.Sprintf("csi_section = $%d", argIndex))
		args = append(args, *req.CSISection)
		argIndex++
	}

	if req.Title != "" {
		setParts = append(setParts, fmt.Sprintf("title = $%d", argIndex))
		args = append(args, req.Title)
		argIndex++
	}

	if req.Description != nil {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *req.Description)
		argIndex++
	}

	if req.SubmittalType != "" {
		setParts = append(setParts, fmt.Sprintf("submittal_type = $%d", argIndex))
		args = append(args, req.SubmittalType)
		argIndex++
	}

	if req.SpecificationSection != nil {
		setParts = append(setParts, fmt.Sprintf("specification_section = $%d", argIndex))
		args = append(args, *req.SpecificationSection)
		argIndex++
	}

	if req.Priority != "" {
		setParts = append(setParts, fmt.Sprintf("priority = $%d", argIndex))
		args = append(args, req.Priority)
		argIndex++
	}

	// Assignment fields
	if req.AssignedTo != nil {
		setParts = append(setParts, fmt.Sprintf("assigned_to = $%d", argIndex))
		args = append(args, *req.AssignedTo)
		argIndex++
	}

	if req.Reviewer != nil {
		setParts = append(setParts, fmt.Sprintf("reviewer = $%d", argIndex))
		args = append(args, *req.Reviewer)
		argIndex++
	}

	if req.Approver != nil {
		setParts = append(setParts, fmt.Sprintf("approver = $%d", argIndex))
		args = append(args, *req.Approver)
		argIndex++
	}

	// Date fields (parse and validate)
	if req.RequiredApprovalDate != nil {
		if date, err := parseDate(*req.RequiredApprovalDate); err == nil {
			setParts = append(setParts, fmt.Sprintf("required_approval_date = $%d", argIndex))
			args = append(args, date)
			argIndex++
		}
	}

	if req.FabricationStartDate != nil {
		if date, err := parseDate(*req.FabricationStartDate); err == nil {
			setParts = append(setParts, fmt.Sprintf("fabrication_start_date = $%d", argIndex))
			args = append(args, date)
			argIndex++
		}
	}

	if req.InstallationDate != nil {
		if date, err := parseDate(*req.InstallationDate); err == nil {
			setParts = append(setParts, fmt.Sprintf("installation_date = $%d", argIndex))
			args = append(args, date)
			argIndex++
		}
	}

	// JSON fields
	if req.DeliveryTracking != nil {
		if jsonData, err := json.Marshal(req.DeliveryTracking); err == nil {
			setParts = append(setParts, fmt.Sprintf("delivery_tracking = $%d", argIndex))
			args = append(args, string(jsonData))
			argIndex++
		}
	}

	if req.TeamAssignments != nil {
		if jsonData, err := json.Marshal(req.TeamAssignments); err == nil {
			setParts = append(setParts, fmt.Sprintf("team_assignments = $%d", argIndex))
			args = append(args, string(jsonData))
			argIndex++
		}
	}

	if req.LinkedDrawings != nil {
		if jsonData, err := json.Marshal(req.LinkedDrawings); err == nil {
			setParts = append(setParts, fmt.Sprintf("linked_drawings = $%d", argIndex))
			args = append(args, string(jsonData))
			argIndex++
		}
	}

	if req.References != nil {
		if jsonData, err := json.Marshal(req.References); err == nil {
			setParts = append(setParts, fmt.Sprintf("submittal_references = $%d", argIndex))
			args = append(args, string(jsonData))
			argIndex++
		}
	}

	if req.ProcurementLog != nil {
		if jsonData, err := json.Marshal(req.ProcurementLog); err == nil {
			setParts = append(setParts, fmt.Sprintf("procurement_log = $%d", argIndex))
			args = append(args, string(jsonData))
			argIndex++
		}
	}

	if req.ApprovalActions != nil {
		if jsonData, err := json.Marshal(req.ApprovalActions); err == nil {
			setParts = append(setParts, fmt.Sprintf("approval_actions = $%d", argIndex))
			args = append(args, string(jsonData))
			argIndex++
		}
	}

	if req.DistributionList != nil {
		if jsonData, err := json.Marshal(req.DistributionList); err == nil {
			setParts = append(setParts, fmt.Sprintf("distribution_list = $%d", argIndex))
			args = append(args, string(jsonData))
			argIndex++
		}
	}

	if req.NotificationSettings != nil {
		if jsonData, err := json.Marshal(req.NotificationSettings); err == nil {
			setParts = append(setParts, fmt.Sprintf("notification_settings = $%d", argIndex))
			args = append(args, string(jsonData))
			argIndex++
		}
	}

	if req.Tags != nil {
		if jsonData, err := json.Marshal(req.Tags); err == nil {
			setParts = append(setParts, fmt.Sprintf("tags = $%d", argIndex))
			args = append(args, string(jsonData))
			argIndex++
		}
	}

	if req.CustomFields != nil {
		if jsonData, err := json.Marshal(req.CustomFields); err == nil {
			setParts = append(setParts, fmt.Sprintf("custom_fields = $%d", argIndex))
			args = append(args, string(jsonData))
			argIndex++
		}
	}

	// Always update updated_by and updated_at
	setParts = append(setParts, fmt.Sprintf("updated_by = $%d", argIndex))
	args = append(args, userID)
	argIndex++

	if len(setParts) == 1 { // Only updated_by was set
		return dao.GetSubmittal(ctx, submittalID)
	}

	query := fmt.Sprintf(`
		UPDATE project.submittals
		SET %s
		WHERE id = $%d AND is_deleted = false`,
		strings.Join(setParts, ", "), argIndex)
	args = append(args, submittalID)

	_, err := dao.DB.ExecContext(ctx, query, args...)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to update submittal")
		return nil, fmt.Errorf("failed to update submittal: %w", err)
	}

	// Add history entry
	history := &models.SubmittalHistory{
		SubmittalID: submittalID,
		Action:      "updated",
		Comment:     &[]string{"Submittal updated"}[0],
		CreatedBy:   userID,
	}
	dao.AddSubmittalHistory(ctx, history)

	return dao.GetSubmittal(ctx, submittalID)
}

// ExecuteWorkflowAction executes a workflow action on a submittal
func (dao *SubmittalDao) ExecuteWorkflowAction(ctx context.Context, submittalID, userID int64, action *models.SubmittalWorkflowAction) (*models.SubmittalResponse, error) {
	var newStatus, newPhase, newBallInCourt string
	var actionDescription string

	switch action.Action {
	case models.WorkflowActionSubmitForReview:
		newStatus = models.SubmittalStatusUnderReview
		newPhase = models.SubmittalPhaseReview
		newBallInCourt = models.BallInCourtArchitect
		actionDescription = "submitted for review"
	case models.WorkflowActionApprove:
		newStatus = models.SubmittalStatusApproved
		newPhase = models.SubmittalPhaseFabrication
		newBallInCourt = models.BallInCourtContractor
		actionDescription = "approved"
	case models.WorkflowActionApproveAsNoted:
		newStatus = models.SubmittalStatusApprovedAsNoted
		newPhase = models.SubmittalPhaseFabrication
		newBallInCourt = models.BallInCourtContractor
		actionDescription = "approved as noted"
	case models.WorkflowActionReviseResubmit:
		newStatus = models.SubmittalStatusReviseResubmit
		newPhase = models.SubmittalPhasePreparation
		newBallInCourt = models.BallInCourtContractor
		actionDescription = "requires revision and resubmission"
	case models.WorkflowActionReject:
		newStatus = models.SubmittalStatusRejected
		newPhase = models.SubmittalPhasePreparation
		newBallInCourt = models.BallInCourtContractor
		actionDescription = "rejected"
	case models.WorkflowActionMarkForInformation:
		newStatus = models.SubmittalStatusForInformationOnly
		newPhase = models.SubmittalPhaseCompleted
		newBallInCourt = models.BallInCourtContractor
		actionDescription = "marked for information only"
	default:
		return nil, fmt.Errorf("invalid workflow action: %s", action.Action)
	}

	// Override ball in court if specified
	if action.BallInCourtTransfer != nil {
		newBallInCourt = *action.BallInCourtTransfer
	}

	query := `
		UPDATE project.submittals
		SET workflow_status = $1, current_phase = $2, ball_in_court = $3,
			reviewer = $4, updated_by = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id = $6 AND is_deleted = false`

	_, err := dao.DB.ExecContext(ctx, query,
		newStatus, newPhase, newBallInCourt, action.NextReviewer, userID, submittalID)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to execute workflow action")
		return nil, fmt.Errorf("failed to execute workflow action: %w", err)
	}

	// Add history entry
	historyComment := actionDescription
	if action.Comments != nil {
		historyComment += ": " + *action.Comments
	}

	history := &models.SubmittalHistory{
		SubmittalID: submittalID,
		Action:      action.Action,
		Comment:     &historyComment,
		CreatedBy:   userID,
	}
	dao.AddSubmittalHistory(ctx, history)

	return dao.GetSubmittal(ctx, submittalID)
}

// DeleteSubmittal soft deletes a submittal

// GetSubmittalStats returns statistics for submittals in a project
func (dao *SubmittalDao) GetSubmittalStats(ctx context.Context, projectID int64) (*models.SubmittalStats, error) {
	stats := &models.SubmittalStats{
		ByStatus:        make(map[string]int),
		ByPriority:      make(map[string]int),
		ByBallInCourt:   make(map[string]int),
		DeliverySummary: make(map[string]int),
	}

	// Get overall stats
	query := `
		SELECT COUNT(*) as total,
			   COUNT(CASE WHEN required_approval_date < CURRENT_TIMESTAMP AND workflow_status IN ('pending_submission', 'under_review') THEN 1 END) as overdue
		FROM project.submittals
		WHERE project_id = $1 AND is_deleted = false`

	err := dao.DB.QueryRowContext(ctx, query, projectID).Scan(&stats.Total, &stats.Overdue)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to get submittal stats")
		return nil, fmt.Errorf("failed to get submittal stats: %w", err)
	}

	// Get stats by workflow status
	query = `
		SELECT workflow_status, COUNT(*)
		FROM project.submittals
		WHERE project_id = $1 AND is_deleted = false
		GROUP BY workflow_status`

	rows, err := dao.DB.QueryContext(ctx, query, projectID)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to get submittal stats by status")
		return stats, nil // Return partial stats
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err == nil {
			stats.ByStatus[status] = count
		}
	}

	// Get stats by priority
	query = `
		SELECT priority, COUNT(*)
		FROM project.submittals
		WHERE project_id = $1 AND is_deleted = false
		GROUP BY priority`

	rows, err = dao.DB.QueryContext(ctx, query, projectID)
	if err != nil {
		return stats, nil
	}
	defer rows.Close()

	for rows.Next() {
		var priority string
		var count int
		if err := rows.Scan(&priority, &count); err == nil {
			stats.ByPriority[priority] = count
		}
	}

	// Get stats by ball in court
	query = `
		SELECT ball_in_court, COUNT(*)
		FROM project.submittals
		WHERE project_id = $1 AND is_deleted = false
		GROUP BY ball_in_court`

	rows, err = dao.DB.QueryContext(ctx, query, projectID)
	if err != nil {
		return stats, nil
	}
	defer rows.Close()

	for rows.Next() {
		var ballInCourt string
		var count int
		if err := rows.Scan(&ballInCourt, &count); err == nil {
			stats.ByBallInCourt[ballInCourt] = count
		}
	}

	return stats, nil
}

// AddSubmittalAttachment adds an attachment to a submittal
func (dao *SubmittalDao) AddSubmittalAttachment(ctx context.Context, attachment *models.SubmittalAttachment) (*models.SubmittalAttachment, error) {
	query := `
		INSERT INTO project.submittal_attachments
		(submittal_id, file_name, file_path, file_size, file_type, attachment_type, uploaded_by, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	err := dao.DB.QueryRowContext(ctx, query,
		attachment.SubmittalID, attachment.FileName, attachment.FilePath, attachment.FileSize,
		attachment.FileType, attachment.AttachmentType, attachment.UploadedBy,
		attachment.CreatedBy, attachment.UpdatedBy,
	).Scan(&attachment.ID, &attachment.CreatedAt, &attachment.UpdatedAt)

	if err != nil {
		dao.Logger.WithError(err).Error("Failed to add submittal attachment")
		return nil, fmt.Errorf("failed to add attachment: %w", err)
	}

	return attachment, nil
}

// GetSubmittalAttachments retrieves attachments for a submittal
func (dao *SubmittalDao) GetSubmittalAttachments(ctx context.Context, submittalID int64) ([]models.SubmittalAttachment, error) {
	query := `
		SELECT id, submittal_id, file_name, file_path, file_size, file_type, attachment_type,
			   uploaded_by, created_at, created_by, updated_at, updated_by, is_deleted
		FROM project.submittal_attachments
		WHERE submittal_id = $1 AND is_deleted = false
		ORDER BY created_at`

	rows, err := dao.DB.QueryContext(ctx, query, submittalID)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to get submittal attachments")
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	defer rows.Close()

	var attachments []models.SubmittalAttachment
	for rows.Next() {
		var attachment models.SubmittalAttachment
		err := rows.Scan(
			&attachment.ID, &attachment.SubmittalID, &attachment.FileName, &attachment.FilePath,
			&attachment.FileSize, &attachment.FileType, &attachment.AttachmentType,
			&attachment.UploadedBy, &attachment.CreatedAt, &attachment.CreatedBy,
			&attachment.UpdatedAt, &attachment.UpdatedBy, &attachment.IsDeleted,
		)
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan attachment row")
			continue
		}
		attachments = append(attachments, attachment)
	}

	return attachments, nil
}

// DeleteSubmittalAttachment soft deletes an attachment

// AddSubmittalHistory adds an entry to the submittal history
func (dao *SubmittalDao) AddSubmittalHistory(ctx context.Context, history *models.SubmittalHistory) error {
	query := `
		INSERT INTO project.submittal_history
		(submittal_id, action, field_name, old_value, new_value, comment, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := dao.DB.ExecContext(ctx, query,
		history.SubmittalID, history.Action, history.FieldName,
		history.OldValue, history.NewValue, history.Comment, history.CreatedBy,
	)

	if err != nil {
		dao.Logger.WithError(err).Error("Failed to add submittal history")
		return fmt.Errorf("failed to add history: %w", err)
	}

	return nil
}

// Helper functions

func (dao *SubmittalDao) generateSubmittalNumber(ctx context.Context, projectID int64) (string, error) {
	var count int
	query := `SELECT COUNT(*) FROM project.submittals WHERE project_id = $1`
	err := dao.DB.QueryRowContext(ctx, query, projectID).Scan(&count)
	if err != nil {
		return "", err
	}

	year := time.Now().Year()
	return fmt.Sprintf("SUB-%d-%03d", year, count+1), nil
}

func parseDate(dateStr string) (*time.Time, error) {
	if dateStr == "" {
		return nil, nil
	}

	// Try different date formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("invalid date format: %s", dateStr)
}

func calculateDaysOpen(createdAt time.Time) int {
	return int(time.Since(createdAt).Hours() / 24)
}

func isOverdue(requiredApprovalDate *time.Time, workflowStatus string) bool {
	if requiredApprovalDate == nil {
		return false
	}

	// Only consider overdue if still pending or under review
	if workflowStatus != models.SubmittalStatusPendingSubmission &&
	   workflowStatus != models.SubmittalStatusUnderReview {
		return false
	}

	return time.Now().After(*requiredApprovalDate)
}