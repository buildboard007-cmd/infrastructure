package data

import (
	"context"
	"database/sql"
	"fmt"
	"infrastructure/lib/models"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// ProjectRepository defines the interface for project data operations
type ProjectRepository interface {
	// Project CRUD operations
	CreateProject(ctx context.Context, orgID int64, project *models.CreateProjectRequest, userID int64) (*models.CreateProjectResponse, error)
	CreateProjectLegacy(ctx context.Context, orgID int64, project *models.LegacyCreateProjectRequest, userID int64) (*models.Project, error)
	GetProjectsByOrg(ctx context.Context, orgID int64) ([]models.Project, error)
	GetProjectByID(ctx context.Context, projectID, orgID int64) (*models.Project, error)
	UpdateProject(ctx context.Context, projectID, orgID int64, project *models.UpdateProjectRequest, userID int64) (*models.Project, error)
	DeleteProject(ctx context.Context, projectID, orgID int64, userID int64) error
	
	// Project Manager operations
	CreateProjectManager(ctx context.Context, projectID int64, manager *models.CreateProjectManagerRequest, userID int64) (*models.ProjectManager, error)
	GetProjectManagersByProject(ctx context.Context, projectID int64) ([]models.ProjectManager, error)
	GetProjectManagerByID(ctx context.Context, managerID, projectID int64) (*models.ProjectManager, error)
	UpdateProjectManager(ctx context.Context, managerID, projectID int64, manager *models.UpdateProjectManagerRequest, userID int64) (*models.ProjectManager, error)
	DeleteProjectManager(ctx context.Context, managerID, projectID int64, userID int64) error
	
	// Project Attachment operations
	CreateProjectAttachment(ctx context.Context, projectID int64, attachment *models.CreateProjectAttachmentRequest, userID int64) (*models.ProjectAttachment, error)
	GetProjectAttachmentsByProject(ctx context.Context, projectID int64) ([]models.ProjectAttachment, error)
	GetProjectAttachmentByID(ctx context.Context, attachmentID, projectID int64) (*models.ProjectAttachment, error)
	DeleteProjectAttachment(ctx context.Context, attachmentID, projectID int64, userID int64) error
	
	// Project User Role operations
	AssignUserToProject(ctx context.Context, projectID int64, assignment *models.CreateProjectUserRoleRequest, userID int64) (*models.ProjectUserRole, error)
	GetProjectUserRoles(ctx context.Context, projectID int64) ([]models.ProjectUserRole, error)
	UpdateProjectUserRole(ctx context.Context, assignmentID, projectID int64, assignment *models.UpdateProjectUserRoleRequest, userID int64) (*models.ProjectUserRole, error)
	RemoveUserFromProject(ctx context.Context, assignmentID, projectID int64, userID int64) error
}

// ProjectDao implements ProjectRepository interface using PostgreSQL
type ProjectDao struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

// NewProjectRepository creates a new ProjectRepository instance
func NewProjectRepository(db *sql.DB) ProjectRepository {
	return &ProjectDao{
		DB:     db,
		Logger: logrus.New(),
	}
}

// CreateProject creates a new project in the organization
func (dao *ProjectDao) CreateProjectLegacy(ctx context.Context, orgID int64, request *models.LegacyCreateProjectRequest, userID int64) (*models.Project, error) {
	var projectID int64
	var createdAt, updatedAt time.Time
	
	// Convert optional fields to sql.Null types  
	projectNumber := sql.NullString{String: request.ProjectNumber, Valid: request.ProjectNumber != ""}
	description := sql.NullString{String: request.Description, Valid: request.Description != ""}
	projectStage := sql.NullString{String: request.ProjectStage, Valid: request.ProjectStage != ""}
	workScope := sql.NullString{String: request.WorkScope, Valid: request.WorkScope != ""}
	projectSector := sql.NullString{String: request.ProjectSector, Valid: request.ProjectSector != ""}
	deliveryMethod := sql.NullString{String: request.DeliveryMethod, Valid: request.DeliveryMethod != ""}
	
	// Handle date fields
	startDate := sql.NullTime{}
	if request.StartDate != "" {
		if t, err := time.Parse("2006-01-02", request.StartDate); err == nil {
			startDate = sql.NullTime{Time: t, Valid: true}
		}
	}
	
	plannedEndDate := sql.NullTime{}
	if request.PlannedEndDate != "" {
		if t, err := time.Parse("2006-01-02", request.PlannedEndDate); err == nil {
			plannedEndDate = sql.NullTime{Time: t, Valid: true}
		}
	}
	
	// Set defaults
	projectPhase := request.ProjectPhase
	if projectPhase == "" {
		projectPhase = "pre_construction"
	}
	
	country := request.Country
	if country == "" {
		country = "USA"
	}
	
	language := request.Language
	if language == "" {
		language = "en"
	}
	
	status := request.Status
	if status == "" {
		status = "active"
	}

	query := `
		INSERT INTO project.projects (
			org_id, location_id, project_number, name, description, project_type,
			project_stage, work_scope, project_sector, delivery_method, project_phase,
			start_date, planned_end_date, budget, contract_value, square_footage,
			address, city, state, zip_code, country, language, latitude, longitude,
			status, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27)
		RETURNING id, created_at, updated_at
	`

	err := dao.DB.QueryRowContext(ctx, query,
		orgID, request.LocationID, projectNumber, request.Name, description, request.ProjectType,
		projectStage, workScope, projectSector, deliveryMethod, projectPhase,
		startDate, plannedEndDate, request.Budget, request.ContractValue, request.SquareFootage,
		request.Address, request.City, request.State, request.ZipCode, country, language,
		request.Latitude, request.Longitude, status, userID, userID,
	).Scan(&projectID, &createdAt, &updatedAt)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"org_id": orgID,
			"name":   request.Name,
			"error":  err.Error(),
		}).Error("Failed to create project")
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"org_id":     orgID,
		"name":       request.Name,
	}).Info("Successfully created project")

	// Return the created project
	return dao.GetProjectByID(ctx, projectID, orgID)
}

