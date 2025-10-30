# Authentication & Authorization Architecture

## Overview

The BuildBoard infrastructure implements a comprehensive authentication and authorization system using AWS Cognito for identity management and JWT tokens for secure API access. The system supports multi-tenant organization isolation, role-based access control, and super admin privileges.

## AWS Cognito Setup

### User Pool Configuration

The system uses a single AWS Cognito User Pool with the following configuration:

```typescript
const userPool = new UserPool(this, 'UserPool', {
    userPoolName: `${serviceName}-UserPool`,
    selfSignUpEnabled: true,
    signInAliases: {
        email: true,
        username: false
    },
    autoVerify: {
        email: true
    },
    passwordPolicy: {
        minLength: 8,
        requireLowercase: true,
        requireUppercase: true,
        requireDigits: true,
        requireSymbols: true
    },
    accountRecovery: AccountRecovery.EMAIL_ONLY,
    customAttributes: {
        isSuperAdmin: new StringAttribute({ mutable: true })
    },
    lambdaTriggers: {
        preTokenGeneration: tokenCustomizerLambda,
        postConfirmation: userSignupLambda
    }
});
```

### App Client Configuration

```typescript
const appClient = userPool.addClient('AppClient', {
    authFlows: {
        userPassword: true,
        userSrp: true,
        custom: true,
        adminUserPassword: true
    },
    generateSecret: false,  // Public client for frontend
    preventUserExistenceErrors: true,
    accessTokenValidity: Duration.hours(1),
    idTokenValidity: Duration.hours(1),
    refreshTokenValidity: Duration.days(30)
});
```

### Key Features

1. **Email-Based Authentication**: Users sign in with email (not username)
2. **Email Verification**: Required for all new accounts
3. **Password Policy**: Strong passwords enforced
4. **Custom Attributes**: `custom:isSuperAdmin` for super admin flag
5. **Lambda Triggers**: Token customization and post-confirmation hooks
6. **Token Validity**: 1-hour access/ID tokens, 30-day refresh tokens

## JWT Token Structure

### ID Token vs Access Token

The system uses **ID tokens** for API authentication (not access tokens):

```
ID Token: Contains user identity and profile claims
- Used for: API Gateway authentication
- Contains: User profile, org_id, locations, roles, permissions
- Size: Larger (includes custom claims)

Access Token: OAuth 2.0 access token
- Used for: OAuth scopes (not used in our system)
- Contains: Minimal identity information
- Size: Smaller

Why ID Tokens?
- ID tokens contain all user context needed for API operations
- No additional database lookups required per request
- Custom claims available in Lambda handler immediately
```

### Standard JWT Claims

Every token includes standard Cognito claims:

```json
{
    "sub": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "email": "user@example.com",
    "email_verified": true,
    "cognito:username": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "iss": "https://cognito-idp.us-east-2.amazonaws.com/us-east-2_XXXXX",
    "aud": "client-id",
    "token_use": "id",
    "auth_time": 1234567890,
    "iat": 1234567890,
    "exp": 1234571490
}
```

### Custom Claims (Added by Token Customizer)

The Token Customizer Lambda enriches tokens with profile data:

```json
{
    "user_id": "123",
    "cognito_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "full_name": "John Doe",
    "phone": "+1234567890",
    "job_title": "Project Manager",
    "status": "active",
    "avatar_url": "https://...",
    "org_id": "456",
    "org_name": "ACME Construction",
    "last_selected_location_id": "789",
    "isSuperAdmin": false,
    "locations": "eyJsb2NhdGlvbnMiOlsuLi5dfQ=="  // Base64-encoded JSON
}
```

### Claims Structure Explained

1. **User Identity**
   - `user_id`: Internal database ID (primary key in iam.users)
   - `cognito_id`: AWS Cognito UUID (sub claim)
   - `email`: User's email address

2. **Personal Information**
   - `first_name`, `last_name`, `full_name`: User's name
   - `phone`: Contact phone (optional)
   - `job_title`: Professional title (optional)
   - `avatar_url`: Profile photo URL (optional)

3. **Account Status**
   - `status`: active/inactive/pending/suspended
   - Used to check if user can access the system

4. **Organization Context**
   - `org_id`: User's organization ID (critical for multi-tenancy)
   - `org_name`: Organization display name

5. **Location Context**
   - `last_selected_location_id`: User's last selected location
   - `locations`: Base64-encoded JSON array of accessible locations with roles

