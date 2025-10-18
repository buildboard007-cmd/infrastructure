package models

import (
	"time"
)

// Submittal represents a construction submittal
type Submittal struct {
	ID                    int64                  `json:"id"`
	ProjectID             int64                  `json:"project_id"`
	OrgID                 *int64                 `json:"org_id,omitempty"`
	LocationID            *int64                 `json:"location_id,omitempty"`
	SubmittalNumber       string                 `json:"submittal_number"`
	PackageName           *string                `json:"package_name,omitempty"`
	CSIDivision           *string                `json:"csi_division,omitempty"`
	CSISection            *string                `json:"csi_section,omitempty"`
	Title                 string                 `json:"title"`
	Description           *string                `json:"description,omitempty"`
	SubmittalType         string                 `json:"submittal_type"`
	SpecificationSection  *string                `json:"specification_section,omitempty"`
	DrawingReference      *string                `json:"drawing_reference,omitempty"`
	TradeType             *string                `json:"trade_type,omitempty"`
	Priority              string                 `json:"priority"`
	Status                string                 `json:"status"`
	CurrentPhase          string                 `json:"current_phase"`
	BallInCourt           string                 `json:"ball_in_court"`
	WorkflowStatus        string                 `json:"workflow_status"`
	RevisionNumber        int                    `json:"revision_number"`
	SubmittedBy           int64                  `json:"submitted_by"`
	SubmittedCompanyID    *int64                 `json:"submitted_company_id,omitempty"`
	ReviewedBy            *int64                 `json:"reviewed_by,omitempty"`
	AssignedTo            *int64                 `json:"assigned_to,omitempty"`
	Reviewer              *int64                 `json:"reviewer,omitempty"`
	Approver              *int64                 `json:"approver,omitempty"`
	SubmittedDate         *time.Time             `json:"submitted_date,omitempty"`
	DueDate               *time.Time             `json:"due_date,omitempty"`
	RequiredApprovalDate  *time.Time             `json:"required_approval_date,omitempty"`
	ReviewedDate          *time.Time             `json:"reviewed_date,omitempty"`
	ApprovalDate          *time.Time             `json:"approval_date,omitempty"`
	FabricationStartDate  *time.Time             `json:"fabrication_start_date,omitempty"`
	InstallationDate      *time.Time             `json:"installation_date,omitempty"`
	ReviewComments        *string                `json:"review_comments,omitempty"`
	LeadTimeDays          *int                   `json:"lead_time_days,omitempty"`
	QuantitySubmitted     *int                   `json:"quantity_submitted,omitempty"`
	UnitOfMeasure         *string                `json:"unit_of_measure,omitempty"`
	DeliveryTracking      map[string]interface{} `json:"delivery_tracking,omitempty"`
	TeamAssignments       map[string]interface{} `json:"team_assignments,omitempty"`
	LinkedDrawings        map[string]interface{} `json:"linked_drawings,omitempty"`
	References            map[string]interface{} `json:"references,omitempty"`
	ProcurementLog        map[string]interface{} `json:"procurement_log,omitempty"`
	ApprovalActions       map[string]interface{} `json:"approval_actions,omitempty"`
	DistributionList      []string               `json:"distribution_list,omitempty"`
	NotificationSettings  map[string]interface{} `json:"notification_settings,omitempty"`
	Tags                  []string               `json:"tags,omitempty"`
	CustomFields          map[string]interface{} `json:"custom_fields,omitempty"`
	CreatedAt             time.Time              `json:"created_at"`
	CreatedBy             int64                  `json:"created_by"`
	UpdatedAt             time.Time              `json:"updated_at"`
	UpdatedBy             int64                  `json:"updated_by"`
	IsDeleted             bool                   `json:"is_deleted"`
}

