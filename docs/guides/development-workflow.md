# Development Workflow Guide

## Table of Contents

- [Setting Up Development Environment](#setting-up-development-environment)
- [Making Code Changes](#making-code-changes)
- [Building and Deploying](#building-and-deploying)
- [Testing Locally vs Deployed](#testing-locally-vs-deployed)
- [Adding New Endpoints](#adding-new-endpoints)
- [Modifying Database Schema](#modifying-database-schema)
- [Updating Access Control](#updating-access-control)
- [Common Development Patterns](#common-development-patterns)
- [Git Workflow](#git-workflow)
- [CDK Development Tips](#cdk-development-tips)
- [Troubleshooting](#troubleshooting)

---

## Setting Up Development Environment

### Prerequisites

Install required tools:

```bash
# Node.js (v18 or later)
node --version

# Go (v1.21 or later)
go version

# AWS CDK CLI (v2.x)
npm install -g aws-cdk
cdk --version

# AWS CLI with profiles configured
aws --version
aws configure list-profiles
```

### AWS Profile Configuration

Configure Dev and Prod profiles:

```bash
# Configure dev profile
aws configure --profile dev
# AWS Access Key ID: [your-dev-key]
# AWS Secret Access Key: [your-dev-secret]
# Default region: us-east-2
# Default output format: json

# Configure prod profile
aws configure --profile prod
# AWS Access Key ID: [your-prod-key]
# AWS Secret Access Key: [your-prod-secret]
# Default region: us-east-2
# Default output format: json

# Verify profiles
aws sts get-caller-identity --profile dev
aws sts get-caller-identity --profile prod
```

### Repository Setup

Clone and set up the project:

```bash
# Clone infrastructure repository
cd /Users/mayur/git_personal/
git clone <infrastructure-repo-url> infrastructure
cd infrastructure

# Install Node.js dependencies
npm install

# Initialize Go modules (already done, but for reference)
cd src
go mod download
cd ..

# Verify build works
npm run build
```

### Environment Variables

Create `.env` file (for local testing only, never commit):

```bash
# .env (in project root)
LOG_LEVEL=DEBUG
IS_LOCAL=true
```

### IDE Setup

**VS Code Recommended Extensions:**
- Go (golang.go)
- AWS Toolkit
- TypeScript and JavaScript
- PostgreSQL (cweijan.vscode-postgresql-client2)
- REST Client (humao.rest-client)

**VS Code Settings:**
```json
{
  "go.useLanguageServer": true,
  "go.formatTool": "gofmt",
  "go.lintTool": "golangci-lint",
  "typescript.preferences.importModuleSpecifier": "relative",
  "editor.formatOnSave": true
}
```

---

## Making Code Changes

### Understanding the Project Structure

```
infrastructure/
├── src/                                    # Go Lambda functions (Backend)
│   ├── infrastructure-{service}/main.go   # Lambda entry points
│   └── lib/                               # Shared libraries
│       ├── api/                           # API utilities
│       ├── auth/                          # Authentication
│       ├── clients/                       # AWS clients (S3, SSM, RDS)
│       ├── data/                          # Repository layer
│       ├── models/                        # Data models
│       └── validators/                    # Input validation
├── lib/                                    # CDK Infrastructure (TypeScript)
│   ├── resources/                         # AWS resource constructs
│   └── infrastructure-stack.ts            # Main stack
└── docs/                                   # Documentation
```

### Modifying Lambda Functions

#### 1. Edit Lambda Handler

**Example: Add new endpoint to Project Management**

```bash
# Open Lambda handler
code src/infrastructure-project-management/main.go
```

**Add handler logic:**

```go
// In handleRequest() function
switch request.Resource {
case "/projects":
    if request.HTTPMethod == "GET" {
        return handleGetProjects(ctx, request, claims, projectRepo, assignmentRepo, logger)
    } else if request.HTTPMethod == "POST" {
        return handleCreateProject(ctx, request, claims, projectRepo, logger)
    }

// Add new endpoint
case "/projects/{projectId}/summary":
    if request.HTTPMethod == "GET" {
        return handleGetProjectSummary(ctx, request, claims, projectRepo, logger)
    }

default:
    return api.ErrorResponse(http.StatusNotFound, "Resource not found", logger), nil
}
```

#### 2. Add Repository Method

**Create method in repository:**

```bash
# Edit repository
code src/lib/data/project_repository.go
```

```go
// Add new method to ProjectRepository interface
type ProjectRepository interface {
    // ... existing methods ...
    GetProjectSummary(ctx context.Context, projectID int64, orgID int64) (*ProjectSummary, error)
}

// Implement method
func (repo *projectRepositoryImpl) GetProjectSummary(ctx context.Context, projectID int64, orgID int64) (*ProjectSummary, error) {
    logger := repo.Logger

    query := `
        SELECT
            p.id, p.project_number, p.name,
            COUNT(DISTINCT i.id) as issue_count,
            COUNT(DISTINCT r.id) as rfi_count
        FROM project.projects p
        LEFT JOIN project.issues i ON i.project_id = p.id AND i.is_deleted = false
        LEFT JOIN project.rfis r ON r.project_id = p.id AND r.is_deleted = false
        WHERE p.id = $1 AND p.org_id = $2 AND p.is_deleted = false
        GROUP BY p.id
    `

    var summary ProjectSummary
    err := repo.DB.QueryRowContext(ctx, query, projectID, orgID).Scan(
        &summary.ID, &summary.ProjectNumber, &summary.Name,
        &summary.IssueCount, &summary.RFICount,
    )

    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("project not found")
    }
    if err != nil {
        logger.Error("Failed to fetch project summary", zap.Error(err))
        return nil, err
    }

    return &summary, nil
}
```

#### 3. Add Model Struct

```bash
# Edit models
code src/lib/models/project.go
```

```go
// Add new model struct
type ProjectSummary struct {
    ID            int64  `json:"id"`
    ProjectNumber string `json:"project_number"`
    Name          string `json:"name"`
    IssueCount    int    `json:"issue_count"`
    RFICount      int    `json:"rfi_count"`
}
```

#### 4. Test Go Code Compiles

```bash
# Compile specific Lambda
cd src/infrastructure-project-management
go build .

# Or compile all Lambda functions
cd /Users/mayur/git_personal/infrastructure
npm run build
```

### Modifying CDK Infrastructure

#### 1. Add New Lambda Function

**Create Lambda directory:**

```bash
mkdir src/infrastructure-new-service
touch src/infrastructure-new-service/main.go
```

**Write Lambda code:**

```go
package main

import (
    "context"
    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "infrastructure/lib/api"
    "infrastructure/lib/auth"
)

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    logger := api.NewLogger("INFO")

    // Extract claims
    claims, err := auth.ExtractClaimsFromRequest(request)
    if err != nil {
        return api.ErrorResponse(http.StatusUnauthorized, "Unauthorized", logger), nil
    }

    // Handle request
    return api.SuccessResponse(map[string]interface{}{
        "message": "Hello from new service",
        "user_id": claims.UserID,
    }, logger), nil
}

func main() {
    lambda.Start(Handler)
}
```

#### 2. Add CDK Construct

**Create Lambda construct:**

```bash
code lib/resources/function_construct/new-service.ts
```

```typescript
import { Construct } from 'constructs';
import { GoFunction } from '@aws-cdk/aws-lambda-go-alpha';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import { Duration } from 'aws-cdk-lib';

export interface NewServiceProps {
  environment?: { [key: string]: string };
}

export class NewServiceFunction extends Construct {
  public readonly function: lambda.Function;

  constructor(scope: Construct, id: string, props?: NewServiceProps) {
    super(scope, id);

    this.function = new GoFunction(this, 'Function', {
      entry: 'src/infrastructure-new-service',
      timeout: Duration.seconds(30),
      memorySize: 512,
      environment: {
        LOG_LEVEL: 'INFO',
        ...props?.environment,
      },
    });
  }
}
```

#### 3. Wire Up API Gateway Routes

**Add routes to API Gateway:**

```bash
code lib/resources/sub_stack/sub-stack.ts
```

```typescript
// Import construct
import { NewServiceFunction } from '../function_construct/new-service';

// In constructor, create function
const newServiceFunction = new NewServiceFunction(this, 'NewService', {
  environment: {
    LOG_LEVEL: 'INFO',
  },
});

// Grant permissions
newServiceFunction.function.addToRolePolicy(/* IAM policies */);

// Add API routes
const newServiceIntegration = new apigateway.LambdaIntegration(newServiceFunction.function);

const newServiceResource = apiRoot.addResource('new-service');
newServiceResource.addMethod('GET', newServiceIntegration, {
  authorizer: cognitoAuthorizer,
  authorizationType: apigateway.AuthorizationType.COGNITO,
});
```

---

## Building and Deploying

### Build Process

The build process compiles TypeScript CDK code and Go Lambda code:

```bash
# Full build (TypeScript + Go)
npm run build

# This runs: cdk synth
# - Compiles TypeScript CDK infrastructure
# - Compiles all Go Lambda functions
# - Generates CloudFormation templates
```

**What happens during build:**
1. TypeScript files in `/lib` are compiled
2. Go Lambda functions in `/src` are compiled for Linux AMD64
3. CloudFormation templates are generated in `cdk.out/`
4. Lambda assets are bundled

### Deploy to Development

**Standard deployment:**

```bash
# Build first
npm run build

# Deploy to dev (from parent directory due to CDK quirk)
cd /Users/mayur/git_personal
npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev

# Return to infrastructure directory
cd infrastructure
```

**Quick deploy without approval:**

```bash
cd /Users/mayur/git_personal
npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev --require-approval never
cd infrastructure
```

**Watch mode (hot reload):**

```bash
# Auto-deploy on file changes
npm run watch
```

### Deploy to Production

**Production requires explicit approval:**

```bash
npm run build
cd /Users/mayur/git_personal
npx cdk deploy "Infrastructure/Prod/Infrastructure-AppStage" --profile prod
cd infrastructure
```

### Deployment Checklist

Before deploying:

- [ ] Code compiles without errors (`npm run build`)
- [ ] Go tests pass (if any)
- [ ] No sensitive data in code (API keys, passwords)
- [ ] Environment-specific values use environment variables
- [ ] Lambda timeout appropriate for endpoint
- [ ] IAM permissions added to Lambda role
- [ ] API Gateway routes configured correctly
- [ ] Changes tested locally or in dev first

---

## Testing Locally vs Deployed

### Testing Deployed APIs

**Get authentication token:**

```bash
# Get ID token
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

**Test endpoint:**

```bash
# Set API base URL
API_BASE="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"

# Test GET endpoint
curl -X GET "$API_BASE/projects?location_id=6" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .

# Test POST endpoint
curl -X POST "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "location_id": 6,
    "basic_info": {
      "name": "Test Project",
      "description": "Testing from CLI"
    }
  }' | jq .
```

**Use test scripts:**

```bash
cd /Users/mayur/git_personal/infrastructure/testing/api
./test-project-user-management.sh
./test-get-projects-access-control.sh
./test-issue-comments.sh
```

### Testing with Postman

**Import collections:**

1. Open Postman
2. Import collection from `/Users/mayur/git_personal/infrastructure/postman/ProjectManagement.postman_collection.json`
3. Set up environment variables:
   - `access_token`: ID token from Cognito
   - `base_url`: `https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main`
   - `project_id`: A valid project ID
   - `location_id`: A valid location ID

**Get token in Postman:**

Use the Infrastructure collection's "Get ID Token" request or run curl command and paste token.

### Local Testing (SAM Local)

**Not commonly used, but available:**

```bash
# Build for local testing
npm run local-build

# Start local API
npm run api

# API available at: http://localhost:3000
```

**Note:** Local testing requires local database connection or mocking.

### Viewing Lambda Logs

**CloudWatch Logs:**

```bash
# Tail logs in real-time
aws logs tail /aws/lambda/infrastructure-project-management \
  --since 1h \
  --follow \
  --profile dev \
  --region us-east-2

# View recent logs
aws logs tail /aws/lambda/infrastructure-project-management \
  --since 30m \
  --profile dev \
  --region us-east-2
```

**In AWS Console:**
1. Go to CloudWatch > Log groups
2. Find `/aws/lambda/infrastructure-{service-name}`
3. View log streams

---

## Adding New Endpoints

### Step-by-Step Process

#### 1. Plan Endpoint Design

**Decide on:**
- HTTP method (GET, POST, PUT, DELETE)
- Resource path (e.g., `/projects/{projectId}/summary`)
- Request parameters/body
- Response structure
- Access control requirements

#### 2. Add Route Handler

**In Lambda main.go:**

```go
func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest, /* deps */) {
    // Extract claims
    claims, err := auth.ExtractClaimsFromRequest(request)
    if err != nil {
        return api.ErrorResponse(http.StatusUnauthorized, "Unauthorized", logger), nil
    }

    // Route to handler
    switch request.Resource {
    case "/your-new-endpoint":
        if request.HTTPMethod == "GET" {
            return handleGetYourEndpoint(ctx, request, claims, repo, logger)
        }
    }
}

func handleGetYourEndpoint(ctx context.Context, request events.APIGatewayProxyRequest,
    claims *auth.Claims, repo Repository, logger *zap.Logger) (events.APIGatewayProxyResponse, error) {

    // Parse parameters
    param := request.QueryStringParameters["param"]

    // Check access control
    if claims.OrgID != expectedOrgID {
        return api.ErrorResponse(http.StatusForbidden, "Access denied", logger), nil
    }

    // Call repository
    result, err := repo.GetSomething(ctx, param, claims.OrgID)
    if err != nil {
        return api.ErrorResponse(http.StatusInternalServerError, "Failed to fetch data", logger), nil
    }

    // Return response
    return api.SuccessResponse(result, logger), nil
}
```

#### 3. Add Repository Method

**In repository file:**

```go
// Add to interface
type Repository interface {
    GetSomething(ctx context.Context, param string, orgID int64) (*Result, error)
}

// Implement
func (repo *repositoryImpl) GetSomething(ctx context.Context, param string, orgID int64) (*Result, error) {
    query := `SELECT ... FROM table WHERE param = $1 AND org_id = $2`
    var result Result
    err := repo.DB.QueryRowContext(ctx, query, param, orgID).Scan(&result.Field1, &result.Field2)
    if err != nil {
        return nil, err
    }
    return &result, nil
}
```

#### 4. Add to CDK Infrastructure

**In sub-stack.ts:**

```typescript
// Add route to API Gateway
const yourResource = apiRoot.addResource('your-endpoint');
yourResource.addMethod('GET', lambdaIntegration, {
  authorizer: cognitoAuthorizer,
  authorizationType: apigateway.AuthorizationType.COGNITO,
});
```

#### 5. Create Postman Request

**Add to collection:**

```json
{
  "name": "Get Your Endpoint",
  "request": {
    "method": "GET",
    "header": [
      {
        "key": "Authorization",
        "value": "Bearer {{access_token}}"
      }
    ],
    "url": {
      "raw": "{{base_url}}/your-endpoint?param=value",
      "host": ["{{base_url}}"],
      "path": ["your-endpoint"],
      "query": [
        {
          "key": "param",
          "value": "value"
        }
      ]
    }
  }
}
```

#### 6. Test Endpoint

```bash
# Build and deploy
npm run build
cd .. && npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev && cd infrastructure

# Test with curl
curl -X GET "$API_BASE/your-endpoint?param=value" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .
```

---

## Modifying Database Schema

### Using MCP for Schema Queries

**Check existing schema:**

Ask MCP:
- "Show me the schema of the projects table"
- "What columns are in the user_assignments table?"
- "List all tables in the project schema"

### Adding New Column

**1. Write SQL migration:**

```sql
-- Add column to existing table
ALTER TABLE project.projects
ADD COLUMN new_field VARCHAR(255);

-- With default value
ALTER TABLE project.projects
ADD COLUMN new_field VARCHAR(255) DEFAULT 'default_value';

-- With NOT NULL constraint
ALTER TABLE project.projects
ADD COLUMN new_field VARCHAR(255) NOT NULL DEFAULT 'default_value';
```

**2. Execute via database client or MCP:**

```bash
# Via psql (if you have access)
psql -h appdb.cdwmaay8wkw4.us-east-2.rds.amazonaws.com \
     -U appdb_admin \
     -d appdb \
     -c "ALTER TABLE project.projects ADD COLUMN new_field VARCHAR(255);"
```

**3. Update Go model:**

```go
// In models/project.go
type Project struct {
    // ... existing fields ...
    NewField string `json:"new_field" db:"new_field"`
}
```

**4. Update repository queries:**

```go
// Add to SELECT query
query := `
    SELECT
        id, project_number, name, new_field
    FROM project.projects
    WHERE id = $1
`

// Add to INSERT query
query := `
    INSERT INTO project.projects
    (name, new_field, org_id, created_by)
    VALUES ($1, $2, $3, $4)
    RETURNING id
`
```

### Creating New Table

**1. Write SQL:**

```sql
CREATE TABLE project.new_table (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL REFERENCES iam.organizations(id),
    project_id BIGINT REFERENCES project.projects(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'active',
    is_deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT,
    updated_by BIGINT
);

-- Create indexes
CREATE INDEX idx_new_table_org_id ON project.new_table(org_id);
CREATE INDEX idx_new_table_project_id ON project.new_table(project_id);
CREATE INDEX idx_new_table_is_deleted ON project.new_table(is_deleted);
```

**2. Create Go model:**

```go
// In models/new_entity.go
package models

import "time"

type NewEntity struct {
    ID          int64     `json:"id" db:"id"`
    OrgID       int64     `json:"org_id" db:"org_id"`
    ProjectID   *int64    `json:"project_id,omitempty" db:"project_id"`
    Name        string    `json:"name" db:"name"`
    Description string    `json:"description" db:"description"`
    Status      string    `json:"status" db:"status"`
    IsDeleted   bool      `json:"is_deleted" db:"is_deleted"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
    CreatedBy   int64     `json:"created_by" db:"created_by"`
    UpdatedBy   int64     `json:"updated_by" db:"updated_by"`
}
```

**3. Create repository:**

```go
// In data/new_entity_repository.go
package data

import (
    "context"
    "database/sql"
    "infrastructure/lib/models"
)

type NewEntityRepository interface {
    Create(ctx context.Context, entity *models.NewEntity) (*models.NewEntity, error)
    GetByID(ctx context.Context, id int64, orgID int64) (*models.NewEntity, error)
    List(ctx context.Context, orgID int64) ([]*models.NewEntity, error)
    Update(ctx context.Context, entity *models.NewEntity) (*models.NewEntity, error)
    Delete(ctx context.Context, id int64, orgID int64) error
}

type newEntityRepositoryImpl struct {
    DB *sql.DB
}

func NewNewEntityRepository(db *sql.DB) NewEntityRepository {
    return &newEntityRepositoryImpl{DB: db}
}

// Implement methods...
```

---

## Updating Access Control

### Understanding Access Levels

1. **Super Admin** (`is_super_admin = true`)
   - Access to everything in organization
   - No additional checks needed

2. **Organization-Level** (`context_type = 'organization'`)
   - Access to all locations and projects in org
   - Check: user has org-level assignment

3. **Location-Level** (`context_type = 'location'`)
   - Access to all projects at assigned locations
   - Check: user has location-level assignment for this location

4. **Project-Level** (`context_type = 'project'`)
   - Access only to assigned projects
   - Check: user has project-level assignment for this project

### Implementing Access Control

**Pattern 1: Simple org-level check:**

```go
// Only same organization can access
if claims.OrgID != resource.OrgID {
    return api.ErrorResponse(http.StatusForbidden, "Access denied", logger), nil
}
```

**Pattern 2: Super admin bypass:**

```go
// Super admins can access, others need specific permission
if !claims.IsSuperAdmin {
    // Additional checks
    if claims.OrgID != resource.OrgID {
        return api.ErrorResponse(http.StatusForbidden, "Access denied", logger), nil
    }
}
```

**Pattern 3: Assignment-based access:**

```go
// Check if user has access to project
if !claims.IsSuperAdmin {
    // Get user's project contexts
    projectContexts, err := assignmentRepo.GetUserContexts(ctx, claims.UserID, "project", claims.OrgID)
    if err != nil {
        return api.ErrorResponse(http.StatusInternalServerError, "Failed to check access", logger), nil
    }

    // Check if user has access to this specific project
    hasAccess := false
    for _, contextID := range projectContexts {
        if contextID == projectID {
            hasAccess = true
            break
        }
    }

    if !hasAccess {
        // Check location-level access
        locationContexts, _ := assignmentRepo.GetUserContexts(ctx, claims.UserID, "location", claims.OrgID)
        for _, locationID := range locationContexts {
            if locationID == project.LocationID {
                hasAccess = true
                break
            }
        }
    }

    if !hasAccess {
        // Check org-level access
        orgContexts, _ := assignmentRepo.GetUserContexts(ctx, claims.UserID, "organization", claims.OrgID)
        if len(orgContexts) > 0 {
            hasAccess = true
        }
    }

    if !hasAccess {
        return api.ErrorResponse(http.StatusForbidden, "Access denied", logger), nil
    }
}
```

**Pattern 4: GET /projects access control (recommended pattern):**

```go
func handleGetProjects(ctx context.Context, request events.APIGatewayProxyRequest,
    claims *auth.Claims, projectRepo data.ProjectRepository,
    assignmentRepo data.AssignmentRepository, logger *zap.Logger) (events.APIGatewayProxyResponse, error) {

    locationID := request.QueryStringParameters["location_id"]

    var projects []models.Project
    var err error

    if claims.IsSuperAdmin {
        // Super admin sees all projects
        if locationID != "" {
            projects, err = projectRepo.GetProjectsByLocationID(ctx, locationID, claims.OrgID)
        } else {
            projects, err = projectRepo.GetProjectsByOrg(ctx, claims.OrgID)
        }
    } else {
        // Check org-level assignment
        orgContexts, _ := assignmentRepo.GetUserContexts(ctx, claims.UserID, "organization", claims.OrgID)
        if len(orgContexts) > 0 {
            // Org-level user sees all projects
            if locationID != "" {
                projects, err = projectRepo.GetProjectsByLocationID(ctx, locationID, claims.OrgID)
            } else {
                projects, err = projectRepo.GetProjectsByOrg(ctx, claims.OrgID)
            }
        } else {
            // Location or project-level user
            locationContexts, _ := assignmentRepo.GetUserContexts(ctx, claims.UserID, "location", claims.OrgID)
            projectContexts, _ := assignmentRepo.GetUserContexts(ctx, claims.UserID, "project", claims.OrgID)

            if len(locationContexts) > 0 {
                // Location-level user
                if locationID != "" {
                    // Verify user has access to this location
                    hasAccess := false
                    for _, locID := range locationContexts {
                        if strconv.FormatInt(locID, 10) == locationID {
                            hasAccess = true
                            break
                        }
                    }
                    if !hasAccess {
                        return api.ErrorResponse(http.StatusForbidden, "No access to this location", logger), nil
                    }
                    projects, err = projectRepo.GetProjectsByLocationID(ctx, locationID, claims.OrgID)
                } else {
                    return api.ErrorResponse(http.StatusBadRequest, "location_id required", logger), nil
                }
            } else if len(projectContexts) > 0 {
                // Project-level user
                projects, err = projectRepo.GetProjectsByIDs(ctx, projectContexts, claims.OrgID)
                if locationID != "" {
                    // Filter to location
                    filtered := []models.Project{}
                    for _, p := range projects {
                        if strconv.FormatInt(p.LocationID, 10) == locationID {
                            filtered = append(filtered, p)
                        }
                    }
                    projects = filtered
                }
            } else {
                // No access
                projects = []models.Project{}
            }
        }
    }

    if err != nil {
        return api.ErrorResponse(http.StatusInternalServerError, "Failed to fetch projects", logger), nil
    }

    return api.SuccessResponse(map[string]interface{}{
        "projects": projects,
        "count":    len(projects),
    }, logger), nil
}
```

### Testing Access Control

**Create test users with different access levels:**

```sql
-- User with org-level access
INSERT INTO iam.user_assignments (user_id, role_id, context_type, context_id, org_id, created_by)
VALUES (20, 1, 'organization', 10, 10, 1);

-- User with location-level access
INSERT INTO iam.user_assignments (user_id, role_id, context_type, context_id, org_id, created_by)
VALUES (21, 2, 'location', 6, 10, 1);

-- User with project-level access
INSERT INTO iam.user_assignments (user_id, role_id, context_type, context_id, org_id, created_by)
VALUES (22, 3, 'project', 29, 10, 1);
```

**Test each level:**

```bash
# Test as org-level user
TOKEN_ORG=$(# get token for user 20)
curl "$API_BASE/projects" -H "Authorization: Bearer $TOKEN_ORG"

# Test as location-level user
TOKEN_LOC=$(# get token for user 21)
curl "$API_BASE/projects?location_id=6" -H "Authorization: Bearer $TOKEN_LOC"

# Test as project-level user
TOKEN_PROJ=$(# get token for user 22)
curl "$API_BASE/projects" -H "Authorization: Bearer $TOKEN_PROJ"
```

---

## Common Development Patterns

### Repository Pattern

**Always use repositories for database access:**

```go
// DON'T: Direct database queries in handler
func handleGetProject(/* ... */) {
    db.Query("SELECT * FROM projects WHERE id = $1", projectID)
}

// DO: Use repository
func handleGetProject(/* ... */, projectRepo data.ProjectRepository) {
    project, err := projectRepo.GetByID(ctx, projectID, claims.OrgID)
}
```

### Error Handling

**Consistent error responses:**

```go
// Not found
if err == sql.ErrNoRows {
    return api.ErrorResponse(http.StatusNotFound, "Project not found", logger), nil
}

// Validation error
if validationErr != nil {
    return api.ErrorResponse(http.StatusBadRequest, validationErr.Error(), logger), nil
}

// Access denied
if !hasAccess {
    return api.ErrorResponse(http.StatusForbidden, "Access denied", logger), nil
}

// Internal error
if err != nil {
    logger.Error("Failed to fetch project", zap.Error(err))
    return api.ErrorResponse(http.StatusInternalServerError, "Internal server error", logger), nil
}
```

### Input Validation

**Validate all inputs:**

```go
// Required fields
if request.Name == "" {
    return api.ErrorResponse(http.StatusBadRequest, "name is required", logger), nil
}

// Format validation
if !isValidEmail(request.Email) {
    return api.ErrorResponse(http.StatusBadRequest, "invalid email format", logger), nil
}

// Range validation
if request.Priority < 1 || request.Priority > 5 {
    return api.ErrorResponse(http.StatusBadRequest, "priority must be between 1 and 5", logger), nil
}
```

### Soft Deletes

**Always use soft deletes:**

```go
// DON'T: Hard delete
db.Exec("DELETE FROM projects WHERE id = $1", projectID)

// DO: Soft delete
db.Exec("UPDATE projects SET is_deleted = true, updated_by = $1 WHERE id = $2", userID, projectID)
```

### Auto-Generated Numbers

**Use sequences for auto-numbering:**

```go
// For projects: PROJ-YYYY-NNNN
func generateProjectNumber(ctx context.Context, db *sql.DB) (string, error) {
    year := time.Now().Year()
    var nextNum int

    query := `
        SELECT COALESCE(MAX(CAST(SUBSTRING(project_number FROM 11) AS INTEGER)), 0) + 1
        FROM project.projects
        WHERE project_number LIKE $1
    `

    err := db.QueryRowContext(ctx, query, fmt.Sprintf("PROJ-%d-%%", year)).Scan(&nextNum)
    if err != nil {
        return "", err
    }

    return fmt.Sprintf("PROJ-%d-%04d", year, nextNum), nil
}
```

### Pagination

**Support pagination on list endpoints:**

```go
// Parse pagination params
page := 1
limit := 20

if pageStr := request.QueryStringParameters["page"]; pageStr != "" {
    page, _ = strconv.Atoi(pageStr)
}
if limitStr := request.QueryStringParameters["limit"]; limitStr != "" {
    limit, _ = strconv.Atoi(limitStr)
}

// Apply to query
offset := (page - 1) * limit
query := `SELECT * FROM table WHERE ... LIMIT $1 OFFSET $2`
rows, err := db.QueryContext(ctx, query, limit, offset)

// Return pagination info
return api.SuccessResponse(map[string]interface{}{
    "data": results,
    "pagination": map[string]interface{}{
        "page":  page,
        "limit": limit,
        "total": totalCount,
    },
}, logger), nil
```

---

## Git Workflow

### Branch Strategy

**Main branches:**
- `main` - Production-ready code
- `dev` - Development branch (if used)
- Feature branches - Individual features

**Feature branch workflow:**

```bash
# Create feature branch from main
git checkout main
git pull origin main
git checkout -b feature/add-project-summary-endpoint

# Make changes and commit
git add .
git commit -m "Add project summary endpoint

- Add GET /projects/{projectId}/summary route
- Implement repository method for summary data
- Add tests for summary endpoint"

# Push feature branch
git push origin feature/add-project-summary-endpoint

# Create pull request on GitHub
```

### Commit Message Guidelines

**Format:**

```
<type>: <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code refactoring
- `docs`: Documentation changes
- `test`: Test additions/changes
- `chore`: Build/tooling changes

**Example:**

```
feat: Add project summary endpoint

- Add GET /projects/{projectId}/summary route
- Implement ProjectRepository.GetProjectSummary method
- Return issue count, RFI count, submittal count
- Add access control checks

Closes #123
```

### Pre-Commit Checklist

Before committing:

- [ ] Code compiles: `npm run build`
- [ ] No console.log or debug statements
- [ ] No commented-out code (remove or explain)
- [ ] No sensitive data (API keys, passwords)
- [ ] Updated relevant documentation
- [ ] Added tests (if applicable)

### Deploying After Merge

```bash
# After PR merged to main
git checkout main
git pull origin main

# Build
npm run build

# Deploy to dev for verification
cd .. && npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev && cd infrastructure

# Verify in dev
# Test endpoints
# Check logs

# Deploy to prod
cd .. && npx cdk deploy "Infrastructure/Prod/Infrastructure-AppStage" --profile prod && cd infrastructure
```

---

## CDK Development Tips

### CDK Best Practices

**1. Use constructs for reusability:**

```typescript
// Create reusable constructs
export class DatabaseConstruct extends Construct {
  constructor(scope: Construct, id: string, props: DatabaseProps) {
    super(scope, id);
    // Database setup
  }
}
```

**2. Environment-specific configuration:**

```typescript
// Use environment variables
const isProduction = process.env.ENV === 'production';

const lambdaConfig = {
  timeout: isProduction ? Duration.seconds(30) : Duration.seconds(60),
  memorySize: isProduction ? 1024 : 512,
};
```

**3. Tag resources:**

```typescript
Tags.of(this).add('Environment', props.environment);
Tags.of(this).add('Project', 'BuildBoard');
Tags.of(this).add('ManagedBy', 'CDK');
```

### CDK Commands

```bash
# List all stacks
npx cdk list

# Synthesize CloudFormation template
npx cdk synth

# Show diff before deployment
npx cdk diff "Infrastructure/Dev/Infrastructure-AppStage" --profile dev

# Deploy specific stack
npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev

# Destroy stack (careful!)
npx cdk destroy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev

# Watch mode (hot reload)
npm run watch
```

### Lambda Configuration Tips

**Timeout guidelines:**
- Simple CRUD: 10-15 seconds
- Complex queries: 20-30 seconds
- File processing: 60-120 seconds

**Memory guidelines:**
- Light operations: 512 MB
- Standard CRUD: 512-1024 MB
- Heavy processing: 1024-2048 MB

**Environment variables pattern:**

```typescript
const lambda = new GoFunction(this, 'Lambda', {
  entry: 'src/infrastructure-service',
  timeout: Duration.seconds(30),
  memorySize: 512,
  environment: {
    LOG_LEVEL: props.logLevel || 'INFO',
    DB_SECRET_NAME: dbSecret.secretName,
    // Don't hardcode sensitive values
  },
});

// Grant permissions
dbSecret.grantRead(lambda);
```

### Troubleshooting CDK

**Common issues:**

**1. Asset hash changes constantly:**
```bash
# Clear CDK cache
rm -rf cdk.out
npm run build
```

**2. Lambda not updating:**
```bash
# Force new deployment
# Change environment variable or description in CDK
```

**3. CloudFormation stack stuck:**
```bash
# Cancel update
aws cloudformation cancel-update-stack \
  --stack-name "Dev-Infrastructure-AppStage" \
  --profile dev \
  --region us-east-2

# Continue rollback if failed
aws cloudformation continue-update-rollback \
  --stack-name "Dev-Infrastructure-AppStage" \
  --profile dev \
  --region us-east-2
```

---

## Troubleshooting

### Build Errors

**Go compilation errors:**

```bash
# Error: undefined: SomeFunction
# Fix: Check imports and function names

# Error: cannot find package
cd src
go mod tidy
go mod download
cd ..
```

**TypeScript errors:**

```bash
# Error: Cannot find module
npm install

# Error: Type errors
# Fix type definitions in *.ts files
```

### Deployment Errors

**Permission denied:**

```bash
# Error: User is not authorized
# Fix: Check AWS profile has correct permissions
aws sts get-caller-identity --profile dev
```

**Resource already exists:**

```bash
# Error: Resource already exists
# Fix: Check if resource exists from previous deployment
# May need to manually delete or import
```

### Runtime Errors

**Lambda timeout:**

```bash
# Error: Task timed out after 30 seconds
# Fix: Increase timeout in CDK
```

**Database connection error:**

```bash
# Error: Failed to connect to database
# Fix: Check security group, VPC configuration
# Check SSM parameter store has correct credentials
```

**Authorization error:**

```bash
# Error: Unauthorized
# Fix: Check JWT token is valid
# Check token customizer is adding claims correctly
# Verify Cognito configuration
```

### Debugging Tips

**1. Add detailed logging:**

```go
logger.Info("Processing request",
    zap.String("resource", request.Resource),
    zap.String("method", request.HTTPMethod),
    zap.Int64("user_id", claims.UserID),
)
```

**2. Check CloudWatch Logs:**

```bash
aws logs tail /aws/lambda/infrastructure-project-management \
  --since 10m \
  --follow \
  --profile dev \
  --region us-east-2
```

**3. Test with curl:**

```bash
# Test endpoint directly
curl -v -X GET "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN"
```

**4. Query database with MCP:**

```
"Show me user 19's assignments"
"What projects are at location 6?"
"Check if project 29 is deleted"
```

**5. Check API Gateway logs:**

Enable CloudWatch logs for API Gateway in AWS Console.

---

## Quick Reference

### Common Commands

```bash
# Build
npm run build

# Deploy to dev
cd .. && npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev && cd infrastructure

# Deploy to prod
cd .. && npx cdk deploy "Infrastructure/Prod/Infrastructure-AppStage" --profile prod && cd infrastructure

# View Lambda logs
aws logs tail /aws/lambda/infrastructure-{service} --since 1h --follow --profile dev --region us-east-2

# Get auth token
TOKEN=$(curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"AuthFlow":"USER_PASSWORD_AUTH","ClientId":"3f0fb5mpivctnvj85tucusf88e","AuthParameters":{"USERNAME":"buildboard007+555@gmail.com","PASSWORD":"Mayur@1234"}}' \
  | jq -r '.AuthenticationResult.IdToken')

# Test API
curl "$API_BASE/endpoint" -H "Authorization: Bearer $TOKEN" | jq .
```

### Key File Locations

- Lambda handlers: `/Users/mayur/git_personal/infrastructure/src/infrastructure-{service}/main.go`
- Repositories: `/Users/mayur/git_personal/infrastructure/src/lib/data/`
- Models: `/Users/mayur/git_personal/infrastructure/src/lib/models/`
- CDK infrastructure: `/Users/mayur/git_personal/infrastructure/lib/`
- API test scripts: `/Users/mayur/git_personal/infrastructure/testing/api/`
- Postman collections: `/Users/mayur/git_personal/infrastructure/postman/`

---

**Last Updated:** 2025-10-27