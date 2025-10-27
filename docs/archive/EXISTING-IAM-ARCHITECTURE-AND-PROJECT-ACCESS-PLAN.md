# Existing IAM Architecture & Project Access Implementation Plan

## 🔍 CURRENT SYSTEM ANALYSIS (What Already Exists)

### 1. **Three-Tier Role Assignment System** ✅ Already Implemented

The system has **THREE distinct role assignment levels**:

```
┌─────────────────────────────────────────────────────────────┐
│                  USER ACCESS HIERARCHY                       │
└─────────────────────────────────────────────────────────────┘

Level 1: ORGANIZATION-LEVEL ROLES
├─ Table: iam.org_user_roles
├─ Assigns: user_id + role_id (org-wide)
└─ Access: ALL projects in the organization

Level 2: LOCATION-LEVEL ROLES
├─ Table: iam.location_user_roles
├─ Assigns: user_id + location_id + role_id
└─ Access: ALL projects in specific locations

Level 3: PROJECT-LEVEL ROLES
├─ Table: project.project_user_roles
├─ Assigns: user_id + project_id + role_id
└─ Access: ONLY specific projects
```

### 2. **Role Definition with `access_level`** ✅ Key Field

From `iam.roles` table:
```sql
access_level: 'organization' | 'location' | 'project'
```

This field defines **WHAT SCOPE** the role operates at:

| Access Level | Meaning | Example Roles |
|--------------|---------|---------------|
| `organization` | Full org access | Super Admin, Company Admin, System Manager |
| `location` | Location-specific | General Superintendent, Site Supervisor |
| `project` | Project-specific | Project Manager, Foreman, Field Worker |

### 3. **Super Admin Flag** ✅ Platform-Level Access

From `iam.users` table:
```sql
is_super_admin: boolean
```

**Included in JWT Claims**:
```go
type Claims struct {
    UserID       int64
    OrgID        int64
    IsSuperAdmin bool  // ← This is already in JWT!
    ...
}
```

### 4. **Current Authorization Pattern** ✅ Partial Implementation

**User Management API** (src/infrastructure-user-management/main.go:55-58):
```go
if !claims.IsSuperAdmin {
    return api.ErrorResponse(http.StatusForbidden,
        "Forbidden: Only super admins can manage users", logger)
}
```

**Project Management API** (src/infrastructure-project-management/main.go):
- ❌ **NO access level checking**
- Currently returns ALL projects in org via `GetProjectsByOrg(orgID)`
- Does NOT filter by user's role assignments

---

## 🎯 THE PROBLEM

### What's Missing:

1. ❌ **No detection of user's access level** (org/location/project)
2. ❌ **No filtering based on role assignments**
3. ❌ **`GET /projects` returns ALL org projects** regardless of user's actual access
4. ❌ **No way to get "My Assigned Projects"**

### Current Behavior:
```
User with project-level role → Sees ALL org projects ❌
User with location-level role → Sees ALL org projects ❌
User with org-level role → Sees ALL org projects ✓
```

### Desired Behavior:
```
User with project-level role → Sees ONLY assigned projects ✓
User with location-level role → Sees projects in assigned locations ✓
User with org-level role → Sees ALL org projects ✓
Super Admin → Sees EVERYTHING across all orgs ✓
```

---

## 📋 IMPLEMENTATION PLAN (Using Existing Architecture)

### Phase 1: Add Helper to Detect User's Access Level

**File**: `src/lib/auth/access_context.go` (NEW)

