package data

import (
	"context"
	"database/sql"
	"fmt"
	"infrastructure/lib/models"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// RFIRepository defines the interface for RFI data operations
type RFIRepository interface {
	CreateRFI(ctx context.Context, rfi *models.RFI) (*models.RFI, error)
	GetRFI(ctx context.Context, rfiID int64) (*models.RFIResponse, error)
	GetRFIsByProject(ctx context.Context, projectID int64, filters map[string]string) ([]models.RFIResponse, error)
	UpdateRFI(ctx context.Context, rfiID int64, updates *models.UpdateRFIRequest, updatedBy int64) error
	UpdateRFIStatus(ctx context.Context, rfiID int64, status string, updatedBy int64, comment string) error
	DeleteRFI(ctx context.Context, rfiID int64, deletedBy int64) error
	SubmitRFI(ctx context.Context, rfiID int64, assignedTo *int64, submittedBy int64) error
	RespondToRFI(ctx context.Context, rfiID int64, response string, responseBy int64) error
	ApproveRFI(ctx context.Context, rfiID int64, approvedBy int64, comments string) error
	RejectRFI(ctx context.Context, rfiID int64, rejectedBy int64, reason string) error
	AddRFIComment(ctx context.Context, comment *models.RFIComment) error
	GetRFIComments(ctx context.Context, rfiID int64) ([]models.RFIComment, error)
	AddRFIAttachment(ctx context.Context, attachment *models.RFIAttachment) (*models.RFIAttachment, error)
	GetRFIAttachments(ctx context.Context, rfiID int64) ([]models.RFIAttachment, error)
	GenerateRFINumber(ctx context.Context, projectID int64) (string, error)
}

// RFIDao implements RFIRepository interface
type RFIDao struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

// NewRFIDao creates a new instance of RFIDao
func NewRFIDao(db *sql.DB, logger *logrus.Logger) RFIRepository {
	return &RFIDao{
		DB:     db,
		Logger: logger,
	}
}

// CreateRFI creates a new RFI
func (dao *RFIDao) CreateRFI(ctx context.Context, rfi *models.RFI) (*models.RFI, error) {
	query := `
		INSERT INTO project.rfis (
			project_id, org_id, location_id, rfi_number, subject,
			question, description, category, discipline, trade_type,
			project_phase, priority, status, submitted_by, assigned_to,
			reviewer_email, approver_email, cc_list, distribution_list,
			submitted_date, due_date, cost_impact, schedule_impact,
			cost_impact_amount, schedule_impact_days, cost_impact_details,
			schedule_impact_details, location_description, drawing_references,
			specification_references, related_submittals, related_change_events,
			related_rfis, workflow_type, requires_approval,
			urgency_justification, business_justification,
			created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, $35, $36, $37, $38, $39
		) RETURNING id, created_at, updated_at`
	
	err := dao.DB.QueryRowContext(ctx, query,
		rfi.ProjectID, rfi.OrgID, rfi.LocationID, rfi.RFINumber, rfi.Subject,
		rfi.Question, rfi.Description, rfi.Category, rfi.Discipline, rfi.TradeType,
		rfi.ProjectPhase, rfi.Priority, rfi.Status, rfi.SubmittedBy, rfi.AssignedTo,
		rfi.ReviewerEmail, rfi.ApproverEmail, pq.Array(rfi.CCList), pq.Array(rfi.DistributionList),
		rfi.SubmittedDate, rfi.DueDate, rfi.CostImpact, rfi.ScheduleImpact,
		rfi.CostImpactAmount, rfi.ScheduleImpactDays, rfi.CostImpactDetails,
		rfi.ScheduleImpactDetails, rfi.LocationDescription, rfi.DrawingReferences,
		rfi.SpecificationReferences, rfi.RelatedSubmittals, rfi.RelatedChangeEvents,
		pq.Array(rfi.RelatedRFIs), rfi.WorkflowType, rfi.RequiresApproval,
		rfi.UrgencyJustification, rfi.BusinessJustification,
		rfi.CreatedBy, rfi.UpdatedBy,
	).Scan(&rfi.ID, &rfi.CreatedAt, &rfi.UpdatedAt)
	
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to create RFI")
		return nil, fmt.Errorf("failed to create RFI: %w", err)
	}
	
	dao.Logger.WithFields(logrus.Fields{
		"rfi_id":     rfi.ID,
		"project_id": rfi.ProjectID,
	}).Info("RFI created successfully")
	
	return rfi, nil
}

