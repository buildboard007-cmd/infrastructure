# RFI Management (Request for Information)

## Overview

The RFI Management system provides a comprehensive workflow for handling construction information requests. RFIs are used when clarification is needed on drawings, specifications, or other project documentation during construction.

**Key Features:**
- Auto-generated RFI numbers with RFI-YYYY-NNNN format (organization-wide sequencing)
- Action-based workflow consolidation (submit, approve, reject, respond)
- Unified request structure compatible with UI expectations
- Rich metadata including cost and schedule impact tracking
- Comment and attachment support
- Distribution list and notification management
- Approval workflow for critical RFIs

---

## Database Schema

### Table: `project.rfis`

**Primary Table:** Stores all RFI information including questions, responses, impact tracking, and workflow status.

| Column | Type | Required | Default | Description |
|--------|------|----------|---------|-------------|
| `id` | bigint | Yes | Auto-increment | Primary key |
| `project_id` | bigint | Yes | - | Project ID (FK to project.projects) |
| `org_id` | bigint | Yes | - | Organization ID (FK to iam.organizations) |
| `location_id` | bigint | No | - | Location ID (FK to iam.locations) |
| `rfi_number` | varchar(50) | Yes | - | Auto-generated (RFI-YYYY-NNNN) |
| `subject` | varchar(500) | Yes | - | RFI subject line |
| `question` | text | Yes | - | Detailed question/clarification needed |
| `description` | text | No | - | Additional description or context |
| `category` | varchar(100) | No | - | Category: DESIGN, SPECIFICATION, SCHEDULE, COORDINATION, GENERAL, SUBMITTAL, CHANGE_EVENT |
| `discipline` | varchar(100) | No | - | Discipline: structural, architectural, mechanical, electrical, etc. |
| `trade_type` | varchar(100) | No | - | Trade type: concrete, steel, plumbing, etc. |
| `project_phase` | varchar(50) | No | - | Project phase when RFI was created |
| `priority` | varchar(50) | Yes | 'medium' | Priority: LOW, MEDIUM, HIGH, URGENT |
| `status` | varchar(50) | Yes | 'draft' | Status: DRAFT, SUBMITTED, UNDER_REVIEW, ANSWERED, CLOSED, VOID, REQUIRES_REVISION |
| `submitted_by` | bigint | Yes | - | User who created the RFI |
| `assigned_to` | bigint | No | - | User assigned to respond |
| `reviewer_email` | varchar(255) | No | - | Reviewer email address |
| `approver_email` | varchar(255) | No | - | Approver email address |
| `response_by` | bigint | No | - | User who provided response |
| `cc_list` | text[] | No | - | CC email list |
| `distribution_list` | text[] | No | - | Distribution email list |
| `submitted_date` | timestamp | No | - | Date RFI was submitted |
| `due_date` | timestamp | No | - | Expected response due date |
| `response_date` | timestamp | No | - | Date response was provided |
| `closed_date` | timestamp | No | - | Date RFI was closed |
| `response` | text | No | - | Response text |
| `response_status` | varchar(50) | No | - | Response status |
| `cost_impact` | boolean | No | false | Has cost impact |
| `schedule_impact` | boolean | No | false | Has schedule impact |
| `cost_impact_amount` | numeric | No | - | Cost impact amount in dollars |
| `schedule_impact_days` | integer | No | 0 | Schedule impact in days |
| `cost_impact_details` | text | No | - | Details about cost impact |
| `schedule_impact_details` | text | No | - | Details about schedule impact |
| `location_description` | varchar(500) | No | - | Physical location description |
| `drawing_references` | text | No | - | Comma-separated drawing references |
| `specification_references` | text | No | - | Comma-separated spec references |
| `related_submittals` | text | No | - | Related submittals |
| `related_change_events` | text | No | - | Related change events |
| `related_rfis` | text[] | No | - | Array of related RFI numbers |
| `workflow_type` | varchar(50) | No | 'STANDARD' | Workflow type: STANDARD, EXPRESS, CUSTOM |
| `requires_approval` | boolean | No | false | Requires formal approval |
| `approval_status` | varchar(50) | No | - | Approval status |
| `approved_by` | bigint | No | - | User who approved |
| `approval_date` | timestamp | No | - | Approval date |
| `approval_comments` | text | No | - | Approval comments |
| `urgency_justification` | text | No | - | Justification for urgent priority |
| `business_justification` | text | No | - | Business justification |
| `days_open` | integer | No | 0 | Calculated days open |
| `is_overdue` | boolean | No | false | Calculated overdue flag |
| `created_at` | timestamp | Yes | CURRENT_TIMESTAMP | Creation timestamp |
| `created_by` | bigint | Yes | - | Creator user ID |
| `updated_at` | timestamp | Yes | CURRENT_TIMESTAMP | Last update timestamp |
| `updated_by` | bigint | Yes | - | Last updater user ID |
| `is_deleted` | boolean | Yes | false | Soft delete flag |

