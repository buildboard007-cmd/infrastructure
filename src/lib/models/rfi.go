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
	Description             string         `json:"description,omitempty"`
	Category                string         `json:"category,omitempty"`
	Discipline              string         `json:"discipline,omitempty"`
	TradeType               string         `json:"trade_type,omitempty"`
	ProjectPhase            string         `json:"project_phase,omitempty"`
	Priority                string         `json:"priority"`
	Status                  string         `json:"status"`
	SubmittedBy             int64          `json:"submitted_by"`
	AssignedTo              *int64         `json:"assigned_to,omitempty"`
	ReviewerEmail           string         `json:"reviewer_email,omitempty"`
	ApproverEmail           string         `json:"approver_email,omitempty"`
	ResponseBy              *int64         `json:"response_by,omitempty"`
	CCList                  []string       `json:"cc_list,omitempty"`
	DistributionList        []string       `json:"distribution_list,omitempty"`
	SubmittedDate           *time.Time     `json:"submitted_date,omitempty"`
	DueDate                 *time.Time     `json:"due_date,omitempty"`
	ResponseDate            *time.Time     `json:"response_date,omitempty"`
	ClosedDate              *time.Time     `json:"closed_date,omitempty"`
	Response                string         `json:"response,omitempty"`
	ResponseStatus          string         `json:"response_status,omitempty"`
	CostImpact              bool           `json:"cost_impact"`
	ScheduleImpact          bool           `json:"schedule_impact"`
	CostImpactAmount        *float64       `json:"cost_impact_amount,omitempty"`
	ScheduleImpactDays      int            `json:"schedule_impact_days,omitempty"`
	CostImpactDetails       string         `json:"cost_impact_details,omitempty"`
	ScheduleImpactDetails   string         `json:"schedule_impact_details,omitempty"`
	LocationDescription     string         `json:"location_description,omitempty"`
	DrawingReferences       string         `json:"drawing_references,omitempty"`
	SpecificationReferences string         `json:"specification_references,omitempty"`
	RelatedSubmittals       string         `json:"related_submittals,omitempty"`
	RelatedChangeEvents     string         `json:"related_change_events,omitempty"`
	RelatedRFIs             []string       `json:"related_rfis,omitempty"`
	WorkflowType            string         `json:"workflow_type"`
	RequiresApproval        bool           `json:"requires_approval"`
	ApprovalStatus          string         `json:"approval_status,omitempty"`
	ApprovedBy              *int64         `json:"approved_by,omitempty"`
	ApprovalDate            *time.Time     `json:"approval_date,omitempty"`
	ApprovalComments        string         `json:"approval_comments,omitempty"`
	UrgencyJustification    string         `json:"urgency_justification,omitempty"`
	BusinessJustification   string         `json:"business_justification,omitempty"`
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
	Filename       string     `json:"filename"`
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

// CreateRFIRequest represents the request to create a new RFI
type CreateRFIRequest struct {
	OrgID                   int64          `json:"org_id" validate:"required"`
	ProjectID               int64          `json:"project_id" validate:"required"`
	LocationID              *int64         `json:"location_id,omitempty"`
	Subject                 string         `json:"subject" validate:"required,max=500"`
	Question                string         `json:"question" validate:"required"`
	Description             string         `json:"description,omitempty"`
	Category                string         `json:"category,omitempty"`
	Discipline              string         `json:"discipline,omitempty"`
	TradeType               string         `json:"trade_type,omitempty"`
	ProjectPhase            string         `json:"project_phase,omitempty"`
	Priority                string         `json:"priority,omitempty"`
	ReviewerEmail           string         `json:"reviewer,omitempty"`
	ApproverEmail           string         `json:"approver,omitempty"`
	CCList                  []string       `json:"ccList,omitempty"`
	DistributionList        []string       `json:"distributionList,omitempty"`
	DueDate                 *time.Time     `json:"dueDate,omitempty"`
	CostImpact              string         `json:"costImpact,omitempty"`
	ScheduleImpact          string         `json:"scheduleImpact,omitempty"`
	CostImpactAmount        *float64       `json:"costImpactAmount,omitempty"`
	ScheduleImpactDays      int            `json:"scheduleImpactDays,omitempty"`
	CostImpactDetails       string         `json:"costImpactDetails,omitempty"`
	ScheduleImpactDetails   string         `json:"scheduleImpactDetails,omitempty"`
	Location                string         `json:"location,omitempty"`
	DrawingReferences       string         `json:"drawingReferences,omitempty"`
	SpecificationReferences string         `json:"specificationReferences,omitempty"`
	RelatedSubmittals       string         `json:"relatedSubmittals,omitempty"`
	RelatedChangeEvents     string         `json:"relatedChangeEvents,omitempty"`
	WorkflowType            string         `json:"workflowType,omitempty"`
	RequiresApproval        bool           `json:"requiresApproval"`
	UrgencyJustification    string         `json:"urgencyJustification,omitempty"`
	BusinessJustification   string         `json:"businessJustification,omitempty"`
	RFINumber               string         `json:"rfiNumber,omitempty"`
	Attachments             []RFIAttachment `json:"attachments,omitempty"`
}

// UpdateRFIRequest represents the request to update an RFI
type UpdateRFIRequest struct {
	Subject                 string         `json:"subject,omitempty"`
	Question                string         `json:"question,omitempty"`
	Description             string         `json:"description,omitempty"`
	Category                string         `json:"category,omitempty"`
	Discipline              string         `json:"discipline,omitempty"`
	TradeType               string         `json:"trade_type,omitempty"`
	ProjectPhase            string         `json:"project_phase,omitempty"`
	Priority                string         `json:"priority,omitempty"`
	AssignedTo              *int64         `json:"assigned_to,omitempty"`
	ReviewerEmail           string         `json:"reviewer,omitempty"`
	ApproverEmail           string         `json:"approver,omitempty"`
	CCList                  []string       `json:"ccList,omitempty"`
	DistributionList        []string       `json:"distributionList,omitempty"`
	DueDate                 *time.Time     `json:"dueDate,omitempty"`
	CostImpact              string         `json:"costImpact,omitempty"`
	ScheduleImpact          string         `json:"scheduleImpact,omitempty"`
	CostImpactAmount        *float64       `json:"costImpactAmount,omitempty"`
	ScheduleImpactDays      int            `json:"scheduleImpactDays,omitempty"`
	CostImpactDetails       string         `json:"costImpactDetails,omitempty"`
	ScheduleImpactDetails   string         `json:"scheduleImpactDetails,omitempty"`
	Location                string         `json:"location,omitempty"`
	DrawingReferences       string         `json:"drawingReferences,omitempty"`
	SpecificationReferences string         `json:"specificationReferences,omitempty"`
	RelatedSubmittals       string         `json:"relatedSubmittals,omitempty"`
	RelatedChangeEvents     string         `json:"relatedChangeEvents,omitempty"`
	UrgencyJustification    string         `json:"urgencyJustification,omitempty"`
	BusinessJustification   string         `json:"businessJustification,omitempty"`
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

// RFI Status constants
const (
	RFIStatusDraft      = "draft"
	RFIStatusSubmitted  = "submitted"
	RFIStatusInReview   = "in_review"
	RFIStatusResponded  = "responded"
	RFIStatusClosed     = "closed"
	RFIStatusCancelled  = "cancelled"
	RFIStatusOnHold     = "on_hold"
)

// RFI Priority constants
const (
	RFIPriorityLow      = "low"
	RFIPriorityMedium   = "medium"
	RFIPriorityHigh     = "high"
	RFIPriorityCritical = "critical"
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