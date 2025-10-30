# Location Management

## Overview

The Location Management system handles physical and virtual locations within an organization. Locations represent offices, warehouses, job sites, and yards where construction activities take place. Each location can have multiple projects and serves as an organizational unit for user access control through location-role assignments. Locations are the key organizational entity between the organization level and project level.

## Database Schema

**Table:** `iam.locations`

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| `id` | bigint | NO | nextval | Primary key, auto-incrementing location ID |
| `org_id` | bigint | NO | - | Foreign key to iam.organizations table |
| `name` | varchar(255) | NO | - | Display name of the location |
| `location_type` | varchar(50) | NO | 'office' | Type of location/facility |
| `address` | text | YES | NULL | Physical street address |
| `city` | varchar(100) | YES | NULL | City name |
| `state` | varchar(50) | YES | NULL | State or province |
| `zip_code` | varchar(20) | YES | NULL | Postal/ZIP code |
| `country` | varchar(100) | YES | 'USA' | Country name |
| `status` | varchar(50) | NO | 'active' | Location operational status |
| `created_at` | timestamp | NO | CURRENT_TIMESTAMP | Record creation timestamp |
| `created_by` | bigint | NO | - | User ID who created this location |
| `updated_at` | timestamp | NO | CURRENT_TIMESTAMP | Last update timestamp |
| `updated_by` | bigint | NO | - | User ID who last updated this record |
| `is_deleted` | boolean | NO | false | Soft delete flag |

### Location Type Values

- **`office`**: Corporate office, branch office, or administrative location
- **`warehouse`**: Storage facility or equipment yard
- **`job_site`**: Active construction site or project location
- **`yard`**: Equipment storage yard or material staging area

### Location Status Values

- **`active`**: Location is operational and can have projects
- **`inactive`**: Location temporarily not in use
- **`under_construction`**: Location is being set up or built
- **`closed`**: Location permanently closed but kept for historical records

## Data Models

**Location:** `/Users/mayur/git_personal/infrastructure/src/lib/models/location.go`

```go
type Location struct {
    ID           int64     `json:"id"`
    OrgID        int64     `json:"org_id"`
    Name         string    `json:"name"`
    LocationType string    `json:"location_type"`
    Address      string    `json:"address,omitempty"`
    City         string    `json:"city,omitempty"`
    State        string    `json:"state,omitempty"`
    ZipCode      string    `json:"zip_code,omitempty"`
    Country      string    `json:"country,omitempty"`
    Status       string    `json:"status"`
    CreatedAt    time.Time `json:"created_at"`
    CreatedBy    int64     `json:"created_by"`
    UpdatedAt    time.Time `json:"updated_at"`
    UpdatedBy    int64     `json:"updated_by"`
}

type CreateLocationRequest struct {
    Name         string `json:"name" binding:"required,min=2,max=255"`
    LocationType string `json:"location_type,omitempty" binding:"omitempty,oneof=office warehouse job_site yard"`
    Address      string `json:"address,omitempty"`
    City         string `json:"city,omitempty" binding:"omitempty,max=100"`
    State        string `json:"state,omitempty" binding:"omitempty,max=50"`
    ZipCode      string `json:"zip_code,omitempty" binding:"omitempty,max=20"`
    Country      string `json:"country,omitempty" binding:"omitempty,max=100"`
    Status       string `json:"status,omitempty" binding:"omitempty,oneof=active inactive under_construction closed"`
}

type UpdateLocationRequest struct {
    Name         string `json:"name,omitempty" binding:"omitempty,min=2,max=255"`
    LocationType string `json:"location_type,omitempty" binding:"omitempty,oneof=office warehouse job_site yard"`
    Address      string `json:"address,omitempty"`
    City         string `json:"city,omitempty" binding:"omitempty,max=100"`
    State        string `json:"state,omitempty" binding:"omitempty,max=50"`
    ZipCode      string `json:"zip_code,omitempty" binding:"omitempty,max=20"`
    Country      string `json:"country,omitempty" binding:"omitempty,max=100"`
    Status       string `json:"status,omitempty" binding:"omitempty,oneof=active inactive under_construction closed"`
}

type LocationListResponse struct {
    Locations []Location `json:"locations"`
    Total     int        `json:"total"`
}
```

