# Quick Start Guide

> Get up to speed in 5 minutes

## What is This Project?

BuildBoard Infrastructure - A serverless construction management system built with:
- **Backend**: Go Lambda functions
- **Infrastructure**: AWS CDK (TypeScript)
- **Database**: PostgreSQL (RDS)
- **Auth**: AWS Cognito
- **Frontend**: React (separate repo at `/Users/mayur/git_personal/ui/frontend`)

## System Architecture in 3 Points

1. **Serverless Multi-Service Architecture**
   - 11 Lambda functions, each managing a specific entity
   - API Gateway with Cognito JWT authorization
   - PostgreSQL database with `iam` + `project` schemas

2. **Hierarchical Access Control** ⭐
   - **Super Admin** → sees everything across all organizations
   - **Organization level** → sees all locations/projects in their org
   - **Location level** → sees all projects at their location
   - **Project level** → sees only assigned projects
   - **Core table**: `iam.user_assignments` controls ALL access

3. **Multi-Tenant Design**
   - Organizations → Locations → Projects → Issues/RFIs/Submittals
   - All data isolated by `org_id`
   - Users belong to one organization

## Key Tables You Must Know

1. **iam.user_assignments** ⭐ MOST IMPORTANT
   - Controls ALL access in the system
   - Structure: `user_id` + `role_id` + `context_type` + `context_id`
   - `context_type`: 'organization' | 'location' | 'project'
   - Replaces deprecated tables: `org_user_roles`, `location_user_roles`, `project_user_roles`

2. **iam.users**
   - User records linked to Cognito
   - `is_super_admin` flag for global admin access

3. **iam.organizations**
   - Top-level tenant entity
   - Multi-tenant isolation boundary

4. **iam.locations**
   - Belongs to organization
   - Groups projects geographically/functionally

5. **project.projects**
   - Construction projects
   - Belongs to location and organization
   - Auto-generated number: `PROJ-YYYY-NNNN`

6. **project.issues / rfis / submittals**
   - Project work items
   - Auto-numbered: `ISS-NNNN`, `RFI-NNNN`, `SUB-NNNN`

7. **Entity-specific attachment tables**
   - `project.issue_attachments`, `project.rfi_attachments`, etc.
   - Centralized pattern: `entity_type` + `entity_id`

## 11 Management Services

Each has: Lambda function + Repository + Data models + Postman collection

1. **Assignment Management** ⭐ - Core access control system
2. **User Management** - User CRUD operations
3. **Organization Management** - Organization setup and configuration
4. **Location Management** - Location CRUD operations
5. **Role Management** - Role definitions and permissions
6. **Permission Management** - Fine-grained permission system
7. **Project Management** - Project CRUD + team assignments
8. **Issue Management** - Issues + comments + workflow
9. **RFI Management** - Request for Information workflow
10. **Submittal Management** - Submittal workflow
11. **Attachment Management** - Centralized file/attachment handling

## Common Development Tasks

### Deploy to Dev
```bash
cd /Users/mayur/git_personal/infrastructure
npm run build
npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev
```

### Query Database (Use MCP Natural Language)
```
"Show me all assignments for user 19"
"List projects at location 6"
"What roles does user 19 have?"
"Show issue_comment_attachments table schema"
```

### Test API Endpoints
```bash
cd /Users/mayur/git_personal/infrastructure/testing/api
./test-project-user-management.sh
./test-get-projects-access-control.sh
./test-issue-comments.sh
```

## Test Credentials

**Primary Test User:**
- Email: `buildboard007+555@gmail.com`
- Password: `Mayur@1234`
- User ID: 19
- Org ID: 10
- Is Super Admin: Yes

**API Base URL (Dev):**
- `https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main`

**Database Connection (via MCP):**
- Host: `appdb.cdwmaay8wkw4.us-east-2.rds.amazonaws.com`
- Database: `appdb`
- Port: 5432
- SSL: Required

## Where to Go Next

### For AI Assistants
1. Read [architecture/access-control-system.md](architecture/access-control-system.md)
2. Read [entities/assignment-management.md](entities/assignment-management.md)
3. Reference [architecture/system-overview.md](architecture/system-overview.md)
4. Check [../CLAUDE.md](../CLAUDE.md) for AI-specific instructions

### For Developers
1. Review [guides/development-workflow.md](guides/development-workflow.md)
2. Understand [architecture/api-architecture.md](architecture/api-architecture.md)
3. Study [architecture/access-control-system.md](architecture/access-control-system.md)
4. Check [guides/testing-guide.md](guides/testing-guide.md)

