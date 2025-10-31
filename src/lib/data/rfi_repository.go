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
	CreateRFI(ctx context.Context, projectID, userID, orgID int64, req *models.CreateRFIRequest) (*models.RFIResponse, error)
	GetRFI(ctx context.Context, rfiID int64) (*models.RFIResponse, error)
	GetRFIsByProject(ctx context.Context, projectID int64, filters map[string]string) ([]models.RFIResponse, error)
	UpdateRFI(ctx context.Context, rfiID, userID, orgID int64, req *models.UpdateRFIRequest) (*models.RFIResponse, error)
	DeleteRFI(ctx context.Context, rfiID int64, deletedBy int64) error
	AddRFIComment(ctx context.Context, rfiID, userID int64, req *models.CreateRFICommentRequest) (*models.RFIComment, error)
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

// Helper function to fetch user details as AssignedUser
func (dao *RFIDao) getUserDetails(ctx context.Context, userID int64) (*models.AssignedUser, error) {
	var user models.AssignedUser
	err := dao.DB.QueryRowContext(ctx, `
		SELECT id, CONCAT(first_name, ' ', last_name) as name
		FROM iam.users
		WHERE id = $1
	`, userID).Scan(&user.ID, &user.Name)

	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Helper function to fetch multiple users as AssignedUser array
func (dao *RFIDao) getUsersDetails(ctx context.Context, userIDs []int64) ([]models.AssignedUser, error) {
	if len(userIDs) == 0 {
		return []models.AssignedUser{}, nil
	}

	rows, err := dao.DB.QueryContext(ctx, `
		SELECT id, CONCAT(first_name, ' ', last_name) as name
		FROM iam.users
		WHERE id = ANY($1)
		ORDER BY first_name, last_name
	`, pq.Array(userIDs))

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.AssignedUser
	for rows.Next() {
		var user models.AssignedUser
		if err := rows.Scan(&user.ID, &user.Name); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// CreateRFI creates a new RFI
func (dao *RFIDao) CreateRFI(ctx context.Context, projectID, userID, orgID int64, req *models.CreateRFIRequest) (*models.RFIResponse, error) {
	dao.Logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"user_id":    userID,
		"org_id":     orgID,
	}).Info("Starting RFI creation")

	// Validate project belongs to organization
	var projectOrgID int64
	err := dao.DB.QueryRowContext(ctx, `
		SELECT org_id FROM project.projects
		WHERE id = $1 AND is_deleted = FALSE
	`, projectID).Scan(&projectOrgID)

	if err == sql.ErrNoRows {
		dao.Logger.WithField("project_id", projectID).Warn("Project not found")
		return nil, fmt.Errorf("project not found")
	}
	if err != nil {
		dao.Logger.WithError(err).WithField("project_id", projectID).Error("Failed to validate project")
		return nil, fmt.Errorf("failed to validate project: %w", err)
	}
	if projectOrgID != orgID {
		dao.Logger.WithFields(logrus.Fields{
			"project_org_id": projectOrgID,
			"user_org_id":    orgID,
		}).Warn("Project does not belong to user's organization")
		return nil, fmt.Errorf("project does not belong to your organization")
	}

	dao.Logger.Info("Project validation successful")

	// Determine status - default is DRAFT
	status := models.RFIStatusDraft
	if req.Status != "" {
		status = req.Status
	}

	// Generate RFI number only if status is OPEN
	// DRAFT RFIs don't get a number until they're moved to OPEN
	var rfiNumber *string
	if status == models.RFIStatusOpen {
		generatedNumber, err := dao.GenerateRFINumber(ctx, projectID)
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to generate RFI number")
			return nil, fmt.Errorf("failed to generate RFI number: %w", err)
		}
		rfiNumber = &generatedNumber
		dao.Logger.WithField("rfi_number", generatedNumber).Info("Generated RFI number for OPEN status")
	} else {
		dao.Logger.Info("DRAFT status - no RFI number generated")
	}

	// Parse due date
	var dueDate *time.Time
	if req.DueDate != "" {
		if parsedDate, err := time.Parse("2006-01-02", req.DueDate); err == nil {
			dueDate = &parsedDate
		}
	}

	// Handle assigned_to array
	assignedTo := req.AssignedTo
	if assignedTo == nil {
		assignedTo = []int64{}
	}

	// Handle nullable fields
	var ballInCourt, receivedFrom sql.NullInt64
	if req.BallInCourt != nil {
		ballInCourt = sql.NullInt64{Int64: *req.BallInCourt, Valid: true}
	}
	if req.ReceivedFrom != nil {
		receivedFrom = sql.NullInt64{Int64: *req.ReceivedFrom, Valid: true}
	}

	// Set defaults
	priority := req.Priority
	if priority == "" {
		priority = models.RFIPriorityMedium
	}

	query := `
		INSERT INTO project.rfis (
			project_id, org_id, location_id, rfi_number, subject,
			description, category, discipline, project_phase, priority,
			status, received_from, assigned_to, ball_in_court,
			distribution_list, due_date, cost_impact, schedule_impact,
			cost_impact_amount, schedule_impact_days, location_description,
			drawing_numbers, specification_sections, related_rfis,
			created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26
		) RETURNING id, created_at, updated_at`

	var rfiID int64
	var createdAt, updatedAt time.Time

	dao.Logger.WithFields(logrus.Fields{
		"project_id":  projectID,
		"org_id":      orgID,
		"location_id": req.LocationID,
		"rfi_number":  rfiNumber,
		"status":      status,
		"subject":     req.Subject,
		"category":    req.Category,
		"priority":    priority,
	}).Info("Executing INSERT query")

	err = dao.DB.QueryRowContext(ctx, query,
		projectID, orgID, req.LocationID, rfiNumber, req.Subject,
		req.Description, req.Category, req.Discipline, req.ProjectPhase, priority,
		status, receivedFrom, pq.Array(assignedTo), ballInCourt,
		pq.Array(req.DistributionList), dueDate, req.CostImpact, req.ScheduleImpact,
		req.CostImpactAmount, req.ScheduleImpactDays, req.LocationDescription,
		pq.Array(req.DrawingNumbers), pq.Array(req.SpecificationSections), pq.Array(req.RelatedRFIs),
		userID, userID,
	).Scan(&rfiID, &createdAt, &updatedAt)

	if err != nil {
		dao.Logger.WithError(err).WithFields(logrus.Fields{
			"project_id":  projectID,
			"location_id": req.LocationID,
			"sql_error":   err.Error(),
		}).Error("Failed to execute INSERT query for RFI")
		return nil, fmt.Errorf("failed to create RFI: %w", err)
	}

	dao.Logger.WithField("rfi_id", rfiID).Info("RFI created successfully, fetching complete RFI data")

	return dao.GetRFI(ctx, rfiID)
}

