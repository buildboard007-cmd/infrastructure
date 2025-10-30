# Organization Management

## Overview

The Organization Management system handles the top-level tenant entity in the multi-tenant infrastructure. Each organization represents a construction company (general contractor, subcontractor, architect, owner, or consultant) with its own isolated data, users, locations, and projects. Organizations are created during SuperAdmin user signup and can be configured through the organization setup wizard.

## Database Schema

**Table:** `iam.organizations`

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| `id` | bigint | NO | nextval | Primary key, auto-incrementing organization ID |
| `name` | varchar(255) | YES | NULL | Organization name (NULL until setup completed) |
| `org_type` | varchar(50) | YES | NULL | Organization type/role in construction industry |
| `license_number` | varchar(100) | YES | NULL | Professional license or contractor license number |
| `address` | text | YES | NULL | Physical address of organization |
| `phone` | varchar(20) | YES | NULL | Primary contact phone number |
| `email` | varchar(255) | YES | NULL | Organization contact email |
| `website` | varchar(255) | YES | NULL | Organization website URL |
| `status` | varchar(50) | NO | 'pending_setup' | Organization account status |
| `created_at` | timestamp | NO | CURRENT_TIMESTAMP | Record creation timestamp |
| `created_by` | bigint | NO | - | User ID who created this organization |
| `updated_at` | timestamp | NO | CURRENT_TIMESTAMP | Last update timestamp |
| `updated_by` | bigint | NO | - | User ID who last updated this record |
| `is_deleted` | boolean | NO | false | Soft delete flag |

### Organization Type Values

- **`general_contractor`**: Primary contractor managing overall construction project
- **`subcontractor`**: Specialized contractor hired by general contractor
- **`architect`**: Design and architectural services firm
- **`owner`**: Property owner or developer
- **`consultant`**: Engineering, inspection, or consulting firm

### Organization Status Values

- **`pending_setup`**: Organization created but not yet configured (during signup)
- **`active`**: Fully configured and operational organization
- **`inactive`**: Temporarily disabled organization
- **`suspended`**: Organization suspended by system administrator

## Data Models

**Location:** `/Users/mayur/git_personal/infrastructure/src/lib/models/organization.go`

```go
type Organization struct {
    ID            int64          `json:"id"`
    Name          string         `json:"name"`
    OrgType       string         `json:"org_type"`
    LicenseNumber sql.NullString `json:"license_number,omitempty"`
    Address       sql.NullString `json:"address,omitempty"`
    Phone         sql.NullString `json:"phone,omitempty"`
    Email         sql.NullString `json:"email,omitempty"`
    Website       sql.NullString `json:"website,omitempty"`
    Status        string         `json:"status"`
    CreatedAt     time.Time      `json:"created_at"`
    CreatedBy     int64          `json:"created_by"`
    UpdatedAt     time.Time      `json:"updated_at"`
    UpdatedBy     int64          `json:"updated_by"`
}

type CreateOrganizationRequest struct {
    Name          string `json:"name" binding:"required,min=3,max=255"`
    OrgType       string `json:"org_type" binding:"required,oneof=general_contractor subcontractor architect owner consultant"`
    LicenseNumber string `json:"license_number,omitempty" binding:"omitempty,max=100"`
    Address       string `json:"address,omitempty"`
    Phone         string `json:"phone,omitempty" binding:"omitempty,max=20"`
    Email         string `json:"email,omitempty" binding:"omitempty,email,max=255"`
    Website       string `json:"website,omitempty" binding:"omitempty,url,max=255"`
    Status        string `json:"status,omitempty" binding:"omitempty,oneof=active inactive pending_setup suspended"`
}

type UpdateOrganizationRequest struct {
    Name          string `json:"name,omitempty" binding:"omitempty,min=3,max=255"`
    OrgType       string `json:"org_type,omitempty" binding:"omitempty,oneof=general_contractor subcontractor architect owner consultant"`
    LicenseNumber string `json:"license_number,omitempty" binding:"omitempty,max=100"`
    Address       string `json:"address,omitempty"`
    Phone         string `json:"phone,omitempty" binding:"omitempty,max=20"`
    Email         string `json:"email,omitempty" binding:"omitempty,email,max=255"`
    Website       string `json:"website,omitempty" binding:"omitempty,url,max=255"`
    Status        string `json:"status,omitempty" binding:"omitempty,oneof=active inactive pending_setup suspended"`
}
```

