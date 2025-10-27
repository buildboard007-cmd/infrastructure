# Database Architecture Analysis - Complete Investigation

## Executive Summary

After thorough investigation of the database schema and codebase, this is a **PROJECT MANAGEMENT APPLICATION** (like Procore/Bluebeam) with the following key findings:

### ✅ Active Architecture (Currently in Use)
- **`iam.user_assignments`** - **PRIMARY TABLE** for ALL user role assignments (org/location/project level)
- **Unified assignment management via Assignment Management API**
- Modern, flexible context-based assignment system

### ❌ Deprecated Tables (Exist in DB but NOT used in code)
- **`iam.org_user_roles`** - Old table, replaced by `user_assignments`
- **`project.project_managers`** - Old table, NOT referenced anywhere in code

### ⚠️ Transitional Tables (Partially deprecated, but still in use)
- **`project.project_user_roles`** - Still used by Project Management API BUT should be replaced by `user_assignments`
- **`iam.location_user_roles`** - Barely used, only 1 reference in location_repository for COUNT
- **`iam.user_location_access`** - Barely used, only 1 reference for deletion in user_management

---

## Detailed Findings

### 1. IAM Schema Tables

#### ✅ **`iam.user_assignments`** - ACTIVE (PRIMARY TABLE)
**Status**: **Main table for ALL role assignments**

**Used By**:
- `infrastructure-assignment-management` (primary API)
- `infrastructure-user-management` (for user queries)

**Purpose**: Unified table for assigning users to roles at different contexts:
- `context_type = 'organization'` → Organization-level assignments (Company Admin)
- `context_type = 'location'` → Location-level assignments (Site Supervisor)
- `context_type = 'project'` → Project-level assignments (Project Manager, Architect, etc.)

**Key Fields**:
```sql
- user_id: User being assigned
- role_id: Role assigned (links to iam.roles)
- context_type: 'organization' | 'location' | 'project'
- context_id: ID of org/location/project
- trade_type: Optional trade specification (electrician, plumber, etc.)
- is_primary: Primary assignment flag
- start_date/end_date: Optional time bounds for assignment
```

**Code References**:
- `src/lib/data/assignment_repository.go` - Complete CRUD operations
- `src/lib/data/user_repository.go` - User queries with assignments
- `src/infrastructure-assignment-management/main.go` - REST API

---

#### ✅ **`iam.roles`** - ACTIVE
**Status**: Active (defines all roles)

**Purpose**: Defines roles with access levels
**Key Fields**:
```sql
- name: Role name (e.g., "Company Admin", "Project Manager")
- access_level: 'organization' | 'location' | 'project'
- category: 'admin' | 'management' | 'field' | 'office' | 'external'
- role_type: 'system' | 'custom'
- org_id: NULL for system roles, org-specific for custom roles
```

---

#### ✅ **`iam.users`** - ACTIVE
**Status**: Active

**Key Fields**:
- `is_super_admin`: Platform-level admin flag
- `org_id`: Organization membership
- `last_selected_location_id`: User's last selected location

---

#### ✅ **`iam.locations`** - ACTIVE
**Status**: Active

**Purpose**: Organization locations (office, warehouse, job site)

---

#### ✅ **`iam.organizations`** - ACTIVE
**Status**: Active

---

#### ✅ **`iam.permissions`** - ACTIVE
**Status**: Active (for fine-grained permissions)

---

#### ✅ **`iam.role_permissions`** - ACTIVE
**Status**: Active (many-to-many between roles and permissions)

---

#### ❌ **`iam.org_user_roles`** - DEPRECATED
**Status**: **NOT USED IN CODE** (Table exists but no references found)

**Recommendation**: **DROP THIS TABLE** - Replaced by `user_assignments` with `context_type='organization'`

**Evidence**: No grep results for `iam.org_user_roles` in any Go files

---

#### ⚠️ **`iam.location_user_roles`** - BARELY USED
**Status**: Barely used, transitional

