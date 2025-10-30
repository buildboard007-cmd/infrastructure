# API Architecture

## Overview

The BuildBoard infrastructure uses a serverless microservices architecture with multiple REST APIs powered by AWS API Gateway and Lambda functions. The system is organized into 11 specialized Lambda management services, each handling specific domain operations with a consistent repository pattern for data access.

## Architecture Components

### API Gateway Organization

The system deploys **multiple independent API Gateways** to organize endpoints by domain:

1. **IAM API** (`iam-api`)
   - User Management
   - Organization Management
   - Location Management
   - Role Management
   - Permission Management

2. **Projects API** (`projects-api`)
   - Project Management
   - Project Managers
   - Project Attachments
   - Project User Assignments

3. **Issues API** (`issues-api`)
   - Issue Management
   - Issue Comments
   - Issue Status Tracking
   - Project-specific Issues

4. **RFIs API** (`rfis-api`)
   - RFI Management
   - RFI Workflow (Submit/Respond)
   - RFI Status Updates

Each API Gateway has:
- Independent REST API endpoint
- Dedicated Cognito User Pools Authorizer
- Shared CORS Lambda integration
- Stage-based deployment (dev/staging/prod)

### Lambda Function Organization

The infrastructure consists of **14 Lambda functions** organized into 3 categories:

#### Management Services (11 Lambdas)
Each service follows identical architectural patterns:

1. **infrastructure-user-management**
   - Routes: POST/GET/PUT/DELETE/PATCH `/users`
   - Cognito integration for user lifecycle
   - Password reset functionality
   - Selected location management

2. **infrastructure-organization-management**
   - Routes: GET/PUT `/org`
   - Organization profile management

3. **infrastructure-location-management**
   - Routes: POST/GET/PUT/DELETE `/locations`
   - Physical location hierarchy

4. **infrastructure-roles-management**
   - Routes: POST/GET/PUT/DELETE `/roles`
   - Role-permission mapping

5. **infrastructure-permissions-management**
   - Routes: POST/GET/PUT/DELETE `/permissions`
   - Permission definitions

6. **infrastructure-project-management**
   - Routes: POST/GET/PUT/DELETE `/projects`
   - Project managers and attachments
   - User assignments

7. **infrastructure-issue-management**
   - Routes: POST/GET/PUT/DELETE/PATCH `/issues`
   - Issue comments and attachments
   - Status workflow

8. **infrastructure-rfi-management**
   - Routes: POST/GET/PUT/DELETE/PATCH `/rfis`
   - RFI workflow (submit/respond/review)

9. **infrastructure-submittal-management**
   - Routes: POST/GET/PUT/DELETE `/submittals`
   - Submittal workflow

10. **infrastructure-assignment-management**
    - Routes: POST/GET/PUT/DELETE `/assignments`
    - User-project assignments

11. **infrastructure-attachment-management**
    - Routes: POST/GET/DELETE `/attachments`
    - S3 file uploads with presigned URLs
    - Entity attachment relationships

#### Authentication Services (2 Lambdas)

12. **infrastructure-token-customizer**
    - Cognito Pre-Token Generation V2.0 trigger
    - Enriches JWT tokens with user profile data
    - Adds custom claims (org_id, locations, roles)

13. **infrastructure-user-signup**
    - Cognito Post-Confirmation trigger
    - Creates IAM database record on signup
    - Handles super admin registration

#### Infrastructure Services (1 Lambda)

14. **infrastructure-api-gateway-cors**
    - Handles OPTIONS preflight requests
    - Validates origin against allowed list
    - Returns CORS headers

## Request/Response Flow

### Standard API Request Flow

```
1. Client Request
   ↓
2. API Gateway (domain-specific)
   ↓
3. Cognito Authorizer
   - Validates JWT ID token
   - Extracts claims to request context
   ↓
4. Lambda Handler (main.go)
   - auth.ExtractClaimsFromRequest()
   - Route by HTTP method and path
   ↓
5. Handler Function
   - Parse request body
   - Validate inputs
   - Call repository method
   ↓
6. Repository (Data Access Layer)
   - Validate org ownership
   - Execute SQL queries
   - Handle transactions
   ↓
7. Database (PostgreSQL RDS)
   - Execute query
   - Return results
   ↓
8. Response Generation
   - api.SuccessResponse() or api.ErrorResponse()
   - JSON serialization
   - CORS headers
   ↓
9. API Gateway Response
   ↓
10. Client receives response
```