### For Specific Entities
- **Understand access control** → [entities/assignment-management.md](entities/assignment-management.md)
- **File attachments** → [entities/attachment-management.md](entities/attachment-management.md)
- **Project issues** → [entities/issue-management.md](entities/issue-management.md)
- **User management** → [entities/user-management.md](entities/user-management.md)

See [entities/README.md](entities/README.md) for complete entity index.

## Critical Rules ⚠️

### ❌ DO NOT:
- Use deprecated tables: `org_user_roles`, `location_user_roles`, `project_user_roles`, `project_managers`
- Use bash/node scripts for database queries (use MCP instead)
- Create test files in project root (use `/testing/api/`)
- Use Cognito Access Tokens (use ID Tokens only)
- Hard delete records (use `is_deleted` flag for soft deletes)
- Skip access control checks in GET endpoints

### ✅ DO:
- Use `user_assignments` table for ALL access control
- Check access with `GetUserContexts()` repository method
- Follow hierarchical access model (Super Admin → Org → Location → Project)
- Use soft deletes with `is_deleted = true`
- Query database using MCP natural language
- Create test scripts in `/testing/api/`
- Return proper HTTP status codes: 400 (bad request), 404 (not found), 403 (forbidden), 500 (error)
- Provide clear, specific error messages

## Project Structure

```
infrastructure/
├── bin/                    # CDK app entry point
├── lib/                    # CDK infrastructure code (TypeScript)
│   ├── resources/         # AWS resource definitions
│   └── stacks/            # CDK stack definitions
├── src/                    # Go Lambda functions
│   ├── infrastructure-assignment-management/
│   ├── infrastructure-attachment-management/
│   ├── infrastructure-issue-management/
│   ├── infrastructure-location-management/
│   ├── infrastructure-organization-management/
│   ├── infrastructure-permissions-management/
│   ├── infrastructure-project-management/
│   ├── infrastructure-rfi-management/
│   ├── infrastructure-roles-management/
│   ├── infrastructure-submittal-management/
│   ├── infrastructure-user-management/
│   └── lib/               # Shared Go libraries
│       ├── api/           # API utilities
│       ├── auth/          # Authentication helpers
│       ├── clients/       # AWS clients (S3, SSM, etc.)
│       ├── constants/     # Constants
│       ├── data/          # Repository layer
│       ├── models/        # Data models
│       └── util/          # Utilities
├── docs/                   # This documentation
│   ├── architecture/      # System architecture
│   ├── entities/          # Entity-specific docs
│   ├── guides/            # How-to guides
│   ├── reference/         # Quick reference
│   ├── migration/         # Change history
│   └── sql/               # Database schemas
├── postman/                # API test collections
├── testing/                # Test scripts
│   └── api/               # API test shell scripts
└── cdk.json                # CDK configuration
```

## Key Concepts

### Auto-Generated Numbers
- **Projects**: `PROJ-2025-0001` (format: PROJ-YYYY-NNNN)
- **Issues**: `ISS-0001` (per project)
- **RFIs**: `RFI-0001` (per project)
- **Submittals**: `SUB-0001` (per project)

### Soft Deletes
All entities use `is_deleted` flag instead of hard deletes:
```sql
UPDATE table_name SET is_deleted = true WHERE id = $1
```

### Access Control Pattern
```go
// Get user's contexts (assignments)
contexts, err := assignmentRepo.GetUserContexts(ctx, userID, orgID)

// Check if user has access to specific project
hasAccess := false
for _, ctx := range contexts {
    if ctx.ContextType == "project" && ctx.ContextID == projectID {
        hasAccess = true
        break
    }
}
```

### Entity Type Pattern (Attachments)
```go
entityType := "issue_comment"  // or "issue", "rfi", "submittal", etc.
entityID := 123
tableName := GetTableName(entityType)  // "project.issue_comment_attachments"
```

## Environment Configuration

### AWS Accounts
- **Dev**: 521805123898 (us-east-2)
- **Prod**: 186375394147 (us-east-2)

### AWS Resources (Dev)
- **API Gateway**: `74zc1md7sc.execute-api.us-east-2.amazonaws.com`
- **Cognito User Pool ID**: `us-east-2_VkTLMp9RZ`
- **Cognito Client ID**: `3f0fb5mpivctnvj85tucusf88e`
- **RDS Database**: `appdb.cdwmaay8wkw4.us-east-2.rds.amazonaws.com`
- **S3 Attachments Bucket**: `buildboard-attachments-dev`

## Support

**Questions? Check:**
- Architecture questions → [architecture/](architecture/)
- Entity-specific questions → [entities/](entities/)
- How-to questions → [guides/](guides/)
- Quick lookups → [reference/](reference/)
- All docs → [README.md](README.md)

---

**Time to productive:** 5 minutes reading this + 15 minutes reviewing architecture docs

**Last Updated:** 2025-10-27