// GetRFI retrieves a single RFI by ID
func (dao *RFIDao) GetRFI(ctx context.Context, rfiID int64) (*models.RFIResponse, error) {
	query := `
		SELECT 
			r.id, r.project_id, r.org_id, r.location_id, r.rfi_number,
			r.subject, r.question, r.description, r.category, r.discipline,
			r.trade_type, r.project_phase, r.priority, r.status,
			r.submitted_by, r.assigned_to, r.reviewer_email, r.approver_email,
			r.cc_list, r.distribution_list, r.submitted_date, r.due_date,
			r.response_date, r.closed_date, r.response, r.response_status,
			r.cost_impact, r.schedule_impact, r.cost_impact_amount,
			r.schedule_impact_days, r.cost_impact_details, r.schedule_impact_details,
			r.location_description, r.drawing_references, r.specification_references,
			r.related_submittals, r.related_change_events, r.related_rfis,
			r.workflow_type, r.requires_approval, r.approval_status,
			r.approved_by, r.approval_date, r.approval_comments,
			r.urgency_justification, r.business_justification,
			r.days_open, r.is_overdue,
			r.created_at, r.created_by, r.updated_at, r.updated_by,
			p.name as project_name,
			l.name as location_name,
			CONCAT(u1.first_name, ' ', u1.last_name) as submitted_by_name,
			CONCAT(u2.first_name, ' ', u2.last_name) as assigned_to_name,
			CONCAT(u3.first_name, ' ', u3.last_name) as response_by_name,
			CONCAT(u4.first_name, ' ', u4.last_name) as approved_by_name,
			(SELECT COUNT(*) FROM project.rfi_comments WHERE rfi_id = r.id AND is_deleted = FALSE) as comment_count,
			(SELECT COUNT(*) FROM project.rfi_attachments WHERE rfi_id = r.id AND is_deleted = FALSE) as attachment_count
		FROM project.rfis r
		LEFT JOIN project.projects p ON r.project_id = p.id
		LEFT JOIN iam.locations l ON r.location_id = l.id
		LEFT JOIN iam.users u1 ON r.submitted_by = u1.id
		LEFT JOIN iam.users u2 ON r.assigned_to = u2.id
		LEFT JOIN iam.users u3 ON r.response_by = u3.id
		LEFT JOIN iam.users u4 ON r.approved_by = u4.id
		WHERE r.id = $1 AND r.is_deleted = FALSE`
	
	var rfi models.RFIResponse
	var ccList, distributionList, relatedRFIs pq.StringArray
	
	// Nullable fields
	var locationID, assignedTo, responseBy, approvedBy sql.NullInt64
	var locationName, assignedToName, responseByName, approvedByName sql.NullString
	var description, category, discipline, tradeType, projectPhase sql.NullString
	var reviewerEmail, approverEmail, response, responseStatus sql.NullString
	var costImpactAmount sql.NullFloat64
	var submittedDate, dueDate, responseDate, closedDate, approvalDate *time.Time
	var locationDesc, drawingRefs, specRefs, relatedSub, relatedChange sql.NullString
	var approvalStatus, approvalComments, urgencyJust, businessJust sql.NullString
	
	err := dao.DB.QueryRowContext(ctx, query, rfiID).Scan(
		&rfi.ID, &rfi.ProjectID, &rfi.OrgID, &locationID, &rfi.RFINumber,
		&rfi.Subject, &rfi.Question, &description, &category, &discipline,
		&tradeType, &projectPhase, &rfi.Priority, &rfi.Status,
		&rfi.SubmittedBy, &assignedTo, &reviewerEmail, &approverEmail,
		&ccList, &distributionList, &submittedDate, &dueDate,
		&responseDate, &closedDate, &response, &responseStatus,
		&rfi.CostImpact, &rfi.ScheduleImpact, &costImpactAmount,
		&rfi.ScheduleImpactDays, &rfi.CostImpactDetails, &rfi.ScheduleImpactDetails,
		&locationDesc, &drawingRefs, &specRefs,
		&relatedSub, &relatedChange, &relatedRFIs,
		&rfi.WorkflowType, &rfi.RequiresApproval, &approvalStatus,
		&approvedBy, &approvalDate, &approvalComments,
		&urgencyJust, &businessJust,
		&rfi.DaysOpen, &rfi.IsOverdue,
		&rfi.CreatedAt, &rfi.CreatedBy, &rfi.UpdatedAt, &rfi.UpdatedBy,
		&rfi.ProjectName,
		&locationName,
		&rfi.SubmittedByName,
		&assignedToName,
		&responseByName,
		&approvedByName,
		&rfi.CommentCount,
		&rfi.AttachmentCount,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("RFI not found")
		}
		dao.Logger.WithError(err).Error("Failed to get RFI")
		return nil, fmt.Errorf("failed to get RFI: %w", err)
	}
	
	// Handle nullable fields
	if locationID.Valid {
		rfi.LocationID = &locationID.Int64
	}
	if assignedTo.Valid {
		rfi.AssignedTo = &assignedTo.Int64
	}
	if responseBy.Valid {
		rfi.ResponseBy = &responseBy.Int64
	}
	if approvedBy.Valid {
		rfi.ApprovedBy = &approvedBy.Int64
	}
	if locationName.Valid {
		rfi.LocationName = locationName.String
	}
	if assignedToName.Valid {
		rfi.AssignedToName = assignedToName.String
	}
	if responseByName.Valid {
		rfi.ResponseByName = responseByName.String
	}
	if approvedByName.Valid {
		rfi.ApprovedByName = approvedByName.String
	}
	
	// Handle other nullable string fields
	if description.Valid {
		rfi.Description = description.String
	}
	if category.Valid {
		rfi.Category = category.String
	}
	if discipline.Valid {
		rfi.Discipline = discipline.String
	}
	if tradeType.Valid {
		rfi.TradeType = tradeType.String
	}
	if projectPhase.Valid {
		rfi.ProjectPhase = projectPhase.String
	}
	if reviewerEmail.Valid {
		rfi.ReviewerEmail = reviewerEmail.String
	}
	if approverEmail.Valid {
		rfi.ApproverEmail = approverEmail.String
	}
	if response.Valid {
		rfi.Response = response.String
	}
	if responseStatus.Valid {
		rfi.ResponseStatus = responseStatus.String
	}
	if costImpactAmount.Valid {
		rfi.CostImpactAmount = &costImpactAmount.Float64
	}
	if locationDesc.Valid {
		rfi.LocationDescription = locationDesc.String
	}
	if drawingRefs.Valid {
		rfi.DrawingReferences = drawingRefs.String
	}
	if specRefs.Valid {
		rfi.SpecificationReferences = specRefs.String
	}
	if relatedSub.Valid {
		rfi.RelatedSubmittals = relatedSub.String
	}
	if relatedChange.Valid {
		rfi.RelatedChangeEvents = relatedChange.String
	}
	if approvalStatus.Valid {
		rfi.ApprovalStatus = approvalStatus.String
	}
	if approvalComments.Valid {
		rfi.ApprovalComments = approvalComments.String
	}
	if urgencyJust.Valid {
		rfi.UrgencyJustification = urgencyJust.String
	}
	if businessJust.Valid {
		rfi.BusinessJustification = businessJust.String
	}
	
	// Handle time fields
	rfi.SubmittedDate = submittedDate
	rfi.DueDate = dueDate
	rfi.ResponseDate = responseDate
	rfi.ClosedDate = closedDate
	rfi.ApprovalDate = approvalDate
	
	// Handle arrays
	rfi.CCList = []string(ccList)
	rfi.DistributionList = []string(distributionList)
	rfi.RelatedRFIs = []string(relatedRFIs)
	
	return &rfi, nil
}