// CreateProject creates a new project following the API contract structure
func (dao *ProjectDao) CreateProject(ctx context.Context, orgID int64, request *models.CreateProjectRequest, userID int64) (*models.CreateProjectResponse, error) {
	// Start transaction for atomic project and manager creation
	tx, err := dao.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate project number (PROJ-YYYY-NNNN format)
	projectNumber, err := dao.generateProjectNumber(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate project number: %w", err)
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", request.Timeline.StartDate)
	if err != nil {
		return &models.CreateProjectResponse{
			Success: false,
			Message: "Validation failed",
			Errors:  map[string][]string{"start_date": {"Invalid date format, expected YYYY-MM-DD"}},
		}, nil
	}

	// Validate start date is not in the past
	if startDate.Before(time.Now().Truncate(24 * time.Hour)) {
		return &models.CreateProjectResponse{
			Success: false,
			Message: "Validation failed",
			Errors:  map[string][]string{"start_date": {"Start date cannot be in the past"}},
		}, nil
	}

	// Handle optional dates
	var substantialCompletionDate, projectFinishDate, warrantyStartDate, warrantyEndDate sql.NullTime
	
	if request.Timeline.SubstantialCompletionDate != "" {
		if t, err := time.Parse("2006-01-02", request.Timeline.SubstantialCompletionDate); err == nil {
			if t.After(startDate) {
				substantialCompletionDate = sql.NullTime{Time: t, Valid: true}
			} else {
				return &models.CreateProjectResponse{
					Success: false,
					Message: "Validation failed", 
					Errors:  map[string][]string{"substantial_completion_date": {"Must be after start_date"}},
				}, nil
			}
		} else {
			return &models.CreateProjectResponse{
				Success: false,
				Message: "Validation failed",
				Errors:  map[string][]string{"substantial_completion_date": {"Invalid date format, expected YYYY-MM-DD"}},
			}, nil
		}
	}

	if request.Timeline.ProjectFinishDate != "" {
		if t, err := time.Parse("2006-01-02", request.Timeline.ProjectFinishDate); err == nil {
			projectFinishDate = sql.NullTime{Time: t, Valid: true}
		}
	}

	if request.Timeline.WarrantyStartDate != "" {
		if t, err := time.Parse("2006-01-02", request.Timeline.WarrantyStartDate); err == nil {
			warrantyStartDate = sql.NullTime{Time: t, Valid: true}
		}
	}

	if request.Timeline.WarrantyEndDate != "" {
		if t, err := time.Parse("2006-01-02", request.Timeline.WarrantyEndDate); err == nil {
			warrantyEndDate = sql.NullTime{Time: t, Valid: true}
		}
	}

	// Set defaults and map language
	language := "en" // Default
	if request.ProjectDetails.Language != "" {
		// Map full language names to codes
		switch strings.ToLower(request.ProjectDetails.Language) {
		case "english":
			language = "en"
		case "spanish":
			language = "es"
		case "french":
			language = "fr"
		default:
			// If it's already a code, use it
			if len(request.ProjectDetails.Language) == 2 {
				language = request.ProjectDetails.Language
			}
		}
	}
	
	status := request.ProjectDetails.Status
	if status == "" {
		status = "active"
	}
	
	country := request.Location.Country
	if country == "" {
		country = "USA"
	}

	// Create project
	var projectID int64
	var createdAt, updatedAt time.Time
	
	query := `
		INSERT INTO project.projects (
			org_id, location_id, project_number, name, description, project_type,
			project_stage, work_scope, project_sector, delivery_method, project_phase,
			start_date, substantial_completion_date, project_finish_date, 
			warranty_start_date, warranty_end_date, budget, square_footage,
			address, city, state, zip_code, country, language, status, 
			created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27)
		RETURNING id, created_at, updated_at
	`

	// Use location_id from request
	locationID := request.LocationID
	
	// Use project_sector as project_type (they have the same valid values)
	projectType := request.ProjectDetails.ProjectSector
	if projectType == "" {
		projectType = "commercial" // Default
	}

	err = tx.QueryRowContext(ctx, query,
		orgID, locationID, projectNumber, request.BasicInfo.Name, 
		sql.NullString{String: request.BasicInfo.Description, Valid: request.BasicInfo.Description != ""},
		projectType,
		request.ProjectDetails.ProjectStage, request.ProjectDetails.WorkScope,
		request.ProjectDetails.ProjectSector, request.ProjectDetails.DeliveryMethod,
		"pre_construction", // Default project_phase
		startDate, substantialCompletionDate, projectFinishDate,
		warrantyStartDate, warrantyEndDate,
		sql.NullFloat64{Float64: request.Financial.Budget, Valid: request.Financial.Budget > 0},
		sql.NullInt64{Int64: request.ProjectDetails.SquareFootage, Valid: request.ProjectDetails.SquareFootage > 0},
		request.Location.Address, sql.NullString{String: request.Location.City, Valid: request.Location.City != ""},
		sql.NullString{String: request.Location.State, Valid: request.Location.State != ""},
		sql.NullString{String: request.Location.ZipCode, Valid: request.Location.ZipCode != ""},
		country, language, status, userID, userID,
	).Scan(&projectID, &createdAt, &updatedAt)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"org_id": orgID,
			"name":   request.BasicInfo.Name,
			"error":  err.Error(),
		}).Error("Failed to create project")
		return nil, fmt.Errorf("failed to create project: %w", err)
	}


	// Commit transaction
	if err = tx.Commit(); err != nil {
		dao.Logger.WithError(err).Error("Failed to commit project creation transaction")
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"project_id":     projectID,
		"project_number": projectNumber,
		"org_id":         orgID,
		"name":           request.BasicInfo.Name,
	}).Info("Successfully created project with manager")

	// Return success response
	return &models.CreateProjectResponse{
		Success: true,
		Message: "Project created successfully",
		Data: models.CreateProjectData{
			ProjectID:     fmt.Sprintf("%d", projectID),
			ProjectNumber: projectNumber,
			Name:          request.BasicInfo.Name,
			Status:        status,
			CreatedAt:     createdAt,
			CreatedBy:     userID,
		},
	}, nil
}

