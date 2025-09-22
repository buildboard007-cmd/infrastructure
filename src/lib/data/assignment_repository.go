package data

import (
	"context"
	"database/sql"
	"fmt"
	"infrastructure/lib/models"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// AssignmentRepository defines the interface for unified assignment operations
type AssignmentRepository interface {
	// Basic CRUD operations
	CreateAssignment(ctx context.Context, req *models.CreateAssignmentRequest, userID int64) (*models.AssignmentResponse, error)
	GetAssignment(ctx context.Context, assignmentID int64, orgID int64) (*models.AssignmentResponse, error)
	UpdateAssignment(ctx context.Context, assignmentID int64, req *models.UpdateAssignmentRequest, userID int64) (*models.AssignmentResponse, error)
	DeleteAssignment(ctx context.Context, assignmentID int64, userID int64) error

	// Bulk operations
	CreateBulkAssignments(ctx context.Context, req *models.BulkAssignmentRequest, userID int64) ([]models.AssignmentResponse, error)
	TransferAssignments(ctx context.Context, req *models.AssignmentTransferRequest, userID int64) error

	// Query operations
	GetAssignments(ctx context.Context, filters *models.AssignmentFilters, orgID int64) (*models.AssignmentListResponse, error)
	GetUserAssignments(ctx context.Context, userID int64, orgID int64) (*models.UserAssignmentSummary, error)
	GetContextAssignments(ctx context.Context, contextType string, contextID int64, orgID int64) (*models.ContextAssignmentSummary, error)

	// Permission checking
	CheckPermission(ctx context.Context, req *models.PermissionCheckRequest, orgID int64) (*models.PermissionCheckResponse, error)
	GetUserContexts(ctx context.Context, userID int64, contextType string, orgID int64) ([]int64, error)

	// Validation and utilities
	ValidateAssignmentContext(ctx context.Context, contextType string, contextID int64, orgID int64) error
	GetActiveAssignments(ctx context.Context, userID int64, orgID int64) ([]models.AssignmentResponse, error)
}

// AssignmentDao implements AssignmentRepository interface using PostgreSQL
type AssignmentDao struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

// NewAssignmentRepository creates a new AssignmentRepository instance
func NewAssignmentRepository(db *sql.DB) AssignmentRepository {
	return &AssignmentDao{
		DB:     db,
		Logger: logrus.New(),
	}
}

// CreateAssignment creates a new user assignment
func (dao *AssignmentDao) CreateAssignment(ctx context.Context, req *models.CreateAssignmentRequest, userID int64) (*models.AssignmentResponse, error) {
	// Validate the context exists and belongs to the organization
	err := dao.ValidateAssignmentContext(ctx, req.ContextType, req.ContextID, 0) // Will be validated in the method
	if err != nil {
		return nil, fmt.Errorf("invalid assignment context: %w", err)
	}

	// Parse optional dates
	var startDate, endDate sql.NullTime
	if req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			startDate = sql.NullTime{Time: t, Valid: true}
		} else {
			return nil, fmt.Errorf("invalid start_date format, expected YYYY-MM-DD")
		}
	}
	if req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			endDate = sql.NullTime{Time: t, Valid: true}
		} else {
			return nil, fmt.Errorf("invalid end_date format, expected YYYY-MM-DD")
		}
	}

	tradeType := sql.NullString{String: req.TradeType, Valid: req.TradeType != ""}

	var assignmentID int64
	var createdAt, updatedAt time.Time

	query := `
		INSERT INTO iam.user_assignments (
			user_id, role_id, context_type, context_id, trade_type, is_primary,
			start_date, end_date, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`

	err = dao.DB.QueryRowContext(ctx, query,
		req.UserID, req.RoleID, req.ContextType, req.ContextID, tradeType,
		req.IsPrimary, startDate, endDate, userID, userID,
	).Scan(&assignmentID, &createdAt, &updatedAt)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"user_id":      req.UserID,
			"role_id":      req.RoleID,
			"context_type": req.ContextType,
			"context_id":   req.ContextID,
			"error":        err.Error(),
		}).Error("Failed to create assignment")
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"assignment_id": assignmentID,
		"user_id":       req.UserID,
		"context_type":  req.ContextType,
		"context_id":    req.ContextID,
	}).Info("Successfully created assignment")

	// Return the created assignment with enriched data
	return dao.GetAssignment(ctx, assignmentID, 0)
}

