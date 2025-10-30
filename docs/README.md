# BuildBoard Infrastructure Documentation

> Complete documentation for the BuildBoard construction management system

---

## Getting Started

### For AI Assistants
Start here to understand the system in 15 minutes:

1. **[QUICK-START.md](QUICK-START.md)** - Get up to speed in 5 minutes
2. **[architecture/access-control-system.md](architecture/access-control-system.md)** - Critical access control model
3. **[entities/assignment-management.md](entities/assignment-management.md)** - Core assignment system
4. **[../CLAUDE.md](../CLAUDE.md)** - AI-specific instructions and rules

### For Developers
Essential reading for new team members:

1. **[QUICK-START.md](QUICK-START.md)** - System overview and test credentials
2. **[guides/development-workflow.md](guides/development-workflow.md)** - Development process and standards
3. **[architecture/api-architecture.md](architecture/api-architecture.md)** - API design patterns
4. **[guides/testing-guide.md](guides/testing-guide.md)** - Testing procedures

---

## Documentation Structure

### Architecture (`architecture/`)
System design and technical architecture documents.

1. **[system-overview.md](architecture/system-overview.md)** - Complete system architecture
2. **[access-control-system.md](architecture/access-control-system.md)** - Access control model (hierarchical)
3. **[api-architecture.md](architecture/api-architecture.md)** - API Gateway, Lambda, endpoints
4. **[database-schema.md](architecture/database-schema.md)** - PostgreSQL schema (`iam` + `project`)
5. **[authentication-flow.md](architecture/authentication-flow.md)** - Cognito JWT authentication
6. **[multi-tenant-design.md](architecture/multi-tenant-design.md)** - Organization isolation model

### Entities (`entities/`)
Documentation for each of the 11 management services.

**Core Services:**
1. **[assignment-management.md](entities/assignment-management.md)** - Access control assignments
2. **[attachment-management.md](entities/attachment-management.md)** - Centralized file handling
3. **[user-management.md](entities/user-management.md)** - User CRUD operations
4. **[organization-management.md](entities/organization-management.md)** - Organization setup

**Hierarchy Services:**
5. **[location-management.md](entities/location-management.md)** - Location CRUD operations
6. **[project-management.md](entities/project-management.md)** - Project CRUD + team assignments

**Access Control:**
7. **[role-management.md](entities/role-management.md)** - Role definitions
8. **[permission-management.md](entities/permission-management.md)** - Fine-grained permissions

**Work Items:**
9. **[issue-management.md](entities/issue-management.md)** - Issues + comments + workflow
10. **[rfi-management.md](entities/rfi-management.md)** - Request for Information workflow
11. **[submittal-management.md](entities/submittal-management.md)** - Submittal workflow

### Guides (`guides/`)
How-to guides and operational procedures.

1. **[development-workflow.md](guides/development-workflow.md)** - Development process and standards
2. **[testing-guide.md](guides/testing-guide.md)** - Testing procedures and best practices
3. **[deployment-guide.md](guides/deployment-guide.md)** - Deployment procedures
4. **[api-usage-guide.md](guides/api-usage-guide.md)** - API integration guide
5. **[super-admin-workflow.md](guides/super-admin-workflow.md)** - Super admin operations

### Reference (`reference/`)
Quick reference materials and lookups.

1. **[api-endpoints.md](reference/api-endpoints.md)** - Complete API endpoint reference
2. **[error-codes.md](reference/error-codes.md)** - HTTP status codes and error messages
3. **[test-credentials.md](reference/test-credentials.md)** - Test users and environments
4. **[database-tables.md](reference/database-tables.md)** - Table reference guide
5. **[common-patterns.md](reference/common-patterns.md)** - Code patterns and snippets

### Migration History (`migration/`)
System changes and migration documentation.

1. **[CHANGES-SUMMARY.md](migration/CHANGES-SUMMARY.md)** - October 2025 database cleanup
2. **[VERIFICATION-user_assignments-can-replace-project_user_roles.md](migration/VERIFICATION-user_assignments-can-replace-project_user_roles.md)** - Migration verification