### Related Tables

**`project.rfi_attachments`** - Attachments for RFIs (photos, drawings, documents)
**`project.rfi_comments`** - Comments and status change history

---

## Data Models

### Core Models

**Location:** `/Users/mayur/git_personal/infrastructure/src/lib/models/rfi.go`

```go
type RFI struct {
    ID                      int64          `json:"id"`
    ProjectID               int64          `json:"project_id"`
    OrgID                   int64          `json:"org_id"`
    LocationID              *int64         `json:"location_id,omitempty"`
    RFINumber               string         `json:"rfi_number"`
    Subject                 string         `json:"subject"`
    Question                string         `json:"question"`
    Description             *string        `json:"description,omitempty"`
    Category                *string        `json:"category,omitempty"`
    Discipline              *string        `json:"discipline,omitempty"`
    TradeType               *string        `json:"trade_type,omitempty"`
    Priority                string         `json:"priority"`
    Status                  string         `json:"status"`
    SubmittedBy             int64          `json:"submitted_by"`
    AssignedTo              *int64         `json:"assigned_to,omitempty"`
    DistributionList        []string       `json:"distribution_list,omitempty"`
    DueDate                 *time.Time     `json:"due_date,omitempty"`
    Response                *string        `json:"response,omitempty"`
    CostImpact              bool           `json:"cost_impact"`
    ScheduleImpact          bool           `json:"schedule_impact"`
    CostImpactAmount        *float64       `json:"cost_impact_amount,omitempty"`
    ScheduleImpactDays      *int           `json:"schedule_impact_days,omitempty"`
    DaysOpen                int            `json:"days_open,omitempty"`
    IsOverdue               bool           `json:"is_overdue"`
    CreatedAt               time.Time      `json:"created_at"`
    UpdatedAt               time.Time      `json:"updated_at"`
    // ... additional fields
}
```

### Request Models

**Unified Structure** (Used for both Create and Update):

```go
type RFIRequest struct {
    // Project Context
    ProjectID  int64  `json:"project_id,omitempty"`
    LocationID *int64 `json:"location_id,omitempty"`

    // Basic Information (snake_case for UI compatibility)
    RFINumber string `json:"rfi_number,omitempty"` // Auto-generated
    Subject   string `json:"subject" binding:"required,max=500"`
    Question  string `json:"question" binding:"required"`

    // Classification
    Priority string `json:"priority" binding:"required,oneof=LOW MEDIUM HIGH URGENT"`
    Category string `json:"category" binding:"required,oneof=DESIGN SPECIFICATION SCHEDULE COORDINATION GENERAL SUBMITTAL CHANGE_EVENT"`

    // Scheduling
    DueDate string `json:"due_date,omitempty"` // YYYY-MM-DD format

    // Assignment
    AssignedTo *string `json:"assigned_to,omitempty"` // User ID as string

    // Communication
    DistributionList []string `json:"distribution_list,omitempty"`

    // References (nested object)
    References *RFIReferences `json:"references,omitempty"`

    // Impact Tracking
    CostImpactAmount   *float64 `json:"cost_impact_amount,omitempty"`
    ScheduleImpactDays *int     `json:"schedule_impact_days,omitempty"`

    // Additional Context
    Description         string `json:"description,omitempty"`
    LocationDescription string `json:"location_description,omitempty"`
    Discipline          string `json:"discipline,omitempty"`
    TradeType           string `json:"trade_type,omitempty"`
}

type RFIReferences struct {
    DrawingNumbers        []string `json:"drawing_numbers,omitempty"`
    SpecificationSections []string `json:"specification_sections,omitempty"`
    RelatedRFIs           []string `json:"related_rfis,omitempty"`
}

type UpdateRFIRequest struct {
    RFIRequest
    // Action-based update fields (for workflow consolidation)
    Action       *string `json:"action,omitempty"`        // submit, approve, reject, respond
    Notes        string  `json:"notes,omitempty"`         // General notes/comments
    ResponseText string  `json:"response_text,omitempty"` // Response text for respond action
}
```