// GetAssignment retrieves a specific assignment by ID with enriched data
func (dao *AssignmentDao) GetAssignment(ctx context.Context, assignmentID int64, orgID int64) (*models.AssignmentResponse, error) {
	query := `
		SELECT
			ua.id, ua.user_id, ua.role_id, ua.context_type, ua.context_id,
			ua.trade_type, ua.is_primary, ua.start_date, ua.end_date,
			ua.created_at, ua.created_by, ua.updated_at, ua.updated_by, ua.is_deleted,
			COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '') as user_name,
			u.email as user_email,
			r.name as role_name,
			CASE ua.context_type
				WHEN 'project' THEN (SELECT name FROM project.projects WHERE id = ua.context_id)
				WHEN 'location' THEN (SELECT name FROM iam.locations WHERE id = ua.context_id)
				WHEN 'organization' THEN (SELECT name FROM iam.organizations WHERE id = ua.context_id)
				ELSE 'Unknown'
			END as context_name
		FROM iam.user_assignments ua
		LEFT JOIN iam.users u ON ua.user_id = u.id
		LEFT JOIN iam.roles r ON ua.role_id = r.id
		WHERE ua.id = $1 AND ua.is_deleted = FALSE
	`

	var assignment models.AssignmentResponse
	var tradeType sql.NullString
	var startDate, endDate sql.NullTime

	err := dao.DB.QueryRowContext(ctx, query, assignmentID).Scan(
		&assignment.ID, &assignment.UserID, &assignment.RoleID, &assignment.ContextType, &assignment.ContextID,
		&tradeType, &assignment.IsPrimary, &startDate, &endDate,
		&assignment.CreatedAt, &assignment.CreatedBy, &assignment.UpdatedAt, &assignment.UpdatedBy, &assignment.IsDeleted,
		&assignment.UserName, &assignment.UserEmail, &assignment.RoleName, &assignment.ContextName,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("assignment not found")
	}
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to get assignment")
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}

	// Handle nullable fields
	if tradeType.Valid {
		assignment.TradeType = &tradeType.String
	}
	if startDate.Valid {
		dateStr := startDate.Time.Format("2006-01-02")
		assignment.StartDate = &dateStr
	}
	if endDate.Valid {
		dateStr := endDate.Time.Format("2006-01-02")
		assignment.EndDate = &dateStr
	}

	return &assignment, nil
}

// UpdateAssignment updates an existing assignment
func (dao *AssignmentDao) UpdateAssignment(ctx context.Context, assignmentID int64, req *models.UpdateAssignmentRequest, userID int64) (*models.AssignmentResponse, error) {
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.RoleID != nil {
		setParts = append(setParts, fmt.Sprintf("role_id = $%d", argIndex))
		args = append(args, *req.RoleID)
		argIndex++
	}

	if req.TradeType != "" {
		setParts = append(setParts, fmt.Sprintf("trade_type = $%d", argIndex))
		args = append(args, sql.NullString{String: req.TradeType, Valid: true})
		argIndex++
	}

	if req.IsPrimary != nil {
		setParts = append(setParts, fmt.Sprintf("is_primary = $%d", argIndex))
		args = append(args, *req.IsPrimary)
		argIndex++
	}

	if req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			setParts = append(setParts, fmt.Sprintf("start_date = $%d", argIndex))
			args = append(args, sql.NullTime{Time: t, Valid: true})
			argIndex++
		} else {
			return nil, fmt.Errorf("invalid start_date format, expected YYYY-MM-DD")
		}
	}

	if req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			setParts = append(setParts, fmt.Sprintf("end_date = $%d", argIndex))
			args = append(args, sql.NullTime{Time: t, Valid: true})
			argIndex++
		} else {
			return nil, fmt.Errorf("invalid end_date format, expected YYYY-MM-DD")
		}
	}

	if len(setParts) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// Always update updated_by
	setParts = append(setParts, fmt.Sprintf("updated_by = $%d", argIndex))
	args = append(args, userID)
	argIndex++

	// Add WHERE clause
	args = append(args, assignmentID)
	whereClause := fmt.Sprintf("WHERE id = $%d AND is_deleted = FALSE", argIndex)

	query := fmt.Sprintf(`
		UPDATE iam.user_assignments
		SET %s
		%s
		RETURNING id
	`, strings.Join(setParts, ", "), whereClause)

	var updatedID int64
	err := dao.DB.QueryRowContext(ctx, query, args...).Scan(&updatedID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("assignment not found")
	}
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to update assignment")
		return nil, fmt.Errorf("failed to update assignment: %w", err)
	}

	return dao.GetAssignment(ctx, assignmentID, 0)
}