// GetRFI retrieves a single RFI by ID
func (dao *RFIDao) GetRFI(ctx context.Context, rfiID int64) (*models.RFIResponse, error) {
	query := `
		SELECT
			r.id, r.project_id, r.org_id, r.location_id, r.rfi_number,
			r.subject, r.description, r.category, r.discipline,
			r.project_phase, r.priority, r.status,
			r.received_from, r.assigned_to, r.ball_in_court,
			r.distribution_list, r.due_date, r.closed_date,
			r.cost_impact, r.schedule_impact, r.cost_impact_amount,
			r.schedule_impact_days, r.location_description,
			r.drawing_numbers, r.specification_sections, r.related_rfis,
			r.created_at, r.created_by, r.updated_at, r.updated_by,
			p.name as project_name,
			l.name as location_name
		FROM project.rfis r
		LEFT JOIN project.projects p ON r.project_id = p.id
		LEFT JOIN iam.locations l ON r.location_id = l.id
		WHERE r.id = $1 AND r.is_deleted = FALSE`

	var rfi models.RFIResponse
	var locationID sql.NullInt64
	var locationName sql.NullString
	var rfiNumber sql.NullString
	var discipline, projectPhase, locationDesc sql.NullString
	var costImpactAmount sql.NullFloat64
	var scheduleImpactDays sql.NullInt32
	var dueDate, closedDate *time.Time
	var receivedFromID, ballInCourtID sql.NullInt64
	var assignedToIDs pq.Int64Array
	var distributionList, drawingNumbers, specSections, relatedRFIs pq.StringArray
	var createdByID, updatedByID int64

	err := dao.DB.QueryRowContext(ctx, query, rfiID).Scan(
		&rfi.ID, &rfi.ProjectID, &rfi.OrgID, &locationID, &rfiNumber,
		&rfi.Subject, &rfi.Description, &rfi.Category, &discipline,
		&projectPhase, &rfi.Priority, &rfi.Status,
		&receivedFromID, &assignedToIDs, &ballInCourtID,
		&distributionList, &dueDate, &closedDate,
		&rfi.CostImpact, &rfi.ScheduleImpact, &costImpactAmount,
		&scheduleImpactDays, &locationDesc,
		&drawingNumbers, &specSections, &relatedRFIs,
		&rfi.CreatedAt, &createdByID, &rfi.UpdatedAt, &updatedByID,
		&rfi.ProjectName, &locationName,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("RFI not found")
	}
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to get RFI")
		return nil, fmt.Errorf("failed to get RFI: %w", err)
	}

	// Handle nullable fields
	if locationID.Valid {
		rfi.LocationID = locationID.Int64
	}
	if locationName.Valid {
		rfi.LocationName = locationName.String
	}
	if rfiNumber.Valid {
		rfi.RFINumber = &rfiNumber.String
	}
	if discipline.Valid {
		rfi.Discipline = &discipline.String
	}
	if projectPhase.Valid {
		rfi.ProjectPhase = &projectPhase.String
	}
	if locationDesc.Valid {
		rfi.LocationDescription = &locationDesc.String
	}
	if costImpactAmount.Valid {
		rfi.CostImpactAmount = &costImpactAmount.Float64
	}
	if scheduleImpactDays.Valid {
		days := int(scheduleImpactDays.Int32)
		rfi.ScheduleImpactDays = &days
	}

	rfi.DueDate = dueDate
	rfi.ClosedDate = closedDate
	rfi.DistributionList = []string(distributionList)
	rfi.DrawingNumbers = []string(drawingNumbers)
	rfi.SpecificationSections = []string(specSections)
	rfi.RelatedRFIs = []string(relatedRFIs)

	// Fetch user details for received_from
	if receivedFromID.Valid {
		if user, err := dao.getUserDetails(ctx, receivedFromID.Int64); err == nil {
			rfi.ReceivedFrom = user
		}
	}

	// Fetch user details for assigned_to
	if len(assignedToIDs) > 0 {
		if users, err := dao.getUsersDetails(ctx, []int64(assignedToIDs)); err == nil {
			rfi.AssignedTo = users
		}
	} else {
		rfi.AssignedTo = []models.AssignedUser{}
	}

	// Fetch user details for ball_in_court
	if ballInCourtID.Valid {
		if user, err := dao.getUserDetails(ctx, ballInCourtID.Int64); err == nil {
			rfi.BallInCourt = user
		}
	}

	// Fetch user details for created_by and updated_by
	if user, err := dao.getUserDetails(ctx, createdByID); err == nil {
		rfi.CreatedBy = *user
	}
	if user, err := dao.getUserDetails(ctx, updatedByID); err == nil {
		rfi.UpdatedBy = *user
	}

	// Fetch attachments
	attachments, err := dao.GetRFIAttachments(ctx, rfiID)
	if err != nil {
		dao.Logger.WithError(err).Warn("Failed to get RFI attachments")
		attachments = []models.RFIAttachment{}
	}
	rfi.Attachments = attachments

	// Fetch comments
	comments, err := dao.GetRFIComments(ctx, rfiID)
	if err != nil {
		dao.Logger.WithError(err).Warn("Failed to get RFI comments")
		comments = []models.RFIComment{}
	}
	rfi.Comments = comments

	return &rfi, nil
}