// GetRFIsByProject retrieves all RFIs for a specific project with optional filters
func (dao *RFIDao) GetRFIsByProject(ctx context.Context, projectID int64, filters map[string]string) ([]models.RFIResponse, error) {
	query := `
		SELECT 
			r.id, r.project_id, r.org_id, r.location_id, r.rfi_number,
			r.subject, r.question, r.description, r.category, r.discipline,
			r.trade_type, r.project_phase, r.priority, r.status,
			r.submitted_by, r.assigned_to, r.reviewer_email, r.approver_email,
			r.cc_list, r.distribution_list, r.submitted_date, r.due_date,
			r.response_date, r.closed_date, r.response, r.response_status,
			r.cost_impact, r.schedule_impact, r.cost_impact_amount,
			r.schedule_impact_days, r.cost_impact_details, r.schedule_impact_details,
			r.location_description, r.drawing_references, r.specification_references,
			r.related_submittals, r.related_change_events, r.related_rfis,
			r.workflow_type, r.requires_approval, r.approval_status,
			r.approved_by, r.approval_date, r.approval_comments,
			r.urgency_justification, r.business_justification,
			r.days_open, r.is_overdue,
			r.created_at, r.created_by, r.updated_at, r.updated_by,
			p.name as project_name,
			l.name as location_name,
			CONCAT(u1.first_name, ' ', u1.last_name) as submitted_by_name,
			CONCAT(u2.first_name, ' ', u2.last_name) as assigned_to_name,
			CONCAT(u3.first_name, ' ', u3.last_name) as response_by_name,
			CONCAT(u4.first_name, ' ', u4.last_name) as approved_by_name
		FROM project.rfis r
		LEFT JOIN project.projects p ON r.project_id = p.id
		LEFT JOIN iam.locations l ON r.location_id = l.id
		LEFT JOIN iam.users u1 ON r.submitted_by = u1.id
		LEFT JOIN iam.users u2 ON r.assigned_to = u2.id
		LEFT JOIN iam.users u3 ON r.response_by = u3.id
		LEFT JOIN iam.users u4 ON r.approved_by = u4.id
		WHERE r.project_id = $1 AND r.is_deleted = FALSE`
	
	// Add filters
	args := []interface{}{projectID}
	argIndex := 2
	
	if status, ok := filters["status"]; ok && status != "" {
		query += fmt.Sprintf(" AND r.status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}
	
	if priority, ok := filters["priority"]; ok && priority != "" {
		query += fmt.Sprintf(" AND r.priority = $%d", argIndex)
		args = append(args, priority)
		argIndex++
	}
	
	if category, ok := filters["category"]; ok && category != "" {
		query += fmt.Sprintf(" AND r.category = $%d", argIndex)
		args = append(args, category)
		argIndex++
	}
	
	if discipline, ok := filters["discipline"]; ok && discipline != "" {
		query += fmt.Sprintf(" AND r.discipline = $%d", argIndex)
		args = append(args, discipline)
		argIndex++
	}
	
	if assignedTo, ok := filters["assigned_to"]; ok && assignedTo != "" {
		query += fmt.Sprintf(" AND r.assigned_to = $%d", argIndex)
		args = append(args, assignedTo)
		argIndex++
	}
	
	if submittedBy, ok := filters["submitted_by"]; ok && submittedBy != "" {
		query += fmt.Sprintf(" AND r.submitted_by = $%d", argIndex)
		args = append(args, submittedBy)
		argIndex++
	}
	
	// Add ordering
	query += " ORDER BY r.created_at DESC"
	
	rows, err := dao.DB.QueryContext(ctx, query, args...)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to query RFIs")
		return nil, fmt.Errorf("failed to query RFIs: %w", err)
	}
	defer rows.Close()
	
	var rfis []models.RFIResponse
	for rows.Next() {
		var rfi models.RFIResponse
		var ccList, distributionList, relatedRFIs pq.StringArray
		
		// Nullable fields
		var locationID, assignedTo, responseBy, approvedBy sql.NullInt64
		var locationName, assignedToName, responseByName, approvedByName sql.NullString
		var description, category, discipline, tradeType, projectPhase sql.NullString
		var reviewerEmail, approverEmail, response, responseStatus sql.NullString
		var costImpactAmount sql.NullFloat64
		var submittedDate, dueDate, responseDate, closedDate, approvalDate *time.Time
		var locationDesc, drawingRefs, specRefs, relatedSub, relatedChange sql.NullString
		var approvalStatus, approvalComments, urgencyJust, businessJust sql.NullString
		
		err := rows.Scan(
			&rfi.ID, &rfi.ProjectID, &rfi.OrgID, &locationID, &rfi.RFINumber,
			&rfi.Subject, &rfi.Question, &description, &category, &discipline,
			&tradeType, &projectPhase, &rfi.Priority, &rfi.Status,
			&rfi.SubmittedBy, &assignedTo, &reviewerEmail, &approverEmail,
			&ccList, &distributionList, &submittedDate, &dueDate,
			&responseDate, &closedDate, &response, &responseStatus,
			&rfi.CostImpact, &rfi.ScheduleImpact, &costImpactAmount,
			&rfi.ScheduleImpactDays, &rfi.CostImpactDetails, &rfi.ScheduleImpactDetails,
			&locationDesc, &drawingRefs, &specRefs,
			&relatedSub, &relatedChange, &relatedRFIs,
			&rfi.WorkflowType, &rfi.RequiresApproval, &approvalStatus,
			&approvedBy, &approvalDate, &approvalComments,
			&urgencyJust, &businessJust,
			&rfi.DaysOpen, &rfi.IsOverdue,
			&rfi.CreatedAt, &rfi.CreatedBy, &rfi.UpdatedAt, &rfi.UpdatedBy,
			&rfi.ProjectName,
			&locationName,
			&rfi.SubmittedByName,
			&assignedToName,
			&responseByName,
			&approvedByName,
		)
		
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan RFI row")
			return nil, fmt.Errorf("failed to scan RFI: %w", err)
		}
		
		// Handle nullable fields (same as GetRFI)
		if locationID.Valid {
			rfi.LocationID = &locationID.Int64
		}
		if assignedTo.Valid {
			rfi.AssignedTo = &assignedTo.Int64
		}
		if responseBy.Valid {
			rfi.ResponseBy = &responseBy.Int64
		}
		if approvedBy.Valid {
			rfi.ApprovedBy = &approvedBy.Int64
		}
		if locationName.Valid {
			rfi.LocationName = locationName.String
		}
		if assignedToName.Valid {
			rfi.AssignedToName = assignedToName.String
		}
		if responseByName.Valid {
			rfi.ResponseByName = responseByName.String
		}
		if approvedByName.Valid {
			rfi.ApprovedByName = approvedByName.String
		}
		
		// Handle other nullable string fields
		if description.Valid {
			rfi.Description = description.String
		}
		if category.Valid {
			rfi.Category = category.String
		}
		if discipline.Valid {
			rfi.Discipline = discipline.String
		}
		if tradeType.Valid {
			rfi.TradeType = tradeType.String
		}
		if projectPhase.Valid {
			rfi.ProjectPhase = projectPhase.String
		}
		if reviewerEmail.Valid {
			rfi.ReviewerEmail = reviewerEmail.String
		}
		if approverEmail.Valid {
			rfi.ApproverEmail = approverEmail.String
		}
		if response.Valid {
			rfi.Response = response.String
		}
		if responseStatus.Valid {
			rfi.ResponseStatus = responseStatus.String
		}
		if costImpactAmount.Valid {
			rfi.CostImpactAmount = &costImpactAmount.Float64
		}
		if locationDesc.Valid {
			rfi.LocationDescription = locationDesc.String
		}
		if drawingRefs.Valid {
			rfi.DrawingReferences = drawingRefs.String
		}
		if specRefs.Valid {
			rfi.SpecificationReferences = specRefs.String
		}
		if relatedSub.Valid {
			rfi.RelatedSubmittals = relatedSub.String
		}
		if relatedChange.Valid {
			rfi.RelatedChangeEvents = relatedChange.String
		}
		if approvalStatus.Valid {
			rfi.ApprovalStatus = approvalStatus.String
		}
		if approvalComments.Valid {
			rfi.ApprovalComments = approvalComments.String
		}
		if urgencyJust.Valid {
			rfi.UrgencyJustification = urgencyJust.String
		}
		if businessJust.Valid {
			rfi.BusinessJustification = businessJust.String
		}
		
		// Handle time fields
		rfi.SubmittedDate = submittedDate
		rfi.DueDate = dueDate
		rfi.ResponseDate = responseDate
		rfi.ClosedDate = closedDate
		rfi.ApprovalDate = approvalDate
		
		// Handle arrays
		rfi.CCList = []string(ccList)
		rfi.DistributionList = []string(distributionList)
		rfi.RelatedRFIs = []string(relatedRFIs)
		
		rfis = append(rfis, rfi)
	}
	
	if err = rows.Err(); err != nil {
		dao.Logger.WithError(err).Error("Error iterating RFI rows")
		return nil, fmt.Errorf("error iterating RFIs: %w", err)
	}
	
	dao.Logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"count":      len(rfis),
	}).Debug("Successfully retrieved RFIs for project")
	
	return rfis, nil
}

