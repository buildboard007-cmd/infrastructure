# Submittal Management

## Overview

The Submittal Management system provides comprehensive tracking and approval workflow for construction submittals. Submittals are contractor-provided documents (shop drawings, product data, samples, etc.) that require review and approval before fabrication or installation.

**Key Features:**
- Auto-generated submittal numbers with SUB-YYYY-NNN format
- Workflow-driven approval process with multiple phases
- Ball-in-court tracking for responsibility assignment
- Rich metadata including CSI division/section tracking
- Delivery tracking and procurement integration
- Team assignment management
- Statistics and reporting

---

## Database Schema

### Table: `project.submittals`

**Primary Table:** Stores all submittal information including workflow state, assignments, dates, and JSONB fields for complex data.

| Column | Type | Required | Default | Description |
|--------|------|----------|---------|-------------|
| `id` | bigint | Yes | Auto-increment | Primary key |
| `project_id` | bigint | Yes | - | Project ID (FK to project.projects) |
| `org_id` | bigint | No | - | Organization ID |
| `location_id` | bigint | No | - | Location ID |
| `submittal_number` | varchar(50) | Yes | - | Auto-generated (SUB-YYYY-NNN) |
| `package_name` | varchar(200) | No | - | Package or grouping name |
| `csi_division` | varchar(2) | No | - | CSI Division (00-49) |
| `csi_section` | varchar(10) | No | - | CSI Section (e.g., 05 12 00) |
| `title` | varchar(255) | Yes | - | Submittal title |
| `description` | text | No | - | Detailed description |
| `submittal_type` | varchar(50) | Yes | - | Type: shop_drawings, product_data, samples, material_certificates, method_statements, test_reports, other |
| `specification_section` | varchar(50) | No | - | Specification section reference |
| `drawing_reference` | varchar(255) | No | - | Drawing reference |
| `trade_type` | varchar(100) | No | - | Trade: concrete, steel, mechanical, electrical, etc. |
| `priority` | varchar(50) | Yes | 'medium' | Priority: critical, high, medium, low |
| `status` | varchar(50) | Yes | 'draft' | Legacy status field |
| `current_phase` | varchar(50) | No | 'preparation' | Phase: preparation, review, approval, fabrication, delivery, installation, completed |
| `ball_in_court` | varchar(50) | No | 'contractor' | Who's responsible: contractor, architect, engineer, owner, subcontractor, vendor |
| `workflow_status` | varchar(50) | No | 'pending_submission' | Status: pending_submission, under_review, approved, approved_as_noted, revise_resubmit, rejected, for_information_only |
| `revision_number` | integer | Yes | 1 | Revision number |
| `submitted_by` | bigint | Yes | - | Submitting user ID |
| `submitted_company_id` | bigint | No | - | Submitting company ID |
| `reviewed_by` | bigint | No | - | Reviewing user ID |
| `assigned_to` | bigint | No | - | Assigned to user ID |
| `reviewer` | bigint | No | - | Reviewer user ID |
| `approver` | bigint | No | - | Approver user ID |
| `submitted_date` | timestamp | No | - | Date submitted |
| `due_date` | timestamp | No | - | Due date |
| `required_approval_date` | timestamp | No | - | Required approval date |
| `reviewed_date` | timestamp | No | - | Date reviewed |
| `approval_date` | timestamp | No | - | Date approved |
| `fabrication_start_date` | timestamp | No | - | Fabrication start date |
| `installation_date` | timestamp | No | - | Installation date |
| `review_comments` | text | No | - | Review comments |
| `lead_time_days` | integer | No | - | Lead time in days |
| `quantity_submitted` | integer | No | - | Quantity submitted |
| `unit_of_measure` | varchar(20) | No | - | Unit of measure |
| `delivery_tracking` | jsonb | No | {} | Delivery tracking details (vendor, dates, status) |
| `team_assignments` | jsonb | No | {} | Team role assignments |
| `linked_drawings` | jsonb | No | {} | Linked drawing references |
| `submittal_references` | jsonb | No | {} | Related submittals, RFIs, issues |
| `procurement_log` | jsonb | No | {} | Procurement tracking details |
| `approval_actions` | jsonb | No | {} | Approval action history |
| `distribution_list` | jsonb | No | [] | Email distribution list |
| `notification_settings` | jsonb | No | {} | Notification preferences |
| `tags` | jsonb | No | [] | Tags for categorization |
| `custom_fields` | jsonb | No | {} | Custom field values |
| `created_at` | timestamp | Yes | CURRENT_TIMESTAMP | Creation timestamp |
| `created_by` | bigint | Yes | - | Creator user ID |
| `updated_at` | timestamp | Yes | CURRENT_TIMESTAMP | Last update timestamp |
| `updated_by` | bigint | Yes | - | Last updater user ID |
| `is_deleted` | boolean | Yes | false | Soft delete flag |