// generateProjectNumber generates a unique project number in PROJ-YYYY-NNNN format
func (dao *ProjectDao) generateProjectNumber(ctx context.Context, orgID int64) (string, error) {
	currentYear := time.Now().Year()
	
	// Find the next available number for this year
	var nextNum int
	query := `
		SELECT COALESCE(MAX(CAST(SUBSTRING(project_number, 11) AS INTEGER)), 0) + 1
		FROM project.projects 
		WHERE org_id = $1 AND project_number LIKE $2
	`
	
	yearPrefix := fmt.Sprintf("PROJ-%d-%%", currentYear)
	err := dao.DB.QueryRowContext(ctx, query, orgID, yearPrefix).Scan(&nextNum)
	if err != nil {
		return "", fmt.Errorf("failed to generate project number: %w", err)
	}
	
	return fmt.Sprintf("PROJ-%d-%04d", currentYear, nextNum), nil
}

// GetProjectsByOrg retrieves all projects for a specific organization
func (dao *ProjectDao) GetProjectsByOrg(ctx context.Context, orgID int64) ([]models.Project, error) {
	query := `
		SELECT id, org_id, location_id, project_number, name, description, project_type,
		       project_stage, work_scope, project_sector, delivery_method, project_phase,
		       start_date, planned_end_date, actual_start_date, actual_end_date,
		       substantial_completion_date, project_finish_date, warranty_start_date, warranty_end_date,
		       budget, contract_value, square_footage, address, city, state, zip_code,
		       country, language, latitude, longitude, status, created_at, created_by, updated_at, updated_by
		FROM project.projects
		WHERE org_id = $1 AND is_deleted = FALSE
		ORDER BY created_at DESC
	`

	rows, err := dao.DB.QueryContext(ctx, query, orgID)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"org_id": orgID,
			"error":  err.Error(),
		}).Error("Failed to query projects")
		return nil, fmt.Errorf("failed to query projects: %w", err)
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var project models.Project
		err := rows.Scan(
			&project.ProjectID, &project.OrgID, &project.LocationID, &project.ProjectNumber,
			&project.Name, &project.Description, &project.ProjectType, &project.ProjectStage,
			&project.WorkScope, &project.ProjectSector, &project.DeliveryMethod, &project.ProjectPhase,
			&project.StartDate, &project.PlannedEndDate, &project.ActualStartDate, &project.ActualEndDate,
			&project.SubstantialCompletionDate, &project.ProjectFinishDate, &project.WarrantyStartDate, &project.WarrantyEndDate,
			&project.Budget, &project.ContractValue, &project.SquareFootage, &project.Address,
			&project.City, &project.State, &project.ZipCode, &project.Country, &project.Language,
			&project.Latitude, &project.Longitude, &project.Status, &project.CreatedAt,
			&project.CreatedBy, &project.UpdatedAt, &project.UpdatedBy,
		)
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan project row")
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, project)
	}

	if err = rows.Err(); err != nil {
		dao.Logger.WithError(err).Error("Error iterating project rows")
		return nil, fmt.Errorf("error iterating projects: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"org_id": orgID,
		"count":  len(projects),
	}).Debug("Successfully retrieved projects for organization")

	return projects, nil
}

