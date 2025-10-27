# Project Access Implementation - FINAL PLAN (Using user_assignments)

## 🎯 THE REAL ARCHITECTURE (Correctly Identified)

### **Single Source of Truth: `iam.user_assignments`** ✅

```sql
CREATE TABLE iam.user_assignments (
    id              bigint PRIMARY KEY,
    user_id         bigint NOT NULL,
    role_id         bigint NOT NULL,
    context_type    varchar(20) NOT NULL,  -- 'organization' | 'location' | 'project'
    context_id      bigint NOT NULL,        -- ID of org/location/project
    trade_type      varchar(255),
    is_primary      boolean DEFAULT false,
    start_date      date,
    end_date        date,
    ...
)
```

### **This table REPLACES the need for:**
- ❌ `iam.org_user_roles` - NOT the main table
- ❌ `iam.location_user_roles` - NOT the main table
- ❌ `project.project_user_roles` - NOT the main table

### **Example Data:**

| user_id | role_id | context_type | context_id | Meaning |
|---------|---------|--------------|------------|---------|
| 2 | 7 (Company Admin) | `organization` | 2 | User 2 is org admin for org 2 |
| 9 | 9 (Site Supervisor) | `location` | 5 | User 9 manages location 5 |
| 14 | 8 (Project Manager) | `project` | 6 | User 14 manages project 6 |
| 14 | 8 (Project Manager) | `project` | 15 | User 14 also manages project 15 |

---

## ✅ WHAT ALREADY EXISTS (Assignment Repository)

The `AssignmentDao` already has ALL the methods we need!

### **Key Method - GetUserContexts()** (Line 733)

```go
// GetUserContexts gets all context IDs of a specific type that a user has access to
func (dao *AssignmentDao) GetUserContexts(
    ctx context.Context,
    userID int64,
    contextType string,  // "project" | "location" | "organization"
    orgID int64,
) ([]int64, error)
```