### Response Models

```go
type RFIResponse struct {
    RFI
    ProjectName           string          `json:"project_name,omitempty"`
    LocationName          string          `json:"location_name,omitempty"`
    SubmittedByName       string          `json:"submitted_by_name,omitempty"`
    AssignedToName        string          `json:"assigned_to_name,omitempty"`
    ResponseByName        string          `json:"response_by_name,omitempty"`
    Attachments           []RFIAttachment `json:"attachments"`
    Comments              []RFIComment    `json:"comments,omitempty"`
    CommentCount          int             `json:"comment_count,omitempty"`
    AttachmentCount       int             `json:"attachment_count,omitempty"`
}

type RFIComment struct {
    ID            int64      `json:"id"`
    RFIID         int64      `json:"rfi_id"`
    Comment       string     `json:"comment"`
    CommentType   string     `json:"comment_type"` // comment, status_change, assignment, response, approval, rejection
    PreviousValue string     `json:"previous_value,omitempty"`
    NewValue      string     `json:"new_value,omitempty"`
    CreatedAt     time.Time  `json:"created_at"`
    CreatedBy     int64      `json:"created_by"`
    CreatedByName string     `json:"created_by_name,omitempty"`
}

type RFIAttachment struct {
    ID             int64     `json:"id"`
    RFIID          int64     `json:"rfi_id"`
    FileName       string    `json:"file_name"`
    FilePath       string    `json:"file_path,omitempty"`
    FileType       string    `json:"file_type,omitempty"`
    FileSize       int64     `json:"file_size,omitempty"`
    S3Bucket       string    `json:"s3_bucket,omitempty"`
    S3Key          string    `json:"s3_key,omitempty"`
    S3URL          string    `json:"s3_url,omitempty"`
    AttachmentType string    `json:"attachment_type"`
    UploadedBy     int64     `json:"uploaded_by"`
    UploadDate     time.Time `json:"upload_date"`
}
```

---

## API Endpoints

**Service Location:** `/Users/mayur/git_personal/infrastructure/src/infrastructure-rfi-management/main.go`

### 1. Create RFI
**POST** `/rfis`

Creates a new RFI with auto-generated RFI number. Status is set to DRAFT.

**Request Body:**
```json
{
    "project_id": 47,
    "location_id": 22,
    "subject": "Clarification on Foundation Detail at Grid Line A-5",
    "question": "The foundation detail shown in drawing S-101 at grid line A-5 conflicts with the specifications in section 03300. The drawing shows a 24-inch square footing, but the specification calls for a 30-inch square footing. Which requirement takes precedence?",
    "description": "Need immediate clarification as we are ready to pour concrete footings in this area. The conflict affects the reinforcement layout and concrete quantity.",
    "category": "DESIGN",
    "discipline": "structural",
    "trade_type": "concrete",
    "project_phase": "construction",
    "priority": "HIGH",
    "distribution_list": [
        "project.manager@company.com",
        "structural.engineer@company.com"
    ],
    "due_date": "2025-10-15",
    "cost_impact_amount": 2500.00,
    "schedule_impact_days": 2,
    "location_description": "Building A, Foundation at Grid Line A-5",
    "references": {
        "drawing_numbers": ["S-101", "S-102"],
        "specification_sections": ["Section 03300 - Cast-in-Place Concrete"],
        "related_rfis": []
    }
}
```

