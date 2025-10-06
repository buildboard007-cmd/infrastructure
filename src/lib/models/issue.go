package models

import (
	"database/sql"
	"time"
)

// Issue represents an issue/punch item in the project
type Issue struct {
	ID          int64         `json:"id"`
	ProjectID   int64         `json:"project_id"`
	IssueNumber string        `json:"issue_number"`
	TemplateID  sql.NullInt64 `json:"template_id,omitempty"`

	// Basic Information
	Title       string `json:"title"`
	Description string `json:"description"`

	// Categorization
	IssueCategory  string         `json:"issue_category"`
	Category       string         `json:"category"`
	DetailCategory sql.NullString `json:"detail_category,omitempty"`
	IssueType      string         `json:"issue_type"` // Existing field

	// Priority & Severity
	Priority string `json:"priority"`
	Severity string `json:"severity"`

	// Root Cause
	RootCause sql.NullString `json:"root_cause,omitempty"`

	// Location Information
	LocationDescription sql.NullString  `json:"location_description,omitempty"`
	LocationBuilding    sql.NullString  `json:"location_building,omitempty"`
	LocationLevel       sql.NullString  `json:"location_level,omitempty"`
	LocationRoom        sql.NullString  `json:"location_room,omitempty"`
	LocationX           sql.NullFloat64 `json:"location_x,omitempty"`
	LocationY           sql.NullFloat64 `json:"location_y,omitempty"`

	// Legacy location fields (keeping for compatibility)
	RoomArea   sql.NullString `json:"room_area,omitempty"`
	FloorLevel sql.NullString `json:"floor_level,omitempty"`

	// Trade & Assignment
	Discipline        sql.NullString `json:"discipline,omitempty"`
	TradeType         sql.NullString `json:"trade_type,omitempty"`
	ReportedBy        int64          `json:"reported_by"`
	AssignedTo        sql.NullInt64  `json:"assigned_to,omitempty"`
	AssignedCompanyID sql.NullInt64  `json:"assigned_company_id,omitempty"`

	// References
	DrawingReference sql.NullString `json:"drawing_reference,omitempty"`
	SpecificationRef sql.NullString `json:"specification_reference,omitempty"`

	// Timeline
	DueDate    *time.Time `json:"due_date,omitempty"`
	ClosedDate *time.Time `json:"closed_date,omitempty"`

	// Distribution
	DistributionList []string `json:"distribution_list,omitempty"`

	// Status
	Status string `json:"status"`

	// Cost Impact
	CostToFix sql.NullFloat64 `json:"cost_to_fix,omitempty"`

	// GPS Coordinates (existing)
	Latitude  sql.NullFloat64 `json:"latitude,omitempty"`
	Longitude sql.NullFloat64 `json:"longitude,omitempty"`

	// Audit fields
	CreatedAt time.Time `json:"created_at"`
	CreatedBy int64     `json:"created_by"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy int64     `json:"updated_by"`
}

// IssueLocationInfo represents location details within an issue
type IssueLocationInfo struct {
	Description    string          `json:"description" binding:"required"`
	Building       string          `json:"building,omitempty"`
	Level          string          `json:"level,omitempty"`
	Room           string          `json:"room,omitempty"`
	Zone           string          `json:"zone,omitempty"`
	Area           string          `json:"area,omitempty"`
	Coordinates    *Coordinates    `json:"coordinates,omitempty"`
	GPSCoordinates *GPSCoordinates `json:"gps_coordinates,omitempty"`
}

// Coordinates represents x,y coordinates on a drawing or floor plan
type Coordinates struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	DrawingID string  `json:"drawing_id,omitempty"`
}

// GPSCoordinates represents GPS latitude/longitude coordinates
type GPSCoordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// IssueRequest represents the unified request structure matching UI expectations
type IssueRequest struct {
	// Project Context (from path parameter and JWT)
	ProjectID  int64 `json:"project_id,omitempty"`  // Set from path parameter
	LocationID int64 `json:"location_id,omitempty"` // Optional

	// Issue Template and Category
	IssueCategory  string `json:"issue_category" binding:"required"`
	Category       string `json:"category" binding:"required"`
	DetailCategory string `json:"detail_category,omitempty"`

	// Basic Information
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`

	// Classification
	Priority  string `json:"priority" binding:"required,oneof=critical high medium low planned"`
	Severity  string `json:"severity,omitempty" binding:"omitempty,oneof=blocking major minor cosmetic"`
	RootCause string `json:"root_cause,omitempty"`

	// Location Details (nested object)
	Location IssueLocationInfo `json:"location" binding:"required"`

	// Trade/Discipline Information
	Discipline string `json:"discipline,omitempty"`
	Trade      string `json:"trade,omitempty"`

	// Assignment and Timeline
	AssignedTo int64  `json:"assigned_to" binding:"required"`
	DueDate    string `json:"due_date" binding:"required"` // ISO date string

	// Distribution
	DistributionList []string `json:"distribution_list,omitempty"`

	// Attachments
	Attachments []IssueAttachment `json:"attachments,omitempty"`

	// Additional metadata
	Tags           []string               `json:"tags,omitempty"`
	CostImpact     *float64               `json:"cost_impact,omitempty"`
	ScheduleImpact *int                   `json:"schedule_impact,omitempty"` // Days
	QualityImpact  string                 `json:"quality_impact,omitempty" binding:"omitempty,oneof=low medium high"`
	CustomFields   map[string]interface{} `json:"custom_fields,omitempty"`

	// Status (for updates only)
	Status string `json:"status,omitempty" binding:"omitempty,oneof=open in_progress ready_for_review closed rejected on_hold"`
}