// GetRFIsByProject retrieves all RFIs for a specific project with optional filters
func (dao *RFIDao) GetRFIsByProject(ctx context.Context, projectID int64, filters map[string]string) ([]models.RFIResponse, error) {
	query := `
		SELECT
			r.id, r.project_id, r.org_id, r.location_id, r.rfi_number,
			r.subject, r.description, r.category, r.discipline,
			r.project_phase, r.priority, r.status,
			r.received_from, r.assigned_to, r.ball_in_court,
			r.distribution_list, r.due_date, r.closed_date,
			r.cost_impact, r.schedule_impact, r.cost_impact_amount,
			r.schedule_impact_days, r.location_description,
			r.drawing_numbers, r.specification_sections, r.related_rfis,
			r.created_at, r.created_by, r.updated_at, r.updated_by,
			p.name as project_name,
			l.name as location_name
		FROM project.rfis r
		LEFT JOIN project.projects p ON r.project_id = p.id
		LEFT JOIN iam.locations l ON r.location_id = l.id
		WHERE r.project_id = $1 AND r.is_deleted = FALSE`

	args := []interface{}{projectID}
	argIndex := 2

	// Add filters
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

	if assignedTo, ok := filters["assigned_to"]; ok && assignedTo != "" {
		query += fmt.Sprintf(" AND $%d = ANY(r.assigned_to)", argIndex)
		args = append(args, assignedTo)
		argIndex++
	}

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
		var locationID sql.NullInt64
		var locationName sql.NullString
		var rfiNumber sql.NullString
		var discipline, projectPhase, locationDesc sql.NullString
		var costImpactAmount sql.NullFloat64
		var scheduleImpactDays sql.NullInt32
		var dueDate, closedDate *time.Time
		var receivedFromID, ballInCourtID sql.NullInt64
		var assignedToIDs pq.Int64Array
		var distributionList, drawingNumbers, specSections, relatedRFIs pq.StringArray
		var createdByID, updatedByID int64

		err := rows.Scan(
			&rfi.ID, &rfi.ProjectID, &rfi.OrgID, &locationID, &rfiNumber,
			&rfi.Subject, &rfi.Description, &rfi.Category, &discipline,
			&projectPhase, &rfi.Priority, &rfi.Status,
			&receivedFromID, &assignedToIDs, &ballInCourtID,
			&distributionList, &dueDate, &closedDate,
			&rfi.CostImpact, &rfi.ScheduleImpact, &costImpactAmount,
			&scheduleImpactDays, &locationDesc,
			&drawingNumbers, &specSections, &relatedRFIs,
			&rfi.CreatedAt, &createdByID, &rfi.UpdatedAt, &updatedByID,
			&rfi.ProjectName, &locationName,
		)

		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan RFI row")
			return nil, fmt.Errorf("failed to scan RFI: %w", err)
		}

		// Handle nullable fields
		if locationID.Valid {
			rfi.LocationID = locationID.Int64
		}
		if locationName.Valid {
			rfi.LocationName = locationName.String
		}
		if rfiNumber.Valid {
			rfi.RFINumber = &rfiNumber.String
		}
		if discipline.Valid {
			rfi.Discipline = &discipline.String
		}
		if projectPhase.Valid {
			rfi.ProjectPhase = &projectPhase.String
		}
		if locationDesc.Valid {
			rfi.LocationDescription = &locationDesc.String
		}
		if costImpactAmount.Valid {
			rfi.CostImpactAmount = &costImpactAmount.Float64
		}
		if scheduleImpactDays.Valid {
			days := int(scheduleImpactDays.Int32)
			rfi.ScheduleImpactDays = &days
		}

		rfi.DueDate = dueDate
		rfi.ClosedDate = closedDate
		rfi.DistributionList = []string(distributionList)
		rfi.DrawingNumbers = []string(drawingNumbers)
		rfi.SpecificationSections = []string(specSections)
		rfi.RelatedRFIs = []string(relatedRFIs)

		// Fetch user details
		if receivedFromID.Valid {
			if user, err := dao.getUserDetails(ctx, receivedFromID.Int64); err == nil {
				rfi.ReceivedFrom = user
			}
		}

		if len(assignedToIDs) > 0 {
			if users, err := dao.getUsersDetails(ctx, []int64(assignedToIDs)); err == nil {
				rfi.AssignedTo = users
			}
		} else {
			rfi.AssignedTo = []models.AssignedUser{}
		}

		if ballInCourtID.Valid {
			if user, err := dao.getUserDetails(ctx, ballInCourtID.Int64); err == nil {
				rfi.BallInCourt = user
			}
		}

		if user, err := dao.getUserDetails(ctx, createdByID); err == nil {
			rfi.CreatedBy = *user
		}
		if user, err := dao.getUserDetails(ctx, updatedByID); err == nil {
			rfi.UpdatedBy = *user
		}

		// Fetch attachments and comments (lightweight for list view)
		attachments, _ := dao.GetRFIAttachments(ctx, rfi.ID)
		if attachments == nil {
			attachments = []models.RFIAttachment{}
		}
		rfi.Attachments = attachments

		comments, _ := dao.GetRFIComments(ctx, rfi.ID)
		if comments == nil {
			comments = []models.RFIComment{}
		}
		rfi.Comments = comments

		rfis = append(rfis, rfi)
	}

	if err = rows.Err(); err != nil {
		dao.Logger.WithError(err).Error("Error iterating RFI rows")
		return nil, fmt.Errorf("error iterating RFIs: %w", err)
	}

	return rfis, nil
}