**Response (201 Created):**
```json
{
    "id": 15,
    "rfi_number": "RFI-2025-0015",
    "project_id": 47,
    "subject": "Clarification on Foundation Detail at Grid Line A-5",
    "question": "The foundation detail...",
    "category": "DESIGN",
    "priority": "HIGH",
    "status": "DRAFT",
    "submitted_by": 5,
    "created_at": "2025-01-15T10:30:00Z",
    "days_open": 0,
    "is_overdue": false,
    "attachments": [],
    "comments": []
}
```

### 2. Get RFIs by Project (Context Query)
**GET** `/contexts/project/{projectId}/rfis`

Retrieves all RFIs for a project with optional filtering.

**Query Parameters:**
- `status` (optional): Filter by status (DRAFT, SUBMITTED, UNDER_REVIEW, ANSWERED, CLOSED, VOID, REQUIRES_REVISION)
- `priority` (optional): Filter by priority (LOW, MEDIUM, HIGH, URGENT)
- `category` (optional): Filter by category
- `discipline` (optional): Filter by discipline
- `assigned_to` (optional): Filter by assigned user
- `submitted_by` (optional): Filter by submitter

**Response (200 OK):**
```json
{
    "context_type": "project",
    "context_id": 47,
    "rfis": [
        {
            "id": 15,
            "rfi_number": "RFI-2025-0015",
            "subject": "Clarification on Foundation Detail",
            "priority": "HIGH",
            "status": "SUBMITTED",
            "submitted_by": 5,
            "submitted_by_name": "John Doe",
            "created_at": "2025-01-15T10:30:00Z",
            "days_open": 3,
            "is_overdue": false
        }
    ]
}
```

### 3. Get RFI by ID
**GET** `/rfis/{rfiId}`

Retrieves detailed information for a specific RFI including all comments and attachments.

**Response (200 OK):**
Returns full RFI object with enriched data (comments, attachments, user names).

### 4. Update RFI (Regular Update)
**PUT** `/rfis/{rfiId}`

Updates RFI information. Can only be updated by submitter or if status is DRAFT.

**Request Body (Partial Update Supported):**
```json
{
    "subject": "UPDATED: Clarification on Foundation Detail at Grid Line A-5",
    "question": "Updated question with additional context...",
    "priority": "URGENT",
    "cost_impact_amount": 3500.00,
    "schedule_impact_days": 3,
    "references": {
        "drawing_numbers": ["S-101", "S-102", "S-105"],
        "specification_sections": ["Section 03300 - Cast-in-Place Concrete"]
    }
}
```

**Response (200 OK):**
Returns updated RFI object.

### 5. Submit RFI (Action-based)
**PUT** `/rfis/{rfiId}`

Submits RFI for review. Changes status from DRAFT to SUBMITTED.

**Request Body:**
```json
{
    "action": "submit",
    "notes": "All required information and attachments have been provided. This RFI is ready for structural review."
}
```

**Response (200 OK):**
```json
{
    "message": "RFI updated successfully"
}
```

### 6. Approve RFI (Action-based)
**PUT** `/rfis/{rfiId}`

Approves an RFI response. Requires approval workflow to be enabled.

**Request Body:**
```json
{
    "action": "approve",
    "notes": "RFI response is comprehensive and addresses all concerns. The structural engineer's recommendation to use 30-inch footings is approved."
}
```

### 7. Reject RFI (Action-based)
**PUT** `/rfis/{rfiId}`

Rejects an RFI. Notes field is required to explain rejection reason.

**Request Body:**
```json
{
    "action": "reject",
    "notes": "Insufficient information provided. Please include soil bearing capacity test results and provide clearer site photos."
}
```

### 8. Respond to RFI (Action-based)
**PUT** `/rfis/{rfiId}`

Provides a response to the RFI. Changes status to ANSWERED. response_text is required.