## Repository Layer

**Location:** `/Users/mayur/git_personal/infrastructure/src/lib/data/location_repository.go`

### Interface

```go
type LocationRepository interface {
    CreateLocation(ctx context.Context, userID, orgID int64, location *models.Location) (*models.Location, error)
    GetLocationsByOrg(ctx context.Context, orgID int64) ([]models.Location, error)
    GetLocationByID(ctx context.Context, locationID, orgID int64) (*models.Location, error)
    UpdateLocation(ctx context.Context, locationID, orgID int64, updateReq *models.UpdateLocationRequest, userID int64) (*models.Location, error)
    DeleteLocation(ctx context.Context, locationID, orgID int64, userID int64) error
    VerifyLocationAccess(ctx context.Context, userID, locationID int64) (bool, error)
}
```

### Implementation

**DAO:** `LocationDao`

**Key Methods:**

- **`CreateLocation`**: Creates a new location within an organization
  - Sets default location_type to 'office' if not provided
  - Sets default status to 'active'
  - Sets default country to 'USA'
  - Records creating user in created_by and updated_by
  - Transaction-based creation ensures atomicity

- **`GetLocationsByOrg`**: Retrieves all locations for an organization
  - Filters by org_id
  - Excludes soft-deleted locations (is_deleted=false)
  - Orders results by name alphabetically
  - Returns empty array if no locations found

- **`GetLocationByID`**: Gets specific location by ID with org validation
  - Validates location belongs to specified organization
  - Returns error if location not found or belongs to different org

- **`UpdateLocation`**: Updates location with flexible partial updates
  - Builds dynamic SQL query based on provided fields
  - Updates only fields that are provided in request
  - Always updates updated_by and updated_at
  - Validates organization ownership

- **`DeleteLocation`**: Soft deletes a location
  - Sets is_deleted=TRUE (keeps record for audit trail)
  - Updates updated_by and updated_at
  - Location no longer appears in queries but data preserved

- **`VerifyLocationAccess`**: Checks if user has access to location
  - Queries iam.location_user_roles table
  - Returns true if user has any role at the location

## API Endpoints

**Lambda Handler:** `/Users/mayur/git_personal/infrastructure/src/infrastructure-location-management/main.go`

### POST /locations
Create a new location within the organization.

**Authorization:** Super Admin only

**Request Body:**
```json
{
  "name": "Headquarters",
  "location_type": "office",
  "address": "123 Main Street",
  "city": "New York",
  "state": "NY",
  "zip_code": "10001",
  "country": "USA",
  "status": "active"
}
```

**Response (201 Created):**
```json
{
  "id": 1,
  "org_id": 1,
  "name": "Headquarters",
  "location_type": "office",
  "address": "123 Main Street",
  "city": "New York",
  "state": "NY",
  "zip_code": "10001",
  "country": "USA",
  "status": "active",
  "created_at": "2025-10-27T10:00:00Z",
  "created_by": 1,
  "updated_at": "2025-10-27T10:00:00Z",
  "updated_by": 1
}
```

### GET /locations
Retrieve all locations for the authenticated user's organization.

**Authorization:** Super Admin only