// UpdateRFI updates an existing RFI
func (dao *RFIDao) UpdateRFI(ctx context.Context, rfiID, userID, orgID int64, req *models.UpdateRFIRequest) (*models.RFIResponse, error) {
	// First check if RFI exists and belongs to org
	rfi, err := dao.GetRFI(ctx, rfiID)
	if err != nil {
		return nil, err
	}
	if rfi.OrgID != orgID {
		return nil, fmt.Errorf("RFI does not belong to your organization")
	}

	var setClauses []string
	var args []interface{}
	argIndex := 1

	// Build dynamic update query
	if req.Subject != "" {
		setClauses = append(setClauses, fmt.Sprintf("subject = $%d", argIndex))
		args = append(args, req.Subject)
		argIndex++
	}

	if req.Description != "" {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, req.Description)
		argIndex++
	}

	if req.Category != "" {
		setClauses = append(setClauses, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, req.Category)
		argIndex++
	}

	if req.Priority != "" {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argIndex))
		args = append(args, req.Priority)
		argIndex++
	}

	if req.Status != "" {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, req.Status)
		argIndex++

		// Generate RFI number when transitioning from DRAFT to OPEN
		if req.Status == models.RFIStatusOpen && (rfi.RFINumber == nil || *rfi.RFINumber == "") {
			generatedNumber, err := dao.GenerateRFINumber(ctx, rfi.ProjectID)
			if err != nil {
				dao.Logger.WithError(err).Error("Failed to generate RFI number during status change")
				return nil, fmt.Errorf("failed to generate RFI number: %w", err)
			}
			setClauses = append(setClauses, fmt.Sprintf("rfi_number = $%d", argIndex))
			args = append(args, generatedNumber)
			argIndex++
			dao.Logger.WithField("rfi_number", generatedNumber).Info("Generated RFI number when changing status from DRAFT to OPEN")
		}

		// Set closed_date when status changes to CLOSE
		if req.Status == models.RFIStatusClose {
			setClauses = append(setClauses, fmt.Sprintf("closed_date = $%d", argIndex))
			args = append(args, time.Now())
			argIndex++
		}
	}

	if req.Discipline != nil {
		setClauses = append(setClauses, fmt.Sprintf("discipline = $%d", argIndex))
		args = append(args, req.Discipline)
		argIndex++
	}

	if req.ProjectPhase != nil {
		setClauses = append(setClauses, fmt.Sprintf("project_phase = $%d", argIndex))
		args = append(args, req.ProjectPhase)
		argIndex++
	}

	if req.ReceivedFrom != nil {
		setClauses = append(setClauses, fmt.Sprintf("received_from = $%d", argIndex))
		args = append(args, *req.ReceivedFrom)
		argIndex++
	}

	if req.AssignedTo != nil {
		setClauses = append(setClauses, fmt.Sprintf("assigned_to = $%d", argIndex))
		args = append(args, pq.Array(req.AssignedTo))
		argIndex++
	}

	if req.BallInCourt != nil {
		setClauses = append(setClauses, fmt.Sprintf("ball_in_court = $%d", argIndex))
		args = append(args, *req.BallInCourt)
		argIndex++
	}

	if req.DueDate != "" {
		if parsedDate, err := time.Parse("2006-01-02", req.DueDate); err == nil {
			setClauses = append(setClauses, fmt.Sprintf("due_date = $%d", argIndex))
			args = append(args, parsedDate)
			argIndex++
		}
	}

	if req.DistributionList != nil {
		setClauses = append(setClauses, fmt.Sprintf("distribution_list = $%d", argIndex))
		args = append(args, pq.Array(req.DistributionList))
		argIndex++
	}

	if req.LocationDescription != nil {
		setClauses = append(setClauses, fmt.Sprintf("location_description = $%d", argIndex))
		args = append(args, req.LocationDescription)
		argIndex++
	}

	if req.DrawingNumbers != nil {
		setClauses = append(setClauses, fmt.Sprintf("drawing_numbers = $%d", argIndex))
		args = append(args, pq.Array(req.DrawingNumbers))
		argIndex++
	}

	if req.SpecificationSections != nil {
		setClauses = append(setClauses, fmt.Sprintf("specification_sections = $%d", argIndex))
		args = append(args, pq.Array(req.SpecificationSections))
		argIndex++
	}

	if req.RelatedRFIs != nil {
		setClauses = append(setClauses, fmt.Sprintf("related_rfis = $%d", argIndex))
		args = append(args, pq.Array(req.RelatedRFIs))
		argIndex++
	}

	setClauses = append(setClauses, fmt.Sprintf("cost_impact = $%d", argIndex))
	args = append(args, req.CostImpact)
	argIndex++

	setClauses = append(setClauses, fmt.Sprintf("schedule_impact = $%d", argIndex))
	args = append(args, req.ScheduleImpact)
	argIndex++

	if req.CostImpactAmount != nil {
		setClauses = append(setClauses, fmt.Sprintf("cost_impact_amount = $%d", argIndex))
		args = append(args, *req.CostImpactAmount)
		argIndex++
	}

	if req.ScheduleImpactDays != nil {
		setClauses = append(setClauses, fmt.Sprintf("schedule_impact_days = $%d", argIndex))
		args = append(args, *req.ScheduleImpactDays)
		argIndex++
	}

	if len(setClauses) == 0 {
		return dao.GetRFI(ctx, rfiID)
	}

	// Add updated_by and updated_at
	setClauses = append(setClauses, fmt.Sprintf("updated_by = $%d", argIndex))
	args = append(args, userID)
	argIndex++

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add WHERE clause parameters
	args = append(args, rfiID)

	query := fmt.Sprintf(`
		UPDATE project.rfis
		SET %s
		WHERE id = $%d AND is_deleted = FALSE
	`, strings.Join(setClauses, ", "), argIndex)

	result, err := dao.DB.ExecContext(ctx, query, args...)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to update RFI")
		return nil, fmt.Errorf("failed to update RFI: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, fmt.Errorf("RFI not found or no changes made")
	}

	return dao.GetRFI(ctx, rfiID)
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

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("RFI not found")
	}

	return nil
}

