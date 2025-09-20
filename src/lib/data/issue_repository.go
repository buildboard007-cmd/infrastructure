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

// IssueRepository defines the interface for issue data operations
type IssueRepository interface {
	// CreateIssue creates a new issue in the project (unified structure, orgID from JWT)
	CreateIssue(ctx context.Context, projectID, userID, orgID int64, issue *models.CreateIssueRequest) (*models.IssueResponse, error)

	// GetIssueByID retrieves a specific issue by ID
	GetIssueByID(ctx context.Context, issueID int64) (*models.IssueResponse, error)

	// GetIssuesByProject retrieves all issues for a specific project
	GetIssuesByProject(ctx context.Context, projectID int64, filters map[string]string) ([]models.IssueResponse, error)

	// UpdateIssue updates an existing issue (unified structure)
	UpdateIssue(ctx context.Context, issueID, userID, orgID int64, updateReq *models.UpdateIssueRequest) (*models.IssueResponse, error)

	// DeleteIssue soft deletes an issue
	DeleteIssue(ctx context.Context, issueID, userID int64) error

	// UpdateIssueStatus updates only the status of an issue
	UpdateIssueStatus(ctx context.Context, issueID, userID int64, status string) error
}

// IssueDao implements IssueRepository interface using PostgreSQL
type IssueDao struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

