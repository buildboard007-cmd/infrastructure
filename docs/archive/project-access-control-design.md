# Project Access Control - Comprehensive Design Plan

## Current System Analysis

### Database Structure
```
iam.users
├── is_super_admin (boolean) - Super admin flag at user level
└── org_id - Organization membership

iam.org_user_roles
├── user_id - Link to user
└── role_id - Organization-level role

iam.roles
├── access_level (org/location/project)
├── category (admin/manager/member/etc)
└── role_type (system/custom)

project.project_user_roles
├── user_id - Specific user assignment
├── project_id - Project assignment
├── role_id - Role on this project
└── is_primary - Primary assignment flag
```

### Access Levels Hierarchy
1. **Super Admin** (`is_super_admin = true`)
   - Full access to ALL projects across ALL organizations
   - Platform-level admin (rare)

2. **Organization Admin** (Role with `access_level = 'org'`)
   - Full access to ALL projects in their organization
   - Can manage all users and projects in org

3. **Location Admin** (Role with `access_level = 'location'`)
   - Access to ALL projects in specific locations
   - Managed via `iam.location_user_roles`

4. **Project User** (Explicitly assigned via `project_user_roles`)
   - Access ONLY to projects they are explicitly assigned to
   - Regular team members, contractors, etc.

---

## API Design - GET /projects Endpoint

### Goal
Make `/projects` endpoint **versatile and intelligent** based on user's access level.

### Design Principles
1. **Security First**: Users should ONLY see projects they have access to
2. **Performance**: Efficient queries with proper indexing
3. **Flexibility**: Support filtering and pagination
4. **Clear Intent**: API behavior should be predictable

---

## Proposed API Endpoints

### 1. **GET /projects** (Main endpoint - context-aware)

**Behavior based on user type:**

| User Type | Query Parameter | Returns |
|-----------|----------------|---------|
| Super Admin | None | ALL projects across ALL orgs |
| Super Admin | `?org_id=X` | All projects in org X |
| Org Admin | None | ALL projects in their org |
| Org Admin | `?location_id=X` | Projects in location X within their org |
| Location Admin | None | ALL projects in their assigned locations |
| Location Admin | `?location_id=X` | Projects in specific location X (if they have access) |
| Project User | None | ONLY explicitly assigned projects |
| Project User | `?location_id=X` | Assigned projects filtered by location X |

**Query Parameters:**
```json
{
  "location_id": "optional - filter by location",
  "status": "optional - active/completed/archived",
  "page": "optional - pagination (default: 1)",
  "limit": "optional - page size (default: 50, max: 100)",
  "include_archived": "optional - boolean (default: false)"
}
```

**Response:**
```json
{
  "projects": [...],
  "total": 150,
  "page": 1,
  "page_size": 50,
  "has_next": true,
  "user_access_level": "org_admin|location_admin|project_user",
  "accessible_locations": [24, 38] // Optional: locations user has access to
}
```

### 2. **GET /my-projects** (Explicit endpoint for assigned projects)

**Purpose**: Always returns ONLY projects explicitly assigned to the user via `project_user_roles`

**Use Case**:
- User dashboard showing "My Active Projects"
- Mobile app showing user's workload
- Personal task list

**Returns**: Projects where `project_user_roles.user_id = current_user`

### 3. **GET /projects/{projectId}** (Existing - no change)

**Access Control**:
- Super Admin: Access any project
- Org Admin: Access any project in their org
- Location Admin: Access projects in their locations
- Project User: Access only if explicitly assigned

---

## Implementation Plan

### Phase 1: Add Access Level Detection Helper

```go
// Add to JWT claims or fetch from DB
type UserAccessContext struct {
    UserID       int64
    OrgID        int64
    IsSuperAdmin bool
    OrgRoles     []Role  // Roles at org level
    LocationIDs  []int64 // Locations user has access to
    IsOrgAdmin   bool    // Derived: has org-level admin role
    IsLocationAdmin bool // Derived: has location-level role
}

func GetUserAccessContext(ctx context.Context, userID, orgID int64) (*UserAccessContext, error) {
    // Query to determine user's access level
    // 1. Check is_super_admin
    // 2. Check org_user_roles for org-level admin
    // 3. Check location_user_roles for location access
    // Return comprehensive access context
}
```

### Phase 2: Implement Smart Query Builder

```go
func (repo *ProjectDao) GetProjectsForUser(
    ctx context.Context,
    userContext *UserAccessContext,
    filters map[string]string,
) ([]Project, error) {

    baseQuery := `SELECT DISTINCT p.* FROM project.projects p`

    if userContext.IsSuperAdmin {
        // No joins needed - access everything
        query = baseQuery + ` WHERE p.is_deleted = FALSE`

    } else if userContext.IsOrgAdmin {
        // Access all in org
        query = baseQuery + ` WHERE p.org_id = $1 AND p.is_deleted = FALSE`

    } else if userContext.IsLocationAdmin {
        // Access all in assigned locations
        query = baseQuery + `
            WHERE p.location_id = ANY($1)
            AND p.org_id = $2
            AND p.is_deleted = FALSE`

    } else {
        // Project user - only assigned projects
        query = baseQuery + `
            INNER JOIN project.project_user_roles pur
                ON p.id = pur.project_id
            WHERE pur.user_id = $1
            AND pur.is_deleted = FALSE
            AND p.is_deleted = FALSE`
    }

    // Add optional filters (location_id, status, etc.)
    // Add pagination
    // Execute query
}
```

### Phase 3: Update Existing Endpoints