### CORS Preflight Flow

```
1. OPTIONS Request
   ↓
2. API Gateway
   ↓
3. infrastructure-api-gateway-cors Lambda
   - Checks request origin header
   - Validates against ALLOWED_ORIGINS (SSM)
   - Returns 200 with CORS headers or 400
   ↓
4. Client receives CORS approval
```

## Repository Pattern (Data Access Layer)

### Pattern Structure

Every repository follows this interface pattern:

```go
// Repository Interface
type EntityRepository interface {
    CreateEntity(ctx context.Context, orgID int64, entity *Model) (*Response, error)
    GetEntityByID(ctx context.Context, entityID, orgID int64) (*Response, error)
    GetEntitiesByOrg(ctx context.Context, orgID int64) ([]Response, error)
    UpdateEntity(ctx context.Context, entityID, orgID int64, entity *Model) (*Response, error)
    DeleteEntity(ctx context.Context, entityID, orgID int64) error
}

// Repository Implementation
type EntityDao struct {
    DB     *sql.DB
    Logger *logrus.Logger
}
```

### Key Principles

1. **Organization Isolation**
   - Every query includes `org_id` validation
   - Prevents cross-organization data access
   - Organization ID always extracted from JWT token (never from request body)

2. **Soft Deletes**
   - Records marked with `is_deleted = TRUE`
   - All queries filter: `WHERE is_deleted = FALSE`
   - Maintains referential integrity

3. **Transaction Support**
   - Complex operations use `db.BeginTx()`
   - Rollback on error
   - Example: Issue creation with attachments

4. **Structured Logging**
   - Every operation logs with context fields
   - Correlation IDs for request tracking
   - Error details for debugging

5. **Nullable Field Handling**
   - Uses `sql.NullString`, `sql.NullInt64`, etc.
   - Explicit Valid flag checking
   - Converts to/from request models

## Handler Pattern (Request Routing)

### Standard Lambda Handler Structure

```go
func LambdaHandler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    // 1. Extract JWT claims
    claims, err := auth.ExtractClaimsFromRequest(request)
    if err != nil {
        return api.ErrorResponse(http.StatusUnauthorized, "Authentication failed", logger), nil
    }

    // 2. Authorization check (if needed)
    if !claims.IsSuperAdmin && requiresSuperAdmin(request.Resource) {
        return api.ErrorResponse(http.StatusForbidden, "Forbidden", logger), nil
    }

    // 3. Route by HTTP method
    switch request.HTTPMethod {
    case http.MethodPost:
        return handleCreate(ctx, request, claims), nil
    case http.MethodGet:
        if request.PathParameters["id"] != "" {
            return handleGetOne(ctx, request, claims), nil
        }
        return handleGetAll(ctx, request, claims), nil
    case http.MethodPut:
        return handleUpdate(ctx, request, claims), nil
    case http.MethodDelete:
        return handleDelete(ctx, request, claims), nil
    case http.MethodPatch:
        return handlePartialUpdate(ctx, request, claims), nil
    default:
        return api.ErrorResponse(http.StatusMethodNotAllowed, "Method not allowed", logger), nil
    }
}
```

### Route Matching Patterns

Handlers use multiple strategies to identify specific routes:

1. **Path Parameters**: `request.PathParameters["userId"]`
2. **Resource Matching**: `request.Resource == "/users/{userId}/reset-password"`
3. **String Contains**: `strings.Contains(request.Resource, "/issues/{issueId}/comments")`

Example from issue-management:
```go
case http.MethodPost:
    // POST /issues/{issueId}/comments - Add comment
    if strings.Contains(request.Resource, "/issues/{issueId}/comments") {
        return handleCreateComment(ctx, issueID, claims.UserID, claims.OrgID, request.Body), nil
    }
    // POST /issues - Create new issue
    if request.Resource == "/issues" {
        return handleCreateIssue(ctx, claims.UserID, claims.OrgID, request.Body), nil
    }
```

## CORS Handling

### Implementation

The `infrastructure-api-gateway-cors` Lambda handles all OPTIONS preflight requests:

```go
func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    requestOrigin := request.Headers["origin"]
    allowedOrigins := strings.Split(ssmParams[ALLOWED_ORIGINS], ",")

    for _, allowedOrigin := range allowedOrigins {
        if allowedOrigin == "*" || allowedOrigin == requestOrigin {
            return events.APIGatewayProxyResponse{
                StatusCode: 200,
                Headers: map[string]string{
                    "Access-Control-Allow-Origin":      requestOrigin,
                    "Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token,geolocation,x-retry",
                    "Access-Control-Allow-Methods":     "GET, PUT, DELETE, POST, OPTIONS, PATCH",
                    "Access-Control-Allow-Credentials": "true",
                },
            }, nil
        }
    }
    return events.APIGatewayProxyResponse{StatusCode: 400}, nil
}
```

### CORS in Response Handlers

Every response includes CORS headers:

```go
func SuccessResponse(statusCode int, data interface{}, logger *logrus.Logger) events.APIGatewayProxyResponse {
    return events.APIGatewayProxyResponse{
        StatusCode: statusCode,
        Body:       string(jsonData),
        Headers: map[string]string{
            "Content-Type":                 "application/json",
            "Access-Control-Allow-Origin":  "*",
            "Access-Control-Allow-Headers": "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
            "Access-Control-Allow-Methods": "GET,POST,PUT,DELETE,OPTIONS",
        },
    }
}
```

## Error Handling Patterns

### Standard Error Responses

The system uses consistent error response structures:

```go
// Error response format
{
    "error": true,
    "message": "Human-readable error message",
    "status": 400
}

// Validation error format
{
    "error": true,
    "message": "Validation failed",
    "status": 400,
    "validation": ["Field X is required", "Field Y is invalid"]
}
```

### Error Categories

1. **Authentication Errors (401)**
   ```go
   return api.ErrorResponse(http.StatusUnauthorized, "Authentication failed", logger), nil
   ```

2. **Authorization Errors (403)**
   ```go
   if !claims.IsSuperAdmin {
       return api.ErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can access", logger), nil
   }
   ```

3. **Validation Errors (400)**
   ```go
   if req.Title == "" {
       return api.ErrorResponse(http.StatusBadRequest, "Title is required", logger), nil
   }
   ```

4. **Not Found Errors (404)**
   ```go
   if err == sql.ErrNoRows {
       return api.ErrorResponse(http.StatusNotFound, "Resource not found", logger), nil
   }
   ```

5. **Internal Server Errors (500)**
   ```go
   return api.ErrorResponse(http.StatusInternalServerError, "Failed to process request", logger), nil
   ```

## HTTP Status Codes

### Success Codes
- **200 OK**: Successful GET/PUT/PATCH/DELETE operations
- **201 Created**: Successful POST operations

### Client Error Codes
- **400 Bad Request**: Invalid input, validation failures
- **401 Unauthorized**: Missing or invalid JWT token
- **403 Forbidden**: Valid token but insufficient permissions
- **404 Not Found**: Resource doesn't exist or belongs to different org
- **405 Method Not Allowed**: Unsupported HTTP method

### Server Error Codes
- **500 Internal Server Error**: Database errors, unexpected failures

## Logging with Logrus

### Configuration

```go
// Initialize structured logger
logger = logrus.New()
logger.SetFormatter(&logrus.JSONFormatter{PrettyPrint: isLocal})

// Set log level based on environment
util.SetLogLevel(logger, os.Getenv("LOG_LEVEL"))
// Supports: DEBUG, INFO, WARN, ERROR
// Production default: ERROR (to reduce CloudWatch costs)
```

### Logging Patterns

1. **Request Logging**
   ```go
   logger.WithFields(logrus.Fields{
       "operation": "LambdaHandler",
       "method":    request.HTTPMethod,
       "path":      request.Path,
       "resource":  request.Resource,
   }).Info("Request received")
   ```

2. **Error Logging**
   ```go
   logger.WithError(err).WithFields(logrus.Fields{
       "user_id":   claims.UserID,
       "org_id":    claims.OrgID,
       "operation": "CreateIssue",
   }).Error("Failed to create issue")
   ```

