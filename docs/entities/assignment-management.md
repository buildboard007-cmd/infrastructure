# Assignment Management Entity Documentation

## 1. Overview

**Assignment Management is THE CORE of the entire access control system.** It is the foundation that determines who has access to what resources in the BuildBoard construction management platform.

The assignment system provides a unified approach to managing user access across all contexts (organizations, projects, locations, departments, equipment, phases). It replaces the deprecated tables (`org_user_roles`, `location_user_roles`, `project_user_roles`, `project_managers`) with a single, flexible `user_assignments` table.

### Key Concepts

- **User Assignment**: Links a user to a specific context (organization, project, location, etc.) with a specific role
- **Context Types**: The type of resource being assigned (organization, project, location, department, equipment, phase)
- **Context ID**: The specific ID of the resource
- **Role**: The role the user has in that context (e.g., Project Manager, Field Engineer, Safety Inspector)
- **Primary Assignment**: Indicates the user's primary/main assignment in that context type
- **Active Period**: Optional start/end dates for time-bound assignments

### Why It's Critical

Assignment management controls:
- User access to projects, locations, and organizations
- Permission inheritance and scoping
- Team membership and collaboration
- Filtering of data (users only see data for their assigned contexts)
- Audit trails (who had access when)

## 2. Database Schema

Table: `iam.user_assignments`

```sql
CREATE TABLE iam.user_assignments (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL,                    -- FK to iam.users
    role_id         BIGINT NOT NULL,                    -- FK to iam.roles
    context_type    VARCHAR(20) NOT NULL,               -- 'organization', 'project', 'location', 'department', 'equipment', 'phase'
    context_id      BIGINT NOT NULL,                    -- ID of the context (project ID, location ID, etc.)
    trade_type      VARCHAR(255),                       -- Optional: 'electrical', 'plumbing', 'hvac', etc.
    is_primary      BOOLEAN DEFAULT FALSE,              -- Is this the user's primary assignment in this context type?
    start_date      DATE,                               -- Optional: When the assignment becomes active
    end_date        DATE,                               -- Optional: When the assignment expires
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by      BIGINT NOT NULL,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by      BIGINT NOT NULL,
    is_deleted      BOOLEAN NOT NULL DEFAULT FALSE
);

-- Indexes
CREATE INDEX idx_user_assignments_user_id ON iam.user_assignments(user_id);
CREATE INDEX idx_user_assignments_context ON iam.user_assignments(context_type, context_id);
CREATE INDEX idx_user_assignments_role_id ON iam.user_assignments(role_id);
CREATE INDEX idx_user_assignments_is_deleted ON iam.user_assignments(is_deleted);
```

### Foreign Key Relationships

- `user_id` → `iam.users.id`
- `role_id` → `iam.roles.id`
- `context_id` → Varies based on `context_type`:
  - `organization` → `iam.organizations.id`
  - `project` → `project.projects.id`
  - `location` → `iam.locations.id`

## 3. Data Models

File: `/Users/mayur/git_personal/infrastructure/src/lib/models/assignment.go`

### UserAssignment

```go
type UserAssignment struct {
    ID          int64          `json:"id"`
    UserID      int64          `json:"user_id"`
    RoleID      int64          `json:"role_id"`
    ContextType string         `json:"context_type"` // "organization", "project", "location", "department", etc.
    ContextID   int64          `json:"context_id"`   // ID of the context
    TradeType   sql.NullString `json:"trade_type,omitempty"`
    IsPrimary   bool           `json:"is_primary"`
    StartDate   sql.NullTime   `json:"start_date,omitempty"`
    EndDate     sql.NullTime   `json:"end_date,omitempty"`
    CreatedAt   time.Time      `json:"created_at"`
    CreatedBy   int64          `json:"created_by"`
    UpdatedAt   time.Time      `json:"updated_at"`
    UpdatedBy   int64          `json:"updated_by"`
    IsDeleted   bool           `json:"is_deleted"`
}
```

### CreateAssignmentRequest

```go
type CreateAssignmentRequest struct {
    UserID      int64  `json:"user_id" binding:"required"`
    RoleID      int64  `json:"role_id" binding:"required"`
    ContextType string `json:"context_type" binding:"required,oneof=organization project location department equipment phase"`
    ContextID   int64  `json:"context_id" binding:"required"`
    TradeType   string `json:"trade_type,omitempty"`
    IsPrimary   bool   `json:"is_primary,omitempty"`
    StartDate   string `json:"start_date,omitempty"` // YYYY-MM-DD format
    EndDate     string `json:"end_date,omitempty"`   // YYYY-MM-DD format
}
```

