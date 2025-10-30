# Access Control System Architecture

**Version:** 1.0
**Last Updated:** 2025-10-27
**Status:** Production (October 2025 Migration Complete)
**Purpose:** THE definitive guide to the hierarchical access control system

---

## Table of Contents

1. [Overview](#overview)
2. [Core Concept: user_assignments Table](#core-concept-user_assignments-table)
3. [Hierarchical Access Model](#hierarchical-access-model)
4. [Context Types](#context-types)
5. [Access Control in Practice](#access-control-in-practice)
6. [Token Customizer Integration](#token-customizer-integration)
7. [Implementation Pattern](#implementation-pattern)
8. [Complete Example: GET /projects](#complete-example-get-projects)
9. [GetUserContexts Method](#getusercontexts-method)
10. [Migration from Legacy System](#migration-from-legacy-system)
11. [Security Model](#security-model)
12. [Best Practices](#best-practices)

---

## Overview

**The access control system is THE CORE of the entire BuildBoard platform.** It determines who can access what resources across the entire construction management application.

### The Fundamental Principle

**Everything flows from the `iam.user_assignments` table.**

This single table, using a flexible **context pattern**, replaced 5 separate assignment tables and unified the entire permission system. It controls:

- Which projects users can see
- Which locations appear in dropdowns
- Which RFIs, submittals, and issues users can access
- What data is returned from every API endpoint
- Which permissions users have for different resources

### Architecture Philosophy

The system implements a **hierarchical, context-based access control model** where:

1. **Users are assigned roles in specific contexts** (organization, location, project)
2. **Access is inherited downward** through the hierarchy
3. **Super admins bypass all restrictions** for administrative purposes
4. **Every API endpoint filters data** based on user access level
5. **JWT tokens carry location access** for client-side UI control

---

## Core Concept: user_assignments Table

### Table Schema

```sql
CREATE TABLE iam.user_assignments (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES iam.users(id),
    role_id         BIGINT NOT NULL REFERENCES iam.roles(id),
    context_type    VARCHAR(50) NOT NULL,     -- The TYPE of resource
    context_id      BIGINT NOT NULL,          -- The SPECIFIC resource ID
    trade_type      VARCHAR(100),             -- Optional specialization
    is_primary      BOOLEAN DEFAULT FALSE,    -- Primary assignment flag
    start_date      DATE,                     -- Time-bound: when active
    end_date        DATE,                     -- Time-bound: when expires
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by      BIGINT NOT NULL,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by      BIGINT NOT NULL,
    is_deleted      BOOLEAN NOT NULL DEFAULT FALSE,

    -- Ensures unique active assignments per user/role/context
    CONSTRAINT unique_user_role_context UNIQUE (user_id, role_id, context_type, context_id, is_deleted)
);

-- Performance indexes
CREATE INDEX idx_user_assignments_user_id ON iam.user_assignments(user_id);
CREATE INDEX idx_user_assignments_context ON iam.user_assignments(context_type, context_id);
CREATE INDEX idx_user_assignments_role_id ON iam.user_assignments(role_id);
CREATE INDEX idx_user_assignments_is_deleted ON iam.user_assignments(is_deleted);

-- Partial index for active assignments (common query pattern)
CREATE INDEX idx_user_assignments_active ON iam.user_assignments(user_id, context_type)
WHERE is_deleted = FALSE
  AND (start_date IS NULL OR start_date <= NOW())
  AND (end_date IS NULL OR end_date >= NOW());
```

### The Context Pattern

The genius of this table is the **context pattern**: `context_type` + `context_id`

Instead of separate tables like:
- `org_user_roles` (deprecated)
- `location_user_roles` (deprecated)
- `project_user_roles` (deprecated)

We have ONE table where:

```sql
-- Organization-level assignment
context_type = 'organization', context_id = 10
-- Meaning: User has this role across entire organization 10

-- Location-level assignment
context_type = 'location', context_id = 6
-- Meaning: User has this role at location 6

-- Project-level assignment
context_type = 'project', context_id = 30
-- Meaning: User has this role on project 30
```

### Time-Bound Assignments

The `start_date` and `end_date` fields enable **temporal access control**:

```sql
-- Contractor assigned for 3 months
INSERT INTO iam.user_assignments (
    user_id, role_id, context_type, context_id,
    start_date, end_date, ...
) VALUES (
    21, 10, 'project', 30,
    '2025-11-01', '2026-01-31', ...
);
```

This assignment is only active between Nov 1, 2025 and Jan 31, 2026. After the end date, the user automatically loses access.

### Soft Deletes

All assignments use **soft deletes** (`is_deleted = TRUE`) instead of hard deletes:

- Maintains audit trail
- Preserves historical data
- Allows "undo" operations
- Required for compliance

---

## Hierarchical Access Model

### The Hierarchy

```
Organization (org_id)
    └── Locations (location_id)
        └── Projects (project_id)
            └── Issues, RFIs, Submittals, Attachments
```

### Access Levels (From Highest to Lowest)

#### 1. Super Admin (`users.is_super_admin = TRUE`)

**Database Field:** `iam.users.is_super_admin`
**JWT Claim:** `isSuperAdmin: true`
**Scope:** Entire organization
**Access:** EVERYTHING across ALL locations and projects

**Check Pattern:**
```go
if claims.IsSuperAdmin {
    // Bypass all access control checks
    // Return ALL resources (optionally filtered by location_id if provided)
}
```

**Use Cases:**
- Company owners
- System administrators
- Executive leadership

---

#### 2. Organization-Level Assignment

**Assignment:** `context_type = 'organization'`, `context_id = <org_id>`
**Scope:** Entire organization
**Access:** ALL locations and ALL projects in the organization

**Query:**
```sql
SELECT DISTINCT context_id FROM iam.user_assignments
WHERE user_id = $1
  AND context_type = 'organization'
  AND org_id = $2
  AND is_deleted = FALSE;
```

**Check Pattern:**
```go
orgContexts, err := assignmentRepository.GetUserContexts(ctx, userID, "organization", orgID)
if len(orgContexts) > 0 {
    // User has org-level access
    // Return ALL projects (optionally filtered by location_id)
}
```

**Use Cases:**
- Organization administrators
- C-suite executives
- Operations managers who need visibility across everything

---

#### 3. Location-Level Assignment

**Assignment:** `context_type = 'location'`, `context_id = <location_id>`
**Scope:** Specific location(s)
**Access:** ALL projects at assigned location(s)

**Query:**
```sql
SELECT DISTINCT context_id FROM iam.user_assignments
WHERE user_id = $1
  AND context_type = 'location'
  AND is_deleted = FALSE;
-- Returns: [6, 7, 22]  (accessible location IDs)
```

**Check Pattern:**
```go
locationContexts, err := assignmentRepository.GetUserContexts(ctx, userID, "location", orgID)
if len(locationContexts) > 0 {
    // User has location-level access
    if locationIDProvided {
        // Check if user has access to THIS specific location
        if contains(locationContexts, locationID) {
            return projectRepository.GetProjectsByLocationID(ctx, locationID, orgID)
        } else {
            return 403 Forbidden  // User doesn't have access to this location
        }
    } else {
        return []  // Force location selection in UI
    }
}
```

**Use Cases:**
- Regional managers overseeing multiple job sites
- Location supervisors
- Area coordinators

**Important:** Users with location-level access MUST select a location in the UI. The API requires `location_id` query parameter.

---

#### 4. Project-Level Assignment

**Assignment:** `context_type = 'project'`, `context_id = <project_id>`
**Scope:** Specific project(s)
**Access:** ONLY assigned project(s)

**Query:**
```sql
SELECT DISTINCT context_id FROM iam.user_assignments
WHERE user_id = $1
  AND context_type = 'project'
  AND is_deleted = FALSE;
-- Returns: [30, 45, 67]  (accessible project IDs)
```

**Check Pattern:**
```go
projectContexts, err := assignmentRepository.GetUserContexts(ctx, userID, "project", orgID)
if len(projectContexts) > 0 {
    // User has project-level access (most restrictive)
    if locationIDProvided {
        // Filter assigned projects to those at this location
        allProjects, _ := projectRepository.GetProjectsByLocationID(ctx, locationID, orgID)
        projects = filterProjectsByIDs(allProjects, projectContexts)
    } else {
        // Return all assigned projects (across all locations)
        projects, err = projectRepository.GetProjectsByIDs(ctx, projectContexts, orgID)
    }
}
```

**Use Cases:**
- Project managers (assigned to specific projects)
- Contractors and subcontractors
- Temporary workers
- External consultants

---

## Context Types

The system currently supports 3 active context types, with 3 more planned for future:

### Active Context Types

#### 1. `organization`
- **Scope:** Company-wide access
- **Example:** VP of Operations assigned to entire organization
- **Inheritance:** Grants access to ALL locations and projects
- **Context ID:** Organization ID from `iam.organizations.id`

#### 2. `location`
- **Scope:** Job site / regional access
- **Example:** Site supervisor assigned to Downtown Office location
- **Inheritance:** Grants access to ALL projects at that location
- **Context ID:** Location ID from `iam.locations.id`

#### 3. `project`
- **Scope:** Individual project access
- **Example:** Electrician assigned to specific construction project
- **Inheritance:** No inheritance - access limited to that project only
- **Context ID:** Project ID from `project.projects.id`

### Future Context Types (Planned)

#### 4. `department` (Future)
- **Scope:** Department-specific resources
- **Example:** HR manager assigned to Human Resources department
- **Context ID:** Department ID (table TBD)

#### 5. `equipment` (Future)
- **Scope:** Equipment operator assignments
- **Example:** Crane operator assigned to specific equipment
- **Context ID:** Equipment ID (table TBD)

#### 6. `phase` (Future)
- **Scope:** Project phase assignments
- **Example:** Foundation contractor assigned to "Foundation" phase
- **Context ID:** Phase ID (table TBD)

---

## Access Control in Practice

### The Golden Rule

**Every API endpoint that returns data MUST filter based on user assignments.**

There are NO exceptions. Even super admins go through the access control flow (they just bypass the restrictions).

### Standard Access Control Flow

```
1. Extract JWT claims from request
   ↓
2. Get user_id, org_id, is_super_admin from claims
   ↓
3. IF super admin → Return ALL resources (bypass)
   ↓
4. ELSE IF org-level assignment → Return ALL resources
   ↓
5. ELSE IF location-level assignment → Return resources at assigned locations
   ↓
6. ELSE IF project-level assignment → Return ONLY assigned resources
   ↓
7. ELSE → Return empty list (no access)
```

### Access Check Code Pattern

This pattern appears in EVERY service that manages resources:

```go
// Step 1: Extract claims
claims, err := auth.ExtractClaimsFromRequest(request)
if err != nil {
    return api.ErrorResponse(http.StatusUnauthorized, "Authentication failed", logger), nil
}

userID := claims.UserID
orgID := claims.OrgID
isSuperAdmin := claims.IsSuperAdmin

// Step 2: Super admin bypass
if isSuperAdmin {
    // Return everything (optionally filtered by provided parameters)
    return allResources, nil
}

// Step 3: Check organization-level access
orgContexts, err := assignmentRepository.GetUserContexts(ctx, userID, "organization", orgID)
if err != nil {
    return api.ErrorResponse(http.StatusInternalServerError, "Failed to check permissions", logger), nil
}

if len(orgContexts) > 0 {
    // User has org-level access → return all resources
    return allResources, nil
}

// Step 4: Check location-level access
locationContexts, err := assignmentRepository.GetUserContexts(ctx, userID, "location", orgID)
if err != nil {
    return api.ErrorResponse(http.StatusInternalServerError, "Failed to check permissions", logger), nil
}

if len(locationContexts) > 0 {
    // User has location-level access
    // Validate they have access to requested location
    if !contains(locationContexts, requestedLocationID) {
        return api.ErrorResponse(http.StatusForbidden, "Access denied to this location", logger), nil
    }
    return resourcesAtLocation, nil
}

// Step 5: Check project-level access
projectContexts, err := assignmentRepository.GetUserContexts(ctx, userID, "project", orgID)
if err != nil {
    return api.ErrorResponse(http.StatusInternalServerError, "Failed to check permissions", logger), nil
}

if len(projectContexts) > 0 {
    // User has project-level access → return only assigned projects
    return filterByProjectIDs(allResources, projectContexts), nil
}

// Step 6: No access at any level
return []Resource{}, nil
```

---

## Token Customizer Integration

### Purpose

The **Token Customizer Lambda** (`infrastructure-token-customizer`) runs during Cognito token generation to enrich JWT tokens with user profile and access data.

**Trigger:** AWS Cognito Pre-Token Generation V2.0
**When:** Every login, token refresh, and password change

### Location Discovery for JWT

The Token Customizer uses assignments to populate the **locations dropdown** that appears in the UI:

```go
// From: /src/infrastructure-token-customizer/main.go

// Step 1: Fetch user profile with access contexts
userQuery := `
    SELECT
        u.id, u.cognito_id, u.email, u.first_name, u.last_name,
        u.org_id, o.name as org_name, u.is_super_admin,
        COALESCE(
            array_agg(DISTINCT
                CASE ua.context_type
                    WHEN 'organization' THEN 'ORG:' || ua.context_id
                    WHEN 'location' THEN 'LOC:' || ua.context_id
                    WHEN 'project' THEN 'PROJ:' || ua.context_id
                END
            ) FILTER (WHERE ua.context_id IS NOT NULL),
            ARRAY[]::text[]
        ) as access_contexts
    FROM iam.users u
    JOIN iam.organizations o ON u.org_id = o.id
    LEFT JOIN iam.user_assignments ua ON u.id = ua.user_id AND ua.is_deleted = false
    WHERE u.cognito_id = $1 AND u.is_deleted = FALSE
    GROUP BY u.id, o.name;
`

// Step 2: Determine accessible locations
if profile.IsSuperAdmin {
    // Super admin → ALL locations in organization
    locationQuery = `
        SELECT DISTINCT l.id, l.name, l.location_type
        FROM iam.locations l
        WHERE l.org_id = $1 AND l.is_deleted = false
        ORDER BY l.name;
    `
} else {
    // Parse access_contexts array
    for _, context := range accessContexts {
        // "ORG:10"  → hasOrgAccess = true
        // "LOC:6"   → accessibleLocationIds = [6]
        // "PROJ:30" → get location of project 30
    }

    if hasOrgAccess {
        // Org-level assignment → ALL locations
        locationQuery = /* same as super admin */
    } else if len(accessibleLocationIds) > 0 {
        // Specific locations accessible
        locationQuery = `
            SELECT DISTINCT l.id, l.name, l.location_type
            FROM iam.locations l
            WHERE l.id = ANY($1::bigint[]) AND l.is_deleted = false
        `
    }
}
```

### JWT Token Structure

After Token Customizer runs, the ID token contains:

```json
{
  "sub": "cognito-uuid-here",
  "email": "user@example.com",
  "user_id": "19",
  "org_id": "10",
  "org_name": "BuildBoard Construction",
  "isSuperAdmin": false,
  "locations": "base64_encoded_json_array_of_locations",
  "first_name": "John",
  "last_name": "Doe",
  "status": "active"
}
```

The `locations` field (Base64-encoded JSON) contains:

```json
[
  { "id": "6", "name": "Downtown Office", "location_type": "office" },
  { "id": "7", "name": "Westside Construction Site", "location_type": "job_site" },
  { "id": "22", "name": "North Warehouse", "location_type": "warehouse" }
]
```

### Frontend Usage

The frontend decodes the locations to populate the location dropdown:

```javascript
// Decode locations from JWT token
const token = parseJWT(idToken);
const locations = JSON.parse(atob(token.locations));

// Populate location dropdown
locations.forEach(location => {
    dropdown.addOption(location.id, location.name);
});
```

---

## Implementation Pattern

### Repository Method: GetUserContexts

**File:** `/src/lib/data/assignment_repository.go`
**Interface:** `AssignmentRepository`
**Method:** `GetUserContexts(ctx, userID, contextType, orgID) ([]int64, error)`

This is THE MOST IMPORTANT method in the entire access control system.

#### Method Signature

```go
// GetUserContexts returns all context IDs of a specific type that user has access to
// This is used throughout the system to filter data by user access level
//
// Parameters:
//   - ctx: Request context
//   - userID: Internal user ID
//   - contextType: "organization", "location", or "project"
//   - orgID: Organization ID (for isolation)
//
// Returns:
//   - []int64: Array of accessible context IDs
//   - error: Database errors
//
// Example: GetUserContexts(ctx, 19, "project", 10)
// Returns: [30, 45, 67] - User 19 can access projects 30, 45, and 67
func (dao *AssignmentDao) GetUserContexts(
    ctx context.Context,
    userID int64,
    contextType string,
    orgID int64,
) ([]int64, error)
```

#### Implementation

```go
func (dao *AssignmentDao) GetUserContexts(ctx context.Context, userID int64, contextType string, orgID int64) ([]int64, error) {
    query := `
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
    `

    rows, err := dao.DB.QueryContext(ctx, query, userID, contextType, orgID)
    if err != nil {
        return nil, fmt.Errorf("failed to get user contexts: %w", err)
    }
    defer rows.Close()

    var contextIDs []int64
    for rows.Next() {
        var contextID int64
        if err := rows.Scan(&contextID); err != nil {
            dao.Logger.WithError(err).Error("Failed to scan context ID")
            continue
        }
        contextIDs = append(contextIDs, contextID)
    }

    return contextIDs, nil
}
```

#### Key Features

1. **Time-aware:** Checks `start_date` and `end_date` to ensure assignment is currently active
2. **Soft-delete aware:** Filters out `is_deleted = TRUE` assignments
3. **Organization-scoped:** Ensures user belongs to the organization
4. **Returns only IDs:** Lightweight - just the context IDs needed for filtering

#### Usage Examples

```go
// Example 1: Get user's accessible projects
projectIDs, err := assignmentRepository.GetUserContexts(ctx, 19, "project", 10)
// Returns: [30, 45, 67]
// Use: SELECT * FROM project.projects WHERE id IN (30, 45, 67)

// Example 2: Get user's accessible locations
locationIDs, err := assignmentRepository.GetUserContexts(ctx, 19, "location", 10)
// Returns: [6, 7]
// Use: SELECT * FROM project.projects WHERE location_id IN (6, 7)

// Example 3: Check for org-level access
orgIDs, err := assignmentRepository.GetUserContexts(ctx, 19, "organization", 10)
// Returns: [10] if user has org-level access
// Returns: [] if user doesn't have org-level access
```

---

## Complete Example: GET /projects

This is the CANONICAL example of access control implementation. Every service should follow this pattern.

**File:** `/src/infrastructure-project-management/main.go`
**Handler:** `handleGetProjects`
**Lines:** 131-279

### Complete Code Walkthrough

```go
// handleGetProjects handles GET /projects with access control
func handleGetProjects(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
    // STEP 1: Extract user information from JWT claims
    userID := claims.UserID          // Internal user ID
    orgID := claims.OrgID            // Organization ID
    isSuperAdmin := claims.IsSuperAdmin  // Super admin flag

    logger.WithFields(logrus.Fields{
        "user_id":        userID,
        "org_id":         orgID,
        "is_super_admin": isSuperAdmin,
        "operation":      "handleGetProjects",
    }).Debug("Processing GET /projects request with access control")

    // STEP 2: Check for optional location_id query parameter
    locationIDStr, hasLocationID := request.QueryStringParameters["location_id"]
    var locationID int64
    if hasLocationID && locationIDStr != "" {
        var parseErr error
        locationID, parseErr = strconv.ParseInt(locationIDStr, 10, 64)
        if parseErr != nil {
            return api.ErrorResponse(http.StatusBadRequest, "Invalid location_id parameter", logger), nil
        }
    }

    var projects []models.Project
    var err error

    // STEP 3: SUPER ADMIN CHECK - Bypass all access control
    if isSuperAdmin {
        logger.Debug("User is super admin - returning all projects")

        if hasLocationID && locationIDStr != "" {
            // Filter by location if provided
            projects, err = projectRepository.GetProjectsByLocationID(ctx, locationID, orgID)
        } else {
            // Return ALL projects in organization
            projects, err = projectRepository.GetProjectsByOrg(ctx, orgID)
        }

        if err != nil {
            logger.WithError(err).Error("Failed to get projects for super admin")
            return api.ErrorResponse(http.StatusInternalServerError, "Failed to get projects", logger), nil
        }
    } else {
        // STEP 4: NON-ADMIN ACCESS CONTROL

        // Check 4a: Organization-level assignment
        orgContexts, err := assignmentRepository.GetUserContexts(ctx, userID, "organization", orgID)
        if err != nil {
            logger.WithError(err).Error("Failed to check org-level assignments")
            return api.ErrorResponse(http.StatusInternalServerError, "Failed to check permissions", logger), nil
        }

        if len(orgContexts) > 0 {
            // User has org-level access - sees all projects
            logger.Debug("User has org-level assignment - returning all projects")

            if hasLocationID && locationIDStr != "" {
                projects, err = projectRepository.GetProjectsByLocationID(ctx, locationID, orgID)
            } else {
                projects, err = projectRepository.GetProjectsByOrg(ctx, orgID)
            }

            if err != nil {
                logger.WithError(err).Error("Failed to get projects for org-level user")
                return api.ErrorResponse(http.StatusInternalServerError, "Failed to get projects", logger), nil
            }
        } else {
            // Check 4b: Location-level assignment
            locationContexts, err := assignmentRepository.GetUserContexts(ctx, userID, "location", orgID)
            if err != nil {
                logger.WithError(err).Error("Failed to check location-level assignments")
                return api.ErrorResponse(http.StatusInternalServerError, "Failed to check permissions", logger), nil
            }

            if len(locationContexts) > 0 {
                // User has location-level access
                logger.WithField("location_count", len(locationContexts)).Debug("User has location-level assignments")

                if hasLocationID && locationIDStr != "" {
                    // IMPORTANT: Check if user has access to THIS specific location
                    hasAccess := false
                    for _, locID := range locationContexts {
                        if locID == locationID {
                            hasAccess = true
                            break
                        }
                    }

                    if !hasAccess {
                        // User tried to access a location they don't have access to
                        logger.Warn("User does not have access to requested location")
                        return api.ErrorResponse(http.StatusForbidden, "You do not have access to projects at this location", logger), nil
                    }

                    // User has access - return projects at this location
                    projects, err = projectRepository.GetProjectsByLocationID(ctx, locationID, orgID)
                } else {
                    // No location specified - force location selection
                    // This is intentional UX: location-level users must select a location first
                    logger.Debug("Location filter required for location-level user")
                    projects = []models.Project{}
                }

                if err != nil {
                    logger.WithError(err).Error("Failed to get projects for location-level user")
                    return api.ErrorResponse(http.StatusInternalServerError, "Failed to get projects", logger), nil
                }
            } else {
                // Check 4c: Project-level assignment
                projectContexts, err := assignmentRepository.GetUserContexts(ctx, userID, "project", orgID)
                if err != nil {
                    logger.WithError(err).Error("Failed to check project-level assignments")
                    return api.ErrorResponse(http.StatusInternalServerError, "Failed to check permissions", logger), nil
                }

                if len(projectContexts) > 0 {
                    // User has project-level access - only sees assigned projects
                    logger.WithField("project_count", len(projectContexts)).Debug("User has project-level assignments")

                    if hasLocationID && locationIDStr != "" {
                        // Get all projects at the location
                        allProjects, err := projectRepository.GetProjectsByLocationID(ctx, locationID, orgID)
                        if err != nil {
                            logger.WithError(err).Error("Failed to get projects by location")
                            return api.ErrorResponse(http.StatusInternalServerError, "Failed to get projects", logger), nil
                        }

                        // Filter to only projects user has access to
                        projects = filterProjectsByIDs(allProjects, projectContexts)
                    } else {
                        // Get only assigned projects (across all locations)
                        projects, err = projectRepository.GetProjectsByIDs(ctx, projectContexts, orgID)
                        if err != nil {
                            logger.WithError(err).Error("Failed to get projects by IDs")
                            return api.ErrorResponse(http.StatusInternalServerError, "Failed to get projects", logger), nil
                        }
                    }
                } else {
                    // No assignments at any level - user has no project access
                    logger.Warn("User has no assignments - no project access")
                    projects = []models.Project{}
                }
            }
        }
    }

    // STEP 5: Return filtered results
    response := models.ProjectListResponse{
        Projects: projects,
        Total:    len(projects),
    }

    logger.WithField("project_count", len(projects)).Debug("Returning projects")
    return api.SuccessResponse(http.StatusOK, response, logger), nil
}
```

### Helper Function: filterProjectsByIDs

```go
// filterProjectsByIDs filters projects to only those in the allowed ID list
// Used when project-level users request projects at a specific location
func filterProjectsByIDs(projects []models.Project, allowedIDs []int64) []models.Project {
    // Build lookup map for O(1) checking
    idMap := make(map[int64]bool)
    for _, id := range allowedIDs {
        idMap[id] = true
    }

    // Filter projects
    filtered := []models.Project{}
    for _, project := range projects {
        if idMap[project.ProjectID] {
            filtered = append(filtered, project)
        }
    }
    return filtered
}
```

### Access Control Decision Matrix

| User Type | location_id Provided? | Result |
|-----------|----------------------|--------|
| Super Admin | No | ALL projects in org |
| Super Admin | Yes | ALL projects at location |
| Org-level | No | ALL projects in org |
| Org-level | Yes | ALL projects at location |
| Location-level | No | Empty list (force selection) |
| Location-level | Yes (has access) | ALL projects at location |
| Location-level | Yes (no access) | 403 Forbidden |
| Project-level | No | ALL assigned projects |
| Project-level | Yes | Assigned projects at location |
| No assignments | Any | Empty list |

---

## GetUserContexts Method

### Why This Method is Critical

`GetUserContexts()` is called in EVERY service that needs to filter data:

- **Project Management:** Filter visible projects
- **Issue Management:** Filter visible issues
- **RFI Management:** Filter visible RFIs
- **Submittal Management:** Filter visible submittals
- **Attachment Management:** Filter visible attachments

### Method Details

**Location:** `/src/lib/data/assignment_repository.go` (lines 732-764)

```go
// GetUserContexts gets all context IDs of a specific type that a user has access to
// This is THE CORE method for access control filtering across the entire platform
func (dao *AssignmentDao) GetUserContexts(
    ctx context.Context,
    userID int64,
    contextType string,
    orgID int64,
) ([]int64, error) {
    query := `
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
    `

    rows, err := dao.DB.QueryContext(ctx, query, userID, contextType, orgID)
    if err != nil {
        return nil, fmt.Errorf("failed to get user contexts: %w", err)
    }
    defer rows.Close()

    var contextIDs []int64
    for rows.Next() {
        var contextID int64
        if err := rows.Scan(&contextID); err != nil {
            dao.Logger.WithError(err).Error("Failed to scan context ID")
            continue
        }
        contextIDs = append(contextIDs, contextID)
    }

    return contextIDs, nil
}
```

### Query Breakdown

```sql
SELECT DISTINCT ua.context_id
FROM iam.user_assignments ua
LEFT JOIN iam.users u ON ua.user_id = u.id
WHERE
    ua.user_id = $1                                      -- Filter to specific user
    AND ua.context_type = $2                             -- Filter to context type (project/location/org)
    AND u.org_id = $3                                    -- Organization isolation
    AND ua.is_deleted = FALSE                            -- Exclude soft-deleted assignments
    AND (ua.start_date IS NULL OR ua.start_date <= NOW()) -- Assignment has started
    AND (ua.end_date IS NULL OR ua.end_date >= NOW())     -- Assignment hasn't expired
ORDER BY ua.context_id
```

### Real-World Usage Examples

#### Example 1: RFI Service Filtering

```go
// File: infrastructure-rfi-management/main.go

func handleGetRFIs(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) {
    userID := claims.UserID
    orgID := claims.OrgID

    // Get user's accessible projects
    projectIDs, err := assignmentRepository.GetUserContexts(ctx, userID, "project", orgID)
    if err != nil {
        return api.ErrorResponse(http.StatusInternalServerError, "Failed to check permissions", logger), nil
    }

    // Query RFIs only from accessible projects
    rfis, err := rfiRepository.GetRFIsByProjectIDs(ctx, projectIDs, orgID)

    return api.SuccessResponse(http.StatusOK, rfis, logger), nil
}
```

#### Example 2: Issue Service Filtering

```go
// File: infrastructure-issue-management/main.go

func handleGetIssues(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) {
    userID := claims.UserID
    orgID := claims.OrgID

    // Get user's accessible projects
    projectIDs, err := assignmentRepository.GetUserContexts(ctx, userID, "project", orgID)
    if err != nil {
        return api.ErrorResponse(http.StatusInternalServerError, "Failed to check permissions", logger), nil
    }

    if len(projectIDs) == 0 {
        // User has no project access - return empty
        return api.SuccessResponse(http.StatusOK, []models.Issue{}, logger), nil
    }

    // Query issues only from accessible projects
    issues, err := issueRepository.GetIssuesByProjectIDs(ctx, projectIDs, orgID)

    return api.SuccessResponse(http.StatusOK, issues, logger), nil
}
```

---

## Migration from Legacy System

### What Was Replaced (October 2025)

The unified `user_assignments` table **replaced 5 separate tables**:

#### 1. ❌ `iam.org_user_roles` (DEPRECATED)
```sql
-- OLD system - organization assignments
CREATE TABLE iam.org_user_roles (
    user_id, role_id, org_id, ...
);
```

**Replaced by:**
```sql
-- NEW system
INSERT INTO iam.user_assignments (user_id, role_id, context_type, context_id, ...)
VALUES (user_id, role_id, 'organization', org_id, ...);
```

#### 2. ❌ `iam.location_user_roles` (DEPRECATED)
```sql
-- OLD system - location assignments
CREATE TABLE iam.location_user_roles (
    user_id, role_id, location_id, ...
);
```

**Replaced by:**
```sql
-- NEW system
INSERT INTO iam.user_assignments (user_id, role_id, context_type, context_id, ...)
VALUES (user_id, role_id, 'location', location_id, ...);
```

#### 3. ❌ `iam.user_location_access` (DEPRECATED)
- Simple mapping table
- Never implemented proper role-based access
- Replaced by location-level assignments

#### 4. ❌ `project.project_user_roles` (DEPRECATED)
```sql
-- OLD system - project assignments
CREATE TABLE project.project_user_roles (
    user_id, role_id, project_id, trade_type, ...
);
```

**Replaced by:**
```sql
-- NEW system
INSERT INTO iam.user_assignments (user_id, role_id, context_type, context_id, trade_type, ...)
VALUES (user_id, role_id, 'project', project_id, trade_type, ...);
```

#### 5. ❌ `project.project_managers` (DEPRECATED)
- Redundant table
- Never used in production
- Functionality covered by project-level assignments

### Migration SQL

```sql
-- Step 1: Migrate organization assignments
INSERT INTO iam.user_assignments (
    user_id, role_id, context_type, context_id,
    created_by, updated_by, created_at, updated_at, is_deleted
)
SELECT
    user_id, role_id, 'organization', org_id,
    created_by, updated_by, created_at, updated_at, is_deleted
FROM iam.org_user_roles
WHERE is_deleted = FALSE;

-- Step 2: Migrate location assignments
INSERT INTO iam.user_assignments (
    user_id, role_id, context_type, context_id,
    created_by, updated_by, created_at, updated_at, is_deleted
)
SELECT
    user_id, role_id, 'location', location_id,
    created_by, updated_by, created_at, updated_at, is_deleted
FROM iam.location_user_roles
WHERE is_deleted = FALSE;

-- Step 3: Migrate project assignments
INSERT INTO iam.user_assignments (
    user_id, role_id, context_type, context_id, trade_type, is_primary,
    start_date, end_date, created_by, updated_by, created_at, updated_at, is_deleted
)
SELECT
    user_id, role_id, 'project', project_id, trade_type, is_primary,
    start_date, end_date, created_by, updated_by, created_at, updated_at, is_deleted
FROM project.project_user_roles
WHERE is_deleted = FALSE;

-- Step 4: Verify migration
SELECT context_type, COUNT(*)
FROM iam.user_assignments
WHERE is_deleted = FALSE
GROUP BY context_type;
```

### Post-Migration Cleanup

```sql
-- After verifying new system works, drop old tables
DROP TABLE IF EXISTS iam.org_user_roles;
DROP TABLE IF EXISTS iam.location_user_roles;
DROP TABLE IF EXISTS iam.user_location_access;
DROP TABLE IF EXISTS project.project_user_roles;
DROP TABLE IF EXISTS project.project_managers;
```

### API Endpoint Changes

**Old Endpoints (DEPRECATED):**
```
POST   /projects/{projectId}/assign-user          # Old project assignment
GET    /projects/{projectId}/users                # Old project users
DELETE /projects/{projectId}/users/{userId}       # Old user removal
```

**New Endpoints (Current):**
```
POST   /assignments                               # Universal assignment creation
GET    /contexts/project/{projectId}/assignments  # Get project team
DELETE /assignments/{assignmentId}                # Delete any assignment

# Also works through project endpoints:
POST   /projects/{projectId}/users                # Wrapper around /assignments
GET    /projects/{projectId}/users                # Wrapper around /contexts/project/{id}/assignments
```

---

## Security Model

### 1. Organization Isolation

**Every query is scoped to the user's organization.**

```go
// Always validate org_id matches user's organization
if resourceOrgID != claims.OrgID {
    return api.ErrorResponse(http.StatusForbidden, "Access denied", logger), nil
}
```

```sql
-- Database queries ALWAYS filter by org_id
SELECT * FROM project.projects
WHERE org_id = $1  -- User's organization
  AND id IN (...)  -- Additional access filters
```

### 2. Super Admin Bypass Pattern

```go
// Super admins bypass restrictions but still respect org boundaries
if claims.IsSuperAdmin {
    // Can access everything in THEIR organization
    return projectRepository.GetProjectsByOrg(ctx, claims.OrgID)
}
```

**Important:** Even super admins are scoped to their organization. They cannot access other organizations' data.

### 3. Soft Deletes for Audit Trail

```sql
-- Assignments are NEVER hard-deleted
UPDATE iam.user_assignments
SET is_deleted = TRUE, updated_by = $1, updated_at = NOW()
WHERE id = $2;

-- All queries filter soft-deleted records
WHERE is_deleted = FALSE
```

**Benefits:**
- Complete audit trail of who had access when
- Ability to restore accidentally deleted assignments
- Compliance with data retention requirements
- Historical reporting capabilities

### 4. Time-Bound Access Control

```sql
-- Assignments can automatically expire
INSERT INTO iam.user_assignments (
    ..., start_date, end_date
) VALUES (
    ..., '2025-11-01', '2026-01-31'  -- Active only for 3 months
);

-- Queries automatically check date ranges
WHERE (start_date IS NULL OR start_date <= NOW())
  AND (end_date IS NULL OR end_date >= NOW())
```

**Use Cases:**
- Temporary contractors
- Seasonal workers
- Project-specific consultants
- Audit/compliance periods

### 5. Role-Based Permissions (Future Enhancement)

**Current State:** Assignments grant access, but don't enforce specific permissions yet.

**Planned:**
```sql
-- Check if user has specific permission in context
SELECT EXISTS(
    SELECT 1
    FROM iam.user_assignments ua
    JOIN iam.role_permissions rp ON ua.role_id = rp.role_id
    JOIN iam.permissions p ON rp.permission_id = p.id
    WHERE ua.user_id = $1
      AND ua.context_type = $2
      AND ua.context_id = $3
      AND p.name = $4  -- e.g., 'rfis:create'
      AND ua.is_deleted = FALSE
)
```

---

## Best Practices

### 1. Always Use GetUserContexts for Filtering

❌ **WRONG:**
```go
// NEVER query all resources and filter in application code
allProjects, _ := projectRepository.GetProjectsByOrg(ctx, orgID)
filteredProjects := filterByUserAccess(allProjects, userID)  // BAD!
```

✅ **CORRECT:**
```go
// ALWAYS get accessible context IDs first, then query database
projectIDs, err := assignmentRepository.GetUserContexts(ctx, userID, "project", orgID)
projects, err := projectRepository.GetProjectsByIDs(ctx, projectIDs, orgID)
```

**Why:** Database filtering is orders of magnitude faster and more secure.

---

### 2. Implement the Standard Access Control Flow

Every endpoint that returns resources should follow this pattern:

```go
func handleGetResources(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) {
    // 1. Extract claims
    userID := claims.UserID
    orgID := claims.OrgID
    isSuperAdmin := claims.IsSuperAdmin

    // 2. Super admin bypass
    if isSuperAdmin {
        return getAllResources(ctx, orgID)
    }

    // 3. Check org-level access
    orgContexts, _ := assignmentRepository.GetUserContexts(ctx, userID, "organization", orgID)
    if len(orgContexts) > 0 {
        return getAllResources(ctx, orgID)
    }

    // 4. Check location-level access
    locationContexts, _ := assignmentRepository.GetUserContexts(ctx, userID, "location", orgID)
    if len(locationContexts) > 0 {
        return getResourcesByLocations(ctx, locationContexts, orgID)
    }

    // 5. Check project-level access
    projectContexts, _ := assignmentRepository.GetUserContexts(ctx, userID, "project", orgID)
    if len(projectContexts) > 0 {
        return getResourcesByProjects(ctx, projectContexts, orgID)
    }

    // 6. No access
    return []Resource{}, nil
}
```

---

### 3. Validate Context Existence Before Creating Assignments

```go
// ALWAYS validate the context exists before creating assignment
err := assignmentRepository.ValidateAssignmentContext(ctx, "project", projectID, orgID)
if err != nil {
    return api.ErrorResponse(http.StatusBadRequest, "Invalid project ID", logger), nil
}

// Now create assignment
assignment, err := assignmentRepository.CreateAssignment(ctx, request, userID)
```

The `ValidateAssignmentContext` method checks:
1. Context exists (project/location/org record exists)
2. Context is not soft-deleted
3. Context belongs to the user's organization (prevents cross-org access)

---

### 4. Use Enriched Assignment Responses

```go
// GetAssignment returns enriched data with JOINs
assignment, err := assignmentRepository.GetAssignment(ctx, assignmentID, orgID)

// Returns:
// {
//   "id": 123,
//   "user_id": 16,
//   "role_id": 8,
//   "user_name": "John Doe",           // Enriched from users table
//   "user_email": "john@example.com",  // Enriched from users table
//   "role_name": "Project Manager",     // Enriched from roles table
//   "context_name": "Downtown Office"   // Enriched from projects/locations/orgs
// }
```

**Benefits:**
- Single query instead of multiple
- Reduces frontend complexity
- Consistent data format

---

### 5. Handle Empty Access Gracefully

```go
projectContexts, err := assignmentRepository.GetUserContexts(ctx, userID, "project", orgID)
if err != nil {
    return api.ErrorResponse(http.StatusInternalServerError, "Failed to check permissions", logger), nil
}

if len(projectContexts) == 0 {
    // User has no project access - return empty array, NOT an error
    return api.SuccessResponse(http.StatusOK, []models.Project{}, logger), nil
}
```

**Why:** Empty access is not an error. It's a valid state (new user, limited access, etc.)

---

### 6. Log Access Control Decisions

```go
logger.WithFields(logrus.Fields{
    "user_id":          userID,
    "org_id":           orgID,
    "is_super_admin":   isSuperAdmin,
    "access_level":     "project",  // or "org" or "location"
    "accessible_count": len(projectContexts),
    "operation":        "handleGetProjects",
}).Debug("Access control check completed")
```

**Benefits:**
- Easier debugging of permission issues
- Audit trail for security reviews
- Performance monitoring

---

### 7. Use Bulk Operations for Team Management

```go
// Assign multiple users to a project in one operation
bulkRequest := &models.BulkAssignmentRequest{
    UserIDs:     []int64{16, 21, 27, 29},
    RoleID:      8,
    ContextType: "project",
    ContextID:   30,
    StartDate:   "2025-11-01",
}

assignments, err := assignmentRepository.CreateBulkAssignments(ctx, bulkRequest, createdBy)
```

**Benefits:**
- Single transaction for all assignments
- Atomic operation (all succeed or all fail)
- Better performance than multiple individual calls

---

### 8. Never Hard-Code Access Checks

❌ **WRONG:**
```go
// NEVER hard-code role checks
if userRole == "Project Manager" {
    // Allow access
}
```

✅ **CORRECT:**
```go
// ALWAYS check assignments dynamically
projectContexts, err := assignmentRepository.GetUserContexts(ctx, userID, "project", orgID)
if contains(projectContexts, projectID) {
    // User has access to this project
}
```

**Why:** Hard-coded checks break when roles change and don't respect the hierarchical access model.

---

### 9. Respect the Location-First UI Pattern

For location-level users, **require location selection**:

```go
if len(locationContexts) > 0 {
    if !locationIDProvided {
        // Force user to select a location
        return api.SuccessResponse(http.StatusOK, []models.Project{}, logger), nil
    }

    // Validate user has access to the requested location
    if !contains(locationContexts, locationID) {
        return api.ErrorResponse(http.StatusForbidden, "Access denied to this location", logger), nil
    }

    // Return projects at selected location
    return projectRepository.GetProjectsByLocationID(ctx, locationID, orgID)
}
```

**Why:** This matches real-world workflows where users work at one location at a time.

---

### 10. Document Access Control in API Responses

```go
// Include access-related metadata in responses
response := models.ProjectListResponse{
    Projects:    projects,
    Total:       len(projects),
    AccessLevel: "project",  // Indicates user's access level
    Filtered:    true,       // Indicates filtering was applied
}
```

**Benefits:**
- Frontend knows why they're seeing certain data
- Debugging is easier
- Better user experience

---

## Summary

The access control system is the **foundational security layer** of the BuildBoard platform. Understanding it is essential for:

- Building new features
- Debugging permission issues
- Ensuring data security
- Maintaining compliance

### Key Takeaways

1. **Everything flows from `user_assignments` table** - this is THE source of truth
2. **`GetUserContexts()` is THE core filtering method** - used in every service
3. **Hierarchical access model:** Super Admin → Org → Location → Project
4. **Always filter at the database level** - never in application code
5. **Follow the standard access control flow** - it's consistent across all services
6. **Token Customizer populates location dropdown** - based on user assignments
7. **GET /projects is the canonical example** - every service should follow this pattern
8. **Migration complete (October 2025)** - all services now use unified system

### Reference Implementations

**Study these files to understand access control:**

1. **Assignment Repository:** `/src/lib/data/assignment_repository.go`
   - `GetUserContexts()` method (lines 732-764)
   - Core access control logic

2. **Project Management Handler:** `/src/infrastructure-project-management/main.go`
   - `handleGetProjects()` (lines 131-279)
   - Complete access control implementation

3. **Token Customizer:** `/src/infrastructure-token-customizer/main.go`
   - `GetUserProfile()` in user_repository.go
   - Location discovery for JWT tokens

4. **User Repository:** `/src/lib/data/user_repository.go`
   - `GetUserProfile()` method
   - Access context aggregation

### The Golden Rules

1. **EVERY endpoint filters by user access** - no exceptions
2. **Super admins bypass but respect org boundaries**
3. **Use `GetUserContexts()` for all filtering**
4. **Validate context existence before assignment**
5. **Soft delete for audit trail**
6. **Log access control decisions**
7. **Handle empty access gracefully**
8. **Follow the hierarchical model**

---

**This document is the single source of truth for access control architecture.**

When in doubt, refer back to this document and the canonical implementations listed above.