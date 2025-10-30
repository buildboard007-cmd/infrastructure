# Project Management

## Overview

The Project Management system provides comprehensive project lifecycle management for construction projects. It handles project creation, tracking, team assignments, and access control based on hierarchical user permissions.

**Key Features:**
- Auto-generated project numbers with PROJ-YYYY-NNNN format
- Nested request structure for better organization
- Access control based on user assignment context (organization/location/project levels)
- Project team management with role-based assignments
- Comprehensive project metadata including timeline, financial, and location information
- Attachment management integration

---

## Database Schema

### Table: `project.projects`

**Primary Table:** Stores all project information including basic details, classification, timeline, financial data, and location information.

| Column | Type | Required | Default | Description |
|--------|------|----------|---------|-------------|
| `id` | bigint | Yes | Auto-increment | Primary key |
| `org_id` | bigint | Yes | - | Organization ID (FK to iam.organizations) |
| `location_id` | bigint | Yes | - | Location ID (FK to iam.locations) |
| `project_number` | varchar(50) | No | - | Auto-generated (PROJ-YYYY-NNNN) |
| `name` | varchar(255) | Yes | - | Project name |
| `description` | text | No | - | Project description |
| `project_type` | varchar(50) | Yes | - | Project type/sector |
| `project_stage` | varchar(50) | No | - | Stage: bidding, pre-construction, course-of-construction, post-construction, warranty |
| `work_scope` | varchar(50) | No | - | Scope: new, renovation, restoration, maintenance |
| `project_sector` | varchar(50) | No | - | Sector: commercial, residential, industrial, hospitality, healthcare, institutional, etc. |
| `delivery_method` | varchar(50) | No | - | Method: design-build, design-bid-build, CM-at-risk, IPD, etc. |
| `project_phase` | varchar(50) | Yes | 'pre_construction' | Current phase |
| `start_date` | date | No | - | Project start date |
| `planned_end_date` | date | No | - | Planned end date |
| `actual_start_date` | date | No | - | Actual start date |
| `actual_end_date` | date | No | - | Actual end date |
| `substantial_completion_date` | date | No | - | Substantial completion date |
| `project_finish_date` | date | No | - | Project finish date |
| `warranty_start_date` | date | No | - | Warranty start date |
| `warranty_end_date` | date | No | - | Warranty end date |
| `budget` | numeric | No | - | Project budget |
| `contract_value` | numeric | No | - | Contract value |
| `square_footage` | integer | No | - | Building square footage |
| `address` | text | No | - | Project address |
| `city` | varchar(100) | No | - | City |
| `state` | varchar(50) | No | - | State/Province |
| `zip_code` | varchar(20) | No | - | Postal code |
| `country` | varchar(100) | No | 'USA' | Country |
| `language` | varchar(10) | No | 'en' | Language code |
| `latitude` | numeric | No | - | GPS latitude |
| `longitude` | numeric | No | - | GPS longitude |
| `status` | varchar(50) | Yes | 'active' | Status: active, inactive, on_hold, completed, cancelled |
| `created_at` | timestamp | Yes | CURRENT_TIMESTAMP | Creation timestamp |
| `created_by` | bigint | Yes | - | Creator user ID |
| `updated_at` | timestamp | Yes | CURRENT_TIMESTAMP | Last update timestamp |
| `updated_by` | bigint | Yes | - | Last updater user ID |
| `is_deleted` | boolean | Yes | false | Soft delete flag |

### Related Tables

**`project.project_user_roles`** - User assignments to projects (deprecated in favor of unified assignments table)
**`project.project_attachments`** - Project attachments (logo, photos, documents)

---

## Data Models

### Core Models

**Location:** `/Users/mayur/git_personal/infrastructure/src/lib/models/project.go`

```go
type Project struct {
    ProjectID                 int64          `json:"project_id"`
    OrgID                     int64          `json:"org_id"`
    LocationID                int64          `json:"location_id"`
    ProjectNumber             sql.NullString `json:"project_number,omitempty"`
    Name                      string         `json:"name"`
    Description               sql.NullString `json:"description,omitempty"`
    ProjectType               string         `json:"project_type"`
    ProjectStage              sql.NullString `json:"project_stage,omitempty"`
    WorkScope                 sql.NullString `json:"work_scope,omitempty"`
    ProjectSector             sql.NullString `json:"project_sector,omitempty"`
    DeliveryMethod            sql.NullString `json:"delivery_method,omitempty"`
    ProjectPhase              string         `json:"project_phase"`
    StartDate                 sql.NullTime   `json:"start_date,omitempty"`
    PlannedEndDate            sql.NullTime   `json:"planned_end_date,omitempty"`
    // ... (additional timeline fields)
    Budget                    sql.NullFloat64 `json:"budget,omitempty"`
    ContractValue             sql.NullFloat64 `json:"contract_value,omitempty"`
    SquareFootage             sql.NullInt64   `json:"square_footage,omitempty"`
    Address                   sql.NullString  `json:"address,omitempty"`
    City                      sql.NullString  `json:"city,omitempty"`
    State                     sql.NullString  `json:"state,omitempty"`
    ZipCode                   sql.NullString  `json:"zip_code,omitempty"`
    Country                   string          `json:"country"`
    Language                  string          `json:"language"`
    Status                    string          `json:"status"`
    CreatedAt                 time.Time       `json:"created_at"`
    CreatedBy                 int64           `json:"created_by"`
    UpdatedAt                 time.Time       `json:"updated_at"`
    UpdatedBy                 int64           `json:"updated_by"`
}
```

