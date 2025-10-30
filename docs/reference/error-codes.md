# Error Codes Reference

> Standard HTTP status codes, error messages, and error handling patterns

---

## Standard HTTP Status Codes

### Success Codes (2xx)

| Code | Status | When Used | Response Format |
|------|--------|-----------|-----------------|
| 200 | OK | Successful GET, PUT, PATCH, DELETE | `{"message": "...", "data": {...}}` |
| 201 | Created | Successful POST (resource created) | `{"message": "...", "data": {...}}` |

### Client Error Codes (4xx)

| Code | Status | When Used | Common Scenarios |
|------|--------|-----------|------------------|
| 400 | Bad Request | Invalid request body, missing required fields, validation errors | Missing fields, invalid JSON, bad data types |
| 401 | Unauthorized | Missing or invalid JWT token | No Authorization header, expired token, invalid signature |
| 403 | Forbidden | User lacks permission for resource | Wrong organization, insufficient role, not project member |
| 404 | Not Found | Resource doesn't exist or is soft-deleted | Invalid ID, resource deleted, wrong endpoint |
| 409 | Conflict | Resource already exists or constraint violation | Duplicate unique field, concurrent update |

### Server Error Codes (5xx)

| Code | Status | When Used | Common Scenarios |
|------|--------|-----------|------------------|
| 500 | Internal Server Error | Unexpected server errors | Database error, AWS service error, code bugs |

---

## Error Response Format

### Standard Error Response

```json
{
  "message": "Human-readable error message",
  "error": "Optional detailed error information"
}
```

### Examples

#### 400 Bad Request
```json
{
  "message": "Missing required field: title",
  "error": "Validation failed"
}
```

#### 403 Forbidden
```json
{
  "message": "Access denied: You do not have permission to access this project"
}
```

#### 404 Not Found
```json
{
  "message": "Project not found or has been deleted"
}
```

#### 500 Internal Server Error
```json
{
  "message": "Internal server error occurred while processing request",
  "error": "Database connection failed"
}
```

---

## Common Error Messages

### Authentication Errors (401)

| Message | Cause | Solution |
|---------|-------|----------|
| `Unauthorized: claims not found in authorizer context` | Missing or invalid JWT token | Include valid ID Token in Authorization header |
| `Unauthorized: token has expired` | Token expired (>1 hour old) | Refresh token using refresh token flow |
| `Unauthorized: invalid token signature` | Token from different environment or tampered | Use correct User Pool token, don't modify token |
| `Unauthorized: user_id not found in claims` | Old token without custom claims | Log in again to get new token with custom claims |

### Authorization Errors (403)

| Message | Cause | Solution |
|---------|-------|----------|
| `Access denied: You do not have permission to access this project` | User not assigned to project | Add user to project team via assignments |
| `Access denied: Organization mismatch` | Resource belongs to different organization | Use correct organization context |
| `Access denied: Super Admin access required` | Endpoint requires Super Admin | Use Super Admin account |
| `User account is not active` | User status is pending/inactive/suspended | Activate user account |
| `Access denied: insufficient permissions` | User role lacks required permission | Assign appropriate role or permission |

### Validation Errors (400)

| Message | Cause | Solution |
|---------|-------|----------|
| `Missing required field: {field_name}` | Required field not provided | Include all required fields in request body |
| `Invalid value for field: {field_name}` | Field value doesn't meet constraints | Check field constraints (length, format, enum values) |
| `Invalid JSON format` | Malformed JSON in request body | Validate JSON syntax |
| `Invalid ID format: must be a positive integer` | Non-numeric or negative ID | Use valid integer IDs |
| `Invalid status transition: {from} to {to}` | Illegal workflow state change | Follow allowed status transitions |
| `Invalid date format: use YYYY-MM-DD` | Wrong date format | Use ISO 8601 date format |
| `Invalid email format` | Malformed email address | Use valid email format |

### Resource Not Found Errors (404)

| Message | Cause | Solution |
|---------|-------|----------|
| `Project not found or has been deleted` | Invalid project ID or soft-deleted | Verify project ID, check if deleted |
| `Issue not found` | Invalid issue ID | Verify issue ID |
| `User not found` | Invalid user ID | Verify user ID |
| `Organization not found` | Invalid organization ID | Verify organization ID |
| `Assignment not found` | Invalid assignment ID | Verify assignment ID |
| `Endpoint not found` | Wrong URL or HTTP method | Check API documentation for correct endpoint |