// generateIssueNumber generates a unique issue number for the project
func (dao *IssueDao) generateIssueNumber(ctx context.Context, projectID int64, category string) (string, error) {
	var projectCode string
	var count int
	
	// Get project code
	err := dao.DB.QueryRowContext(ctx, `
		SELECT COALESCE(project_number, 'PRJ-' || id) 
		FROM project.projects 
		WHERE id = $1
	`, projectID).Scan(&projectCode)
	
	if err != nil {
		return "", fmt.Errorf("failed to get project code: %w", err)
	}
	
	// Get the count of issues for this project and category
	categoryPrefix := strings.ToUpper(string(category[0:2]))
	err = dao.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) + 1
		FROM project.issues 
		WHERE project_id = $1 AND category = $2
	`, projectID, category).Scan(&count)
	
	if err != nil {
		return "", fmt.Errorf("failed to get issue count: %w", err)
	}
	
	// Format: PROJECT-CA-0001
	return fmt.Sprintf("%s-%s-%04d", projectCode, categoryPrefix, count), nil
}

// CreateIssue creates a new issue in the project with unified structure
func (dao *IssueDao) CreateIssue(ctx context.Context, projectID, userID, orgID int64, req *models.CreateIssueRequest) (*models.IssueResponse, error) {
	// Validate project belongs to organization
	var projectOrgID int64
	err := dao.DB.QueryRowContext(ctx, `
		SELECT org_id FROM project.projects
		WHERE id = $1 AND is_deleted = FALSE
	`, projectID).Scan(&projectOrgID)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to validate project: %w", err)
	}
	if projectOrgID != orgID {
		return nil, fmt.Errorf("project does not belong to your organization")
	}
	// Handle template ID (not in current request structure)
	var templateID sql.NullInt64
	// Start transaction
	tx, err := dao.DB.BeginTx(ctx, nil)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to start transaction for issue creation")
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Generate issue number using the flatter structure
	issueNumber, err := dao.generateIssueNumber(ctx, projectID, req.Category)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to generate issue number")
		return nil, err
	}

	// Set defaults from flatter structure
	priority := req.Priority
	if priority == "" {
		priority = models.IssuePriorityMedium
	}
	severity := req.Severity
	if severity == "" {
		severity = models.IssueSeverityMinor
	}

	// Get location_id from project if not provided in request
	locationID := req.LocationID
	if locationID == 0 {
		err = dao.DB.QueryRowContext(ctx, `
			SELECT location_id FROM project.projects
			WHERE id = $1
		`, projectID).Scan(&locationID)
		if err != nil {
			dao.Logger.WithError(err).Warn("Failed to get location ID from project")
		}
	}
	
	// Parse due date from flatter structure
	var dueDate *time.Time
	if req.DueDate != "" {
		parsedDate, err := time.Parse("2006-01-02", req.DueDate)
		if err != nil {
			return nil, fmt.Errorf("invalid due date format: %w", err)
		}
		dueDate = &parsedDate
	}

	// Handle assigned to from flatter structure
	var assignedToID sql.NullInt64
	if req.AssignedTo != 0 {
		assignedToID = sql.NullInt64{Int64: req.AssignedTo, Valid: true}
	}
	
	// Create the issue
	var issueID int64
	var createdAt, updatedAt time.Time
	
	// Map issue type from issue_category in flatter structure
	issueType := "general"
	if req.IssueCategory != "" {
		switch req.IssueCategory {
		case "quality", "safety", "deficiency", "punch_item", "code_violation":
			issueType = req.IssueCategory
		}
	}

	// Handle coordinates from unified structure
	var locationX, locationY sql.NullFloat64
	var latitude, longitude sql.NullFloat64
	if req.Location.Coordinates != nil {
		locationX = sql.NullFloat64{Float64: req.Location.Coordinates.X, Valid: true}
		locationY = sql.NullFloat64{Float64: req.Location.Coordinates.Y, Valid: true}
	}

	// Handle GPS coordinates from flatter structure
	if req.Location.GPSCoordinates != nil {
		latitude = sql.NullFloat64{Float64: req.Location.GPSCoordinates.Latitude, Valid: true}
		longitude = sql.NullFloat64{Float64: req.Location.GPSCoordinates.Longitude, Valid: true}
	}
	
	err = tx.QueryRowContext(ctx, `
		INSERT INTO project.issues (
			project_id, issue_number, template_id,
			title, description, 
			issue_type, category, detail_category,
			priority, severity,
			root_cause,
			location_description, location_building, location_level, location_room,
			location_x, location_y,
			room_area, floor_level,
			discipline, trade_type,
			reported_by, assigned_to, assigned_company_id,
			drawing_reference, specification_reference,
			due_date, distribution_list,
			status,
			latitude, longitude,
			created_by, updated_by
		) VALUES (
			$1, $2, $3,
			$4, $5,
			$6, $7, $8,
			$9, $10,
			$11,
			$12, $13, $14, $15,
			$16, $17,
			$18, $19,
			$20, $21,
			$22, $23, $24,
			$25, $26,
			$27, $28,
			$29,
			$30, $31,
			$32, $33
		)
		RETURNING id, created_at, updated_at
	`,
		projectID, issueNumber, templateID,
		req.Title, req.Description,
		issueType, sql.NullString{String: req.IssueCategory, Valid: req.IssueCategory != ""}, sql.NullString{String: req.DetailCategory, Valid: req.DetailCategory != ""},
		priority, severity,
		sql.NullString{String: req.RootCause, Valid: req.RootCause != ""},
		sql.NullString{String: req.Location.Description, Valid: req.Location.Description != ""},
		sql.NullString{String: req.Location.Building, Valid: req.Location.Building != ""},
		sql.NullString{String: req.Location.Level, Valid: req.Location.Level != ""},
		sql.NullString{String: req.Location.Room, Valid: req.Location.Room != ""},
		locationX, locationY,
		sql.NullString{String: req.Location.Room, Valid: req.Location.Room != ""}, // room_area = room for now
		sql.NullString{String: req.Location.Level, Valid: req.Location.Level != ""}, // floor_level = level for now
		sql.NullString{String: req.Discipline, Valid: req.Discipline != ""}, // discipline from flatter structure
		sql.NullString{String: req.Trade, Valid: req.Trade != ""}, // trade from flatter structure
		userID, assignedToID, sql.NullInt64{}, // assigned_company_id not in request for now
		sql.NullString{}, sql.NullString{}, // drawing_reference, specification_reference not in request
		dueDate, pq.Array(req.DistributionList),
		models.IssueStatusOpen,
		latitude, longitude,
		userID, userID,
	).Scan(&issueID, &createdAt, &updatedAt)
	
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"user_id":    userID,
			"error":      err.Error(),
		}).Error("Failed to create issue")
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}
	
	// Commit transaction
	if err = tx.Commit(); err != nil {
		dao.Logger.WithError(err).Error("Failed to commit issue creation transaction")
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	dao.Logger.WithFields(logrus.Fields{
		"issue_id":     issueID,
		"issue_number": issueNumber,
		"project_id":   projectID,
		"user_id":      userID,
	}).Info("Successfully created issue")
	
	// Get the created issue with full details
	return dao.GetIssueByID(ctx, issueID)
}

// GetIssueByID retrieves a specific issue by ID
func (dao *IssueDao) GetIssueByID(ctx context.Context, issueID int64) (*models.IssueResponse, error) {
	var response models.IssueResponse
	var distributionList pq.StringArray
	
	// Database scan variables (using sql.Null* types for nullable columns)
	var templateID sql.NullInt64
	var category, detailCategory, rootCause sql.NullString
	var locationDescription, locationBuilding, locationLevel, locationRoom sql.NullString
	var locationX, locationY sql.NullFloat64
	var roomArea, floorLevel, discipline, tradeType sql.NullString
	var assignedTo, assignedCompanyID sql.NullInt64
	var drawingRef, specRef sql.NullString
	var dueDate, closedDate *time.Time
	var costToFix, latitude, longitude sql.NullFloat64
	var projectName, reportedByName, assignedToName, assignedCompanyName sql.NullString
	
	query := `
		SELECT 
			i.id, i.project_id, i.issue_number, i.template_id,
			i.title, i.description,
			i.category, i.detail_category, i.issue_type,
			i.priority, i.severity,
			i.root_cause,
			i.location_description, i.location_building, i.location_level, i.location_room,
			i.location_x, i.location_y,
			i.room_area, i.floor_level,
			i.discipline, i.trade_type,
			i.reported_by, i.assigned_to, i.assigned_company_id,
			i.drawing_reference, i.specification_reference,
			i.due_date, i.closed_date,
			i.distribution_list,
			i.status,
			i.cost_to_fix,
			i.latitude, i.longitude,
			i.created_at, i.created_by, i.updated_at, i.updated_by,
			p.name as project_name,
			CONCAT(u1.first_name, ' ', u1.last_name) as reported_by_name,
			CONCAT(u2.first_name, ' ', u2.last_name) as assigned_to_name,
			o.name as assigned_company_name,
			EXTRACT(DAY FROM (CURRENT_TIMESTAMP - i.created_at)) as days_open,
			CASE WHEN i.due_date < CURRENT_TIMESTAMP AND i.status != 'closed' THEN true ELSE false END as is_overdue
		FROM project.issues i
		LEFT JOIN project.projects p ON i.project_id = p.id
		LEFT JOIN iam.users u1 ON i.reported_by = u1.id
		LEFT JOIN iam.users u2 ON i.assigned_to = u2.id
		LEFT JOIN iam.organizations o ON i.assigned_company_id = o.id
		WHERE i.id = $1 AND i.is_deleted = FALSE
	`
	
	err := dao.DB.QueryRowContext(ctx, query, issueID).Scan(
		&response.ID, &response.ProjectID, &response.IssueNumber, &templateID,
		&response.Title, &response.Description,
		&category, &detailCategory, &response.IssueType,
		&response.Priority, &response.Severity,
		&rootCause,
		&locationDescription, &locationBuilding, &locationLevel, &locationRoom,
		&locationX, &locationY,
		&roomArea, &floorLevel,
		&discipline, &tradeType,
		&response.ReportedBy, &assignedTo, &assignedCompanyID,
		&drawingRef, &specRef,
		&dueDate, &closedDate,
		&distributionList,
		&response.Status,
		&costToFix,
		&latitude, &longitude,
		&response.CreatedAt, &response.CreatedBy, &response.UpdatedAt, &response.UpdatedBy,
		&projectName,
		&reportedByName,
		&assignedToName,
		&assignedCompanyName,
		&response.DaysOpen,
		&response.IsOverdue,
	)
	
	if err == sql.ErrNoRows {
		dao.Logger.WithField("issue_id", issueID).Warn("Issue not found")
		return nil, fmt.Errorf("issue not found")
	}
	
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"issue_id": issueID,
			"error":    err.Error(),
		}).Error("Failed to get issue")
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}
	
	response.DistributionList = []string(distributionList)
	
	// Convert nullable database types to clean response types
	if projectName.Valid {
		response.ProjectName = projectName.String
	}
	if reportedByName.Valid {
		response.ReportedByName = reportedByName.String
	}
	if assignedToName.Valid {
		response.AssignedToName = assignedToName.String
	}
	if assignedCompanyName.Valid {
		response.AssignedCompanyName = assignedCompanyName.String
	}
	
	// Handle nullable fields - only set if valid
	if templateID.Valid {
		response.TemplateID = &templateID.Int64
	}
	if category.Valid && category.String != "" {
		response.Category = category.String
	}
	if detailCategory.Valid && detailCategory.String != "" {
		response.DetailCategory = detailCategory.String
	}
	if rootCause.Valid && rootCause.String != "" {
		response.RootCause = rootCause.String
	}
	if locationDescription.Valid && locationDescription.String != "" {
		response.LocationDescription = locationDescription.String
	}
	if locationBuilding.Valid && locationBuilding.String != "" {
		response.LocationBuilding = locationBuilding.String
	}
	if locationLevel.Valid && locationLevel.String != "" {
		response.LocationLevel = locationLevel.String
	}
	if locationRoom.Valid && locationRoom.String != "" {
		response.LocationRoom = locationRoom.String
	}
	if locationX.Valid {
		response.LocationX = &locationX.Float64
	}
	if locationY.Valid {
		response.LocationY = &locationY.Float64
	}
	if roomArea.Valid && roomArea.String != "" {
		response.RoomArea = roomArea.String
	}
	if floorLevel.Valid && floorLevel.String != "" {
		response.FloorLevel = floorLevel.String
	}
	if discipline.Valid && discipline.String != "" {
		response.Discipline = discipline.String
	}
	if tradeType.Valid && tradeType.String != "" {
		response.TradeType = tradeType.String
	}
	if assignedTo.Valid {
		response.AssignedTo = &assignedTo.Int64
	}
	if assignedCompanyID.Valid {
		response.AssignedCompanyID = &assignedCompanyID.Int64
	}
	if drawingRef.Valid && drawingRef.String != "" {
		response.DrawingReference = drawingRef.String
	}
	if specRef.Valid && specRef.String != "" {
		response.SpecificationRef = specRef.String
	}
	if dueDate != nil {
		response.DueDate = dueDate
	}
	if closedDate != nil {
		response.ClosedDate = closedDate
	}
	if costToFix.Valid {
		response.CostToFix = &costToFix.Float64
	}
	if latitude.Valid {
		response.Latitude = &latitude.Float64
	}
	if longitude.Valid {
		response.Longitude = &longitude.Float64
	}
	
	return &response, nil
}

// GetIssuesByProject retrieves all issues for a specific project with optional filters
func (dao *IssueDao) GetIssuesByProject(ctx context.Context, projectID int64, filters map[string]string) ([]models.IssueResponse, error) {
	// Build query with filters
	query := `
		SELECT 
			i.id, i.project_id, i.issue_number, i.template_id,
			i.title, i.description,
			i.category, i.detail_category, i.issue_type,
			i.priority, i.severity,
			i.root_cause,
			i.location_description, i.location_building, i.location_level, i.location_room,
			i.location_x, i.location_y,
			i.room_area, i.floor_level,
			i.discipline, i.trade_type,
			i.reported_by, i.assigned_to, i.assigned_company_id,
			i.drawing_reference, i.specification_reference,
			i.due_date, i.closed_date,
			i.distribution_list,
			i.status,
			i.cost_to_fix,
			i.latitude, i.longitude,
			i.created_at, i.created_by, i.updated_at, i.updated_by,
			p.name as project_name,
			CONCAT(u1.first_name, ' ', u1.last_name) as reported_by_name,
			CONCAT(u2.first_name, ' ', u2.last_name) as assigned_to_name,
			o.name as assigned_company_name,
			EXTRACT(DAY FROM (CURRENT_TIMESTAMP - i.created_at)) as days_open,
			CASE WHEN i.due_date < CURRENT_TIMESTAMP AND i.status != 'closed' THEN true ELSE false END as is_overdue
		FROM project.issues i
		LEFT JOIN project.projects p ON i.project_id = p.id
		LEFT JOIN iam.users u1 ON i.reported_by = u1.id
		LEFT JOIN iam.users u2 ON i.assigned_to = u2.id
		LEFT JOIN iam.organizations o ON i.assigned_company_id = o.id
		WHERE i.project_id = $1 AND i.is_deleted = FALSE
	`
	
	// Add filters
	args := []interface{}{projectID}
	argIndex := 2
	
	if status, ok := filters["status"]; ok && status != "" {
		query += fmt.Sprintf(" AND i.status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}
	
	if priority, ok := filters["priority"]; ok && priority != "" {
		query += fmt.Sprintf(" AND i.priority = $%d", argIndex)
		args = append(args, priority)
		argIndex++
	}
	
	if category, ok := filters["category"]; ok && category != "" {
		query += fmt.Sprintf(" AND i.category = $%d", argIndex)
		args = append(args, category)
		argIndex++
	}
	
	if assignedTo, ok := filters["assigned_to"]; ok && assignedTo != "" {
		query += fmt.Sprintf(" AND i.assigned_to = $%d", argIndex)
		args = append(args, assignedTo)
		argIndex++
	}
	
	// Add ordering
	query += " ORDER BY i.created_at DESC"
	
	rows, err := dao.DB.QueryContext(ctx, query, args...)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"error":      err.Error(),
		}).Error("Failed to query issues")
		return nil, fmt.Errorf("failed to query issues: %w", err)
	}
	defer rows.Close()
	
	var issues []models.IssueResponse
	for rows.Next() {
		var issue models.IssueResponse
		var distributionList pq.StringArray
		
		// Database scan variables (using sql.Null* types for nullable columns)
		var templateID sql.NullInt64
		var category, detailCategory, rootCause sql.NullString
		var locationDescription, locationBuilding, locationLevel, locationRoom sql.NullString
		var locationX, locationY sql.NullFloat64
		var roomArea, floorLevel, discipline, tradeType sql.NullString
		var assignedTo, assignedCompanyID sql.NullInt64
		var drawingRef, specRef sql.NullString
		var dueDate, closedDate *time.Time
		var costToFix, latitude, longitude sql.NullFloat64
		var projectName, reportedByName, assignedToName, assignedCompanyName sql.NullString
		
		err := rows.Scan(
			&issue.ID, &issue.ProjectID, &issue.IssueNumber, &templateID,
			&issue.Title, &issue.Description,
			&category, &detailCategory, &issue.IssueType,
			&issue.Priority, &issue.Severity,
			&rootCause,
			&locationDescription, &locationBuilding, &locationLevel, &locationRoom,
			&locationX, &locationY,
			&roomArea, &floorLevel,
			&discipline, &tradeType,
			&issue.ReportedBy, &assignedTo, &assignedCompanyID,
			&drawingRef, &specRef,
			&dueDate, &closedDate,
			&distributionList,
			&issue.Status,
			&costToFix,
			&latitude, &longitude,
			&issue.CreatedAt, &issue.CreatedBy, &issue.UpdatedAt, &issue.UpdatedBy,
			&projectName,
			&reportedByName,
			&assignedToName,
			&assignedCompanyName,
			&issue.DaysOpen,
			&issue.IsOverdue,
		)
		
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan issue row")
			return nil, fmt.Errorf("failed to scan issue: %w", err)
		}
		
		issue.DistributionList = []string(distributionList)
		
		// Convert nullable database types to clean response types
		if projectName.Valid {
			issue.ProjectName = projectName.String
		}
		if reportedByName.Valid {
			issue.ReportedByName = reportedByName.String
		}
		if assignedToName.Valid {
			issue.AssignedToName = assignedToName.String
		}
		if assignedCompanyName.Valid {
			issue.AssignedCompanyName = assignedCompanyName.String
		}
		
		// Handle nullable fields - only set if valid
		if templateID.Valid {
			issue.TemplateID = &templateID.Int64
		}
		if category.Valid && category.String != "" {
			issue.Category = category.String
		}
		if detailCategory.Valid && detailCategory.String != "" {
			issue.DetailCategory = detailCategory.String
		}
		if rootCause.Valid && rootCause.String != "" {
			issue.RootCause = rootCause.String
		}
		if locationDescription.Valid && locationDescription.String != "" {
			issue.LocationDescription = locationDescription.String
		}
		if locationBuilding.Valid && locationBuilding.String != "" {
			issue.LocationBuilding = locationBuilding.String
		}
		if locationLevel.Valid && locationLevel.String != "" {
			issue.LocationLevel = locationLevel.String
		}
		if locationRoom.Valid && locationRoom.String != "" {
			issue.LocationRoom = locationRoom.String
		}
		if locationX.Valid {
			issue.LocationX = &locationX.Float64
		}
		if locationY.Valid {
			issue.LocationY = &locationY.Float64
		}
		if roomArea.Valid && roomArea.String != "" {
			issue.RoomArea = roomArea.String
		}
		if floorLevel.Valid && floorLevel.String != "" {
			issue.FloorLevel = floorLevel.String
		}
		if discipline.Valid && discipline.String != "" {
			issue.Discipline = discipline.String
		}
		if tradeType.Valid && tradeType.String != "" {
			issue.TradeType = tradeType.String
		}
		if assignedTo.Valid {
			issue.AssignedTo = &assignedTo.Int64
		}
		if assignedCompanyID.Valid {
			issue.AssignedCompanyID = &assignedCompanyID.Int64
		}
		if drawingRef.Valid && drawingRef.String != "" {
			issue.DrawingReference = drawingRef.String
		}
		if specRef.Valid && specRef.String != "" {
			issue.SpecificationRef = specRef.String
		}
		if dueDate != nil {
			issue.DueDate = dueDate
		}
		if closedDate != nil {
			issue.ClosedDate = closedDate
		}
		if costToFix.Valid {
			issue.CostToFix = &costToFix.Float64
		}
		if latitude.Valid {
			issue.Latitude = &latitude.Float64
		}
		if longitude.Valid {
			issue.Longitude = &longitude.Float64
		}
		
		issues = append(issues, issue)
	}
	
	if err = rows.Err(); err != nil {
		dao.Logger.WithError(err).Error("Error iterating issue rows")
		return nil, fmt.Errorf("error iterating issues: %w", err)
	}
	
	dao.Logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"count":      len(issues),
	}).Debug("Successfully retrieved issues for project")
	
	return issues, nil
}

// UpdateIssue updates an existing issue
func (dao *IssueDao) UpdateIssue(ctx context.Context, issueID, userID, orgID int64, req *models.UpdateIssueRequest) (*models.IssueResponse, error) {
	// First validate that issue exists and belongs to user's organization
	var projectID, projectOrgID int64
	err := dao.DB.QueryRowContext(ctx, `
		SELECT p.id, p.org_id
		FROM project.issues i
		JOIN project.projects p ON i.project_id = p.id
		WHERE i.id = $1 AND i.is_deleted = FALSE AND p.is_deleted = FALSE
	`, issueID).Scan(&projectID, &projectOrgID)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("issue not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to validate issue: %w", err)
	}
	if projectOrgID != orgID {
		return nil, fmt.Errorf("issue does not belong to your organization")
	}

	// Build dynamic update query using flatter structure
	setParts := []string{"updated_by = $1", "updated_at = CURRENT_TIMESTAMP"}
	args := []interface{}{userID}
	argIndex := 2

	// Handle basic info from flatter structure
	if req.Title != "" {
		setParts = append(setParts, fmt.Sprintf("title = $%d", argIndex))
		args = append(args, req.Title)
		argIndex++
	}

	if req.Description != "" {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, req.Description)
		argIndex++
	}

	// Handle classification from flatter structure
	if req.Category != "" {
		setParts = append(setParts, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, req.Category)
		argIndex++
	}

	if req.DetailCategory != "" {
		setParts = append(setParts, fmt.Sprintf("detail_category = $%d", argIndex))
		args = append(args, req.DetailCategory)
		argIndex++
	}

	if req.Priority != "" {
		setParts = append(setParts, fmt.Sprintf("priority = $%d", argIndex))
		args = append(args, req.Priority)
		argIndex++
	}

	if req.Severity != "" {
		setParts = append(setParts, fmt.Sprintf("severity = $%d", argIndex))
		args = append(args, req.Severity)
		argIndex++
	}

	if req.RootCause != "" {
		setParts = append(setParts, fmt.Sprintf("root_cause = $%d", argIndex))
		args = append(args, req.RootCause)
		argIndex++
	}

	// Handle location (nested object)
	if req.Location.Description != "" {
		setParts = append(setParts, fmt.Sprintf("location_description = $%d", argIndex))
		args = append(args, req.Location.Description)
		argIndex++
		
		if req.Location.Building != "" {
			setParts = append(setParts, fmt.Sprintf("location_building = $%d", argIndex))
			args = append(args, req.Location.Building)
			argIndex++
		}
		
		if req.Location.Level != "" {
			setParts = append(setParts, fmt.Sprintf("location_level = $%d", argIndex))
			args = append(args, req.Location.Level)
			argIndex++
		}
		
		if req.Location.Room != "" {
			setParts = append(setParts, fmt.Sprintf("location_room = $%d", argIndex))
			args = append(args, req.Location.Room)
			argIndex++
		}
		
		if req.Location.Coordinates != nil {
			setParts = append(setParts, fmt.Sprintf("location_x = $%d", argIndex))
			args = append(args, req.Location.Coordinates.X)
			argIndex++
			
			setParts = append(setParts, fmt.Sprintf("location_y = $%d", argIndex))
			args = append(args, req.Location.Coordinates.Y)
			argIndex++
		}

		// Handle GPS coordinates
		if req.Location.GPSCoordinates != nil {
			setParts = append(setParts, fmt.Sprintf("latitude = $%d", argIndex))
			args = append(args, req.Location.GPSCoordinates.Latitude)
			argIndex++

			setParts = append(setParts, fmt.Sprintf("longitude = $%d", argIndex))
			args = append(args, req.Location.GPSCoordinates.Longitude)
			argIndex++
		}
	}

	if req.Discipline != "" {
		setParts = append(setParts, fmt.Sprintf("discipline = $%d", argIndex))
		args = append(args, req.Discipline)
		argIndex++
	}

	if req.Trade != "" {
		setParts = append(setParts, fmt.Sprintf("trade_type = $%d", argIndex))
		args = append(args, req.Trade)
		argIndex++
	}

	if req.AssignedTo != 0 {
		// Handle assigned to as int64 from flatter structure
		assignedToID := sql.NullInt64{Int64: req.AssignedTo, Valid: true}
		setParts = append(setParts, fmt.Sprintf("assigned_to = $%d", argIndex))
		args = append(args, assignedToID)
		argIndex++
	}

	if req.DueDate != "" {
		parsedDate, err := time.Parse("2006-01-02", req.DueDate)
		if err != nil {
			return nil, fmt.Errorf("invalid due date format: %w", err)
		}
		setParts = append(setParts, fmt.Sprintf("due_date = $%d", argIndex))
		args = append(args, parsedDate)
		argIndex++
	}
	
	if req.Status != "" {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, req.Status)
		argIndex++
		
		// If closing the issue, set closed_date
		if req.Status == models.IssueStatusClosed {
			setParts = append(setParts, "closed_date = CURRENT_TIMESTAMP")
		}
	}
	
	if req.DistributionList != nil {
		setParts = append(setParts, fmt.Sprintf("distribution_list = $%d", argIndex))
		args = append(args, pq.Array(req.DistributionList))
		argIndex++
	}
	
	// Add WHERE condition
	args = append(args, issueID)
	
	query := fmt.Sprintf(`
		UPDATE project.issues 
		SET %s
		WHERE id = $%d AND is_deleted = FALSE
		RETURNING updated_at
	`, strings.Join(setParts, ", "), argIndex)
	
	var updatedAt time.Time
	err = dao.DB.QueryRowContext(ctx, query, args...).Scan(&updatedAt)
	
	if err == sql.ErrNoRows {
		dao.Logger.WithField("issue_id", issueID).Warn("Issue not found for update")
		return nil, fmt.Errorf("issue not found")
	}
	
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"issue_id": issueID,
			"error":    err.Error(),
		}).Error("Failed to update issue")
		return nil, fmt.Errorf("failed to update issue: %w", err)
	}
	
	dao.Logger.WithFields(logrus.Fields{
		"issue_id": issueID,
		"user_id":  userID,
	}).Info("Successfully updated issue")
	
	// Return updated issue
	return dao.GetIssueByID(ctx, issueID)
}

// DeleteIssue soft deletes an issue
func (dao *IssueDao) DeleteIssue(ctx context.Context, issueID, userID int64) error {
	result, err := dao.DB.ExecContext(ctx, `
		UPDATE project.issues 
		SET is_deleted = TRUE, updated_by = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND is_deleted = FALSE
	`, userID, issueID)
	
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"issue_id": issueID,
			"error":    err.Error(),
		}).Error("Failed to delete issue")
		return fmt.Errorf("failed to delete issue: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		dao.Logger.WithField("issue_id", issueID).Warn("Issue not found for deletion")
		return fmt.Errorf("issue not found")
	}
	
	dao.Logger.WithFields(logrus.Fields{
		"issue_id": issueID,
		"user_id":  userID,
	}).Info("Successfully soft deleted issue")
	
	return nil
}

// UpdateIssueStatus updates only the status of an issue
func (dao *IssueDao) UpdateIssueStatus(ctx context.Context, issueID, userID int64, status string) error {
	query := `
		UPDATE project.issues 
		SET status = $1, updated_by = $2, updated_at = CURRENT_TIMESTAMP
	`
	args := []interface{}{status, userID}
	
	// If closing the issue, set closed_date
	if status == models.IssueStatusClosed {
		query += ", closed_date = CURRENT_TIMESTAMP"
	}
	
	query += " WHERE id = $3 AND is_deleted = FALSE"
	args = append(args, issueID)
	
	result, err := dao.DB.ExecContext(ctx, query, args...)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"issue_id": issueID,
			"status":   status,
			"error":    err.Error(),
		}).Error("Failed to update issue status")
		return fmt.Errorf("failed to update issue status: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		dao.Logger.WithField("issue_id", issueID).Warn("Issue not found for status update")
		return fmt.Errorf("issue not found")
	}
	
	dao.Logger.WithFields(logrus.Fields{
		"issue_id": issueID,
		"status":   status,
		"user_id":  userID,
	}).Info("Successfully updated issue status")
	
	return nil
}