### Related Tables

**`project.submittal_attachments`** - Attachments for submittals (drawings, specs, samples)
**`project.submittal_history`** - Audit trail of all changes

---

## Data Models

### Core Models

**Location:** `/Users/mayur/git_personal/infrastructure/src/lib/models/submittal.go`

```go
type SubmittalResponse struct {
    ID                    int64                  `json:"id"`
    ProjectID             int64                  `json:"project_id"`
    OrgID                 int64                  `json:"org_id"`
    LocationID            *int64                 `json:"location_id,omitempty"`
    SubmittalNumber       string                 `json:"submittal_number"`
    PackageName           *string                `json:"package_name,omitempty"`
    CSIDivision           *string                `json:"csi_division,omitempty"`
    CSISection            *string                `json:"csi_section,omitempty"`
    Title                 string                 `json:"title"`
    Description           *string                `json:"description,omitempty"`
    SubmittalType         string                 `json:"submittal_type"`
    SpecificationSection  *string                `json:"specification_section,omitempty"`
    Priority              string                 `json:"priority"`
    CurrentPhase          string                 `json:"current_phase"`
    BallInCourt           string                 `json:"ball_in_court"`
    WorkflowStatus        string                 `json:"workflow_status"`
    RevisionNumber        int                    `json:"revision_number"`
    SubmittedBy           int64                  `json:"submitted_by"`
    AssignedTo            *int64                 `json:"assigned_to,omitempty"`
    Reviewer              *int64                 `json:"reviewer,omitempty"`
    Approver              *int64                 `json:"approver,omitempty"`
    SubmittedDate         *time.Time             `json:"submitted_date,omitempty"`
    RequiredApprovalDate  *time.Time             `json:"required_approval_date,omitempty"`
    FabricationStartDate  *time.Time             `json:"fabrication_start_date,omitempty"`
    InstallationDate      *time.Time             `json:"installation_date,omitempty"`
    DeliveryTracking      map[string]interface{} `json:"delivery_tracking"`
    TeamAssignments       map[string]interface{} `json:"team_assignments"`
    LinkedDrawings        map[string]interface{} `json:"linked_drawings"`
    References            map[string]interface{} `json:"submittal_references"`
    ProcurementLog        map[string]interface{} `json:"procurement_log"`
    ApprovalActions       map[string]interface{} `json:"approval_actions"`
    DistributionList      []string               `json:"distribution_list"`
    NotificationSettings  map[string]interface{} `json:"notification_settings"`
    Tags                  []string               `json:"tags"`
    CustomFields          map[string]interface{} `json:"custom_fields"`
    DaysOpen              int                    `json:"days_open"`
    IsOverdue             bool                   `json:"is_overdue"`
    Attachments           []SubmittalAttachment  `json:"attachments"`
    AttachmentCount       int                    `json:"attachment_count"`
    ProjectName           string                 `json:"project_name,omitempty"`
    CreatedAt             time.Time              `json:"created_at"`
    UpdatedAt             time.Time              `json:"updated_at"`
}
```

### Request Models