**Request Body:**
```json
{
    "action": "respond",
    "response_text": "After reviewing both the drawings and specifications, the specification takes precedence. Please use the 30-inch square footing as specified in section 03300. The drawing will be revised in the next update.",
    "notes": "Response provided with detailed clarification on foundation requirements."
}
```

### 9. Add RFI Comment
**POST** `/rfis/{rfiId}/comments`

Adds a comment to the RFI timeline.

**Request Body:**
```json
{
    "comment": "Site visit completed today. Confirmed that the existing conditions match the drawing dimensions."
}
```

**Response (201 Created):**
```json
{
    "id": 42,
    "rfi_id": 15,
    "comment": "Site visit completed today...",
    "comment_type": "comment",
    "created_by": 5,
    "created_by_name": "John Doe",
    "created_at": "2025-01-16T14:22:00Z"
}
```

### 10. Add RFI Attachment (Centralized Service)
**POST** `/rfis/{rfiId}/attachments`

**Note:** RFI attachments are now handled by the centralized attachment management service.

---

## Repository Methods

**Location:** `/Users/mayur/git_personal/infrastructure/src/lib/data/rfi_repository.go`

### Key Methods

```go
// RFI CRUD
CreateRFI(ctx, projectID, userID, orgID, req) (*RFIResponse, error)
GetRFI(ctx, rfiID) (*RFIResponse, error)
GetRFIsByProject(ctx, projectID, filters) ([]RFIResponse, error)
UpdateRFI(ctx, rfiID, userID, orgID, req) (*RFIResponse, error)
DeleteRFI(ctx, rfiID, deletedBy) error

// Workflow Actions
SubmitRFI(ctx, rfiID, assignedTo, submittedBy) error
RespondToRFI(ctx, rfiID, response, responseBy) error
ApproveRFI(ctx, rfiID, approvedBy, comments) error
RejectRFI(ctx, rfiID, rejectedBy, reason) error
UpdateRFIStatus(ctx, rfiID, status, updatedBy, comment) error

// Comments and Attachments
AddRFIComment(ctx, comment) error
GetRFIComments(ctx, rfiID) ([]RFIComment, error)
AddRFIAttachment(ctx, attachment) (*RFIAttachment, error)
GetRFIAttachments(ctx, rfiID) ([]RFIAttachment, error)

// Auto-numbering
GenerateRFINumber(ctx, projectID) (string, error)
```

### Auto-Numbering Logic

RFIs use an organization-wide auto-numbering system with the format **RFI-YYYY-NNNN**:

```go
func GenerateRFINumber(ctx, projectID) (string, error) {
    // Get project's year (from start_date or created_at)
    var projectYear int
    query := `
        SELECT EXTRACT(YEAR FROM COALESCE(start_date, created_at))
        FROM project.projects WHERE id = $1
    `
    db.QueryRow(query, projectID).Scan(&projectYear)

    // Get the next sequence number for this organization and year
    sequenceQuery := `
        SELECT COALESCE(MAX(
            CAST(SUBSTRING(rfi_number FROM '[0-9]+$') AS INTEGER)
        ), 0)
        FROM project.rfis r
        JOIN project.projects p ON r.project_id = p.id
        WHERE p.org_id = (SELECT org_id FROM project.projects WHERE id = $1)
        AND rfi_number LIKE $2
    `

    prefix := fmt.Sprintf("RFI-%d-", projectYear)
    db.QueryRow(sequenceQuery, projectID, prefix+"%").Scan(&maxNumber)

    // Generate new RFI number
    return fmt.Sprintf("%s%04d", prefix, maxNumber+1), nil
}
```

**Example:** RFI-2025-0001, RFI-2025-0002, etc.

**Important:** Numbering is organization-wide, not project-specific. All projects in an organization share the same RFI number sequence for a given year.

---

## Workflow States and Actions

### RFI Statuses

1. **DRAFT** - Initial state, RFI is being prepared
2. **SUBMITTED** - RFI has been submitted for review
3. **UNDER_REVIEW** - RFI is being reviewed
4. **ANSWERED** - Response has been provided
5. **CLOSED** - RFI is closed/resolved
6. **VOID** - RFI is voided/cancelled
7. **REQUIRES_REVISION** - RFI needs to be revised