// GetProjectByID retrieves a specific project by ID with organization validation
func (dao *ProjectDao) GetProjectByID(ctx context.Context, projectID, orgID int64) (*models.Project, error) {
	var project models.Project
	query := `
		SELECT id, org_id, location_id, project_number, name, description, project_type,
		       project_stage, work_scope, project_sector, delivery_method, project_phase,
		       start_date, planned_end_date, actual_start_date, actual_end_date,
		       substantial_completion_date, project_finish_date, warranty_start_date, warranty_end_date,
		       budget, contract_value, square_footage, address, city, state, zip_code,
		       country, language, latitude, longitude, status, created_at, created_by, updated_at, updated_by
		FROM project.projects
		WHERE id = $1 AND org_id = $2 AND is_deleted = FALSE
	`

	err := dao.DB.QueryRowContext(ctx, query, projectID, orgID).Scan(
		&project.ProjectID, &project.OrgID, &project.LocationID, &project.ProjectNumber,
		&project.Name, &project.Description, &project.ProjectType, &project.ProjectStage,
		&project.WorkScope, &project.ProjectSector, &project.DeliveryMethod, &project.ProjectPhase,
		&project.StartDate, &project.PlannedEndDate, &project.ActualStartDate, &project.ActualEndDate,
		&project.SubstantialCompletionDate, &project.ProjectFinishDate, &project.WarrantyStartDate, &project.WarrantyEndDate,
		&project.Budget, &project.ContractValue, &project.SquareFootage, &project.Address,
		&project.City, &project.State, &project.ZipCode, &project.Country, &project.Language,
		&project.Latitude, &project.Longitude, &project.Status, &project.CreatedAt,
		&project.CreatedBy, &project.UpdatedAt, &project.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		dao.Logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"org_id":     orgID,
		}).Warn("Project not found")
		return nil, fmt.Errorf("project not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"org_id":     orgID,
			"error":      err.Error(),
		}).Error("Failed to get project")
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &project, nil
}

// UpdateProject updates an existing project
func (dao *ProjectDao) UpdateProject(ctx context.Context, projectID, orgID int64, request *models.UpdateProjectRequest, userID int64) (*models.Project, error) {
	// Build dynamic update query based on provided fields
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if request.LocationID != 0 {
		setParts = append(setParts, fmt.Sprintf("location_id = $%d", argIndex))
		args = append(args, request.LocationID)
		argIndex++
	}
	
	if request.ProjectNumber != "" {
		setParts = append(setParts, fmt.Sprintf("project_number = $%d", argIndex))
		args = append(args, request.ProjectNumber)
		argIndex++
	}
	
	if request.Name != "" {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, request.Name)
		argIndex++
	}
	
	if request.Description != "" {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, request.Description)
		argIndex++
	}
	
	if request.ProjectType != "" {
		setParts = append(setParts, fmt.Sprintf("project_type = $%d", argIndex))
		args = append(args, request.ProjectType)
		argIndex++
	}
	
	if request.Status != "" {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, request.Status)
		argIndex++
	}

	// Always update updated_by and updated_at
	setParts = append(setParts, fmt.Sprintf("updated_by = $%d", argIndex))
	args = append(args, userID)
	argIndex++

	// Add WHERE clause parameters
	args = append(args, projectID, orgID)
	whereClause := fmt.Sprintf("WHERE id = $%d AND org_id = $%d AND is_deleted = FALSE", argIndex, argIndex+1)

	if len(setParts) == 1 { // Only updated_by was set
		return nil, fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf(`
		UPDATE project.projects 
		SET %s
		%s
		RETURNING id, org_id, location_id, project_number, name, description, project_type,
		          project_stage, work_scope, project_sector, delivery_method, project_phase,
		          start_date, planned_end_date, actual_start_date, actual_end_date,
		          substantial_completion_date, project_finish_date, warranty_start_date, warranty_end_date,
		          budget, contract_value, square_footage, address, city, state, zip_code,
		          country, language, latitude, longitude, status, created_at, created_by, updated_at, updated_by
	`, 
		setParts[0]+", "+setParts[1:][0], // Join SET parts with commas
		whereClause,
	)

	var project models.Project
	err := dao.DB.QueryRowContext(ctx, query, args...).Scan(
		&project.ProjectID, &project.OrgID, &project.LocationID, &project.ProjectNumber,
		&project.Name, &project.Description, &project.ProjectType, &project.ProjectStage,
		&project.WorkScope, &project.ProjectSector, &project.DeliveryMethod, &project.ProjectPhase,
		&project.StartDate, &project.PlannedEndDate, &project.ActualStartDate, &project.ActualEndDate,
		&project.SubstantialCompletionDate, &project.ProjectFinishDate, &project.WarrantyStartDate, &project.WarrantyEndDate,
		&project.Budget, &project.ContractValue, &project.SquareFootage, &project.Address,
		&project.City, &project.State, &project.ZipCode, &project.Country, &project.Language,
		&project.Latitude, &project.Longitude, &project.Status, &project.CreatedAt,
		&project.CreatedBy, &project.UpdatedAt, &project.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		dao.Logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"org_id":     orgID,
		}).Warn("Project not found for update")
		return nil, fmt.Errorf("project not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"org_id":     orgID,
			"error":      err.Error(),
		}).Error("Failed to update project")
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"project_id": project.ProjectID,
		"org_id":     orgID,
		"name":       project.Name,
	}).Info("Successfully updated project")

	return &project, nil
}

