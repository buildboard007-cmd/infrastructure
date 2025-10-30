# Permission Management

## Overview

The Permission Management system provides fine-grained access control capabilities for the construction management platform. Permissions define specific actions users can perform on resources (e.g., create projects, view RFIs, manage users). Permissions are assigned to roles, and users inherit permissions through their role assignments.

## Database Schema

### Table: `iam.permissions`

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| `id` | bigint | NO | nextval() | Primary key, auto-incrementing permission ID |
| `code` | varchar(100) | NO | - | Unique permission code (e.g., "projects.create") |
| `name` | varchar(150) | NO | - | Human-readable permission name |
| `description` | text | YES | NULL | Optional detailed description |
| `permission_type` | varchar(50) | NO | 'system' | Type: 'system' (built-in) or 'custom' (org-defined) |
| `module` | varchar(50) | NO | - | Module/feature area (e.g., "projects", "rfis", "users") |
| `resource_type` | varchar(50) | YES | NULL | Resource being accessed (e.g., "project", "rfi", "user") |
| `action_type` | varchar(50) | YES | NULL | Action being performed (e.g., "create", "read", "update", "delete") |
| `created_at` | timestamp | NO | CURRENT_TIMESTAMP | Record creation timestamp |
| `created_by` | bigint | NO | - | User ID who created the permission |
| `updated_at` | timestamp | NO | CURRENT_TIMESTAMP | Last update timestamp |
| `updated_by` | bigint | NO | - | User ID who last updated |
| `is_deleted` | boolean | NO | false | Soft delete flag |

### Related Tables

- **iam.role_permissions**: Many-to-many relationship mapping permissions to roles
- **iam.roles**: Roles that can have permissions assigned

## Permission Structure

### Permission Code Naming Convention
Permissions follow a hierarchical naming pattern:
```
{module}.{action}
{module}.{resource}.{action}
```

**Examples**:
- `projects.create` - Create projects
- `projects.read` - View projects
- `projects.update` - Edit projects
- `projects.delete` - Delete projects
- `rfis.create` - Create RFIs
- `rfis.approve` - Approve RFIs
- `users.manage` - Manage users
- `locations.assign` - Assign locations

### Permission Types

| Type | Description | Example Use Case |
|------|-------------|------------------|
| `system` | Built-in permissions defined by the platform | Core CRUD operations on standard entities |
| `custom` | Organization-specific permissions | Custom workflows, specialized operations |

### Permission Hierarchy

```
Module (e.g., "projects")
    ├── Resource Type (e.g., "project")
    │   ├── create
    │   ├── read
    │   ├── update
    │   ├── delete
    │   └── manage (full control)
    │
    └── Action Type
        ├── Basic CRUD (create, read, update, delete)
        ├── Workflow Actions (submit, approve, reject, close)
        └── Administrative (assign, manage, configure)
```

## Data Models

### Permission Struct
**Location**: `/Users/mayur/git_personal/infrastructure/src/lib/models/permission.go`

```go
type Permission struct {
    PermissionID   int64     `json:"permission_id"`
    PermissionName string    `json:"permission_name"`
    Description    string    `json:"description,omitempty"`
    OrgID          int64     `json:"org_id"`
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
}
```

**Note**: The current model uses simplified field names. The database has more detailed fields (`code`, `name`, `permission_type`, `module`, `resource_type`, `action_type`) that aren't yet reflected in the Go model.

### CreatePermissionRequest Struct
```go
type CreatePermissionRequest struct {
    PermissionName string `json:"permission_name" binding:"required,min=2,max=100"`
    Description    string `json:"description,omitempty"`
}
```

### UpdatePermissionRequest Struct
```go
type UpdatePermissionRequest struct {
    PermissionName string `json:"permission_name,omitempty" binding:"omitempty,min=2,max=100"`
    Description    string `json:"description,omitempty"`
}
```