### Request Models

**Unified Nested Structure** (Used for both Create and Update):

```go
type CreateProjectRequest struct {
    LocationID     int64          `json:"location_id"`
    BasicInfo      BasicInfo      `json:"basic_info"`
    ProjectDetails ProjectDetails `json:"project_details"`
    Location       LocationInfo   `json:"location"`
    Timeline       Timeline       `json:"timeline"`
    Financial      Financial      `json:"financial"`
    Attachments    Attachments    `json:"attachments,omitempty"`
}

type BasicInfo struct {
    Name        string `json:"name" binding:"required,max=255"`
    Description string `json:"description,omitempty"`
}

type ProjectDetails struct {
    ProjectStage   string `json:"project_stage"`   // bidding, pre-construction, course-of-construction, etc.
    WorkScope      string `json:"work_scope"`      // new, renovation, restoration, maintenance
    ProjectSector  string `json:"project_sector"`  // commercial, residential, industrial, etc.
    DeliveryMethod string `json:"delivery_method"` // design-build, design-bid-build, CM-at-risk, etc.
    SquareFootage  int64  `json:"square_footage,omitempty"`
    Language       string `json:"language,omitempty"`
    Status         string `json:"status,omitempty"`
}

type LocationInfo struct {
    Address string `json:"address"`
    City    string `json:"city,omitempty"`
    State   string `json:"state,omitempty"`
    ZipCode string `json:"zip_code,omitempty"`
    Country string `json:"country,omitempty"`
}

type Timeline struct {
    StartDate                 string `json:"start_date"` // YYYY-MM-DD
    SubstantialCompletionDate string `json:"substantial_completion_date,omitempty"`
    ProjectFinishDate         string `json:"project_finish_date,omitempty"`
    WarrantyStartDate         string `json:"warranty_start_date,omitempty"`
    WarrantyEndDate           string `json:"warranty_end_date,omitempty"`
}

type Financial struct {
    Budget float64 `json:"budget,omitempty"`
}
```

---

## API Endpoints

**Service Location:** `/Users/mayur/git_personal/infrastructure/src/infrastructure-project-management/main.go`

### 1. Create Project
**POST** `/projects`

Creates a new project with auto-generated project number.

**Request Body:**
```json
{
    "location_id": 1,
    "basic_info": {
        "name": "Downtown Office Complex",
        "description": "A modern office complex with sustainable design"
    },
    "project_details": {
        "project_stage": "pre-construction",
        "work_scope": "new",
        "project_sector": "commercial",
        "delivery_method": "design-build",
        "square_footage": 50000,
        "language": "en",
        "status": "active"
    },
    "location": {
        "address": "123 Business District, Downtown, NY 10001",
        "city": "New York",
        "state": "NY",
        "zip_code": "10001",
        "country": "USA"
    },
    "timeline": {
        "start_date": "2025-09-15",
        "substantial_completion_date": "2026-12-01",
        "project_finish_date": "2027-01-15"
    },
    "financial": {
        "budget": 5000000.00
    }
}
```

**Response (201 Created):**
```json
{
    "success": true,
    "message": "Project created successfully",
    "data": {
        "project_id": "42",
        "project_number": "PROJ-2025-0001",
        "name": "Downtown Office Complex",
        "status": "active",
        "created_at": "2025-01-15T10:30:00Z",
        "created_by": 5
    }
}
```

### 2. Get Projects (with Access Control)
**GET** `/projects` or **GET** `/projects?location_id={location_id}`

Retrieves projects based on user access level:
- **Super Admin**: Sees ALL projects in organization
- **Org-level Assignment**: Sees ALL projects
- **Location-level Assignment**: Sees projects at assigned locations only
- **Project-level Assignment**: Sees ONLY assigned projects
- **No Assignment**: Returns empty list

**Query Parameters:**
- `location_id` (optional): Filter projects by location

