# Role Management

## Overview

The Role Management system provides a flexible role-based access control (RBAC) framework for the construction management platform. Roles define what level of access users have within organizations, locations, and projects. The system supports both system-wide standard roles and organization-specific custom roles.

## Database Schema

### Table: `iam.roles`

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| `id` | bigint | NO | nextval() | Primary key, auto-incrementing role ID |
| `org_id` | bigint | YES | NULL | Organization ID (NULL for standard roles, specific org for custom roles) |
| `name` | varchar(100) | NO | - | Role name (max 100 characters) |
| `description` | text | YES | NULL | Optional detailed description of the role |
| `role_type` | varchar(50) | NO | 'custom' | Type: 'standard' (system-wide) or 'custom' (org-specific) |
| `category` | varchar(50) | NO | - | Category: 'management', 'field', 'office', 'external', 'admin' |
| `access_level` | varchar(50) | NO | 'location' | Scope: 'organization', 'location', or 'project' |
| `created_at` | timestamp | NO | CURRENT_TIMESTAMP | Record creation timestamp |
| `created_by` | bigint | NO | - | User ID who created the role |
| `updated_at` | timestamp | NO | CURRENT_TIMESTAMP | Last update timestamp |
| `updated_by` | bigint | NO | - | User ID who last updated the role |
| `is_deleted` | boolean | NO | false | Soft delete flag |

### Related Tables

- **iam.role_permissions**: Many-to-many relationship mapping roles to permissions
- **iam.user_assignments**: Maps users to roles at different scopes (organization, location, project)

## Role Types

### Standard Roles
- Available across all organizations
- `org_id` is NULL
- Created by super admins
- System-wide consistency
- Examples: Super Admin, System Manager, Field Technician

### Custom Roles
- Specific to one organization
- `org_id` references specific organization
- Created by organization admins or super admins
- Tailored to organization needs
- Examples: Company Admin, Project Manager, Site Supervisor, Field Worker

## Role Categories

| Category | Description | Typical Roles |
|----------|-------------|---------------|
| `management` | Leadership and supervisory roles | Project Manager, Site Supervisor, Company Admin |
| `field` | On-site construction workers | Field Worker, Equipment Operator, Safety Inspector |
| `office` | Administrative and office staff | Office Admin, Accountant, Document Controller |
| `external` | Third-party contractors and vendors | Contractor, Consultant, Vendor |
| `admin` | System-level administrators | Super Admin, System Manager |

## Access Levels

| Level | Scope | Description |
|-------|-------|-------------|
| `organization` | Entire organization | Access to all locations and projects within the organization |
| `location` | Specific location(s) | Access to all projects at assigned location(s) |
| `project` | Specific project(s) | Access limited to assigned project(s) only |

## Data Models

### Role Struct
**Location**: `/Users/mayur/git_personal/infrastructure/src/lib/models/role.go`

```go
type Role struct {
    ID                       int64     `json:"id"`
    OrgID                    *int64    `json:"org_id,omitempty"`
    Name                     string    `json:"name"`
    Description              *string   `json:"description,omitempty"`
    RoleType                 string    `json:"role_type"`
    Category                 string    `json:"category"`
    AccessLevel              string    `json:"access_level"`
    CreatedAt                time.Time `json:"created_at"`
    CreatedBy                int64     `json:"created_by"`
    UpdatedAt                time.Time `json:"updated_at"`
    UpdatedBy                int64     `json:"updated_by"`
    IsDeleted                bool      `json:"is_deleted"`
}
```

### RoleRequest Struct (Create/Update)
```go
type RoleRequest struct {
    Name                     string `json:"name" binding:"required,min=2,max=100"`
    Description              string `json:"description,omitempty"`
    RoleType                 string `json:"role_type,omitempty"`          // 'standard' or 'custom'
    Category                 string `json:"category" binding:"required"`   // Category enum
    AccessLevel              string `json:"access_level,omitempty"`       // Level enum
}
```

### RoleWithPermissions Struct
```go
type RoleWithPermissions struct {
    Role
    Permissions []Permission `json:"permissions"`
}
```

### RoleListResponse Struct
```go
type RoleListResponse struct {
    Roles []Role `json:"roles"`
    Total int    `json:"total"`
}
```

## Repository Layer

### Interface: `RoleRepository`
**Location**: `/Users/mayur/git_personal/infrastructure/src/lib/data/role_repository.go`

```go
type RoleRepository interface {
    CreateRole(ctx context.Context, orgID int64, role *models.Role) (*models.Role, error)
    GetRolesByOrg(ctx context.Context, orgID int64) ([]models.Role, error)
    GetRoleByID(ctx context.Context, roleID, orgID int64) (*models.Role, error)
    UpdateRole(ctx context.Context, roleID, orgID int64, role *models.Role) (*models.Role, error)
    DeleteRole(ctx context.Context, roleID, orgID int64) error
    GetRoleWithPermissions(ctx context.Context, roleID, orgID int64) (*models.RoleWithPermissions, error)
}
```