// UpdateRFI updates an existing RFI
func (dao *RFIDao) UpdateRFI(ctx context.Context, rfiID int64, updates *models.UpdateRFIRequest, updatedBy int64) error {
	var setClauses []string
	var args []interface{}
	argIndex := 1

	// Build update query dynamically based on provided fields
	if updates.Subject != "" {
		setClauses = append(setClauses, fmt.Sprintf("subject = $%d", argIndex))
		args = append(args, updates.Subject)
		argIndex++
	}
	
	if updates.Question != "" {
		setClauses = append(setClauses, fmt.Sprintf("question = $%d", argIndex))
		args = append(args, updates.Question)
		argIndex++
	}
	
	if updates.Description != "" {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, updates.Description)
		argIndex++
	}
	
	if updates.Category != "" {
		setClauses = append(setClauses, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, updates.Category)
		argIndex++
	}
	
	if updates.Discipline != "" {
		setClauses = append(setClauses, fmt.Sprintf("discipline = $%d", argIndex))
		args = append(args, updates.Discipline)
		argIndex++
	}
	
	if updates.Priority != "" {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argIndex))
		args = append(args, updates.Priority)
		argIndex++
	}
	
	if updates.AssignedTo != nil {
		setClauses = append(setClauses, fmt.Sprintf("assigned_to = $%d", argIndex))
		args = append(args, *updates.AssignedTo)
		argIndex++
	}
	
	if updates.DueDate != "" {
		// Parse the date string and convert to time.Time
		if parsedDate, err := time.Parse("2006-01-02", updates.DueDate); err == nil {
			setClauses = append(setClauses, fmt.Sprintf("due_date = $%d", argIndex))
			args = append(args, parsedDate)
			argIndex++
		}
	}
	
	// Always update updated_by and updated_at
	setClauses = append(setClauses, fmt.Sprintf("updated_by = $%d", argIndex))
	args = append(args, updatedBy)
	argIndex++
	
	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++
	
	// Add WHERE clause
	args = append(args, rfiID)
	
	query := fmt.Sprintf(
		"UPDATE project.rfis SET %s WHERE id = $%d AND is_deleted = FALSE",
		strings.Join(setClauses, ", "),
		argIndex,
	)
	
	result, err := dao.DB.ExecContext(ctx, query, args...)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to update RFI")
		return fmt.Errorf("failed to update RFI: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("RFI not found or already deleted")
	}
	
	dao.Logger.WithFields(logrus.Fields{
		"rfi_id":     rfiID,
		"updated_by": updatedBy,
	}).Info("RFI updated successfully")
	
	return nil
}