```go
package auth

import (
    "context"
    "database/sql"
)

// AccessContext represents user's complete access scope
type AccessContext struct {
    UserID       int64
    OrgID        int64
    IsSuperAdmin bool

    // Derived access information
    HasOrgRole      bool    // Has role in org_user_roles
    OrgRoleIDs      []int64 // Role IDs from org level

    HasLocationRole bool    // Has role in location_user_roles
    LocationIDs     []int64 // Locations user has access to
    LocationRoleIDs []int64 // Role IDs from location level

    HasProjectRole  bool    // Has role in project_user_roles
    ProjectIDs      []int64 // Projects explicitly assigned
    ProjectRoleIDs  []int64 // Role IDs from project level

    // Computed access level (highest level of access)
    EffectiveAccessLevel string // "super_admin" | "org" | "location" | "project"
}

// GetUserAccessContext fetches user's complete access context
func GetUserAccessContext(ctx context.Context, db *sql.DB, userID, orgID int64, isSuperAdmin bool) (*AccessContext, error) {
    accessCtx := &AccessContext{
        UserID:       userID,
        OrgID:        orgID,
        IsSuperAdmin: isSuperAdmin,
    }

    // If super admin, they have full access
    if isSuperAdmin {
        accessCtx.EffectiveAccessLevel = "super_admin"
        return accessCtx, nil
    }

    // Check org-level roles
    orgRoles, err := getOrgRoles(ctx, db, userID, orgID)
    if err != nil {
        return nil, err
    }
    if len(orgRoles) > 0 {
        accessCtx.HasOrgRole = true
        accessCtx.OrgRoleIDs = orgRoles
        accessCtx.EffectiveAccessLevel = "org"
        return accessCtx, nil // Org admin has full org access
    }

    // Check location-level roles
    locations, locationRoles, err := getLocationRoles(ctx, db, userID)
    if err != nil {
        return nil, err
    }
    if len(locations) > 0 {
        accessCtx.HasLocationRole = true
        accessCtx.LocationIDs = locations
        accessCtx.LocationRoleIDs = locationRoles
        accessCtx.EffectiveAccessLevel = "location"
        return accessCtx, nil
    }

    // Check project-level roles
    projects, projectRoles, err := getProjectRoles(ctx, db, userID)
    if err != nil {
        return nil, err
    }
    if len(projects) > 0 {
        accessCtx.HasProjectRole = true
        accessCtx.ProjectIDs = projects
        accessCtx.ProjectRoleIDs = projectRoles
        accessCtx.EffectiveAccessLevel = "project"
        return accessCtx, nil
    }

    // No roles assigned - no access
    accessCtx.EffectiveAccessLevel = "none"
    return accessCtx, nil
}

// Helper functions
func getOrgRoles(ctx context.Context, db *sql.DB, userID, orgID int64) ([]int64, error) {
    query := `
        SELECT our.role_id
        FROM iam.org_user_roles our
        JOIN iam.roles r ON our.role_id = r.id
        WHERE our.user_id = $1
          AND r.org_id = $2
          AND r.access_level = 'organization'
          AND our.is_deleted = false
          AND r.is_deleted = false
    `
    rows, err := db.QueryContext(ctx, query, userID, orgID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var roleIDs []int64
    for rows.Next() {
        var roleID int64
        if err := rows.Scan(&roleID); err != nil {
            return nil, err
        }
        roleIDs = append(roleIDs, roleID)
    }
    return roleIDs, nil
}

func getLocationRoles(ctx context.Context, db *sql.DB, userID int64) ([]int64, []int64, error) {
    query := `
        SELECT DISTINCT lur.location_id, lur.role_id
        FROM iam.location_user_roles lur
        JOIN iam.roles r ON lur.role_id = r.id
        WHERE lur.user_id = $1
          AND r.access_level = 'location'
          AND lur.is_deleted = false
          AND r.is_deleted = false
    `
    rows, err := db.QueryContext(ctx, query, userID)
    if err != nil {
        return nil, nil, err
    }
    defer rows.Close()

    var locationIDs, roleIDs []int64
    for rows.Next() {
        var locationID, roleID int64
        if err := rows.Scan(&locationID, &roleID); err != nil {
            return nil, nil, err
        }
        locationIDs = append(locationIDs, locationID)
        roleIDs = append(roleIDs, roleID)
    }
    return locationIDs, roleIDs, nil
}

func getProjectRoles(ctx context.Context, db *sql.DB, userID int64) ([]int64, []int64, error) {
    query := `
        SELECT DISTINCT pur.project_id, pur.role_id
        FROM project.project_user_roles pur
        JOIN iam.roles r ON pur.role_id = r.id
        WHERE pur.user_id = $1
          AND r.access_level = 'project'
          AND pur.is_deleted = false
          AND r.is_deleted = false
    `
    rows, err := db.QueryContext(ctx, query, userID)
    if err != nil {
        return nil, nil, err
    }
    defer rows.Close()

    var projectIDs, roleIDs []int64
    for rows.Next() {
        var projectID, roleID int64
        if err := rows.Scan(&projectID, &roleID); err != nil {
            return nil, nil, err
        }
        projectIDs = append(projectIDs, projectID)
        roleIDs = append(roleIDs, roleID)
    }
    return projectIDs, roleIDs, nil
}
```