### Conflict Errors (409)

| Message | Cause | Solution |
|---------|-------|----------|
| `User with this email already exists` | Duplicate email in organization | Use different email or update existing user |
| `Role with this name already exists in organization` | Duplicate role name | Use different role name |
| `Assignment already exists for this user and context` | Duplicate assignment | Update existing assignment instead |

### Server Errors (500)

| Message | Cause | Solution |
|---------|-------|----------|
| `Internal server error occurred while processing request` | Unexpected error | Check CloudWatch logs, contact support |
| `Database error: connection timeout` | Database connectivity issue | Retry request, check database status |
| `Error uploading file to S3` | S3 service error | Retry upload, check S3 permissions |
| `Error sending notification` | Email/SNS service error | Check notification service status |

---

## Error Handling Patterns in Go

### Creating Error Responses

```go
// api/api.go utility functions
package api

import (
    "encoding/json"
    "github.com/aws/aws-lambda-go/events"
)

// ErrorResponse creates a standard error response
func ErrorResponse(statusCode int, message string) (events.APIGatewayProxyResponse, error) {
    body := map[string]interface{}{
        "message": message,
    }

    jsonBody, _ := json.Marshal(body)

    return events.APIGatewayProxyResponse{
        StatusCode: statusCode,
        Body:       string(jsonBody),
        Headers: map[string]string{
            "Content-Type": "application/json",
            "Access-Control-Allow-Origin": "*",
        },
    }, nil
}

// ErrorResponseWithDetails creates error response with additional details
func ErrorResponseWithDetails(statusCode int, message string, details string) (events.APIGatewayProxyResponse, error) {
    body := map[string]interface{}{
        "message": message,
        "error":   details,
    }

    jsonBody, _ := json.Marshal(body)

    return events.APIGatewayProxyResponse{
        StatusCode: statusCode,
        Body:       string(jsonBody),
        Headers: map[string]string{
            "Content-Type": "application/json",
            "Access-Control-Allow-Origin": "*",
        },
    }, nil
}
```

### Common Error Response Patterns

```go
// 400 Bad Request - Missing required field
if request.Title == "" {
    return api.ErrorResponse(400, "Missing required field: title")
}

// 400 Bad Request - Invalid value
if request.Priority != "critical" && request.Priority != "high" &&
   request.Priority != "medium" && request.Priority != "low" {
    return api.ErrorResponse(400, "Invalid value for field: priority. Must be one of: critical, high, medium, low")
}

// 401 Unauthorized - Claims extraction failed
claims, err := auth.ExtractClaimsFromRequest(request)
if err != nil {
    return api.ErrorResponse(401, "Unauthorized: "+err.Error())
}

// 403 Forbidden - Organization mismatch
if claims.OrgID != project.OrgID && !claims.IsSuperAdmin {
    return api.ErrorResponse(403, "Access denied: Organization mismatch")
}

// 403 Forbidden - Not project member
if !hasAccess {
    return api.ErrorResponse(403, "Access denied: You do not have permission to access this project")
}

// 404 Not Found - Resource doesn't exist
project, err := projectRepo.GetProjectByID(ctx, projectID, claims.OrgID)
if err != nil {
    if err == sql.ErrNoRows {
        return api.ErrorResponse(404, "Project not found or has been deleted")
    }
    return api.ErrorResponse(500, "Internal server error occurred while processing request")
}

// 409 Conflict - Duplicate resource
_, err = userRepo.GetUserByEmail(ctx, email, orgID)
if err == nil {
    return api.ErrorResponse(409, "User with this email already exists")
}

// 500 Internal Server Error - Database error
if err != nil {
    logger.WithError(err).Error("Database error")
    return api.ErrorResponse(500, "Internal server error occurred while processing request")
}
```

### Logging Error Details

```go
// Log error with context (never expose in response to client)
logger.WithFields(logrus.Fields{
    "user_id": claims.UserID,
    "project_id": projectID,
    "error": err.Error(),
    "operation": "GetProject",
}).Error("Failed to retrieve project from database")

// Return generic error to client
return api.ErrorResponse(500, "Internal server error occurred while processing request")
```

---

## Frontend Error Handling

### JavaScript/TypeScript Pattern

