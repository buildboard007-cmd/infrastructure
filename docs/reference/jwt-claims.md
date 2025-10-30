# JWT Claims Reference

> Complete guide to JWT token structure, custom claims, and usage patterns

**Token Type:** AWS Cognito ID Token (NOT Access Token)
**Customizer:** Pre-Token Generation V2.0 Lambda trigger

---

## Standard Cognito Claims

These claims are automatically included by AWS Cognito:

| Claim | Type | Description | Example |
|-------|------|-------------|---------|
| `sub` | string | Cognito User UUID (unique identifier) | `"a1b2c3d4-..."` |
| `email_verified` | boolean | Email verification status | `true` |
| `iss` | string | Token issuer (Cognito User Pool URL) | `"https://cognito-idp.us-east-2.amazonaws.com/us-east-2_VkTLMp9RZ"` |
| `cognito:username` | string | Cognito username (usually email) | `"buildboard007+555@gmail.com"` |
| `aud` | string | Audience (Cognito App Client ID) | `"3f0fb5mpivctnvj85tucusf88e"` |
| `token_use` | string | Token type | `"id"` |
| `auth_time` | number | Authentication timestamp (epoch) | `1698765432` |
| `exp` | number | Expiration timestamp (epoch) | `1698769032` |
| `iat` | number | Issued at timestamp (epoch) | `1698765432` |

---

## Custom Claims (Added by Token Customizer)

These claims are injected by the Pre-Token Generation Lambda:

### User Identity Claims

| Claim | Type | Description | Example |
|-------|------|-------------|---------|
| `user_id` | string | Internal database user ID | `"19"` |
| `cognito_id` | string | AWS Cognito UUID (same as `sub`) | `"a1b2c3d4-..."` |
| `email` | string | User email address | `"buildboard007+555@gmail.com"` |
| `first_name` | string | User first name | `"Mayur"` |
| `last_name` | string | User last name | `"Patel"` |
| `full_name` | string | Computed full name | `"Mayur Patel"` |
| `phone` | string | Contact phone number | `"+1-555-123-4567"` |
| `job_title` | string | Professional title | `"Project Manager"` |
| `status` | string | Account status | `"active"` |
| `avatar_url` | string | Profile photo URL | `"https://..."` |

### Organization & Location Claims

| Claim | Type | Description | Example |
|-------|------|-------------|---------|
| `org_id` | string | Organization ID | `"10"` |
| `org_name` | string | Organization name | `"BuildBoard Construction"` |
| `last_selected_location_id` | string | User's last selected location (UI preference) | `"6"` |

### Access Control Claims

| Claim | Type | Description | Example |
|-------|------|-------------|---------|
| `isSuperAdmin` | boolean | Global admin flag | `true` |
| `locations` | string | Base64-encoded JSON of accessible locations with roles | `"eyJsb2NhdGlvbnMi..."` |

---

## Locations Claim Structure

The `locations` claim contains Base64-encoded JSON with user's accessible locations and their roles at each location.

### Encoding Format

```javascript
// Original data structure
const locationsData = [
  {
    "location_id": 6,
    "location_name": "Main Office",
    "roles": [
      {
        "role_id": 1,
        "role_name": "Project Manager",
        "access_level": "project"
      }
    ]
  }
];

// Encoded in JWT
const locationsEncoded = btoa(JSON.stringify(locationsData));
// "eyJsb2NhdGlvbnMiOlsiNiJdLCJyb2xlcyI6eyI2IjpbMV19fQ=="
```

### Decoding in Frontend

```javascript
// JavaScript/TypeScript
const locationsJSON = atob(token.locations);
const locations = JSON.parse(locationsJSON);

// Access location data
locations.forEach(loc => {
  console.log(`Location: ${loc.location_name}`);
  loc.roles.forEach(role => {
    console.log(`  Role: ${role.role_name} (${role.access_level})`);
  });
});
```

### Decoding in Go