**Response (200 OK):**
```json
{
  "locations": [
    {
      "id": 1,
      "org_id": 1,
      "name": "Headquarters",
      "location_type": "office",
      "address": "123 Main Street",
      "city": "New York",
      "state": "NY",
      "zip_code": "10001",
      "country": "USA",
      "status": "active",
      "created_at": "2025-10-27T10:00:00Z",
      "created_by": 1,
      "updated_at": "2025-10-27T10:00:00Z",
      "updated_by": 1
    },
    {
      "id": 2,
      "org_id": 1,
      "name": "Downtown Construction Site",
      "location_type": "job_site",
      "address": "456 Build Avenue",
      "city": "New York",
      "state": "NY",
      "zip_code": "10002",
      "country": "USA",
      "status": "active",
      "created_at": "2025-10-27T11:00:00Z",
      "created_by": 1,
      "updated_at": "2025-10-27T11:00:00Z",
      "updated_by": 1
    }
  ],
  "total": 2
}
```

### GET /locations/{locationId}
Get details of a specific location by ID.

**Authorization:** Super Admin only

**Path Parameters:**
- `locationId` (required): Location ID

**Response (200 OK):**
```json
{
  "id": 1,
  "org_id": 1,
  "name": "Headquarters",
  "location_type": "office",
  "address": "123 Main Street",
  "city": "New York",
  "state": "NY",
  "zip_code": "10001",
  "country": "USA",
  "status": "active",
  "created_at": "2025-10-27T10:00:00Z",
  "created_by": 1,
  "updated_at": "2025-10-27T10:00:00Z",
  "updated_by": 1
}
```

**Error Response (404 Not Found):**
```json
{
  "error": "Location not found"
}
```

### PUT /locations/{locationId}
Update location information with flexible partial updates.

**Authorization:** Super Admin only

**Path Parameters:**
- `locationId` (required): Location ID

**Request Body (all fields optional):**
```json
{
  "name": "Corporate Headquarters",
  "location_type": "office",
  "address": "456 Business Avenue",
  "city": "New York",
  "state": "NY",
  "zip_code": "10001",
  "country": "USA",
  "status": "active"
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "org_id": 1,
  "name": "Corporate Headquarters",
  "location_type": "office",
  "address": "456 Business Avenue",
  "city": "New York",
  "state": "NY",
  "zip_code": "10001",
  "country": "USA",
  "status": "active",
  "created_at": "2025-10-27T10:00:00Z",
  "created_by": 1,
  "updated_at": "2025-10-27T14:30:00Z",
  "updated_by": 1
}
```

### DELETE /locations/{locationId}
Soft delete a location (removes from active queries but preserves data).

**Authorization:** Super Admin only

**Path Parameters:**
- `locationId` (required): Location ID

**Response (204 No Content):**
No response body.

**Error Response (404 Not Found):**
```json
{
  "error": "Location not found"
}
```

## Location-Based Access Control

Locations serve as the organizational unit for user access control:

### User-Location-Role Relationship

Users are assigned to locations with specific roles:

```
User ─┬─ Location A → Role: Project Manager
      ├─ Location B → Role: Field Supervisor
      └─ Location C → Role: Safety Officer
```

**Table:** `iam.location_user_roles`
- Links users to locations with assigned roles
- Determines what projects/data users can access
- Users can have different roles at different locations

### Location Selection

Users can select their active location for the current session:

**Field:** `iam.users.last_selected_location_id`
- Stores user's last selected location preference
- Used by frontend to pre-select location in dropdown
- Updated via PUT /users/{userId}/selected-location/{locationId}
- Included in JWT token for convenient access

## Postman Collection

**File:** `/Users/mayur/git_personal/infrastructure/postman/Infrastructure.postman_collection.json`

**Location Management Requests:**
- Create Location
- Get All Locations
- Get Location by ID
- Update Location
- Delete Location

**Example:**
```bash
# Create location
POST https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/locations
Authorization: Bearer {{access_token}}
Content-Type: application/json

{
  "name": "Headquarters",
  "location_type": "office",
  "address": "123 Main Street",
  "city": "New York",
  "state": "NY",
  "zip_code": "10001",
  "country": "USA",
  "status": "active"
}

# Get all locations
GET https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/locations
Authorization: Bearer {{access_token}}

# Get specific location
GET https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/locations/1
Authorization: Bearer {{access_token}}

# Update location
PUT https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/locations/1
Authorization: Bearer {{access_token}}
Content-Type: application/json

{
  "name": "Corporate Headquarters",
  "address": "456 Business Avenue"
}

# Delete location
DELETE https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/locations/1
Authorization: Bearer {{access_token}}
```