// SubmittalAttachment represents a file attached to a submittal
type SubmittalAttachment struct {
	ID             int64     `json:"id"`
	SubmittalID    int64     `json:"submittal_id"`
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

// SubmittalReview represents a review of a submittal
type SubmittalReview struct {
	ID             int64      `json:"id"`
	SubmittalID    int64      `json:"submittal_id"`
	RevisionNumber int        `json:"revision_number"`
	ReviewerID     int64      `json:"reviewer_id"`
	ReviewStatus   string     `json:"review_status"`
	ReviewComments *string    `json:"review_comments,omitempty"`
	ReviewDate     *time.Time `json:"review_date,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	CreatedBy      int64      `json:"created_by"`
	UpdatedAt      time.Time  `json:"updated_at"`
	UpdatedBy      int64      `json:"updated_by"`
	IsDeleted      bool       `json:"is_deleted"`
}

// SubmittalHistory represents audit trail for submittals
type SubmittalHistory struct {
	ID          int64     `json:"id"`
	SubmittalID int64     `json:"submittal_id"`
	Action      string    `json:"action"`
	FieldName   *string   `json:"field_name,omitempty"`
	OldValue    *string   `json:"old_value,omitempty"`
	NewValue    *string   `json:"new_value,omitempty"`
	Comment     *string   `json:"comment,omitempty"`
	CreatedBy   int64     `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// SubmittalRequest represents the unified request structure for create/update operations
type SubmittalRequest struct {
	// Project Context (from path parameter and JWT)
	ProjectID    int64  `json:"project_id,omitempty"`    // Set from path parameter
	LocationID   *int64 `json:"location_id,omitempty"`   // Optional

	// Basic Information
	SubmittalNumber       string  `json:"submittal_number,omitempty"`   // Auto-generated if not provided
	PackageName           *string `json:"package_name,omitempty"`
	CSIDivision           *string `json:"csi_division,omitempty"`
	CSISection            *string `json:"csi_section,omitempty"`
	Title                 string  `json:"title" binding:"required,max=255"`
	Description           *string `json:"description,omitempty"`
	SubmittalType         string  `json:"submittal_type" binding:"required"`
	SpecificationSection  *string `json:"specification_section,omitempty"`

	// Classification
	Priority              string  `json:"priority" binding:"required,oneof=low medium high urgent"`
	CurrentPhase          *string `json:"current_phase,omitempty"`
	BallInCourt           *string `json:"ball_in_court,omitempty"`
	WorkflowStatus        *string `json:"workflow_status,omitempty"`

	// Assignment
	AssignedTo            *int64  `json:"assigned_to,omitempty"`
	Reviewer              *int64  `json:"reviewer,omitempty"`
	Approver              *int64  `json:"approver,omitempty"`

	// Scheduling (dates as strings in YYYY-MM-DD format)
	SubmissionDate        *string `json:"submission_date,omitempty"`
	RequiredApprovalDate  *string `json:"required_approval_date,omitempty"`
	FabricationStartDate  *string `json:"fabrication_start_date,omitempty"`
	InstallationDate      *string `json:"installation_date,omitempty"`

	// JSON Fields
	DeliveryTracking      map[string]interface{} `json:"delivery_tracking,omitempty"`
	TeamAssignments       map[string]interface{} `json:"team_assignments,omitempty"`
	LinkedDrawings        map[string]interface{} `json:"linked_drawings,omitempty"`
	References            map[string]interface{} `json:"references,omitempty"`
	ProcurementLog        map[string]interface{} `json:"procurement_log,omitempty"`
	ApprovalActions       map[string]interface{} `json:"approval_actions,omitempty"`
	DistributionList      []string               `json:"distribution_list,omitempty"`
	NotificationSettings  map[string]interface{} `json:"notification_settings,omitempty"`
	Tags                  []string               `json:"tags,omitempty"`
	CustomFields          map[string]interface{} `json:"custom_fields,omitempty"`

	// File handling
	Attachments           []string `json:"attachments,omitempty"`
}

// CreateSubmittalRequest uses the unified structure
type CreateSubmittalRequest SubmittalRequest

// UpdateSubmittalRequest extends SubmittalRequest with action-based fields for consolidation
type UpdateSubmittalRequest struct {
	SubmittalRequest
	// Action-based workflow updates (for consolidation)
	Action        *string `json:"action,omitempty"`         // submit_for_review, approve, approve_as_noted, revise_resubmit, reject, mark_for_information
	Comments      *string `json:"comments,omitempty"`       // Action comments
	Conditions    *string `json:"conditions,omitempty"`     // Conditions of approval
	RevisionNotes *string `json:"revision_notes,omitempty"` // Required revisions
	NextReviewer  *int64  `json:"next_reviewer,omitempty"`  // User ID for next reviewer
}

// SubmittalWorkflowAction represents a workflow action request
type SubmittalWorkflowAction struct {
	Action               string  `json:"action"`                          // Required action
	Comments             *string `json:"comments,omitempty"`              // Action comments
	Conditions           *string `json:"conditions,omitempty"`            // Conditions of approval
	RevisionNotes        *string `json:"revision_notes,omitempty"`        // Required revisions
	NextReviewer         *int64  `json:"next_reviewer,omitempty"`         // User ID
	BallInCourtTransfer  *string `json:"ball_in_court_transfer,omitempty"` // Transfer responsibility
}

// SubmittalResponse represents the enhanced response when returning a submittal
type SubmittalResponse struct {
	Submittal
	ProjectName           string               `json:"project_name,omitempty"`
	LocationName          string               `json:"location_name,omitempty"`
	SubmittedByName       string               `json:"submitted_by_name,omitempty"`
	AssignedToName        string               `json:"assigned_to_name,omitempty"`
	ReviewerName          string               `json:"reviewer_name,omitempty"`
	ApproverName          string               `json:"approver_name,omitempty"`
	Attachments           []SubmittalAttachment `json:"attachments"`
	Reviews               []SubmittalReview    `json:"reviews,omitempty"`
	AttachmentCount       int                  `json:"attachment_count,omitempty"`
	ReviewCount           int                  `json:"review_count,omitempty"`
	DaysOpen              int                  `json:"days_open,omitempty"`
	IsOverdue             bool                 `json:"is_overdue"`
	CreatedByName         string               `json:"created_by_name,omitempty"`
	LastModifiedByName    string               `json:"last_modified_by_name,omitempty"`
}

// SubmittalListResponse represents a paginated list of submittals
type SubmittalListResponse struct {
	Submittals []SubmittalResponse `json:"submittals"`
	TotalCount int                 `json:"total_count"`
	Page       int                 `json:"page,omitempty"`
	PageSize   int                 `json:"page_size,omitempty"`
	HasNext    bool                `json:"has_next"`
	HasPrev    bool                `json:"has_previous"`
}

// SubmittalStats represents submittal statistics
type SubmittalStats struct {
	Total           int            `json:"total"`
	ByStatus        map[string]int `json:"by_status"`
	ByPriority      map[string]int `json:"by_priority"`
	ByBallInCourt   map[string]int `json:"by_ball_in_court"`
	Overdue         int            `json:"overdue"`
	DeliverySummary map[string]int `json:"delivery_summary"`
}

// Submittal Status constants
const (
	SubmittalStatusDraft               = "draft"
	SubmittalStatusPendingSubmission   = "pending_submission"
	SubmittalStatusUnderReview         = "under_review"
	SubmittalStatusApproved            = "approved"
	SubmittalStatusApprovedAsNoted     = "approved_as_noted"
	SubmittalStatusReviseResubmit      = "revise_resubmit"
	SubmittalStatusRejected            = "rejected"
	SubmittalStatusForInformationOnly  = "for_information_only"
)

// Submittal Type constants
const (
	SubmittalTypeShopDrawings         = "shop_drawings"
	SubmittalTypeProductData          = "product_data"
	SubmittalTypeSamples              = "samples"
	SubmittalTypeMaterialCertificates = "material_certificates"
	SubmittalTypeMethodStatements     = "method_statements"
	SubmittalTypeTestReports          = "test_reports"
	SubmittalTypeOther                = "other"
)

// Submittal Priority constants
const (
	SubmittalPriorityCritical = "critical"
	SubmittalPriorityHigh     = "high"
	SubmittalPriorityMedium   = "medium"
	SubmittalPriorityLow      = "low"
)

// Submittal Phase constants
const (
	SubmittalPhasePreparation  = "preparation"
	SubmittalPhaseReview       = "review"
	SubmittalPhaseApproval     = "approval"
	SubmittalPhaseFabrication  = "fabrication"
	SubmittalPhaseDelivery     = "delivery"
	SubmittalPhaseInstallation = "installation"
	SubmittalPhaseCompleted    = "completed"
)

// Ball In Court constants
const (
	BallInCourtContractor    = "contractor"
	BallInCourtArchitect     = "architect"
	BallInCourtEngineer      = "engineer"
	BallInCourtOwner         = "owner"
	BallInCourtSubcontractor = "subcontractor"
	BallInCourtVendor        = "vendor"
)

// Workflow Actions
const (
	WorkflowActionSubmitForReview    = "submit_for_review"
	WorkflowActionApprove            = "approve"
	WorkflowActionApproveAsNoted     = "approve_as_noted"
	WorkflowActionReviseResubmit     = "revise_resubmit"
	WorkflowActionReject             = "reject"
	WorkflowActionMarkForInformation = "mark_for_information"
)