### Priority Levels

- **LOW** - Non-critical, informational
- **MEDIUM** - Standard priority (default)
- **HIGH** - Important, needs timely response
- **URGENT** - Critical, requires immediate attention

### Categories

- **DESIGN** - Design clarifications
- **SPECIFICATION** - Specification conflicts or questions
- **SCHEDULE** - Schedule-related questions
- **COORDINATION** - Trade coordination issues
- **GENERAL** - General questions
- **SUBMITTAL** - Submittal-related questions
- **CHANGE_EVENT** - Related to change orders

### Workflow Actions

```go
// Consolidated in PUT /rfis/{id} via action field

1. submit:
   - Changes status: DRAFT → SUBMITTED
   - Sets submitted_date
   - Can optionally assign to user

2. approve:
   - Sets approval_status = 'approved'
   - Sets approved_by and approval_date
   - Adds approval comment

3. reject:
   - Sets approval_status = 'rejected'
   - Requires rejection reason in notes
   - Adds rejection comment

4. respond:
   - Changes status: SUBMITTED/UNDER_REVIEW → ANSWERED
   - Sets response, response_by, response_date
   - Requires response_text
   - Adds response comment
```

---

## Postman Collection

**Location:** `/Users/mayur/git_personal/infrastructure/postman/RFIManagement.postman_collection.json`

**Available Requests:**
1. Create RFI
2. Get RFIs by Project (Context Query)
3. Get RFI by ID
4. Update RFI (Full Update)
5. Submit RFI (Action-based)
6. Approve RFI (Action-based)
7. Reject RFI (Action-based)
8. Respond to RFI (Action-based)
9. Add RFI Comment
10. Add RFI Attachment

**Environment Variables:**
- `access_token`: JWT ID token (not access token!)
- `project_id`: Project ID (default: 47)
- `location_id`: Location ID (default: 22)
- `rfi_id`: Auto-populated after creation

**Base URL:** `https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main`

---

## Best Practices

### Date Handling
- Always use YYYY-MM-DD format for dates
- `due_date` is optional but recommended for tracking
- System calculates `days_open` and `is_overdue` automatically

### Status Values and Transitions
**Valid Transitions:**
- DRAFT → SUBMITTED (via submit action)
- SUBMITTED → UNDER_REVIEW (automatic or manual)
- UNDER_REVIEW → ANSWERED (via respond action)
- ANSWERED → CLOSED (manual)
- Any status → VOID (manual, with justification)
- Any status → REQUIRES_REVISION (via reject action)

### Priority Guidelines
- **URGENT**: Use sparingly, requires urgency_justification
- **HIGH**: For items blocking work or with significant impact
- **MEDIUM**: Standard RFIs (default)
- **LOW**: Informational or future planning items

### Category Selection
Choose the most specific category:
- **DESIGN** for conflicts in design intent
- **SPECIFICATION** for spec document questions
- **COORDINATION** for multi-trade coordination
- **SUBMITTAL** for submittal-related clarifications

### Cost and Schedule Impact
- Set `cost_impact_amount` for financial tracking
- Set `schedule_impact_days` for schedule analysis
- Provide details in `cost_impact_details` and `schedule_impact_details`
- Impacts can be updated as situation evolves

### References
Use the nested `references` object for better organization:
```json
{
    "references": {
        "drawing_numbers": ["S-101", "S-102"],
        "specification_sections": ["Section 03300"],
        "related_rfis": ["RFI-2025-0010"]
    }
}
```

### Distribution Lists
- Use `distribution_list` for email notifications
- Include all stakeholders who need visibility
- Can be updated as needed throughout RFI lifecycle

### Comment Types
- **comment** - General comments
- **status_change** - Automatic when status changes
- **assignment** - Automatic when RFI is assigned
- **response** - Automatic when response is provided
- **approval** - Automatic when RFI is approved
- **rejection** - Automatic when RFI is rejected

---

## Validation Rules

### Organization Validation
- RFI creation validates that project belongs to user's organization
- Returns error if project doesn't exist or belongs to different org

