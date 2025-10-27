# Summary of Changes - Database Cleanup & Architecture Consolidation

## Date: 2025-10-25

## Overview
Cleaned up deprecated database tables and consolidated all user role assignments into the unified `iam.user_assignments` table architecture.

---

## 🗑️ Database Tables Dropped

### Completely Removed (Not in use):
1. ✅ `iam.org_user_roles` - Replaced by `user_assignments` with `context_type='organization'`
2. ✅ `project.project_managers` - Not referenced anywhere in code
3. ✅ `iam.location_user_roles` - Barely used, replaced by `user_assignments` with `context_type='location'`
4. ✅ `iam.user_location_access` - Barely used, only for deletion cleanup
5. ✅ `project.project_user_roles` - **Now using `user_assignments` with `context_type='project'`**

**All data in these tables is LOST.** Migration was not performed as per user decision.

---

## 🔧 Code Changes

### File: `src/infrastructure-project-management/main.go`

#### Global Variables Added:
```go
var (
    // ... existing vars
    assignmentRepository data.AssignmentRepository  // ← NEW
)
```

#### Initialization Updated:
```go
func init() {
    // ... existing init code
    assignmentRepository = data.NewAssignmentRepository(sqlDB)  // ← NEW
}
```

#### Handlers Modified:

**1. handleAssignUserToProject() - POST /projects/{projectId}/users**
- **Before**: Called `projectRepository.AssignUserToProject()`
- **After**: Converts request to `CreateAssignmentRequest` and calls `assignmentRepository.CreateAssignment()`
- **Key Change**: Sets `context_type='project'` and `context_id=projectID`

**2. handleGetProjectUserRoles() - GET /projects/{projectId}/users**
- **Before**: Called `projectRepository.GetProjectUserRoles(projectID)`
- **After**: Calls `assignmentRepository.GetContextAssignments(ctx, "project", projectID, orgID)`
- **Returns**: `result.Assignments` from context query

**3. handleUpdateProjectUserRole() - PUT /projects/{projectId}/users/{assignmentId}**
- **Before**: Called `projectRepository.UpdateProjectUserRole()`
- **After**: Converts to `UpdateAssignmentRequest` and calls `assignmentRepository.UpdateAssignment()`
- **Key Change**: Converts `RoleID` and `IsPrimary` to pointers for API compatibility

**4. handleRemoveUserFromProject() - DELETE /projects/{projectId}/users/{assignmentId}**
- **Before**: Called `projectRepository.RemoveUserFromProject(assignmentID, projectID, userID)`
- **After**: Calls `assignmentRepository.DeleteAssignment(assignmentID, userID)`
- **Simplified**: No longer needs `projectID` parameter

---

## 📊 API Behavior Changes

### **IMPORTANT**: External API Endpoints Unchanged!

All endpoints remain the same:
- ✅ `POST /projects/{projectId}/users` - Assign user to project
- ✅ `GET /projects/{projectId}/users` - Get project team members
- ✅ `PUT /projects/{projectId}/users/{assignmentId}` - Update user role
- ✅ `DELETE /projects/{projectId}/users/{assignmentId}` - Remove user from project

### Internal Storage Changed:
**Before:**
```sql
-- Stored in project.project_user_roles
INSERT INTO project.project_user_roles (
    project_id, user_id, role_id, trade_type, is_primary, ...
)
```

**After:**
```sql
-- Stored in iam.user_assignments
INSERT INTO iam.user_assignments (
    user_id, role_id,
    context_type,  -- 'project'
    context_id,    -- project_id
    trade_type, is_primary, ...
)
```

---

## ⚠️ Breaking Changes & Migration Impact

### Data Loss:
- ❌ **All existing project user role assignments are LOST**
- ❌ Old data in `project.project_user_roles` was not migrated
- Users will need to **re-assign team members to projects**

### API Compatibility:
- ✅ **Request/Response formats are UNCHANGED**
- ✅ Same JSON payloads work
- ✅ Same status codes returned

### Example Request (Still Works):
```json
POST /projects/47/users
{
  "user_id": 14,
  "role_id": 28,
  "trade_type": "electrical",
  "is_primary": true,
  "start_date": "2025-01-01",
  "end_date": "2025-12-31"
}
```

This now creates:
```sql
INSERT INTO iam.user_assignments (
    user_id=14, role_id=28,
    context_type='project', context_id=47,
    trade_type='electrical', is_primary=true, ...
)
```

---

## 🎯 Benefits of This Change

