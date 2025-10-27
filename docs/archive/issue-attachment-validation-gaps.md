# Attachment & Issue Management API - Validation Gaps Analysis

## Problem
APIs are returning generic 500 errors instead of proper validation errors (400/403/404) with descriptive messages.

## Grey Areas Identified

### 1. Attachment Upload-URL API (`/attachments/upload-url`)

**Current Issues:**
- ✅ Validates required fields (entity_type, project_id, location_id, file_name)
- ✅ Validates file type
- ❌ **MISSING**: Validate entity (issue/RFI/submittal) exists
- ❌ **MISSING**: Validate entity belongs to the specified project
- ❌ **MISSING**: Validate project exists and belongs to org
- ❌ **MISSING**: Validate project belongs to the specified location
- ❌ **MISSING**: Validate entity is not deleted
- ❌ **MISSING**: Foreign key violations return 500 instead of descriptive 400 error

**Error Scenarios to Handle:**
1. Issue ID 999 doesn't exist → Return 404 "Issue not found"
2. Issue 85 belongs to project 59, but request says project 60 → Return 400 "Issue does not belong to the specified project"
3. Project 60 belongs to org 10, but user is from org 13 → Return 403 "Access denied"
4. Project 60 belongs to location 24, but request says location 1 → Return 400 "Project does not belong to the specified location"
5. Issue is deleted → Return 404 "Issue not found"

### 2. Issue Management API

**Current State:**
- ✅ Good validation in most handlers
- ✅ Validates project belongs to org
- ✅ Validates assigned_to user exists and belongs to org
- ❌ **MISSING**: Some foreign key errors could be more descriptive

### 3. RFI Management API (Need to check)

### 4. Submittal Management API (Need to check)

## Recommended Fixes

### For Attachment Management API

Add validation helper functions:

```go
// ValidateEntityAccess validates that:
// 1. Entity exists
// 2. Entity belongs to specified project
// 3. Project belongs to specified org
// 4. Project belongs to specified location (if location validation needed)
// 5. Entity is not deleted
func ValidateEntityAccess(ctx context.Context, db *sql.DB, entityType string, entityID, projectID, locationID, orgID int64) error
```

### HTTP Status Code Guidelines

- **400 Bad Request**: Invalid input, validation failures, business logic violations
  - Missing required fields
  - Invalid format
  - Entity doesn't belong to specified parent
  - Invalid enum values

- **403 Forbidden**: User authenticated but doesn't have access
  - Entity belongs to different organization
  - No permission for the operation

- **404 Not Found**: Resource doesn't exist
  - Entity ID not found in database
  - Entity is soft-deleted

- **500 Internal Server Error**: Unexpected errors only
  - Database connection failures
  - Unexpected database errors (not constraint violations)
  - System errors

## Implementation Priority

1. **HIGH**: Add entity validation to attachment upload-URL API
2. **MEDIUM**: Add better error messages for foreign key violations
3. **LOW**: Review and improve error handling in RFI and Submittal APIs