```go
type CreateSubmittalRequest struct {
    ProjectID              int64                  `json:"project_id" binding:"required"`
    LocationID             *int64                 `json:"location_id,omitempty"`
    SubmittalNumber        string                 `json:"submittal_number,omitempty"` // Auto-generated if empty
    PackageName            *string                `json:"package_name,omitempty"`
    CSIDivision            *string                `json:"csi_division,omitempty"`
    CSISection             *string                `json:"csi_section,omitempty"`
    Title                  string                 `json:"title" binding:"required"`
    Description            *string                `json:"description,omitempty"`
    SubmittalType          string                 `json:"submittal_type" binding:"required"`
    SpecificationSection   *string                `json:"specification_section,omitempty"`
    Priority               *string                `json:"priority,omitempty"`
    CurrentPhase           *string                `json:"current_phase,omitempty"`
    BallInCourt            *string                `json:"ball_in_court,omitempty"`
    WorkflowStatus         *string                `json:"workflow_status,omitempty"`
    AssignedTo             *int64                 `json:"assigned_to,omitempty"`
    Reviewer               *int64                 `json:"reviewer,omitempty"`
    Approver               *int64                 `json:"approver,omitempty"`
    SubmissionDate         *string                `json:"submitted_date,omitempty"`
    RequiredApprovalDate   *string                `json:"required_approval_date,omitempty"`
    FabricationStartDate   *string                `json:"fabrication_start_date,omitempty"`
    InstallationDate       *string                `json:"installation_date,omitempty"`
    DeliveryTracking       map[string]interface{} `json:"delivery_tracking,omitempty"`
    TeamAssignments        map[string]interface{} `json:"team_assignments,omitempty"`
    LinkedDrawings         map[string]interface{} `json:"linked_drawings,omitempty"`
    References             map[string]interface{} `json:"references,omitempty"`
    ProcurementLog         map[string]interface{} `json:"procurement_log,omitempty"`
    ApprovalActions        map[string]interface{} `json:"approval_actions,omitempty"`
    DistributionList       []string               `json:"distribution_list,omitempty"`
    NotificationSettings   map[string]interface{} `json:"notification_settings,omitempty"`
    Tags                   []string               `json:"tags,omitempty"`
    CustomFields           map[string]interface{} `json:"custom_fields,omitempty"`
}

type UpdateSubmittalRequest struct {
    // Supports partial updates - all fields optional except when using action
    PackageName            *string                `json:"package_name,omitempty"`
    CSIDivision            *string                `json:"csi_division,omitempty"`
    CSISection             *string                `json:"csi_section,omitempty"`
    Title                  string                 `json:"title,omitempty"`
    Description            *string                `json:"description,omitempty"`
    Priority               string                 `json:"priority,omitempty"`
    AssignedTo             *int64                 `json:"assigned_to,omitempty"`
    Reviewer               *int64                 `json:"reviewer,omitempty"`
    Approver               *int64                 `json:"approver,omitempty"`
    RequiredApprovalDate   *string                `json:"required_approval_date,omitempty"`
    FabricationStartDate   *string                `json:"fabrication_start_date,omitempty"`
    InstallationDate       *string                `json:"installation_date,omitempty"`
    DeliveryTracking       map[string]interface{} `json:"delivery_tracking,omitempty"`
    TeamAssignments        map[string]interface{} `json:"team_assignments,omitempty"`
    LinkedDrawings         map[string]interface{} `json:"linked_drawings,omitempty"`
    References             map[string]interface{} `json:"references,omitempty"`
    ProcurementLog         map[string]interface{} `json:"procurement_log,omitempty"`
    ApprovalActions        map[string]interface{} `json:"approval_actions,omitempty"`
    DistributionList       []string               `json:"distribution_list,omitempty"`
    Tags                   []string               `json:"tags,omitempty"`
    CustomFields           map[string]interface{} `json:"custom_fields,omitempty"`

    // For workflow actions
    Action       *string `json:"action,omitempty"`
    Comments     *string `json:"comments,omitempty"`
    NextReviewer *int64  `json:"next_reviewer,omitempty"`
}

type SubmittalWorkflowAction struct {
    Action              string  `json:"action" binding:"required"`
    Comments            *string `json:"comments,omitempty"`
    NextReviewer        *int64  `json:"next_reviewer,omitempty"`
    BallInCourtTransfer *string `json:"ball_in_court_transfer,omitempty"`
}
```

