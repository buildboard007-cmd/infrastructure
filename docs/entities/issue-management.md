# Issue Management System

## Overview

The Issue Management system provides comprehensive issue tracking, punch list management, and deficiency reporting for construction projects. It supports the complete lifecycle of construction issues from identification through resolution, including commenting, status tracking, and attachment management.

**Key Features:**
- Auto-generated issue numbers (format: `{PROJECT_CODE}-{CATEGORY}-NNNN`)
- Flexible categorization (issue type, category, detail category, priority, severity)
- Assignment and workflow management
- Comment system with attachments
- Activity logging for status changes
- Location tracking (building, level, room, coordinates, GPS)
- Distribution lists for notifications
- Cost tracking and root cause analysis
- Integration with attachments, RFIs, and submittals

**Use Cases:**
- Punch list items during project closeout
- Quality control deficiency tracking
- Safety issue reporting
- Design clarification requests
- General project issue tracking

---

## Database Schema

### Issues Table (`project.issues`)

```sql
CREATE TABLE project.issues (
    -- Primary Key
    id                      BIGSERIAL PRIMARY KEY,

    -- Project Association
    project_id              BIGINT NOT NULL REFERENCES project.projects(id),
    issue_number            VARCHAR(50) NOT NULL UNIQUE,
    template_id             BIGINT REFERENCES project.issue_templates(id),

    -- Basic Information
    title                   VARCHAR(255) NOT NULL,
    description             TEXT NOT NULL,

    -- Categorization
    issue_type              VARCHAR(50) NOT NULL DEFAULT 'general',
    issue_category          VARCHAR(100),
    category                VARCHAR(100),
    detail_category         VARCHAR(100),

    -- Priority & Severity
    priority                VARCHAR(50) NOT NULL DEFAULT 'medium',
    severity                VARCHAR(50) NOT NULL DEFAULT 'minor',
    root_cause              TEXT,

    -- Location Information
    location_description    VARCHAR(255),
    location_building       VARCHAR(100),
    location_level          VARCHAR(50),
    location_room           VARCHAR(100),
    location_x              NUMERIC(10,2),
    location_y              NUMERIC(10,2),

    -- Legacy Location Fields
    room_area               VARCHAR(100),
    floor_level             VARCHAR(50),

    -- Trade & Assignment
    discipline              VARCHAR(100),
    trade_type              VARCHAR(100),
    reported_by             BIGINT NOT NULL REFERENCES iam.users(id),
    assigned_to             BIGINT REFERENCES iam.users(id),
    assigned_company_id     BIGINT REFERENCES iam.companies(id),

    -- References
    drawing_reference       VARCHAR(255),
    specification_reference VARCHAR(255),

    -- Timeline
    due_date                DATE,
    closed_date             TIMESTAMP,

    -- Distribution
    distribution_list       TEXT[],

    -- Status
    status                  VARCHAR(50) NOT NULL DEFAULT 'open',

    -- Cost Impact
    cost_to_fix             NUMERIC(15,2) DEFAULT 0.00,

    -- GPS Coordinates
    latitude                NUMERIC(10,6),
    longitude               NUMERIC(10,6),

    -- Audit Fields
    created_at              TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by              BIGINT NOT NULL REFERENCES iam.users(id),
    updated_at              TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by              BIGINT NOT NULL REFERENCES iam.users(id),
    is_deleted              BOOLEAN NOT NULL DEFAULT FALSE
);

-- Indexes
CREATE INDEX idx_issues_project_id ON project.issues(project_id);
CREATE INDEX idx_issues_status ON project.issues(status);
CREATE INDEX idx_issues_assigned_to ON project.issues(assigned_to);
CREATE INDEX idx_issues_created_at ON project.issues(created_at);
CREATE UNIQUE INDEX idx_issues_issue_number ON project.issues(issue_number);
```

**Status Values:**
- `open` - New issue, needs attention
- `in_progress` - Work in progress
- `ready_for_review` - Completed, awaiting verification
- `closed` - Resolved and verified
- `rejected` - Issue rejected/invalid
- `on_hold` - Temporarily paused

**Priority Values:**
- `critical` - Immediate attention required, project blocker
- `high` - Important, affects schedule or safety
- `medium` - Standard priority
- `low` - Minor issue, can be deferred
- `planned` - Scheduled for future work

**Severity Values:**
- `blocking` - Prevents other work
- `major` - Significant impact
- `minor` - Small impact
- `cosmetic` - Visual/aesthetic only

### Issue Comments Table (`project.issue_comments`)