### PermissionListResponse Struct
```go
type PermissionListResponse struct {
    Permissions []Permission `json:"permissions"`
    Total       int          `json:"total"`
}
```

### Permission Assignment Structs
```go
// For assigning permission to role
type AssignPermissionRequest struct {
    PermissionID int64 `json:"permission_id" binding:"required"`
}

// For removing permission from role
type UnassignPermissionRequest struct {
    PermissionID int64 `json:"permission_id" binding:"required"`
}
```

## Repository Layer

### Interface: `PermissionRepository`
**Location**: `/Users/mayur/git_personal/infrastructure/src/lib/data/permission_repository.go`

```go
type PermissionRepository interface {
    CreatePermission(ctx context.Context, orgID int64, permission *models.Permission) (*models.Permission, error)
    GetPermissionsByOrg(ctx context.Context, orgID int64) ([]models.Permission, error)
    GetPermissionByID(ctx context.Context, permissionID, orgID int64) (*models.Permission, error)
    UpdatePermission(ctx context.Context, permissionID, orgID int64, permission *models.Permission) (*models.Permission, error)
    DeletePermission(ctx context.Context, permissionID, orgID int64) error
}
```

### Key Repository Methods

#### CreatePermission
- Inserts new permission into `iam.permission` table (note: code references old table name)
- Associates permission with organization
- Returns created permission with generated ID and timestamps

```sql
INSERT INTO iam.permission (permission_name, description, org_id)
VALUES ($1, $2, $3)
RETURNING permission_id, created_at, updated_at
```

#### GetPermissionsByOrg
- Retrieves all permissions for a specific organization
- Orders by permission name alphabetically
- Used for listing available permissions when assigning to roles

```sql
SELECT permission_id, permission_name, description, org_id, created_at, updated_at
FROM iam.permission
WHERE org_id = $1
ORDER BY permission_name ASC
```

#### DeletePermission
- Performs transaction-based deletion
- First removes all role-permission assignments
- Then deletes the permission record
- Validates organization ownership

```sql
-- Step 1: Remove assignments
DELETE FROM iam.role_permission WHERE permission_id = $1

-- Step 2: Delete permission
DELETE FROM iam.permission WHERE permission_id = $1 AND org_id = $2
```

## API Endpoints

### Service: `infrastructure-permissions-management`
**Location**: `/Users/mayur/git_personal/infrastructure/src/infrastructure-permissions-management/main.go`

**Base URL**: `https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/permissions`

**Authentication**: JWT Bearer token (ID token, not access token)

**Authorization**: Super admin access required for all operations

### Endpoints

#### 1. Create Permission
```
POST /permissions
```

**Request Body**:
```json
{
  "permission_name": "projects.archive",
  "description": "Archive completed projects"
}
```

**Response** (201 Created):
```json
{
  "permission_id": 25,
  "permission_name": "projects.archive",
  "description": "Archive completed projects",
  "org_id": 2,
  "created_at": "2025-10-27T10:30:00Z",
  "updated_at": "2025-10-27T10:30:00Z"
}
```

**Notes**:
- `org_id` is extracted from JWT token
- Permission name should follow naming convention (module.action)
- Name must be unique within organization

#### 2. List Organization Permissions
```
GET /permissions
```

**Response** (200 OK):
```json
{
  "permissions": [
    {
      "permission_id": 1,
      "permission_name": "projects.create",
      "description": "Create new projects",
      "org_id": 2,
      "created_at": "2025-09-13T09:00:00Z",
      "updated_at": "2025-09-13T09:00:00Z"
    },
    {
      "permission_id": 2,
      "permission_name": "projects.read",
      "description": "View projects",
      "org_id": 2,
      "created_at": "2025-09-13T09:00:00Z",
      "updated_at": "2025-09-13T09:00:00Z"
    }
  ],
  "total": 2
}
```

**Notes**:
- Returns permissions for authenticated user's organization
- Ordered alphabetically by permission name

