package models

import (
	"database/sql"
	"time"
)

// Project represents a construction project based on project.projects table
type Project struct {
	ProjectID                  int64            `json:"project_id"`
	OrgID                      int64            `json:"org_id"`
	LocationID                 int64            `json:"location_id"`
	ProjectNumber              sql.NullString   `json:"project_number,omitempty"`
	Name                       string           `json:"name"`
	Description                sql.NullString   `json:"description,omitempty"`
	ProjectType                string           `json:"project_type"`
	ProjectStage               sql.NullString   `json:"project_stage,omitempty"`
	WorkScope                  sql.NullString   `json:"work_scope,omitempty"`
	ProjectSector              sql.NullString   `json:"project_sector,omitempty"`
	DeliveryMethod             sql.NullString   `json:"delivery_method,omitempty"`
	ProjectPhase               string           `json:"project_phase"`
	StartDate                  sql.NullTime     `json:"start_date,omitempty"`
	PlannedEndDate             sql.NullTime     `json:"planned_end_date,omitempty"`
	ActualStartDate            sql.NullTime     `json:"actual_start_date,omitempty"`
	ActualEndDate              sql.NullTime     `json:"actual_end_date,omitempty"`
	SubstantialCompletionDate  sql.NullTime     `json:"substantial_completion_date,omitempty"`
	ProjectFinishDate          sql.NullTime     `json:"project_finish_date,omitempty"`
	WarrantyStartDate          sql.NullTime     `json:"warranty_start_date,omitempty"`
	WarrantyEndDate            sql.NullTime     `json:"warranty_end_date,omitempty"`
	Budget                     sql.NullFloat64  `json:"budget,omitempty"`
	ContractValue              sql.NullFloat64  `json:"contract_value,omitempty"`
	SquareFootage              sql.NullInt64    `json:"square_footage,omitempty"`
	Address                    sql.NullString   `json:"address,omitempty"`
	City                       sql.NullString   `json:"city,omitempty"`
	State                      sql.NullString   `json:"state,omitempty"`
	ZipCode                    sql.NullString   `json:"zip_code,omitempty"`
	Country                    string           `json:"country"`
	Language                   string           `json:"language"`
	Latitude                   sql.NullFloat64  `json:"latitude,omitempty"`
	Longitude                  sql.NullFloat64  `json:"longitude,omitempty"`
	Status                     string           `json:"status"`
	CreatedAt                  time.Time        `json:"created_at"`
	CreatedBy                  int64            `json:"created_by"`
	UpdatedAt                  time.Time        `json:"updated_at"`
	UpdatedBy                  int64            `json:"updated_by"`
}

// CreateProjectRequest represents the request payload for creating a new project
// Matches the API contract with nested structure
type CreateProjectRequest struct {
	BasicInfo       BasicInfo       `json:"basic_info"`
	ProjectDetails  ProjectDetails  `json:"project_details"`
	Location        LocationInfo    `json:"location"`
	Timeline        Timeline        `json:"timeline"`
	Financial       Financial       `json:"financial"`
	ProjectManager  ProjectManagerInfo  `json:"project_manager"`
	Attachments     Attachments     `json:"attachments,omitempty"`
}

// BasicInfo represents basic project information
type BasicInfo struct {
	Name        string `json:"name" binding:"required,max=255"`
	Description string `json:"description,omitempty" binding:"max=1000"`
	Address     string `json:"address" binding:"required,max=500"`
}

// ProjectDetails represents project-specific details
type ProjectDetails struct {
	ProjectStage    string  `json:"project_stage" binding:"required,oneof=bidding course-of-construction pre-construction post-construction warranty"`
	WorkScope       string  `json:"work_scope" binding:"required,oneof=new renovation restoration maintenance"`
	ProjectSector   string  `json:"project_sector" binding:"required,oneof=commercial residential industrial hospitality healthcare institutional mixed-use civil-infrastructure recreation aviation specialized"`
	DeliveryMethod  string  `json:"delivery_method" binding:"required,oneof=design-build design-bid-build construction-manager-at-risk integrated-project-delivery construction-manager-as-agent public-private-partnership other"`
	SquareFootage   int64   `json:"square_footage,omitempty" binding:"min=0"`
	Language        string  `json:"language,omitempty" binding:"oneof=en es fr de it pt zh ja ko ar"`
	Status          string  `json:"status,omitempty" binding:"oneof=active inactive on-hold completed cancelled"`
}

// LocationInfo represents location details
type LocationInfo struct {
	City    string `json:"city,omitempty" binding:"max=100"`
	State   string `json:"state,omitempty" binding:"max=50"`
	ZipCode string `json:"zip_code,omitempty" binding:"max=20"`
	Country string `json:"country,omitempty" binding:"max=100"`
}

// Timeline represents project timeline
type Timeline struct {
	StartDate                 string `json:"start_date" binding:"required"`
	SubstantialCompletionDate string `json:"substantial_completion_date,omitempty"`
	ProjectFinishDate         string `json:"project_finish_date,omitempty"`
	WarrantyStartDate         string `json:"warranty_start_date,omitempty"`
	WarrantyEndDate           string `json:"warranty_end_date,omitempty"`
}

