package models

import (
	"time"
)

// RFI represents a Request for Information
type RFI struct {
	ID                      int64          `json:"id"`
	ProjectID               int64          `json:"project_id"`
	OrgID                   int64          `json:"org_id"`
	LocationID              *int64         `json:"location_id,omitempty"`
	RFINumber               string         `json:"rfi_number"`
	Subject                 string         `json:"subject"`
	Question                string         `json:"question"`
	Description             *string `json:"description,omitempty"`
	Category                *string `json:"category,omitempty"`
	Discipline              *string `json:"discipline,omitempty"`
	TradeType               *string `json:"trade_type,omitempty"`
	ProjectPhase            *string `json:"project_phase,omitempty"`
	Priority                string         `json:"priority"`
	Status                  string         `json:"status"`
	SubmittedBy             int64          `json:"submitted_by"`
	AssignedTo              *int64         `json:"assigned_to,omitempty"`
	ReviewerEmail           *string `json:"reviewer_email,omitempty"`
	ApproverEmail           *string `json:"approver_email,omitempty"`
	ResponseBy              *int64         `json:"response_by,omitempty"`
	CCList                  []string       `json:"cc_list,omitempty"`
	DistributionList        []string       `json:"distribution_list,omitempty"`
	SubmittedDate           *time.Time     `json:"submitted_date,omitempty"`
	DueDate                 *time.Time     `json:"due_date,omitempty"`
	ResponseDate            *time.Time     `json:"response_date,omitempty"`
	ClosedDate              *time.Time     `json:"closed_date,omitempty"`
	Response                *string `json:"response,omitempty"`
	ResponseStatus          *string `json:"response_status,omitempty"`
	CostImpact              bool           `json:"cost_impact"`
	ScheduleImpact          bool           `json:"schedule_impact"`
	CostImpactAmount        *float64       `json:"cost_impact_amount,omitempty"`
	ScheduleImpactDays      *int           `json:"schedule_impact_days,omitempty"`
	CostImpactDetails       *string `json:"cost_impact_details,omitempty"`
	ScheduleImpactDetails   *string `json:"schedule_impact_details,omitempty"`
	LocationDescription     *string `json:"location_description,omitempty"`
	DrawingReferences       *string `json:"drawing_references,omitempty"`
	SpecificationReferences *string `json:"specification_references,omitempty"`
	RelatedSubmittals       *string `json:"related_submittals,omitempty"`
	RelatedChangeEvents     *string `json:"related_change_events,omitempty"`
	RelatedRFIs             []string       `json:"related_rfis,omitempty"`
	WorkflowType            string         `json:"workflow_type"`
	RequiresApproval        bool           `json:"requires_approval"`
	ApprovalStatus          *string `json:"approval_status,omitempty"`
	ApprovedBy              *int64         `json:"approved_by,omitempty"`
	ApprovalDate            *time.Time     `json:"approval_date,omitempty"`
	ApprovalComments        *string `json:"approval_comments,omitempty"`
	UrgencyJustification    *string `json:"urgency_justification,omitempty"`
	BusinessJustification   *string `json:"business_justification,omitempty"`
	DaysOpen                int            `json:"days_open,omitempty"`
	IsOverdue               bool           `json:"is_overdue"`
	CreatedAt               time.Time      `json:"created_at"`
	CreatedBy               int64          `json:"created_by"`
	UpdatedAt               time.Time      `json:"updated_at"`
	UpdatedBy               int64          `json:"updated_by"`
	IsDeleted               bool           `json:"is_deleted"`
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

// RFIComment represents a comment on an RFI
type RFIComment struct {
	ID            int64      `json:"id"`
	RFIID         int64      `json:"rfi_id"`
	Comment       string     `json:"comment"`
	CommentType   string     `json:"comment_type"`
	PreviousValue string     `json:"previous_value,omitempty"`
	NewValue      string     `json:"new_value,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	CreatedBy     int64      `json:"created_by"`
	CreatedByName string     `json:"created_by_name,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
	UpdatedBy     int64      `json:"updated_by"`
	IsDeleted     bool       `json:"is_deleted"`
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
	ProjectID  int64  `json:"project_id,omitempty"`  // Set from path parameter
	LocationID *int64 `json:"location_id,omitempty"` // Optional

	// Basic Information (matching UI snake_case)
	RFINumber string `json:"rfi_number,omitempty"` // Auto-generated if not provided
	Subject   string `json:"subject" binding:"required,max=500"`
	Question  string `json:"question" binding:"required"`

	// Classification (matching UI values)
	Priority string `json:"priority" binding:"required,oneof=LOW MEDIUM HIGH URGENT"`
	Category string `json:"category" binding:"required,oneof=DESIGN SPECIFICATION SCHEDULE COORDINATION GENERAL SUBMITTAL CHANGE_EVENT"`

	// Scheduling
	DueDate string `json:"due_date,omitempty"` // YYYY-MM-DD format

	// Assignment
	AssignedTo *string `json:"assigned_to,omitempty"` // User ID as string (ball-in-court)

	// Communication
	DistributionList []string `json:"distribution_list,omitempty"`

	// Attachments
	Attachments []string `json:"attachments,omitempty"` // Array of file URLs

	// References (nested object like UI expects)
	References *RFIReferences `json:"references,omitempty"`

	// Status (for updates only)
	Status   string `json:"status,omitempty" binding:"omitempty,oneof=DRAFT SUBMITTED UNDER_REVIEW ANSWERED CLOSED VOID REQUIRES_REVISION"`
	Response string `json:"response,omitempty"` // For answering RFIs

	// Additional fields for comprehensive RFI management
	ResponseAttachments []string `json:"response_attachments,omitempty"`
	Description         string   `json:"description,omitempty"`
	Discipline          string   `json:"discipline,omitempty"`
	TradeType           string   `json:"trade_type,omitempty"`
	ProjectPhase        string   `json:"project_phase,omitempty"`
	CostImpactAmount    *float64 `json:"cost_impact_amount,omitempty"`
	ScheduleImpactDays  *int     `json:"schedule_impact_days,omitempty"`
	LocationDescription string   `json:"location_description,omitempty"`
}

// CreateRFIRequest uses the unified structure
type CreateRFIRequest RFIRequest

// UpdateRFIRequest extends RFIRequest with action-based fields for consolidation
type UpdateRFIRequest struct {
	RFIRequest
	// Action-based update fields (for consolidation)
	Action       *string `json:"action,omitempty"`        // submit, approve, reject, respond
	Notes        string  `json:"notes,omitempty"`         // General notes/comments
	ResponseText string  `json:"response_text,omitempty"` // Response text for respond action
}

// UpdateRFIStatusRequest represents the request to update RFI status
type UpdateRFIStatusRequest struct {
	Status   string `json:"status" validate:"required"`
	Comment  string `json:"comment,omitempty"`
}

// RFIResponse represents the response when returning an RFI
type RFIResponse struct {
	RFI
	ProjectName           string          `json:"project_name,omitempty"`
	LocationName          string          `json:"location_name,omitempty"`
	SubmittedByName       string          `json:"submitted_by_name,omitempty"`
	AssignedToName        string          `json:"assigned_to_name,omitempty"`
	ResponseByName        string          `json:"response_by_name,omitempty"`
	ApprovedByName        string          `json:"approved_by_name,omitempty"`
	Attachments           []RFIAttachment `json:"attachments,omitempty"`
	Comments              []RFIComment    `json:"comments,omitempty"`
	CommentCount          int             `json:"comment_count,omitempty"`
	AttachmentCount       int             `json:"attachment_count,omitempty"`
}

// RFIListResponse represents a list of RFIs
type RFIListResponse struct {
	RFIs       []RFIResponse `json:"rfis"`
	TotalCount int           `json:"total_count"`
	Page       int           `json:"page,omitempty"`
	PageSize   int           `json:"page_size,omitempty"`
}

// SubmitRFIRequest represents the request to submit an RFI for review
type SubmitRFIRequest struct {
	AssignedTo *int64 `json:"assigned_to,omitempty"`
	Comment    string `json:"comment,omitempty"`
}

// RespondToRFIRequest represents the request to respond to an RFI
type RespondToRFIRequest struct {
	Response       string     `json:"response" validate:"required"`
	ResponseStatus string     `json:"response_status,omitempty"`
	ClosedDate     *time.Time `json:"closed_date,omitempty"`
}

// ApproveRFIRequest represents the request to approve an RFI
type ApproveRFIRequest struct {
	ApprovalComments string `json:"approval_comments,omitempty"`
}

// RejectRFIRequest represents the request to reject an RFI
type RejectRFIRequest struct {
	RejectionReason string `json:"rejection_reason" validate:"required"`
}

// RFI Status constants (matching UI expectations)
const (
	RFIStatusDraft           = "DRAFT"
	RFIStatusSubmitted       = "SUBMITTED"
	RFIStatusUnderReview     = "UNDER_REVIEW"
	RFIStatusAnswered        = "ANSWERED"
	RFIStatusClosed          = "CLOSED"
	RFIStatusVoid            = "VOID"
	RFIStatusRequiresRevision = "REQUIRES_REVISION"
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
	RFICommentTypeResponse     = "response"
	RFICommentTypeApproval     = "approval"
	RFICommentTypeRejection    = "rejection"
)

// RFI Workflow Type constants
const (
	RFIWorkflowStandard = "STANDARD"
	RFIWorkflowExpress  = "EXPRESS"
	RFIWorkflowCustom   = "CUSTOM"
)

