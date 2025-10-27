# Infrastructure Documentation

**Construction Management Application - Complete Documentation**

---

## 🚀 Getting Started

**New to this project or AI assistant reading this codebase?**

1. **Start here:** [DOCUMENTATION-INDEX.md](DOCUMENTATION-INDEX.md) - Quick navigation guide
2. **Read this:** [APPLICATION-ARCHITECTURE.md](APPLICATION-ARCHITECTURE.md) - Complete system reference (⭐ **ESSENTIAL**)
3. **Check this:** [../CLAUDE.md](../CLAUDE.md) - Project-specific AI assistant instructions

---

## 📚 Core Documentation

### Primary Reference
- **[APPLICATION-ARCHITECTURE.md](APPLICATION-ARCHITECTURE.md)** (30K) ⭐
  - Complete system overview and technology stack
  - Full database schema with all tables
  - Access control & permission system
  - API architecture with all endpoints
  - Authentication & authorization flow
  - Testing strategy and workflows
  - **Read this first for complete context**

### Navigation & History
- **[DOCUMENTATION-INDEX.md](DOCUMENTATION-INDEX.md)** (5.6K)
  - Quick reference index to all documentation
  - Links to specialized guides
  - Common mistakes to avoid

- **[CHANGES-SUMMARY.md](CHANGES-SUMMARY.md)** (8.5K)
  - October 2025 database cleanup and migration
  - Dropped tables and architectural changes
  - Migration impact and testing checklist

---

## 📖 Specialized Guides

### Technical Deep Dives
- **[assignment-architecture.md](assignment-architecture.md)** (12K)
  - Unified assignment management system
  - Permission inheritance hierarchy
  - Context types and access patterns

- **[VERIFICATION-user_assignments-can-replace-project_user_roles.md](VERIFICATION-user_assignments-can-replace-project_user_roles.md)** (7.9K)
  - Technical verification of migration
  - Schema comparison and API mapping

### User Guides
- **[USER_GUIDE_ENTITY_ATTACHMENTS.md](USER_GUIDE_ENTITY_ATTACHMENTS.md)** (17K)
  - Centralized attachment management
  - Entity attachment patterns
  - API usage examples

- **[super-admin-workflow.md](super-admin-workflow.md)** (19K)
  - Super admin onboarding process
  - Organization setup workflow
  - Email verification and configuration

### Operations
- **[deployment-guide.md](deployment-guide.md)** (8.9K)
  - Deployment procedures
  - CI/CD pipeline documentation
  - Environment management

- **[api-super-admin-restrictions.md](api-super-admin-restrictions.md)** (5.9K)
  - API endpoints requiring super admin
  - Permission restrictions by service
  - Authorization patterns

---

## 🎯 Quick Reference

### Test User Credentials
```
Email: buildboard007+555@gmail.com
Password: Mayur@1234
User ID: 19
Org ID: 10
Super Admin: Yes
```

### Environments
```
Dev Account:  521805123898
Prod Account: 186375394147
Region:       us-east-2
```

### API
```
Base URL: https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main
Auth:     Bearer token (ID Token from Cognito)
```

### Database
```
Host:     appdb.cdwmaay8wkw4.us-east-2.rds.amazonaws.com
Database: appdb
Schemas:  iam, project
Access:   Via MCP (natural language queries)
```

---

## 🔧 Development Workflows

### Build and Deploy
```bash
# Build
npm run build

# Deploy to Dev
npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev
```

### Database Queries
Use MCP natural language:
```
"Show me all user assignments for user 19"
"What projects are at location 6?"
"List all active RFIs in project 29"
```

### API Testing
```bash
# Get authentication token
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

# Use token for API calls
curl -X GET "https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/projects" \
  -H "Authorization: Bearer $TOKEN"
```

---

## 🧪 Testing

### Test Scripts
Located in `/testing/api/`:
- `test-project-user-management.sh` - Project assignment CRUD
- `test-get-projects-access-control.sh` - Access control verification
- `test-issue-comments.sh` - Issue comment system
- `test-comment-attachment.sh` - Comment attachments

### Postman Collections
Located in `/postman/`:
- `ProjectManagement.postman_collection.json`
- `IssueManagement.postman_collection.json`
- `RFIManagement.postman_collection.json`
- `SubmittalManagement.postman_collection.json`

---

## ⚠️ Important Notes

### Deprecated Tables (DO NOT USE)
These tables were dropped in October 2025:
- ❌ `iam.org_user_roles`
- ❌ `iam.location_user_roles`
- ❌ `iam.user_location_access`
- ❌ `project.project_user_roles`
- ❌ `project.project_managers`

**Use instead:** `iam.user_assignments` with appropriate `context_type`

### Access Control Hierarchy
```
Super Admin (is_super_admin = TRUE)
    └─ Sees EVERYTHING

Organization Assignment (context_type = 'organization')
    └─ Sees ALL locations & projects

Location Assignment (context_type = 'location')
    └─ Sees ALL projects at assigned locations

Project Assignment (context_type = 'project')
    └─ Sees ONLY assigned projects
```

---

## 🗄️ Archive

Historical planning documents available in [archive/](archive/) for reference only.

These documents are **outdated** and reference deprecated tables and old architecture.
**Do not use them for current development.**

See [archive/README.md](archive/README.md) for details.

---

## 📞 Support

For questions about:
- **Architecture:** See APPLICATION-ARCHITECTURE.md
- **Access Control:** See assignment-architecture.md
- **Deployments:** See deployment-guide.md
- **API Usage:** Check Postman collections
- **Database:** Use MCP natural language queries

---

**Last Updated:** 2025-10-25
**Documentation Version:** 2.0 (After cleanup)