// DeleteProject deletes a project (soft delete)
func (dao *ProjectDao) DeleteProject(ctx context.Context, projectID, orgID int64, userID int64) error {
	result, err := dao.DB.ExecContext(ctx, `
		UPDATE project.projects 
		SET is_deleted = TRUE, updated_by = $1
		WHERE id = $2 AND org_id = $3 AND is_deleted = FALSE
	`, userID, projectID, orgID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"org_id":     orgID,
			"error":      err.Error(),
		}).Error("Failed to delete project")
		return fmt.Errorf("failed to delete project: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		dao.Logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"org_id":     orgID,
		}).Warn("Project not found for deletion")
		return fmt.Errorf("project not found")
	}

	dao.Logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"org_id":     orgID,
	}).Info("Successfully deleted project")

	return nil
}

// CreateProjectManager creates a new project manager
func (dao *ProjectDao) CreateProjectManager(ctx context.Context, projectID int64, request *models.CreateProjectManagerRequest, userID int64) (*models.ProjectManager, error) {
	var managerID int64
	var createdAt, updatedAt time.Time
	
	officeContact := sql.NullString{String: request.OfficeContact, Valid: request.OfficeContact != ""}
	mobileContact := sql.NullString{String: request.MobileContact, Valid: request.MobileContact != ""}

	query := `
		INSERT INTO project.project_managers (
			project_id, name, company, role, email, office_contact, mobile_contact, is_primary, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`

	err := dao.DB.QueryRowContext(ctx, query,
		projectID, request.Name, request.Company, request.Role, request.Email,
		officeContact, mobileContact, request.IsPrimary, userID, userID,
	).Scan(&managerID, &createdAt, &updatedAt)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"name":       request.Name,
			"error":      err.Error(),
		}).Error("Failed to create project manager")
		return nil, fmt.Errorf("failed to create project manager: %w", err)
	}

	return &models.ProjectManager{
		ID:            managerID,
		ProjectID:     projectID,
		Name:          request.Name,
		Company:       request.Company,
		Role:          request.Role,
		Email:         request.Email,
		OfficeContact: officeContact,
		MobileContact: mobileContact,
		IsPrimary:     request.IsPrimary,
		CreatedAt:     createdAt,
		CreatedBy:     userID,
		UpdatedAt:     updatedAt,
		UpdatedBy:     userID,
	}, nil
}

// GetProjectManagersByProject retrieves all project managers for a project
func (dao *ProjectDao) GetProjectManagersByProject(ctx context.Context, projectID int64) ([]models.ProjectManager, error) {
	query := `
		SELECT id, project_id, name, company, role, email, office_contact, mobile_contact, 
		       is_primary, created_at, created_by, updated_at, updated_by
		FROM project.project_managers
		WHERE project_id = $1 AND is_deleted = FALSE
		ORDER BY is_primary DESC, created_at ASC
	`

	rows, err := dao.DB.QueryContext(ctx, query, projectID)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"error":      err.Error(),
		}).Error("Failed to query project managers")
		return nil, fmt.Errorf("failed to query project managers: %w", err)
	}
	defer rows.Close()

	var managers []models.ProjectManager
	for rows.Next() {
		var manager models.ProjectManager
		err := rows.Scan(
			&manager.ID, &manager.ProjectID, &manager.Name, &manager.Company, &manager.Role,
			&manager.Email, &manager.OfficeContact, &manager.MobileContact, &manager.IsPrimary,
			&manager.CreatedAt, &manager.CreatedBy, &manager.UpdatedAt, &manager.UpdatedBy,
		)
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan project manager row")
			return nil, fmt.Errorf("failed to scan project manager: %w", err)
		}
		managers = append(managers, manager)
	}

	return managers, nil
}

