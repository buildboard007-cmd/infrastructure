# API Super Admin Restrictions Documentation

## Overview
This document lists all APIs that currently require super admin privileges in the infrastructure system. These restrictions may need to be reviewed and potentially modified based on business requirements.

## Current Super Admin Required APIs

### 1. User Management Service
**File**: `src/infrastructure-user-management/main.go`

| Endpoint | Method | Description | Restriction Location | Notes |
|----------|--------|-------------|---------------------|-------|
| `/users` | POST | Create new user | Lines 54-56 | Only super admins can create users |
| `/users` | GET | List all users | Lines 54-56 | Only super admins can view all users |
| `/users/{userId}` | PUT | Update user details | Lines 54-56 | Only super admins can update user details |
| `/users/{userId}` | DELETE | Delete user | Lines 54-56 | Only super admins can delete users |
| `/users/{userId}/location` | PUT | Update user location | Lines 235-237 | **Exception**: Users can update their own location, super admins can update any user's location |

**Authorization Check**:
```go
if request.Resource != "/users/{userId}/location" && !claims.IsSuperAdmin {
    return api.ErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can manage users", logger), nil
}
```

### 2. Organization Management Service
**File**: `src/infrastructure-organization-management/main.go`

| Endpoint | Method | Description | Restriction Location | Notes |
|----------|--------|-------------|---------------------|-------|
| `/organizations/*` | ALL | All organization operations | Lines 58-60 | Only super admins can manage organizations |

**Authorization Check**:
```go
if !claims.IsSuperAdmin {
    return api.ErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can manage organization", logger), nil
}
```

### 3. Location Management Service
**File**: `src/infrastructure-location-management/main.go`

| Endpoint | Method | Description | Restriction Location | Notes |
|----------|--------|-------------|---------------------|-------|
| `/locations/*` | ALL | All location operations | Lines 49-51 | Only super admins can manage locations |

**Authorization Check**:
```go
if !claims.IsSuperAdmin {
    return api.ErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can manage locations", logger), nil
}
```

### 4. Roles Management Service
**File**: `src/infrastructure-roles-management/main.go`

| Endpoint | Method | Description | Restriction Location | Notes |
|----------|--------|-------------|---------------------|-------|
| `/roles/*` | ALL | All role operations | Lines 50-52 | Only super admins can manage roles |

**Authorization Check**:
```go
if !claims.IsSuperAdmin {
    return api.ErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can manage roles", logger), nil
}
```

### 5. Permissions Management Service
**File**: `src/infrastructure-permissions-management/main.go`

| Endpoint | Method | Description | Restriction Location | Notes |
|----------|--------|-------------|---------------------|-------|
| `/permissions/*` | ALL | All permission operations | Lines 49-51 | Only super admins can manage permissions |

**Authorization Check**:
```go
if !claims.IsSuperAdmin {
    return api.ErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can manage permissions", logger), nil
}
```

## Authentication Flow

1. All APIs use AWS Cognito for authentication via API Gateway
2. JWT tokens contain `IsSuperAdmin` claim extracted from the database
3. The token customizer Lambda (`src/infrastructure-token-customizer/main.go`) adds the `IsSuperAdmin` flag to the JWT during authentication
4. Each Lambda function validates the `IsSuperAdmin` claim before allowing access to protected endpoints

## Potential Changes to Consider

### 1. Role-Based Access Control (RBAC) Enhancement
Consider implementing more granular permissions instead of binary super admin checks:
- Organization Admin: Can manage users within their organization
- Location Manager: Can manage specific locations
- User Manager: Can create/update users but not delete

### 2. Delegation of User Management
Allow organization admins to:
- Create users within their organization
- Update users within their organization
- View users within their organization

### 3. Location Access Control
Consider allowing:
- Location managers to update location details
- Project managers to view location information
- Users to view locations they have access to

### 4. API Changes Needed for RBAC

#### User Management
- Modify GET /users to return filtered list based on user's organization/role
- Allow organization admins to create users with POST /users (with org validation)
- Allow users to view their own profile with GET /users/{userId}

#### Location Management
- Allow users to GET /locations based on their assigned locations
- Allow location managers to update their assigned locations

#### Organization Management
- Allow organization admins to update their own organization details
- Allow users to view their organization information

## Implementation Notes

To implement these changes:

1. Update the authorization checks in each Lambda function
2. Modify the database queries to filter results based on user permissions
3. Update the token customizer to include more detailed permission claims
4. Test thoroughly to ensure no security vulnerabilities are introduced

## Security Considerations

- Always validate organization membership before allowing any operations
- Implement audit logging for all administrative actions
- Consider implementing approval workflows for sensitive operations
- Regular security reviews of permission assignments

## Related Files

- `/src/lib/auth/auth.go` - Authentication utilities
- `/src/infrastructure-token-customizer/main.go` - JWT token customization
- `/src/lib/models/user.go` - User model with IsSuperAdmin field
- `/src/lib/data/user_repository.go` - User data access layer

## Last Updated
Date: 2025-01-14