```go
import (
    "encoding/base64"
    "encoding/json"
)

type LocationWithRoles struct {
    LocationID   int64  `json:"location_id"`
    LocationName string `json:"location_name"`
    Roles        []Role `json:"roles"`
}

type Role struct {
    RoleID      int64  `json:"role_id"`
    RoleName    string `json:"role_name"`
    AccessLevel string `json:"access_level"`
}

// Decode locations from JWT
func DecodeLocations(locationsEncoded string) ([]LocationWithRoles, error) {
    decoded, err := base64.StdEncoding.DecodeString(locationsEncoded)
    if err != nil {
        return nil, err
    }

    var locations []LocationWithRoles
    err = json.Unmarshal(decoded, &locations)
    return locations, err
}
```

---

## Extracting Claims in Lambda Functions

### Go Implementation

```go
package auth

import (
    "fmt"
    "strconv"
    "github.com/aws/aws-lambda-go/events"
)

type Claims struct {
    UserID       int64  `json:"user_id"`
    Email        string `json:"email"`
    CognitoID    string `json:"sub"`
    OrgID        int64  `json:"org_id"`
    IsSuperAdmin bool   `json:"isSuperAdmin"`
}

// ExtractClaimsFromRequest extracts JWT claims from API Gateway request
func ExtractClaimsFromRequest(request events.APIGatewayProxyRequest) (*Claims, error) {
    // Claims are in request.RequestContext.Authorizer["claims"]
    var claimsMap map[string]interface{}
    var ok bool

    if authClaims, exists := request.RequestContext.Authorizer["claims"]; exists {
        claimsMap, ok = authClaims.(map[string]interface{})
    }

    if !ok {
        // Fallback to direct authorizer context
        claimsMap = request.RequestContext.Authorizer
        ok = (claimsMap != nil)
    }

    if !ok || claimsMap == nil {
        return nil, fmt.Errorf("claims not found in authorizer context")
    }

    // Parse user_id (string or number)
    var userID int64
    if userIDValue, exists := claimsMap["user_id"]; exists {
        if userIDStr, ok := userIDValue.(string); ok {
            userID, _ = strconv.ParseInt(userIDStr, 10, 64)
        } else if userIDFloat, ok := userIDValue.(float64); ok {
            userID = int64(userIDFloat)
        }
    }

    // Parse org_id (string or number)
    var orgID int64
    if orgIDValue, exists := claimsMap["org_id"]; exists {
        if orgIDStr, ok := orgIDValue.(string); ok {
            orgID, _ = strconv.ParseInt(orgIDStr, 10, 64)
        } else if orgIDFloat, ok := orgIDValue.(float64); ok {
            orgID = int64(orgIDFloat)
        }
    }

    // Extract string claims
    email, _ := claimsMap["email"].(string)
    cognitoID, _ := claimsMap["sub"].(string)

    // Extract boolean
    isSuperAdmin, _ := claimsMap["isSuperAdmin"].(bool)

    return &Claims{
        UserID:       userID,
        Email:        email,
        CognitoID:    cognitoID,
        OrgID:        orgID,
        IsSuperAdmin: isSuperAdmin,
    }, nil
}
```

### Usage in Lambda Handler

```go
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    // Extract claims
    claims, err := auth.ExtractClaimsFromRequest(request)
    if err != nil {
        return api.ErrorResponse(401, "Unauthorized: "+err.Error())
    }

    // Use claims for authorization
    if !claims.IsSuperAdmin && claims.OrgID != requiredOrgID {
        return api.ErrorResponse(403, "Forbidden: Access denied")
    }

    // Access user info
    userID := claims.UserID
    orgID := claims.OrgID

    // Continue with business logic...
}
```

---

## Token Lifecycle

### 1. User Authentication

```bash
# User logs in with Cognito
POST https://cognito-idp.us-east-2.amazonaws.com/
{
  "AuthFlow": "USER_PASSWORD_AUTH",
  "ClientId": "3f0fb5mpivctnvj85tucusf88e",
  "AuthParameters": {
    "USERNAME": "buildboard007+555@gmail.com",
    "PASSWORD": "Mayur@1234"
  }
}
```

### 2. Token Generation Trigger