**Query:**
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
```

**This method returns:**
- If `context_type = 'project'` → List of project IDs user has access to
- If `context_type = 'location'` → List of location IDs user has access to
- If `context_type = 'organization'` → List of org IDs user has access to

---

## 🚀 IMPLEMENTATION (SIMPLE!)

### Step 1: Detect User's Access Level

```go
// In project management main.go
func getUserAccessLevel(ctx context.Context, userID, orgID int64, isSuperAdmin bool, assignmentRepo data.AssignmentRepository) (string, []int64, error) {
    // Super admin has full access
    if isSuperAdmin {
        return "super_admin", nil, nil
    }

    // Check org-level assignments
    orgContexts, err := assignmentRepo.GetUserContexts(ctx, userID, "organization", orgID)
    if err != nil {
        return "", nil, err
    }
    if len(orgContexts) > 0 {
        // User has org-level role
        return "organization", orgContexts, nil
    }

    // Check location-level assignments
    locationContexts, err := assignmentRepo.GetUserContexts(ctx, userID, "location", orgID)
    if err != nil {
        return "", nil, err
    }
    if len(locationContexts) > 0 {
        // User has location-level roles
        return "location", locationContexts, nil
    }

    // Check project-level assignments
    projectContexts, err := assignmentRepo.GetUserContexts(ctx, userID, "project", orgID)
    if err != nil {
        return "", nil, err
    }
    if len(projectContexts) > 0 {
        // User has project-level roles
        return "project", projectContexts, nil
    }

    // No assignments - no access
    return "none", nil, nil
}
```

### Step 2: Update GET /projects Endpoint

**File**: `src/infrastructure-project-management/main.go`

```go
// handleGetProjects - UPDATED
func handleGetProjects(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {

    // Determine user's access level
    accessLevel, contextIDs, err := getUserAccessLevel(ctx, claims.UserID, claims.OrgID, claims.IsSuperAdmin, assignmentRepository)
    if err != nil {
        logger.WithError(err).Error("Failed to determine access level")
        return api.ErrorResponse(http.StatusInternalServerError, "Failed to determine access", logger), nil
    }

    // Get filters
    filters := request.QueryStringParameters
    if filters == nil {
        filters = make(map[string]string)
    }

    // Fetch projects based on access level
    var projects []models.Project

    switch accessLevel {
    case "super_admin":
        // Super admin sees everything
        projects, err = projectRepository.GetAllProjects(ctx)

    case "organization":
        // Org admin sees all in their org
        projects, err = projectRepository.GetProjectsByOrg(ctx, claims.OrgID)

    case "location":
        // Location admin sees projects in their locations
        projects, err = projectRepository.GetProjectsByLocations(ctx, contextIDs, claims.OrgID)

    case "project":
        // Project user sees ONLY assigned projects
        projects, err = projectRepository.GetProjectsByIDs(ctx, contextIDs, claims.OrgID)

    default:
        // No access
        projects = []models.Project{}
    }

    if err != nil {
        logger.WithError(err).Error("Failed to get projects")
        return api.ErrorResponse(http.StatusInternalServerError, "Failed to get projects", logger), nil
    }

    // Apply additional filters (location_id, status, etc.)
    // ... filter logic

    response := models.ProjectListResponse{
        Projects:    projects,
        Total:       len(projects),
        AccessLevel: accessLevel, // Tell frontend user's access level
    }

    return api.SuccessResponse(http.StatusOK, response, logger), nil
}
```

### Step 3: Add Repository Methods (if needed)

**File**: `src/lib/data/project_repository.go`

```go
// GetProjectsByLocations gets projects in specific locations
func (dao *ProjectDao) GetProjectsByLocations(ctx context.Context, locationIDs []int64, orgID int64) ([]models.Project, error) {
    query := `
        SELECT * FROM project.projects
        WHERE location_id = ANY($1)
          AND org_id = $2
          AND is_deleted = FALSE
        ORDER BY updated_at DESC
    `

    rows, err := dao.DB.QueryContext(ctx, query, pq.Array(locationIDs), orgID)
    // ... scan and return
}

// GetProjectsByIDs gets specific projects by IDs
func (dao *ProjectDao) GetProjectsByIDs(ctx context.Context, projectIDs []int64, orgID int64) ([]models.Project, error) {
    query := `
        SELECT * FROM project.projects
        WHERE id = ANY($1)
          AND org_id = $2
          AND is_deleted = FALSE
        ORDER BY updated_at DESC
    `

    rows, err := dao.DB.QueryContext(ctx, query, pq.Array(projectIDs), orgID)
    // ... scan and return
}
```

---

## 📊 ACCESS FLOW

### Example: User 14 (buildboard007+BKakadiya@gmail.com)

**Assignments:**
```
user_id=14, context_type='project', context_id=6  (Project Manager)
user_id=14, context_type='project', context_id=15 (Project Manager)
```

**Flow:**
1. User 14 calls `GET /projects`
2. System calls `GetUserContexts(14, 'organization', orgID)` → Returns []  (no org-level)
3. System calls `GetUserContexts(14, 'location', orgID)` → Returns []  (no location-level)
4. System calls `GetUserContexts(14, 'project', orgID)` → Returns [6, 15] ✓
5. Access level = `"project"`, contextIDs = [6, 15]
6. Query: `SELECT * FROM projects WHERE id IN (6, 15)`
7. **User sees ONLY projects 6 and 15** ✓

### Example: User 2 (buildboard007@gmail.com)

**Assignments:**
```
user_id=2, context_type='organization', context_id=2 (Company Admin)
```

**Flow:**
1. User 2 calls `GET /projects`
2. System calls `GetUserContexts(2, 'organization', orgID)` → Returns [2] ✓
3. Access level = `"organization"`, contextIDs = [2]
4. Query: `SELECT * FROM projects WHERE org_id = 2`
5. **User sees ALL projects in org 2** ✓

---

## 🔒 ASSIGNMENT MANAGEMENT API (Already Exists!)

The assignment management API already handles creating assignments:

**Endpoint**: `POST /assignments`

**Request Body:**
```json
{
  "user_id": 26,
  "role_id": 28,
  "context_type": "project",
  "context_id": 47,
  "is_primary": true
}
```

**This creates:**
- User 26 assigned to Project 47 with Role 28 (Architect)
- Automatically makes project 47 visible to user 26 via `GET /projects`

---

## 🎯 KEY BENEFITS

✅ **Already Implemented** - Assignment repository exists with all methods
✅ **Single Source of Truth** - `user_assignments` table
✅ **Active/Inactive Support** - `start_date` and `end_date` handled automatically
✅ **Trade Type Support** - Role can specify trade (electrician, plumber, etc.)
✅ **Primary Assignment** - Track primary project/location per user
✅ **No Breaking Changes** - Just enhance `/projects` endpoint

---

## 📝 IMPLEMENTATION CHECKLIST

### Phase 1: Add Access Detection Helper
- [ ] Create `getUserAccessLevel()` function in project management
- [ ] Initialize `assignmentRepository` in project management `init()`
- [ ] Test access level detection

### Phase 2: Update Project Repository
- [ ] Add `GetProjectsByLocations(locationIDs, orgID)` method
- [ ] Add `GetProjectsByIDs(projectIDs, orgID)` method
- [ ] Test repository methods

### Phase 3: Update GET /projects
- [ ] Integrate access level detection
- [ ] Query projects based on access level
- [ ] Add `access_level` to response
- [ ] Test all scenarios

### Phase 4: Add GET /my-projects (Optional)
- [ ] Create endpoint that ALWAYS uses `context_type='project'`
- [ ] Returns explicit project assignments only

### Phase 5: Testing
- [ ] Test super admin → sees everything
- [ ] Test org admin → sees all org projects
- [ ] Test location admin → sees location projects
- [ ] Test project user → sees assigned projects only
- [ ] Test user with no assignments → sees nothing
- [ ] Test cross-org security

---

## 🔄 INITIALIZATION (Project Management)

**File**: `src/infrastructure-project-management/main.go`

```go
var (
    logger               *logrus.Logger
    projectRepository    data.ProjectRepository
    assignmentRepository data.AssignmentRepository  // ← ADD THIS
    sqlDB                *sql.DB
)

func init() {
    // ... existing init code

    // Initialize assignment repository
    assignmentRepository = data.NewAssignmentRepository(sqlDB)  // ← ADD THIS
}
```

---

## 📊 SAMPLE QUERIES

### Get User's Project Assignments:
```sql
SELECT DISTINCT context_id
FROM iam.user_assignments
WHERE user_id = 14
  AND context_type = 'project'
  AND is_deleted = false
  AND (start_date IS NULL OR start_date <= NOW())
  AND (end_date IS NULL OR end_date >= NOW());
-- Returns: [6, 15]
```

### Get Projects for Project-Level User:
```sql
SELECT * FROM project.projects
WHERE id IN (6, 15)
  AND org_id = 10
  AND is_deleted = false;
```

### Get User's Location Assignments:
```sql
SELECT DISTINCT context_id
FROM iam.user_assignments
WHERE user_id = 9
  AND context_type = 'location'
  AND is_deleted = false;
-- Returns: [5]
```

### Get Projects in User's Locations:
```sql
SELECT * FROM project.projects
WHERE location_id IN (5)
  AND org_id = 10
  AND is_deleted = false;
```

---

## 🎉 CONCLUSION

The architecture is **PERFECT** - it just needs to be connected!

1. ✅ `user_assignments` table exists
2. ✅ `AssignmentDao.GetUserContexts()` method exists
3. ✅ Assignment management API works
4. ❌ Project management API doesn't use it yet

**Solution**: Just call `GetUserContexts()` and filter projects accordingly!

**Estimated Time**: 2-3 hours to implement and test

**Risk**: LOW - No database changes, no breaking changes, just using what exists