**Response (200 OK):**
```json
{
    "projects": [
        {
            "project_id": 42,
            "project_number": "PROJ-2025-0001",
            "name": "Downtown Office Complex",
            "description": "A modern office complex",
            "project_stage": "pre-construction",
            "status": "active",
            "budget": 5000000.00,
            "start_date": "2025-09-15T00:00:00Z",
            "created_at": "2025-01-15T10:30:00Z"
        }
    ],
    "total": 1
}
```

### 3. Get Project by ID
**GET** `/projects/{projectId}`

Retrieves detailed information for a specific project.

**Response (200 OK):**
Returns full project object with all fields.

### 4. Update Project
**PUT** `/projects/{projectId}`

Updates project information using the same nested structure as create.

**Request Body:** Same structure as Create Project (partial updates supported)

**Response (200 OK):**
Returns updated project object.

### 5. Assign User to Project
**POST** `/projects/{projectId}/users`

Creates a unified assignment for a user to the project.

**Request Body:**
```json
{
    "user_id": 10,
    "role_id": 3,
    "trade_type": "electrical",
    "is_primary": true,
    "start_date": "2024-06-01",
    "end_date": "2025-12-31"
}
```

**Response (201 Created):**
Returns assignment object from unified assignments table.

### 6. Get Project Team
**GET** `/projects/{projectId}/users`

Lists all user assignments for the project.

**Response (200 OK):**
Returns array of assignment objects.

### 7. Update User Assignment
**PUT** `/projects/{projectId}/users/{assignmentId}`

Updates a user's project assignment.

### 8. Remove User from Project
**DELETE** `/projects/{projectId}/users/{assignmentId}`

Soft deletes user assignment.

---

## Repository Methods

**Location:** `/Users/mayur/git_personal/infrastructure/src/lib/data/project_repository.go`

### Key Methods

```go
// Project CRUD
CreateProject(ctx, orgID, project, userID) (*CreateProjectResponse, error)
GetProjectsByOrg(ctx, orgID) ([]Project, error)
GetProjectsByLocationID(ctx, locationID, orgID) ([]Project, error)
GetProjectsByIDs(ctx, projectIDs, orgID) ([]Project, error)
GetProjectByID(ctx, projectID, orgID) (*Project, error)
UpdateProject(ctx, projectID, orgID, project, userID) (*Project, error)

// Project Team
AssignUserToProject(ctx, projectID, assignment, userID) (*ProjectUserRole, error)
GetProjectUserRoles(ctx, projectID) ([]ProjectUserRole, error)
UpdateProjectUserRole(ctx, assignmentID, projectID, assignment, userID) (*ProjectUserRole, error)
RemoveUserFromProject(ctx, assignmentID, projectID, userID) error

// Attachments
CreateProjectAttachment(ctx, projectID, attachment, userID) (*ProjectAttachment, error)
GetProjectAttachmentsByProject(ctx, projectID) ([]ProjectAttachment, error)
GetProjectAttachmentByID(ctx, attachmentID, projectID) (*ProjectAttachment, error)
DeleteProjectAttachment(ctx, attachmentID, projectID, userID) error
```

### Auto-Numbering Logic

Projects use an auto-numbering system with the format **PROJ-YYYY-NNNN**:

```go
func generateProjectNumber(ctx, orgID) (string, error) {
    currentYear := time.Now().Year()

    // Find next available number for this year
    query := `
        SELECT COALESCE(MAX(CAST(SUBSTRING(project_number, 11) AS INTEGER)), 0) + 1
        FROM project.projects
        WHERE org_id = $1 AND project_number LIKE $2
    `

    yearPrefix := fmt.Sprintf("PROJ-%d-%%", currentYear)
    // Execute query and get nextNum

    return fmt.Sprintf("PROJ-%d-%04d", currentYear, nextNum), nil
}
```

**Example:** PROJ-2025-0001, PROJ-2025-0002, etc.

---

## Access Control

The GET `/projects` endpoint implements hierarchical access control based on the unified `iam.user_assignments` table:

### Access Levels

1. **Super Admin** (from JWT claims)
   - Sees all projects in organization
   - No filtering applied

2. **Organization-level Assignment** (`context_type='organization'`)
   - Query: `SELECT * FROM iam.user_assignments WHERE user_id=? AND context_type='organization' AND context_id=org_id`
   - Sees all projects in organization

3. **Location-level Assignment** (`context_type='location'`)
   - Query: `SELECT * FROM iam.user_assignments WHERE user_id=? AND context_type='location'`
   - Sees projects at assigned locations only
   - With location_id filter: Returns 403 if user doesn't have access to that location

4. **Project-level Assignment** (`context_type='project'`)
   - Query: `SELECT * FROM iam.user_assignments WHERE user_id=? AND context_type='project'`
   - Sees only assigned projects

5. **No Assignment**
   - Returns empty array

### Implementation Flow

