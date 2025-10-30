# User Management

## Overview

The User Management system handles all user-related operations within the infrastructure application, including user creation, authentication integration with AWS Cognito, profile management, location preferences, and user lifecycle management. It provides both SuperAdmin and regular user management capabilities with role-based access control.

## Database Schema

**Table:** `iam.users`

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| `id` | bigint | NO | nextval | Primary key, auto-incrementing user ID |
| `org_id` | bigint | NO | - | Foreign key to iam.organizations table |
| `cognito_id` | varchar(255) | NO | - | AWS Cognito user UUID (sub attribute) |
| `email` | varchar(255) | NO | - | User's email address (must match Cognito) |
| `first_name` | varchar(100) | YES | NULL | User's first name |
| `last_name` | varchar(100) | YES | NULL | User's last name |
| `phone` | varchar(20) | YES | NULL | Optional contact phone number |
| `mobile` | varchar(20) | YES | NULL | Optional mobile phone number |
| `job_title` | varchar(100) | YES | NULL | User's professional title/position |
| `employee_id` | varchar(50) | YES | NULL | Optional employee/staff identifier |
| `avatar_url` | varchar(500) | YES | NULL | URL to user's profile photo |
| `last_selected_location_id` | bigint | YES | NULL | Last location selected by user in UI |
| `is_super_admin` | boolean | NO | false | SuperAdmin privilege flag |
| `status` | varchar(50) | NO | 'pending' | User account status |
| `created_at` | timestamp | NO | CURRENT_TIMESTAMP | Record creation timestamp |
| `created_by` | bigint | NO | - | User ID who created this record |
| `updated_at` | timestamp | NO | CURRENT_TIMESTAMP | Last update timestamp |
| `updated_by` | bigint | NO | - | User ID who last updated this record |
| `is_deleted` | boolean | NO | false | Soft delete flag |

### User Status Values

- **`pending`**: User created but not yet confirmed email (normal users)
- **`pending_org_setup`**: SuperAdmin user who needs to complete organization setup
- **`active`**: Fully active user with complete access
- **`inactive`**: Temporarily disabled user account
- **`suspended`**: User account suspended by administrator

## Data Models

### Core Models

**Location:** `/Users/mayur/git_personal/infrastructure/src/lib/models/user.go`

```go
type User struct {
    UserID                 int64          `json:"user_id"`
    CognitoID              string         `json:"cognito_id"`
    Email                  string         `json:"email"`
    FirstName              sql.NullString `json:"first_name"`
    LastName               sql.NullString `json:"last_name"`
    Phone                  sql.NullString `json:"phone,omitempty"`
    Mobile                 sql.NullString `json:"mobile,omitempty"`
    JobTitle               sql.NullString `json:"job_title,omitempty"`
    EmployeeID             sql.NullString `json:"employee_id,omitempty"`
    AvatarURL              sql.NullString `json:"avatar_url,omitempty"`
    LastSelectedLocationID sql.NullInt64  `json:"last_selected_location_id,omitempty"`
    Status                 string         `json:"status"`
    IsSuperAdmin           bool           `json:"is_super_admin"`
    OrgID                  int64          `json:"org_id"`
    CreatedAt              time.Time      `json:"created_at"`
    UpdatedAt              time.Time      `json:"updated_at"`
}

type UserWithLocationsAndRoles struct {
    User
    LocationRoleAssignments []UserLocationRoleAssignment `json:"location_role_assignments"`
}

type CreateUserRequest struct {
    Email                   string `json:"email" binding:"required,email"`
    FirstName               string `json:"first_name" binding:"required,min=2,max=50"`
    LastName                string `json:"last_name" binding:"required,min=2,max=50"`
    Phone                   string `json:"phone,omitempty"`
    Mobile                  string `json:"mobile,omitempty"`
    JobTitle                string `json:"job_title,omitempty"`
    EmployeeID              string `json:"employee_id,omitempty"`
    AvatarURL               string `json:"avatar_url,omitempty"`
    LastSelectedLocationID  int64  `json:"last_selected_location_id,omitempty"`
}

type UpdateUserRequest struct {
    Email                   string `json:"email,omitempty" binding:"omitempty,email"`
    FirstName               string `json:"first_name,omitempty"`
    LastName                string `json:"last_name,omitempty"`
    Phone                   string `json:"phone,omitempty"`
    Mobile                  string `json:"mobile,omitempty"`
    JobTitle                string `json:"job_title,omitempty"`
    EmployeeID              string `json:"employee_id,omitempty"`
    AvatarURL               string `json:"avatar_url,omitempty"`
    LastSelectedLocationID  int64  `json:"last_selected_location_id,omitempty"`
    Status                  string `json:"status,omitempty" binding:"omitempty,oneof=pending active inactive suspended"`
}
```

## Repository Layer

**Location:** `/Users/mayur/git_personal/infrastructure/src/lib/data/user_management_repository.go`

### Interface