### UpdateAssignmentRequest

```go
type UpdateAssignmentRequest struct {
    RoleID      *int64 `json:"role_id,omitempty"`
    TradeType   string `json:"trade_type,omitempty"`
    IsPrimary   *bool  `json:"is_primary,omitempty"`
    StartDate   string `json:"start_date,omitempty"`
    EndDate     string `json:"end_date,omitempty"`
}
```

### BulkAssignmentRequest

```go
type BulkAssignmentRequest struct {
    UserIDs     []int64 `json:"user_ids" binding:"required,min=1"`
    RoleID      int64   `json:"role_id" binding:"required"`
    ContextType string  `json:"context_type" binding:"required,oneof=organization project location department equipment phase"`
    ContextID   int64   `json:"context_id" binding:"required"`
    TradeType   string  `json:"trade_type,omitempty"`
    IsPrimary   bool    `json:"is_primary,omitempty"`
    StartDate   string  `json:"start_date,omitempty"`
    EndDate     string  `json:"end_date,omitempty"`
}
```

### AssignmentResponse

```go
type AssignmentResponse struct {
    ID          int64     `json:"id"`
    UserID      int64     `json:"user_id"`
    RoleID      int64     `json:"role_id"`
    ContextType string    `json:"context_type"`
    ContextID   int64     `json:"context_id"`
    TradeType   *string   `json:"trade_type,omitempty"`
    IsPrimary   bool      `json:"is_primary"`
    StartDate   *string   `json:"start_date,omitempty"`
    EndDate     *string   `json:"end_date,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    CreatedBy   int64     `json:"created_by"`
    UpdatedAt   time.Time `json:"updated_at"`
    UpdatedBy   int64     `json:"updated_by"`
    IsDeleted   bool      `json:"is_deleted"`

    // Enriched fields (joined from other tables)
    UserName    string `json:"user_name,omitempty"`
    UserEmail   string `json:"user_email,omitempty"`
    RoleName    string `json:"role_name,omitempty"`
    ContextName string `json:"context_name,omitempty"`
}
```

### Context Type Constants

```go
const (
    ContextTypeOrganization = "organization"
    ContextTypeProject      = "project"
    ContextTypeLocation     = "location"
    ContextTypeDepartment   = "department"
    ContextTypeEquipment    = "equipment"
    ContextTypePhase        = "phase"
)
```

## 4. API Endpoints

API Base URL: `https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main`

Lambda: `infrastructure-assignment-management`

### Core CRUD Operations

#### Create Assignment
```
POST /assignments
```

**Request Body:**
```json
{
    "user_id": 16,
    "role_id": 8,
    "context_type": "project",
    "context_id": 30,
    "trade_type": "electrical",
    "is_primary": true,
    "start_date": "2025-09-20",
    "end_date": "2025-12-31"
}
```

**Response:** `201 Created`
```json
{
    "id": 123,
    "user_id": 16,
    "role_id": 8,
    "context_type": "project",
    "context_id": 30,
    "trade_type": "electrical",
    "is_primary": true,
    "start_date": "2025-09-20",
    "end_date": "2025-12-31",
    "user_name": "John Doe",
    "user_email": "john@example.com",
    "role_name": "Project Manager",
    "context_name": "Downtown Office Building",
    "created_at": "2025-10-27T12:00:00Z",
    "updated_at": "2025-10-27T12:00:00Z"
}
```

#### Get Assignment by ID
```
GET /assignments/{assignmentId}
```

**Response:** `200 OK` (same structure as create response)

#### Update Assignment
```
PUT /assignments/{assignmentId}
```

**Request Body:**
```json
{
    "role_id": 9,
    "trade_type": "plumbing",
    "is_primary": false,
    "end_date": "2025-06-30"
}
```

**Response:** `200 OK` (returns updated assignment with enriched data)

#### Delete Assignment
```
DELETE /assignments/{assignmentId}
```

**Response:** `200 OK`
```json
{
    "message": "Assignment deleted successfully"
}
```

Note: This performs a soft delete (sets `is_deleted = true`)

### Context-Based Operations

#### Get Project/Location Team
```
GET /contexts/{contextType}/{contextId}/assignments
```

**Example:**
```
GET /contexts/project/30/assignments
```

**Response:** `200 OK`
```json
{
    "context_type": "project",
    "context_id": 30,
    "context_name": "Downtown Office Building",
    "org_id": 7,
    "assignments": [
        {
            "id": 123,
            "user_id": 16,
            "role_id": 8,
            "user_name": "John Doe",
            "user_email": "john@example.com",
            "role_name": "Project Manager",
            "is_primary": true,
            "trade_type": "general"
        },
        {
            "id": 124,
            "user_id": 21,
            "role_id": 9,
            "user_name": "Jane Smith",
            "user_email": "jane@example.com",
            "role_name": "Field Engineer",
            "is_primary": false,
            "trade_type": "electrical"
        }
    ]
}
```

## 5. Repository Methods

File: `/Users/mayur/git_personal/infrastructure/src/lib/data/assignment_repository.go`

### Interface Definition

```go
type AssignmentRepository interface {
    // Basic CRUD operations
    CreateAssignment(ctx context.Context, req *models.CreateAssignmentRequest, userID int64) (*models.AssignmentResponse, error)
    GetAssignment(ctx context.Context, assignmentID int64, orgID int64) (*models.AssignmentResponse, error)
    UpdateAssignment(ctx context.Context, assignmentID int64, req *models.UpdateAssignmentRequest, userID int64) (*models.AssignmentResponse, error)
    DeleteAssignment(ctx context.Context, assignmentID int64, userID int64) error

    // Bulk operations
    CreateBulkAssignments(ctx context.Context, req *models.BulkAssignmentRequest, userID int64) ([]models.AssignmentResponse, error)
    TransferAssignments(ctx context.Context, req *models.AssignmentTransferRequest, userID int64) error

    // Query operations
    GetAssignments(ctx context.Context, filters *models.AssignmentFilters, orgID int64) (*models.AssignmentListResponse, error)
    GetUserAssignments(ctx context.Context, userID int64, orgID int64) (*models.UserAssignmentSummary, error)
    GetContextAssignments(ctx context.Context, contextType string, contextID int64, orgID int64) (*models.ContextAssignmentSummary, error)

    // Permission checking - CRITICAL FOR ACCESS CONTROL
    CheckPermission(ctx context.Context, req *models.PermissionCheckRequest, orgID int64) (*models.PermissionCheckResponse, error)
    GetUserContexts(ctx context.Context, userID int64, contextType string, orgID int64) ([]int64, error)

    // Validation and utilities
    ValidateAssignmentContext(ctx context.Context, contextType string, contextID int64, orgID int64) error
    GetActiveAssignments(ctx context.Context, userID int64, orgID int64) ([]models.AssignmentResponse, error)
}
```

### Key Methods Explained

#### GetUserContexts (Critical for Access Control)

**Purpose:** Returns all context IDs of a specific type that a user has access to. This is used throughout the system to filter data.

**Example:** Get all projects a user can access
```go
projectIDs, err := assignmentRepo.GetUserContexts(ctx, userID, "project", orgID)
// Returns: [30, 45, 67, 89]
```

**SQL Query:**
```sql
SELECT DISTINCT ua.context_id
FROM iam.user_assignments ua
LEFT JOIN iam.users u ON ua.user_id = u.id
WHERE ua.user_id = $1
    AND ua.context_type = $2
    AND u.org_id = $3
    AND ua.is_deleted = FALSE
    AND (ua.start_date IS NULL OR ua.start_date <= NOW())
    AND (ua.end_date IS NULL OR ua.end_date >= NOW())