3. **Debug Logging**
   ```go
   if logger.IsLevelEnabled(logrus.DebugLevel) {
       logger.WithFields(logrus.Fields{
           "cognito_id": userProfile.CognitoID,
           "org_name":   userProfile.OrgName,
       }).Debug("Processing user profile")
   }
   ```

## Environment Variable Management via SSM

### Configuration Pattern

Every Lambda initializes SSM client and fetches parameters during cold start:

```go
func init() {
    // Initialize SSM client
    ssmClient := clients.NewSSMClient(isLocal)
    ssmRepository = &data.SSMDao{
        SSM:    ssmClient,
        Logger: logger,
    }

    // Fetch all parameters
    ssmParams, err = ssmRepository.GetParameters()
    if err != nil {
        logger.Fatal("Error getting SSM parameters")
    }

    // Access parameters
    dbEndpoint := ssmParams[constants.DATABASE_RDS_ENDPOINT]
    userPoolID := ssmParams[constants.COGNITO_USER_POOL_ID]
}
```

### SSM Parameter Paths

All parameters use consistent naming:

```go
const (
    ALLOWED_ORIGINS          = "/infrastructure/ALLOWED_ORIGINS"
    DATABASE_RDS_ENDPOINT    = "/infrastructure/DATABASE_RDS_ENDPOINT"
    DATABASE_PORT            = "/infrastructure/DATABASE_PORT"
    DATABASE_NAME            = "/infrastructure/DATABASE_NAME"
    DATABASE_USERNAME        = "/infrastructure/DATABASE_USERNAME"
    DATABASE_PASSWORD        = "/infrastructure/DATABASE_PASSWORD"
    SSL_MODE                 = "/infrastructure/SSL_MODE"
    COGNITO_USER_POOL_ID     = "/infrastructure/COGNITO_USER_POOL_ID"
    COGNITO_CLIENT_ID        = "/infrastructure/COGNITO_CLIENT_ID"
)
```

### Benefits

1. **Security**: No hardcoded credentials
2. **Environment Separation**: Different values per stage (dev/prod)
3. **Rotation**: Update credentials without code deployment
4. **Centralization**: Single source of truth for configuration

## Lambda Cold Start Optimization

### Global Variable Pattern

All Lambdas use global variables to reuse expensive resources across invocations:

```go
// Global variables initialized once during cold start
var (
    logger              *logrus.Logger
    isLocal             bool
    ssmRepository       data.SSMRepository
    ssmParams           map[string]string
    sqlDB               *sql.DB
    userRepository      data.UserRepository
    cognitoClient       *cognitoidentityprovider.Client
)

func init() {
    // Expensive initialization happens once
    // - Logger setup
    // - SSM parameter fetch
    // - Database connection pool
    // - AWS service clients
}

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    // Fast: reuses global variables
    claims, _ := auth.ExtractClaimsFromRequest(request)
    // Process request using initialized resources
}
```

### Optimization Benefits

1. **Database Connection Pooling**
   - Single `*sql.DB` instance reused across invocations
   - Connection pool managed by `database/sql` package
   - Dramatically reduces connection overhead

2. **AWS Client Reuse**
   - Cognito, SSM, S3 clients initialized once
   - Avoids SDK initialization overhead
   - Maintains connection pools

3. **Configuration Caching**
   - SSM parameters fetched once at cold start
   - No repeated parameter store calls
   - Reduces latency and costs

4. **Logger Initialization**
   - Formatter and level set once
   - Consistent logging across invocations

### Cold Start vs Warm Invocation

```
Cold Start (first invocation):
- init() executes (2-5 seconds)
  - SSM parameter fetch
  - Database connection
  - AWS client initialization
- Handler executes

Warm Invocation (subsequent calls):
- init() skipped
- Handler executes immediately (50-200ms)
```

## Deployment with CDK

### Stack Organization

```
MultiApiMainStack (main-stack)
  └─ MultiApiSubStack (nested-stack)
      ├─ LambdaConstruct (creates all 14 Lambdas)
      ├─ CognitoConstruct (User Pool + Authorizers)
      └─ API Gateway Resources
          ├─ IAM API
          ├─ Projects API
          ├─ Issues API
          └─ RFIs API
```

### Lambda Creation Pattern

Each Lambda is created with:

```typescript
const lambda = new Function(this, 'UserManagementLambda', {
    runtime: Runtime.GO_1_X,
    handler: 'main',
    code: Code.fromAsset('path/to/compiled/binary'),
    environment: {
        IS_LOCAL: 'false',
        LOG_LEVEL: 'ERROR'
    },
    timeout: Duration.seconds(30),
    memorySize: 512,
    vpc: vpc,  // VPC for database access
});
```

### API Gateway Integration

```typescript
// Create authorizer
const cognitoAuthorizer = new CognitoUserPoolsAuthorizer(this, 'Authorizer', {
    cognitoUserPools: [userPool],
    authorizerName: 'CognitoAuthorizer'
});

// Create Lambda integration
const lambdaIntegration = new LambdaIntegration(lambda);

// Define routes
const resource = api.root.addResource('users');
resource.addMethod('GET', lambdaIntegration, { authorizer: cognitoAuthorizer });
resource.addMethod('POST', lambdaIntegration, { authorizer: cognitoAuthorizer });
resource.addMethod('OPTIONS', corsIntegration);  // No authorizer for OPTIONS
```

### Deployment Commands

```bash
# Build Go binaries
npm run build

# Deploy to dev
npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev

# Deploy specific API
npx cdk deploy IamApiStack --profile dev
```

## Performance Characteristics

### Typical Response Times

- **Cold Start**: 2-5 seconds (first invocation)
- **Warm Invocation**: 50-200ms
- **Simple CRUD**: 100-300ms
- **Complex Queries**: 300-800ms
- **File Upload (Presigned URL)**: 100-200ms

### Optimization Strategies

1. **Global Variables**: Reuse connections and clients
2. **Connection Pooling**: Database connection reuse
3. **Parallel Queries**: Use goroutines for independent operations
4. **Prepared Statements**: SQL query caching
5. **Index Optimization**: Database indexes on frequently queried columns
6. **Pagination**: Limit result sets for list operations

## Security Considerations

### Data Isolation

1. **Organization-Level Isolation**
   - Every query validates `org_id` from JWT
   - No cross-organization data access
   - Repository layer enforces isolation

2. **Project-Level Isolation**
   - Validate project belongs to user's organization
   - Check user has access to specific project

3. **Super Admin Restrictions**
   - Certain operations require `isSuperAdmin` flag
   - Examples: user creation, role management

### Input Validation

1. **Request Body Parsing**
   ```go
   var req models.CreateRequest
   if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
       return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
   }
   ```

2. **Required Field Validation**
   ```go
   if req.Title == "" {
       return api.ErrorResponse(http.StatusBadRequest, "Title is required", logger)
   }
   ```

3. **Type Validation**
   ```go
   userID, err := strconv.ParseInt(request.PathParameters["userId"], 10, 64)
   if err != nil {
       return api.ErrorResponse(http.StatusBadRequest, "Invalid user ID", logger)
   }
   ```

4. **Organization Ownership Validation**
   ```go
   // Always validate entity belongs to user's org
   var entityOrgID int64
   err := db.QueryRow("SELECT org_id FROM entities WHERE id = $1", entityID).Scan(&entityOrgID)
   if entityOrgID != claims.OrgID {
       return api.ErrorResponse(http.StatusForbidden, "Access denied", logger)
   }
   ```

## Best Practices

### Repository Methods

1. Always accept `context.Context` as first parameter
2. Always include `orgID` for multi-tenant isolation
3. Return structured response models, not database models
4. Use transactions for multi-step operations
5. Log with structured fields for debugging

### Handler Functions

1. Extract claims first, validate authentication
2. Check authorization before processing
3. Parse and validate input
4. Call single repository method
5. Return structured response

### Error Handling

1. Never expose database errors to clients
2. Log detailed errors with context
3. Return user-friendly error messages
4. Always return proper HTTP status codes
5. Use correlation IDs for request tracking

### Testing

1. Test with valid JWT tokens (ID tokens, not access tokens)
2. Test organization isolation
3. Test validation errors
4. Test authorization failures
5. Test database error scenarios

## Related Documentation

- [Authentication & Authorization Architecture](./authentication-authorization.md)
- [Database Schema Guide](../sql/)
- [API Quick Start Guide](../QUICK-START.md)