```go
func handleGetProjects(ctx, request, claims) (response, error) {
    if claims.IsSuperAdmin {
        // Return all projects
        return projectRepository.GetProjectsByOrg(ctx, orgID)
    }

    // Check org-level assignments
    orgContexts := assignmentRepository.GetUserContexts(ctx, userID, "organization", orgID)
    if len(orgContexts) > 0 {
        return projectRepository.GetProjectsByOrg(ctx, orgID)
    }

    // Check location-level assignments
    locationContexts := assignmentRepository.GetUserContexts(ctx, userID, "location", orgID)
    if len(locationContexts) > 0 {
        if hasLocationFilter {
            // Verify user has access to filtered location
            if !userHasAccessToLocation(locationID, locationContexts) {
                return 403 Forbidden
            }
            return projectRepository.GetProjectsByLocationID(ctx, locationID, orgID)
        }
        return []  // Force location selection for location-level users
    }

    // Check project-level assignments
    projectContexts := assignmentRepository.GetUserContexts(ctx, userID, "project", orgID)
    if len(projectContexts) > 0 {
        return projectRepository.GetProjectsByIDs(ctx, projectContexts, orgID)
    }

    return []  // No access
}
```

---

## Postman Collection

**Location:** `/Users/mayur/git_personal/infrastructure/postman/ProjectManagement.postman_collection.json`

**Available Requests:**
1. Create Project (Nested Structure)
2. Get Projects (with optional location filter)
3. Get All Projects
4. Get Project by ID
5. Update Project (Unified Nested Structure)
6. Create Project Attachment
7. Get Project Attachments
8. Get Project Attachment by ID
9. Delete Project Attachment
10. Assign User to Project
11. Get Project User Roles
12. Update Project User Role
13. Remove User from Project

**Environment Variables:**
- `access_token`: JWT ID token (not access token!)
- `project_id`: Auto-populated after creation
- `location_id`: Must be set before creating projects
- `user_id`: For user assignments
- `role_id`: For role assignments
- `assignment_id`: Auto-populated after assignment

---

## Testing

### Test Scripts

**Location:** `/Users/mayur/git_personal/infrastructure/testing/api/test-get-projects-access-control.sh`

**Purpose:** Validates the access control logic for GET /projects endpoint.

**Test Cases:**
1. GET /projects without filter - Verifies user sees appropriate projects based on access level
2. GET /projects with location_id filter - Verifies location filtering respects user access
3. Response structure validation

**Test User:**
- Email: `buildboard007+555@gmail.com`
- Password: `Mayur@1234`

**Usage:**
```bash
cd /Users/mayur/git_personal/infrastructure/testing/api
./test-get-projects-access-control.sh
```

### Additional Test Files

**Location:** `/Users/mayur/git_personal/infrastructure/testing/api/`
- `test-project-user-management.sh` - Tests project team management

---

## Best Practices

### Date Handling
- Always use YYYY-MM-DD format for dates
- For historical projects, past start dates are allowed
- Validation ensures completion dates are after start dates

### Status Values
- **Project Stage:** bidding, pre-construction, course-of-construction, post-construction, warranty
- **Work Scope:** new, renovation, restoration, maintenance
- **Project Sector:** commercial, residential, industrial, hospitality, healthcare, institutional, mixed-use, civil-infrastructure, recreation, aviation, specialized
- **Delivery Method:** design-build, design-bid-build, construction-manager-at-risk, integrated-project-delivery, construction-manager-as-agent, public-private-partnership, other
- **Status:** active, inactive, on_hold, completed, cancelled

### Language Mapping
The API accepts both language codes and full names:
- "English" or "en" → "en"
- "Spanish" or "es" → "es"
- "French" or "fr" → "fr"

### Organization Validation
- Project creation validates that location_id belongs to the user's organization
- Returns validation error if location doesn't exist or doesn't belong to org

### Audit Trail
All projects track:
- `created_by` / `created_at` - Original creator and timestamp
- `updated_by` / `updated_at` - Last modifier and timestamp
- `is_deleted` - Soft delete flag (never hard delete)

---

## Related Documentation

- [Assignment Management](./assignment-management.md) - User assignment system
- [RFI Management](./rfi-management.md) - Request for Information workflow
- [Submittal Management](./submittal-management.md) - Submittal workflow
- [Issue Management](./issue-management.md) - Issue tracking system

---

## Implementation Notes

1. **Unified Assignments:** Project team management now uses the unified `iam.user_assignments` table instead of `project.project_user_roles`
2. **Attachment Management:** Project attachments may be migrated to centralized attachment service
3. **Access Control:** The hierarchical access control system is critical for multi-tenant security
4. **Transaction Safety:** Project creation uses database transactions to ensure atomicity
5. **Auto-numbering:** Project numbers reset annually (e.g., PROJ-2025-0001, PROJ-2026-0001)