## Multi-Tenant Isolation

All location operations are scoped to the authenticated user's organization:

**JWT Token Claims:**
```json
{
  "user_id": 123,
  "org_id": 1,
  "isSuperAdmin": true,
  "last_selected_location_id": 456
}
```

**Query Pattern:**
```sql
SELECT * FROM iam.locations
WHERE org_id = $1 AND is_deleted = FALSE
ORDER BY name ASC
```

Users from one organization cannot:
- View locations from other organizations
- Create locations in other organizations
- Update or delete locations in other organizations

## Data Hierarchy

Locations fit into the data hierarchy as follows:

```
Organization (iam.organizations)
  └─ Location (iam.locations)
      ├─ Projects (project.projects)
      │   ├─ RFIs (project.rfis)
      │   ├─ Submittals (project.submittals)
      │   └─ Issues (project.issues)
      └─ User Assignments (iam.location_user_roles)
```

## Common Use Cases

### Office Location
Primary administrative location for organization:
```json
{
  "name": "Corporate Headquarters",
  "location_type": "office",
  "address": "123 Business Street",
  "city": "New York",
  "state": "NY",
  "status": "active"
}
```

### Job Site Location
Active construction project site:
```json
{
  "name": "Downtown High-Rise Project",
  "location_type": "job_site",
  "address": "456 Construction Avenue",
  "city": "New York",
  "state": "NY",
  "status": "active"
}
```

### Warehouse Location
Equipment and material storage:
```json
{
  "name": "Equipment Storage Facility",
  "location_type": "warehouse",
  "address": "789 Storage Road",
  "city": "Brooklyn",
  "state": "NY",
  "status": "active"
}
```

### Equipment Yard
Outdoor storage for heavy equipment:
```json
{
  "name": "Heavy Equipment Yard",
  "location_type": "yard",
  "address": "1010 Industrial Parkway",
  "city": "Queens",
  "state": "NY",
  "status": "active"
}
```

## Related Entities

- **Organizations**: Locations belong to one organization (iam.locations.org_id)
- **Users**: Users have location preferences (iam.users.last_selected_location_id)
- **Projects**: Projects are created at locations (project.projects.location_id)
- **User Assignments**: Users assigned to locations with roles (iam.location_user_roles)

## Testing

**Test User:**
- Email: buildboard007+555@gmail.com
- Password: Mayur@1234
- Use ID token for API Gateway authentication

**API Endpoint:**
```
https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/locations
```

## Security Considerations

1. **SuperAdmin Only**: Currently only SuperAdmin users can manage locations
2. **Organization Isolation**: Strict enforcement of org_id boundaries
3. **Soft Delete**: Locations are never hard-deleted to preserve audit trail
4. **Access Verification**: VerifyLocationAccess method validates user access
5. **Audit Trail**: created_by, updated_by, created_at, updated_at tracked

## Business Rules

1. **Unique Names**: Location names should be unique within an organization (not enforced by database constraint but recommended)
2. **Active Locations**: Only active locations should be used for new projects
3. **Soft Delete Only**: Never hard delete locations to maintain referential integrity
4. **Default Values**: office/active/USA used as sensible defaults
5. **Required Fields**: Only name is strictly required; all other fields optional

## Future Enhancements

Potential improvements to location management:

1. **Geolocation**: Add latitude/longitude for map integration
2. **Location Hierarchy**: Support parent-child relationships (region → office → job site)
3. **Time Zones**: Store timezone for accurate scheduling
4. **Operating Hours**: Track location business hours
5. **Contact Information**: Add location-specific contacts
6. **Images**: Support location photos and site plans
7. **Normal User Access**: Allow non-SuperAdmin users to view their assigned locations