### Phase 2: Update Project Repository

**File**: `src/lib/data/project_repository.go`

Add new method:
```go
// GetProjectsForUser returns projects based on user's access level
func (dao *ProjectDao) GetProjectsForUser(
    ctx context.Context,
    accessCtx *auth.AccessContext,
    filters map[string]string,
) ([]models.Project, error) {

    var query string
    var args []interface{}

    switch accessCtx.EffectiveAccessLevel {
    case "super_admin":
        // Super admin sees EVERYTHING (even across orgs if needed)
        query = `SELECT * FROM project.projects WHERE is_deleted = FALSE`

    case "org":
        // Org-level role: see ALL projects in organization
        query = `
            SELECT * FROM project.projects
            WHERE org_id = $1 AND is_deleted = FALSE
        `
        args = append(args, accessCtx.OrgID)

    case "location":
        // Location-level role: see projects in assigned locations
        query = `
            SELECT * FROM project.projects
            WHERE location_id = ANY($1)
              AND org_id = $2
              AND is_deleted = FALSE
        `
        args = append(args, pq.Array(accessCtx.LocationIDs), accessCtx.OrgID)

    case "project":
        // Project-level role: see ONLY assigned projects
        query = `
            SELECT * FROM project.projects
            WHERE id = ANY($1)
              AND org_id = $2
              AND is_deleted = FALSE
        `
        args = append(args, pq.Array(accessCtx.ProjectIDs), accessCtx.OrgID)

    default:
        // No access
        return []models.Project{}, nil
    }

    // Add optional filters (location_id, status, etc.)
    if locationID := filters["location_id"]; locationID != "" {
        // Add location filter to query
    }

    // Add pagination
    // Add ORDER BY

    rows, err := dao.DB.QueryContext(ctx, query, args...)
    // ... scan and return projects
}
```

### Phase 3: Update Project Management API

**File**: `src/infrastructure-project-management/main.go`

```go
// handleGetProjects - UPDATED
func handleGetProjects(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
    // Get user's access context
    accessCtx, err := auth.GetUserAccessContext(ctx, sqlDB, claims.UserID, claims.OrgID, claims.IsSuperAdmin)
    if err != nil {
        logger.WithError(err).Error("Failed to get user access context")
        return api.ErrorResponse(http.StatusInternalServerError, "Failed to determine access level", logger), nil
    }

    // Get filters from query params
    filters := request.QueryStringParameters
    if filters == nil {
        filters = make(map[string]string)
    }

    // Fetch projects based on user's access level
    projects, err := projectRepository.GetProjectsForUser(ctx, accessCtx, filters)
    if err != nil {
        logger.WithError(err).Error("Failed to get projects")
        return api.ErrorResponse(http.StatusInternalServerError, "Failed to get projects", logger), nil
    }

    response := models.ProjectListResponse{
        Projects: projects,
        Total:    len(projects),
        AccessLevel: accessCtx.EffectiveAccessLevel, // ← Tell frontend user's access level
    }

    return api.SuccessResponse(http.StatusOK, response, logger), nil
}
```

### Phase 4: Add "My Projects" Endpoint (Optional)

**Explicitly returns user's assigned projects regardless of access level**

```go
// handleGetMyProjects - NEW
case request.Resource == "/my-projects" && request.HTTPMethod == "GET":
    return handleGetMyProjects(ctx, request, claims)

func handleGetMyProjects(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
    // ALWAYS return project-level assignments (explicit assignments only)
    projects, err := projectRepository.GetUserAssignedProjects(ctx, claims.UserID, claims.OrgID)
    if err != nil {
        logger.WithError(err).Error("Failed to get assigned projects")
        return api.ErrorResponse(http.StatusInternalServerError, "Failed to get assigned projects", logger), nil
    }

    response := models.ProjectListResponse{
        Projects: projects,
        Total:    len(projects),
    }

    return api.SuccessResponse(http.StatusOK, response, logger), nil
}
```

---

## 🔒 ACCESS CONTROL MATRIX