```sql
CREATE TABLE project.issue_comments (
    id              BIGSERIAL PRIMARY KEY,
    issue_id        BIGINT NOT NULL REFERENCES project.issues(id),
    comment         TEXT NOT NULL,
    comment_type    VARCHAR(50) NOT NULL DEFAULT 'comment',
    previous_value  VARCHAR(255),
    new_value       VARCHAR(255),
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by      BIGINT NOT NULL REFERENCES iam.users(id),
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by      BIGINT NOT NULL REFERENCES iam.users(id),
    is_deleted      BOOLEAN NOT NULL DEFAULT FALSE
);

-- Indexes
CREATE INDEX idx_issue_comments_issue_id ON project.issue_comments(issue_id);
CREATE INDEX idx_issue_comments_created_at ON project.issue_comments(created_at);
```

**Comment Types:**
- `comment` - User-added comment
- `activity` - System-generated activity log (status changes, assignments)

### Issue Attachments Table

See [Attachment Management Documentation](./attachment-management.md) for complete schema.

```sql
CREATE TABLE project.issue_attachments (
    id              BIGSERIAL PRIMARY KEY,
    issue_id        BIGINT NOT NULL REFERENCES project.issues(id),
    file_name       VARCHAR(255) NOT NULL,
    file_path       VARCHAR(500) NOT NULL,
    file_size       BIGINT,
    file_type       VARCHAR(50),
    attachment_type VARCHAR(50) NOT NULL DEFAULT 'before_photo',
    uploaded_by     BIGINT NOT NULL REFERENCES iam.users(id),
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by      BIGINT NOT NULL REFERENCES iam.users(id),
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by      BIGINT NOT NULL REFERENCES iam.users(id),
    is_deleted      BOOLEAN NOT NULL DEFAULT FALSE
);
```

**Attachment Types for Issues:**
- `before_photo` - Photo before work
- `progress_photo` - Work in progress photo
- `after_photo` - Photo after completion
- `issue_document` - General documentation

---

## Data Models

### Issue Model

```go
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
    IssueType      string         `json:"issue_type"`

    // Priority & Severity
    Priority string `json:"priority"`
    Severity string `json:"severity"`
    RootCause sql.NullString `json:"root_cause,omitempty"`

    // Location Information
    LocationDescription sql.NullString  `json:"location_description,omitempty"`
    LocationBuilding    sql.NullString  `json:"location_building,omitempty"`
    LocationLevel       sql.NullString  `json:"location_level,omitempty"`
    LocationRoom        sql.NullString  `json:"location_room,omitempty"`
    LocationX           sql.NullFloat64 `json:"location_x,omitempty"`
    LocationY           sql.NullFloat64 `json:"location_y,omitempty"`

    // Trade & Assignment
    Discipline        sql.NullString `json:"discipline,omitempty"`
    TradeType         sql.NullString `json:"trade_type,omitempty"`
    ReportedBy        int64          `json:"reported_by"`
    AssignedTo        sql.NullInt64  `json:"assigned_to,omitempty"`
    AssignedCompanyID sql.NullInt64  `json:"assigned_company_id,omitempty"`

    // Timeline
    DueDate    *time.Time `json:"due_date,omitempty"`
    ClosedDate *time.Time `json:"closed_date,omitempty"`

    // Status
    Status string `json:"status"`

    // Cost Impact
    CostToFix sql.NullFloat64 `json:"cost_to_fix,omitempty"`

    // Audit fields
    CreatedAt time.Time `json:"created_at"`
    CreatedBy int64     `json:"created_by"`
    UpdatedAt time.Time `json:"updated_at"`
    UpdatedBy int64     `json:"updated_by"`

    // Related Data
    Comments []IssueComment `json:"comments,omitempty"`
}
```

### Issue Request Model

```go
type IssueRequest struct {
    // Project Context
    ProjectID  int64 `json:"project_id,omitempty"`
    LocationID int64 `json:"location_id,omitempty"`

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

    // Location Details
    Location IssueLocationInfo `json:"location" binding:"required"`

    // Trade/Discipline Information
    Discipline string `json:"discipline,omitempty"`
    Trade      string `json:"trade,omitempty"`

    // Assignment and Timeline
    AssignedTo int64  `json:"assigned_to" binding:"required"`
    DueDate    string `json:"due_date" binding:"required"` // ISO date: "2025-10-15"

    // Distribution
    DistributionList []string `json:"distribution_list,omitempty"`

    // Status (for updates)
    Status string `json:"status,omitempty" binding:"omitempty,oneof=open in_progress ready_for_review closed rejected on_hold"`
}
```