6. **Super Admin Flag**
   - `isSuperAdmin`: Boolean flag for super admin access
   - Grants elevated permissions across the system

## Token Customizer Lambda (Pre-Token Generation)

### Purpose

The Token Customizer Lambda intercepts Cognito's token generation process and enriches JWT tokens with user profile data from the IAM database.

### Trigger Type

```
Trigger: Pre Token Generation V2.0
Event: TokenGeneration_Authentication, TokenGeneration_RefreshTokens, etc.
Version: V2.0 (enhanced version with better performance)
```

### Processing Flow

```
1. Cognito initiates token generation
   ↓
2. Token Customizer Lambda triggered
   ↓
3. Extract Cognito user ID from event.UserName
   ↓
4. Fetch complete user profile from database
   - Query: user_summary view
   - Includes: user info, org, locations, roles
   ↓
5. Build custom claims structure
   - Convert database model to JWT claims
   - Base64-encode complex data (locations)
   ↓
6. Add claims to both ID and Access tokens
   ↓
7. Activate pending users (if first login)
   ↓
8. Return enriched token to Cognito
   ↓
9. Cognito issues token to user
```

### Key Features

1. **Database Integration**
   ```go
   // Fetch complete user profile in single query
   userProfile, err := userRepository.GetUserProfile(cognitoID)
   ```

2. **Automatic User Activation**
   ```go
   // Activate pending users on first successful login
   if currentStatus == "pending" && event.TriggerSource == "TokenGeneration_Authentication" {
       userUpdate := &models.User{Status: "active"}
       _, err = userMgmtRepo.UpdateUser(ctx, userID, orgID, userUpdate, userID)
   }
   ```

3. **Complex Data Encoding**
   ```go
   // Base64-encode locations to minimize token size
   locationsJSON, _ := json.Marshal(profile.Locations)
   locationsEncoded := base64.StdEncoding.EncodeToString(locationsJSON)
   ```

4. **Graceful Error Handling**
   ```go
   // Never fail authentication due to database errors
   if err != nil {
       logger.Error("Failed to fetch user profile, proceeding without custom claims")
       return event, nil  // Return unchanged event
   }
   ```

5. **Token Customization for Both Token Types**
   ```go
   event.Response.ClaimsAndScopeOverrideDetails = events.ClaimsAndScopeOverrideDetailsV2_0{
       IDTokenGeneration: events.IDTokenGenerationV2_0{
           ClaimsToAddOrOverride: customClaims,
       },
       AccessTokenGeneration: events.AccessTokenGenerationV2_0{
           ClaimsToAddOrOverride: customClaims,
       },
   }
   ```

### Supported Trigger Sources

```go
func isValidTriggerSourceV2(triggerSource string) bool {
    validSources := []string{
        "TokenGeneration_HostedAuth",           // Cognito Hosted UI
        "TokenGeneration_Authentication",       // Direct auth
        "TokenGeneration_NewPasswordChallenge", // Password change
        "TokenGeneration_AuthenticateDevice",   // Device auth
        "TokenGeneration_RefreshTokens",        // Token refresh
    }
    // Check if trigger is in supported list
}
```

### Performance Optimization

1. **Cold Start Optimization**
   - Database connection pooled globally
   - SSM parameters cached
   - Repository initialized once

2. **Single Database Query**
   - Uses optimized `user_summary` view
   - Fetches all data in one query
   - No N+1 query problems

3. **Connection Reuse**
   - Global `*sql.DB` instance
   - Reused across Lambda invocations
   - Minimal connection overhead

## PostConfirmation Lambda (User Signup)

### Purpose

The PostConfirmation Lambda handles user registration by creating IAM database records after successful Cognito signup and email verification.

### Trigger Type

```
Trigger: Post Confirmation
Event: PostConfirmation_ConfirmSignUp, PostConfirmation_ConfirmForgotPassword
```

### Signup Scenarios

#### 1. Super Admin Signup (Self-Registration)

```
Flow:
1. User signs up via Cognito (email + password)
2. Cognito sends verification email
3. User confirms email
4. PostConfirmation Lambda triggered
5. Creates user with status "pending_org_setup"
6. User must complete org setup wizard
```