ORDER BY ua.context_id
```

**Usage in Other Services:**
This method is called in RFI, Issue, and Submittal services to filter results:
```go
// In RFI service
projectIDs, err := assignmentRepo.GetUserContexts(ctx, userID, "project", orgID)
// Then query: SELECT * FROM rfis WHERE project_id IN (projectIDs)
```

#### CreateAssignment

Creates a new assignment with validation:
1. Validates the context exists (project, location, organization)
2. Parses optional date fields
3. Inserts into database
4. Returns enriched response with user name, role name, context name

#### GetContextAssignments

Gets all users assigned to a specific context (e.g., project team):
```go
team, err := assignmentRepo.GetContextAssignments(ctx, "project", 30, orgID)
// Returns all users on project 30 with their roles
```

#### ValidateAssignmentContext

Ensures the context exists before creating an assignment:
```go
err := dao.ValidateAssignmentContext(ctx, "project", 30, orgID)
// Checks if project 30 exists and is not deleted
```

## 6. Lambda Handler

File: `/Users/mayur/git_personal/infrastructure/src/infrastructure-assignment-management/main.go`

### Request Flow

1. **Authentication**: JWT token validated via API Gateway authorizer
2. **Claims Extraction**: User ID, Org ID, Email extracted from token
3. **Route Matching**: Request routed to appropriate handler based on path and method
4. **Handler Execution**: Specific handler processes request
5. **Response**: JSON response with status code

### Handler Functions

```go
// Main request router
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