### Issue Response Model

```go
type IssueResponse struct {
    // Core fields
    ID          int64  `json:"id"`
    ProjectID   int64  `json:"project_id"`
    IssueNumber string `json:"issue_number"`

    // Basic Information
    Title       string `json:"title"`
    Description string `json:"description"`

    // Categorization
    Category       string `json:"category,omitempty"`
    DetailCategory string `json:"detail_category,omitempty"`

    // Priority & Severity
    Priority string `json:"priority"`
    Severity string `json:"severity"`

    // Location Information
    LocationDescription string   `json:"location_description,omitempty"`
    LocationBuilding    string   `json:"location_building,omitempty"`
    LocationLevel       string   `json:"location_level,omitempty"`
    LocationRoom        string   `json:"location_room,omitempty"`

    // Assignment
    ReportedBy        int64  `json:"reported_by"`
    AssignedTo        *int64 `json:"assigned_to,omitempty"`

    // Timeline
    DueDate    *time.Time `json:"due_date,omitempty"`
    ClosedDate *time.Time `json:"closed_date,omitempty"`

    // Status
    Status string `json:"status"`

    // Audit fields
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`

    // Computed fields
    ProjectName    string `json:"project_name,omitempty"`
    AssignedToName string `json:"assigned_to_name,omitempty"`
    DaysOpen       int    `json:"days_open,omitempty"`
    IsOverdue      bool   `json:"is_overdue"`

    // Related Data
    Attachments []IssueAttachment `json:"attachments"`
    Comments    []IssueComment    `json:"comments,omitempty"`
}
```

### Issue Comment Model

```go
type IssueComment struct {
    ID            int64                    `json:"id"`
    IssueID       int64                    `json:"issue_id"`
    Comment       string                   `json:"comment"`
    CommentType   string                   `json:"comment_type"` // 'comment' or 'activity'
    PreviousValue string                   `json:"previous_value,omitempty"`
    NewValue      string                   `json:"new_value,omitempty"`
    Attachments   []IssueCommentAttachment `json:"attachments"`
    CreatedAt     time.Time                `json:"created_at"`
    CreatedBy     int64                    `json:"created_by"`
    CreatedByName string                   `json:"created_by_name,omitempty"`
    UpdatedAt     time.Time                `json:"updated_at"`
    UpdatedBy     int64                    `json:"updated_by"`
    IsDeleted     bool                     `json:"is_deleted"`
}

type CreateCommentRequest struct {
    Comment       string  `json:"comment" binding:"required"`
    AttachmentIDs []int64 `json:"attachment_ids,omitempty"`
}
```

---

## API Endpoints

**Base URL:** `https://{api-gateway-url}/main`

**Authentication:** All endpoints require JWT ID token in `Authorization: Bearer {token}` header

### Issue CRUD Operations

#### 1. Create Issue

```http
POST /issues
Content-Type: application/json
Authorization: Bearer {jwt_token}

{
  "project_id": 49,
  "location_id": 24,
  "issue_category": "quality",
  "category": "deficiency",
  "detail_category": "concrete",
  "title": "Wall crack in conference room",
  "description": "Large crack observed in west wall, needs immediate attention",
  "priority": "high",
  "severity": "major",
  "root_cause": "Settlement",
  "location": {
    "description": "Conference Room A",
    "building": "Building 1",
    "level": "Floor 2",
    "room": "Room 201",
    "coordinates": {
      "x": 100.5,
      "y": 250.3
    }
  },
  "discipline": "Civil",
  "trade": "Concrete",
  "assigned_to": 25,
  "due_date": "2025-10-15",
  "distribution_list": ["pm@example.com", "super@example.com"]
}

Response (201 Created):
{
  "id": 72,
  "issue_number": "PRJ-DE-0001",
  "title": "Wall crack in conference room",
  "status": "open",
  "priority": "high",
  "severity": "major",
  "project_id": 49,
  "assigned_to": 25,
  "due_date": "2025-10-15",
  "created_at": "2025-10-06T20:15:30Z",
  "created_by": 19,
  "is_overdue": false,
  "days_open": 0,
  "attachments": []
}
```

**Auto-Generated Issue Number:**
- Format: `{PROJECT_CODE}-{CATEGORY_PREFIX}-NNNN`
- Example: `PRJ-DE-0001` = Project "PRJ", Category "Deficiency", Number 0001
- Sequential numbering per project per category

**Validation:**
- `project_id`, `title`, `description`, `priority`, `assigned_to`, `due_date` are required
- `assigned_to` user must exist and belong to the same organization
- Project must belong to user's organization