#### 3. Get Permission Details
```
GET /permissions/{permissionId}
```

**Response** (200 OK):
```json
{
  "permission_id": 1,
  "permission_name": "projects.create",
  "description": "Create new projects",
  "org_id": 2,
  "created_at": "2025-09-13T09:00:00Z",
  "updated_at": "2025-09-13T09:00:00Z"
}
```

**Notes**:
- Validates organization ownership
- Returns 404 if permission not found or doesn't belong to organization

#### 4. Update Permission
```
PUT /permissions/{permissionId}
```

**Request Body**:
```json
{
  "permission_name": "projects.create.advanced",
  "description": "Create advanced project types"
}
```

**Response** (200 OK):
```json
{
  "permission_id": 1,
  "permission_name": "projects.create.advanced",
  "description": "Create advanced project types",
  "org_id": 2,
  "created_at": "2025-09-13T09:00:00Z",
  "updated_at": "2025-10-27T11:00:00Z"
}
```

**Notes**:
- Can update name and/or description
- Must validate organization ownership
- Changing permission name affects existing role assignments

#### 5. Delete Permission
```
DELETE /permissions/{permissionId}
```

**Response** (204 No Content)

**Notes**:
- Hard delete (removes from database)
- Removes all role-permission assignments in transaction
- Validates organization ownership
- Use with caution - affects all roles with this permission

## Permission Check Patterns

### Super Admin Bypass Pattern
Super admins have implicit access to all operations without explicit permission checks:

```go
claims, err := auth.ExtractClaimsFromRequest(request)
if err != nil {
    return api.ErrorResponse(http.StatusUnauthorized, "Authentication failed", logger), nil
}

// Super admins bypass all permission checks
if !claims.IsSuperAdmin {
    return api.ErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can manage permissions", logger), nil
}
```

### JWT Claims Structure
**Location**: `/Users/mayur/git_personal/infrastructure/src/lib/auth/auth.go`

```go
type Claims struct {
    UserID       int64  `json:"user_id"`
    Email        string `json:"email"`
    CognitoID    string `json:"sub"`       // Cognito user ID
    OrgID        int64  `json:"org_id"`     // User's organization
    IsSuperAdmin bool   `json:"isSuperAdmin"` // Bypass flag
}
```

### Permission Check Flow
```
Request → Extract JWT Claims → Check IsSuperAdmin
    |
    ├─→ If Super Admin → Allow (bypass permission check)
    |
    └─→ If Not Super Admin → Check User's Role Permissions
            |
            └─→ Query: Does user's role have required permission?
                    |
                    ├─→ Yes → Allow
                    └─→ No → Deny (403 Forbidden)
```

### Role-Permission Check Query
```sql
-- Check if user has specific permission through their roles
SELECT COUNT(*) > 0 as has_permission
FROM iam.user_assignments ua
JOIN iam.role_permissions rp ON ua.role_id = rp.role_id
JOIN iam.permissions p ON rp.permission_id = p.id
WHERE ua.user_id = $1
  AND p.code = $2
  AND ua.is_deleted = FALSE
  AND rp.is_deleted = FALSE
  AND p.is_deleted = FALSE
```

## Role-Permission Mapping

### Table: `iam.role_permissions`

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| `role_id` | bigint | NO | Foreign key to iam.roles.id |
| `permission_id` | bigint | NO | Foreign key to iam.permissions.id |
| `created_at` | timestamp | NO | Assignment timestamp |
| `created_by` | bigint | NO | User who created assignment |
| `updated_at` | timestamp | NO | Last update timestamp |
| `updated_by` | bigint | NO | User who last updated |
| `is_deleted` | boolean | NO | Soft delete flag |

**Primary Key**: `(role_id, permission_id)`

### RolePermissionRepository Interface
**Location**: `/Users/mayur/git_personal/infrastructure/src/lib/data/role_permission_repository.go`