// Individual handlers
func handleCreateAssignment(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims)
func handleGetAssignment(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims)
func handleUpdateAssignment(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims)
func handleDeleteAssignment(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims)
func handleGetContextAssignments(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims)
```

### Initialization (Cold Start)

```go
func init() {
    // 1. Parse environment variables (IS_LOCAL, LOG_LEVEL)
    // 2. Initialize structured logging (logrus)
    // 3. Create SSM client for parameter store
    // 4. Retrieve database credentials from SSM
    // 5. Create PostgreSQL connection
    // 6. Initialize AssignmentRepository
}
```

## 7. Access Control

### Who Can Manage Assignments?

- **Super Admins**: Can manage all assignments in the system
- **Organization Admins**: Can manage assignments within their organization
- **Project Managers**: Can manage assignments for their specific projects (if they have appropriate permissions)

### Permission Model

Assignments themselves control permissions through:
1. **Role Assignment**: User gets a role in a specific context
2. **Role Permissions**: Role has associated permissions (from `iam.role_permissions`)
3. **Context Scoping**: Permissions only apply within the assigned context
4. **Inheritance**: Organization-level roles may inherit to projects/locations

### Example Permission Check

```
User: John (user_id: 16)
Assignment: Project Manager on Project 30
Question: Can John view RFIs on Project 30?

Check:
1. Get John's assignments: context_type='project', context_id=30
2. Get role permissions for "Project Manager"
3. Check if "rfi.read" permission exists
4. Return: Yes/No
```

## 8. Related Entities

### Direct Relationships

- **Users** (`iam.users`): The user being assigned
- **Roles** (`iam.roles`): The role granted in the assignment
- **Organizations** (`iam.organizations`): Context when context_type='organization'
- **Projects** (`project.projects`): Context when context_type='project'
- **Locations** (`iam.locations`): Context when context_type='location'

### Indirect Relationships

- **RFIs**: Filtered by user's project assignments
- **Submittals**: Filtered by user's project assignments
- **Issues**: Filtered by user's project assignments
- **Permissions**: Granted through role assignments

## 9. Common Workflows

### Workflow 1: Assign User to Project

```bash
# 1. Create assignment
POST /assignments
{
    "user_id": 16,
    "role_id": 8,
    "context_type": "project",
    "context_id": 30,
    "is_primary": true
}

# 2. Verify assignment
GET /contexts/project/30/assignments

# 3. User can now access project 30 data
```

### Workflow 2: Get User's Accessible Projects

```go
// In backend code
projectIDs, err := assignmentRepo.GetUserContexts(ctx, userID, "project", orgID)
// Use projectIDs to filter queries
```

### Workflow 3: Transfer User from One Project to Another

```bash
# Option 1: Delete old, create new
DELETE /assignments/{oldAssignmentId}
POST /assignments { ... new project ... }

# Option 2: Update assignment
PUT /assignments/{assignmentId}
{
    "context_id": 45  // New project ID
}
```

### Workflow 4: Time-Bound Assignment

```bash
# Assign contractor for 3 months
POST /assignments
{
    "user_id": 21,
    "role_id": 10,
    "context_type": "project",
    "context_id": 30,
    "start_date": "2025-11-01",
    "end_date": "2026-01-31"
}