### JSONB Field Structures

**delivery_tracking:**
```json
{
    "anticipated_delivery_date": "2024-05-10",
    "order_date": "2024-03-20",
    "delivery_status": "ordered",
    "tracking_number": "TRK-123456",
    "vendor_info": {
        "name": "Steel Fabricators Inc",
        "contact_email": "contact@steelfab.com",
        "contact_phone": "+1-555-0123"
    },
    "expected_arrival": "2024-05-15"
}
```

**team_assignments:**
```json
{
    "lead_architect": 2,
    "lead_engineer": 4,
    "project_manager": 1,
    "contractor_rep": 5,
    "subcontractor_rep": 6,
    "qa_reviewer": 7
}
```

**linked_drawings:**
```json
{
    "drawing_numbers": ["S-101", "S-102", "S-201"],
    "drawing_revisions": ["R2", "R2", "R3"],
    "detail_references": ["DET-1", "DET-2", "DET-5"]
}
```

**submittal_references:**
```json
{
    "specification_sections": ["051200", "051300"],
    "related_submittals": ["SUB-2024-002"],
    "related_rfis": ["RFI-2024-001"],
    "related_issues": ["ISS-2024-005"]
}
```

**procurement_log:**
```json
{
    "item_category": "Structural Steel Components",
    "manufacturer": "Steel Corp International",
    "model_number": "SC-2024-BEAM-A",
    "quantity": 150,
    "unit_cost": 275.00,
    "total_cost": 41250.00,
    "lead_time_weeks": 8,
    "procurement_method": "direct_purchase",
    "budget_line_item": "05-001-STEEL",
    "vendor_po_number": "PO-2024-0156"
}
```

---

## API Endpoints

**Service Location:** `/Users/mayur/git_personal/infrastructure/src/infrastructure-submittal-management/main.go`

### 1. Create Submittal
**POST** `/submittals`

Creates a new submittal with auto-generated submittal number.

**Request Body:**
```json
{
    "project_id": 47,
    "location_id": 22,
    "package_name": "Structural Steel Package - Phase 1",
    "csi_division": "05",
    "csi_section": "05 12 00",
    "title": "Shop Drawings for Steel Beam Connections",
    "description": "Detailed shop drawings for structural steel beam-to-column connections",
    "submittal_type": "shop_drawings",
    "specification_section": "051200",
    "priority": "high",
    "current_phase": "preparation",
    "ball_in_court": "contractor",
    "workflow_status": "pending_submission",
    "assigned_to": 1,
    "reviewer": 2,
    "approver": 3,
    "required_approval_date": "2024-03-15",
    "fabrication_start_date": "2024-04-01",
    "installation_date": "2024-05-15",
    "delivery_tracking": {
        "anticipated_delivery_date": "2024-05-10",
        "delivery_status": "not_ordered",
        "vendor_info": {
            "name": "Steel Fabricators Inc"
        }
    },
    "team_assignments": {
        "lead_architect": 2,
        "lead_engineer": 4,
        "project_manager": 1
    },
    "linked_drawings": {
        "drawing_numbers": ["S-101", "S-102"],
        "drawing_revisions": ["R1", "R1"]
    },
    "references": {
        "specification_sections": ["051200"],
        "related_rfis": ["RFI-2024-001"]
    },
    "procurement_log": {
        "manufacturer": "Steel Corp International",
        "quantity": 150,
        "unit_cost": 275.00
    },
    "distribution_list": [
        "architect@company.com",
        "engineer@company.com"
    ],
    "tags": ["urgent", "structural", "phase1"]
}
```