## Repository Layer

**Location:** `/Users/mayur/git_personal/infrastructure/src/lib/data/org_repository.go`

### Interface

```go
type OrgRepository interface {
    CreateOrganization(ctx context.Context, userID int64, org *models.Organization) (*models.Organization, error)
    UpdateOrganization(ctx context.Context, userID int64, orgID int64, updateReq *models.UpdateOrganizationRequest) (*models.Organization, error)
    GetOrganizationByUserID(ctx context.Context, userID int64) (*models.Organization, error)
    GetOrganizationByID(ctx context.Context, orgID int64) (*models.Organization, error)
    DeleteOrganization(ctx context.Context, orgID int64, userID int64) error
}
```

### Implementation

**DAO:** `OrgDao`

**Key Methods:**

- **`CreateOrganization`**: Creates a new organization record
  - Sets default org_type to 'general_contractor' if not provided
  - Sets default status to 'pending_setup'
  - Records creating user ID in created_by and updated_by

- **`UpdateOrganization`**: Updates organization information with flexible partial updates
  - Builds dynamic SQL query based on provided fields
  - Updates updated_by and updated_at automatically
  - Triggers automatic activation workflow (see below)
  - Returns updated organization record

- **`GetOrganizationByUserID`**: Retrieves organization for a specific user
  - Joins iam.users and iam.organizations tables
  - Ensures user belongs to the returned organization

- **`GetOrganizationByID`**: Gets organization by ID (direct lookup)

- **`DeleteOrganization`**: Soft deletes organization
  - Sets is_deleted=TRUE
  - Updates updated_by and updated_at

### Automatic Activation Workflow

When an organization is updated (typically during org setup wizard), the system automatically checks if the user should be activated:

**Method:** `checkAndUpdateUserStatus`

**Logic:**
1. Check if the user associated with the organization has status='pending_org_setup'
2. If yes, activate both user and organization atomically:
   - Update user status from 'pending_org_setup' to 'active'
   - Update organization status from 'pending_setup' to 'active'
   - Both updates happen in a single database transaction

This ensures that SuperAdmin users who complete the organization setup wizard are immediately granted full access to the system.

## API Endpoints

**Lambda Handler:** `/Users/mayur/git_personal/infrastructure/src/infrastructure-organization-management/main.go`

### GET /org
Retrieve the organization information for the authenticated user.

**Authorization:** Super Admin only

**Response (200 OK):**
```json
{
  "id": 1,
  "name": "Acme Corporation",
  "org_type": "general_contractor",
  "license_number": "GC-12345",
  "address": "123 Business Street, New York, NY 10001",
  "phone": "+1-555-0123",
  "email": "contact@acme.com",
  "website": "https://acme.com",
  "status": "active",
  "created_at": "2025-10-01T10:00:00Z",
  "created_by": 1,
  "updated_at": "2025-10-15T14:30:00Z",
  "updated_by": 1
}
```

### PUT /org
Update the organization information (typically used during org setup wizard).

**Authorization:** Super Admin only

**Request Body (all fields optional):**
```json
{
  "name": "Acme Corporation",
  "org_type": "general_contractor",
  "license_number": "GC-12345",
  "address": "123 Business Street, New York, NY 10001",
  "phone": "+1-555-0123",
  "email": "contact@acme.com",
  "website": "https://acme.com"
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "name": "Acme Corporation",
  "org_type": "general_contractor",
  "license_number": "GC-12345",
  "address": "123 Business Street, New York, NY 10001",
  "phone": "+1-555-0123",
  "email": "contact@acme.com",
  "website": "https://acme.com",
  "status": "active",
  "created_at": "2025-10-01T10:00:00Z",
  "created_by": 1,
  "updated_at": "2025-10-27T10:30:00Z",
  "updated_by": 1
}
```

**Note:** If the SuperAdmin user has status='pending_org_setup', both the user and organization will be automatically activated after this update.

## Organization Creation Flow

Organizations are automatically created during SuperAdmin user signup:

1. **User signs up via Cognito** with `custom:isSuperAdmin=true`
2. **User confirms email** with verification code
3. **PostConfirmation Lambda** is triggered (`infrastructure-user-signup`):
   ```go
   // Create organization with NULL name - will be set during org setup
   INSERT INTO iam.organizations (name, org_type, status, created_by, updated_by)
   VALUES (NULL, NULL, 'pending_setup', 1, 1)
   ```
