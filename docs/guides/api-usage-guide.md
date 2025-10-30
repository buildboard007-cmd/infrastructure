# API Usage Guide

## Table of Contents

- [Overview](#overview)
- [API Authentication](#api-authentication)
- [Common Request Patterns](#common-request-patterns)
- [Pagination](#pagination)
- [Filtering and Sorting](#filtering-and-sorting)
- [Error Handling](#error-handling)
- [Rate Limits](#rate-limits)
- [Best Practices](#best-practices)
- [Permission Levels](#permission-levels)
- [Super Admin Special Permissions](#super-admin-special-permissions)
- [Organization-Level Permissions](#organization-level-permissions)
- [Project-Level Permissions](#project-level-permissions)
- [API Reference by Service](#api-reference-by-service)

---

## Overview

The BuildBoard API is a RESTful API built on AWS API Gateway with Lambda backends. All endpoints require authentication via AWS Cognito and use JSON for request/response bodies.

### Base URLs

**Development:**
```
https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main
```

**Production:**
```
https://api.buildboard.com
```

### API Characteristics

- **Authentication:** AWS Cognito ID tokens (JWT)
- **Content Type:** `application/json`
- **Authorization:** Role-based with hierarchical access control
- **Response Format:** JSON
- **Error Format:** Standardized error responses
- **Versioning:** Versioned via URL path (future: `/v1`, `/v2`)

---

## API Authentication

### Authentication Flow

1. **User signs in** with email/password to Cognito
2. **Cognito returns** ID token, access token, and refresh token
3. **Client includes** ID token in `Authorization` header
4. **API Gateway validates** token with Cognito
5. **Token Customizer Lambda** enriches token with user profile
6. **Lambda handler** extracts claims and processes request

### Getting ID Token

**Using curl:**

```bash
TOKEN=$(curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{
    "AuthFlow":"USER_PASSWORD_AUTH",
    "ClientId":"3f0fb5mpivctnvj85tucusf88e",
    "AuthParameters":{
      "USERNAME":"buildboard007+555@gmail.com",
      "PASSWORD":"Mayur@1234"
    }
  }' | jq -r '.AuthenticationResult.IdToken')

echo "Token: $TOKEN"
```

**Response structure:**

```json
{
  "AuthenticationResult": {
    "AccessToken": "eyJraWQ...",
    "ExpiresIn": 3600,
    "IdToken": "eyJraWQ...",
    "RefreshToken": "eyJjdHk...",
    "TokenType": "Bearer"
  },
  "ChallengeParameters": {}
}
```

### Using ID Token in Requests

**Standard pattern:**

```bash
curl -X GET "https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/projects?location_id=6" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

**Important:** Use **ID token**, not access token. API Gateway expects ID tokens for authentication.

### Token Claims

**Standard Cognito claims:**

```json
{
  "sub": "12345678-1234-1234-1234-123456789012",
  "email": "user@example.com",
  "email_verified": true,
  "cognito:username": "user@example.com"
}
```

**Custom claims (added by Token Customizer):**

```json
{
  "user_id": 19,
  "org_id": 10,
  "isSuperAdmin": true,
  "first_name": "John",
  "last_name": "Doe",
  "full_name": "John Doe",
  "locations": [6, 7, 22, 24],
  "last_selected_location_id": 6
}
```

### Token Expiration

- **ID Token:** Expires after 1 hour
- **Access Token:** Expires after 1 hour
- **Refresh Token:** Expires after 30 days

**Refresh token:**

```bash
curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{
    "AuthFlow":"REFRESH_TOKEN_AUTH",
    "ClientId":"3f0fb5mpivctnvj85tucusf88e",
    "AuthParameters":{
      "REFRESH_TOKEN":"'$REFRESH_TOKEN'"
    }
  }' | jq -r '.AuthenticationResult.IdToken'
```

---

## Common Request Patterns

### GET Request (List Resources)

**Pattern:**

```bash
GET /resources?param1=value1&param2=value2
Authorization: Bearer {token}
```

**Example - List projects:**

```bash
curl -X GET "$API_BASE/projects?location_id=6&page=1&limit=20" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .
```

**Response:**

```json
{
  "projects": [
    {
      "id": 29,
      "project_number": "PROJ-2025-0001",
      "name": "Main Office Renovation",
      "location_id": 6,
      "org_id": 10,
      "status": "active"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 45,
    "total_pages": 3
  }
}
```

### GET Request (Single Resource)

**Pattern:**

```bash
GET /resources/{id}
Authorization: Bearer {token}
```

**Example - Get project:**

```bash
curl -X GET "$API_BASE/projects/29" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .
```

**Response:**

```json
{
  "project": {
    "id": 29,
    "project_number": "PROJ-2025-0001",
    "name": "Main Office Renovation",
    "description": "Complete renovation of headquarters",
    "location_id": 6,
    "org_id": 10,
    "status": "active",
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-20T14:45:00Z"
  }
}
```

### POST Request (Create Resource)

**Pattern:**

```bash
POST /resources
Authorization: Bearer {token}
Content-Type: application/json

{
  "field1": "value1",
  "field2": "value2"
}
```

**Example - Create project:**

```bash
curl -X POST "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "location_id": 6,
    "basic_info": {
      "name": "New Construction Project",
      "description": "Building a new facility"
    },
    "project_details": {
      "project_stage": "planning",
      "work_scope": "new_construction"
    },
    "timeline": {
      "start_date": "2025-06-01",
      "planned_end_date": "2026-12-31"
    },
    "financial": {
      "budget": 5000000
    }
  }' | jq .
```

**Response:**

```json
{
  "project": {
    "id": 45,
    "project_number": "PROJ-2025-0002",
    "name": "New Construction Project",
    "location_id": 6,
    "org_id": 10,
    "status": "active",
    "created_at": "2025-10-27T12:00:00Z"
  },
  "message": "Project created successfully"
}
```

### PUT Request (Update Resource)

**Pattern:**

```bash
PUT /resources/{id}
Authorization: Bearer {token}
Content-Type: application/json

{
  "field1": "updated_value"
}
```

**Example - Update project:**

```bash
curl -X PUT "$API_BASE/projects/45" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "basic_info": {
      "name": "Updated Project Name",
      "description": "Updated description"
    }
  }' | jq .
```

**Response:**

```json
{
  "project": {
    "id": 45,
    "name": "Updated Project Name",
    "description": "Updated description",
    "updated_at": "2025-10-27T13:00:00Z"
  },
  "message": "Project updated successfully"
}
```

### DELETE Request (Soft Delete)

**Pattern:**

```bash
DELETE /resources/{id}
Authorization: Bearer {token}
```

**Example - Delete project:**

```bash
curl -X DELETE "$API_BASE/projects/45" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .
```

**Response:**

```json
{
  "message": "Project deleted successfully"
}
```

**Note:** All deletes are soft deletes. Resources are marked with `is_deleted = true` but not removed from the database.

---

## Pagination

### Pagination Parameters

**Query parameters:**

- `page` - Page number (default: 1)
- `limit` - Items per page (default: 20, max: 100)

**Example:**

```bash
# First page, 10 items
curl "$API_BASE/projects?location_id=6&page=1&limit=10" \
  -H "Authorization: Bearer $TOKEN"

# Second page, 10 items
curl "$API_BASE/projects?location_id=6&page=2&limit=10" \
  -H "Authorization: Bearer $TOKEN"

# Third page, 50 items
curl "$API_BASE/projects?location_id=6&page=3&limit=50" \
  -H "Authorization: Bearer $TOKEN"
```

### Pagination Response

**Standard pagination object:**

```json
{
  "data": [...],
  "pagination": {
    "page": 2,
    "limit": 20,
    "total": 87,
    "total_pages": 5,
    "has_next": true,
    "has_previous": true
  }
}
```

**Pagination fields:**

- `page` - Current page number
- `limit` - Items per page
- `total` - Total number of items
- `total_pages` - Total number of pages
- `has_next` - Boolean, true if more pages exist
- `has_previous` - Boolean, true if previous pages exist

### Calculating Offsets

**Backend calculation:**

```go
offset := (page - 1) * limit
query := `SELECT * FROM table WHERE ... LIMIT $1 OFFSET $2`
rows, err := db.Query(query, limit, offset)
```

### Iterating Through Pages

**Example script:**

```bash
#!/bin/bash

page=1
has_next=true

while [ "$has_next" = "true" ]; do
    response=$(curl -s "$API_BASE/projects?location_id=6&page=$page&limit=20" \
      -H "Authorization: Bearer $TOKEN")

    # Process results
    echo "$response" | jq '.projects[] | .id, .name'

    # Check for next page
    has_next=$(echo "$response" | jq -r '.pagination.has_next')
    page=$((page + 1))
done
```

---

## Filtering and Sorting

### Filtering

**Common filter patterns:**

```bash
# Filter by location
GET /projects?location_id=6

# Filter by status
GET /issues?status=open

# Filter by date range
GET /rfis?start_date=2025-01-01&end_date=2025-12-31

# Filter by priority
GET /issues?priority=high

# Filter by assigned user
GET /issues?assigned_to=19

# Multiple filters
GET /issues?status=open&priority=high&assigned_to=19
```

**Example:**

```bash
# Get open high-priority issues
curl "$API_BASE/issues?status=open&priority=high" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .

# Get RFIs due this month
curl "$API_BASE/rfis?due_after=2025-10-01&due_before=2025-10-31" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .
```

### Sorting

**Sort parameters:**

- `sort_by` - Field to sort by
- `sort_order` - `asc` or `desc` (default: `asc`)

**Example:**

```bash
# Sort projects by name (ascending)
curl "$API_BASE/projects?location_id=6&sort_by=name&sort_order=asc" \
  -H "Authorization: Bearer $TOKEN"

# Sort issues by priority (descending) then date (ascending)
curl "$API_BASE/issues?sort_by=priority,created_at&sort_order=desc,asc" \
  -H "Authorization: Bearer $TOKEN"
```

### Search

**Search parameter:**

- `search` or `q` - Search term

**Example:**

```bash
# Search projects by name
curl "$API_BASE/projects?location_id=6&search=renovation" \
  -H "Authorization: Bearer $TOKEN"

# Search users by name or email
curl "$API_BASE/users?search=john" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Error Handling

### Standard Error Response

**Format:**

```json
{
  "error": "Error message",
  "code": "ERROR_CODE",
  "status": 400
}
```

### HTTP Status Codes

| Code | Status | Description | When to Use |
|------|--------|-------------|-------------|
| 200 | OK | Success | Successful GET, PUT, DELETE |
| 201 | Created | Resource created | Successful POST |
| 400 | Bad Request | Invalid input | Validation errors, missing fields |
| 401 | Unauthorized | Auth required | Missing or invalid token |
| 403 | Forbidden | No permission | User lacks access to resource |
| 404 | Not Found | Resource not found | Invalid ID, deleted resource |
| 409 | Conflict | Resource conflict | Duplicate entry, constraint violation |
| 500 | Internal Server Error | Server error | Database error, Lambda error |

### Error Examples

**400 Bad Request:**

```json
{
  "error": "Validation failed: name is required",
  "status": 400
}
```

**401 Unauthorized:**

```json
{
  "error": "Unauthorized: Invalid or expired token",
  "status": 401
}
```

**403 Forbidden:**

```json
{
  "error": "Forbidden: You do not have access to this project",
  "status": 403
}
```

**404 Not Found:**

```json
{
  "error": "Project not found",
  "status": 404
}
```

**500 Internal Server Error:**

```json
{
  "error": "Internal server error",
  "status": 500
}
```

### Handling Errors in Client Code

**JavaScript/TypeScript:**

```typescript
async function getProject(projectId: number): Promise<Project> {
  const response = await fetch(`${API_BASE}/projects/${projectId}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    }
  });

  if (!response.ok) {
    const error = await response.json();
    switch (response.status) {
      case 401:
        // Redirect to login
        window.location.href = '/login';
        break;
      case 403:
        // Show access denied message
        alert('You do not have permission to view this project');
        break;
      case 404:
        // Show not found message
        alert('Project not found');
        break;
      default:
        // Show generic error
        alert(`Error: ${error.error}`);
    }
    throw new Error(error.error);
  }

  const data = await response.json();
  return data.project;
}
```

**Bash/curl:**

```bash
response=$(curl -s -w "\n%{http_code}" "$API_BASE/projects/29" \
  -H "Authorization: Bearer $TOKEN")

body=$(echo "$response" | head -n -1)
status=$(echo "$response" | tail -n 1)

if [ "$status" -eq 200 ]; then
    echo "Success: $body"
elif [ "$status" -eq 404 ]; then
    echo "Error: Project not found"
elif [ "$status" -eq 403 ]; then
    echo "Error: Access denied"
else
    echo "Error: HTTP $status - $body"
fi
```

---

## Rate Limits

### Current Rate Limits

**API Gateway Throttling:**
- **Account-level:** 10,000 requests per second
- **Burst capacity:** 5,000 requests
- **Per-user:** No specific limit (uses account-level)

**Best Practices:**
- Implement exponential backoff for retries
- Cache responses when possible
- Use pagination instead of fetching all results
- Batch operations when available

### Rate Limit Headers

**Response headers:**

```
X-RateLimit-Limit: 10000
X-RateLimit-Remaining: 9995
X-RateLimit-Reset: 1635724800
```

### Handling Rate Limits

**429 Too Many Requests response:**

```json
{
  "error": "Rate limit exceeded",
  "status": 429,
  "retry_after": 60
}
```

**Retry with exponential backoff:**

```typescript
async function fetchWithRetry(url: string, options: RequestInit, maxRetries = 3): Promise<Response> {
  let retries = 0;
  while (retries < maxRetries) {
    const response = await fetch(url, options);

    if (response.status === 429) {
      const retryAfter = parseInt(response.headers.get('Retry-After') || '1');
      const backoff = Math.pow(2, retries) * 1000; // Exponential backoff
      await new Promise(resolve => setTimeout(resolve, Math.max(retryAfter * 1000, backoff)));
      retries++;
      continue;
    }

    return response;
  }
  throw new Error('Max retries exceeded');
}
```

---

## Best Practices

### 1. Always Use HTTPS

```bash
# Good
https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/projects

# Bad (will not work)
http://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/projects
```

### 2. Include Content-Type Header

```bash
# For POST/PUT requests
curl -X POST "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Project"}'
```

### 3. Handle Token Expiration

```typescript
// Check token expiration before request
function isTokenExpired(token: string): boolean {
  const payload = JSON.parse(atob(token.split('.')[1]));
  return payload.exp * 1000 < Date.now();
}

// Refresh if expired
if (isTokenExpired(token)) {
  token = await refreshToken();
}
```

### 4. Use Pagination for Large Datasets

```bash
# Don't fetch all at once
curl "$API_BASE/projects?location_id=6&limit=100"

# Do paginate
curl "$API_BASE/projects?location_id=6&page=1&limit=20"
```

### 5. Validate Input Before Sending

```typescript
// Validate required fields
if (!projectData.name || !projectData.location_id) {
  throw new Error('Missing required fields');
}

// Validate field types and ranges
if (typeof projectData.budget !== 'number' || projectData.budget < 0) {
  throw new Error('Invalid budget');
}
```

### 6. Cache Responses When Appropriate

```typescript
// Cache static data (roles, permissions)
const cache = new Map();

async function getRoles(): Promise<Role[]> {
  if (cache.has('roles')) {
    return cache.get('roles');
  }

  const roles = await fetchRoles();
  cache.set('roles', roles);
  return roles;
}
```

### 7. Use Proper HTTP Methods

- **GET** - Retrieve data (idempotent)
- **POST** - Create new resource
- **PUT** - Update entire resource (idempotent)
- **PATCH** - Update partial resource (if supported)
- **DELETE** - Remove resource (idempotent)

### 8. Log API Errors

```typescript
try {
  const response = await fetch(url, options);
  if (!response.ok) {
    console.error('API Error:', {
      url,
      status: response.status,
      error: await response.text()
    });
  }
} catch (error) {
  console.error('Network Error:', error);
}
```

---

## Permission Levels

### Access Control Hierarchy

```
Super Admin (is_super_admin = true)
    └── Full access to all resources in organization

Organization-Level (context_type = 'organization')
    └── Access to all locations and projects in organization

Location-Level (context_type = 'location')
    └── Access to all projects at assigned location(s)

Project-Level (context_type = 'project')
    └── Access only to assigned project(s)
```

### Checking User Permissions

**Via JWT token:**

```bash
# Decode token to see permissions
echo $TOKEN | cut -d. -f2 | base64 -d | jq '{
  user_id,
  org_id,
  isSuperAdmin,
  locations
}'
```

**Via API:**

```bash
# Get current user profile
curl "$API_BASE/users/profile" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '{
    user_id: .id,
    is_super_admin,
    assignments: .assignments
  }'
```

---

## Super Admin Special Permissions

### What Super Admins Can Do

Super admins (`is_super_admin = true`) have **full access** to all resources in their organization:

- ✅ View all projects across all locations
- ✅ Create/update/delete projects at any location
- ✅ Manage all users in the organization
- ✅ Create/update/delete locations
- ✅ Manage roles and permissions
- ✅ Assign users to any project/location
- ✅ Access all issues, RFIs, submittals
- ✅ Manage organization settings

### Super Admin Required Endpoints

The following endpoints **require super admin access**:

#### User Management (`/users`)

| Endpoint | Method | Description | Super Admin Only |
|----------|--------|-------------|------------------|
| `POST /users` | POST | Create new user | ✅ Yes |
| `GET /users` | GET | List all users | ✅ Yes |
| `PUT /users/{userId}` | PUT | Update user | ✅ Yes |
| `DELETE /users/{userId}` | DELETE | Soft delete user | ✅ Yes |
| `PUT /users/{userId}/location` | PUT | Update user location | ⚠️ Users can update own |

**Code location:** `src/infrastructure-user-management/main.go` (lines 54-56)

**Authorization check:**

```go
if request.Resource != "/users/{userId}/location" && !claims.IsSuperAdmin {
    return api.ErrorResponse(http.StatusForbidden,
        "Forbidden: Only super admins can manage users", logger), nil
}
```

#### Organization Management (`/organizations`)

| Endpoint | Method | Description | Super Admin Only |
|----------|--------|-------------|------------------|
| `GET /org` | GET | Get organization details | ✅ Yes |
| `PUT /org` | PUT | Update organization | ✅ Yes |
| `POST /organizations` | POST | Create organization | ✅ Yes (system-level) |

**Code location:** `src/infrastructure-organization-management/main.go` (lines 58-60)

**Authorization check:**

```go
if !claims.IsSuperAdmin {
    return api.ErrorResponse(http.StatusForbidden,
        "Forbidden: Only super admins can manage organization", logger), nil
}
```

#### Location Management (`/locations`)

| Endpoint | Method | Description | Super Admin Only |
|----------|--------|-------------|------------------|
| `POST /locations` | POST | Create location | ✅ Yes |
| `GET /locations` | GET | List locations | ✅ Yes |
| `GET /locations/{locationId}` | GET | Get location | ✅ Yes |
| `PUT /locations/{locationId}` | PUT | Update location | ✅ Yes |
| `DELETE /locations/{locationId}` | DELETE | Soft delete location | ✅ Yes |

**Code location:** `src/infrastructure-location-management/main.go` (lines 49-51)

**Authorization check:**

```go
if !claims.IsSuperAdmin {
    return api.ErrorResponse(http.StatusForbidden,
        "Forbidden: Only super admins can manage locations", logger), nil
}
```

#### Role Management (`/roles`)

| Endpoint | Method | Description | Super Admin Only |
|----------|--------|-------------|------------------|
| `POST /roles` | POST | Create role | ✅ Yes |
| `GET /roles` | GET | List roles | ✅ Yes |
| `PUT /roles/{roleId}` | PUT | Update role | ✅ Yes |
| `DELETE /roles/{roleId}` | DELETE | Soft delete role | ✅ Yes |

**Code location:** `src/infrastructure-roles-management/main.go` (lines 50-52)

#### Permission Management (`/permissions`)

| Endpoint | Method | Description | Super Admin Only |
|----------|--------|-------------|------------------|
| `GET /permissions` | GET | List permissions | ✅ Yes |
| `POST /roles/{roleId}/permissions` | POST | Assign permission to role | ✅ Yes |
| `DELETE /roles/{roleId}/permissions/{permissionId}` | DELETE | Remove permission from role | ✅ Yes |

**Code location:** `src/infrastructure-permissions-management/main.go` (lines 49-51)

---

## Organization-Level Permissions

### Organization-Level Users

Users with **organization-level assignments** (`context_type = 'organization'`) have access to:

- ✅ All locations in the organization
- ✅ All projects across all locations
- ✅ All issues, RFIs, submittals in all projects
- ✅ Can view all users in organization
- ⚠️ Cannot create/delete users (super admin only)
- ⚠️ Cannot manage locations (super admin only)
- ⚠️ Cannot manage roles/permissions (super admin only)

### Creating Organization-Level Assignment

```bash
# Super admin creates org-level assignment
curl -X POST "$API_BASE/assignments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 20,
    "role_id": 1,
    "context_type": "organization",
    "context_id": 10
  }' | jq .
```

### Organization-Level API Usage

**List all projects (no location filter needed):**

```bash
curl "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN_ORG" \
  | jq .
```

**List projects at specific location:**

```bash
curl "$API_BASE/projects?location_id=6" \
  -H "Authorization: Bearer $TOKEN_ORG" \
  | jq .
```

**Access any project:**

```bash
curl "$API_BASE/projects/29" \
  -H "Authorization: Bearer $TOKEN_ORG" \
  | jq .
```

---

## Project-Level Permissions

### Project-Level Users

Users with **project-level assignments** (`context_type = 'project'`) have access to:

- ✅ Only assigned project(s)
- ✅ Issues, RFIs, submittals in assigned project(s)
- ✅ Can create/update issues, RFIs, submittals
- ✅ Can add comments and attachments
- ⚠️ Cannot access other projects
- ⚠️ Cannot see organization or location details
- ❌ Cannot create/delete projects
- ❌ Cannot manage users

### Creating Project-Level Assignment

```bash
# Super admin assigns user to project
curl -X POST "$API_BASE/assignments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 22,
    "role_id": 3,
    "context_type": "project",
    "context_id": 29
  }' | jq .
```

### Project-Level API Usage

**List projects (returns only assigned projects):**

```bash
curl "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN_PROJ" \
  | jq .
# Returns: [ { "id": 29, "name": "Assigned Project" } ]
```

**Access assigned project:**

```bash
curl "$API_BASE/projects/29" \
  -H "Authorization: Bearer $TOKEN_PROJ" \
  | jq .
# Returns: 200 OK with project details
```

**Try to access unassigned project:**

```bash
curl "$API_BASE/projects/30" \
  -H "Authorization: Bearer $TOKEN_PROJ" \
  | jq .
# Returns: 403 Forbidden
```

**Create issue in assigned project:**

```bash
curl -X POST "$API_BASE/projects/29/issues" \
  -H "Authorization: Bearer $TOKEN_PROJ" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Issue from project user",
    "issue_type": "quality_issue",
    "priority": "medium"
  }' | jq .
# Returns: 201 Created
```

---

## API Reference by Service

### User Management

**Base path:** `/users`

| Endpoint | Method | Description | Access Level |
|----------|--------|-------------|--------------|
| `POST /users` | POST | Create new user | Super Admin |
| `GET /users` | GET | List all users | Super Admin |
| `GET /users/profile` | GET | Get current user profile | Any authenticated user |
| `GET /users/{userId}` | GET | Get user by ID | Super Admin |
| `PUT /users/{userId}` | PUT | Update user | Super Admin |
| `DELETE /users/{userId}` | DELETE | Soft delete user | Super Admin |
| `PUT /users/{userId}/location` | PUT | Update user's selected location | User or Super Admin |

### Organization Management

**Base path:** `/org`, `/organizations`

| Endpoint | Method | Description | Access Level |
|----------|--------|-------------|--------------|
| `GET /org` | GET | Get current organization | Super Admin |
| `PUT /org` | PUT | Update organization | Super Admin |

### Location Management

**Base path:** `/locations`

| Endpoint | Method | Description | Access Level |
|----------|--------|-------------|--------------|
| `POST /locations` | POST | Create location | Super Admin |
| `GET /locations` | GET | List locations | Super Admin |
| `GET /locations/{locationId}` | GET | Get location details | Super Admin |
| `PUT /locations/{locationId}` | PUT | Update location | Super Admin |
| `DELETE /locations/{locationId}` | DELETE | Soft delete location | Super Admin |

### Role Management

**Base path:** `/roles`

| Endpoint | Method | Description | Access Level |
|----------|--------|-------------|--------------|
| `POST /roles` | POST | Create role | Super Admin |
| `GET /roles` | GET | List roles | Super Admin |
| `GET /roles/{roleId}` | GET | Get role details | Super Admin |
| `PUT /roles/{roleId}` | PUT | Update role | Super Admin |
| `DELETE /roles/{roleId}` | DELETE | Soft delete role | Super Admin |

### Permission Management

**Base path:** `/permissions`

| Endpoint | Method | Description | Access Level |
|----------|--------|-------------|--------------|
| `GET /permissions` | GET | List all permissions | Super Admin |
| `POST /roles/{roleId}/permissions` | POST | Assign permission to role | Super Admin |
| `GET /roles/{roleId}/permissions` | GET | List role permissions | Super Admin |
| `DELETE /roles/{roleId}/permissions/{permissionId}` | DELETE | Remove permission from role | Super Admin |

### Assignment Management

**Base path:** `/assignments`

| Endpoint | Method | Description | Access Level |
|----------|--------|-------------|--------------|
| `POST /assignments` | POST | Create user assignment | Super Admin, Org Admin |
| `GET /assignments` | GET | List assignments | Super Admin, Org Admin |
| `GET /assignments/{assignmentId}` | GET | Get assignment details | Super Admin, Org Admin |
| `PUT /assignments/{assignmentId}` | PUT | Update assignment | Super Admin, Org Admin |
| `DELETE /assignments/{assignmentId}` | DELETE | Soft delete assignment | Super Admin, Org Admin |
| `GET /assignments/users/{userId}/contexts` | GET | Get user's contexts | Any user (own contexts) |

### Project Management

**Base path:** `/projects`

| Endpoint | Method | Description | Access Level |
|----------|--------|-------------|--------------|
| `POST /projects` | POST | Create project | Super Admin, Org Admin |
| `GET /projects` | GET | List projects (with access control) | Any user (filtered) |
| `GET /projects/{projectId}` | GET | Get project details | Users with access |
| `PUT /projects/{projectId}` | PUT | Update project | Super Admin, Org Admin |
| `DELETE /projects/{projectId}` | DELETE | Soft delete project | Super Admin, Org Admin |

### Issue Management

**Base path:** `/issues`

| Endpoint | Method | Description | Access Level |
|----------|--------|-------------|--------------|
| `POST /projects/{projectId}/issues` | POST | Create issue | Project members |
| `GET /projects/{projectId}/issues` | GET | List issues | Project members |
| `GET /issues/{issueId}` | GET | Get issue details | Project members |
| `PUT /issues/{issueId}` | PUT | Update issue | Project members |
| `DELETE /issues/{issueId}` | DELETE | Soft delete issue | Super Admin, Issue creator |
| `POST /issues/{issueId}/comments` | POST | Add comment | Project members |
| `GET /issues/{issueId}/comments` | GET | List comments | Project members |

### RFI Management

**Base path:** `/rfis`

| Endpoint | Method | Description | Access Level |
|----------|--------|-------------|--------------|
| `POST /projects/{projectId}/rfis` | POST | Create RFI | Project members |
| `GET /projects/{projectId}/rfis` | GET | List RFIs | Project members |
| `GET /rfis/{rfiId}` | GET | Get RFI details | Project members |
| `PUT /rfis/{rfiId}` | PUT | Update RFI | Project members |
| `DELETE /rfis/{rfiId}` | DELETE | Soft delete RFI | Super Admin, RFI creator |

### Submittal Management

**Base path:** `/submittals`

| Endpoint | Method | Description | Access Level |
|----------|--------|-------------|--------------|
| `POST /projects/{projectId}/submittals` | POST | Create submittal | Project members |
| `GET /projects/{projectId}/submittals` | GET | List submittals | Project members |
| `GET /submittals/{submittalId}` | GET | Get submittal details | Project members |
| `PUT /submittals/{submittalId}` | PUT | Update submittal | Project members |
| `DELETE /submittals/{submittalId}` | DELETE | Soft delete submittal | Super Admin, Submittal creator |

### Attachment Management

**Base path:** `/attachments`

| Endpoint | Method | Description | Access Level |
|----------|--------|-------------|--------------|
| `POST /attachments` | POST | Upload attachment | Project members |
| `GET /attachments` | GET | List attachments | Project members |
| `GET /attachments/{attachmentId}` | GET | Get attachment metadata | Project members |
| `GET /attachments/{attachmentId}/download` | GET | Get download URL | Project members |
| `DELETE /attachments/{attachmentId}` | DELETE | Soft delete attachment | Super Admin, Uploader |

---

## Quick Reference

### Common curl Commands

```bash
# Get authentication token
TOKEN=$(curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"AuthFlow":"USER_PASSWORD_AUTH","ClientId":"3f0fb5mpivctnvj85tucusf88e","AuthParameters":{"USERNAME":"buildboard007+555@gmail.com","PASSWORD":"Mayur@1234"}}' \
  | jq -r '.AuthenticationResult.IdToken')

# List projects
curl "$API_BASE/projects?location_id=6" -H "Authorization: Bearer $TOKEN"

# Get single project
curl "$API_BASE/projects/29" -H "Authorization: Bearer $TOKEN"

# Create project
curl -X POST "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"location_id":6,"basic_info":{"name":"Test"}}'

# Update project
curl -X PUT "$API_BASE/projects/29" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"basic_info":{"name":"Updated"}}'

# Delete project
curl -X DELETE "$API_BASE/projects/29" \
  -H "Authorization: Bearer $TOKEN"
```

### Postman Environment Variables

```json
{
  "base_url": "https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main",
  "access_token": "eyJraWQ...",
  "user_id": "19",
  "org_id": "10",
  "location_id": "6",
  "project_id": "29"
}
```

---

**Last Updated:** 2025-10-27