// CreateIssueRequest uses the unified structure
type CreateIssueRequest IssueRequest

// UpdateIssueRequest uses the same unified structure
type UpdateIssueRequest IssueRequest

// IssueResponse represents the clean response for an issue (without sql.Null* types)
type IssueResponse struct {
	// Core fields
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"project_id"`
	IssueNumber string `json:"issue_number"`
	TemplateID  *int64 `json:"template_id,omitempty"`

	// Basic Information
	Title       string `json:"title"`
	Description string `json:"description"`

	// Categorization
	IssueCategory  string `json:"issue_category,omitempty"`
	Category       string `json:"category,omitempty"`
	DetailCategory string `json:"detail_category,omitempty"`
	IssueType      string `json:"issue_type"`

	// Priority & Severity
	Priority string `json:"priority"`
	Severity string `json:"severity"`

	// Root Cause
	RootCause string `json:"root_cause,omitempty"`

	// Location Information
	LocationDescription string   `json:"location_description,omitempty"`
	LocationBuilding    string   `json:"location_building,omitempty"`
	LocationLevel       string   `json:"location_level,omitempty"`
	LocationRoom        string   `json:"location_room,omitempty"`
	LocationX           *float64 `json:"location_x,omitempty"`
	LocationY           *float64 `json:"location_y,omitempty"`

	// Legacy location fields
	RoomArea   string `json:"room_area,omitempty"`
	FloorLevel string `json:"floor_level,omitempty"`

	// Trade & Assignment
	Discipline        string `json:"discipline,omitempty"`
	TradeType         string `json:"trade_type,omitempty"`
	ReportedBy        int64  `json:"reported_by"`
	AssignedTo        *int64 `json:"assigned_to,omitempty"`
	AssignedCompanyID *int64 `json:"assigned_company_id,omitempty"`

	// References
	DrawingReference string `json:"drawing_reference,omitempty"`
	SpecificationRef string `json:"specification_reference,omitempty"`

	// Timeline
	DueDate    *time.Time `json:"due_date,omitempty"`
	ClosedDate *time.Time `json:"closed_date,omitempty"`

	// Distribution
	DistributionList []string `json:"distribution_list,omitempty"`

	// Status
	Status string `json:"status"`

	// Cost Impact
	CostToFix *float64 `json:"cost_to_fix,omitempty"`

	// GPS Coordinates
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`

	// Audit fields
	CreatedAt time.Time `json:"created_at"`
	CreatedBy int64     `json:"created_by"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy int64     `json:"updated_by"`

	// Additional computed fields
	ProjectName         string `json:"project_name,omitempty"`
	ReportedByName      string `json:"reported_by_name,omitempty"`
	AssignedToName      string `json:"assigned_to_name,omitempty"`
	AssignedCompanyName string `json:"assigned_company_name,omitempty"`
	DaysOpen            int    `json:"days_open,omitempty"`
	IsOverdue           bool   `json:"is_overdue"`

	// Attachments
	Attachments []IssueAttachment `json:"attachments,omitempty"`
}

// IssueListResponse represents the response for listing issues
type IssueListResponse struct {
	Issues   []IssueResponse `json:"issues"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

// IssueTemplate represents a reusable template for creating issues
type IssueTemplate struct {
	ID                 int64          `json:"id"`
	OrgID              int64          `json:"org_id"`
	Name               string         `json:"name"`
	Category           string         `json:"category"`
	DetailCategory     sql.NullString `json:"detail_category,omitempty"`
	DefaultPriority    sql.NullString `json:"default_priority,omitempty"`
	DefaultSeverity    sql.NullString `json:"default_severity,omitempty"`
	DefaultDescription sql.NullString `json:"default_description,omitempty"`
	IsActive           bool           `json:"is_active"`
	CreatedAt          time.Time      `json:"created_at"`
	CreatedBy          int64          `json:"created_by"`
	UpdatedAt          time.Time      `json:"updated_at"`
	UpdatedBy          int64          `json:"updated_by"`
}

// Issue Status Constants
const (
	IssueStatusOpen           = "open"
	IssueStatusInProgress     = "in_progress"
	IssueStatusReadyForReview = "ready_for_review"
	IssueStatusClosed         = "closed"
	IssueStatusRejected       = "rejected"
	IssueStatusOnHold         = "on_hold"
)

// Issue Priority Constants
const (
	IssuePriorityCritical = "critical"
	IssuePriorityHigh     = "high"
	IssuePriorityMedium   = "medium"
	IssuePriorityLow      = "low"
	IssuePriorityPlanned  = "planned"
)

// Issue Severity Constants
const (
	IssueSeverityBlocking = "blocking"
	IssueSeverityMajor    = "major"
	IssueSeverityMinor    = "minor"
	IssueSeverityCosmetic = "cosmetic"
)

// IssueAttachment represents a file attached to an issue
type IssueAttachment struct {
	ID             int64     `json:"id"`
	IssueID        int64     `json:"issue_id"`
	FileName       string    `json:"file_name"`
	FilePath       string    `json:"file_path"`
	FileSize       *int64    `json:"file_size,omitempty"`
	FileType       *string   `json:"file_type,omitempty"`
	AttachmentType string    `json:"attachment_type"`
	UploadedBy     int64     `json:"uploaded_by"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      int64     `json:"created_by"`
	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedBy      int64     `json:"updated_by"`
	IsDeleted      bool      `json:"is_deleted"`
}
