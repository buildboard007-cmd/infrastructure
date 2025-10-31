package models

import (
	"time"
)

// AssignedUser represents a user assignment with ID and name
type AssignedUser struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// RFI represents a Request for Information
type RFI struct {
	ID                      int64          `json:"id"`
	ProjectID               int64          `json:"project_id"`
	OrgID                   int64          `json:"org_id"`
	LocationID              int64          `json:"location_id"`
	RFINumber               *string        `json:"rfi_number,omitempty"`
	Subject                 string         `json:"subject"`
	Description             string         `json:"description"`
	Category                string         `json:"category"`
	Discipline              *string        `json:"discipline,omitempty"`
	ProjectPhase            *string        `json:"project_phase,omitempty"`
	Priority                string         `json:"priority"`
	Status                  string         `json:"status"`
	ReceivedFrom            *int64         `json:"received_from,omitempty"`
	AssignedToIDs           []int64        `json:"-"` // Internal field for DB storage
	BallInCourt             *int64         `json:"ball_in_court,omitempty"`
	DistributionList        []string       `json:"distribution_list,omitempty"`
	DueDate                 *time.Time     `json:"due_date,omitempty"`
	ClosedDate              *time.Time     `json:"closed_date,omitempty"`
	CostImpact              bool           `json:"cost_impact"`
	ScheduleImpact          bool           `json:"schedule_impact"`
	CostImpactAmount        *float64       `json:"cost_impact_amount,omitempty"`
	ScheduleImpactDays      *int           `json:"schedule_impact_days,omitempty"`
	LocationDescription     *string        `json:"location_description,omitempty"`
	DrawingNumbers          []string       `json:"drawing_numbers,omitempty"`
	SpecificationSections   []string       `json:"specification_sections,omitempty"`
	RelatedRFIs             []string       `json:"related_rfis,omitempty"`
	CreatedAt               time.Time      `json:"created_at"`
	CreatedBy               int64          `json:"created_by"`
	UpdatedAt               time.Time      `json:"updated_at"`
	UpdatedBy               int64          `json:"updated_by"`
	IsDeleted               bool           `json:"-"` // Hidden from JSON response
}

// RFIAttachment represents an attachment for an RFI
type RFIAttachment struct {
	ID             int64      `json:"id"`
	RFIID          int64      `json:"rfi_id"`
	FileName       string     `json:"file_name"`
	FilePath       string     `json:"file_path,omitempty"`
	FileType       string     `json:"file_type,omitempty"`
	FileSize       int64      `json:"file_size,omitempty"`
	Description    string     `json:"description,omitempty"`
	S3Bucket       string     `json:"s3_bucket,omitempty"`
	S3Key          string     `json:"s3_key,omitempty"`
	S3URL          string     `json:"s3_url,omitempty"`
	AttachmentType string     `json:"attachment_type"`
	UploadedBy     int64      `json:"uploaded_by"`
	UploadDate     time.Time  `json:"upload_date"`
	CreatedAt      time.Time  `json:"created_at"`
	CreatedBy      int64      `json:"created_by"`
	UpdatedAt      time.Time  `json:"updated_at"`
	UpdatedBy      int64      `json:"updated_by"`
	IsDeleted      bool       `json:"is_deleted"`
}