// GetProjectManagerByID retrieves a specific project manager by ID
func (dao *ProjectDao) GetProjectManagerByID(ctx context.Context, managerID, projectID int64) (*models.ProjectManager, error) {
	var manager models.ProjectManager
	query := `
		SELECT id, project_id, name, company, role, email, office_contact, mobile_contact, 
		       is_primary, created_at, created_by, updated_at, updated_by
		FROM project.project_managers
		WHERE id = $1 AND project_id = $2 AND is_deleted = FALSE
	`

	err := dao.DB.QueryRowContext(ctx, query, managerID, projectID).Scan(
		&manager.ID, &manager.ProjectID, &manager.Name, &manager.Company, &manager.Role,
		&manager.Email, &manager.OfficeContact, &manager.MobileContact, &manager.IsPrimary,
		&manager.CreatedAt, &manager.CreatedBy, &manager.UpdatedAt, &manager.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project manager not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"manager_id": managerID,
			"project_id": projectID,
			"error":      err.Error(),
		}).Error("Failed to get project manager")
		return nil, fmt.Errorf("failed to get project manager: %w", err)
	}

	return &manager, nil
}

// UpdateProjectManager updates an existing project manager
func (dao *ProjectDao) UpdateProjectManager(ctx context.Context, managerID, projectID int64, request *models.UpdateProjectManagerRequest, userID int64) (*models.ProjectManager, error) {
	// Build dynamic update query based on provided fields
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if request.Name != "" {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, request.Name)
		argIndex++
	}
	
	if request.Company != "" {
		setParts = append(setParts, fmt.Sprintf("company = $%d", argIndex))
		args = append(args, request.Company)
		argIndex++
	}
	
	if request.Role != "" {
		setParts = append(setParts, fmt.Sprintf("role = $%d", argIndex))
		args = append(args, request.Role)
		argIndex++
	}
	
	if request.Email != "" {
		setParts = append(setParts, fmt.Sprintf("email = $%d", argIndex))
		args = append(args, request.Email)
		argIndex++
	}

	// Always update updated_by
	setParts = append(setParts, fmt.Sprintf("updated_by = $%d", argIndex))
	args = append(args, userID)
	argIndex++

	// Add WHERE clause parameters
	args = append(args, managerID, projectID)
	whereClause := fmt.Sprintf("WHERE id = $%d AND project_id = $%d AND is_deleted = FALSE", argIndex, argIndex+1)

	if len(setParts) == 1 { // Only updated_by was set
		return nil, fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf(`
		UPDATE project.project_managers 
		SET %s
		%s
		RETURNING id, project_id, name, company, role, email, office_contact, mobile_contact, 
		          is_primary, created_at, created_by, updated_at, updated_by
	`, 
		setParts[0]+", "+setParts[1:][0], // Join SET parts with commas
		whereClause,
	)

	var manager models.ProjectManager
	err := dao.DB.QueryRowContext(ctx, query, args...).Scan(
		&manager.ID, &manager.ProjectID, &manager.Name, &manager.Company, &manager.Role,
		&manager.Email, &manager.OfficeContact, &manager.MobileContact, &manager.IsPrimary,
		&manager.CreatedAt, &manager.CreatedBy, &manager.UpdatedAt, &manager.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project manager not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"manager_id": managerID,
			"project_id": projectID,
			"error":      err.Error(),
		}).Error("Failed to update project manager")
		return nil, fmt.Errorf("failed to update project manager: %w", err)
	}

	return &manager, nil
}

// DeleteProjectManager deletes a project manager (soft delete)
func (dao *ProjectDao) DeleteProjectManager(ctx context.Context, managerID, projectID int64, userID int64) error {
	result, err := dao.DB.ExecContext(ctx, `
		UPDATE project.project_managers 
		SET is_deleted = TRUE, updated_by = $1
		WHERE id = $2 AND project_id = $3 AND is_deleted = FALSE
	`, userID, managerID, projectID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"manager_id": managerID,
			"project_id": projectID,
			"error":      err.Error(),
		}).Error("Failed to delete project manager")
		return fmt.Errorf("failed to delete project manager: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("project manager not found")
	}

	return nil
}

