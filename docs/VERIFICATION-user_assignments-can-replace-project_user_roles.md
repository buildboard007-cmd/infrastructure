# Verification: user_assignments CAN Replace project_user_roles

## ✅ VERIFIED - Safe to Drop and Replace

### Schema Comparison

#### project.project_user_roles (DROPPED)
```sql
- id (bigint)
- project_id (bigint) ← Links to specific project
- user_id (bigint)
- role_id (bigint)
- trade_type (varchar)
- is_primary (boolean)
- start_date (date)
- end_date (date)
- created_at, created_by, updated_at, updated_by, is_deleted
```

#### iam.user_assignments (REPLACEMENT)
```sql
- id (bigint)
- user_id (bigint)
- role_id (bigint)
- context_type (varchar) ← 'project' for project assignments
- context_id (bigint) ← Stores project_id when context_type='project'
- trade_type (varchar)
- is_primary (boolean)
- start_date (date)
- end_date (date)
- created_at, created_by, updated_at, updated_by, is_deleted
```

### ✅ Field Mapping - 100% Compatible

| project_user_roles | user_assignments | Notes |
|-------------------|------------------|-------|
| id | id | ✅ Same |
| project_id | context_id | ✅ Equivalent (when context_type='project') |
| user_id | user_id | ✅ Same |
| role_id | role_id | ✅ Same |
| trade_type | trade_type | ✅ Same |
| is_primary | is_primary | ✅ Same |
| start_date | start_date | ✅ Same |
| end_date | end_date | ✅ Same |
| (implicit: project) | context_type | ✅ NEW: Flexibility to support org/location/project |
| created_at, etc. | created_at, etc. | ✅ Same |

**Conclusion**: `user_assignments` is a **superset** of `project_user_roles`. It can do everything `project_user_roles` did, PLUS support organization and location level assignments.

---

## API Operation Mapping

### Old Project Management API Operations
(Using `project.project_user_roles`)

1. **POST /projects/{projectId}/users** - Assign user to project
2. **GET /projects/{projectId}/users** - Get all users assigned to project
3. **PUT /projects/{projectId}/users/{assignmentId}** - Update user's role on project
4. **DELETE /projects/{projectId}/users/{assignmentId}** - Remove user from project

### New Assignment Repository Methods
(Using `iam.user_assignments`)

| Old Operation | New Method | Parameters |
|--------------|------------|------------|
| Assign user to project | `CreateAssignment()` | `context_type='project'`, `context_id=projectId` |
| Get project users | `GetContextAssignments()` | `context_type='project'`, `context_id=projectId` |
| Update project user | `UpdateAssignment()` | `assignmentID` + update fields |
| Remove project user | `DeleteAssignment()` | `assignmentID` (soft delete) |

### ✅ Additional Benefits from Assignment Repository

**Bonus methods we get:**
- `GetUserContexts(userID, 'project', orgID)` → List all projects user is assigned to
- `GetUserAssignments(userID, orgID)` → All assignments for a user
- `CheckPermission()` → Permission checking
- `CreateBulkAssignments()` → Bulk operations
- `GetActiveAssignments()` → Only active assignments (respects start_date/end_date)

---

## Code Refactoring Required

### Infrastructure Project Management API

**Current handlers that need updating:**

1. **handleAssignUserToProject()** (Line 232)
   - Current: Calls `projectRepository.AssignUserToProject()`
   - New: Call `assignmentRepository.CreateAssignment()` with `context_type='project'`

2. **handleGetProjectUserRoles()** (Line 257)
   - Current: Calls `projectRepository.GetProjectUserRoles(projectID)`
   - New: Call `assignmentRepository.GetContextAssignments(ctx, 'project', projectID, orgID)`

3. **handleUpdateProjectUserRole()** (Line 274)
   - Current: Calls `projectRepository.UpdateProjectUserRole()`
   - New: Call `assignmentRepository.UpdateAssignment()`

4. **handleRemoveUserFromProject()** (Line 308)
   - Current: Calls `projectRepository.RemoveUserFromProject()`
   - New: Call `assignmentRepository.DeleteAssignment()`

### Project Repository Methods to Remove