---

## Quick Reference

### Test Credentials

**Primary Test User:**
```
Email:     buildboard007+555@gmail.com
Password:  Mayur@1234
User ID:   19
Org ID:    10
Super Admin: Yes
```

### Environments

**Development:**
```
AWS Account: 521805123898
Region:      us-east-2
API Base:    https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main
```

**Production:**
```
AWS Account: 186375394147
Region:      us-east-2
```

### AWS Resources (Dev)

```
API Gateway:        74zc1md7sc.execute-api.us-east-2.amazonaws.com
Cognito User Pool:  us-east-2_VkTLMp9RZ
Cognito Client ID:  3f0fb5mpivctnvj85tucusf88e
RDS Database:       appdb.cdwmaay8wkw4.us-east-2.rds.amazonaws.com
S3 Bucket:          buildboard-attachments-dev
```

### Key Concepts

**Access Control Hierarchy:**
```
Super Admin (is_super_admin = true)
    └─ Sees EVERYTHING across all organizations

Organization Assignment (context_type = 'organization')
    └─ Sees ALL locations and projects in their org

Location Assignment (context_type = 'location')
    └─ Sees ALL projects at assigned locations

Project Assignment (context_type = 'project')
    └─ Sees ONLY assigned projects
```

**Auto-Generated Numbers:**
```
Projects:   PROJ-2025-0001  (format: PROJ-YYYY-NNNN)
Issues:     ISS-0001        (per project, sequential)
RFIs:       RFI-0001        (per project, sequential)
Submittals: SUB-0001        (per project, sequential)
```

**Critical Table:**
```
iam.user_assignments - Controls ALL access in the system
  - Replaces deprecated: org_user_roles, location_user_roles,
    project_user_roles, project_managers
  - Structure: user_id + role_id + context_type + context_id
  - Context types: 'organization' | 'location' | 'project'
```

---

## Testing

### Postman Collections

Located in `/postman/`:
- `AssignmentManagement.postman_collection.json`
- `AttachmentManagement.postman_collection.json`
- `Infrastructure.postman_collection.json` (Legacy - comprehensive)
- `IssueManagement.postman_collection.json`
- `ProjectManagement.postman_collection.json`
- `RFIManagement.postman_collection.json`
- `RolesManagement.postman_collection.json`
- `SubmittalManagement.postman_collection.json`

### Test Scripts

Located in `/testing/api/`:
- `test-project-user-management.sh` - Project assignment CRUD
- `test-get-projects-access-control.sh` - Access control verification
- `test-issue-comments.sh` - Issue comment system
- `test-comment-attachment.sh` - Comment attachments

**See:** `/testing/README.md` for complete testing guidelines

---

## Project Structure

```
infrastructure/
├── bin/                           # CDK app entry point
├── lib/                           # CDK infrastructure (TypeScript)
│   ├── resources/                # AWS resource definitions
│   │   ├── lambda/               # Lambda function constructs
│   │   └── sub_stack/            # Sub-stacks (RDS, Cognito, etc.)
│   └── stacks/                   # Main CDK stacks
├── src/                           # Go Lambda functions
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
│   └── lib/                      # Shared Go libraries
│       ├── api/                  # API utilities
│       ├── auth/                 # Authentication helpers
│       ├── clients/              # AWS clients (S3, SSM, etc.)
│       ├── constants/            # Constants
│       ├── data/                 # Repository layer (database)
│       ├── models/               # Data models
│       └── util/                 # Utilities
├── docs/                          # This documentation
│   ├── architecture/             # System architecture docs
│   ├── entities/                 # Entity-specific docs
│   ├── guides/                   # How-to guides
│   ├── reference/                # Quick reference materials
│   ├── migration/                # Change history
│   ├── sql/                      # Database schemas
│   └── archive/                  # Historical documents
├── postman/                       # Postman API collections
├── testing/                       # Test scripts
│   ├── api/                      # API test scripts
│   ├── auth/                     # Auth test scripts
│   ├── database/                 # Database validation
│   └── utilities/                # Helper scripts
└── cdk.json                       # CDK configuration
```