// Financial represents financial information
type Financial struct {
	Budget float64 `json:"budget,omitempty" binding:"min=0"`
}

// ProjectManagerInfo represents project manager details
type ProjectManagerInfo struct {
	Name          string `json:"name" binding:"required,max=255"`
	Company       string `json:"company" binding:"required,max=255"`
	Role          string `json:"role" binding:"required,oneof=general-contractor owners-representative program-manager consultant architect engineer inspector"`
	Email         string `json:"email" binding:"required,email,max=255"`
	OfficeContact string `json:"office_contact,omitempty" binding:"max=20"`
	MobileContact string `json:"mobile_contact,omitempty" binding:"max=20"`
}

// Attachments represents file attachments
type Attachments struct {
	Logo         *FileAttachment `json:"logo,omitempty"`
	ProjectPhoto *FileAttachment `json:"project_photo,omitempty"`
}

// FileAttachment represents a file attachment
type FileAttachment struct {
	FileName string `json:"file_name"`
	FileData string `json:"file_data"` // Base64 encoded file data
	FileType string `json:"file_type"`
}

// CreateProjectResponse represents the response for project creation
type CreateProjectResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    CreateProjectData      `json:"data,omitempty"`
	Errors  map[string][]string    `json:"errors,omitempty"`
}

// CreateProjectData represents the data returned after project creation
type CreateProjectData struct {
	ProjectID     string    `json:"project_id"`
	ProjectNumber string    `json:"project_number"`
	Name          string    `json:"name"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	CreatedBy     int64     `json:"created_by"`
}

// Legacy CreateProjectRequest for backward compatibility
type LegacyCreateProjectRequest struct {
	LocationID                 int64    `json:"location_id" binding:"required"`
	ProjectNumber              string   `json:"project_number,omitempty"`
	Name                       string   `json:"name" binding:"required,min=2,max=255"`
	Description                string   `json:"description,omitempty"`
	ProjectType                string   `json:"project_type" binding:"required"`
	ProjectStage               string   `json:"project_stage,omitempty"`
	WorkScope                  string   `json:"work_scope,omitempty"`
	ProjectSector              string   `json:"project_sector,omitempty"`
	DeliveryMethod             string   `json:"delivery_method,omitempty"`
	ProjectPhase               string   `json:"project_phase,omitempty"`
	StartDate                  string   `json:"start_date,omitempty"`
	PlannedEndDate             string   `json:"planned_end_date,omitempty"`
	ActualStartDate            string   `json:"actual_start_date,omitempty"`
	ActualEndDate              string   `json:"actual_end_date,omitempty"`
	SubstantialCompletionDate  string   `json:"substantial_completion_date,omitempty"`
	ProjectFinishDate          string   `json:"project_finish_date,omitempty"`
	WarrantyStartDate          string   `json:"warranty_start_date,omitempty"`
	WarrantyEndDate            string   `json:"warranty_end_date,omitempty"`
	Budget                     float64  `json:"budget,omitempty"`
	ContractValue              float64  `json:"contract_value,omitempty"`
	SquareFootage              int64    `json:"square_footage,omitempty"`
	Address                    string   `json:"address,omitempty"`
	City                       string   `json:"city,omitempty"`
	State                      string   `json:"state,omitempty"`
	ZipCode                    string   `json:"zip_code,omitempty"`
	Country                    string   `json:"country,omitempty"`
	Language                   string   `json:"language,omitempty"`
	Latitude                   float64  `json:"latitude,omitempty"`
	Longitude                  float64  `json:"longitude,omitempty"`
	Status                     string   `json:"status,omitempty"`
}

// UpdateProjectRequest represents the request payload for updating an existing project
type UpdateProjectRequest struct {
	LocationID                 int64    `json:"location_id,omitempty"`
	ProjectNumber              string   `json:"project_number,omitempty"`
	Name                       string   `json:"name,omitempty" binding:"omitempty,min=2,max=255"`
	Description                string   `json:"description,omitempty"`
	ProjectType                string   `json:"project_type,omitempty"`
	ProjectStage               string   `json:"project_stage,omitempty"`
	WorkScope                  string   `json:"work_scope,omitempty"`
	ProjectSector              string   `json:"project_sector,omitempty"`
	DeliveryMethod             string   `json:"delivery_method,omitempty"`
	ProjectPhase               string   `json:"project_phase,omitempty"`
	StartDate                  string   `json:"start_date,omitempty"`
	PlannedEndDate             string   `json:"planned_end_date,omitempty"`
	ActualStartDate            string   `json:"actual_start_date,omitempty"`
	ActualEndDate              string   `json:"actual_end_date,omitempty"`
	SubstantialCompletionDate  string   `json:"substantial_completion_date,omitempty"`
	ProjectFinishDate          string   `json:"project_finish_date,omitempty"`
	WarrantyStartDate          string   `json:"warranty_start_date,omitempty"`
	WarrantyEndDate            string   `json:"warranty_end_date,omitempty"`
	Budget                     float64  `json:"budget,omitempty"`
	ContractValue              float64  `json:"contract_value,omitempty"`
	SquareFootage              int64    `json:"square_footage,omitempty"`
	Address                    string   `json:"address,omitempty"`
	City                       string   `json:"city,omitempty"`
	State                      string   `json:"state,omitempty"`
	ZipCode                    string   `json:"zip_code,omitempty"`
	Country                    string   `json:"country,omitempty"`
	Language                   string   `json:"language,omitempty"`
	Latitude                   float64  `json:"latitude,omitempty"`
	Longitude                  float64  `json:"longitude,omitempty"`
	Status                     string   `json:"status,omitempty"`
}

// ProjectListResponse represents the response for listing projects
type ProjectListResponse struct {
	Projects []Project `json:"projects"`
	Total    int       `json:"total"`
}

// ProjectManager represents a project manager based on project.project_managers table
type ProjectManager struct {
	ID            int64          `json:"id"`
	ProjectID     int64          `json:"project_id"`
	Name          string         `json:"name"`
	Company       string         `json:"company"`
	Role          string         `json:"role"`
	Email         string         `json:"email"`
	OfficeContact sql.NullString `json:"office_contact,omitempty"`
	MobileContact sql.NullString `json:"mobile_contact,omitempty"`
	IsPrimary     bool           `json:"is_primary"`
	CreatedAt     time.Time      `json:"created_at"`
	CreatedBy     int64          `json:"created_by"`
	UpdatedAt     time.Time      `json:"updated_at"`
	UpdatedBy     int64          `json:"updated_by"`
}

// CreateProjectManagerRequest represents the request payload for creating a project manager
type CreateProjectManagerRequest struct {
	Name          string `json:"name" binding:"required,min=2,max=255"`
	Company       string `json:"company" binding:"required,min=2,max=255"`
	Role          string `json:"role" binding:"required"`
	Email         string `json:"email" binding:"required,email"`
	OfficeContact string `json:"office_contact,omitempty"`
	MobileContact string `json:"mobile_contact,omitempty"`
	IsPrimary     bool   `json:"is_primary,omitempty"`
}

// UpdateProjectManagerRequest represents the request payload for updating a project manager
type UpdateProjectManagerRequest struct {
	Name          string `json:"name,omitempty" binding:"omitempty,min=2,max=255"`
	Company       string `json:"company,omitempty" binding:"omitempty,min=2,max=255"`
	Role          string `json:"role,omitempty"`
	Email         string `json:"email,omitempty" binding:"omitempty,email"`
	OfficeContact string `json:"office_contact,omitempty"`
	MobileContact string `json:"mobile_contact,omitempty"`
	IsPrimary     bool   `json:"is_primary,omitempty"`
}

// ProjectAttachment represents a project attachment based on project.project_attachments table
type ProjectAttachment struct {
	ID             int64     `json:"id"`
	ProjectID      int64     `json:"project_id"`
	FileName       string    `json:"file_name"`
	FilePath       string    `json:"file_path"`
	FileSize       int64     `json:"file_size,omitempty"`
	FileType       string    `json:"file_type,omitempty"`
	AttachmentType string    `json:"attachment_type"`
	UploadedBy     int64     `json:"uploaded_by"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      int64     `json:"created_by"`
	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedBy      int64     `json:"updated_by"`
}