// RFICommentAttachment represents a file attached to an RFI comment
type RFICommentAttachment struct {
	ID             int64     `json:"id"`
	CommentID      int64     `json:"comment_id"`
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

// RFIComment represents a comment on an RFI
type RFIComment struct {
	ID            int64                   `json:"id"`
	RFIID         int64                   `json:"rfi_id"`
	Comment       string                  `json:"comment"`
	CommentType   string                  `json:"comment_type"`
	PreviousValue string                  `json:"previous_value,omitempty"`
	NewValue      string                  `json:"new_value,omitempty"`
	Attachments   []RFICommentAttachment  `json:"attachments"`
	CreatedAt     time.Time               `json:"created_at"`
	CreatedBy     int64                   `json:"created_by"`
	CreatedByName string                  `json:"created_by_name,omitempty"`
	UpdatedAt     time.Time               `json:"updated_at"`
	UpdatedBy     int64                   `json:"updated_by"`
	IsDeleted     bool                    `json:"is_deleted"`
}

// CreateRFICommentRequest for adding a comment to an RFI
type CreateRFICommentRequest struct {
	Comment       string  `json:"comment" binding:"required"`
	AttachmentIDs []int64 `json:"attachment_ids,omitempty"`
}

// RFIReferences represents drawing and specification references
type RFIReferences struct {
	DrawingNumbers        []string `json:"drawing_numbers,omitempty"`
	SpecificationSections []string `json:"specification_sections,omitempty"`
	RelatedRFIs           []string `json:"related_rfis,omitempty"`
}

// RFIRequest represents the unified request structure for both create and update operations (UI Compatible)
type RFIRequest struct {
	// Project Context (from path parameter and JWT)
	ProjectID  int64 `json:"project_id,omitempty"` // Set from path parameter
	LocationID int64 `json:"location_id" binding:"required"` // Required

	// Basic Information
	Subject     string `json:"subject" binding:"required,max=500"`
	Description string `json:"description" binding:"required"`

	// Classification
	Priority     string  `json:"priority" binding:"required,oneof=LOW MEDIUM HIGH URGENT"`
	Category     string  `json:"category" binding:"required,oneof=DESIGN SPECIFICATION SCHEDULE COORDINATION GENERAL SUBMITTAL CHANGE_EVENT"`
	Discipline   *string `json:"discipline,omitempty"`
	ProjectPhase *string `json:"project_phase,omitempty"`

	// Scheduling
	DueDate string `json:"due_date,omitempty"` // YYYY-MM-DD format

	// Assignment
	ReceivedFrom *int64  `json:"received_from,omitempty"` // User ID who sent/created this RFI
	AssignedTo   []int64 `json:"assigned_to,omitempty"`   // Array of user IDs assigned to this RFI
	BallInCourt  *int64  `json:"ball_in_court,omitempty"` // User ID who currently needs to take action

	// Communication
	DistributionList []string `json:"distribution_list,omitempty"`

	// Location and References
	LocationDescription   *string  `json:"location_description,omitempty"`
	DrawingNumbers        []string `json:"drawing_numbers,omitempty"`
	SpecificationSections []string `json:"specification_sections,omitempty"`
	RelatedRFIs           []string `json:"related_rfis,omitempty"`

	// Impact Assessment
	CostImpact         bool     `json:"cost_impact"`
	ScheduleImpact     bool     `json:"schedule_impact"`
	CostImpactAmount   *float64 `json:"cost_impact_amount,omitempty"`
	ScheduleImpactDays *int     `json:"schedule_impact_days,omitempty"`

	// Status (for updates only)
	Status string `json:"status,omitempty" binding:"omitempty,oneof=DRAFT OPEN CLOSE"`

	// Attachments
	Attachments []string `json:"attachments,omitempty"` // Array of file URLs
}

// CreateRFIRequest uses the unified structure
type CreateRFIRequest RFIRequest

// UpdateRFIRequest uses the same structure as create
type UpdateRFIRequest RFIRequest

// RFIResponse represents the response when returning an RFI
type RFIResponse struct {
	ID                    int64            `json:"id"`
	ProjectID             int64            `json:"project_id"`
	ProjectName           string           `json:"project_name,omitempty"`
	OrgID                 int64            `json:"org_id"`
	LocationID            int64            `json:"location_id"`
	LocationName          string           `json:"location_name,omitempty"`
	RFINumber             *string          `json:"rfi_number,omitempty"`
	Subject               string           `json:"subject"`
	Description           string           `json:"description"`
	Category              string           `json:"category"`
	Discipline            *string          `json:"discipline,omitempty"`
	ProjectPhase          *string          `json:"project_phase,omitempty"`
	Priority              string           `json:"priority"`
	Status                string           `json:"status"`
	ReceivedFrom          *AssignedUser    `json:"received_from,omitempty"`
	AssignedTo            []AssignedUser   `json:"assigned_to"`
	BallInCourt           *AssignedUser    `json:"ball_in_court,omitempty"`
	DistributionList      []string         `json:"distribution_list,omitempty"`
	DueDate               *time.Time       `json:"due_date,omitempty"`
	ClosedDate            *time.Time       `json:"closed_date,omitempty"`
	CostImpact            bool             `json:"cost_impact"`
	ScheduleImpact        bool             `json:"schedule_impact"`
	CostImpactAmount      *float64         `json:"cost_impact_amount,omitempty"`
	ScheduleImpactDays    *int             `json:"schedule_impact_days,omitempty"`
	LocationDescription   *string          `json:"location_description,omitempty"`
	DrawingNumbers        []string         `json:"drawing_numbers,omitempty"`
	SpecificationSections []string         `json:"specification_sections,omitempty"`
	RelatedRFIs           []string         `json:"related_rfis,omitempty"`
	Attachments           []RFIAttachment  `json:"attachments"`
	Comments              []RFIComment     `json:"comments"`
	CreatedAt             time.Time        `json:"created_at"`
	CreatedBy             AssignedUser     `json:"created_by"`
	UpdatedAt             time.Time        `json:"updated_at"`
	UpdatedBy             AssignedUser     `json:"updated_by"`
}

// RFIListResponse represents a list of RFIs
type RFIListResponse struct {
	RFIs       []RFIResponse `json:"rfis"`
	TotalCount int           `json:"total_count"`
	Page       int           `json:"page,omitempty"`
	PageSize   int           `json:"page_size,omitempty"`
}

// RFI Status constants (matching UI expectations)
const (
	RFIStatusDraft = "DRAFT"
	RFIStatusOpen  = "OPEN"
	RFIStatusClose = "CLOSE"
)

// RFI Priority constants (matching UI expectations)
const (
	RFIPriorityLow    = "LOW"
	RFIPriorityMedium = "MEDIUM"
	RFIPriorityHigh   = "HIGH"
	RFIPriorityUrgent = "URGENT"
)

// RFI Category constants (matching UI expectations)
const (
	RFICategoryDesign        = "DESIGN"
	RFICategorySpecification = "SPECIFICATION"
	RFICategorySchedule      = "SCHEDULE"
	RFICategoryCoordination  = "COORDINATION"
	RFICategoryGeneral       = "GENERAL"
	RFICategorySubmittal     = "SUBMITTAL"
	RFICategoryChangeEvent   = "CHANGE_EVENT"
)

// RFI Comment Type constants
const (
	RFICommentTypeComment      = "comment"
	RFICommentTypeStatusChange = "status_change"
	RFICommentTypeAssignment   = "assignment"
)