# Assignment automatically becomes inactive after end_date
```

### Workflow 5: Bulk Assign Multiple Users to Project

```bash
POST /assignments/bulk
{
    "user_ids": [16, 21, 27, 29],
    "role_id": 9,
    "context_type": "project",
    "context_id": 30
}
```

## 10. Postman Collection

File: `/Users/mayur/git_personal/infrastructure/postman/AssignmentManagement.postman_collection.json`

### Collection Variables

```json
{
    "access_token": "",           // JWT ID Token from Cognito
    "project_id": "30",           // Default project for testing
    "user_id": "16",              // Default user for testing
    "assignment_id": ""           // Set automatically by create requests
}
```

### Test Authentication

```bash
# Test user credentials
Email: buildboard007+555@gmail.com
Password: Mayur@1234

# Valid test data
Org ID: 7
User IDs: 16, 21, 27, 29
Project IDs: 30, and others
```

### Collection Requests

1. **Create Assignment** - POST /assignments
2. **Get Assignment by ID** - GET /assignments/{id}
3. **Update Assignment** - PUT /assignments/{id}
4. **Delete Assignment** - DELETE /assignments/{id}
5. **Get Project Team** - GET /contexts/project/{id}/assignments

Each request includes:
- Pre-request scripts (if needed)
- Test scripts to validate responses
- Example request bodies
- Expected response formats

## 11. Testing

### Test Script

File: `/Users/mayur/git_personal/infrastructure/testing/api/test-project-user-management.sh`

This script tests the complete assignment workflow:

```bash
#!/bin/bash
# Run from infrastructure root
./testing/api/test-project-user-management.sh
```

**Test Steps:**
1. Get authentication token
2. Verify project exists
3. Assign user to project
4. Get project users (team list)
5. Update user role
6. Remove user from project
7. Verify user is removed (soft deleted)

**Expected Output:**
```
=== Getting Authentication Token ===
✅ Token obtained

=== Step 1: Verify Project Exists ===
✅ Project found: Maria Resort 1234

=== Step 2: Assign User to Project ===
✅ Assignment created with ID: 123

=== Step 3: Get Project Users ===
✅ Retrieved 5 users assigned to project
✅ Verified assignment 123 is in project users list

=== Step 4: Update User Role ===
✅ Assignment updated successfully (role_id: 28 → 29)

=== Step 5: Remove User from Project ===
✅ User removed from project successfully

=== Step 6: Verify User is Removed ===
✅ Verified assignment is no longer in project users list (soft-deleted)

=== ✅ All Tests Completed Successfully! ===
```

### Manual Testing with cURL

```bash
# 1. Get authentication token
TOKEN=$(curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"AuthFlow":"USER_PASSWORD_AUTH","ClientId":"3f0fb5mpivctnvj85tucusf88e","AuthParameters":{"USERNAME":"buildboard007+555@gmail.com","PASSWORD":"Mayur@1234"}}' \
  | jq -r '.AuthenticationResult.IdToken')

# 2. Create assignment
curl -X POST "https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/assignments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 16,
    "role_id": 8,
    "context_type": "project",
    "context_id": 30,
    "is_primary": true
  }'

# 3. Get project team
curl -X GET "https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/contexts/project/30/assignments" \
  -H "Authorization: Bearer $TOKEN"
```

## 12. Troubleshooting

### Common Issues

#### Issue: "Invalid assignment context"

**Cause:** The context (project, location, organization) doesn't exist or is deleted.

**Solution:**
```bash
# Verify context exists
GET /projects/{contextId}  # For projects
GET /locations/{contextId} # For locations
GET /organizations/{contextId} # For organizations
```

#### Issue: "Assignment not found"

**Cause:** Assignment ID doesn't exist or is soft-deleted.

**Solution:**
```sql
-- Check if assignment exists
SELECT * FROM iam.user_assignments WHERE id = 123;

-- Check if soft-deleted
SELECT * FROM iam.user_assignments WHERE id = 123 AND is_deleted = true;
```

#### Issue: User can't see project data after assignment

**Cause:** Assignment might not be active yet (start_date in future) or already expired (end_date in past).

**Solution:**
```sql
-- Check assignment active status
SELECT *,
    CASE
        WHEN start_date IS NULL OR start_date <= NOW() THEN 'started'
        ELSE 'not_started'
    END as start_status,
    CASE
        WHEN end_date IS NULL OR end_date >= NOW() THEN 'active'
        ELSE 'expired'
    END as end_status