```typescript
async function createProject(projectData: ProjectRequest): Promise<Project> {
  try {
    const response = await fetch(`${API_BASE_URL}/projects`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${idToken}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(projectData)
    });

    const data = await response.json();

    if (!response.ok) {
      // Handle specific error codes
      switch (response.status) {
        case 400:
          throw new ValidationError(data.message);
        case 401:
          throw new AuthenticationError('Please log in again');
        case 403:
          throw new AuthorizationError(data.message);
        case 404:
          throw new NotFoundError(data.message);
        case 409:
          throw new ConflictError(data.message);
        case 500:
          throw new ServerError('Server error. Please try again later.');
        default:
          throw new Error(data.message || 'Unknown error occurred');
      }
    }

    return data.data as Project;
  } catch (error) {
    // Handle network errors
    if (error instanceof TypeError) {
      throw new NetworkError('Network error. Please check your connection.');
    }
    throw error;
  }
}
```

### Display User-Friendly Messages

```typescript
// Map technical errors to user-friendly messages
const ERROR_MESSAGES = {
  'Missing required field: title': 'Please enter a project title',
  'Access denied: You do not have permission to access this project':
    'You don\'t have access to this project. Please contact your administrator.',
  'Project not found or has been deleted':
    'This project no longer exists or has been removed.',
  'Internal server error occurred while processing request':
    'Something went wrong. Please try again in a few moments.',
};

function getUserFriendlyError(technicalMessage: string): string {
  return ERROR_MESSAGES[technicalMessage] || technicalMessage;
}
```

---

## Access Control Error Hierarchy

### Hierarchical Access Check Order

```go
// 1. Check authentication (401)
claims, err := auth.ExtractClaimsFromRequest(request)
if err != nil {
    return api.ErrorResponse(401, "Unauthorized: "+err.Error())
}

// 2. Check user is active (403)
user, err := userRepo.GetUserByID(ctx, claims.UserID)
if err != nil || user.Status != "active" {
    return api.ErrorResponse(403, "User account is not active")
}

// 3. Check organization access (403)
if claims.OrgID != resource.OrgID && !claims.IsSuperAdmin {
    return api.ErrorResponse(403, "Access denied: Organization mismatch")
}

// 4. Check context-specific access (403)
contexts, err := assignmentRepo.GetUserContexts(ctx, claims.UserID, claims.OrgID)
hasAccess := checkContextAccess(contexts, requiredContextType, requiredContextID)
if !hasAccess && !claims.IsSuperAdmin {
    return api.ErrorResponse(403, "Access denied: You do not have permission to access this resource")
}

// 5. Check resource exists (404)
resource, err := repo.GetByID(ctx, resourceID)
if err != nil {
    if err == sql.ErrNoRows {
        return api.ErrorResponse(404, "Resource not found or has been deleted")
    }
    return api.ErrorResponse(500, "Internal server error occurred while processing request")
}

// 6. Proceed with operation
```

---

## Testing Error Scenarios

### Postman Test Scripts

```javascript
// Test 400 Bad Request
pm.test("Status code is 400 for missing required field", function () {
    pm.response.to.have.status(400);
});

pm.test("Error message indicates missing field", function () {
    var jsonData = pm.response.json();
    pm.expect(jsonData.message).to.include("Missing required field");
});

// Test 403 Forbidden
pm.test("Status code is 403 for unauthorized access", function () {
    pm.response.to.have.status(403);
});

pm.test("Error message indicates access denied", function () {
    var jsonData = pm.response.json();
    pm.expect(jsonData.message).to.include("Access denied");
});

// Test 404 Not Found
pm.test("Status code is 404 for non-existent resource", function () {
    pm.response.to.have.status(404);
});

// Test 500 Internal Server Error
pm.test("Status code is 500 for server error", function () {
    pm.response.to.have.status(500);
});
```

---

## Best Practices

### DO:
- ✅ Return specific, actionable error messages
- ✅ Use appropriate HTTP status codes
- ✅ Log detailed error information server-side
- ✅ Return generic messages for security-sensitive errors
- ✅ Include CORS headers in all error responses
- ✅ Validate input early and return 400 errors quickly
- ✅ Distinguish between authentication (401) and authorization (403)

### DON'T:
- ❌ Expose stack traces or internal details to clients
- ❌ Return database error messages directly
- ❌ Use 200 status code for errors
- ❌ Return different error formats from different endpoints
- ❌ Log sensitive data (passwords, tokens) even in error logs
- ❌ Assume error message format - always check status code

---

**Last Updated:** 2025-10-27