```go
type RolePermissionRepository interface {
    AssignPermissionToRole(ctx context.Context, roleID, permissionID, orgID int64) error
    UnassignPermissionFromRole(ctx context.Context, roleID, permissionID, orgID int64) error
    IsPermissionAssignedToRole(ctx context.Context, roleID, permissionID int64) (bool, error)
}
```

### AssignPermissionToRole
Validates both role and permission belong to the same organization before assignment:

```go
// Step 1: Validate role exists and belongs to org
SELECT org_id FROM iam.roles
WHERE id = $roleID AND org_id = $orgID AND is_deleted = FALSE

// Step 2: Validate permission exists and belongs to org
SELECT org_id FROM iam.permission
WHERE permission_id = $permissionID AND org_id = $orgID

// Step 3: Create assignment (handles duplicates gracefully)
INSERT INTO iam.role_permission (role_id, permission_id)
VALUES ($roleID, $permissionID)
ON CONFLICT (role_id, permission_id) DO NOTHING
```

### UnassignPermissionFromRole
Removes permission from role with validation:

```go
// Step 1: Validate both exist in same org
SELECT COUNT(*)
FROM iam.roles r
JOIN iam.permission p ON p.org_id = r.org_id
WHERE r.id = $roleID AND p.permission_id = $permissionID
  AND r.org_id = $orgID AND r.is_deleted = FALSE

// Step 2: Remove assignment
DELETE FROM iam.role_permission
WHERE role_id = $roleID AND permission_id = $permissionID
```

## Permission Categories by Module

### Projects Module
```
projects.create          - Create new projects
projects.read            - View project details
projects.update          - Edit project information
projects.delete          - Delete projects
projects.archive         - Archive completed projects
projects.assign          - Assign users to projects
projects.manage          - Full project management
```

### RFIs Module
```
rfis.create              - Create RFIs
rfis.read                - View RFIs
rfis.update              - Edit RFIs
rfis.respond             - Respond to RFIs
rfis.approve             - Approve RFI responses
rfis.close               - Close RFIs
```

### Users Module
```
users.create             - Create new users
users.read               - View user information
users.update             - Edit user details
users.delete             - Delete users
users.assign             - Assign users to roles/locations
users.manage             - Full user management
```

### Locations Module
```
locations.create         - Create locations
locations.read           - View locations
locations.update         - Edit locations
locations.delete         - Delete locations
locations.assign         - Assign users to locations
```

### Submittals Module
```
submittals.create        - Create submittals
submittals.read          - View submittals
submittals.update        - Edit submittals
submittals.review        - Review submittals
submittals.approve       - Approve submittals
submittals.reject        - Reject submittals
```

### Roles & Permissions Module
```
roles.create             - Create roles
roles.read               - View roles
roles.update             - Edit roles
roles.delete             - Delete roles
permissions.manage       - Manage permission assignments
```

## Security & Authorization

### Organization Isolation
- Permissions are organization-scoped
- `org_id` extracted from JWT token, never from request body
- All operations validate organization ownership
- Cross-organization permission access is prevented

### Super Admin Privileges
Super admins have special characteristics:
- **Bypass all permission checks** - Don't need explicit permissions
- Can manage permissions for all organizations
- Can create system-level permissions
- Always have `isSuperAdmin: true` in JWT claims

### Permission Validation Flow
```
1. Extract JWT claims from request
2. Check if user is super admin
   ├─→ Yes: Grant access immediately
   └─→ No: Continue to permission check
3. Query user's roles and associated permissions
4. Check if required permission exists
   ├─→ Yes: Grant access
   └─→ No: Deny with 403 Forbidden
```