### Key Repository Methods

#### CreateRole
- Inserts new role into `iam.roles` table
- Sets `org_id` to NULL for standard roles, specific org for custom roles
- Returns created role with generated ID and timestamps

#### GetRolesByOrg
- Retrieves all roles for an organization
- Returns both standard roles (org_id = NULL) AND custom roles (org_id = specific org)
- Filters out soft-deleted roles (`is_deleted = FALSE`)
- Orders by name alphabetically

```sql
SELECT id, name, description, org_id, role_type, category, access_level, created_at, updated_at
FROM iam.roles
WHERE (org_id = $1 OR role_type = 'standard') AND is_deleted = FALSE
ORDER BY name ASC
```

#### DeleteRole
- Soft deletes role (sets `is_deleted = TRUE`)
- Also removes all role-permission assignments in transaction
- Validates organization ownership before deletion

## API Endpoints

### Service: `infrastructure-roles-management`
**Location**: `/Users/mayur/git_personal/infrastructure/src/infrastructure-roles-management/main.go`

**Base URL**: `https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/roles`

**Authentication**: JWT Bearer token (ID token, not access token)

**Authorization**: Super admin access required for all operations

### Endpoints

#### 1. Create Role
```
POST /roles
```

**Request Body**:
```json
{
  "name": "Senior Project Manager",
  "description": "Manages large-scale construction projects",
  "role_type": "custom",
  "category": "management",
  "access_level": "project"
}
```

**Response** (201 Created):
```json
{
  "id": 15,
  "org_id": 2,
  "name": "Senior Project Manager",
  "description": "Manages large-scale construction projects",
  "role_type": "custom",
  "category": "management",
  "access_level": "project",
  "created_at": "2025-10-27T10:30:00Z",
  "created_by": 1,
  "updated_at": "2025-10-27T10:30:00Z",
  "updated_by": 1,
  "is_deleted": false
}
```

**Notes**:
- `org_id` is extracted from JWT token, not request body
- Defaults: `role_type` = "custom", `access_level` = "project"
- Standard roles have `org_id` = null

#### 2. List Organization Roles
```
GET /roles
```

**Response** (200 OK):
```json
{
  "roles": [
    {
      "id": 7,
      "org_id": 2,
      "name": "Company Admin",
      "description": "Full access within organization",
      "role_type": "custom",
      "category": "management",
      "access_level": "organization",
      "created_at": "2025-09-13T09:59:17Z",
      "updated_at": "2025-09-13T09:59:17Z",
      "is_deleted": false
    },
    {
      "id": 12,
      "org_id": null,
      "name": "Super Admin",
      "description": "System-wide administrator with full access",
      "role_type": "standard",
      "category": "admin",
      "access_level": "organization",
      "created_at": "2025-09-21T20:00:10Z",
      "updated_at": "2025-09-21T20:01:09Z",
      "is_deleted": false
    }
  ],
  "total": 2
}
```

**Notes**:
- Returns both standard roles AND organization-specific custom roles
- Standard roles available to all organizations

#### 3. Get Role Details
```
GET /roles/{roleId}
```

**Response** (200 OK):
```json
{
  "id": 7,
  "org_id": 2,
  "name": "Company Admin",
  "description": "Full access within organization",
  "role_type": "custom",
  "category": "management",
  "access_level": "organization",
  "created_at": "2025-09-13T09:59:17Z",
  "updated_at": "2025-09-13T09:59:17Z",
  "is_deleted": false,
  "permissions": [
    {
      "permission_id": 1,
      "permission_name": "projects.create",
      "description": "Create new projects",
      "org_id": 2
    }
  ]
}
```

**Notes**:
- Includes associated permissions
- Validates organization ownership

#### 4. Update Role
```
PUT /roles/{roleId}
```

**Request Body**:
```json
{
  "name": "Updated Role Name",
  "description": "Updated role description",
  "category": "management",
  "access_level": "location"
}
```

**Response** (200 OK):
```json
{
  "id": 7,
  "name": "Updated Role Name",
  "description": "Updated role description",
  "role_type": "custom",
  "category": "management",
  "access_level": "location",
  "updated_at": "2025-10-27T11:00:00Z"
}
```

**Notes**:
- Only name and description can be updated
- role_type, category, and access_level are immutable after creation

#### 5. Delete Role
```
DELETE /roles/{roleId}
```

**Response** (204 No Content)