4. **User record created** with `status='pending_org_setup'` and linked to new organization
5. **User signs in** and is prompted to complete organization setup wizard
6. **User submits org details** via PUT /org endpoint
7. **System automatically activates** both user and organization

## Multi-Tenant Isolation

Organizations provide the top-level isolation boundary for multi-tenant data:

### Data Hierarchy
```
Organization (iam.organizations)
  ├─ Users (iam.users.org_id)
  ├─ Locations (iam.locations.org_id)
  │   └─ Projects (project.projects.location_id)
  │       ├─ RFIs
  │       ├─ Submittals
  │       └─ Issues
  └─ Roles (iam.roles.org_id)
```

### Isolation Enforcement

**JWT Token Claims:**
```json
{
  "user_id": 123,
  "org_id": 1,
  "isSuperAdmin": true,
  "email": "admin@acme.com"
}
```

**Repository Pattern:**
- All repository methods accept `orgID` parameter
- Database queries include `WHERE org_id = $1` clauses
- Users can only access data within their organization
- Organization ID extracted from JWT token by Lambda authorizer

**Example Query:**
```sql
SELECT * FROM iam.users
WHERE org_id = $1 AND is_deleted = FALSE
```

## Postman Collection

**File:** `/Users/mayur/git_personal/infrastructure/postman/Infrastructure.postman_collection.json`

**Organization Management Requests:**
- Get Organization Info
- Update Organization Name

**Example:**
```bash
# Get organization
GET https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/org
Authorization: Bearer {{access_token}}

# Update organization
PUT https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/org
Authorization: Bearer {{access_token}}
Content-Type: application/json

{
  "name": "Acme Corporation",
  "org_type": "general_contractor",
  "license_number": "GC-12345",
  "address": "123 Business Street, New York, NY 10001",
  "phone": "+1-555-0123",
  "email": "contact@acme.com",
  "website": "https://acme.com"
}
```

## Organization Setup Wizard

The typical SuperAdmin onboarding flow:

### Step 1: Signup
- User signs up with email and password
- `custom:isSuperAdmin=true` set in Cognito
- User receives email verification code

### Step 2: Email Confirmation
- User enters verification code
- PostConfirmation Lambda creates:
  - Organization with status='pending_setup'
  - User with status='pending_org_setup'

### Step 3: First Login
- User signs in with credentials
- JWT token includes `isSuperAdmin=true` and `org_id`
- Frontend detects `status='pending_org_setup'`
- Redirects to organization setup wizard

### Step 4: Organization Setup
- User fills out organization details form:
  - Organization name (required)
  - Organization type (required)
  - License number (optional)
  - Address, phone, email, website (optional)
- Frontend calls PUT /org endpoint
- System validates and updates organization

### Step 5: Automatic Activation
- Backend detects user has `status='pending_org_setup'`
- Automatically activates both:
  - User: status → 'active'
  - Organization: status → 'active'
- User granted full system access

### Step 6: Location Creation
- User can now create locations
- User can invite team members
- User can create projects

## Related Entities

- **Users**: All users belong to one organization (iam.users.org_id)
- **Locations**: Organizations have multiple locations (iam.locations.org_id)
- **Roles**: Organizations define custom roles (iam.roles.org_id)
- **Projects**: Projects belong to locations within organizations

## Testing

**Test User:**
- Email: buildboard007+555@gmail.com
- Password: Mayur@1234
- Organization: Test organization
- Use ID token for API Gateway authentication

**API Endpoint:**
```
https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/org
```

## Security Considerations

1. **SuperAdmin Only**: Only SuperAdmin users can view/update organization settings
2. **Single Organization**: Each user belongs to exactly one organization
3. **Soft Delete**: Organizations are never hard-deleted, preserving audit trail
4. **Automatic Activation**: Secure activation flow during initial setup
5. **Audit Trail**: created_by, updated_by, created_at, updated_at tracked for all changes

## Business Rules

1. **One Organization Per User**: Users cannot switch between organizations
2. **Mandatory Setup**: SuperAdmin must complete org setup before full access
3. **Type Immutability**: Organization type should generally not change after setup
4. **Status Transitions**: pending_setup → active (one-way during setup)
5. **Name Requirement**: Organization name is NULL until setup completed