#### 2. Get Issue by ID

```http
GET /issues/{issueId}
Authorization: Bearer {jwt_token}

Response (200 OK):
{
  "id": 72,
  "issue_number": "PRJ-DE-0001",
  "title": "Wall crack in conference room",
  "description": "Large crack observed in west wall, needs immediate attention",
  "status": "open",
  "priority": "high",
  "severity": "major",
  "category": "deficiency",
  "location_description": "Conference Room A",
  "location_building": "Building 1",
  "location_level": "Floor 2",
  "location_room": "Room 201",
  "assigned_to": 25,
  "assigned_to_name": "John Doe",
  "reported_by": 19,
  "due_date": "2025-10-15",
  "project_id": 49,
  "project_name": "Main Office Building",
  "days_open": 5,
  "is_overdue": false,
  "created_at": "2025-10-06T20:15:30Z",
  "updated_at": "2025-10-06T20:15:30Z",
  "attachments": [
    {
      "id": 6,
      "issue_id": 72,
      "file_name": "wall_crack_photo.jpg",
      "file_path": "10/24/49/issues/72/20251006201530_wall_crack_photo.jpg",
      "file_size": 524288,
      "file_type": "image/jpeg",
      "attachment_type": "before_photo",
      "uploaded_by": 19,
      "created_at": "2025-10-06T20:16:00Z"
    }
  ],
  "comments": [
    {
      "id": 1,
      "issue_id": 72,
      "comment": "Inspected the crack, appears to be structural",
      "comment_type": "comment",
      "created_by": 25,
      "created_by_name": "John Doe",
      "created_at": "2025-10-07T10:30:00Z",
      "attachments": []
    },
    {
      "id": 2,
      "issue_id": 72,
      "comment": "Status changed from open to in_progress",
      "comment_type": "activity",
      "previous_value": "open",
      "new_value": "in_progress",
      "created_by": 25,
      "created_by_name": "John Doe",
      "created_at": "2025-10-07T11:00:00Z",
      "attachments": []
    }
  ]
}
```

**Access Control:**
- Issue must belong to a project in user's organization
- Returns 403 if organization mismatch
- Returns 404 if issue not found or deleted

#### 3. List Issues for Project

```http
GET /projects/{projectId}/issues?status={status}&priority={priority}&assigned_to={userId}&page=1&page_size=50
Authorization: Bearer {jwt_token}

Response (200 OK):
{
  "issues": [
    {
      "id": 72,
      "issue_number": "PRJ-DE-0001",
      "title": "Wall crack in conference room",
      "status": "open",
      "priority": "high",
      "assigned_to": 25,
      "due_date": "2025-10-15",
      "days_open": 5,
      "is_overdue": false
    },
    {
      "id": 73,
      "issue_number": "PRJ-SA-0001",
      "title": "Missing safety railing",
      "status": "in_progress",
      "priority": "critical",
      "assigned_to": 30,
      "due_date": "2025-10-10",
      "days_open": 10,
      "is_overdue": true
    }
  ],
  "total": 2,
  "page": 1,
  "page_size": 50
}
```

**Query Parameters:**
- `status`: Filter by status (open, in_progress, closed, etc.)
- `priority`: Filter by priority (critical, high, medium, low, planned)
- `severity`: Filter by severity (blocking, major, minor, cosmetic)
- `assigned_to`: Filter by assigned user ID
- `reported_by`: Filter by reporter user ID
- `category`: Filter by category
- `page`: Page number (default: 1)
- `page_size`: Results per page (default: 50, max: 100)

#### 4. Update Issue

```http
PUT /issues/{issueId}
Content-Type: application/json
Authorization: Bearer {jwt_token}

{
  "title": "Wall crack in conference room - URGENT",
  "description": "Large crack observed in west wall, structural engineer consulted",
  "priority": "critical",
  "severity": "blocking",
  "status": "in_progress",
  "assigned_to": 30,
  "due_date": "2025-10-08"
}

Response (200 OK):
{
  "id": 72,
  "issue_number": "PRJ-DE-0001",
  "title": "Wall crack in conference room - URGENT",
  "status": "in_progress",
  "priority": "critical",
  "severity": "blocking",
  "assigned_to": 30,
  "due_date": "2025-10-08",
  "updated_at": "2025-10-07T14:30:00Z"
}
```

**Behavior:**
- Status changes automatically create activity log entries
- Assignment changes create activity log entries
- Access control validates organization membership

#### 5. Update Issue Status (Partial Update)