---

## Common Development Tasks

### Build and Deploy
```bash
# Build the project
cd /Users/mayur/git_personal/infrastructure
npm run build

# Deploy to Dev
npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev
```

### Query Database
Use MCP natural language (NEVER use bash/node scripts):
```
"Show me all assignments for user 19"
"List projects at location 6"
"What roles does user 19 have?"
"Show table schema for issue_comment_attachments"
"How many active RFIs are in project 29?"
```

### Test API Endpoints
```bash
cd /Users/mayur/git_personal/infrastructure/testing/api
./test-project-user-management.sh
./test-get-projects-access-control.sh
./test-issue-comments.sh
```

### Get Authentication Token
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

# Use the token
curl -X GET "https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/projects" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Critical Rules

### DO NOT:
- Use deprecated tables: `org_user_roles`, `location_user_roles`, `project_user_roles`, `project_managers`
- Use bash/node scripts for database queries (use MCP instead)
- Create test files in project root (use `/testing/api/`)
- Use Cognito Access Tokens (use ID Tokens only)
- Hard delete records (use `is_deleted` flag)
- Skip access control checks in GET endpoints
- Create files unless explicitly asked
- Proactively create documentation files

### DO:
- Use `iam.user_assignments` for ALL access control
- Check access with `GetUserContexts()` repository method
- Follow hierarchical access model (Super Admin > Org > Location > Project)
- Use soft deletes with `is_deleted = true`
- Query database using MCP natural language
- Create test scripts in `/testing/api/`
- Return proper HTTP status codes: 400, 403, 404, 500
- Provide clear, specific error messages
- Always prefer editing existing files over creating new ones

---

## Support

### Questions About:

**Architecture & Design:**
- System overview → [architecture/system-overview.md](architecture/system-overview.md)
- Access control → [architecture/access-control-system.md](architecture/access-control-system.md)
- Database design → [architecture/database-schema.md](architecture/database-schema.md)

**Specific Entities:**
- Assignment system → [entities/assignment-management.md](entities/assignment-management.md)
- File attachments → [entities/attachment-management.md](entities/attachment-management.md)
- Project issues → [entities/issue-management.md](entities/issue-management.md)
- All entities → [entities/](entities/)

**Development & Operations:**
- Development process → [guides/development-workflow.md](guides/development-workflow.md)
- Testing procedures → [guides/testing-guide.md](guides/testing-guide.md)
- Deployment → [guides/deployment-guide.md](guides/deployment-guide.md)

**Quick Lookups:**
- API endpoints → [reference/api-endpoints.md](reference/api-endpoints.md)
- Test credentials → [reference/test-credentials.md](reference/test-credentials.md)
- Error codes → [reference/error-codes.md](reference/error-codes.md)

---

## Additional Resources

### Legacy Documentation
- **[APPLICATION-ARCHITECTURE.md](APPLICATION-ARCHITECTURE.md)** (31K) - Comprehensive system reference
- **[assignment-architecture.md](assignment-architecture.md)** (12K) - Original assignment system design
- **[USER_GUIDE_ENTITY_ATTACHMENTS.md](USER_GUIDE_ENTITY_ATTACHMENTS.md)** (17K) - Attachment system guide
- **[api-super-admin-restrictions.md](api-super-admin-restrictions.md)** (6K) - Super admin API restrictions

### Database Schemas
- **[sql/database-schema.sql](sql/database-schema.sql)** - Complete database DDL
- **[sql/alter-org-nullable.sql](sql/alter-org-nullable.sql)** - Schema migration scripts

### Archive
Historical planning documents in [archive/](archive/) - **for reference only**.
These documents are outdated and reference deprecated tables. Do not use for current development.

---

**Last Updated:** 2025-10-27
**Documentation Version:** 3.0 (Reorganized structure)

**Time to Productive:**
- AI Assistants: 15 minutes (QUICK-START + access-control + assignment docs)
- Developers: 30 minutes (QUICK-START + guides + entity docs)