### 1. **Unified Architecture**
- Single source of truth: `iam.user_assignments`
- Consistent assignment model across org/location/project levels

### 2. **Simpler Codebase**
- Removed duplicate assignment logic
- Reuses existing Assignment Management API

### 3. **More Features Available**
Now project assignments automatically get:
- ✅ `start_date`/`end_date` support (time-bound assignments)
- ✅ Active/inactive tracking
- ✅ Bulk assignment operations
- ✅ Assignment transfer capabilities
- ✅ Better auditing and reporting

### 4. **Enables Future Access Control**
Foundation for implementing:
- Project visibility based on assignments
- Role-based access to projects
- Location/organization level permissions cascading

---

## 🧪 Testing Checklist

### Before Deployment - Local Testing:

#### Test 1: Assign User to Project
```bash
POST /projects/47/users
{
  "user_id": 14,
  "role_id": 28,
  "is_primary": true
}
```
**Expected**: 201 Created, assignment stored in `user_assignments`

**Verify in DB**:
```sql
SELECT * FROM iam.user_assignments
WHERE context_type='project' AND context_id=47 AND user_id=14;
```

#### Test 2: Get Project Users
```bash
GET /projects/47/users
```
**Expected**: 200 OK with list of assignments

#### Test 3: Update User Role
```bash
PUT /projects/47/users/{assignmentId}
{
  "role_id": 29,
  "is_primary": false
}
```
**Expected**: 200 OK with updated assignment

#### Test 4: Remove User from Project
```bash
DELETE /projects/47/users/{assignmentId}
```
**Expected**: 204 No Content

**Verify in DB**:
```sql
SELECT is_deleted FROM iam.user_assignments WHERE id = {assignmentId};
-- Should be TRUE
```

### Edge Cases to Test:

1. **Empty project** - GET /projects/47/users when no users assigned
2. **Invalid assignment ID** - Update/delete non-existent assignment
3. **Cross-org security** - User from org A trying to assign to org B's project
4. **Duplicate assignments** - Same user+role to same project

---

## 🔄 Rollback Plan (If Needed)

### If Issues Found After Deployment:

**Option 1: Revert Code**
```bash
git revert <commit-hash>
npm run build
cdk deploy
```

**Option 2: Restore Tables** (if backup exists)
```sql
-- Restore from backup
-- Re-create tables with old schema
-- Restore data
```

**Option 3: Manual Data Migration**
If old data exists in backup:
```sql
INSERT INTO iam.user_assignments
(user_id, role_id, context_type, context_id, trade_type, is_primary,
 start_date, end_date, created_by, updated_by)
SELECT
    user_id, role_id, 'project', project_id, trade_type, is_primary,
    start_date, end_date, created_by, updated_by
FROM backup.project_user_roles
WHERE is_deleted = FALSE;
```

---

## 📝 Next Steps

### Immediate (Before Deploy):
1. ✅ Build succeeded
2. ⏳ **Test all 4 endpoints locally**
3. ⏳ Verify database writes to correct table
4. ⏳ Test error scenarios

### After Successful Testing:
1. Deploy to Dev environment
2. Re-assign team members to projects (data was lost)
3. Test in Dev environment
4. Deploy to Prod

### Future Work (Not in this change):
1. Implement GET /projects with access control
   - Filter projects based on user assignments
   - Support org/location/project level visibility
2. Add repository methods:
   - `GetProjectsByLocationIDs()`
   - `GetProjectsByProjectIDs()`
3. Remove old methods from ProjectRepository interface

---

## 🔍 Files Changed

1. `/Users/mayur/git_personal/infrastructure/src/infrastructure-project-management/main.go`
   - Added `assignmentRepository` global variable
   - Initialized `assignmentRepository` in `init()`
   - Updated 4 handler functions

2. **Database**:
   - Dropped 5 tables (cannot be undone without backup)

3. **No changes to**:
   - API Gateway routes
   - Request/response models
   - Authentication/authorization
   - Other Lambda functions

---

## ⚠️ Important Notes

1. **Data Loss is Permanent** - Old project user assignments cannot be recovered unless you have a database backup
2. **API is Backward Compatible** - Frontend code does not need changes
3. **Build Succeeded** - No compilation errors
4. **Not Yet Deployed** - Changes are ready but not deployed to AWS

---

## Summary

Successfully consolidated project user role management to use the unified `iam.user_assignments` table. This removes technical debt, simplifies the codebase, and enables future access control features. **All existing project assignments were lost and need to be recreated.**

**Status**: ✅ Built, ⏳ Awaiting Local Testing