// AddRFIComment adds a comment to an RFI with optional attachments
func (dao *RFIDao) AddRFIComment(ctx context.Context, rfiID, userID int64, req *models.CreateRFICommentRequest) (*models.RFIComment, error) {
	var comment models.RFIComment

	query := `
		INSERT INTO project.rfi_comments (
			rfi_id, comment, comment_type, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, rfi_id, comment, comment_type, created_at, created_by, updated_at, updated_by, is_deleted`

	err := dao.DB.QueryRowContext(ctx, query,
		rfiID, req.Comment, models.RFICommentTypeComment,
		userID, userID,
	).Scan(
		&comment.ID, &comment.RFIID, &comment.Comment, &comment.CommentType,
		&comment.CreatedAt, &comment.CreatedBy, &comment.UpdatedAt,
		&comment.UpdatedBy, &comment.IsDeleted,
	)

	if err != nil {
		dao.Logger.WithError(err).Error("Failed to add RFI comment")
		return nil, fmt.Errorf("failed to add RFI comment: %w", err)
	}

	// Link attachments if provided
	if len(req.AttachmentIDs) > 0 {
		_, err = dao.DB.ExecContext(ctx, `
			UPDATE project.rfi_comment_attachments
			SET comment_id = $1, updated_by = $2, updated_at = NOW()
			WHERE id = ANY($3)
			AND comment_id IS NULL
			AND created_by = $2
			AND is_deleted = FALSE
		`, comment.ID, userID, pq.Array(req.AttachmentIDs))

		if err != nil {
			dao.Logger.WithError(err).Warn("Failed to link attachments to RFI comment")
		}
	}

	// Fetch attachments for this comment
	comment.Attachments = dao.getRFICommentAttachments(ctx, comment.ID)

	// Fetch creator name
	var createdByName string
	err = dao.DB.QueryRowContext(ctx, `
		SELECT CONCAT(first_name, ' ', last_name)
		FROM iam.users
		WHERE id = $1
	`, userID).Scan(&createdByName)

	if err == nil {
		comment.CreatedByName = createdByName
	}

	return &comment, nil
}