- Cognito triggers Pre-Token Generation V2.0 Lambda
- Lambda fetches user profile from `iam.users` and `iam.user_assignments`
- Lambda injects custom claims into token
- If user status is `pending`, Lambda auto-activates to `active`

### 3. Token Response

```json
{
  "AuthenticationResult": {
    "IdToken": "eyJraWQiOiJ...",
    "AccessToken": "eyJraWQiOiJ...",
    "RefreshToken": "eyJjdHkiOiJ...",
    "ExpiresIn": 3600,
    "TokenType": "Bearer"
  }
}
```

### 4. API Request with Token

```bash
GET /projects
Authorization: Bearer eyJraWQiOiJ...
```

### 5. API Gateway Validation

- API Gateway validates JWT signature
- Cognito User Pool authorizer verifies token
- Claims are passed to Lambda in `request.RequestContext.Authorizer`

---

## Token Customization Triggers

### Supported Trigger Sources

| Trigger Source | When Fired | Use Case |
|----------------|------------|----------|
| `TokenGeneration_Authentication` | Username/password login | Standard login flow |
| `TokenGeneration_HostedAuth` | Cognito Hosted UI | Hosted UI authentication |
| `TokenGeneration_RefreshTokens` | Token refresh | Refresh token flow |
| `TokenGeneration_NewPasswordChallenge` | Password change | First-time login with temp password |

### Activation Logic

```go
// Automatically activate pending users on first successful authentication
if currentStatus == "pending" && triggerSource == "TokenGeneration_Authentication" {
    userUpdate := &models.User{Status: "active"}
    userMgmtRepo.UpdateUser(ctx, userID, orgID, userUpdate, userID)
}
```

---

## Claim Validation Patterns

### Check Super Admin Access

```go
if !claims.IsSuperAdmin {
    return api.ErrorResponse(403, "Super Admin access required")
}
```

### Check Organization Access

```go
if claims.OrgID != resource.OrgID && !claims.IsSuperAdmin {
    return api.ErrorResponse(403, "Access denied to this organization")
}
```

### Check User is Active

```go
// Status is NOT in JWT - must query database
user, err := userRepo.GetUserByID(ctx, claims.UserID)
if err != nil || user.Status != "active" {
    return api.ErrorResponse(403, "User account is not active")
}
```

---

## Important Notes

1. **Use ID Tokens, NOT Access Tokens**
   - ID tokens contain custom claims
   - Access tokens are for OAuth scopes only
   - Always send ID token in `Authorization: Bearer <id_token>` header

2. **Token Size Limitations**
   - JWT tokens have ~8KB size limit
   - Locations data is Base64-encoded to minimize size
   - Only essential data is included in token

3. **Token Refresh**
   - Tokens expire after 1 hour (3600 seconds)
   - Use refresh token to get new ID token
   - Custom claims are re-populated on each refresh

4. **Claims are Read-Only**
   - Claims represent token generation time state
   - Changes to user data require token refresh
   - Always query database for real-time data

5. **Roles are NOT in JWT**
   - Only locations are in JWT
   - Roles are fetched per-project when needed
   - Use `iam.user_assignments` table for current roles

6. **Super Admin Flag**
   - `isSuperAdmin` bypasses all access control
   - Super Admin can access any organization
   - Use sparingly - only for platform administrators

---

## Debugging Token Issues

### Decode JWT Online

Use [jwt.io](https://jwt.io) to decode and inspect token contents (dev only - never paste production tokens!)

### View Claims in CloudWatch

```go
logger.WithFields(logrus.Fields{
    "user_id": claims.UserID,
    "org_id": claims.OrgID,
    "email": claims.Email,
    "is_super_admin": claims.IsSuperAdmin,
}).Debug("Processing request with claims")
```

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| `claims not found` | Using Access Token instead of ID Token | Use ID Token |
| `user_id missing` | Old token before customizer | Refresh token |
| `org_id mismatch` | User switched organizations | Refresh token |
| `Invalid signature` | Token from different environment | Check User Pool ID |

---

**Last Updated:** 2025-10-27