| User Type | Access Level | `/projects` Returns | `/my-projects` Returns |
|-----------|--------------|---------------------|------------------------|
| **Super Admin** | `super_admin` | ALL projects (all orgs) | Projects explicitly assigned |
| **Org Admin** (via org_user_roles) | `org` | ALL projects in their org | Projects explicitly assigned |
| **Location Manager** (via location_user_roles) | `location` | Projects in their locations | Projects explicitly assigned |
| **Project User** (via project_user_roles) | `project` | ONLY assigned projects | ONLY assigned projects |
| **No Roles** | `none` | Empty list | Empty list |

---

## 🗂️ DATABASE QUERIES REFERENCE

### Check User's Org-Level Roles:
```sql
SELECT r.id, r.name, r.access_level
FROM iam.org_user_roles our
JOIN iam.roles r ON our.role_id = r.id
WHERE our.user_id = $1
  AND r.org_id = $2
  AND r.access_level = 'organization'
  AND our.is_deleted = false
  AND r.is_deleted = false;
```

### Check User's Location-Level Roles:
```sql
SELECT DISTINCT lur.location_id, r.id, r.name
FROM iam.location_user_roles lur
JOIN iam.roles r ON lur.role_id = r.id
WHERE lur.user_id = $1
  AND r.access_level = 'location'
  AND lur.is_deleted = false
  AND r.is_deleted = false;
```

### Check User's Project-Level Assignments:
```sql
SELECT DISTINCT pur.project_id, r.id, r.name
FROM project.project_user_roles pur
JOIN iam.roles r ON pur.role_id = r.id
WHERE pur.user_id = $1
  AND r.access_level = 'project'
  AND pur.is_deleted = false
  AND r.is_deleted = false;
```

---

## 📊 RESPONSE FORMAT

### Enhanced Project List Response:
```json
{
  "projects": [...],
  "total": 25,
  "page": 1,
  "page_size": 50,
  "access_level": "location",  // ← NEW: Tell frontend user's access level
  "accessible_locations": [24, 38]  // ← NEW: Optional context for location users
}
```

Frontend can use this to:
- Show/hide admin controls
- Display appropriate UI
- Enable/disable project creation
- Show access scope indicator

---

## 🚀 IMPLEMENTATION STEPS

### Step 1: Create Access Context Helper
- [ ] Create `src/lib/auth/access_context.go`
- [ ] Implement `GetUserAccessContext()`
- [ ] Add unit tests

### Step 2: Update Project Repository
- [ ] Add `GetProjectsForUser()` method
- [ ] Handle all access levels (super_admin, org, location, project)
- [ ] Support filters and pagination

### Step 3: Update Project Management API
- [ ] Modify `handleGetProjects()` to use access context
- [ ] Add access_level to response
- [ ] Test all user types

### Step 4: Add My Projects Endpoint (Optional)
- [ ] Add `/my-projects` route
- [ ] Implement `GetUserAssignedProjects()` repository method
- [ ] Test explicit assignments

### Step 5: Update API Gateway Routes
- [ ] Add `/my-projects` route to CDK
- [ ] Deploy infrastructure

### Step 6: Testing
- [ ] Test super admin access
- [ ] Test org-level role users
- [ ] Test location-level role users
- [ ] Test project-level role users
- [ ] Test users with no roles
- [ ] Test cross-org security

---

## 🎯 KEY BENEFITS

✅ **Uses Existing Architecture** - No database changes needed
✅ **Respects Three-Tier Role System** - Org/Location/Project levels
✅ **Backward Compatible** - No breaking changes
✅ **Secure by Default** - Users only see what they should
✅ **Performance Optimized** - Efficient queries per access level
✅ **Frontend-Friendly** - Response includes access level metadata

---

## ⚠️ IMPORTANT NOTES

1. **JWT Already Contains `is_super_admin`** ✅ No changes needed
2. **Role `access_level` field exists** ✅ Already designed correctly
3. **Three assignment tables exist** ✅ org/location/project separation
4. **No breaking changes** ✅ Enhances existing `/projects` endpoint
5. **Caching recommended** - Cache access context for 5 minutes

---

## 🔄 NEXT STEPS

**Review this plan with team, then:**
1. Approve approach
2. Create access context helper (Phase 1)
3. Update project repository (Phase 2)
4. Modify API endpoint (Phase 3)
5. Add comprehensive tests
6. Deploy and monitor