```http
PATCH /issues/{issueId}/status
Content-Type: application/json
Authorization: Bearer {jwt_token}

{
  "status": "closed"
}

Response (200 OK):
{
  "message": "Issue status updated successfully",
  "status": "closed"
}
```

**Status Workflow:**
```
open → in_progress → ready_for_review → closed
  ↓         ↓              ↓               ↑
  └──────→ on_hold ──────────────────────┘
  └──────→ rejected
```

**Activity Logging:**
- Status changes create automatic activity log entries
- Includes previous and new status values
- Activity appears in issue comments feed

#### 6. Delete Issue (Soft Delete)

```http
DELETE /issues/{issueId}
Authorization: Bearer {jwt_token}

Response (200 OK):
{
  "message": "Issue deleted successfully"
}
```

**Behavior:**
- Soft delete: `is_deleted` set to `TRUE`
- Issue remains in database but excluded from queries
- Attachments and comments remain but are orphaned

---

### Comment Operations

#### 7. Create Comment

```http
POST /issues/{issueId}/comments
Content-Type: application/json
Authorization: Bearer {jwt_token}

{
  "comment": "Inspected the crack today. Structural engineer recommends immediate repair.",
  "attachment_ids": [15, 16]
}

Response (201 Created):
{
  "id": 5,
  "issue_id": 72,
  "comment": "Inspected the crack today. Structural engineer recommends immediate repair.",
  "comment_type": "comment",
  "created_by": 25,
  "created_by_name": "John Doe",
  "created_at": "2025-10-07T15:00:00Z",
  "attachments": [
    {
      "id": 15,
      "comment_id": 5,
      "file_name": "inspection_photo.jpg",
      "file_path": "10/24/49/comments/5/20251007150000_inspection_photo.jpg",
      "file_size": 234567,
      "file_type": "image/jpeg",
      "attachment_type": "photo",
      "uploaded_by": 25,
      "created_at": "2025-10-07T14:58:00Z"
    }
  ]
}
```

**With Attachments Workflow:**
1. Pre-upload attachments: `POST /attachments/upload-url` with `entity_type: "issue_comment"`, `entity_id: 0`
2. Upload files to S3 using presigned URLs
3. Create comment with `attachment_ids: [15, 16]`
4. System links attachments to comment by updating `comment_id` in attachment records

#### 8. Get Issue Comments

```http
GET /issues/{issueId}/comments
Authorization: Bearer {jwt_token}

Response (200 OK):
[
  {
    "id": 1,
    "issue_id": 72,
    "comment": "Initial inspection completed",
    "comment_type": "comment",
    "created_by": 19,
    "created_by_name": "Jane Smith",
    "created_at": "2025-10-06T20:30:00Z",
    "attachments": []
  },
  {
    "id": 2,
    "issue_id": 72,
    "comment": "Status changed from open to in_progress",
    "comment_type": "activity",
    "previous_value": "open",
    "new_value": "in_progress",
    "created_by": 25,
    "created_by_name": "John Doe",
    "created_at": "2025-10-07T10:00:00Z",
    "attachments": []
  },
  {
    "id": 3,
    "issue_id": 72,
    "comment": "Repair work started",
    "comment_type": "comment",
    "created_by": 25,
    "created_by_name": "John Doe",
    "created_at": "2025-10-07T11:30:00Z",
    "attachments": [
      {
        "id": 10,
        "comment_id": 3,
        "file_name": "repair_progress.jpg",
        "file_type": "image/jpeg",
        "created_at": "2025-10-07T11:28:00Z"
      }
    ]
  }
]
```

**Comment Types:**
- `comment`: User-created comments
- `activity`: System-generated activity logs (status changes, assignments)

**Ordering:** Comments returned in chronological order (oldest first)

---

## Auto-Numbering System

### Issue Number Format

```
{PROJECT_CODE}-{CATEGORY_PREFIX}-NNNN

Examples:
- PRJ-DE-0001 = Project "PRJ", Category "Deficiency", Issue #1
- BLDG-SA-0023 = Project "BLDG", Category "Safety", Issue #23
- 49-QU-0005 = Project ID 49, Category "Quality", Issue #5
```

### Generation Logic

