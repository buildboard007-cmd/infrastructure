# Application Architecture - Complete Context Guide

**Last Updated:** 2025-10-25
**Purpose:** Complete architectural overview for AI assistants and new developers

---

## Table of Contents
1. [System Overview](#system-overview)
2. [Database Architecture](#database-architecture)
3. [Access Control & Permission System](#access-control--permission-system)
4. [API Architecture](#api-architecture)
5. [Authentication & Authorization](#authentication--authorization)
6. [Project Structure](#project-structure)
7. [Key Architectural Decisions](#key-architectural-decisions)
8. [Testing Strategy](#testing-strategy)

---

## System Overview

### Technology Stack
- **Backend**: Go (Golang) - Lambda functions
- **Infrastructure**: AWS CDK (TypeScript)
- **Database**: PostgreSQL (RDS) with two schemas: `iam`, `project`
- **API Gateway**: AWS API Gateway with Cognito Authorizer
- **Authentication**: AWS Cognito
- **Deployment**: AWS Lambda + API Gateway
- **Frontend**: React (separate repository at `/Users/mayur/git_personal/ui/frontend`)

### Architecture Style
- **Serverless**: Event-driven Lambda functions
- **Microservices**: Each domain has its own Lambda function
- **Repository Pattern**: Data access abstraction layer
- **RESTful APIs**: Standard REST endpoints

### AWS Infrastructure
- **Environment**: Dev (521805123898), Prod (186375394147)
- **Region**: us-east-2
- **API Gateway**: https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main
- **Database Host**: appdb.cdwmaay8wkw4.us-east-2.rds.amazonaws.com
- **Cognito Client ID**: 3f0fb5mpivctnvj85tucusf88e

---

## Database Architecture

### Schema Organization

**Two Primary Schemas:**

#### 1. `iam` Schema - Identity & Access Management
Tables for users, roles, permissions, and assignments.

#### 2. `project` Schema - Project Management
Tables for projects, issues, RFIs, submittals, and attachments.

### Core Tables

#### Users (`iam.users`)
```sql
- id (bigint, PK)
- cognito_id (varchar) - Links to AWS Cognito
- email (varchar, unique)
- first_name, last_name
- org_id (bigint) - Organization membership
- is_super_admin (boolean) - Super admin flag
- phone_number, job_title, profile_image_url
- is_active, is_deleted
- created_at, created_by, updated_at, updated_by
```

**Key Points:**
- `cognito_id` links user to AWS Cognito authentication
- `org_id` determines organizational membership
- `is_super_admin` grants full access across the organization

#### Organizations (`iam.organizations`)
```sql
- id (bigint, PK)
- name (varchar)
- company_type, industry, company_size
- address, city, state, zip_code, country
- website, logo_url
- is_active, is_deleted
- created_at, created_by, updated_at, updated_by
```

#### Locations (`iam.locations`)
```sql
- id (bigint, PK)
- org_id (bigint, FK)
- name (varchar)
- address, city, state, zip_code, country
- timezone, phone_number
- is_active, is_deleted
- created_at, created_by, updated_at, updated_by
```

**Hierarchy**: Organization → Locations → Projects

#### Roles (`iam.roles`)
```sql
- id (bigint, PK)
- org_id (bigint, FK)
- name (varchar) - e.g., "Project Manager", "Superintendent"
- description (text)
- is_system_role (boolean) - Built-in vs custom roles
- is_active, is_deleted
- created_at, created_by, updated_at, updated_by
```

#### Permissions (`iam.permissions`)
```sql
- id (bigint, PK)
- name (varchar) - e.g., "projects:read", "rfis:create"
- resource (varchar) - Resource type
- action (varchar) - Action type
- description (text)
- created_at, created_by, updated_at, updated_by
```

#### Role Permissions (`iam.role_permissions`)
```sql
- id (bigint, PK)
- role_id (bigint, FK)
- permission_id (bigint, FK)
- created_at, created_by
```

Maps permissions to roles.

#### **User Assignments (`iam.user_assignments`)** ⭐ **MOST IMPORTANT TABLE**
```sql
- id (bigint, PK)
- user_id (bigint, FK)
- role_id (bigint, FK)
- context_type (varchar) - 'organization', 'location', or 'project'
- context_id (bigint) - ID of org/location/project
- trade_type (varchar) - Optional trade specialization
- is_primary (boolean) - Primary assignment flag
- start_date, end_date (date) - Time-bound assignments
- is_active, is_deleted
- created_at, created_by, updated_at, updated_by

UNIQUE CONSTRAINT: (user_id, role_id, context_type, context_id, is_deleted)
```

**This is the CORE table for the entire access control system.**

**Context Types:**
- `organization`: User assigned at org level → sees ALL locations & projects
- `location`: User assigned at location level → sees ALL projects at that location
- `project`: User assigned at project level → sees ONLY that project

#### Projects (`project.projects`)
```sql
- id (bigint, PK) - Internal ID
- project_id (bigint) - Same as id
- org_id (bigint, FK)
- location_id (bigint, FK)
- project_number (varchar, auto-generated) - e.g., "PROJ-2025-0001"
- name, description
- project_type, project_stage, work_scope, project_sector
- delivery_method, project_phase
- start_date, planned_end_date, actual_start_date, actual_end_date
- substantial_completion_date, project_finish_date
- warranty_start_date, warranty_end_date
- budget, contract_value, square_footage
- address, city, state, zip_code, country
- latitude, longitude
- language, status
- is_deleted
- created_at, created_by, updated_at, updated_by
```

#### Issues (`project.issues`)
```sql
- id (bigint, PK)
- project_id (bigint, FK)
- org_id (bigint, FK)
- issue_number (varchar, auto-generated) - e.g., "ISS-0001"
- title, description
- issue_type, priority, status, severity
- due_date, resolved_date
- assigned_to (bigint, FK to users)
- location_description
- is_deleted
- created_at, created_by, updated_at, updated_by
```

#### RFIs (`project.rfis`)
```sql
- id (bigint, PK)
- project_id (bigint, FK)
- org_id (bigint, FK)
- rfi_number (varchar, auto-generated) - e.g., "RFI-0001"
- subject, question
- priority, status
- due_date, response_date
- assigned_to (bigint, FK to users)
- response_text, response_by
- is_deleted
- created_at, created_by, updated_at, updated_by
```

#### Submittals (`project.submittals`)
```sql
- id (bigint, PK)
- project_id (bigint, FK)
- org_id (bigint, FK)
- submittal_number (varchar, auto-generated) - e.g., "SUB-0001"
- title, description
- submittal_type, status, priority
- spec_section, drawing_reference
- received_date, due_date, approved_date
- submitted_by, reviewed_by
- review_notes
- is_deleted
- created_at, created_by, updated_at, updated_by
```

#### Attachments (`project.attachments`)
```sql
- id (bigint, PK)
- org_id (bigint, FK)
- entity_type (varchar) - 'project', 'issue', 'rfi', 'submittal', 'comment'
- entity_id (bigint) - ID of the parent entity
- file_name, file_path, file_size, file_type
- attachment_type (varchar) - 'project_photo', 'document', etc.
- s3_bucket, s3_key
- uploaded_by (bigint, FK to users)
- is_deleted
- created_at, created_by, updated_at, updated_by
```

**Centralized attachment system** - all attachments stored in one table with entity_type/entity_id pattern.

#### Comments
- `project.issue_comments` - Comments on issues
- `project.rfi_comments` - Comments on RFIs
- `project.submittal_comments` - Comments on submittals

### Dropped/Deprecated Tables

**These tables were removed in October 2025:**
1. ❌ `iam.org_user_roles` → Replaced by `user_assignments` with `context_type='organization'`
2. ❌ `iam.location_user_roles` → Replaced by `user_assignments` with `context_type='location'`
3. ❌ `iam.user_location_access` → Deprecated
4. ❌ `project.project_user_roles` → Replaced by `user_assignments` with `context_type='project'`
5. ❌ `project.project_managers` → Never used

**DO NOT reference these tables in new code.**

---

## Access Control & Permission System

### Hierarchical Access Model

```
Organization (org_id)
    └── Locations (location_id)
        └── Projects (project_id)
            └── Issues, RFIs, Submittals
```

### User Access Levels

#### 1. **Super Admin** (`is_super_admin = TRUE`)
- **Scope**: Entire organization
- **Access**: ALL resources across ALL locations and projects
- **Check**: JWT token contains `isSuperAdmin: true`
- **Use Case**: System administrators, company owners

#### 2. **Organization-Level Assignment** (`context_type = 'organization'`)
- **Scope**: Entire organization
- **Access**: ALL locations and ALL projects in the organization
- **Check**: `user_assignments` has record with `context_type='organization'` and `context_id=org_id`
- **Use Case**: Organization admins, executives

#### 3. **Location-Level Assignment** (`context_type = 'location'`)
- **Scope**: Specific location(s)
- **Access**: ALL projects at assigned location(s)
- **Check**: `user_assignments` has record with `context_type='location'` and `context_id=location_id`
- **Use Case**: Regional managers, location supervisors

#### 4. **Project-Level Assignment** (`context_type = 'project'`)
- **Scope**: Specific project(s)
- **Access**: ONLY assigned project(s)
- **Check**: `user_assignments` has record with `context_type='project'` and `context_id=project_id`
- **Use Case**: Project managers, contractors, subcontractors

### Access Control Implementation

#### Location Dropdown Population (Token Customizer)

**User's visible locations determined by:**
```
IF is_super_admin = TRUE:
    → Return ALL locations in organization

ELSE IF has organization-level assignment:
    → Return ALL locations in organization

ELSE IF has location-level assignments:
    → Return assigned locations

ELSE IF has project-level assignments:
    → Return parent locations of assigned projects
```

**This logic is in:** `/src/infrastructure-token-customizer/main.go`

#### GET /projects Access Control

**Projects visible to user determined by:**
```
IF is_super_admin = TRUE:
    → Return ALL projects (optionally filtered by location_id)

ELSE IF has organization-level assignment:
    → Return ALL projects (optionally filtered by location_id)

ELSE IF has location-level assignment(s):
    IF location_id provided:
        IF user has access to that location:
            → Return projects at that location
        ELSE:
            → Return 403 Forbidden
    ELSE:
        → Return empty list (force location selection)

ELSE IF has project-level assignment(s):
    IF location_id provided:
        → Return assigned projects filtered to that location
    ELSE:
        → Return ALL assigned projects

ELSE:
    → Return empty list (no access)
```

**This logic is in:** `/src/infrastructure-project-management/main.go`

### Repository Methods for Access Control

#### AssignmentRepository (`src/lib/data/assignment_repository.go`)

**Key Methods:**
```go
// Get all context IDs of a specific type that user has access to
GetUserContexts(ctx, userID, contextType, orgID) ([]int64, error)
// Returns: [contextID1, contextID2, ...]

// Get all assignments for a specific context
GetContextAssignments(ctx, contextType, contextID, orgID) (*AssignmentListResponse, error)

// Create new assignment
CreateAssignment(ctx, request, createdBy) (*Assignment, error)

// Update assignment
UpdateAssignment(ctx, assignmentID, request, updatedBy) (*Assignment, error)

// Soft delete assignment
DeleteAssignment(ctx, assignmentID, updatedBy) error

// Get user's active assignments
GetUserAssignments(ctx, userID, orgID) (*AssignmentListResponse, error)
```

#### ProjectRepository (`src/lib/data/project_repository.go`)

**Key Methods:**
```go
// Get all projects in organization
GetProjectsByOrg(ctx, orgID) ([]Project, error)

// Get projects at specific location
GetProjectsByLocationID(ctx, locationID, orgID) ([]Project, error)

// Get projects by list of IDs (for project-level users)
GetProjectsByIDs(ctx, projectIDs []int64, orgID) ([]Project, error)

// Get single project
GetProjectByID(ctx, projectID, orgID) (*Project, error)
```

---

## API Architecture

### Lambda Functions

Each domain has its own Lambda function in `/src/`:

1. **infrastructure-api-gateway-cors** - CORS handler
2. **infrastructure-token-customizer** - JWT token enrichment
3. **infrastructure-user-signup** - User registration
4. **infrastructure-user-management** - User CRUD
5. **infrastructure-organization-management** - Organization CRUD
6. **infrastructure-location-management** - Location CRUD
7. **infrastructure-roles-management** - Role CRUD
8. **infrastructure-permissions-management** - Permission CRUD
9. **infrastructure-assignment-management** - User assignment CRUD ⭐
10. **infrastructure-project-management** - Project CRUD + user assignments
11. **infrastructure-issue-management** - Issue CRUD
12. **infrastructure-rfi-management** - RFI CRUD
13. **infrastructure-submittal-management** - Submittal CRUD
14. **infrastructure-attachment-management** - Centralized attachment system

### API Endpoints

#### Authentication
- `POST /auth/signup` - User registration

#### Users
- `GET /users` - List users (with pagination)
- `GET /users/{userId}` - Get user details
- `GET /users/profile` - Get current user profile
- `PUT /users/{userId}` - Update user
- `DELETE /users/{userId}` - Soft delete user

#### Organizations
- `POST /organizations` - Create organization
- `GET /organizations` - List organizations
- `GET /organizations/{orgId}` - Get organization
- `PUT /organizations/{orgId}` - Update organization

#### Locations
- `POST /locations` - Create location
- `GET /locations` - List locations (filtered by org)
- `GET /locations/{locationId}` - Get location
- `PUT /locations/{locationId}` - Update location

#### Roles
- `POST /roles` - Create role
- `GET /roles` - List roles (filtered by org)
- `GET /roles/{roleId}` - Get role
- `PUT /roles/{roleId}` - Update role
- `DELETE /roles/{roleId}` - Soft delete role

#### Permissions
- `GET /permissions` - List all permissions
- `GET /permissions/{permissionId}` - Get permission

#### Role Permissions
- `POST /roles/{roleId}/permissions` - Assign permission to role
- `GET /roles/{roleId}/permissions` - List role permissions
- `DELETE /roles/{roleId}/permissions/{permissionId}` - Remove permission from role

#### Assignments ⭐
- `POST /assignments` - Create user assignment
- `GET /assignments` - List assignments (filtered by user/context)
- `GET /assignments/{assignmentId}` - Get assignment
- `PUT /assignments/{assignmentId}` - Update assignment
- `DELETE /assignments/{assignmentId}` - Soft delete assignment
- `GET /assignments/users/{userId}/contexts` - Get user's contexts
- `POST /assignments/bulk` - Bulk create assignments
- `POST /assignments/transfer` - Transfer assignments

#### Projects
- `POST /projects` - Create project
- `GET /projects` - List projects ⭐ **WITH ACCESS CONTROL**
  - Query params: `location_id` (optional)
  - Returns projects based on user access level
- `GET /projects/{projectId}` - Get project
- `PUT /projects/{projectId}` - Update project

#### Project User Assignments
- `POST /projects/{projectId}/users` - Assign user to project
- `GET /projects/{projectId}/users` - Get project team members
- `PUT /projects/{projectId}/users/{assignmentId}` - Update user role
- `DELETE /projects/{projectId}/users/{assignmentId}` - Remove user from project

**Note:** These now use `user_assignments` table with `context_type='project'`

#### Issues
- `POST /projects/{projectId}/issues` - Create issue
- `GET /projects/{projectId}/issues` - List issues
- `GET /issues/{issueId}` - Get issue
- `PUT /issues/{issueId}` - Update issue
- `DELETE /issues/{issueId}` - Soft delete issue

#### Issue Comments
- `POST /issues/{issueId}/comments` - Add comment
- `GET /issues/{issueId}/comments` - List comments
- `PUT /issues/{issueId}/comments/{commentId}` - Update comment
- `DELETE /issues/{issueId}/comments/{commentId}` - Delete comment

#### RFIs
- `POST /projects/{projectId}/rfis` - Create RFI
- `GET /projects/{projectId}/rfis` - List RFIs
- `GET /rfis/{rfiId}` - Get RFI
- `PUT /rfis/{rfiId}` - Update RFI
- `DELETE /rfis/{rfiId}` - Soft delete RFI

#### RFI Comments
- `POST /rfis/{rfiId}/comments` - Add comment
- `GET /rfis/{rfiId}/comments` - List comments

#### Submittals
- `POST /projects/{projectId}/submittals` - Create submittal
- `GET /projects/{projectId}/submittals` - List submittals
- `GET /submittals/{submittalId}` - Get submittal
- `PUT /submittals/{submittalId}` - Update submittal
- `DELETE /submittals/{submittalId}` - Soft delete submittal

#### Submittal Comments
- `POST /submittals/{submittalId}/comments` - Add comment
- `GET /submittals/{submittalId}/comments` - List comments

#### Attachments (Centralized)
- `POST /attachments` - Upload attachment
  - Body: `entity_type`, `entity_id`, `file_name`, etc.
- `GET /attachments?entity_type={type}&entity_id={id}` - List attachments
- `GET /attachments/{attachmentId}` - Get attachment
- `DELETE /attachments/{attachmentId}` - Soft delete attachment

---

## Authentication & Authorization

### Cognito Authentication Flow

1. **User Login** → Cognito returns ID Token and Access Token
2. **API Request** → Client sends ID Token in `Authorization: Bearer {token}` header
3. **API Gateway Authorizer** → Validates token with Cognito
4. **Token Customizer Lambda** → Enriches token with user data
5. **Lambda Handler** → Extracts claims from token

### JWT Token Claims

**Standard Claims:**
```json
{
  "sub": "cognito-user-id",
  "email": "user@example.com",
  "cognito:username": "username"
}
```

**Custom Claims (Added by Token Customizer):**
```json
{
  "user_id": 19,
  "org_id": 10,
  "isSuperAdmin": true,
  "locations": [6, 7, 22, 24],
  "accessContexts": ["ORG:10", "PROJ:29"]
}
```

### Claims Extraction in Go

```go
// src/lib/auth/auth.go
type Claims struct {
    UserID       int64  `json:"user_id"`
    Email        string `json:"email"`
    CognitoID    string `json:"sub"`
    OrgID        int64  `json:"org_id"`
    IsSuperAdmin bool   `json:"isSuperAdmin"`
}

claims, err := auth.ExtractClaimsFromRequest(request)
```

### Authorization Patterns

**Pattern 1: Organization-level check**
```go
if claims.OrgID != resourceOrgID {
    return api.ErrorResponse(http.StatusForbidden, "Access denied", logger), nil
}
```

**Pattern 2: Super Admin bypass**
```go
if !claims.IsSuperAdmin {
    // Check specific permissions
}
```

**Pattern 3: Assignment-based access**
```go
assignments, err := assignmentRepository.GetUserContexts(ctx, claims.UserID, "project", claims.OrgID)
if !contains(assignments, projectID) {
    return api.ErrorResponse(http.StatusForbidden, "Access denied", logger), nil
}
```

---

## Project Structure

```
infrastructure/
├── bin/                           # CDK app entry point
├── lib/                           # CDK infrastructure code
│   ├── resources/
│   │   ├── api_gateway/          # API Gateway construct
│   │   ├── cognito/              # Cognito user pool
│   │   ├── lambda/               # Lambda constructs
│   │   └── sub_stack/            # Nested stack
│   └── infrastructure-stack.ts   # Main CDK stack
├── src/                          # Go Lambda functions
│   ├── infrastructure-api-gateway-cors/
│   ├── infrastructure-token-customizer/
│   ├── infrastructure-user-signup/
│   ├── infrastructure-user-management/
│   ├── infrastructure-organization-management/
│   ├── infrastructure-location-management/
│   ├── infrastructure-roles-management/
│   ├── infrastructure-permissions-management/
│   ├── infrastructure-assignment-management/
│   ├── infrastructure-project-management/
│   ├── infrastructure-issue-management/
│   ├── infrastructure-rfi-management/
│   ├── infrastructure-submittal-management/
│   ├── infrastructure-attachment-management/
│   └── lib/                      # Shared Go libraries
│       ├── api/                  # API utilities
│       ├── auth/                 # Authentication
│       ├── clients/              # AWS clients (SSM, RDS, S3)
│       ├── constants/            # Constants
│       ├── data/                 # Repository implementations
│       │   ├── assignment_repository.go ⭐
│       │   ├── project_repository.go
│       │   ├── user_repository.go
│       │   ├── issue_repository.go
│       │   ├── rfi_repository.go
│       │   ├── submittal_repository.go
│       │   └── attachment_repository.go
│       ├── models/               # Data models
│       └── validators/           # Input validation
├── docs/                         # Documentation
│   ├── APPLICATION-ARCHITECTURE.md (this file)
│   ├── CHANGES-SUMMARY.md
│   ├── assignment-architecture.md
│   └── VERIFICATION-*.md
├── testing/                      # Test scripts
│   ├── api/                      # API endpoint tests
│   │   ├── test-project-user-management.sh
│   │   └── test-get-projects-access-control.sh
│   ├── auth/                     # Auth tests
│   ├── database/                 # DB validation scripts
│   └── utilities/                # Helper scripts
├── postman/                      # Postman collections
│   ├── ProjectManagement.postman_collection.json
│   ├── IssueManagement.postman_collection.json
│   └── ...
├── cdk.json                      # CDK configuration
├── package.json                  # NPM dependencies
└── CLAUDE.md                     # Instructions for AI assistants
```

---

## Key Architectural Decisions

### 1. Unified Assignment Management (October 2025)

**Decision:** Consolidate all user-role assignments into a single `user_assignments` table with context pattern.

**Rationale:**
- Eliminates duplicate logic across org/location/project levels
- Single source of truth for all assignments
- Easier to query and manage access control
- Supports hierarchical permission inheritance
- Extensible to new context types (department, equipment, phase)

**Migration:** Dropped 5 legacy tables (org_user_roles, location_user_roles, user_location_access, project_user_roles, project_managers)

### 2. Centralized Attachment Management

**Decision:** Single `attachments` table with `entity_type`/`entity_id` pattern instead of separate tables per entity.

**Rationale:**
- Avoids duplicate attachment logic
- Consistent attachment handling across all entities
- Easier to implement attachment features (upload, download, delete)
- Single Lambda function handles all attachments

**Supported Entities:** project, issue, rfi, submittal, comment

### 3. Soft Deletes

**Decision:** Use `is_deleted` flag instead of hard deletes.

**Rationale:**
- Maintains data integrity and audit trail
- Allows "undo" operations
- Preserves foreign key relationships
- Required for compliance and auditing

**Implementation:** All tables have `is_deleted BOOLEAN DEFAULT FALSE`

### 4. Auto-generated Numbers

**Decision:** Auto-generate unique numbers for projects, issues, RFIs, submittals.

**Format:**
- Projects: `PROJ-YYYY-NNNN` (e.g., PROJ-2025-0001)
- Issues: `ISS-NNNN` (e.g., ISS-0001)
- RFIs: `RFI-NNNN` (e.g., RFI-0001)
- Submittals: `SUB-NNNN` (e.g., SUB-0001)

**Implementation:** PostgreSQL sequences + trigger/application logic

### 5. Location-First UI Pattern

**Decision:** Users select location first, then see projects at that location.

**Rationale:**
- Simplifies access control logic
- Matches real-world workflow (users work at one location at a time)
- Reduces query complexity
- Improves UI performance

**Implementation:**
- Token customizer populates location dropdown based on access
- GET /projects requires `location_id` for non-admin users

### 6. Repository Pattern

**Decision:** Abstract all database access through repository interfaces.

**Benefits:**
- Testability (can mock repositories)
- Consistency across services
- Single place to change queries
- Type safety with Go interfaces

**Example:**
```go
type ProjectRepository interface {
    CreateProject(ctx, orgID, request, userID) (*ProjectResponse, error)
    GetProjectsByOrg(ctx, orgID) ([]Project, error)
    GetProjectByID(ctx, projectID, orgID) (*Project, error)
    // ...
}
```

### 7. Nested Request Structure for Projects

**Decision:** Group project fields into logical sections in API requests.

**Structure:**
```json
{
  "location_id": 6,
  "basic_info": { "name": "...", "description": "..." },
  "project_details": { "project_stage": "...", "work_scope": "..." },
  "location": { "address": "...", "city": "..." },
  "timeline": { "start_date": "...", "completion_date": "..." },
  "financial": { "budget": 5000000 }
}
```

**Benefits:**
- Cleaner API contracts
- Logical grouping of related fields
- Easier to maintain and extend

---

## Testing Strategy

### Test User Credentials
- **Email:** buildboard007+555@gmail.com
- **Password:** Mayur@1234
- **User ID:** 19
- **Org ID:** 10
- **Is Super Admin:** TRUE

### Authentication for API Tests

**Get ID Token:**
```bash
TOKEN=$(curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{
    "AuthFlow":"USER_PASSWORD_AUTH",
    "ClientId":"3f0fb5mpivctnvj85tucusf88e",
    "AuthParameters":{
      "USERNAME":"buildboard007+555@gmail.com",
      "PASSWORD":"Mayur@1234"
    }
  }' | jq -r '.AuthenticationResult.IdToken')
```

**Use in API Calls:**
```bash
curl -X GET "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN"
```

### Test Script Locations

All test scripts in `/testing/api/`:
- `test-project-user-management.sh` - Project assignment CRUD
- `test-get-projects-access-control.sh` - Access control verification
- `test-issue-comments.sh` - Issue comment system
- `test-comment-attachment.sh` - Comment attachments

### Database Queries via MCP

**Use natural language to query database:**
```
"Show me all users in organization 10"
"What are the active projects at location 6?"
"List all assignments for user 19"
```

The MCP server `postgres-construction` automatically handles queries.

### Postman Collections

Located in `/postman/`:
- `ProjectManagement.postman_collection.json`
- `IssueManagement.postman_collection.json`
- `RFIManagement.postman_collection.json`
- `SubmittalManagement.postman_collection.json`

**Variables Needed:**
- `access_token` - ID token from Cognito
- `project_id` - Project ID for testing
- `location_id` - Location ID for filtering
- `user_id` - User ID for assignments
- `role_id` - Role ID for assignments

---

## Common Development Workflows

### Adding a New Lambda Function

1. **Create Go code** in `/src/infrastructure-{name}/main.go`
2. **Add CDK construct** in `/lib/resources/lambda/`
3. **Wire up API Gateway** routes in `/lib/resources/api_gateway/`
4. **Deploy**: `npm run build && npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev`

### Adding a New Database Table

1. **Create migration SQL** (if using migrations)
2. **Update models** in `/src/lib/models/`
3. **Create/update repository** in `/src/lib/data/`
4. **Add repository to Lambda** in relevant service's `main.go`
5. **Test with MCP**: Query new table to verify

### Modifying Access Control

1. **Understand hierarchy**: Super Admin → Org → Location → Project
2. **Check user assignments**: Query `user_assignments` table
3. **Update handler logic**: Modify `handleXXX` function in Lambda
4. **Use repository methods**: `GetUserContexts()` for access checks
5. **Test with different user levels**: Super admin, org admin, project user

### Deploying Changes

**Dev Environment:**
```bash
npm run build
npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev
```

**Prod Environment:**
```bash
# Manual approval required in CodePipeline
# Changes automatically deploy through CI/CD pipeline
```

### Debugging Issues

1. **Check CloudWatch Logs**: Each Lambda has its own log group
2. **Verify token claims**: Check JWT token contains correct user_id, org_id
3. **Query database**: Use MCP to verify data state
4. **Test with curl**: Use test scripts in `/testing/api/`
5. **Check access control**: Verify user has correct assignments

---

## Important Context for AI Assistants

### When Reading This Document

You now have complete context of:
1. ✅ Database schema and table relationships
2. ✅ Access control hierarchy and implementation
3. ✅ API endpoints and Lambda architecture
4. ✅ Authentication and authorization flow
5. ✅ Deprecated tables to avoid
6. ✅ Key architectural decisions and rationale
7. ✅ Testing strategies and credentials

### What NOT To Do

❌ **DO NOT** reference these deprecated tables:
- `iam.org_user_roles`
- `iam.location_user_roles`
- `iam.user_location_access`
- `project.project_user_roles`
- `project.project_managers`

❌ **DO NOT** use bash for database queries - use MCP natural language

❌ **DO NOT** create test files in project root - use `/testing/` subdirectories

❌ **DO NOT** use Access Tokens - always use ID Tokens for API Gateway

### What TO Do

✅ **DO** use `user_assignments` table for all assignment operations

✅ **DO** check access control using `GetUserContexts()` repository method

✅ **DO** follow the hierarchical access model: Super Admin → Org → Location → Project

✅ **DO** use soft deletes (`is_deleted = TRUE`) instead of hard deletes

✅ **DO** query database using natural language with MCP

✅ **DO** create test scripts in `/testing/api/` directory

✅ **DO** update Postman collections when adding new endpoints

✅ **DO** use repository pattern for all data access

---

## Quick Reference

### Key Files
- Access Control Logic: `/src/infrastructure-project-management/main.go` (lines 131-295)
- Token Customizer: `/src/infrastructure-token-customizer/main.go` (lines 218-252)
- Assignment Repository: `/src/lib/data/assignment_repository.go`
- User Repository: `/src/lib/data/user_repository.go` (GetUserProfile method)
- Auth Claims: `/src/lib/auth/auth.go`

### Key SQL Queries

**Check user access level:**
```sql
SELECT u.email, u.is_super_admin,
       ua.context_type, ua.context_id
FROM iam.users u
LEFT JOIN iam.user_assignments ua ON u.id = ua.user_id
WHERE u.email = 'buildboard007+555@gmail.com'
  AND ua.is_deleted = FALSE;
```

**Get user's projects:**
```sql
SELECT p.id, p.name, p.project_number
FROM project.projects p
JOIN iam.user_assignments ua ON ua.context_id = p.id
WHERE ua.user_id = 19
  AND ua.context_type = 'project'
  AND ua.is_deleted = FALSE
  AND p.is_deleted = FALSE;
```

### Environment Variables (Lambda)
- `LOG_LEVEL` - DEBUG or ERROR
- `IS_LOCAL` - true for local testing
- Database credentials loaded from SSM Parameter Store

---

**Document Version:** 1.0
**Last Updated:** 2025-10-25
**Maintained By:** Development Team