**Only Reference Found**:
```go
// src/lib/data/location_repository.go
SELECT COUNT(*) FROM iam.location_user_roles
```

**Recommendation**: **MIGRATE to `user_assignments`** with `context_type='location'`, then DROP

---

#### ⚠️ **`iam.user_location_access`** - BARELY USED
**Status**: Barely used, transitional

**Only Reference Found**:
```go
// src/lib/data/user_management_repository.go
UPDATE iam.user_location_access SET is_deleted = TRUE, updated_by = $1 WHERE user_id = $2
```

**Purpose**: Seems to be for tracking which locations a user has access to (possibly for UI preferences)

**Recommendation**: **Evaluate if still needed**, possibly merge into `user_assignments` or keep for UI state

---

### 2. Project Schema Tables

#### ✅ **`project.projects`** - ACTIVE
**Status**: Active (core table)

---

#### ⚠️ **`project.project_user_roles`** - TRANSITIONAL
**Status**: **Still used BUT should be replaced**

**Used By**:
- `infrastructure-project-management` (AssignUserToProject, GetProjectUserRoles, etc.)

**Current Usage**:
```go
// src/lib/data/project_repository.go
INSERT INTO project.project_user_roles (...)
SELECT ... FROM project.project_user_roles WHERE project_id = $1
UPDATE project.project_user_roles SET ...
```

**Problem**: This duplicates the functionality of `iam.user_assignments` with `context_type='project'`

**Recommendation**:
1. **MIGRATE all project_user_roles data to `user_assignments`**
2. **Update Project Management API to use Assignment API instead**
3. **DROP project_user_roles table after migration**

---

#### ❌ **`project.project_managers`** - DEPRECATED
**Status**: **NOT USED IN CODE** (Table exists but no references found)

**Recommendation**: **DROP THIS TABLE** - Likely replaced by `project_user_roles` which is itself being replaced by `user_assignments`

**Evidence**: No grep results for `project.project_managers` in any Go files

---

#### ✅ **`project.issues`** - ACTIVE
**Status**: Active (core feature)

**Used By**: `infrastructure-issue-management`

---

#### ✅ **`project.issue_attachments`** - ACTIVE
**Status**: Active

---

#### ✅ **`project.issue_comments`** - ACTIVE
**Status**: Active

---

#### ✅ **`project.issue_comment_attachments`** - ACTIVE
**Status**: Active

---

#### ✅ **`project.issue_templates`** - ACTIVE
**Status**: Active

---

#### ✅ **`project.rfis`** - ACTIVE
**Status**: Active (Request for Information)

**Used By**: `infrastructure-rfi-management`

---

#### ✅ **`project.rfi_attachments`** - ACTIVE
**Status**: Active

---

#### ✅ **`project.rfi_comments`** - ACTIVE
**Status**: Active

---

#### ✅ **`project.submittals`** - ACTIVE
**Status**: Active (Submittal management)

**Used By**: `infrastructure-submittal-management`

---

#### ✅ **`project.submittal_attachments`** - ACTIVE
**Status**: Active

---

#### ✅ **`project.submittal_items`** - ACTIVE
**Status**: Active

---

#### ✅ **`project.submittal_reviews`** - ACTIVE
**Status**: Active

---

#### ✅ **`project.project_attachments`** - ACTIVE
**Status**: Active

---

## Architecture Comparison

### OLD Architecture (Partially Deprecated)
```
Organization Level: iam.org_user_roles (DEPRECATED ❌)
Location Level:     iam.location_user_roles (BARELY USED ⚠️)
Project Level:      project.project_user_roles (STILL USED ⚠️)
```

### NEW Architecture (Current/Recommended)
```
ALL LEVELS: iam.user_assignments
  - context_type = 'organization' + context_id = org_id
  - context_type = 'location' + context_id = location_id
  - context_type = 'project' + context_id = project_id
```

---

## Key Architecture Patterns

### Access Levels (from `iam.roles.access_level`)
1. **Organization Level** - Full access to all projects in org
2. **Location Level** - Access to all projects in specific locations
3. **Project Level** - Access to specific projects only