```go
func (dao *IssueDao) generateIssueNumber(ctx context.Context, projectID int64, category string) (string, error) {
    // 1. Get project code (project_number or fallback to "PRJ-{id}")
    var projectCode string
    err := dao.DB.QueryRowContext(ctx, `
        SELECT COALESCE(project_number, 'PRJ-' || id)
        FROM project.projects
        WHERE id = $1
    `, projectID).Scan(&projectCode)

    // 2. Get sequential count for this project + category
    var count int
    categoryPrefix := strings.ToUpper(string(category[0:2]))
    err = dao.DB.QueryRowContext(ctx, `
        SELECT COUNT(*) + 1
        FROM project.issues
        WHERE project_id = $1 AND category = $2
    `, projectID, category).Scan(&count)

    // 3. Format: PROJECT-CA-0001
    return fmt.Sprintf("%s-%s-%04d", projectCode, categoryPrefix, count), nil
}
```

**Uniqueness:**
- Issue numbers are **globally unique** (enforced by unique constraint)
- Sequential per project per category
- Category prefix uses first 2 letters of category (e.g., "DE" for "deficiency")

**Examples by Category:**
- `deficiency` → `DE`
- `safety` → `SA`
- `quality` → `QU`
- `design` → `DE`
- `general` → `GE`

---

## Workflow & Status Transitions

### Issue Lifecycle

```
┌─────────────────────────────────────────────────┐
│ 1. CREATION (status: open)                      │
│    - Issue reported                             │
│    - Auto-assigned issue number                 │
│    - Assigned to responsible party              │
└────────────────┬────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────┐
│ 2. IN PROGRESS (status: in_progress)            │
│    - Work started                               │
│    - Photos and updates added                   │
│    - Comments for collaboration                 │
└────────────────┬────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────┐
│ 3. READY FOR REVIEW (status: ready_for_review)  │
│    - Work completed                             │
│    - After photos uploaded                      │
│    - Awaiting inspection                        │
└────────────────┬────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────┐
│ 4. CLOSED (status: closed)                      │
│    - Issue verified and accepted                │
│    - closed_date timestamp set                  │
│    - Removed from active issues list            │
└─────────────────────────────────────────────────┘

Alternative Paths:
- ON HOLD: Temporarily paused (waiting for materials, etc.)
- REJECTED: Issue determined invalid or duplicate
```

### Status Transition Rules

**Valid Transitions:**

| From | To | Trigger |
|------|----|----|
| `open` | `in_progress` | Work started |
| `open` | `on_hold` | Blocked/waiting |
| `open` | `rejected` | Invalid issue |
| `in_progress` | `ready_for_review` | Work completed |
| `in_progress` | `on_hold` | Work paused |
| `ready_for_review` | `closed` | Inspection passed |
| `ready_for_review` | `in_progress` | Rework needed |
| `on_hold` | `in_progress` | Resumed |
| `on_hold` | `closed` | Cancelled/resolved |

**Activity Logging:**

Every status change creates an activity log entry:

```json
{
  "id": 123,
  "issue_id": 72,
  "comment": "Status changed from open to in_progress",
  "comment_type": "activity",
  "previous_value": "open",
  "new_value": "in_progress",
  "created_by": 25,
  "created_at": "2025-10-07T10:00:00Z"
}
```

---

## Access Control

### Organization-Based Access

All issue operations validate organization membership:

```
User JWT → org_id
    ↓
Issue → project_id → Project → org_id
    ↓
Validation: JWT org_id == Project org_id
```

### Access Rules

**Create Issue:**
- Project must belong to user's organization
- Assigned user must belong to same organization
- Returns 400 if assigned user doesn't exist or org mismatch

**Read Issue:**
- Issue's project must belong to user's organization
- Returns 403 if org mismatch
- Returns 404 if issue not found or deleted

**Update Issue:**
- Issue's project must belong to user's organization
- Assigned user must belong to same organization
- Returns 403 if org mismatch

**Delete Issue:**
- Issue's project must belong to user's organization
- Returns 403 if org mismatch

**Comments:**
- Issue must belong to user's organization
- All comment operations inherit issue access control

### Error Responses

```json
// Invalid project (not in user's org)
{
  "error": "Invalid project ID. Project does not belong to your organization.",
  "status": 400
}

// Invalid assigned user (not in user's org)
{
  "error": "Invalid assigned_to user ID. User 25 does not belong to your organization.",
  "status": 400
}

// Issue not found or no access
{
  "error": "Issue does not belong to your organization",
  "status": 403
}
```

---

## Repository Methods

**File:** `/Users/mayur/git_personal/infrastructure/src/lib/data/issue_repository.go`

### Interface Definition