**Response (201 Created):**
```json
{
    "id": 25,
    "submittal_number": "SUB-2024-025",
    "project_id": 47,
    "package_name": "Structural Steel Package - Phase 1",
    "title": "Shop Drawings for Steel Beam Connections",
    "submittal_type": "shop_drawings",
    "priority": "high",
    "current_phase": "preparation",
    "ball_in_court": "contractor",
    "workflow_status": "pending_submission",
    "revision_number": 1,
    "days_open": 0,
    "is_overdue": false,
    "created_at": "2024-01-15T10:30:00Z"
}
```

### 2. Get Submittals by Project
**GET** `/contexts/project/{projectId}/submittals`

Retrieves submittals with pagination and filtering.

**Query Parameters:**
- `page` (default: 1): Page number
- `limit` (default: 20, max: 100): Items per page
- `sort` (default: created_at): Sort field
- `order` (default: desc): Sort order (asc/desc)
- `status`: Filter by workflow_status
- `priority`: Filter by priority
- `csi_division`: Filter by CSI division
- `ball_in_court`: Filter by ball in court
- `search`: Search in package_name, title, description, submittal_number

**Response (200 OK):**
```json
{
    "submittals": [
        {
            "id": 25,
            "submittal_number": "SUB-2024-025",
            "package_name": "Structural Steel Package - Phase 1",
            "title": "Shop Drawings for Steel Beam Connections",
            "priority": "high",
            "workflow_status": "under_review",
            "ball_in_court": "architect",
            "days_open": 5,
            "is_overdue": false
        }
    ],
    "total_count": 1,
    "page": 1,
    "page_size": 20,
    "has_next": false,
    "has_prev": false
}
```

### 3. Get Single Submittal
**GET** `/submittals/{submittalId}`

Retrieves full submittal details with attachments.

**Response (200 OK):**
Returns complete submittal object with all JSONB fields expanded and attachments list.

### 4. Update Submittal
**PUT** `/submittals/{submittalId}`

Updates submittal information. Supports partial updates.

**Request Body (Partial Update):**
```json
{
    "package_name": "Structural Steel Package - Phase 1 (Updated)",
    "title": "Updated: Shop Drawings for Steel Beam Connections - Rev A",
    "priority": "critical",
    "delivery_tracking": {
        "delivery_status": "ordered",
        "tracking_number": "TRK-123456",
        "order_date": "2024-03-18"
    },
    "tags": ["urgent", "structural", "phase1", "revised"]
}
```

**Response (200 OK):**
Returns updated submittal object.

### 5. Submit for Review (Workflow Action)
**POST** `/submittals/{submittalId}/workflow`

Submits submittal for review, changing workflow status and ball in court.

**Request Body:**
```json
{
    "action": "submit_for_review",
    "comments": "Submittal is ready for architectural review. All contractor comments have been addressed.",
    "next_reviewer": 2,
    "ball_in_court_transfer": "architect"
}
```

**Response (200 OK):**
```json
{
    "id": 25,
    "workflow_status": "under_review",
    "current_phase": "review",
    "ball_in_court": "architect",
    "reviewer": 2
}
```

### 6. Approve (Workflow Action)
**POST** `/submittals/{submittalId}/workflow`

Approves submittal without conditions.

**Request Body:**
```json
{
    "action": "approve",
    "comments": "Submittal approved. Proceed with fabrication.",
    "ball_in_court_transfer": "contractor"
}
```

**New Status:** `approved`, Phase: `fabrication`, Ball in Court: `contractor`

### 7. Approve as Noted (Workflow Action)
**POST** `/submittals/{submittalId}/workflow`

Approves submittal with noted conditions.

**Request Body:**
```json
{
    "action": "approve_as_noted",
    "comments": "Submittal approved with minor revisions noted.",
    "conditions": "1. Verify weld symbols on Detail 3A. 2. Add fireproofing notes. 3. Confirm bolt grades.",
    "ball_in_court_transfer": "contractor"
}
```

**New Status:** `approved_as_noted`, Phase: `fabrication`, Ball in Court: `contractor`