### Update Permissions
- Can update if RFI is in DRAFT status (any user)
- Can update if user is the submitter (any status)
- Otherwise, returns "Cannot update RFI in current status"

### Required Fields for Actions

**Submit Action:**
- No additional fields required
- Optional: assigned_to, notes

**Approve Action:**
- Optional: notes (recommended)

**Reject Action:**
- Required: notes (rejection reason)

**Respond Action:**
- Required: response_text
- Optional: notes

---

## Testing

### Test User
- Email: `buildboard007+555@gmail.com`
- Password: `Mayur@1234`

### Sample Test Scenarios

**1. Create and Submit RFI:**
```bash
# Create RFI (status = DRAFT)
POST /rfis

# Submit for review (status = SUBMITTED)
PUT /rfis/{id} with action="submit"
```

**2. Complete RFI Workflow:**
```bash
# Create → Submit → Respond → Close
POST /rfis
PUT /rfis/{id} action="submit"
PUT /rfis/{id} action="respond" response_text="..."
PUT /rfis/{id} status="CLOSED"
```

**3. RFI with Approval:**
```bash
# Create with requires_approval=true
POST /rfis requires_approval=true

# Submit and approve
PUT /rfis/{id} action="submit"
PUT /rfis/{id} action="approve"
```

**4. RFI Rejection and Revision:**
```bash
# Submit RFI
PUT /rfis/{id} action="submit"

# Reject with reason
PUT /rfis/{id} action="reject" notes="Insufficient information"

# Revise and resubmit
PUT /rfis/{id} (update fields)
PUT /rfis/{id} action="submit"
```

---

## Related Documentation

- [Project Management](./project-management.md) - Parent project management
- [Submittal Management](./submittal-management.md) - Related submittal workflow
- [Issue Management](./issue-management.md) - Issue tracking system
- [Attachment Management](./attachment-management.md) - Centralized attachment service

---

## Implementation Notes

1. **Consolidated Endpoints:** The API has been consolidated from 15+ endpoints to 6 core endpoints using action-based workflow updates

2. **Organization-wide Numbering:** RFI numbers are unique across all projects in an organization for a given year

3. **Soft Delete:** All RFIs use soft delete (`is_deleted=true`) to maintain audit trail

4. **Automatic Calculations:**
   - `days_open`: Calculated as current_date - created_at
   - `is_overdue`: Calculated based on due_date and status

5. **Comment Tracking:** All workflow actions automatically create comment entries with appropriate types

6. **Attachment Management:** Attachments are being migrated to centralized attachment service

7. **UI Compatibility:** Request/response models use snake_case and match UI expectations

8. **Transaction Safety:** Status changes and comment creation use database transactions for atomicity

---

## Constants Reference

### Status Constants
```go
const (
    RFIStatusDraft           = "DRAFT"
    RFIStatusSubmitted       = "SUBMITTED"
    RFIStatusUnderReview     = "UNDER_REVIEW"
    RFIStatusAnswered        = "ANSWERED"
    RFIStatusClosed          = "CLOSED"
    RFIStatusVoid            = "VOID"
    RFIStatusRequiresRevision = "REQUIRES_REVISION"
)
```

### Priority Constants
```go
const (
    RFIPriorityLow    = "LOW"
    RFIPriorityMedium = "MEDIUM"
    RFIPriorityHigh   = "HIGH"
    RFIPriorityUrgent = "URGENT"
)
```

### Category Constants
```go
const (
    RFICategoryDesign        = "DESIGN"
    RFICategorySpecification = "SPECIFICATION"
    RFICategorySchedule      = "SCHEDULE"
    RFICategoryCoordination  = "COORDINATION"
    RFICategoryGeneral       = "GENERAL"
    RFICategorySubmittal     = "SUBMITTAL"
    RFICategoryChangeEvent   = "CHANGE_EVENT"
)
```

### Workflow Type Constants
```go
const (
    RFIWorkflowStandard = "STANDARD"
    RFIWorkflowExpress  = "EXPRESS"
    RFIWorkflowCustom   = "CUSTOM"
)
```