// CreateProjectAttachment creates a new project attachment
func (dao *ProjectDao) CreateProjectAttachment(ctx context.Context, projectID int64, request *models.CreateProjectAttachmentRequest, userID int64) (*models.ProjectAttachment, error) {
	var attachmentID int64
	var createdAt, updatedAt time.Time

	query := `
		INSERT INTO project.project_attachments (
			project_id, file_name, file_path, file_size, file_type, attachment_type, uploaded_by, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	err := dao.DB.QueryRowContext(ctx, query,
		projectID, request.FileName, request.FilePath, request.FileSize, request.FileType,
		request.AttachmentType, userID, userID, userID,
	).Scan(&attachmentID, &createdAt, &updatedAt)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"file_name":  request.FileName,
			"error":      err.Error(),
		}).Error("Failed to create project attachment")
		return nil, fmt.Errorf("failed to create project attachment: %w", err)
	}

	return &models.ProjectAttachment{
		ID:             attachmentID,
		ProjectID:      projectID,
		FileName:       request.FileName,
		FilePath:       request.FilePath,
		FileSize:       request.FileSize,
		FileType:       request.FileType,
		AttachmentType: request.AttachmentType,
		UploadedBy:     userID,
		CreatedAt:      createdAt,
		CreatedBy:      userID,
		UpdatedAt:      updatedAt,
		UpdatedBy:      userID,
	}, nil
}

// GetProjectAttachmentsByProject retrieves all attachments for a project
func (dao *ProjectDao) GetProjectAttachmentsByProject(ctx context.Context, projectID int64) ([]models.ProjectAttachment, error) {
	query := `
		SELECT id, project_id, file_name, file_path, file_size, file_type, attachment_type, 
		       uploaded_by, created_at, created_by, updated_at, updated_by
		FROM project.project_attachments
		WHERE project_id = $1 AND is_deleted = FALSE
		ORDER BY created_at DESC
	`

	rows, err := dao.DB.QueryContext(ctx, query, projectID)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"error":      err.Error(),
		}).Error("Failed to query project attachments")
		return nil, fmt.Errorf("failed to query project attachments: %w", err)
	}
	defer rows.Close()

	var attachments []models.ProjectAttachment
	for rows.Next() {
		var attachment models.ProjectAttachment
		err := rows.Scan(
			&attachment.ID, &attachment.ProjectID, &attachment.FileName, &attachment.FilePath,
			&attachment.FileSize, &attachment.FileType, &attachment.AttachmentType, &attachment.UploadedBy,
			&attachment.CreatedAt, &attachment.CreatedBy, &attachment.UpdatedAt, &attachment.UpdatedBy,
		)
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan project attachment row")
			return nil, fmt.Errorf("failed to scan project attachment: %w", err)
		}
		attachments = append(attachments, attachment)
	}

	return attachments, nil
}

// GetProjectAttachmentByID retrieves a specific project attachment by ID
func (dao *ProjectDao) GetProjectAttachmentByID(ctx context.Context, attachmentID, projectID int64) (*models.ProjectAttachment, error) {
	var attachment models.ProjectAttachment
	query := `
		SELECT id, project_id, file_name, file_path, file_size, file_type, attachment_type, 
		       uploaded_by, created_at, created_by, updated_at, updated_by
		FROM project.project_attachments
		WHERE id = $1 AND project_id = $2 AND is_deleted = FALSE
	`

	err := dao.DB.QueryRowContext(ctx, query, attachmentID, projectID).Scan(
		&attachment.ID, &attachment.ProjectID, &attachment.FileName, &attachment.FilePath,
		&attachment.FileSize, &attachment.FileType, &attachment.AttachmentType, &attachment.UploadedBy,
		&attachment.CreatedAt, &attachment.CreatedBy, &attachment.UpdatedAt, &attachment.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project attachment not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"attachment_id": attachmentID,
			"project_id":    projectID,
			"error":         err.Error(),
		}).Error("Failed to get project attachment")
		return nil, fmt.Errorf("failed to get project attachment: %w", err)
	}

	return &attachment, nil
}

// DeleteProjectAttachment deletes a project attachment (soft delete)
func (dao *ProjectDao) DeleteProjectAttachment(ctx context.Context, attachmentID, projectID int64, userID int64) error {
	result, err := dao.DB.ExecContext(ctx, `
		UPDATE project.project_attachments 
		SET is_deleted = TRUE, updated_by = $1
		WHERE id = $2 AND project_id = $3 AND is_deleted = FALSE
	`, userID, attachmentID, projectID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"attachment_id": attachmentID,
			"project_id":    projectID,
			"error":         err.Error(),
		}).Error("Failed to delete project attachment")
		return fmt.Errorf("failed to delete project attachment: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("project attachment not found")
	}

	return nil
}

// AssignUserToProject assigns a user to a project with a specific role
func (dao *ProjectDao) AssignUserToProject(ctx context.Context, projectID int64, request *models.CreateProjectUserRoleRequest, userID int64) (*models.ProjectUserRole, error) {
	var assignmentID int64
	var createdAt, updatedAt time.Time
	
	tradeType := sql.NullString{String: request.TradeType, Valid: request.TradeType != ""}
	
	startDate := sql.NullTime{}
	if request.StartDate != "" {
		if t, err := time.Parse("2006-01-02", request.StartDate); err == nil {
			startDate = sql.NullTime{Time: t, Valid: true}
		}
	}
	
	endDate := sql.NullTime{}
	if request.EndDate != "" {
		if t, err := time.Parse("2006-01-02", request.EndDate); err == nil {
			endDate = sql.NullTime{Time: t, Valid: true}
		}
	}

	query := `
		INSERT INTO project.project_user_roles (
			project_id, user_id, role_id, trade_type, is_primary, start_date, end_date, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	err := dao.DB.QueryRowContext(ctx, query,
		projectID, request.UserID, request.RoleID, tradeType, request.IsPrimary,
		startDate, endDate, userID, userID,
	).Scan(&assignmentID, &createdAt, &updatedAt)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"user_id":    request.UserID,
			"role_id":    request.RoleID,
			"error":      err.Error(),
		}).Error("Failed to assign user to project")
		return nil, fmt.Errorf("failed to assign user to project: %w", err)
	}

	return &models.ProjectUserRole{
		ID:        assignmentID,
		ProjectID: projectID,
		UserID:    request.UserID,
		RoleID:    request.RoleID,
		TradeType: tradeType,
		IsPrimary: request.IsPrimary,
		StartDate: startDate,
		EndDate:   endDate,
		CreatedAt: createdAt,
		CreatedBy: userID,
		UpdatedAt: updatedAt,
		UpdatedBy: userID,
	}, nil
}