### 8. Revise and Resubmit (Workflow Action)
**POST** `/submittals/{submittalId}/workflow`

Requests revisions to the submittal.

**Request Body:**
```json
{
    "action": "revise_resubmit",
    "comments": "Several issues need to be addressed before approval.",
    "revision_notes": "1. Connection details don't match drawings. 2. Missing seismic bracing details. 3. Update weld specifications.",
    "ball_in_court_transfer": "contractor"
}
```

**New Status:** `revise_resubmit`, Phase: `preparation`, Ball in Court: `contractor`

### 9. Reject (Workflow Action)
**POST** `/submittals/{submittalId}/workflow`

Rejects the submittal.

**Request Body:**
```json
{
    "action": "reject",
    "comments": "Submittal does not meet project requirements. Please resubmit with corrected information.",
    "ball_in_court_transfer": "contractor"
}
```

**New Status:** `rejected`, Phase: `preparation`, Ball in Court: `contractor`

### 10. Mark for Information (Workflow Action)
**POST** `/submittals/{submittalId}/workflow`

Marks submittal as for information only (no approval required).

**Request Body:**
```json
{
    "action": "mark_for_information",
    "comments": "Received for information purposes only."
}
```

**New Status:** `for_information_only`, Phase: `completed`, Ball in Court: `contractor`

### 11. Get Submittal Statistics
**GET** `/contexts/project/{projectId}/submittals/stats`

Returns statistics for submittals in a project.

**Response (200 OK):**
```json
{
    "total": 45,
    "overdue": 3,
    "by_status": {
        "pending_submission": 5,
        "under_review": 12,
        "approved": 20,
        "approved_as_noted": 6,
        "revise_resubmit": 2
    },
    "by_priority": {
        "critical": 3,
        "high": 15,
        "medium": 20,
        "low": 7
    },
    "by_ball_in_court": {
        "contractor": 15,
        "architect": 18,
        "engineer": 8,
        "owner": 4
    },
    "delivery_summary": {}
}
```

### 12. Add Submittal Attachment (Centralized Service)
**POST** `/submittals/{submittalId}/attachments`

**Note:** Submittal attachments are now handled by the centralized attachment management service.

---

## Repository Methods

**Location:** `/Users/mayur/git_personal/infrastructure/src/lib/data/submittal_repository.go`

### Key Methods

```go
// Submittal CRUD
CreateSubmittal(ctx, projectID, userID, orgID, req) (*SubmittalResponse, error)
GetSubmittal(ctx, submittalID) (*SubmittalResponse, error)
GetSubmittalsByProject(ctx, projectID, filters) ([]SubmittalResponse, error)
UpdateSubmittal(ctx, submittalID, userID, orgID, req) (*SubmittalResponse, error)

// Workflow Actions
ExecuteWorkflowAction(ctx, submittalID, userID, action) (*SubmittalResponse, error)

// Statistics
GetSubmittalStats(ctx, projectID) (*SubmittalStats, error)

// Attachments
AddSubmittalAttachment(ctx, attachment) (*SubmittalAttachment, error)
GetSubmittalAttachments(ctx, submittalID) ([]SubmittalAttachment, error)

// History
AddSubmittalHistory(ctx, history) error
```

### Auto-Numbering Logic

Submittals use a project-specific auto-numbering system with the format **SUB-YYYY-NNN**:

```go
func generateSubmittalNumber(ctx, projectID) (string, error) {
    var count int
    query := `SELECT COUNT(*) FROM project.submittals WHERE project_id = $1`
    db.QueryRow(query, projectID).Scan(&count)

    year := time.Now().Year()
    return fmt.Sprintf("SUB-%d-%03d", year, count+1), nil
}
```

**Example:** SUB-2024-001, SUB-2024-002, etc.

**Important:** Numbering is project-specific, with annual reset.

---

## Workflow States and Actions

### Submittal Types