**Notes**:
- Soft delete (sets `is_deleted = TRUE`)
- Removes all role-permission assignments in transaction
- Validates organization ownership

#### 6. Assign Permission to Role
```
POST /roles/{roleId}/permissions
```

**Request Body**:
```json
{
  "permission_id": 5
}
```

**Response** (200 OK):
```json
{
  "message": "Permission assigned successfully"
}
```

#### 7. Remove Permission from Role
```
DELETE /roles/{roleId}/permissions
```

**Request Body**:
```json
{
  "permission_id": 5
}
```

**Response** (200 OK):
```json
{
  "message": "Permission unassigned successfully"
}
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

### Assignment Logic
- Validates both role and permission belong to same organization
- Uses `ON CONFLICT DO NOTHING` to handle duplicate assignments gracefully
- Transaction-based to ensure data consistency

## Security & Authorization

### Super Admin Access Pattern
All role management operations require super admin privileges:

```go
claims, err := auth.ExtractClaimsFromRequest(request)
if err != nil {
    return api.ErrorResponse(http.StatusUnauthorized, "Authentication failed", logger), nil
}

if !claims.IsSuperAdmin {
    return api.ErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can manage roles", logger), nil
}
```

### Organization Isolation
- `org_id` extracted from JWT token, never from request body
- All operations validate organization ownership
- Standard roles (org_id = NULL) accessible to all organizations
- Custom roles (org_id = specific org) restricted to that organization

### JWT Claims Structure
```go
type Claims struct {
    UserID       int64  `json:"user_id"`
    Email        string `json:"email"`
    CognitoID    string `json:"sub"`
    OrgID        int64  `json:"org_id"`
    IsSuperAdmin bool   `json:"isSuperAdmin"`
}
```

## Postman Collection

**File**: `/Users/mayur/git_personal/infrastructure/postman/RolesManagement.postman_collection.json`

### Variables
- `access_token`: JWT ID token from Cognito
- `role_id`: ID of created role (set by tests)
- `existing_role_id`: ID of existing role for testing (default: 7)
- `permission_id`: Permission to assign/remove (default: 1)

### Test User
- Email: `buildboard007+555@gmail.com`
- Password: `Mayur@1234`
- Must have super admin privileges

## Construction Workflow

### 1. Role Creation Workflow
```
Super Admin → Create Standard Role → Available to All Orgs
    |
    └→ Create Custom Role → Available to Specific Org Only
```

### 2. Role Assignment Workflow
```
Role Created → Permissions Assigned → Role Assigned to User → User Gets Access
```

### 3. Access Hierarchy
```
Organization Level (Full Access)
    └→ Location Level (Multiple Projects)
        └→ Project Level (Single Project)
```

## Example Roles

### Standard Roles (System-Wide)
```json
[
  {
    "name": "Super Admin",
    "role_type": "standard",
    "category": "admin",
    "access_level": "organization",
    "org_id": null
  },
  {
    "name": "System Manager",
    "role_type": "standard",
    "category": "management",
    "access_level": "organization",
    "org_id": null
  },
  {
    "name": "Field Technician",
    "role_type": "standard",
    "category": "field",
    "access_level": "project",
    "org_id": null
  }
]
```

### Custom Roles (Organization-Specific)
```json
[
  {
    "name": "Company Admin",
    "role_type": "custom",
    "category": "management",
    "access_level": "organization",
    "org_id": 2
  },
  {
    "name": "Project Manager",
    "role_type": "custom",
    "category": "management",
    "access_level": "project",
    "org_id": 2
  },
  {
    "name": "Site Supervisor",
    "role_type": "custom",
    "category": "management",
    "access_level": "location",
    "org_id": 2
  },
  {
    "name": "Field Worker",
    "role_type": "custom",
    "category": "field",
    "access_level": "project",
    "org_id": 2
  }
]
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
  "error": "Forbidden: Only super admins can manage roles"
}
```

**404 Not Found**:
```json
{
  "error": "Role not found"
}
```

**400 Bad Request**:
```json
{
  "error": "Role name must be between 2 and 100 characters"
}
```

**500 Internal Server Error**:
```json
{
  "error": "Failed to create role"
}
```

## Best Practices

1. **Role Naming**: Use clear, descriptive names that reflect the role's purpose
2. **Standard vs Custom**: Use standard roles for common roles across organizations
3. **Access Levels**: Choose the most restrictive access level appropriate for the role
4. **Soft Deletes**: Never hard delete roles; use soft delete to maintain audit trail
5. **Permission Assignment**: Assign only necessary permissions (principle of least privilege)
6. **Organization Isolation**: Always validate organization ownership in multi-tenant operations

## Related Documentation

- [Permission Management](./permission-management.md)
- User Assignment System (TBD)
- Authentication & JWT (TBD)