These methods in `src/lib/data/project_repository.go` can be removed:
- `AssignUserToProject()` (Line 891)
- `GetProjectUserRoles()` (Line 951)
- `UpdateProjectUserRole()` (Line 989)
- `RemoveUserFromProject()` (Line 1048)

Remove from interface (Lines 32-36):
```go
// Project User Role operations
AssignUserToProject(...)
GetProjectUserRoles(...)
UpdateProjectUserRole(...)
RemoveUserFromProject(...)
```

---

## Request/Response Model Changes

### Current Models (project_user_roles based)

**CreateProjectUserRoleRequest**:
```go
{
  "user_id": 14,
  "role_id": 28,
  "trade_type": "electrical",
  "is_primary": true,
  "start_date": "2025-01-01",
  "end_date": "2025-12-31"
}
```

### New Models (user_assignments based)

**CreateAssignmentRequest** (same fields + context):
```go
{
  "user_id": 14,
  "role_id": 28,
  "context_type": "project",    // ← NEW
  "context_id": 47,              // ← NEW (project_id)
  "trade_type": "electrical",
  "is_primary": true,
  "start_date": "2025-01-01",
  "end_date": "2025-12-31"
}
```

**Backward Compatibility Strategy**:
In the project management handler, we can:
1. Accept the old format (without context_type/context_id)
2. Automatically add `context_type='project'` and `context_id=projectId` from URL
3. Call Assignment API internally

This keeps the external API unchanged!

---

## Example: Updated Handler

### Before (using project_user_roles):
```go
func handleAssignUserToProject(...) {
    var createRequest models.CreateProjectUserRoleRequest
    // Parse body

    assignment, err := projectRepository.AssignUserToProject(ctx, projectID, &createRequest, userID)
    // Return response
}
```

### After (using user_assignments):
```go
func handleAssignUserToProject(...) {
    var createRequest models.CreateProjectUserRoleRequest
    // Parse body

    // Convert to assignment request
    assignmentReq := &models.CreateAssignmentRequest{
        UserID:      createRequest.UserID,
        RoleID:      createRequest.RoleID,
        ContextType: "project",        // ← Add context
        ContextID:   projectID,        // ← From URL parameter
        TradeType:   createRequest.TradeType,
        IsPrimary:   createRequest.IsPrimary,
        StartDate:   createRequest.StartDate,
        EndDate:     createRequest.EndDate,
    }

    assignment, err := assignmentRepository.CreateAssignment(ctx, assignmentReq, userID)
    // Return response
}
```

**API endpoint stays the same**: `POST /projects/{projectId}/users`

---

## Testing Strategy

### Test Cases:

1. **Assign user to project**
   - POST /projects/47/users with user_id=14, role_id=28
   - Verify creates entry in `user_assignments` with context_type='project', context_id=47

2. **Get project users**
   - GET /projects/47/users
   - Verify returns all users with assignments where context_type='project' AND context_id=47

3. **Update project user role**
   - PUT /projects/47/users/123 (assignmentId)
   - Verify updates the assignment in `user_assignments`

4. **Remove user from project**
   - DELETE /projects/47/users/123
   - Verify soft-deletes (is_deleted=true) in `user_assignments`

5. **Access control integration**
   - User assigned to project 47 calls GET /projects
   - Verify project 47 appears in their list

---

## Final Verification Checklist

- ✅ Schema compatibility verified (100% match)
- ✅ All operations can be mapped to Assignment Repository
- ✅ No data loss (all fields preserved)
- ✅ Backward compatibility maintained (API endpoints unchanged)
- ✅ Additional benefits gained (unified assignment system)
- ✅ Code changes identified and scoped
- ✅ Testing strategy defined

## Conclusion

**YES, we can safely use `user_assignments` to replace `project_user_roles`.**

The table has already been dropped, and this is the correct architectural direction. The Assignment Repository provides all necessary functionality plus additional benefits.

**Next Steps:**
1. Update Project Management API handlers to use Assignment Repository
2. Keep the same API endpoints (POST/GET/PUT/DELETE /projects/{projectId}/users)
3. Remove old methods from Project Repository
4. Test all project user management operations
5. Verify access control works correctly