**Current:**
```go
// GET /projects
handleGetProjects() {
    projects := projectRepository.GetProjectsByOrg(orgID)
}
```

**New:**
```go
// GET /projects
handleGetProjects() {
    userContext := GetUserAccessContext(ctx, claims.UserID, claims.OrgID)
    projects := projectRepository.GetProjectsForUser(ctx, userContext, filters)
}

// GET /my-projects
handleGetMyProjects() {
    // Always use project_user_roles regardless of admin status
    projects := projectRepository.GetAssignedProjects(ctx, claims.UserID, claims.OrgID)
}
```

---

## Access Control Matrix

| Action | Super Admin | Org Admin | Location Admin | Project User |
|--------|-------------|-----------|----------------|--------------|
| **View Projects** |
| All org projects | ✅ All orgs | ✅ Their org | ❌ | ❌ |
| Location projects | ✅ | ✅ | ✅ Their locations | ❌ |
| Assigned projects | ✅ | ✅ | ✅ | ✅ Only assigned |
| **Create Project** | ✅ | ✅ | ✅ In their locations | ❌ |
| **Edit Project** | ✅ Any | ✅ In their org | ✅ In their locations | ⚠️ Limited (if assigned) |
| **Delete Project** | ✅ Any | ✅ In their org | ⚠️ In their locations | ❌ |
| **Assign Users** | ✅ | ✅ | ✅ In their locations | ❌ |

---

## Database Optimization

### Recommended Indexes

```sql
-- For project user access queries
CREATE INDEX idx_project_user_roles_user_project
ON project.project_user_roles(user_id, project_id)
WHERE is_deleted = FALSE;

-- For location-based queries
CREATE INDEX idx_projects_location_org
ON project.projects(location_id, org_id)
WHERE is_deleted = FALSE;

-- For org admin queries
CREATE INDEX idx_projects_org_status
ON project.projects(org_id, status)
WHERE is_deleted = FALSE;
```

---

## Security Considerations

### 1. **Defense in Depth**
- Always validate org_id from JWT claims (not query params)
- Never trust frontend-provided user_id
- Validate project ownership before ANY modification

### 2. **Audit Trail**
```go
// Log all project access
logger.WithFields({
    "user_id": userID,
    "access_level": userContext.AccessLevel,
    "project_count": len(projects),
    "filters": filters,
}).Info("Project list accessed")
```

### 3. **Rate Limiting**
- Limit /projects endpoint to prevent data harvesting
- Higher limits for admins, lower for regular users

---

## Migration Strategy

### Step 1: Add Helper Functions (No breaking changes)
- Add `GetUserAccessContext()`
- Add `GetProjectsForUser()` to repository

### Step 2: Update GET /projects (Backward compatible)
- Enhance existing endpoint
- Returns same structure, just smarter filtering

### Step 3: Add GET /my-projects (New endpoint)
- Explicitly for user-assigned projects
- Helps with UI separation

### Step 4: Add Response Metadata
- Include `user_access_level` in response
- Frontend can adjust UI based on access level

---

## Frontend Integration

### Example Usage

```typescript
// Dashboard - show all accessible projects
const { data } = await api.get('/projects', {
  params: { location_id: currentLocation }
});

// User's personal view - only assigned projects
const { data } = await api.get('/my-projects');

// Admin panel - all org projects
const { data } = await api.get('/projects');
// Automatically shows all if user is org admin

// Conditional UI rendering
if (data.user_access_level === 'org_admin') {
  showAdminControls();
} else {
  showUserControls();
}
```

---

## Testing Scenarios

### Test Cases

1. **Super Admin**
   - Can see projects across multiple orgs
   - Can filter by org_id

2. **Org Admin**
   - Sees ALL projects in their org
   - Cannot see projects from other orgs
   - Can filter by location within org

3. **Location Admin**
   - Sees projects ONLY in assigned locations
   - Cannot see projects in other locations
   - Can manage projects in their locations

4. **Project User**
   - Sees ONLY assigned projects
   - Cannot see other projects even in same location
   - `/my-projects` returns same as `/projects`

5. **Cross-Organization Security**
   - User in Org A cannot access projects from Org B
   - Even if they know the project_id

---

## Performance Considerations

### Caching Strategy

```go
// Cache user access context for 5 minutes
cacheKey := fmt.Sprintf("user_access:%d", userID)
if cached := cache.Get(cacheKey); cached != nil {
    return cached.(*UserAccessContext)
}

// Fetch from DB and cache
context := fetchUserAccessContext(userID)
cache.Set(cacheKey, context, 5*time.Minute)
```

### Query Optimization
- Use DISTINCT only when needed
- Proper JOIN strategy based on access level
- Limit result sets with pagination
- Use COUNT(*) OVER() for total count in single query

---

## Summary

### Key Benefits
✅ **Versatile**: Single endpoint adapts to user's access level
✅ **Secure**: Automatic filtering based on permissions
✅ **Performant**: Optimized queries per access level
✅ **Maintainable**: Clear access control logic in one place
✅ **Flexible**: Supports filtering and pagination
✅ **Future-proof**: Easy to add new access levels

### Implementation Complexity
- **Low Risk**: Backward compatible changes
- **Medium Effort**: ~2-3 days for full implementation
- **High Value**: Solves access control across entire platform

### Next Steps
1. Review and approve this design
2. Create database indexes
3. Implement helper functions
4. Update GET /projects endpoint
5. Add GET /my-projects endpoint
6. Write comprehensive tests
7. Update API documentation
8. Deploy and monitor