// GetRFIComments retrieves all comments for an RFI with attachments
func (dao *RFIDao) GetRFIComments(ctx context.Context, rfiID int64) ([]models.RFIComment, error) {
	query := `
		SELECT
			c.id, c.rfi_id, c.comment, c.comment_type,
			c.previous_value, c.new_value,
			c.created_at, c.created_by,
			CONCAT(u.first_name, ' ', u.last_name) as created_by_name,
			c.updated_at, c.updated_by
		FROM project.rfi_comments c
		LEFT JOIN iam.users u ON c.created_by = u.id
		WHERE c.rfi_id = $1 AND c.is_deleted = FALSE
		ORDER BY c.created_at DESC`

	rows, err := dao.DB.QueryContext(ctx, query, rfiID)
	if err != nil {
		return nil, fmt.Errorf("failed to get RFI comments: %w", err)
	}
	defer rows.Close()

	var comments []models.RFIComment
	for rows.Next() {
		var comment models.RFIComment
		err := rows.Scan(
			&comment.ID, &comment.RFIID, &comment.Comment, &comment.CommentType,
			&comment.PreviousValue, &comment.NewValue,
			&comment.CreatedAt, &comment.CreatedBy, &comment.CreatedByName,
			&comment.UpdatedAt, &comment.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}

		// Fetch attachments for this comment
		comment.Attachments = dao.getRFICommentAttachments(ctx, comment.ID)

		comments = append(comments, comment)
	}

	return comments, nil
}