```go
type UserManagementRepository interface {
    CreateNormalUser(ctx context.Context, orgID int64, request *models.CreateUserRequest, createdBy int64) (*models.CreateUserResponse, error)
    GetUsersByOrg(ctx context.Context, orgID int64) ([]models.UserWithLocationsAndRoles, error)
    GetUserByID(ctx context.Context, userID, orgID int64) (*models.UserWithLocationsAndRoles, error)
    GetUserByCognitoID(ctx context.Context, cognitoID string, orgID int64) (*models.UserWithLocationsAndRoles, error)
    UpdateUser(ctx context.Context, userID, orgID int64, user *models.User, updatedBy int64) (*models.User, error)
    DeleteUser(ctx context.Context, userID, orgID int64) error
    SendPasswordResetEmail(ctx context.Context, userEmail string) error
}
```

### Implementation

**DAO:** `UserManagementDao`

**Key Methods:**

- **`CreateNormalUser`**: Creates a new user with AWS Cognito integration
  - Generates temporary password
  - Creates user in Cognito with `custom:isSuperAdmin=false`
  - Creates database record with `status='pending'`
  - Sends welcome email with temporary password automatically
  - Returns user details with temporary password

- **`GetUsersByOrg`**: Retrieves all users for an organization with their location-role assignments

- **`GetUserByID`**: Gets specific user by ID with organization validation

- **`UpdateUser`**: Flexible partial updates supporting any field combination
  - Automatically syncs email changes with AWS Cognito
  - Updates `email_verified=true` in Cognito after email change
  - Supports status-only updates, location-only updates, or full profile updates

- **`DeleteUser`**: Soft deletes user and all associated assignments
  - Removes user-location-role assignments
  - Sets `is_deleted=TRUE` on user record
  - Transactional operation ensures data integrity

- **`SendPasswordResetEmail`**: Triggers AWS Cognito password reset email

## API Endpoints

**Lambda Handler:** `/Users/mayur/git_personal/infrastructure/src/infrastructure-user-management/main.go`

### POST /users
Create a new normal user (non-super admin) with Cognito integration.

**Authorization:** Super Admin only

**Request Body:**
```json
{
  "email": "john.doe@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "phone": "+1234567890",
  "mobile": "+1987654321",
  "job_title": "Software Engineer",
  "employee_id": "EMP001",
  "avatar_url": "https://example.com/photos/john.jpg"
}
```

**Response (201 Created):**
```json
{
  "data": {
    "user_id": 123,
    "cognito_id": "a1b2c3d4-...",
    "email": "john.doe@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "status": "pending",
    "is_super_admin": false,
    "org_id": 1,
    "location_role_assignments": []
  },
  "temporary_password": "TempPass123!",
  "message": "User created successfully. Welcome email with temporary password sent."
}
```

### GET /users
Retrieve all users for the authenticated user's organization.

**Authorization:** Super Admin only

**Response (200 OK):**
```json
{
  "data": {
    "users": [
      {
        "user_id": 123,
        "email": "john.doe@example.com",
        "first_name": "John",
        "last_name": "Doe",
        "status": "active",
        "location_role_assignments": []
      }
    ],
    "total": 1
  }
}
```

### GET /users/{userId}
Get details of a specific user by ID.

**Authorization:** Super Admin only

**Path Parameters:**
- `userId` (required): User ID

**Response (200 OK):**
```json
{
  "data": {
    "user_id": 123,
    "cognito_id": "a1b2c3d4-...",
    "email": "john.doe@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "phone": "+1234567890",
    "mobile": "+1987654321",
    "job_title": "Software Engineer",
    "employee_id": "EMP001",
    "status": "active",
    "is_super_admin": false,
    "org_id": 1,
    "location_role_assignments": []
  }
}
```

### PUT /users/{userId}
Update user profile information with flexible partial updates.

**Authorization:** Super Admin only

**Request Body (all fields optional):**
```json
{
  "email": "updated.email@example.com",
  "first_name": "John",
  "last_name": "Smith",
  "phone": "+1234567890",
  "mobile": "+1987654321",
  "job_title": "Senior Software Engineer",
  "employee_id": "EMP001",
  "avatar_url": "https://example.com/photos/john-updated.jpg",
  "status": "active"
}
```

**Response (200 OK):**
```json
{
  "data": {
    "user_id": 123,
    "email": "updated.email@example.com",
    "first_name": "John",
    "last_name": "Smith",
    "status": "active",
    "updated_at": "2025-10-27T10:30:00Z"
  }
}
```

### DELETE /users/{userId}
Soft delete a user and all associated assignments.

**Authorization:** Super Admin only

**Response (200 OK):**
```json
{
  "data": {
    "message": "User deleted successfully"
  }
}
```

### PATCH /users/{userId}/reset-password
Send password reset email to the user via AWS Cognito.

**Authorization:** Super Admin only

**Response (200 OK):**
```json
{
  "data": {
    "message": "Password reset email sent successfully"
  }
}
```