// GetProjectUserRoles retrieves all user role assignments for a project
func (dao *ProjectDao) GetProjectUserRoles(ctx context.Context, projectID int64) ([]models.ProjectUserRole, error) {
	query := `
		SELECT id, project_id, user_id, role_id, trade_type, is_primary, start_date, end_date,
		       created_at, created_by, updated_at, updated_by
		FROM project.project_user_roles
		WHERE project_id = $1 AND is_deleted = FALSE
		ORDER BY is_primary DESC, created_at ASC
	`

	rows, err := dao.DB.QueryContext(ctx, query, projectID)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"error":      err.Error(),
		}).Error("Failed to query project user roles")
		return nil, fmt.Errorf("failed to query project user roles: %w", err)
	}
	defer rows.Close()

	var assignments []models.ProjectUserRole
	for rows.Next() {
		var assignment models.ProjectUserRole
		err := rows.Scan(
			&assignment.ID, &assignment.ProjectID, &assignment.UserID, &assignment.RoleID,
			&assignment.TradeType, &assignment.IsPrimary, &assignment.StartDate, &assignment.EndDate,
			&assignment.CreatedAt, &assignment.CreatedBy, &assignment.UpdatedAt, &assignment.UpdatedBy,
		)
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan project user role row")
			return nil, fmt.Errorf("failed to scan project user role: %w", err)
		}
		assignments = append(assignments, assignment)
	}

	return assignments, nil
}

// UpdateProjectUserRole updates an existing project user role assignment
func (dao *ProjectDao) UpdateProjectUserRole(ctx context.Context, assignmentID, projectID int64, request *models.UpdateProjectUserRoleRequest, userID int64) (*models.ProjectUserRole, error) {
	// Build dynamic update query based on provided fields
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if request.RoleID != 0 {
		setParts = append(setParts, fmt.Sprintf("role_id = $%d", argIndex))
		args = append(args, request.RoleID)
		argIndex++
	}

	// Always update updated_by
	setParts = append(setParts, fmt.Sprintf("updated_by = $%d", argIndex))
	args = append(args, userID)
	argIndex++

	// Add WHERE clause parameters
	args = append(args, assignmentID, projectID)
	whereClause := fmt.Sprintf("WHERE id = $%d AND project_id = $%d AND is_deleted = FALSE", argIndex, argIndex+1)

	if len(setParts) == 1 { // Only updated_by was set
		return nil, fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf(`
		UPDATE project.project_user_roles 
		SET %s
		%s
		RETURNING id, project_id, user_id, role_id, trade_type, is_primary, start_date, end_date,
		          created_at, created_by, updated_at, updated_by
	`, 
		setParts[0]+", "+setParts[1:][0], // Join SET parts with commas
		whereClause,
	)

	var assignment models.ProjectUserRole
	err := dao.DB.QueryRowContext(ctx, query, args...).Scan(
		&assignment.ID, &assignment.ProjectID, &assignment.UserID, &assignment.RoleID,
		&assignment.TradeType, &assignment.IsPrimary, &assignment.StartDate, &assignment.EndDate,
		&assignment.CreatedAt, &assignment.CreatedBy, &assignment.UpdatedAt, &assignment.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project user role assignment not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"assignment_id": assignmentID,
			"project_id":    projectID,
			"error":         err.Error(),
		}).Error("Failed to update project user role")
		return nil, fmt.Errorf("failed to update project user role: %w", err)
	}

	return &assignment, nil
}

// RemoveUserFromProject removes a user from a project (soft delete)
func (dao *ProjectDao) RemoveUserFromProject(ctx context.Context, assignmentID, projectID int64, userID int64) error {
	result, err := dao.DB.ExecContext(ctx, `
		UPDATE project.project_user_roles 
		SET is_deleted = TRUE, updated_by = $1
		WHERE id = $2 AND project_id = $3 AND is_deleted = FALSE
	`, userID, assignmentID, projectID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"assignment_id": assignmentID,
			"project_id":    projectID,
			"error":         err.Error(),
		}).Error("Failed to remove user from project")
		return fmt.Errorf("failed to remove user from project: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("project user role assignment not found")
	}

	return nil
}