// CreateProjectAttachmentRequest represents the request payload for creating a project attachment
type CreateProjectAttachmentRequest struct {
	FileName       string `json:"file_name" binding:"required,min=1,max=255"`
	FilePath       string `json:"file_path" binding:"required,min=1,max=500"`
	FileSize       int64  `json:"file_size,omitempty"`
	FileType       string `json:"file_type,omitempty"`
	AttachmentType string `json:"attachment_type" binding:"required"`
}

// ProjectUserRole represents a user's role assignment to a project based on project.project_user_roles table
type ProjectUserRole struct {
	ID        int64          `json:"id"`
	ProjectID int64          `json:"project_id"`
	UserID    int64          `json:"user_id"`
	RoleID    int64          `json:"role_id"`
	TradeType sql.NullString `json:"trade_type,omitempty"`
	IsPrimary bool           `json:"is_primary"`
	StartDate sql.NullTime   `json:"start_date,omitempty"`
	EndDate   sql.NullTime   `json:"end_date,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	CreatedBy int64          `json:"created_by"`
	UpdatedAt time.Time      `json:"updated_at"`
	UpdatedBy int64          `json:"updated_by"`
}

// CreateProjectUserRoleRequest represents the request payload for assigning a user to a project
type CreateProjectUserRoleRequest struct {
	UserID    int64  `json:"user_id" binding:"required"`
	RoleID    int64  `json:"role_id" binding:"required"`
	TradeType string `json:"trade_type,omitempty"`
	IsPrimary bool   `json:"is_primary,omitempty"`
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
}

// UpdateProjectUserRoleRequest represents the request payload for updating a project user role
type UpdateProjectUserRoleRequest struct {
	RoleID    int64  `json:"role_id,omitempty"`
	TradeType string `json:"trade_type,omitempty"`
	IsPrimary bool   `json:"is_primary,omitempty"`
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
}