FROM iam.user_assignments
WHERE id = 123;
```

**Fix:** Update dates or remove them:
```bash
PUT /assignments/123
{
    "start_date": "2025-10-27",
    "end_date": null
}
```

#### Issue: Duplicate assignments

**Cause:** No unique constraint on (user_id, role_id, context_type, context_id).

**Solution:**
```sql
-- Find duplicates
SELECT user_id, role_id, context_type, context_id, COUNT(*)
FROM iam.user_assignments
WHERE is_deleted = FALSE
GROUP BY user_id, role_id, context_type, context_id
HAVING COUNT(*) > 1;

-- Delete extra assignments
DELETE FROM iam.user_assignments
WHERE id = {duplicate_id};
```

#### Issue: GetUserContexts returns empty array

**Cause:** User has no active assignments for that context type.

**Debug:**
```sql
-- Check all user assignments
SELECT * FROM iam.user_assignments
WHERE user_id = 16
  AND context_type = 'project'
  AND is_deleted = FALSE;

-- Check if dates are preventing activation
SELECT *,
    (start_date IS NULL OR start_date <= NOW()) as is_started,
    (end_date IS NULL OR end_date >= NOW()) as not_expired
FROM iam.user_assignments
WHERE user_id = 16
  AND context_type = 'project'
  AND is_deleted = FALSE;
```

#### Issue: 401 Unauthorized

**Cause:** Using access token instead of ID token, or token expired.

**Solution:**
```bash
# Get fresh ID token (not access token!)
TOKEN=$(curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"AuthFlow":"USER_PASSWORD_AUTH","ClientId":"3f0fb5mpivctnvj85tucusf88e","AuthParameters":{"USERNAME":"buildboard007+555@gmail.com","PASSWORD":"Mayur@1234"}}' \
  | jq -r '.AuthenticationResult.IdToken')
```

### Database Queries for Debugging

```sql
-- Get all assignments for a user
SELECT ua.*,
       u.email,
       r.name as role_name,
       CASE ua.context_type
         WHEN 'project' THEN (SELECT name FROM project.projects WHERE id = ua.context_id)
         WHEN 'location' THEN (SELECT name FROM iam.locations WHERE id = ua.context_id)
         WHEN 'organization' THEN (SELECT name FROM iam.organizations WHERE id = ua.context_id)
       END as context_name
FROM iam.user_assignments ua
JOIN iam.users u ON ua.user_id = u.id
JOIN iam.roles r ON ua.role_id = r.id
WHERE ua.user_id = 16 AND ua.is_deleted = FALSE;

-- Get all users on a project
SELECT ua.*,
       u.email,
       u.first_name || ' ' || u.last_name as user_name,
       r.name as role_name
FROM iam.user_assignments ua
JOIN iam.users u ON ua.user_id = u.id
JOIN iam.roles r ON ua.role_id = r.id
WHERE ua.context_type = 'project'
  AND ua.context_id = 30
  AND ua.is_deleted = FALSE;

-- Find expired assignments
SELECT * FROM iam.user_assignments
WHERE end_date < NOW() AND is_deleted = FALSE;

-- Find assignments starting in the future
SELECT * FROM iam.user_assignments
WHERE start_date > NOW() AND is_deleted = FALSE;
```

## Migration from Deprecated Tables

### Deprecated Tables Being Replaced

1. **org_user_roles** → assignments with context_type='organization'
2. **location_user_roles** → assignments with context_type='location'
3. **project_user_roles** → assignments with context_type='project'
4. **project_managers** → assignments with context_type='project' and appropriate role

### Migration Strategy

For each deprecated table, migrate data to user_assignments:

```sql
-- Example: Migrate project_user_roles to user_assignments
INSERT INTO iam.user_assignments (
    user_id, role_id, context_type, context_id,
    trade_type, is_primary, start_date, end_date,
    created_at, created_by, updated_at, updated_by, is_deleted
)
SELECT
    user_id, role_id, 'project', project_id,
    trade_type, is_primary, start_date, end_date,
    created_at, created_by, updated_at, updated_by, is_deleted
FROM project.project_user_roles
WHERE is_deleted = FALSE;
```

## Summary

Assignment Management is the **foundation of access control** in BuildBoard. It:
- Unifies all role assignments across organizations, projects, and locations
- Enables flexible, time-bound, and context-specific user access
- Powers the GetUserContexts() method that filters data across all services
- Replaces deprecated tables with a single, scalable solution
- Provides audit trails for who had access to what and when

**Key takeaway:** Understanding assignments is essential for understanding how users access and interact with any data in the system.