### Super Admin Pattern
- User flag: `iam.users.is_super_admin = true`
- Has access to EVERYTHING across ALL organizations
- Platform-level administration

### Regular User Flow
1. User belongs to organization (`users.org_id`)
2. User gets role assignments via `user_assignments`
3. Assignment specifies context (org/location/project) and context_id
4. Role has access_level and permissions

---

## Critical Discovery: Dual System Problem

### The Problem
**Two parallel systems exist:**

1. **Assignment Management API** (`infrastructure-assignment-management`)
   - Uses `iam.user_assignments` table
   - Modern, unified approach
   - Handles org/location/project assignments

2. **Project Management API** (`infrastructure-project-management`)
   - Uses `project.project_user_roles` table
   - Old approach
   - Only handles project-level assignments
   - **NOT integrated with Assignment Management API**

### Impact on Project Access Control
When fetching projects for a user, the system needs to check:
- ✅ Super admin flag
- ✅ Organization-level assignments (from `user_assignments`)
- ✅ Location-level assignments (from `user_assignments`)
- ❌ **Project-level assignments are SPLIT** between:
  - `user_assignments` (if assigned via Assignment API)
  - `project.project_user_roles` (if assigned via old Project API)

This creates **inconsistent access control** and **data fragmentation**.

---

## Recommended Actions

### Immediate (High Priority)

#### 1. Identify Tables to Drop
**Confirmed Deprecated (not in code)**:
- ❌ `iam.org_user_roles` - DROP
- ❌ `project.project_managers` - DROP

**User Decision Required**:
- ⚠️ `iam.location_user_roles` - Barely used (1 COUNT query)
- ⚠️ `iam.user_location_access` - Used for user deletion cleanup only

#### 2. Migrate Project User Roles
**Goal**: Consolidate ALL assignments into `iam.user_assignments`

**Migration Steps**:
1. Copy all data from `project.project_user_roles` → `user_assignments` with `context_type='project'`
2. Update Project Management API to use Assignment API/Repository
3. Verify no data loss
4. Drop `project.project_user_roles`

#### 3. Fix Project Access Control
After migration, implement proper project filtering in GET /projects:

```go
// Pseudocode
func GetUserAccessibleProjects(userID, orgID) {
    if user.IsSuperAdmin {
        return ALL projects
    }

    // Check org-level assignment
    if HasAssignment(userID, "organization", orgID) {
        return ALL projects in org
    }

    // Check location-level assignments
    locationIDs := GetUserContexts(userID, "location", orgID)
    if len(locationIDs) > 0 {
        return projects WHERE location_id IN (locationIDs)
    }

    // Check project-level assignments (NOW ALL IN user_assignments!)
    projectIDs := GetUserContexts(userID, "project", orgID)
    return projects WHERE id IN (projectIDs)
}
```

---

## Summary

### Active Tables (Keep)
- ✅ `iam.user_assignments` - **PRIMARY TABLE**
- ✅ `iam.roles`, `iam.users`, `iam.organizations`, `iam.locations`
- ✅ `iam.permissions`, `iam.role_permissions`
- ✅ `project.projects`, `project.issues`, `project.rfis`, `project.submittals`
- ✅ All attachment and comment tables

### Deprecated Tables (Drop)
- ❌ `iam.org_user_roles` - Not used
- ❌ `project.project_managers` - Not used

### Transitional Tables (Migrate then Drop)
- ⚠️ `project.project_user_roles` - Migrate to `user_assignments`
- ⚠️ `iam.location_user_roles` - Evaluate/migrate/drop
- ⚠️ `iam.user_location_access` - Evaluate if needed

### Next Steps
1. **User Decision**: Confirm which tables to drop
2. **Migration Plan**: Create migration scripts for transitional tables
3. **API Updates**: Update Project Management API to use `user_assignments`
4. **Testing**: Comprehensive testing of access control
5. **Documentation**: Update API docs with correct architecture