# Documentation Index - Construction Management Application

**Complete documentation for AI assistants and developers.**

---

## 📚 Core Documentation

### [APPLICATION-ARCHITECTURE.md](./APPLICATION-ARCHITECTURE.md) ⭐ **START HERE**
**Complete application context guide - Read this first!**

Contains:
- System overview and technology stack
- Complete database schema with all tables
- Access control & permission system (hierarchical model)
- API architecture with all endpoints
- Authentication & authorization flow
- Project structure
- Key architectural decisions
- Testing strategy
- Common development workflows

**Read this to get complete context of the entire application.**

---

## 📋 Change History

### [CHANGES-SUMMARY.md](./CHANGES-SUMMARY.md)
Documentation of the October 2025 database cleanup and consolidation.

**What changed:**
- Dropped 5 deprecated tables
- Consolidated all assignments into `user_assignments` table
- Migrated project user management to unified assignment system

**Key for understanding:**
- What tables to NOT use anymore
- Migration from old system to new
- API behavior changes (internal only, APIs unchanged)

---

## 🏗️ Architecture Deep Dives

### [assignment-architecture.md](./assignment-architecture.md)
**Detailed documentation of the unified assignment management system.**

Contains:
- Permission inheritance hierarchy
- Context types (organization, location, project, department, equipment, phase)
- Assignment repository API reference
- Access control patterns
- Code examples

**Read this when working on:**
- User assignments
- Access control logic
- Permission management

### [VERIFICATION-user_assignments-can-replace-project_user_roles.md](./VERIFICATION-user_assignments-can-replace-project_user_roles.md)
Technical verification that `user_assignments` can fully replace the legacy `project_user_roles` table.

**Contains:**
- Schema comparison (field-by-field mapping)
- API operation mapping
- Code refactoring guide
- Testing strategy

---

## 🧪 Testing Documentation

### Test Scripts Location
All test scripts are in `/testing/api/` directory:
- `test-project-user-management.sh` - Tests project assignment CRUD
- `test-get-projects-access-control.sh` - Verifies access control implementation
- `test-issue-comments.sh` - Tests issue comment system
- `test-comment-attachment.sh` - Tests comment attachments

### Postman Collections
Located in `/postman/` directory:
- `ProjectManagement.postman_collection.json`
- `IssueManagement.postman_collection.json`
- `RFIManagement.postman_collection.json`
- `SubmittalManagement.postman_collection.json`

---

## 🚀 Quick Start for AI Assistants

### First Time Reading This Codebase?

1. **Read [APPLICATION-ARCHITECTURE.md](./APPLICATION-ARCHITECTURE.md)** - Get complete system context
2. **Read [CLAUDE.md](../CLAUDE.md)** - Understand project-specific instructions
3. **Review test user credentials** - In APPLICATION-ARCHITECTURE.md "Testing Strategy" section
4. **Check Postman collections** - Understand API contracts

### Working on Specific Features?

**Access Control / Permissions:**
- Read: [assignment-architecture.md](./assignment-architecture.md)
- See: `/src/lib/data/assignment_repository.go`
- Example: `/src/infrastructure-project-management/main.go` (lines 131-295)

**Database Changes:**
- Read: [APPLICATION-ARCHITECTURE.md](./APPLICATION-ARCHITECTURE.md) "Database Architecture" section
- Check: What tables are deprecated in CHANGES-SUMMARY.md
- Query: Use MCP natural language for database exploration

**Adding New APIs:**
- Read: [APPLICATION-ARCHITECTURE.md](./APPLICATION-ARCHITECTURE.md) "API Architecture" section
- Follow: "Adding a New Lambda Function" workflow
- Test: Create test script in `/testing/api/`

---

## 📖 Key Concepts

### Access Control Hierarchy
```
Super Admin (is_super_admin = TRUE)
    └─ Sees EVERYTHING

Organization Assignment (context_type = 'organization')
    └─ Sees ALL locations & projects in org

Location Assignment (context_type = 'location')
    └─ Sees ALL projects at assigned location(s)

Project Assignment (context_type = 'project')
    └─ Sees ONLY assigned project(s)
```

### The Most Important Table
**`iam.user_assignments`** is the core of the entire access control system.

```sql
user_id + role_id + context_type + context_id = Access Grant
```

### Location-First UI Pattern
1. User selects location from dropdown
2. System shows projects at that location (filtered by user access)
3. User works within that location context

---

## ❌ Common Mistakes to Avoid

### 1. Using Deprecated Tables
**DON'T** reference these tables (dropped October 2025):
- `iam.org_user_roles`, `iam.location_user_roles`, `iam.user_location_access`
- `project.project_user_roles`, `project.project_managers`

**DO** use `iam.user_assignments` with appropriate `context_type`.

### 2. Using Access Tokens Instead of ID Tokens
**DON'T** use Access Tokens from Cognito.

**DO** use ID Tokens (contains custom claims).

### 3. Creating Test Files in Root
**DO** create test files in `/testing/api/` directory.

### 4. Using Bash for Database Queries
**DO** use natural language with MCP.

### 5. Hard Deletes
**DO** set `is_deleted = TRUE` for soft deletes.

---

## 📞 Key Information

### Test User
- **Email:** buildboard007+555@gmail.com
- **Password:** Mayur@1234
- **User ID:** 19, **Org ID:** 10, **Super Admin:** Yes

### Environments
- **Dev:** 521805123898, **Prod:** 186375394147, **Region:** us-east-2

### API
- **Base URL:** https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main

### Database
- **Host:** appdb.cdwmaay8wkw4.us-east-2.rds.amazonaws.com
- **Access:** Via MCP (natural language queries)

---

**Last Updated:** 2025-10-25