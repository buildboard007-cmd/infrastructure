# Entity Documentation Index

This directory contains comprehensive documentation for all core entities in the BuildBoard construction management system.

## Core Entities

### 1. [Assignment Management](./assignment-management.md)
**THE CORE ACCESS CONTROL SYSTEM** - Unified assignment system replacing all deprecated role tables. Manages user assignments to organizations, projects, locations, departments, equipment, and phases through the `user_assignments` table. This is the foundation of the entire access control and permission system.

### 2. Organization Management
Organization entity documentation - construction companies (general contractors, subcontractors, architects, owners, consultants).

### 3. Location Management
Location entity documentation - offices, warehouses, job sites, and yards within organizations.

### 4. Project Management
Project entity documentation - construction projects with stages, phases, types, and delivery methods.

### 5. User Management
User entity documentation - user accounts with Cognito authentication, roles, and permissions.

### 6. Role Management
Role entity documentation - system and custom roles with construction-specific categories and access levels.

### 7. Permission Management
Permission entity documentation - granular permissions for resources and actions across modules.

### 8. RFI Management
RFI (Request for Information) entity documentation - questions, responses, and workflow tracking.

### 9. Submittal Management
Submittal entity documentation - shop drawings, product data, samples, and approval workflows.

### 10. Issue Management
Issue entity documentation - quality, safety, deficiencies, punch items, and code violations.

### 11. Attachment Management
Attachment entity documentation - file uploads for projects, RFIs, submittals, and issues with S3 storage.

## Key Relationships

- **Assignments** link Users to Organizations, Projects, and Locations with specific Roles
- **Projects** belong to Organizations and Locations
- **RFIs, Submittals, and Issues** belong to Projects
- **Attachments** can be associated with multiple entity types
- **Permissions** are granted through Role assignments in specific contexts

## Deprecated Tables (Replaced by Assignments)

The following tables are deprecated and replaced by the unified `user_assignments` system:
- `org_user_roles` - replaced by assignments with context_type='organization'
- `location_user_roles` - replaced by assignments with context_type='location'
- `project_user_roles` - replaced by assignments with context_type='project'
- `project_managers` - replaced by assignments with context_type='project'

## Documentation Standards

Each entity documentation file includes:
1. Overview and purpose
2. Database schema
3. Data models (Go structs)
4. API endpoints
5. Repository methods
6. Lambda handlers
7. Access control
8. Related entities
9. Common workflows
10. Postman collection
11. Testing scripts
12. Troubleshooting

## Getting Started

Start with [Assignment Management](./assignment-management.md) to understand the core access control system, then explore other entities based on your needs.