### PATCH /users/{userId}/location
Update user's selected location (any user can update their own).

**Authorization:** User can update their own location OR Super Admin

**Request Body:**
```json
{
  "location_id": 456
}
```

**Response (200 OK):**
```json
{
  "data": {
    "message": "Location updated successfully",
    "user_id": 123,
    "location_id": 456
  }
}
```

### PUT /users/{userId}/selected-location/{locationId}
Simplified endpoint to update user's selected location preference.

**Authorization:** User can only update their own selected location

**Path Parameters:**
- `userId` (required): User ID
- `locationId` (required): Location ID to set as selected

**Response (200 OK):**
```json
{
  "data": {
    "message": "Selected location updated successfully",
    "user_id": 123,
    "location_id": 456
  }
}
```

## AWS Cognito Integration

### User Creation Flow

1. **Super Admin creates user via POST /users**
2. **Lambda calls AWS Cognito AdminCreateUser:**
   - Username: user's email
   - Temporary password: randomly generated
   - Custom attribute: `custom:isSuperAdmin=false`
   - Email verified: true
   - Welcome email: automatically sent by Cognito
3. **Database record created:**
   - cognito_id: from Cognito response
   - status: 'pending'
   - User receives welcome email with temporary password
4. **User confirms and sets new password**
5. **User can sign in and access system**

### User Signup Process (SuperAdmin)

**Lambda Handler:** `/Users/mayur/git_personal/infrastructure/src/infrastructure-user-signup/main.go`

This is the Cognito PostConfirmation trigger Lambda that runs after email confirmation.

**Flow:**
1. User signs up via Cognito with `custom:isSuperAdmin=true`
2. User verifies email with confirmation code
3. PostConfirmation Lambda is triggered:
   - Creates new organization with status='pending_setup'
   - Creates user record with status='pending_org_setup'
   - User ID stored with organization
4. User signs in and must complete org setup wizard
5. After org setup, status changes to 'active'

### Email Synchronization

When updating user email via PUT /users/{userId}:
1. System updates Cognito email first via `AdminUpdateUserAttributes`
2. Sets `email_verified=true` in Cognito
3. Then updates database email
4. If Cognito update fails, database update is skipped
5. Ensures Cognito and database remain synchronized

## User Status Field

The `status` field tracks the user's account lifecycle:

- **`pending`**: Normal user created, awaiting first login/confirmation
- **`pending_org_setup`**: SuperAdmin user who needs to complete organization setup wizard
- **`active`**: Fully activated user with complete profile and access
- **`inactive`**: Temporarily disabled (user cannot login)
- **`suspended`**: Account suspended by administrator (stronger than inactive)

Status transitions are managed by:
- User signup process (pending_org_setup)
- Organization setup completion (pending_org_setup → active)
- Admin actions (active ↔ inactive ↔ suspended)

## SuperAdmin Flag

The `is_super_admin` boolean field determines user privileges:

- **`true`**: SuperAdmin user with full system access
  - Can manage all users in their organization
  - Can create/update/delete other users
  - Can manage organization settings
  - Can manage all locations
  - Access to all administrative functions

- **`false`**: Normal user with role-based access
  - Access controlled by location-role assignments
  - Can update own profile and selected location
  - Cannot create or manage other users
  - Cannot modify organization settings

SuperAdmin status is:
- Set during signup for the first user (`custom:isSuperAdmin=true` in Cognito)
- Set to `false` for all users created via POST /users
- Stored in both Cognito custom attribute and database
- Included in JWT token for authorization decisions

## Postman Collection

**File:** `/Users/mayur/git_personal/infrastructure/postman/Infrastructure.postman_collection.json`

**User Management Requests:**
- Create User (No Location Required)
- Get All Users
- Get User by ID
- Update User
- Send Password Reset Email
- Delete User
- Set User Selected Location (Simplified)

## Multi-Tenant Isolation

All user operations are scoped to the authenticated user's organization:
- `org_id` filter applied to all queries
- JWT token contains `org_id` claim
- Users cannot access users from other organizations
- Repository methods validate organization membership

## Testing

**Test User:**
- Email: buildboard007+555@gmail.com
- Password: Mayur@1234
- Use ID token (not access token) for API Gateway authentication

**API Endpoint:**
```
https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/users
```

## Related Entities

- **Organizations**: Users belong to one organization (iam.organizations)
- **Locations**: Users can have a last_selected_location_id preference
- **Roles**: Users have location-role assignments (via iam.user_location_access)
- **Cognito**: Users authenticated via AWS Cognito User Pool

## Security Considerations

1. **Authentication**: All endpoints require valid JWT token from AWS Cognito
2. **Authorization**: Most operations restricted to SuperAdmin users
3. **Soft Delete**: Users are never hard-deleted, preserving audit trail
4. **Email Sync**: Email updates synchronized with Cognito automatically
5. **Password Security**: Temporary passwords randomly generated, reset via Cognito
6. **Org Isolation**: Strict enforcement of organization boundaries