```go
type IssueRepository interface {
    CreateIssue(ctx context.Context, projectID, userID, orgID int64, issue *models.CreateIssueRequest) (*models.IssueResponse, error)
    GetIssueByID(ctx context.Context, issueID int64) (*models.IssueResponse, error)
    GetIssuesByProject(ctx context.Context, projectID int64, filters map[string]string) ([]models.IssueResponse, error)
    UpdateIssue(ctx context.Context, issueID, userID, orgID int64, updateReq *models.UpdateIssueRequest) (*models.IssueResponse, error)
    DeleteIssue(ctx context.Context, issueID, userID int64) error
    GetIssueAttachments(ctx context.Context, issueID int64) ([]models.IssueAttachment, error)
    UpdateIssueStatus(ctx context.Context, issueID, userID int64, status string) error
    CreateComment(ctx context.Context, issueID, userID int64, req *models.CreateCommentRequest) (*models.IssueComment, error)
    GetIssueComments(ctx context.Context, issueID int64) ([]models.IssueComment, error)
    CreateActivityLog(ctx context.Context, issueID, userID int64, activityMsg, previousValue, newValue string) error
}
```

### Key Methods

#### CreateIssue

```go
func (dao *IssueDao) CreateIssue(ctx context.Context, projectID, userID, orgID int64, req *models.CreateIssueRequest) (*models.IssueResponse, error) {
    // 1. Validate project belongs to org
    var projectOrgID int64
    err := dao.DB.QueryRowContext(ctx, `
        SELECT org_id FROM project.projects
        WHERE id = $1 AND is_deleted = FALSE
    `, projectID).Scan(&projectOrgID)

    if projectOrgID != orgID {
        return nil, fmt.Errorf("project does not belong to your organization")
    }

    // 2. Start transaction
    tx, err := dao.DB.BeginTx(ctx, nil)
    defer tx.Rollback()

    // 3. Generate issue number
    issueNumber, err := dao.generateIssueNumber(ctx, projectID, req.Category)

    // 4. Insert issue
    query := `
        INSERT INTO project.issues (
            project_id, issue_number, title, description, category,
            priority, severity, status, assigned_to, due_date,
            reported_by, created_by, updated_by
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
        RETURNING id, created_at, updated_at
    `

    // 5. Commit transaction
    tx.Commit()

    return issueResponse, nil
}
```

#### CreateComment

```go
func (dao *IssueDao) CreateComment(ctx context.Context, issueID, userID int64, req *models.CreateCommentRequest) (*models.IssueComment, error) {
    // 1. Insert comment
    query := `
        INSERT INTO project.issue_comments (
            issue_id, comment, comment_type, created_by, updated_by
        ) VALUES ($1, $2, 'comment', $3, $4)
        RETURNING id, created_at, updated_at
    `

    var commentID int64
    err := dao.DB.QueryRowContext(ctx, query, issueID, req.Comment, userID, userID).
        Scan(&commentID, &createdAt, &updatedAt)

    // 2. Link attachments (if provided)
    if len(req.AttachmentIDs) > 0 {
        for _, attachmentID := range req.AttachmentIDs {
            _, err := dao.DB.ExecContext(ctx, `
                UPDATE project.issue_comment_attachments
                SET comment_id = $1, updated_by = $2, updated_at = $3
                WHERE id = $4 AND comment_id IS NULL
            `, commentID, userID, time.Now(), attachmentID)
        }
    }

    return comment, nil
}
```

#### CreateActivityLog

```go
func (dao *IssueDao) CreateActivityLog(ctx context.Context, issueID, userID int64, activityMsg, previousValue, newValue string) error {
    _, err := dao.DB.ExecContext(ctx, `
        INSERT INTO project.issue_comments (
            issue_id, comment, comment_type, previous_value, new_value,
            created_by, updated_by
        ) VALUES ($1, $2, 'activity', $3, $4, $5, $6)
    `, issueID, activityMsg, previousValue, newValue, userID, userID)

    return err
}
```

---

## Testing

### Postman Collection

**File:** `/Users/mayur/git_personal/infrastructure/postman/IssueManagement.postman_collection.json`

**Collection includes:**
- Create issue
- Get issue by ID
- List issues for project
- Update issue
- Update issue status
- Delete issue
- Create comment
- Create comment with attachments
- Get issue comments
- Activity log verification

### Test Scripts

#### test-issue-comments.sh

**File:** `/Users/mayur/git_personal/infrastructure/testing/api/test-issue-comments.sh`

Comprehensive test covering:

```bash
#!/bin/bash

# 1. Get authentication token
TOKEN=$(curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" ...)

# 2. Verify issue exists
curl -s -X GET "$API_BASE/issues/$ISSUE_ID" -H "Authorization: Bearer $TOKEN"

# 3. Create comment WITHOUT attachments
curl -s -X POST "$API_BASE/issues/$ISSUE_ID/comments" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"comment": "Testing comment functionality"}'

# 4. Upload attachment for comment (entity_type=issue_comment, entity_id=0)
curl -s -X POST "$API_BASE/attachments/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"entity_type":"issue_comment","entity_id":0,"project_id":11,...}'

# 5. Create comment WITH attachment
curl -s -X POST "$API_BASE/issues/$ISSUE_ID/comments" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"comment":"This comment has an attachment","attachment_ids":[15]}'

# 6. Get all comments
curl -s -X GET "$API_BASE/issues/$ISSUE_ID/comments" \
  -H "Authorization: Bearer $TOKEN"

# 7. Change issue status (triggers activity log)
curl -s -X PATCH "$API_BASE/issues/$ISSUE_ID/status" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"status":"in_progress"}'

# 8. Verify activity log was created
curl -s -X GET "$API_BASE/issues/$ISSUE_ID/comments" \
  -H "Authorization: Bearer $TOKEN" | jq '[.[] | select(.comment_type == "activity")]'
```

**Test Output:**
```
✅ Token obtained
✅ Issue found with status: open
✅ Comment created with ID: 45
✅ Attachment record created with ID: 15
✅ Comment with attachment created with ID: 46
✅ Retrieved 2 comments
✅ Status updated to: in_progress
✅ Found 1 activity log entries
Activity Logs:
- Status changed from open to in_progress (by user 19)
```

---

## Integration with Other Systems

### Attachments

Issues integrate with the centralized attachment system:

```
Issue 72 → issue_attachments table → S3 path: 10/24/49/issues/72/
```

**See:** [Attachment Management Documentation](./attachment-management.md)

### Comments with Attachments

Comment attachments use pre-upload pattern:

```
1. Upload attachment (entity_type=issue_comment, entity_id=0)
2. Create comment with attachment_ids
3. System links attachments to comment
```

### RFIs & Submittals

Issues can reference RFIs and submittals (future enhancement):

```json
{
  "issue_id": 72,
  "related_rfi_id": 45,
  "related_submittal_id": 23
}
```

---

## Best Practices

### 1. Issue Creation

```
✅ DO: Provide detailed descriptions
✅ DO: Assign to specific responsible party
✅ DO: Set realistic due dates
✅ DO: Use proper categories and priorities
✅ DO: Add location information

❌ DON'T: Create duplicate issues
❌ DON'T: Use vague titles
❌ DON'T: Skip assignment
```

### 2. Comments & Updates

```
✅ DO: Add comments for important updates
✅ DO: Attach photos for visual documentation
✅ DO: Update status as work progresses

❌ DON'T: Change status without adding context
❌ DON'T: Skip documentation of completed work
```

### 3. Attachments

```
✅ DO: Add before/progress/after photos
✅ DO: Use descriptive file names
✅ DO: Upload relevant documentation

❌ DON'T: Upload unrelated files
❌ DON'T: Exceed 100MB per file
```

### 4. Status Management

```
✅ DO: Follow proper workflow (open → in_progress → closed)
✅ DO: Use "ready_for_review" before closing
✅ DO: Document closure with final photos

❌ DON'T: Jump directly to closed
❌ DON'T: Leave issues in limbo
```

---

## Code References

**Models:** `/Users/mayur/git_personal/infrastructure/src/lib/models/issue.go`
**API Handler:** `/Users/mayur/git_personal/infrastructure/src/infrastructure-issue-management/main.go`
**Repository:** `/Users/mayur/git_personal/infrastructure/src/lib/data/issue_repository.go`
**Postman:** `/Users/mayur/git_personal/infrastructure/postman/IssueManagement.postman_collection.json`
**Tests:** `/Users/mayur/git_personal/infrastructure/testing/api/test-issue-comments.sh`

---

## Summary

The Issue Management system provides **comprehensive tracking** for construction deficiencies, punch list items, and project issues. Key features:

1. **Auto-Generated Numbering:** Unique, sequential issue numbers per project and category
2. **Flexible Workflow:** Customizable status transitions with activity logging
3. **Rich Commenting:** Comments with attachments and activity logs in unified feed
4. **Complete Audit Trail:** All changes tracked with timestamps and user attribution
5. **Attachment Integration:** Seamless integration with centralized attachment system
6. **Access Control:** Organization-based validation for security
7. **Location Tracking:** Multiple location formats (building/level/room, coordinates, GPS)
8. **Assignment Management:** User and company assignment with validation

This system enables construction teams to effectively track, manage, and resolve issues throughout the project lifecycle.