- **shop_drawings** - Fabrication shop drawings
- **product_data** - Product specifications and data sheets
- **samples** - Physical samples for approval
- **material_certificates** - Material test certificates
- **method_statements** - Construction method statements
- **test_reports** - Test and inspection reports
- **other** - Other submittal types

### Priority Levels

- **critical** - Critical path item, blocks work
- **high** - Important, needs timely review
- **medium** - Standard priority (default)
- **low** - Non-critical, informational

### Phases

1. **preparation** - Being prepared by contractor
2. **review** - Under review by design team
3. **approval** - In approval process
4. **fabrication** - Approved, in fabrication
5. **delivery** - Being delivered
6. **installation** - Being installed
7. **completed** - Completed/closed

### Workflow Statuses

1. **pending_submission** - Not yet submitted (default)
2. **under_review** - Submitted and under review
3. **approved** - Approved without conditions
4. **approved_as_noted** - Approved with noted conditions
5. **revise_resubmit** - Requires revision and resubmission
6. **rejected** - Rejected, do not proceed
7. **for_information_only** - For information, no approval needed

### Ball in Court

Tracks who currently has responsibility:
- **contractor** - Contractor's responsibility
- **architect** - Architect reviewing
- **engineer** - Engineer reviewing
- **owner** - Owner reviewing
- **subcontractor** - Subcontractor's responsibility
- **vendor** - Vendor/supplier's responsibility

### Workflow Actions

```go
const (
    WorkflowActionSubmitForReview   = "submit_for_review"
    WorkflowActionApprove           = "approve"
    WorkflowActionApproveAsNoted    = "approve_as_noted"
    WorkflowActionReviseResubmit    = "revise_resubmit"
    WorkflowActionReject            = "reject"
    WorkflowActionMarkForInformation = "mark_for_information"
)

// Workflow Transitions:

submit_for_review:
    Status: pending_submission → under_review
    Phase: preparation → review
    Ball: contractor → architect

approve:
    Status: under_review → approved
    Phase: review → fabrication
    Ball: architect → contractor

approve_as_noted:
    Status: under_review → approved_as_noted
    Phase: review → fabrication
    Ball: architect → contractor

revise_resubmit:
    Status: under_review → revise_resubmit
    Phase: review → preparation
    Ball: architect → contractor

reject:
    Status: under_review → rejected
    Phase: review → preparation
    Ball: architect → contractor

mark_for_information:
    Status: any → for_information_only
    Phase: any → completed
    Ball: any → contractor
```

---

## Postman Collection

**Location:** `/Users/mayur/git_personal/infrastructure/postman/SubmittalManagement.postman_collection.json`

**Available Requests:**
1. Create Submittal
2. Get Submittals by Project (Context Query with filters)
3. Get Single Submittal
4. Update Submittal (Regular Update)
5. Submit for Review (Workflow Action)
6. Approve as Noted (Workflow Action)
7. Revise and Resubmit (Workflow Action)
8. Get Submittal Statistics
9. Add Submittal Attachment

**Environment Variables:**
- `access_token`: JWT ID token
- `project_id`: Project ID (default: 47)
- `location_id`: Location ID (default: 22)
- `submittal_id`: Auto-populated after creation
- `attachment_id`: Auto-populated after attachment upload

**Base URL:** `https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main`

---

## Best Practices

### CSI Division/Section
- Use 2-digit division codes (00-49)
- Use full section format for CSI Section (e.g., "05 12 00")
- CSI standardization helps with reporting and filtering

### Submittal Packages
- Group related submittals using `package_name`
- Examples: "Structural Steel Package - Phase 1", "MEP Equipment - Building A"

### Priority Assignment
- **critical**: Items on critical path, blocking construction
- **high**: Important items with near-term need
- **medium**: Standard submittals (default)
- **low**: Future items, informational

### Date Management
- `required_approval_date`: When approval is needed
- `fabrication_start_date`: When fabrication should start
- `installation_date`: Planned installation date
- System calculates `is_overdue` based on required_approval_date

### Ball in Court Tracking
- Always specify who has current responsibility
- Update when transferring responsibility
- Critical for workflow accountability