// AddRFIAttachment adds an attachment to an RFI
func (dao *RFIDao) AddRFIAttachment(ctx context.Context, attachment *models.RFIAttachment) (*models.RFIAttachment, error) {
	query := `
		INSERT INTO project.rfi_attachments (
			rfi_id, file_name, file_path, file_type, file_size,
			description, s3_bucket, s3_key, s3_url, attachment_type,
			uploaded_by, upload_date, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at, updated_at`

	err := dao.DB.QueryRowContext(ctx, query,
		attachment.RFIID, attachment.FileName, attachment.FilePath,
		attachment.FileType, attachment.FileSize, attachment.Description,
		attachment.S3Bucket, attachment.S3Key, attachment.S3URL,
		attachment.AttachmentType, attachment.UploadedBy,
		attachment.UploadDate, attachment.CreatedBy, attachment.CreatedBy,
	).Scan(&attachment.ID, &attachment.CreatedAt, &attachment.UpdatedAt)

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
			id, rfi_id, file_name, file_path, file_type, file_size,
			description, s3_bucket, s3_key, s3_url, attachment_type,
			uploaded_by, upload_date, created_at, created_by,
			updated_at, updated_by
		FROM project.rfi_attachments
		WHERE rfi_id = $1 AND is_deleted = FALSE
		ORDER BY created_at DESC`

	rows, err := dao.DB.QueryContext(ctx, query, rfiID)
	if err != nil {
		return nil, fmt.Errorf("failed to get RFI attachments: %w", err)
	}
	defer rows.Close()

	var attachments []models.RFIAttachment
	for rows.Next() {
		var att models.RFIAttachment
		err := rows.Scan(
			&att.ID, &att.RFIID, &att.FileName, &att.FilePath,
			&att.FileType, &att.FileSize, &att.Description,
			&att.S3Bucket, &att.S3Key, &att.S3URL, &att.AttachmentType,
			&att.UploadedBy, &att.UploadDate, &att.CreatedAt,
			&att.CreatedBy, &att.UpdatedAt, &att.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		attachments = append(attachments, att)
	}

	return attachments, nil
}

// GenerateRFINumber generates a unique RFI number for a project
func (dao *RFIDao) GenerateRFINumber(ctx context.Context, projectID int64) (string, error) {
	var maxNumber sql.NullInt64
	year := time.Now().Year()

	// Get the maximum RFI number for this project and year
	// Extract the numeric part from rfi_number format: RFI-YYYY-NNNN
	err := dao.DB.QueryRowContext(ctx, `
		SELECT MAX(CAST(SUBSTRING(rfi_number FROM 'RFI-[0-9]+-([0-9]+)') AS INTEGER))
		FROM project.rfis
		WHERE project_id = $1
		AND EXTRACT(YEAR FROM created_at) = $2
		AND rfi_number IS NOT NULL
		AND is_deleted = false
	`, projectID, year).Scan(&maxNumber)

	if err != nil {
		return "", fmt.Errorf("failed to generate RFI number: %w", err)
	}

	// Next number is max + 1, or 1 if no RFIs exist
	nextNumber := int64(1)
	if maxNumber.Valid {
		nextNumber = maxNumber.Int64 + 1
	}

	return fmt.Sprintf("RFI-%d-%04d", year, nextNumber), nil
}

// getRFICommentAttachments retrieves all attachments for a specific comment
func (dao *RFIDao) getRFICommentAttachments(ctx context.Context, commentID int64) []models.RFICommentAttachment {
	query := `
		SELECT id, comment_id, file_name, file_path, file_size, file_type,
		       attachment_type, uploaded_by, created_at, created_by,
		       updated_at, updated_by, is_deleted
		FROM project.rfi_comment_attachments
		WHERE comment_id = $1 AND is_deleted = FALSE
		ORDER BY created_at ASC`

	rows, err := dao.DB.QueryContext(ctx, query, commentID)
	if err != nil {
		dao.Logger.WithError(err).Warn("Failed to get RFI comment attachments")
		return []models.RFICommentAttachment{}
	}
	defer rows.Close()

	var attachments []models.RFICommentAttachment
	for rows.Next() {
		var att models.RFICommentAttachment
		var fileSize sql.NullInt64
		var fileType sql.NullString

		err := rows.Scan(
			&att.ID, &att.CommentID, &att.FileName, &att.FilePath,
			&fileSize, &fileType, &att.AttachmentType,
			&att.UploadedBy, &att.CreatedAt, &att.CreatedBy,
			&att.UpdatedAt, &att.UpdatedBy, &att.IsDeleted,
		)

		if err != nil {
			dao.Logger.WithError(err).Warn("Failed to scan RFI comment attachment")
			continue
		}

		if fileSize.Valid {
			att.FileSize = &fileSize.Int64
		}
		if fileType.Valid {
			att.FileType = &fileType.String
		}

		attachments = append(attachments, att)
	}

	if attachments == nil {
		return []models.RFICommentAttachment{}
	}

	return attachments
}