// DeleteAssignment soft deletes an assignment
func (dao *AssignmentDao) DeleteAssignment(ctx context.Context, assignmentID int64, userID int64) error {
	result, err := dao.DB.ExecContext(ctx, `
		UPDATE iam.user_assignments
		SET is_deleted = TRUE, updated_by = $1
		WHERE id = $2 AND is_deleted = FALSE
	`, userID, assignmentID)

	if err != nil {
		dao.Logger.WithError(err).Error("Failed to delete assignment")
		return fmt.Errorf("failed to delete assignment: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("assignment not found")
	}

	dao.Logger.WithFields(logrus.Fields{
		"assignment_id": assignmentID,
		"deleted_by":    userID,
	}).Info("Successfully deleted assignment")

	return nil
}

// CreateBulkAssignments creates multiple assignments at once
func (dao *AssignmentDao) CreateBulkAssignments(ctx context.Context, req *models.BulkAssignmentRequest, userID int64) ([]models.AssignmentResponse, error) {
	// Validate the context
	err := dao.ValidateAssignmentContext(ctx, req.ContextType, req.ContextID, 0)
	if err != nil {
		return nil, fmt.Errorf("invalid assignment context: %w", err)
	}

	// Parse optional dates
	var startDate, endDate sql.NullTime
	if req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			startDate = sql.NullTime{Time: t, Valid: true}
		} else {
			return nil, fmt.Errorf("invalid start_date format, expected YYYY-MM-DD")
		}
	}
	if req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			endDate = sql.NullTime{Time: t, Valid: true}
		} else {
			return nil, fmt.Errorf("invalid end_date format, expected YYYY-MM-DD")
		}
	}

	tradeType := sql.NullString{String: req.TradeType, Valid: req.TradeType != ""}

	// Use transaction for bulk insert
	tx, err := dao.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var assignmentIDs []int64

	query := `
		INSERT INTO iam.user_assignments (
			user_id, role_id, context_type, context_id, trade_type, is_primary,
			start_date, end_date, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	for _, userIDToAssign := range req.UserIDs {
		var assignmentID int64
		err = tx.QueryRowContext(ctx, query,
			userIDToAssign, req.RoleID, req.ContextType, req.ContextID, tradeType,
			req.IsPrimary, startDate, endDate, userID, userID,
		).Scan(&assignmentID)

		if err != nil {
			dao.Logger.WithFields(logrus.Fields{
				"user_id":      userIDToAssign,
				"context_type": req.ContextType,
				"context_id":   req.ContextID,
				"error":        err.Error(),
			}).Error("Failed to create bulk assignment for user")
			return nil, fmt.Errorf("failed to create assignment for user %d: %w", userIDToAssign, err)
		}

		assignmentIDs = append(assignmentIDs, assignmentID)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit bulk assignments: %w", err)
	}

	// Fetch all created assignments
	var assignments []models.AssignmentResponse
	for _, assignmentID := range assignmentIDs {
		assignment, err := dao.GetAssignment(ctx, assignmentID, 0)
		if err != nil {
			dao.Logger.WithError(err).Warn("Failed to fetch created assignment details")
			continue
		}
		assignments = append(assignments, *assignment)
	}

	dao.Logger.WithFields(logrus.Fields{
		"user_count":   len(req.UserIDs),
		"context_type": req.ContextType,
		"context_id":   req.ContextID,
		"created_count": len(assignments),
	}).Info("Successfully created bulk assignments")

	return assignments, nil
}

// ValidateAssignmentContext validates that a context exists and belongs to the organization
func (dao *AssignmentDao) ValidateAssignmentContext(ctx context.Context, contextType string, contextID int64, orgID int64) error {
	var query string
	var exists bool

	switch contextType {
	case models.ContextTypeOrganization:
		query = "SELECT EXISTS(SELECT 1 FROM iam.organizations WHERE id = $1 AND is_deleted = FALSE)"
	case models.ContextTypeProject:
		query = "SELECT EXISTS(SELECT 1 FROM project.projects WHERE id = $1 AND is_deleted = FALSE)"
	case models.ContextTypeLocation:
		query = "SELECT EXISTS(SELECT 1 FROM iam.locations WHERE id = $1 AND is_deleted = FALSE)"
	default:
		return fmt.Errorf("unsupported context type: %s", contextType)
	}

	err := dao.DB.QueryRowContext(ctx, query, contextID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to validate context: %w", err)
	}

	if !exists {
		return fmt.Errorf("%s with ID %d not found or deleted", contextType, contextID)
	}

	return nil
}

// GetAssignments retrieves assignments with filters
func (dao *AssignmentDao) GetAssignments(ctx context.Context, filters *models.AssignmentFilters, orgID int64) (*models.AssignmentListResponse, error) {
	whereConditions := []string{"ua.is_deleted = FALSE"}
	args := []interface{}{}
	argIndex := 1

	// Build WHERE conditions based on filters
	if filters.UserID != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("ua.user_id = $%d", argIndex))
		args = append(args, *filters.UserID)
		argIndex++
	}

	if filters.RoleID != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("ua.role_id = $%d", argIndex))
		args = append(args, *filters.RoleID)
		argIndex++
	}

	if filters.ContextType != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("ua.context_type = $%d", argIndex))
		args = append(args, filters.ContextType)
		argIndex++
	}

	if filters.ContextID != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("ua.context_id = $%d", argIndex))
		args = append(args, *filters.ContextID)
		argIndex++
	}

	if filters.OrganizationID != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("u.org_id = $%d", argIndex))
		args = append(args, *filters.OrganizationID)
		argIndex++
	}

	if filters.IsPrimary != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("ua.is_primary = $%d", argIndex))
		args = append(args, *filters.IsPrimary)
		argIndex++
	}

	if filters.TradeType != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("ua.trade_type = $%d", argIndex))
		args = append(args, filters.TradeType)
		argIndex++
	}

	// Active filter based on dates
	if filters.IsActive != nil && *filters.IsActive {
		now := time.Now()
		whereConditions = append(whereConditions, fmt.Sprintf("(ua.start_date IS NULL OR ua.start_date <= $%d)", argIndex))
		args = append(args, now)
		argIndex++
		whereConditions = append(whereConditions, fmt.Sprintf("(ua.end_date IS NULL OR ua.end_date >= $%d)", argIndex))
		args = append(args, now)
		argIndex++
	}

	// Pagination
	page := 1
	pageSize := 50
	if filters.Page > 0 {
		page = filters.Page
	}
	if filters.PageSize > 0 && filters.PageSize <= 100 {
		pageSize = filters.PageSize
	}
	offset := (page - 1) * pageSize

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM iam.user_assignments ua
		LEFT JOIN iam.users u ON ua.user_id = u.id
		WHERE %s
	`, strings.Join(whereConditions, " AND "))

	var total int
	err := dao.DB.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count assignments: %w", err)
	}

	// Main query with enriched data
	query := fmt.Sprintf(`
		SELECT
			ua.id, ua.user_id, ua.role_id, ua.context_type, ua.context_id,
			ua.trade_type, ua.is_primary, ua.start_date, ua.end_date,
			ua.created_at, ua.created_by, ua.updated_at, ua.updated_by, ua.is_deleted,
			COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '') as user_name,
			u.email as user_email,
			r.name as role_name,
			CASE ua.context_type
				WHEN 'project' THEN (SELECT name FROM project.projects WHERE id = ua.context_id)
				WHEN 'location' THEN (SELECT name FROM iam.locations WHERE id = ua.context_id)
				WHEN 'organization' THEN (SELECT name FROM iam.organizations WHERE id = ua.context_id)
				ELSE 'Unknown'
			END as context_name
		FROM iam.user_assignments ua
		LEFT JOIN iam.users u ON ua.user_id = u.id
		LEFT JOIN iam.roles r ON ua.role_id = r.id
		WHERE %s
		ORDER BY ua.created_at DESC
		LIMIT $%d OFFSET $%d
	`, strings.Join(whereConditions, " AND "), argIndex, argIndex+1)

	args = append(args, pageSize, offset)

	rows, err := dao.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query assignments: %w", err)
	}
	defer rows.Close()

	var assignments []models.AssignmentResponse
	for rows.Next() {
		var assignment models.AssignmentResponse
		var tradeType sql.NullString
		var startDate, endDate sql.NullTime

		err := rows.Scan(
			&assignment.ID, &assignment.UserID, &assignment.RoleID, &assignment.ContextType, &assignment.ContextID,
			&tradeType, &assignment.IsPrimary, &startDate, &endDate,
			&assignment.CreatedAt, &assignment.CreatedBy, &assignment.UpdatedAt, &assignment.UpdatedBy, &assignment.IsDeleted,
			&assignment.UserName, &assignment.UserEmail, &assignment.RoleName, &assignment.ContextName,
		)
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan assignment row")
			continue
		}

		// Handle nullable fields
		if tradeType.Valid {
			assignment.TradeType = &tradeType.String
		}
		if startDate.Valid {
			dateStr := startDate.Time.Format("2006-01-02")
			assignment.StartDate = &dateStr
		}
		if endDate.Valid {
			dateStr := endDate.Time.Format("2006-01-02")
			assignment.EndDate = &dateStr
		}

		assignments = append(assignments, assignment)
	}

	return &models.AssignmentListResponse{
		Assignments: assignments,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

// GetUserAssignments gets all assignments for a specific user
func (dao *AssignmentDao) GetUserAssignments(ctx context.Context, userID int64, orgID int64) (*models.UserAssignmentSummary, error) {
	filters := &models.AssignmentFilters{
		UserID:         &userID,
		OrganizationID: &orgID,
	}

	assignmentList, err := dao.GetAssignments(ctx, filters, orgID)
	if err != nil {
		return nil, err
	}

	// Get user details
	var userName, userEmail, orgName string
	userQuery := `
		SELECT
			COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '') as user_name,
			u.email,
			o.name as org_name
		FROM iam.users u
		LEFT JOIN iam.organizations o ON u.org_id = o.id
		WHERE u.id = $1
	`
	err = dao.DB.QueryRowContext(ctx, userQuery, userID).Scan(&userName, &userEmail, &orgName)
	if err != nil {
		return nil, fmt.Errorf("failed to get user details: %w", err)
	}

	// Calculate statistics
	activeCount := 0
	assignmentsByType := make(map[string]int)

	for _, assignment := range assignmentList.Assignments {
		assignmentsByType[assignment.ContextType]++
	}

	return &models.UserAssignmentSummary{
		UserID:            userID,
		UserName:          userName,
		UserEmail:         userEmail,
		OrgID:             orgID,
		OrgName:           orgName,
		TotalAssignments:  assignmentList.Total,
		ActiveAssignments: activeCount,
		AssignmentsByType: assignmentsByType,
		Assignments:       assignmentList.Assignments,
	}, nil
}

// GetContextAssignments gets all assignments for a specific context
func (dao *AssignmentDao) GetContextAssignments(ctx context.Context, contextType string, contextID int64, orgID int64) (*models.ContextAssignmentSummary, error) {
	filters := &models.AssignmentFilters{
		ContextType:    contextType,
		ContextID:      &contextID,
		OrganizationID: &orgID,
	}

	assignmentList, err := dao.GetAssignments(ctx, filters, orgID)
	if err != nil {
		return nil, err
	}

	return &models.ContextAssignmentSummary{
		ContextType: contextType,
		ContextID:   contextID,
		ContextName: "Context",
		OrgID:       orgID,
		Assignments: assignmentList.Assignments,
	}, nil
}

// TransferAssignments transfers assignments from one user to another
func (dao *AssignmentDao) TransferAssignments(ctx context.Context, req *models.AssignmentTransferRequest, userID int64) error {
	tx, err := dao.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var query string
	var args []interface{}

	if len(req.AssignmentIDs) > 0 {
		// Transfer specific assignments
		placeholders := make([]string, len(req.AssignmentIDs))
		args = append(args, req.ToUserID, userID)
		for i, id := range req.AssignmentIDs {
			placeholders[i] = "$" + strconv.Itoa(i+3)
			args = append(args, id)
		}

		query = fmt.Sprintf(`
			UPDATE iam.user_assignments
			SET user_id = $1, updated_by = $2
			WHERE id IN (%s) AND user_id = %d AND is_deleted = FALSE
		`, strings.Join(placeholders, ","), req.FromUserID)
	} else {
		// Transfer all active assignments
		now := time.Now()
		query = `
			UPDATE iam.user_assignments
			SET user_id = $1, updated_by = $2
			WHERE user_id = $3 AND is_deleted = FALSE
				AND (start_date IS NULL OR start_date <= $4)
				AND (end_date IS NULL OR end_date >= $5)
		`
		args = []interface{}{req.ToUserID, userID, req.FromUserID, now, now}
	}

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to transfer assignments: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no assignments found to transfer")
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transfer: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"from_user_id":    req.FromUserID,
		"to_user_id":      req.ToUserID,
		"transferred_count": rowsAffected,
		"transferred_by":  userID,
	}).Info("Successfully transferred assignments")

	return nil
}