Implementation:
```go
func determineSignupType(event events.CognitoEventUserPoolsPostConfirmation) string {
    isSuperAdmin := event.Request.UserAttributes["custom:isSuperAdmin"]
    if isSuperAdmin == "true" {
        return "superadmin_signup"
    }
    return "unknown"  // Reject other signup types
}

// Create super admin user
if signupRequest.SignupType == "superadmin_signup" {
    user := &models.User{
        CognitoID: cognitoID,
        Email:     email,
        Status:    "pending_org_setup",
        OrgID:     systemOrgID,  // Temporary "System" org
    }
    _, err := createSuperAdminUser(ctx, user)
}
```

#### 2. Invited User Activation

```
Flow:
1. Admin creates user via dashboard (status = "pending")
2. Admin invites user via Cognito AdminCreateUser
3. User receives email with temporary password
4. User sets permanent password
5. PostConfirmation Lambda triggered
6. Updates user status to "active"
7. User can now login with full access
```

Implementation:
```go
// Detect invited user (already exists with pending status)
existingUser, err := getUserByCognitoID(cognitoID)
if err == nil && existingUser.Status == "pending" {
    // Activate the user
    err = updateUserStatus(existingUser.ID, "active")
}
```

### Error Handling Strategy

```go
func Handler(ctx context.Context, event events.CognitoEventUserPoolsPostConfirmation) (events.CognitoEventUserPoolsPostConfirmation, error) {
    // Never return error - always succeed Cognito confirmation
    err := processSignup(signupRequest)
    if err != nil {
        logger.Error("Failed to process signup, user can still login")
        return event, nil  // Return success to Cognito
    }
    return event, nil
}
```

**Why always return success?**
- Cognito signup should never fail due to database issues
- Users can still authenticate with Cognito
- Admin can manually fix user records if needed
- System remains available during database outages

## Claims Extraction in auth.go

### ExtractClaimsFromRequest Function

Every Lambda handler extracts claims from the API Gateway request context:

```go
type Claims struct {
    UserID       int64  `json:"user_id"`
    Email        string `json:"email"`
    CognitoID    string `json:"sub"`
    OrgID        int64  `json:"org_id"`
    IsSuperAdmin bool   `json:"isSuperAdmin"`
}

func ExtractClaimsFromRequest(request events.APIGatewayProxyRequest) (*Claims, error) {
    // Get claims from authorizer context
    claimsMap := request.RequestContext.Authorizer["claims"]

    // Extract user_id (handle both string and float64)
    var userID int64
    if userIDValue, exists := claimsMap["user_id"]; exists {
        if userIDStr, ok := userIDValue.(string); ok {
            userID, _ = strconv.ParseInt(userIDStr, 10, 64)
        } else if userIDFloat, ok := userIDValue.(float64); ok {
            userID = int64(userIDFloat)
        }
    }

    // Extract org_id (same type handling)
    var orgID int64
    if orgIDValue, exists := claimsMap["org_id"]; exists {
        if orgIDStr, ok := orgIDValue.(string); ok {
            orgID, _ = strconv.ParseInt(orgIDStr, 10, 64)
        } else if orgIDFloat, ok := orgIDValue.(float64); ok {
            orgID = int64(orgIDFloat)
        }
    }

    // Extract email and cognito_id
    email := claimsMap["email"].(string)
    cognitoID := claimsMap["sub"].(string)

    // Extract isSuperAdmin flag
    var isSuperAdmin bool
    if superAdminValue, exists := claimsMap["isSuperAdmin"]; exists {
        isSuperAdmin, _ = superAdminValue.(bool)
    }

    return &Claims{
        UserID:       userID,
        Email:        email,
        CognitoID:    cognitoID,
        OrgID:        orgID,
        IsSuperAdmin: isSuperAdmin,
    }, nil
}
```

### Type Handling Complexity

JWT claims can be represented as different types in Go:
- Strings: `"123"`
- Numbers: `123.0` (parsed as float64)
- Booleans: `true/false` or `"true"/"false"`

The extraction function handles all these cases for maximum compatibility.

## Authorization Patterns in Lambda Handlers

### Pattern 1: Super Admin Only

Certain operations require super admin privileges:

```go
func LambdaHandler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    claims, err := auth.ExtractClaimsFromRequest(request)
    if err != nil {
        return api.ErrorResponse(http.StatusUnauthorized, "Authentication failed", logger), nil
    }

    // Check if user is super admin
    if !claims.IsSuperAdmin {
        logger.WithField("user_id", claims.UserID).Warn("User is not a super admin")
        return api.ErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can access", logger), nil
    }

    // Process request
    return handleRequest(ctx, request, claims), nil
}
```