// UpdateRFIStatus updates the status of an RFI
func (dao *RFIDao) UpdateRFIStatus(ctx context.Context, rfiID int64, status string, updatedBy int64, comment string) error {
	tx, err := dao.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Get current status
	var currentStatus string
	err = tx.QueryRowContext(ctx, 
		"SELECT status FROM project.rfis WHERE id = $1 AND is_deleted = FALSE",
		rfiID,
	).Scan(&currentStatus)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("RFI not found")
		}
		return fmt.Errorf("failed to get current status: %w", err)
	}
	
	// Update status
	query := `
		UPDATE project.rfis 
		SET status = $1, updated_by = $2, updated_at = $3
		WHERE id = $4 AND is_deleted = FALSE`
	
	_, err = tx.ExecContext(ctx, query, status, updatedBy, time.Now(), rfiID)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	
	// Add status change comment
	commentQuery := `
		INSERT INTO project.rfi_comments (
			rfi_id, comment, comment_type, previous_value, new_value,
			created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	commentText := comment
	if commentText == "" {
		commentText = fmt.Sprintf("Status changed from %s to %s", currentStatus, status)
	}
	
	_, err = tx.ExecContext(ctx, commentQuery,
		rfiID, commentText, models.RFICommentTypeStatusChange,
		currentStatus, status, updatedBy, updatedBy,
	)
	
	if err != nil {
		return fmt.Errorf("failed to add status change comment: %w", err)
	}
	
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	dao.Logger.WithFields(logrus.Fields{
		"rfi_id":        rfiID,
		"old_status":    currentStatus,
		"new_status":    status,
		"updated_by":    updatedBy,
	}).Info("RFI status updated successfully")
	
	return nil
}

// DeleteRFI soft deletes an RFI
func (dao *RFIDao) DeleteRFI(ctx context.Context, rfiID int64, deletedBy int64) error {
	query := `
		UPDATE project.rfis 
		SET is_deleted = TRUE, updated_by = $1, updated_at = $2
		WHERE id = $3 AND is_deleted = FALSE`
	
	result, err := dao.DB.ExecContext(ctx, query, deletedBy, time.Now(), rfiID)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to delete RFI")
		return fmt.Errorf("failed to delete RFI: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("RFI not found or already deleted")
	}
	
	dao.Logger.WithFields(logrus.Fields{
		"rfi_id":     rfiID,
		"deleted_by": deletedBy,
	}).Info("RFI deleted successfully")
	
	return nil
}

// SubmitRFI submits an RFI for review
func (dao *RFIDao) SubmitRFI(ctx context.Context, rfiID int64, assignedTo *int64, submittedBy int64) error {
	query := `
		UPDATE project.rfis 
		SET status = $1, submitted_date = $2, assigned_to = $3, updated_by = $4, updated_at = $5
		WHERE id = $6 AND is_deleted = FALSE AND status = $7`
	
	result, err := dao.DB.ExecContext(ctx, query,
		models.RFIStatusSubmitted, time.Now(), assignedTo, submittedBy, time.Now(),
		rfiID, models.RFIStatusDraft,
	)
	
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to submit RFI")
		return fmt.Errorf("failed to submit RFI: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("RFI not found, already deleted, or not in draft status")
	}
	
	return nil
}

// RespondToRFI adds a response to an RFI
func (dao *RFIDao) RespondToRFI(ctx context.Context, rfiID int64, response string, responseBy int64) error {
	query := `
		UPDATE project.rfis 
		SET response = $1, response_by = $2, response_date = $3, 
		    status = $4, updated_by = $5, updated_at = $6
		WHERE id = $7 AND is_deleted = FALSE`
	
	_, err := dao.DB.ExecContext(ctx, query,
		response, responseBy, time.Now(),
		models.RFIStatusResponded, responseBy, time.Now(),
		rfiID,
	)
	
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to respond to RFI")
		return fmt.Errorf("failed to respond to RFI: %w", err)
	}
	
	return nil
}

// ApproveRFI approves an RFI
func (dao *RFIDao) ApproveRFI(ctx context.Context, rfiID int64, approvedBy int64, comments string) error {
	query := `
		UPDATE project.rfis 
		SET approval_status = 'approved', approved_by = $1, approval_date = $2,
		    approval_comments = $3, updated_by = $4, updated_at = $5
		WHERE id = $6 AND is_deleted = FALSE AND requires_approval = TRUE`
	
	_, err := dao.DB.ExecContext(ctx, query,
		approvedBy, time.Now(), comments, approvedBy, time.Now(), rfiID,
	)
	
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to approve RFI")
		return fmt.Errorf("failed to approve RFI: %w", err)
	}
	
	return nil
}

// RejectRFI rejects an RFI
func (dao *RFIDao) RejectRFI(ctx context.Context, rfiID int64, rejectedBy int64, reason string) error {
	query := `
		UPDATE project.rfis 
		SET approval_status = 'rejected', approved_by = $1, approval_date = $2,
		    approval_comments = $3, updated_by = $4, updated_at = $5
		WHERE id = $6 AND is_deleted = FALSE AND requires_approval = TRUE`
	
	_, err := dao.DB.ExecContext(ctx, query,
		rejectedBy, time.Now(), reason, rejectedBy, time.Now(), rfiID,
	)
	
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to reject RFI")
		return fmt.Errorf("failed to reject RFI: %w", err)
	}
	
	return nil
}

// AddRFIComment adds a comment to an RFI
func (dao *RFIDao) AddRFIComment(ctx context.Context, comment *models.RFIComment) error {
	query := `
		INSERT INTO project.rfi_comments (
			rfi_id, comment, comment_type, previous_value, new_value,
			created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`
	
	err := dao.DB.QueryRowContext(ctx, query,
		comment.RFIID, comment.Comment, comment.CommentType,
		comment.PreviousValue, comment.NewValue,
		comment.CreatedBy, comment.UpdatedBy,
	).Scan(&comment.ID, &comment.CreatedAt, &comment.UpdatedAt)
	
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to add RFI comment")
		return fmt.Errorf("failed to add RFI comment: %w", err)
	}
	
	return nil
}

// GetRFIComments retrieves all comments for an RFI
func (dao *RFIDao) GetRFIComments(ctx context.Context, rfiID int64) ([]models.RFIComment, error) {
	query := `
		SELECT 
			c.id, c.rfi_id, c.comment, c.comment_type,
			c.previous_value, c.new_value,
			c.created_at, c.created_by, c.updated_at, c.updated_by,
			CONCAT(u.first_name, ' ', u.last_name) as created_by_name
		FROM project.rfi_comments c
		LEFT JOIN iam.users u ON c.created_by = u.id
		WHERE c.rfi_id = $1 AND c.is_deleted = FALSE
		ORDER BY c.created_at DESC`
	
	rows, err := dao.DB.QueryContext(ctx, query, rfiID)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to get RFI comments")
		return nil, fmt.Errorf("failed to get RFI comments: %w", err)
	}
	defer rows.Close()
	
	var comments []models.RFIComment
	for rows.Next() {
		var comment models.RFIComment
		var previousValue, newValue, createdByName sql.NullString
		
		err := rows.Scan(
			&comment.ID, &comment.RFIID, &comment.Comment, &comment.CommentType,
			&previousValue, &newValue,
			&comment.CreatedAt, &comment.CreatedBy, &comment.UpdatedAt, &comment.UpdatedBy,
			&createdByName,
		)
		
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan RFI comment")
			return nil, fmt.Errorf("failed to scan RFI comment: %w", err)
		}
		
		if previousValue.Valid {
			comment.PreviousValue = previousValue.String
		}
		if newValue.Valid {
			comment.NewValue = newValue.String
		}
		if createdByName.Valid {
			comment.CreatedByName = createdByName.String
		}
		
		comments = append(comments, comment)
	}
	
	return comments, nil
}

// AddRFIAttachment adds an attachment to an RFI
func (dao *RFIDao) AddRFIAttachment(ctx context.Context, attachment *models.RFIAttachment) (*models.RFIAttachment, error) {
	query := `
		INSERT INTO project.rfi_attachments (
			rfi_id, filename, file_type, file_size, description,
			s3_bucket, s3_key, s3_url, attachment_type,
			uploaded_by, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, upload_date, created_at, updated_at`
	
	err := dao.DB.QueryRowContext(ctx, query,
		attachment.RFIID, attachment.Filename, attachment.FileType,
		attachment.FileSize, attachment.Description,
		attachment.S3Bucket, attachment.S3Key, attachment.S3URL,
		attachment.AttachmentType, attachment.UploadedBy,
		attachment.CreatedBy, attachment.UpdatedBy,
	).Scan(&attachment.ID, &attachment.UploadDate, &attachment.CreatedAt, &attachment.UpdatedAt)
	
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to add RFI attachment")
		return nil, fmt.Errorf("failed to add RFI attachment: %w", err)
	}
	
	return attachment, nil
}

// GetRFIAttachments retrieves all attachments for an RFI
func (dao *RFIDao) GetRFIAttachments(ctx context.Context, rfiID int64) ([]models.RFIAttachment, error) {
	query := `
		SELECT 
			id, rfi_id, filename, file_type, file_size, description,
			s3_bucket, s3_key, s3_url, attachment_type,
			uploaded_by, upload_date, created_at, created_by,
			updated_at, updated_by
		FROM project.rfi_attachments
		WHERE rfi_id = $1 AND is_deleted = FALSE
		ORDER BY created_at DESC`
	
	rows, err := dao.DB.QueryContext(ctx, query, rfiID)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to get RFI attachments")
		return nil, fmt.Errorf("failed to get RFI attachments: %w", err)
	}
	defer rows.Close()
	
	var attachments []models.RFIAttachment
	for rows.Next() {
		var attachment models.RFIAttachment
		var fileType, description, s3Bucket, s3Key, s3URL sql.NullString
		var fileSize sql.NullInt64
		
		err := rows.Scan(
			&attachment.ID, &attachment.RFIID, &attachment.Filename,
			&fileType, &fileSize, &description,
			&s3Bucket, &s3Key, &s3URL,
			&attachment.AttachmentType, &attachment.UploadedBy,
			&attachment.UploadDate, &attachment.CreatedAt, &attachment.CreatedBy,
			&attachment.UpdatedAt, &attachment.UpdatedBy,
		)
		
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan RFI attachment")
			return nil, fmt.Errorf("failed to scan RFI attachment: %w", err)
		}
		
		if fileType.Valid {
			attachment.FileType = fileType.String
		}
		if fileSize.Valid {
			attachment.FileSize = fileSize.Int64
		}
		if description.Valid {
			attachment.Description = description.String
		}
		if s3Bucket.Valid {
			attachment.S3Bucket = s3Bucket.String
		}
		if s3Key.Valid {
			attachment.S3Key = s3Key.String
		}
		if s3URL.Valid {
			attachment.S3URL = s3URL.String
		}
		
		attachments = append(attachments, attachment)
	}
	
	return attachments, nil
}

// GenerateRFINumber generates a unique RFI number for a project
func (dao *RFIDao) GenerateRFINumber(ctx context.Context, projectID int64) (string, error) {
	// Get project details
	var projectNumber string
	var projectYear int
	
	query := `
		SELECT 
			COALESCE(project_number, CAST(id AS VARCHAR)),
			EXTRACT(YEAR FROM COALESCE(start_date, created_at))
		FROM project.projects 
		WHERE id = $1`
	
	err := dao.DB.QueryRowContext(ctx, query, projectID).Scan(&projectNumber, &projectYear)
	if err != nil {
		return "", fmt.Errorf("failed to get project details: %w", err)
	}
	
	// Get the next sequence number for this project
	var maxNumber int
	sequenceQuery := `
		SELECT COALESCE(MAX(
			CAST(
				SUBSTRING(rfi_number FROM '[0-9]+$') AS INTEGER
			)
		), 0)
		FROM project.rfis
		WHERE project_id = $1 AND rfi_number LIKE $2`
	
	prefix := fmt.Sprintf("RFI-%d-", projectYear)
	err = dao.DB.QueryRowContext(ctx, sequenceQuery, projectID, prefix+"%").Scan(&maxNumber)
	if err != nil {
		return "", fmt.Errorf("failed to get max RFI number: %w", err)
	}
	
	// Generate new RFI number
	newNumber := fmt.Sprintf("%s%04d", prefix, maxNumber+1)
	
	dao.Logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"rfi_number": newNumber,
	}).Debug("Generated RFI number")
	
	return newNumber, nil
}