### Delivery Tracking
Use `delivery_tracking` JSONB for comprehensive tracking:
```json
{
    "order_date": "2024-03-20",
    "delivery_status": "ordered|shipped|delivered",
    "anticipated_delivery_date": "2024-05-10",
    "tracking_number": "TRK-123456",
    "vendor_info": {
        "name": "Vendor Name",
        "contact": "contact@vendor.com"
    }
}
```

### Team Assignments
Use `team_assignments` for role tracking:
```json
{
    "lead_architect": 2,
    "lead_engineer": 4,
    "project_manager": 1,
    "contractor_rep": 5,
    "qa_reviewer": 7
}
```

### Linked Drawings
Maintain drawing references with revisions:
```json
{
    "drawing_numbers": ["S-101", "S-102"],
    "drawing_revisions": ["R3", "R2"],
    "detail_references": ["DET-1", "DET-5"]
}
```

### Distribution Lists
- Include all stakeholders who need submittal notifications
- Can be updated throughout workflow
- Triggers notifications on status changes

### Tags for Organization
Use tags for custom categorization:
- `["urgent", "structural", "phase1"]`
- `["long-lead", "critical-path"]`
- `["owner-supplied", "expedited"]`

---

## Statistics and Reporting

The system provides comprehensive statistics via `/contexts/project/{projectId}/submittals/stats`:

- **Total Count**: Total number of submittals
- **Overdue Count**: Submittals past required_approval_date
- **By Status**: Breakdown by workflow_status
- **By Priority**: Distribution across priority levels
- **By Ball in Court**: Who currently has responsibility
- **Delivery Summary**: Delivery status tracking

---

## Related Documentation

- [Project Management](./project-management.md) - Parent project management
- [RFI Management](./rfi-management.md) - Related RFI workflow
- [Issue Management](./issue-management.md) - Issue tracking system
- [Attachment Management](./attachment-management.md) - Centralized attachment service

---

## Implementation Notes

1. **JSONB Fields:** Extensive use of JSONB for flexible complex data structures
2. **Workflow-Driven:** Strong emphasis on workflow states and transitions
3. **Ball in Court:** Unique accountability tracking feature
4. **CSI Integration:** Built-in support for CSI MasterFormat classification
5. **Procurement Integration:** Procurement log for cost tracking
6. **Delivery Tracking:** Built-in delivery and logistics tracking
7. **History Tracking:** All actions logged to submittal_history table
8. **Soft Delete:** Uses is_deleted flag to maintain audit trail
9. **Calculated Fields:** days_open and is_overdue calculated on-the-fly
10. **Revision Tracking:** Revision number automatically managed

---

## Testing

### Test User
- Email: `buildboard007+555@gmail.com`
- Password: `Mayur@1234`

### Sample Test Workflows

**1. Simple Approval Workflow:**
```bash
POST /submittals                              # Create
POST /submittals/{id}/workflow action=submit  # Submit
POST /submittals/{id}/workflow action=approve # Approve
```

**2. Approval with Conditions:**
```bash
POST /submittals
POST /submittals/{id}/workflow action=submit
POST /submittals/{id}/workflow action=approve_as_noted
```

**3. Revision and Resubmit:**
```bash
POST /submittals
POST /submittals/{id}/workflow action=submit
POST /submittals/{id}/workflow action=revise_resubmit
PUT /submittals/{id}                          # Update
POST /submittals/{id}/workflow action=submit  # Resubmit
POST /submittals/{id}/workflow action=approve
```

**4. Information Only:**
```bash
POST /submittals
POST /submittals/{id}/workflow action=mark_for_information
```

---

## Consolidation Benefits

The submittal API has been consolidated to 8-10 core endpoints (from 15+):

1. Reduced API complexity
2. Action-based workflow in single endpoint
3. Context-based queries for flexibility
4. Consistent with other management APIs
5. Better API Gateway resource utilization
6. Simplified client integration
7. Improved maintainability