**Operations requiring super admin:**
- User creation/deletion
- Organization management
- Role creation/deletion
- Permission management
- Location creation (in some cases)

### Pattern 2: Self-Service Operations

Users can perform certain operations on their own data:

```go
func handleLocationUpdate(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) events.APIGatewayProxyResponse {
    userID, _ := strconv.ParseInt(request.PathParameters["userId"], 10, 64)

    // Allow users to update their own location, or super admins to update any user's location
    if !claims.IsSuperAdmin && claims.UserID != userID {
        return api.ErrorResponse(http.StatusForbidden, "You can only update your own location", logger)
    }

    // Process update
    return processLocationUpdate(ctx, userID, claims)
}
```

**Self-service operations:**
- Update own profile
- Change selected location
- View own projects
- View own assignments

### Pattern 3: Organization-Scoped Access

Most operations are scoped to user's organization:

```go
func handleGetUsers(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) events.APIGatewayProxyResponse {
    // Automatically scoped to user's organization
    users, err := userRepository.GetUsersByOrg(ctx, claims.OrgID)
    if err != nil {
        return api.ErrorResponse(http.StatusInternalServerError, "Failed to get users", logger)
    }
    return api.SuccessResponse(http.StatusOK, users, logger)
}
```

**Organization-scoped operations:**
- List users (only in user's org)
- List projects (only in user's org)
- List locations (only in user's org)
- All CRUD operations validate org ownership

### Pattern 4: Conditional Authorization

Some endpoints have different rules based on the resource:

```go
func LambdaHandler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    claims, err := auth.ExtractClaimsFromRequest(request)
    if err != nil {
        return api.ErrorResponse(http.StatusUnauthorized, "Authentication failed", logger), nil
    }

    // Allow any user to update their own selected location
    // Otherwise require super admin
    if request.Resource == "/users/{userId}/location" {
        // Self-service allowed
        return handleLocationUpdate(ctx, request, claims), nil
    }

    // All other user management requires super admin
    if !claims.IsSuperAdmin {
        return api.ErrorResponse(http.StatusForbidden, "Forbidden", logger), nil
    }

    // Route to appropriate handler
    return routeRequest(ctx, request, claims), nil
}
```

## Super Admin Detection

### is_super_admin Flag

Super admins are identified by the `is_super_admin` flag in the database:

```sql
-- iam.users table
CREATE TABLE iam.users (
    id BIGSERIAL PRIMARY KEY,
    cognito_id UUID UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    org_id BIGINT REFERENCES iam.organizations(id),
    is_super_admin BOOLEAN DEFAULT FALSE,
    status VARCHAR(50) DEFAULT 'active',
    -- other fields...
);
```

### Super Admin Creation

Super admins are created through self-registration:

1. User signs up with `custom:isSuperAdmin=true` attribute
2. PostConfirmation Lambda creates user with `is_super_admin = TRUE`
3. Token Customizer Lambda adds `isSuperAdmin: true` to JWT
4. All subsequent requests include super admin flag

### Super Admin Privileges

Super admins have elevated permissions:

```
✓ Create/update/delete users
✓ Manage organization settings
✓ Create/update/delete locations
✓ Create/update/delete roles
✓ Assign permissions to roles
✓ View all projects in organization
✓ Access admin dashboard
✓ Invite new users
```

Regular users have limited permissions:
```
✓ View own profile
✓ Update own profile (limited fields)
✓ Change own selected location
✓ View assigned projects
✗ Cannot create users
✗ Cannot modify organization
✗ Cannot manage roles/permissions
```

## Organization Extraction from Tokens

### Critical Security Rule

**NEVER trust org_id from request body - ALWAYS use org_id from JWT token**

### Correct Pattern

```go
func handleCreateIssue(ctx context.Context, userID, orgID int64, body string) events.APIGatewayProxyResponse {
    var createReq models.CreateIssueRequest
    json.Unmarshal([]byte(body), &createReq)

    // CORRECT: Use org_id from JWT token (passed as parameter from handler)
    issue, err := issueRepository.CreateIssue(ctx, projectID, userID, orgID, &createReq)

    // orgID is from claims.OrgID extracted from JWT
    // Never use createReq.OrgID even if present in request body
}
```

### Why This Matters

```
Bad Request Example (Security Vulnerability):
POST /issues
{
    "title": "Issue",
    "org_id": 999  ← Attacker tries to create issue in different org
}

If we used org_id from request body:
- Attacker could access other organizations' data
- Multi-tenancy would be completely broken

Correct Approach:
1. Extract org_id from JWT token (claims.OrgID)
2. Ignore any org_id in request body
3. Use JWT org_id for all database operations
4. Validate resources belong to JWT org_id
```

### Repository Validation

Repositories double-check organization ownership:

```go
func (dao *IssueDao) CreateIssue(ctx context.Context, projectID, userID, orgID int64, req *models.CreateIssueRequest) (*models.IssueResponse, error) {
    // Validate project belongs to organization (from JWT)
    var projectOrgID int64
    err := dao.DB.QueryRowContext(ctx, `
        SELECT org_id FROM project.projects
        WHERE id = $1 AND is_deleted = FALSE
    `, projectID).Scan(&projectOrgID)

    if projectOrgID != orgID {
        return nil, fmt.Errorf("project does not belong to your organization")
    }

    // Now safe to create issue
    // ...
}
```

This provides defense-in-depth:
1. **Handler Level**: Only passes org_id from JWT
2. **Repository Level**: Validates entity belongs to that org
3. **Database Level**: Foreign key constraints

## Permission Checking Patterns

### Current Implementation

The system uses **role-based access control (RBAC)** with the following structure:

```
Organizations
  └─ Locations
      └─ Users (with roles at location)
          └─ Roles
              └─ Permissions

Example:
- User "John" at Location "Building A"
- Has Role "Project Manager"
- Role has Permissions ["create_issue", "approve_rfi", "view_submittals"]
```

### Permission Storage

Permissions are stored in the JWT token:

```json
{
    "locations": "eyJ..."  // Base64-encoded JSON
}

// Decoded locations:
[
    {
        "location_id": "123",
        "location_name": "Building A",
        "roles": [
            {
                "role_id": "456",
                "role_name": "Project Manager",
                "permissions": ["create_issue", "approve_rfi", "view_submittals"]
            }
        ]
    }
]
```

### Permission Checking (Future Enhancement)

Current system validates at super admin level only. Granular permission checking would look like:

```go
// Future implementation
func hasPermission(claims *auth.Claims, permission string, locationID int64) bool {
    // Decode locations from JWT
    locations := decodeLocations(claims.Locations)

    // Find location
    for _, loc := range locations {
        if loc.LocationID == locationID {
            // Check all roles at this location
            for _, role := range loc.Roles {
                if contains(role.Permissions, permission) {
                    return true
                }
            }
        }
    }
    return false
}

// Usage in handler
func handleCreateIssue(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) events.APIGatewayProxyResponse {
    if !hasPermission(claims, "create_issue", locationID) {
        return api.ErrorResponse(http.StatusForbidden, "Missing permission: create_issue", logger)
    }
    // Process request
}
```

## Token Expiration and Refresh

### Token Lifetimes

```
ID Token: 1 hour
Access Token: 1 hour
Refresh Token: 30 days
```

### Token Refresh Flow

```
1. Frontend detects expired ID token (401 error or JWT expiry check)
   ↓
2. Frontend calls Cognito InitiateAuth with refresh token
   ↓
3. Cognito validates refresh token
   ↓
4. Token Customizer Lambda triggered (TokenGeneration_RefreshTokens)
   ↓
5. Fetch latest user profile from database
   ↓
6. Generate new ID token with current user data
   ↓
7. Return new ID token to frontend
   ↓
8. Frontend stores new token and retries request
```

### Silent Token Refresh

Frontend should implement silent refresh:

```typescript
// Example frontend code
async function getValidToken() {
    const token = localStorage.getItem('id_token');
    const expiresAt = localStorage.getItem('expires_at');

    // Refresh if expired or expiring soon (within 5 minutes)
    if (Date.now() >= expiresAt - 5 * 60 * 1000) {
        const refreshToken = localStorage.getItem('refresh_token');
        const newToken = await refreshTokenWithCognito(refreshToken);
        localStorage.setItem('id_token', newToken.id_token);
        localStorage.setItem('expires_at', Date.now() + 3600 * 1000);
        return newToken.id_token;
    }

    return token;
}
```

### Token Refresh Benefits

1. **Always Current Data**: User profile refreshed on every token renewal
2. **Status Changes**: Account status changes reflected immediately
3. **Role Updates**: Permission changes applied on next refresh
4. **Organization Changes**: Org updates reflected in new tokens

## Security Best Practices

### Token Storage (Frontend)

```typescript
// CORRECT: Use secure storage
localStorage.setItem('id_token', token);  // Acceptable for ID tokens
sessionStorage.setItem('id_token', token);  // More secure, lost on close

// INCORRECT: Never store in cookies without HttpOnly/Secure flags
document.cookie = `token=${token}`;  // Vulnerable to XSS
```

### Token Transmission

```typescript
// CORRECT: Use Authorization header
fetch('/api/users', {
    headers: {
        'Authorization': `Bearer ${idToken}`
    }
});

// INCORRECT: Never send in URL or query parameters
fetch(`/api/users?token=${idToken}`);  // Token leaked in logs
```

### Token Validation

API Gateway automatically validates:
1. Token signature (using Cognito public keys)
2. Token expiration (exp claim)
3. Token issuer (iss claim)
4. Token audience (aud claim)

Lambda handlers only need to:
1. Extract claims from request context
2. Validate organization ownership
3. Check super admin flag if needed

### Defense in Depth

The system implements multiple security layers:

```
Layer 1: API Gateway
- Validates JWT signature
- Checks token expiration
- Verifies issuer and audience

Layer 2: Lambda Handler
- Extracts org_id from JWT (not request body)
- Checks super admin flag
- Validates input

Layer 3: Repository
- Validates entity belongs to org_id
- Checks soft delete status
- Enforces foreign key relationships

Layer 4: Database
- Foreign key constraints
- Check constraints
- Row-level security (future)
```

## Testing Authentication

### Getting Test Tokens

```bash
# Test user credentials
EMAIL="buildboard007+555@gmail.com"
PASSWORD="Mayur@1234"

# Login to get tokens
aws cognito-idp initiate-auth \
    --region us-east-2 \
    --auth-flow USER_PASSWORD_AUTH \
    --client-id <app-client-id> \
    --auth-parameters USERNAME=$EMAIL,PASSWORD=$PASSWORD

# Response includes:
# - IdToken (use this for API calls)
# - AccessToken (not used)
# - RefreshToken (use for token refresh)
```

### Testing API Endpoints

```bash
# Get ID token from login response
ID_TOKEN="eyJraWQiOiJ..."

# Call API with ID token
curl -X GET https://api.example.com/users \
    -H "Authorization: Bearer $ID_TOKEN" \
    -H "Content-Type: application/json"
```

### Common Testing Mistakes

```
❌ Using Access Token instead of ID Token
- Access tokens don't contain custom claims
- API will fail to extract org_id

❌ Expired tokens
- Tokens expire after 1 hour
- Get fresh token if requests fail with 401

❌ Wrong token format
- Must use "Bearer <token>" format
- Don't send token in cookies or query params

❌ Testing without email verification
- Users must verify email before tokens work
- Check Cognito email verification status
```

## Troubleshooting Guide

### "Authentication failed" (401)

```
Possible causes:
1. Token expired (check exp claim)
2. Invalid token signature
3. Token from wrong user pool
4. Token not in Authorization header

Solution:
- Refresh token using refresh token
- Verify Authorization header format: "Bearer <token>"
- Check token issuer matches Cognito user pool
```

### "Forbidden" (403)

```
Possible causes:
1. Not a super admin (requires isSuperAdmin: true)
2. Accessing resource in different organization
3. Insufficient permissions for operation

Solution:
- Check claims.IsSuperAdmin in JWT
- Verify claims.OrgID matches resource org
- Confirm user has required role/permissions
```

### "Claims not found in authorizer context"

```
Possible causes:
1. API Gateway authorizer not configured
2. Cognito authorizer failing silently
3. Token validation failing at API Gateway

Solution:
- Verify API Gateway has Cognito authorizer attached
- Check CloudWatch logs for authorizer errors
- Validate token using jwt.io
```

### Token doesn't contain custom claims

```
Possible causes:
1. Token Customizer Lambda failed
2. Database connection issues
3. User not found in IAM database

Solution:
- Check Token Customizer Lambda CloudWatch logs
- Verify user exists in iam.users table
- Check database connection from Lambda VPC
```

## Related Documentation

- [API Architecture](./api-architecture.md)
- [Database Schema](../sql/)
- [Quick Start Guide](../QUICK-START.md)
- [API Testing Guide](../api/)