### Example Permission Check Implementation
```go
// Pseudo-code for permission checking
func CheckPermission(userID int64, permissionCode string) (bool, error) {
    // Super admins bypass
    if user.IsSuperAdmin {
        return true, nil
    }

    // Query user's permissions through roles
    query := `
        SELECT COUNT(*) > 0
        FROM iam.user_assignments ua
        JOIN iam.role_permissions rp ON ua.role_id = rp.role_id
        JOIN iam.permissions p ON rp.permission_id = p.id
        WHERE ua.user_id = $1
          AND p.code = $2
          AND ua.is_deleted = FALSE
          AND rp.is_deleted = FALSE
    `

    var hasPermission bool
    err := db.QueryRow(query, userID, permissionCode).Scan(&hasPermission)
    return hasPermission, err
}
```

## Construction Workflow

### 1. Permission Setup Workflow
```
Super Admin → Define Permissions → Assign to Roles → Users Inherit via Role
```

### 2. Permission Check Workflow
```
User Action → Check Super Admin → Check Role Permissions → Allow/Deny
```

### 3. Multi-Level Permission Strategy
```
Organization Level Permissions (Broad Access)
    └→ Location Level Permissions (Regional Access)
        └→ Project Level Permissions (Specific Access)
```

## Example Permission Sets

### Project Manager Permission Set
```json
[
  "projects.create",
  "projects.read",
  "projects.update",
  "projects.assign",
  "rfis.create",
  "rfis.read",
  "rfis.respond",
  "submittals.create",
  "submittals.read",
  "submittals.review",
  "users.read"
]
```

### Field Worker Permission Set
```json
[
  "projects.read",
  "rfis.create",
  "rfis.read",
  "submittals.read"
]
```

### Company Admin Permission Set
```json
[
  "projects.*",
  "users.manage",
  "locations.manage",
  "roles.read",
  "permissions.read"
]
```

### Super Admin Permission Set
```
ALL PERMISSIONS (implicit, no explicit assignments needed)
```

## Error Handling

### Common Error Responses

**401 Unauthorized**:
```json
{
  "error": "Authentication failed"
}
```

**403 Forbidden**:
```json
{
  "error": "Forbidden: Only super admins can manage permissions"
}
```

**404 Not Found**:
```json
{
  "error": "Permission not found"
}
```

**400 Bad Request**:
```json
{
  "error": "Permission name must be between 2 and 100 characters"
}
```

**500 Internal Server Error**:
```json
{
  "error": "Failed to create permission"
}
```

## Best Practices

1. **Naming Convention**: Use consistent `module.action` or `module.resource.action` format
2. **Granularity**: Create specific permissions rather than overly broad ones
3. **Principle of Least Privilege**: Grant only necessary permissions
4. **Super Admin Usage**: Reserve super admin for true system administrators
5. **Organization Scoping**: Always validate organization ownership
6. **Permission Documentation**: Maintain clear descriptions for each permission
7. **Audit Trail**: Track who creates and modifies permissions
8. **Testing**: Test permission checks for both positive and negative cases

## Known Issues & Limitations

### Model-Schema Mismatch
The current Go models use simplified field names:
- Model uses: `permission_name`, `org_id` (though org_id not in current model)
- Database has: `code`, `name`, `permission_type`, `module`, `resource_type`, `action_type`

This mismatch suggests:
- The permission system is under active development
- The database schema supports richer permission modeling than currently used
- Future updates may leverage `module`, `resource_type`, `action_type` for hierarchical permissions

### Table Name Inconsistency
- Code references `iam.permission` (singular)
- Actual table is `iam.permissions` (plural)
- Queries may fail if table name is incorrect in deployed code

## Future Enhancements

Based on the database schema, the system appears designed for:
1. **Hierarchical Permissions**: Using `module`, `resource_type`, `action_type` fields
2. **Permission Types**: Distinguishing system vs custom permissions
3. **Wildcard Permissions**: Supporting patterns like `projects.*` for all project actions
4. **Dynamic Permission Checks**: Runtime evaluation based on permission structure

## Related Documentation

- [Role Management](./role-management.md)
- User Assignment System (TBD)
- Authentication & JWT (TBD)
- API Authorization Patterns (TBD)