// CheckPermission checks if a user has permission for a specific context
func (dao *AssignmentDao) CheckPermission(ctx context.Context, req *models.PermissionCheckRequest, orgID int64) (*models.PermissionCheckResponse, error) {
	// This is a simplified permission check - you would expand this based on your role-permission mapping
	query := `
		SELECT ua.id, r.name, ua.context_type, ua.context_id
		FROM iam.user_assignments ua
		LEFT JOIN iam.roles r ON ua.role_id = r.id
		WHERE ua.user_id = $1
			AND ua.context_type = $2
			AND ua.context_id = $3
			AND ua.is_deleted = FALSE
			AND (ua.start_date IS NULL OR ua.start_date <= NOW())
			AND (ua.end_date IS NULL OR ua.end_date >= NOW())
	`

	var assignmentID int64
	var roleName, contextType string
	var contextID int64

	err := dao.DB.QueryRowContext(ctx, query, req.UserID, req.ContextType, req.ContextID).Scan(
		&assignmentID, &roleName, &contextType, &contextID,
	)

	if err == sql.ErrNoRows {
		return &models.PermissionCheckResponse{
			HasPermission: false,
			Reason:        "No direct assignment found",
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to check permission: %w", err)
	}

	// Simple permission logic - you would expand this based on your requirements
	hasPermission := true // Assuming any role assignment grants basic access

	return &models.PermissionCheckResponse{
		HasPermission: hasPermission,
		UserRoles:     []string{roleName},
	}, nil
}

// GetUserContexts gets all context IDs of a specific type that a user has access to
func (dao *AssignmentDao) GetUserContexts(ctx context.Context, userID int64, contextType string, orgID int64) ([]int64, error) {
	query := `
		SELECT DISTINCT ua.context_id
		FROM iam.user_assignments ua
		LEFT JOIN iam.users u ON ua.user_id = u.id
		WHERE ua.user_id = $1
			AND ua.context_type = $2
			AND u.org_id = $3
			AND ua.is_deleted = FALSE
			AND (ua.start_date IS NULL OR ua.start_date <= NOW())
			AND (ua.end_date IS NULL OR ua.end_date >= NOW())
		ORDER BY ua.context_id
	`

	rows, err := dao.DB.QueryContext(ctx, query, userID, contextType, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user contexts: %w", err)
	}
	defer rows.Close()

	var contextIDs []int64
	for rows.Next() {
		var contextID int64
		if err := rows.Scan(&contextID); err != nil {
			dao.Logger.WithError(err).Error("Failed to scan context ID")
			continue
		}
		contextIDs = append(contextIDs, contextID)
	}

	return contextIDs, nil
}

// GetActiveAssignments gets all active assignments for a user
func (dao *AssignmentDao) GetActiveAssignments(ctx context.Context, userID int64, orgID int64) ([]models.AssignmentResponse, error) {
	isActive := true
	filters := &models.AssignmentFilters{
		UserID:         &userID,
		OrganizationID: &orgID,
		IsActive:       &isActive,
	}

	assignmentList, err := dao.GetAssignments(ctx, filters, orgID)
	if err != nil {
		return nil, err
	}

	return assignmentList.Assignments, nil
}