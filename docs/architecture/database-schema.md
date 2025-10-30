# Database Schema Documentation - BuildBoard Infrastructure

**Last Updated:** 2025-10-27
**Purpose:** Comprehensive database schema design and patterns documentation

---

## Table of Contents
1. [Schema Philosophy](#schema-philosophy)
2. [Schema Organization](#schema-organization)
3. [Design Patterns](#design-patterns)
4. [IAM Schema Tables](#iam-schema-tables)
5. [Project Schema Tables](#project-schema-tables)
6. [Indexes and Performance](#indexes-and-performance)
7. [Data Types and Constraints](#data-types-and-constraints)
8. [Migration History](#migration-history)
9. [Query Patterns](#query-patterns)

---

## Schema Philosophy

### Separation of Concerns

The database is organized into two logical schemas to separate identity/access management from core business functionality:

```
appdb (PostgreSQL Database)
├── iam (Identity and Access Management)
│   └── User authentication, roles, permissions, assignments
└── project (Construction Management)
    └── Projects, issues, RFIs, submittals, attachments
```

### Why Two Schemas?

**1. Logical Separation**
- **IAM Schema:** Who can access the system and what they can do
- **Project Schema:** What work is being done and its status
- Clear boundaries reduce cognitive load
- Easier to reason about security vs. business logic

**2. Security Model**
- IAM schema controls access to project schema
- Separation simplifies permission auditing
- Different backup/recovery requirements
- Potential for different database users with schema-level permissions

**3. Scalability**
- IAM schema changes less frequently (stable)
- Project schema evolves with business requirements
- Can split into separate databases in future if needed
- Microservices can own specific schemas

**4. Multi-Tenancy**
- IAM schema handles organization-level concerns
- Project schema contains organization-specific data
- Both enforce org_id isolation

### Design Principles

1. **Normalization:** 3NF (Third Normal Form) for data integrity
2. **Denormalization:** Strategic for read-heavy operations (e.g., project_number)
3. **Soft Deletes:** Never hard delete, always use is_deleted flag
4. **Audit Trail:** All tables track created_by, created_at, updated_by, updated_at
5. **Multi-Tenancy:** All tables include org_id for tenant isolation
6. **Immutable Data:** Historical records preserved (assignments, comments)

---

## Schema Organization

### IAM Schema (iam)

**Purpose:** Identity, Access Management, and Authorization

**Core Entities:**
- Organizations (tenants)
- Users (people)
- Locations (physical sites)
- Roles (job functions)
- Permissions (system capabilities)
- Role Permissions (role → permission mapping)
- User Assignments (user → role → context mapping)

**Key Characteristics:**
- Changes infrequently
- Small data volume per organization
- High read, low write
- Critical for security

### Project Schema (project)

**Purpose:** Construction Management Business Logic

**Core Entities:**
- Projects (construction jobs)
- Issues (quality/safety/deficiency tracking)
- RFIs (Request for Information)
- Submittals (approval workflow)
- Attachments (files and photos)
- Comments (activity tracking)

**Key Characteristics:**
- Changes frequently
- Large data volume (files, photos)
- High read and write
- Business-critical data

---

## Design Patterns

### Pattern 1: Soft Delete (Everywhere)

**Implementation:**
```sql
is_deleted BOOLEAN DEFAULT FALSE NOT NULL
```

**Usage:**
```sql
-- Always filter deleted records
SELECT * FROM iam.users WHERE is_deleted = FALSE;

-- Soft delete operation
UPDATE iam.users SET is_deleted = TRUE, updated_by = $1, updated_at = NOW()
WHERE id = $2;
```

**Why Soft Deletes?**
- **Audit Trail:** Maintain complete history of all changes
- **Data Recovery:** "Undo" operations when users delete accidentally
- **Referential Integrity:** Foreign keys remain valid
- **Compliance:** Legal requirements to retain data
- **Analytics:** Historical reporting includes deleted records

**Trade-offs:**
- Must always include `is_deleted = FALSE` in queries
- Indexes still include deleted records (filtered indexes help)
- Storage grows over time (plan for archival)
- Unique constraints more complex (need conditional uniqueness)

### Pattern 2: Audit Trail (Standard Columns)

**Implementation:**
```sql
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
created_by BIGINT NOT NULL,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
updated_by BIGINT NOT NULL
```

**Automatic Updates:**
```sql
-- Trigger function (reused across all tables)
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Applied to every table
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON iam.users
    FOR EACH ROW
    EXECUTE PROCEDURE update_updated_at_column();
```

**Why Audit Trail?**
- **Accountability:** Know who changed what and when
- **Debugging:** Trace issues to specific changes
- **Compliance:** Regulatory requirements (SOC 2, HIPAA)
- **User Behavior:** Understand system usage patterns

**Usage:**
```go
// Application code sets created_by and updated_by
repo.CreateProject(ctx, ProjectRequest{
    Name: "New Project",
    // ... other fields
}, claims.UserID) // created_by = claims.UserID
```

### Pattern 3: Auto-Generated Numbers

**Why Auto-Generated Numbers?**
- **User-Friendly:** ISS-0001 is easier than UUID or internal ID
- **Sequential:** Indicates order of creation
- **Searchable:** Users can reference by number
- **Formatted:** Project numbers include year (PROJ-2025-0001)

**Implementation Strategies:**

#### Strategy A: Database Sequences
```sql
-- Create sequence per entity type
CREATE SEQUENCE project.issue_number_seq START WITH 1;

-- Use in application code
SELECT CONCAT('ISS-', LPAD(nextval('project.issue_number_seq')::TEXT, 4, '0'));
-- Result: ISS-0001, ISS-0002, etc.
```

#### Strategy B: Application-Generated (Current Implementation)
```go
// Query for next number
func (r *IssueRepository) GetNextIssueNumber(ctx context.Context, projectID int64) (string, error) {
    var maxNumber int
    query := `
        SELECT COALESCE(MAX(CAST(SUBSTRING(issue_number FROM 5) AS INTEGER)), 0)
        FROM project.issues
        WHERE project_id = $1 AND is_deleted = FALSE
    `
    err := r.db.QueryRowContext(ctx, query, projectID).Scan(&maxNumber)
    nextNumber := maxNumber + 1
    return fmt.Sprintf("ISS-%04d", nextNumber), nil
}
```

**Number Formats:**
- **Projects:** `PROJ-YYYY-NNNN` (e.g., PROJ-2025-0001)
  - YYYY = year (for annual reset)
  - NNNN = 4-digit sequence within year
- **Issues:** `ISS-NNNN` (e.g., ISS-0001)
  - Per-project numbering
- **RFIs:** `RFI-NNNN` (e.g., RFI-0042)
  - Per-project numbering
- **Submittals:** `SUB-NNNN` (e.g., SUB-0123)
  - Per-project numbering

**Concurrency Considerations:**
- Database-level sequences are atomic (no race conditions)
- Application-generated requires transaction isolation
- Consider SELECT ... FOR UPDATE for application-generated numbers

### Pattern 4: Context-Based Assignment (user_assignments)

**The Most Important Table:**
```sql
CREATE TABLE iam.user_assignments (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES iam.users(id),
    role_id BIGINT NOT NULL REFERENCES iam.roles(id),
    context_type VARCHAR(20) NOT NULL, -- 'organization', 'location', 'project'
    context_id BIGINT NOT NULL,         -- ID of the context entity
    trade_type VARCHAR(255),            -- Optional trade specialization
    is_primary BOOLEAN DEFAULT FALSE,   -- Primary assignment flag
    start_date DATE,                    -- Time-bounded assignments
    end_date DATE,
    -- Standard audit columns
    CONSTRAINT unique_user_role_context UNIQUE (user_id, role_id, context_type, context_id, is_deleted)
);
```

**Why This Pattern?**
- **Single Source of Truth:** All assignments in one table
- **Hierarchical Access:** Organization → Location → Project
- **Flexible:** Easily add new context types (department, equipment, phase)
- **Queryable:** Simple SQL to find all user contexts
- **Extensible:** Additional fields (start_date, end_date, trade_type)

**Context Types:**

1. **Organization** (context_type = 'organization')
   - User assigned at org level → sees ALL locations and ALL projects
   - Example: CEO, CFO, Operations Manager
   ```sql
   INSERT INTO iam.user_assignments (user_id, role_id, context_type, context_id)
   VALUES (123, 5, 'organization', 10); -- User 123 is Org Admin of Org 10
   ```

2. **Location** (context_type = 'location')
   - User assigned at location level → sees ALL projects at that location
   - Example: Regional Manager, Location Superintendent
   ```sql
   INSERT INTO iam.user_assignments (user_id, role_id, context_type, context_id)
   VALUES (456, 7, 'location', 22); -- User 456 is Location Manager of Location 22
   ```

3. **Project** (context_type = 'project')
   - User assigned at project level → sees ONLY that specific project
   - Example: Project Manager, Subcontractor, Inspector
   ```sql
   INSERT INTO iam.user_assignments (user_id, role_id, context_type, context_id)
   VALUES (789, 9, 'project', 55); -- User 789 is Project Manager of Project 55
   ```

**Access Control Queries:**
```sql
-- Get all projects user has access to
SELECT DISTINCT p.*
FROM project.projects p
LEFT JOIN iam.user_assignments ua ON (
    (ua.context_type = 'organization' AND ua.context_id = p.org_id) OR
    (ua.context_type = 'location' AND ua.context_id = p.location_id) OR
    (ua.context_type = 'project' AND ua.context_id = p.id)
)
WHERE ua.user_id = $1 AND ua.is_deleted = FALSE AND p.is_deleted = FALSE;

-- Get all context IDs of a specific type for a user
SELECT context_id
FROM iam.user_assignments
WHERE user_id = $1 AND context_type = $2 AND is_deleted = FALSE;
```

### Pattern 5: JSONB for Flexible Metadata

**Use Case:** Store flexible, non-queryable metadata

**Example (Potential Future Use):**
```sql
ALTER TABLE project.projects ADD COLUMN metadata JSONB;

-- Store custom fields per organization
UPDATE project.projects SET metadata = '{
    "custom_field_1": "value",
    "project_tags": ["high-priority", "leed-certified"],
    "client_specific": {"po_number": "PO-12345"}
}'::JSONB WHERE id = 123;

-- Query JSONB (indexed with GIN)
SELECT * FROM project.projects WHERE metadata->>'custom_field_1' = 'value';
```

**When to Use JSONB:**
- Custom fields that vary by organization
- Non-relational metadata
- Sparse data (many NULL values if normalized)
- Flexible schemas that evolve

**When NOT to Use JSONB:**
- Queryable/filterable fields → use proper columns
- Foreign key relationships → use proper tables
- Data requiring referential integrity

### Pattern 6: Enumerated Types (CHECK Constraints)

**Implementation:**
```sql
status VARCHAR(50) DEFAULT 'active' NOT NULL
    CONSTRAINT projects_status_check
    CHECK (status IN ('active', 'inactive', 'on_hold', 'completed', 'cancelled'))
```

**Why CHECK Constraints Instead of ENUM Types?**
- **Flexibility:** Easier to add new values (ALTER TABLE vs. recreate type)
- **Portability:** Works across database systems
- **Application Control:** Values documented in code
- **Migration Friendly:** No type dependencies

**Common Enumerations:**
- **Status:** active, inactive, deleted, suspended
- **Priority:** low, medium, high, critical
- **Project Type:** commercial, residential, industrial, etc.
- **Issue Severity:** minor, major, blocking, cosmetic

### Pattern 7: Centralized Attachments

**Old Pattern (Deprecated):**
```
project_attachments (project_id)
issue_attachments (issue_id)
rfi_attachments (rfi_id)
submittal_attachments (submittal_id)
```
- 4 separate tables
- Duplicate logic for uploads
- Inconsistent handling

**New Pattern (Current):**
```sql
CREATE TABLE project.attachments (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL,
    entity_type VARCHAR(50) NOT NULL,  -- 'project', 'issue', 'rfi', 'submittal', 'comment'
    entity_id BIGINT NOT NULL,         -- ID of parent entity
    file_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT,
    file_type VARCHAR(100),
    attachment_type VARCHAR(50),       -- 'photo', 'document', 'drawing', etc.
    s3_bucket VARCHAR(255),
    s3_key VARCHAR(500),
    uploaded_by BIGINT NOT NULL REFERENCES iam.users(id),
    -- Standard audit columns
);

-- Index for fast lookups
CREATE INDEX idx_attachments_entity ON project.attachments(entity_type, entity_id);
```

**Benefits:**
- Single Lambda function handles all attachments
- Consistent upload/download/delete logic
- Easy to add new entity types
- Centralized S3 management
- Simplified backup/restore

**Usage:**
```sql
-- Attach photo to issue
INSERT INTO project.attachments (org_id, entity_type, entity_id, file_name, ...)
VALUES (10, 'issue', 123, 'crack-in-wall.jpg', ...);

-- Get all attachments for an issue
SELECT * FROM project.attachments
WHERE entity_type = 'issue' AND entity_id = 123 AND is_deleted = FALSE;

-- Get all attachments across entity types
SELECT entity_type, COUNT(*) FROM project.attachments
WHERE org_id = 10 AND is_deleted = FALSE
GROUP BY entity_type;
```

---

## IAM Schema Tables

### organizations

**Purpose:** Multi-tenant organizations (customers)

**Schema:**
```sql
CREATE TABLE iam.organizations (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    org_type VARCHAR(50) NOT NULL,
        -- 'general_contractor', 'subcontractor', 'architect', 'owner', 'consultant'
    license_number VARCHAR(100),
    address TEXT,
    phone VARCHAR(20),
    email VARCHAR(255),
    website VARCHAR(255),
    status VARCHAR(50) DEFAULT 'pending_setup' NOT NULL,
        -- 'active', 'inactive', 'pending_setup', 'suspended'
    -- Standard audit columns
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE NOT NULL
);
```

**Indexes:**
- `idx_organizations_company_type` ON (org_type)
- `idx_organizations_status` ON (status)
- `idx_organizations_is_deleted` ON (is_deleted)

**Key Points:**
- Root of multi-tenant hierarchy
- All other tables reference org_id
- Status 'pending_setup' during onboarding
- org_type determines available features

### users

**Purpose:** People who access the system

**Schema:**
```sql
CREATE TABLE iam.users (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL REFERENCES iam.organizations(id),
    cognito_id VARCHAR(255) NOT NULL,  -- Links to AWS Cognito
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    phone VARCHAR(20),
    mobile VARCHAR(20),
    job_title VARCHAR(100),
    employee_id VARCHAR(50),
    avatar_url VARCHAR(500),
    last_selected_location_id BIGINT REFERENCES iam.locations(id),
    is_super_admin BOOLEAN DEFAULT FALSE NOT NULL,
    status VARCHAR(50) DEFAULT 'pending' NOT NULL,
        -- 'active', 'inactive', 'pending', 'pending_org_setup', 'suspended'
    -- Standard audit columns
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE NOT NULL
);
```

**Indexes:**
- `idx_users_org_id` ON (org_id)
- `idx_users_cognito_id` ON (cognito_id) -- Critical for authentication
- `idx_users_email` ON (email)
- `idx_users_employee_id` ON (employee_id)
- `idx_users_status` ON (status)
- `idx_users_is_deleted` ON (is_deleted)
- `idx_users_last_selected_location` ON (last_selected_location_id)

**Key Points:**
- `cognito_id` links to AWS Cognito authentication
- `is_super_admin = TRUE` grants full org access (bypasses assignments)
- `last_selected_location_id` improves UX (remember user preference)
- Multiple users can share same email in different orgs (org-scoped uniqueness)

### locations

**Purpose:** Physical sites/offices where work happens

**Schema:**
```sql
CREATE TABLE iam.locations (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL REFERENCES iam.organizations(id),
    name VARCHAR(255) NOT NULL,
    location_type VARCHAR(50) DEFAULT 'office' NOT NULL,
        -- 'office', 'warehouse', 'job_site', 'yard'
    address TEXT,
    city VARCHAR(100),
    state VARCHAR(50),
    zip_code VARCHAR(20),
    country VARCHAR(100) DEFAULT 'USA',
    status VARCHAR(50) DEFAULT 'active' NOT NULL,
        -- 'active', 'inactive', 'under_construction', 'closed'
    -- Standard audit columns
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE NOT NULL
);
```

**Indexes:**
- `idx_locations_org_id` ON (org_id)
- `idx_locations_type_status` ON (location_type, status)
- `idx_locations_is_deleted` ON (is_deleted)

**Key Points:**
- Middle tier of hierarchy: Organization → Locations → Projects
- Location-first UI pattern (users select location, then see projects)
- Can have multiple location types per organization

### roles

**Purpose:** Job functions and responsibilities

**Schema:**
```sql
CREATE TABLE iam.roles (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT REFERENCES iam.organizations(id), -- NULL for system roles
    name VARCHAR(100) NOT NULL,
    description TEXT,
    role_type VARCHAR(50) DEFAULT 'custom' NOT NULL,
        -- 'system', 'custom'
    construction_role_category VARCHAR(50) NOT NULL,
        -- 'management', 'field', 'office', 'external', 'admin'
    access_level VARCHAR(50) DEFAULT 'location' NOT NULL,
        -- 'organization', 'location', 'project'
    -- Standard audit columns
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE NOT NULL
);
```

**Indexes:**
- `idx_roles_org_id` ON (org_id)
- `idx_roles_type_category` ON (role_type, construction_role_category)
- `idx_roles_access_level` ON (access_level)
- `idx_roles_is_deleted` ON (is_deleted)

**Key Points:**
- `role_type = 'system'` → Built-in roles (org_id = NULL)
- `role_type = 'custom'` → Organization-specific roles
- `access_level` is descriptive (actual access via user_assignments.context_type)
- Examples: "Project Manager", "Superintendent", "Quality Inspector"

### permissions

**Purpose:** System capabilities (what can be done)

**Schema:**
```sql
CREATE TABLE iam.permissions (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(100) NOT NULL,  -- 'projects:create', 'issues:read'
    name VARCHAR(150) NOT NULL,
    description TEXT,
    permission_type VARCHAR(50) DEFAULT 'system' NOT NULL,
        -- 'system', 'custom'
    module VARCHAR(50) NOT NULL,  -- 'projects', 'issues', 'rfis', 'users'
    resource_type VARCHAR(50),    -- 'project', 'issue', 'rfi'
    action_type VARCHAR(50),      -- 'create', 'read', 'update', 'delete'
    -- Standard audit columns
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE NOT NULL
);
```

**Indexes:**
- `idx_permissions_code` ON (code) -- Unique identifier for permission
- `idx_permissions_module` ON (module)
- `idx_permissions_resource_action` ON (resource_type, action_type)
- `idx_permissions_type` ON (permission_type)
- `idx_permissions_is_deleted` ON (is_deleted)

**Key Points:**
- Code follows pattern: `<module>:<action>` (e.g., 'projects:create')
- Module groups related permissions
- System permissions are built-in
- Future: Custom permissions per organization

### role_permissions

**Purpose:** Maps permissions to roles (many-to-many)

**Schema:**
```sql
CREATE TABLE iam.role_permissions (
    role_id BIGINT NOT NULL REFERENCES iam.roles(id),
    permission_id BIGINT NOT NULL REFERENCES iam.permissions(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE NOT NULL,
    PRIMARY KEY (role_id, permission_id)
);
```

**Indexes:**
- `idx_role_permissions_is_deleted` ON (is_deleted)

**Key Points:**
- Composite primary key (role_id, permission_id)
- One role can have many permissions
- One permission can belong to many roles
- Soft delete preserves permission history

### user_assignments

**Purpose:** Assigns users to roles in specific contexts (THE MOST IMPORTANT TABLE)

**Schema:**
```sql
CREATE TABLE iam.user_assignments (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES iam.users(id),
    role_id BIGINT NOT NULL REFERENCES iam.roles(id),
    context_type VARCHAR(20) NOT NULL,
        -- 'organization', 'location', 'project'
    context_id BIGINT NOT NULL,
        -- ID of organization, location, or project
    trade_type VARCHAR(255),
        -- Optional: 'electrical', 'plumbing', 'hvac', etc.
    is_primary BOOLEAN DEFAULT FALSE,
        -- Is this the user's primary role in this context?
    start_date DATE,
        -- When assignment becomes active
    end_date DATE,
        -- When assignment expires (NULL = permanent)
    -- Standard audit columns
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE NOT NULL,

    CONSTRAINT unique_user_role_context
        UNIQUE (user_id, role_id, context_type, context_id, is_deleted)
);
```

**Indexes:**
- `idx_user_assignments_user_id` ON (user_id)
- `idx_user_assignments_role_id` ON (role_id)
- `idx_user_assignments_context` ON (context_type, context_id)
- `idx_user_assignments_is_deleted` ON (is_deleted)

**Key Points:**
- Replaces 5 deprecated tables (see Migration History section)
- Hierarchical access: org → location → project
- Unique constraint prevents duplicate assignments
- `trade_type` for trade-specific access control
- Time-bounded assignments with start_date and end_date

**Deprecated Tables (DO NOT USE):**
- ❌ `iam.org_user_roles` → Use `user_assignments` with `context_type='organization'`
- ❌ `iam.location_user_roles` → Use `user_assignments` with `context_type='location'`
- ❌ `iam.user_location_access` → Deprecated (not needed)
- ❌ `project.project_user_roles` → Use `user_assignments` with `context_type='project'`
- ❌ `project.project_managers` → Never used

---

## Project Schema Tables

### projects

**Purpose:** Construction projects/jobs

**Schema:**
```sql
CREATE TABLE project.projects (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL REFERENCES iam.organizations(id),
    location_id BIGINT NOT NULL REFERENCES iam.locations(id),
    project_number VARCHAR(50),  -- PROJ-2025-0001 (auto-generated)
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Project Classification
    project_type VARCHAR(50) NOT NULL,
        -- 'commercial', 'residential', 'industrial', 'hospitality', 'healthcare',
        -- 'institutional', 'mixed-use', 'civil-infrastructure', 'recreation',
        -- 'aviation', 'specialized'
    project_stage VARCHAR(50),
        -- 'bidding', 'course-of-construction', 'pre-construction',
        -- 'post-construction', 'warranty'
    work_scope VARCHAR(50),
        -- 'new', 'renovation', 'restoration', 'maintenance'
    project_sector VARCHAR(50),
        -- Same values as project_type
    delivery_method VARCHAR(50),
        -- 'design-build', 'design-bid-build', 'construction-manager-at-risk',
        -- 'integrated-project-delivery', 'construction-manager-as-agent',
        -- 'public-private-partnership', 'other'
    project_phase VARCHAR(50) DEFAULT 'pre_construction' NOT NULL,
        -- 'pre_construction', 'design', 'permitting', 'construction',
        -- 'closeout', 'warranty'

    -- Timeline
    start_date DATE,
    planned_end_date DATE,
    actual_start_date DATE,
    actual_end_date DATE,
    substantial_completion_date DATE,
    project_finish_date DATE,
    warranty_start_date DATE,
    warranty_end_date DATE,

    -- Financial
    budget NUMERIC(15, 2),
    contract_value NUMERIC(15, 2),
    square_footage INTEGER,

    -- Location Details
    address TEXT,
    city VARCHAR(100),
    state VARCHAR(50),
    zip_code VARCHAR(20),
    country VARCHAR(100) DEFAULT 'USA',
    latitude NUMERIC(10, 8),
    longitude NUMERIC(11, 8),

    -- Metadata
    language VARCHAR(10) DEFAULT 'en',
    status VARCHAR(50) DEFAULT 'active' NOT NULL,
        -- 'active', 'inactive', 'on_hold', 'completed', 'cancelled'

    -- Standard audit columns
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE NOT NULL
);
```

**Indexes:**
- `idx_projects_org_id` ON (org_id)
- `idx_projects_location_id` ON (location_id)
- `idx_projects_number` ON (project_number)
- `idx_projects_type_phase` ON (project_type, project_phase)
- `idx_projects_stage` ON (project_stage)
- `idx_projects_sector` ON (project_sector)
- `idx_projects_delivery_method` ON (delivery_method)
- `idx_projects_dates` ON (start_date, planned_end_date)
- `idx_projects_status` ON (status)
- `idx_projects_is_deleted` ON (is_deleted)

**Key Points:**
- Rich metadata for construction-specific needs
- Multiple date fields track project lifecycle
- Geolocation support (latitude/longitude) for site mapping
- Project numbering: PROJ-YYYY-NNNN format

### issues

**Purpose:** Quality, safety, deficiency, and punch item tracking

**Schema:**
```sql
CREATE TABLE project.issues (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES project.projects(id),
    org_id BIGINT NOT NULL,
    issue_number VARCHAR(50) NOT NULL,  -- ISS-0001 (per-project)

    -- Issue Details
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    issue_type VARCHAR(50) DEFAULT 'general' NOT NULL,
        -- 'quality', 'safety', 'deficiency', 'punch_item', 'code_violation', 'general'
    severity VARCHAR(50) DEFAULT 'minor' NOT NULL,
        -- 'blocking', 'major', 'minor', 'cosmetic'
    priority VARCHAR(50) DEFAULT 'medium' NOT NULL,
        -- 'critical', 'high', 'medium', 'low', 'planned'
    status VARCHAR(50) DEFAULT 'open' NOT NULL,
        -- 'open', 'in_progress', 'ready_for_review', 'closed', 'rejected', 'on_hold'

    -- Location Details
    location_description VARCHAR(255),
    room_area VARCHAR(100),
    floor_level VARCHAR(50),
    location_building VARCHAR(100),
    location_level VARCHAR(50),
    location_room VARCHAR(100),
    location_x NUMERIC(10, 4),
    location_y NUMERIC(10, 4),
    latitude NUMERIC(10, 8),
    longitude NUMERIC(11, 8),

    -- Assignment
    trade_type VARCHAR(100),
    reported_by BIGINT NOT NULL REFERENCES iam.users(id),
    assigned_to BIGINT REFERENCES iam.users(id),
    assigned_company_id BIGINT REFERENCES iam.organizations(id),

    -- Timeline
    due_date DATE,
    closed_date TIMESTAMP,

    -- Additional Details
    cost_to_fix NUMERIC(15, 2) DEFAULT 0.00,
    drawing_reference VARCHAR(255),
    specification_reference VARCHAR(255),
    root_cause TEXT,
    discipline VARCHAR(100),
    distribution_list TEXT[],

    -- Template Support
    template_id BIGINT REFERENCES project.issue_templates(id),
    category VARCHAR(100),
    detail_category VARCHAR(100),
    issue_category VARCHAR(100),

    -- Standard audit columns
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE NOT NULL
);
```

**Indexes:**
- `idx_issues_project_id` ON (project_id)
- `idx_issues_number` ON (issue_number)
- `idx_issues_type_severity` ON (issue_type, severity)
- `idx_issues_status` ON (status)
- `idx_issues_priority` ON (priority)
- `idx_issues_reported_by` ON (reported_by)
- `idx_issues_assigned_to` ON (assigned_to)
- `idx_issues_trade_type` ON (trade_type)
- `idx_issues_due_date` ON (due_date)
- `idx_issues_category` ON (category)
- `idx_issues_template` ON (template_id)
- `idx_issues_discipline` ON (discipline)
- `idx_issues_issue_category` ON (issue_category)
- `idx_issues_is_deleted` ON (is_deleted)

**Key Points:**
- Comprehensive location tracking (building, level, room, coordinates)
- Template support for standardized issues
- Assignment to user or company
- Distribution list for notifications (TEXT[])

### issue_comments

**Purpose:** Comments and activity tracking on issues

**Schema:**
```sql
CREATE TABLE project.issue_comments (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL REFERENCES project.issues(id),
    comment TEXT NOT NULL,
    comment_type VARCHAR(50) DEFAULT 'comment' NOT NULL,
        -- 'comment', 'status_change', 'assignment', 'resolution'
    previous_value VARCHAR(255),  -- For audit trail of changes
    new_value VARCHAR(255),       -- For audit trail of changes
    -- Standard audit columns
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_by BIGINT NOT NULL REFERENCES iam.users(id),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE NOT NULL
);
```

**Indexes:**
- `idx_issue_comments_issue_id` ON (issue_id)
- `idx_issue_comments_type` ON (comment_type)
- `idx_issue_comments_is_deleted` ON (is_deleted)

**Key Points:**
- Comments can be plain text or track field changes
- `comment_type = 'status_change'` with previous_value/new_value for audit
- Users can edit their own comments (updated_at tracks)

### rfis (Request for Information)

**Purpose:** Track questions and clarifications during construction

**Schema:**
```sql
CREATE TABLE project.rfis (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES project.projects(id),
    org_id BIGINT NOT NULL,
    rfi_number VARCHAR(50) NOT NULL,  -- RFI-0001 (per-project)

    -- RFI Content
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    question TEXT NOT NULL,
    response TEXT,

    -- Classification
    priority VARCHAR(50) DEFAULT 'medium' NOT NULL,
        -- 'low', 'medium', 'high', 'critical'
    status VARCHAR(50) DEFAULT 'draft' NOT NULL,
        -- 'draft', 'submitted', 'in_review', 'responded', 'closed', 'cancelled'

    -- References
    location_description VARCHAR(255),
    drawing_reference VARCHAR(255),
    specification_reference VARCHAR(255),
    trade_type VARCHAR(100),

    -- Assignment and Response
    submitted_by BIGINT NOT NULL REFERENCES iam.users(id),
    assigned_to BIGINT REFERENCES iam.users(id),
    response_by BIGINT REFERENCES iam.users(id),

    -- Timeline
    submitted_date TIMESTAMP,
    due_date TIMESTAMP,
    response_date TIMESTAMP,

    -- Impact Assessment
    cost_impact NUMERIC(15, 2) DEFAULT 0.00,
    schedule_impact_days INTEGER DEFAULT 0,

    -- Standard audit columns
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE NOT NULL
);
```

**Indexes:**
- `idx_rfis_project_id` ON (project_id)
- `idx_rfis_number` ON (rfi_number)
- `idx_rfis_status` ON (status)
- `idx_rfis_priority` ON (priority)
- `idx_rfis_submitted_by` ON (submitted_by)
- `idx_rfis_assigned_to` ON (assigned_to)
- `idx_rfis_due_date` ON (due_date)
- `idx_rfis_trade_type` ON (trade_type)
- `idx_rfis_is_deleted` ON (is_deleted)

**Key Points:**
- Tracks question → response workflow
- Impact assessment (cost and schedule)
- Drawing/spec references for context

### rfi_comments

**Purpose:** Comments on RFIs

**Schema:** Similar to issue_comments, references `rfi_id`

### submittals

**Purpose:** Product data, shop drawings, samples submission and approval workflow

**Schema:**
```sql
CREATE TABLE project.submittals (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES project.projects(id),
    org_id BIGINT NOT NULL,
    submittal_number VARCHAR(50) NOT NULL,  -- SUB-0001 (per-project)

    -- Submittal Details
    title VARCHAR(255) NOT NULL,
    description TEXT,
    submittal_type VARCHAR(50) NOT NULL,
        -- 'shop_drawings', 'product_data', 'samples', 'design_mix',
        -- 'test_reports', 'certificates', 'operation_manuals', 'warranties'

    -- Classification
    specification_section VARCHAR(50),
    drawing_reference VARCHAR(255),
    trade_type VARCHAR(100),
    priority VARCHAR(50) DEFAULT 'medium' NOT NULL,
        -- 'low', 'medium', 'high', 'critical'
    status VARCHAR(50) DEFAULT 'draft' NOT NULL,
        -- 'draft', 'submitted', 'under_review', 'approved',
        -- 'approved_with_comments', 'rejected', 'resubmit_required'
    revision_number INTEGER DEFAULT 1 NOT NULL,

    -- Parties
    submitted_by BIGINT NOT NULL REFERENCES iam.users(id),
    submitted_company_id BIGINT REFERENCES iam.organizations(id),
    reviewed_by BIGINT REFERENCES iam.users(id),

    -- Timeline
    submitted_date TIMESTAMP,
    due_date TIMESTAMP,
    reviewed_date TIMESTAMP,
    approval_date TIMESTAMP,

    -- Review Details
    review_comments TEXT,
    lead_time_days INTEGER,

    -- Quantity Information
    quantity_submitted INTEGER,
    unit_of_measure VARCHAR(20),

    -- Standard audit columns
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE NOT NULL
);
```

**Indexes:**
- `idx_submittals_project_id` ON (project_id)
- `idx_submittals_number` ON (submittal_number)
- `idx_submittals_type` ON (submittal_type)
- `idx_submittals_status` ON (status)
- `idx_submittals_priority` ON (priority)
- `idx_submittals_submitted_by` ON (submitted_by)
- `idx_submittals_reviewed_by` ON (reviewed_by)
- `idx_submittals_spec_section` ON (specification_section)
- `idx_submittals_trade_type` ON (trade_type)
- `idx_submittals_due_date` ON (due_date)
- `idx_submittals_is_deleted` ON (is_deleted)

**Key Points:**
- Revision tracking (revision_number increments)
- Multiple submittal types for different construction phases
- Tracks full approval workflow with dates

### submittal_items

**Purpose:** Line items within a submittal

**Schema:**
```sql
CREATE TABLE project.submittal_items (
    id BIGSERIAL PRIMARY KEY,
    submittal_id BIGINT NOT NULL REFERENCES project.submittals(id),
    item_number VARCHAR(50),
    item_description TEXT NOT NULL,
    manufacturer VARCHAR(255),
    model_number VARCHAR(100),
    quantity INTEGER,
    unit_price NUMERIC(15, 2),
    total_price NUMERIC(15, 2),
    status VARCHAR(50) DEFAULT 'pending' NOT NULL,
        -- 'pending', 'approved', 'rejected', 'approved_with_comments'
    comments TEXT,
    -- Standard audit columns
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE NOT NULL
);
```

**Key Points:**
- Detail-level tracking within submittals
- Product specifications (manufacturer, model)
- Pricing information

### submittal_reviews

**Purpose:** Review history for submittals

**Schema:**
```sql
CREATE TABLE project.submittal_reviews (
    id BIGSERIAL PRIMARY KEY,
    submittal_id BIGINT NOT NULL REFERENCES project.submittals(id),
    revision_number INTEGER NOT NULL,
    reviewer_id BIGINT NOT NULL REFERENCES iam.users(id),
    review_status VARCHAR(50) NOT NULL,
        -- 'approved', 'approved_with_comments', 'rejected', 'resubmit_required'
    review_comments TEXT,
    review_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    -- Standard audit columns
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE NOT NULL
);
```

**Key Points:**
- Full review audit trail
- Tracks each revision's review
- Multiple reviewers possible

### attachments (Centralized)

**Purpose:** All file attachments across the system

**Schema:**
```sql
CREATE TABLE project.attachments (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL REFERENCES iam.organizations(id),
    entity_type VARCHAR(50) NOT NULL,
        -- 'project', 'issue', 'rfi', 'submittal', 'comment'
    entity_id BIGINT NOT NULL,  -- ID of parent entity
    file_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT,
    file_type VARCHAR(100),
    attachment_type VARCHAR(50),
        -- 'photo', 'document', 'drawing', 'certificate', etc.
    s3_bucket VARCHAR(255),
    s3_key VARCHAR(500),
    uploaded_by BIGINT NOT NULL REFERENCES iam.users(id),
    -- Standard audit columns
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE NOT NULL
);
```

**Indexes:**
- `idx_attachments_org_id` ON (org_id)
- `idx_attachments_entity` ON (entity_type, entity_id)
- `idx_attachments_type` ON (attachment_type)
- `idx_attachments_uploaded_by` ON (uploaded_by)
- `idx_attachments_is_deleted` ON (is_deleted)

**Key Points:**
- Replaces 4+ separate attachment tables
- Polymorphic association via entity_type/entity_id
- S3 integration (bucket + key)
- File metadata (size, type)

---

## Indexes and Performance

### Indexing Strategy

**1. Primary Keys (Automatic)**
- BIGSERIAL creates btree index automatically
- Optimal for single-record lookups by ID

**2. Foreign Keys**
- Always indexed for join performance
- Example: `idx_projects_org_id`, `idx_issues_project_id`

**3. Query Filters**
- Index columns used in WHERE clauses
- Example: `idx_users_status`, `idx_projects_status`

**4. Compound Indexes**
- Index multiple columns used together
- Example: `idx_projects_type_phase`, `idx_issues_type_severity`
- Order matters: most selective column first

**5. Soft Delete**
- All tables have `idx_<table>_is_deleted`
- PostgreSQL can use index-only scans

**6. Date Ranges**
- Index date columns for timeline queries
- Example: `idx_projects_dates` ON (start_date, planned_end_date)

**7. Text Search (Future)**
- GIN indexes for full-text search
- Example: `CREATE INDEX idx_issues_fulltext ON project.issues USING GIN(to_tsvector('english', title || ' ' || description));`

### Query Performance Tips

```sql
-- Good: Uses index on project_id and is_deleted
SELECT * FROM project.issues
WHERE project_id = 123 AND is_deleted = FALSE;

-- Good: Compound index on type and severity
SELECT * FROM project.issues
WHERE issue_type = 'quality' AND severity = 'major' AND is_deleted = FALSE;

-- Bad: Leading wildcard prevents index usage
SELECT * FROM iam.users WHERE email LIKE '%@example.com';

-- Good: Index can be used
SELECT * FROM iam.users WHERE email LIKE 'john%';

-- Good: Use covering indexes for frequently accessed columns
CREATE INDEX idx_users_email_name ON iam.users(email, first_name, last_name)
WHERE is_deleted = FALSE;
```

### Monitoring Queries

```sql
-- Find missing indexes (unused indexes)
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0 AND indexname NOT LIKE '%_pkey';

-- Table sizes
SELECT schemaname, tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname IN ('iam', 'project')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Slow queries
SELECT query, calls, total_time, mean_time
FROM pg_stat_statements
WHERE query NOT LIKE '%pg_%'
ORDER BY mean_time DESC
LIMIT 20;
```

---

## Data Types and Constraints

### Common Data Types

| Type | Use Case | Example |
|------|----------|---------|
| BIGSERIAL | Auto-incrementing primary keys | `id BIGSERIAL PRIMARY KEY` |
| BIGINT | Foreign keys, large integers | `user_id BIGINT` |
| VARCHAR(n) | Short text with max length | `email VARCHAR(255)` |
| TEXT | Long text, no length limit | `description TEXT` |
| NUMERIC(p,s) | Decimal numbers (money) | `budget NUMERIC(15,2)` |
| BOOLEAN | True/false flags | `is_deleted BOOLEAN` |
| DATE | Calendar dates | `start_date DATE` |
| TIMESTAMP | Date + time | `created_at TIMESTAMP` |
| JSONB | Flexible metadata | `metadata JSONB` |
| TEXT[] | Array of strings | `distribution_list TEXT[]` |

### Constraint Types

**1. PRIMARY KEY**
```sql
id BIGSERIAL PRIMARY KEY
```

**2. FOREIGN KEY**
```sql
org_id BIGINT NOT NULL REFERENCES iam.organizations(id)
```

**3. UNIQUE**
```sql
CONSTRAINT unique_user_role_context UNIQUE (user_id, role_id, context_type, context_id)
```

**4. CHECK**
```sql
CONSTRAINT projects_status_check
    CHECK (status IN ('active', 'inactive', 'on_hold', 'completed', 'cancelled'))
```

**5. NOT NULL**
```sql
email VARCHAR(255) NOT NULL
```

**6. DEFAULT**
```sql
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
is_deleted BOOLEAN DEFAULT FALSE NOT NULL
```

---

## Migration History

### October 2025: Unified Assignment Management

**Migration:** Consolidate all assignment tables into `iam.user_assignments`

**Dropped Tables:**
1. `iam.org_user_roles` → Replaced by `user_assignments` with `context_type='organization'`
2. `iam.location_user_roles` → Replaced by `user_assignments` with `context_type='location'`
3. `iam.user_location_access` → Deprecated (location access determined by assignments)
4. `project.project_user_roles` → Replaced by `user_assignments` with `context_type='project'`
5. `project.project_managers` → Never used, removed

**Migration SQL:**
```sql
-- Create new unified table
CREATE TABLE iam.user_assignments (...);

-- Migrate org-level assignments
INSERT INTO iam.user_assignments (user_id, role_id, context_type, context_id, created_at, created_by, updated_at, updated_by)
SELECT user_id, role_id, 'organization', org_id, created_at, created_by, updated_at, updated_by
FROM iam.org_user_roles WHERE is_deleted = FALSE;

-- Migrate location-level assignments
INSERT INTO iam.user_assignments (user_id, role_id, context_type, context_id, created_at, created_by, updated_at, updated_by)
SELECT user_id, role_id, 'location', location_id, created_at, created_by, updated_at, updated_by
FROM iam.location_user_roles WHERE is_deleted = FALSE;

-- Migrate project-level assignments
INSERT INTO iam.user_assignments (user_id, role_id, context_type, context_id, trade_type, is_primary, start_date, end_date, created_at, created_by, updated_at, updated_by)
SELECT user_id, role_id, 'project', project_id, trade_type, is_primary, start_date, end_date, created_at, created_by, updated_at, updated_by
FROM project.project_user_roles WHERE is_deleted = FALSE;

-- Drop old tables
DROP TABLE iam.org_user_roles;
DROP TABLE iam.location_user_roles;
DROP TABLE iam.user_location_access;
DROP TABLE project.project_user_roles;
DROP TABLE project.project_managers;
```

**Rationale:**
- Single source of truth for assignments
- Eliminates duplicate code
- Hierarchical access control
- Extensible to new context types (department, equipment, phase)

### October 2025: Centralized Attachment Management

**Migration:** Consolidate entity-specific attachment tables into `project.attachments`

**Deprecated Tables:**
- `project.project_attachments` (still exists in schema, deprecated)
- `project.issue_attachments` (still exists in schema, deprecated)
- `project.rfi_attachments` (still exists in schema, deprecated)
- `project.submittal_attachments` (still exists in schema, deprecated)

**New Table:**
- `project.attachments` with entity_type/entity_id pattern

**Future Migration:**
```sql
-- Create new centralized table
CREATE TABLE project.attachments (...);

-- Migrate project attachments
INSERT INTO project.attachments (org_id, entity_type, entity_id, file_name, file_path, ...)
SELECT p.org_id, 'project', pa.project_id, pa.file_name, pa.file_path, ...
FROM project.project_attachments pa
JOIN project.projects p ON p.id = pa.project_id
WHERE pa.is_deleted = FALSE;

-- Similar for issue, rfi, submittal attachments...
```

---

## Query Patterns

### Access Control Queries

**Get projects user has access to:**
```sql
WITH user_contexts AS (
    -- Org-level access
    SELECT DISTINCT p.id
    FROM project.projects p
    JOIN iam.user_assignments ua ON ua.context_type = 'organization' AND ua.context_id = p.org_id
    WHERE ua.user_id = $1 AND ua.is_deleted = FALSE AND p.is_deleted = FALSE

    UNION

    -- Location-level access
    SELECT DISTINCT p.id
    FROM project.projects p
    JOIN iam.user_assignments ua ON ua.context_type = 'location' AND ua.context_id = p.location_id
    WHERE ua.user_id = $1 AND ua.is_deleted = FALSE AND p.is_deleted = FALSE

    UNION

    -- Project-level access
    SELECT DISTINCT p.id
    FROM project.projects p
    JOIN iam.user_assignments ua ON ua.context_type = 'project' AND ua.context_id = p.id
    WHERE ua.user_id = $1 AND ua.is_deleted = FALSE AND p.is_deleted = FALSE
)
SELECT p.* FROM project.projects p
JOIN user_contexts uc ON uc.id = p.id
WHERE p.org_id = $2;
```

**Get user's locations for dropdown:**
```sql
-- Super admin: all locations
SELECT DISTINCT l.*
FROM iam.locations l
WHERE l.org_id = $1 AND l.is_deleted = FALSE
AND EXISTS (SELECT 1 FROM iam.users WHERE id = $2 AND is_super_admin = TRUE);

-- Org-level: all locations
SELECT DISTINCT l.*
FROM iam.locations l
JOIN iam.user_assignments ua ON ua.context_type = 'organization' AND ua.context_id = l.org_id
WHERE ua.user_id = $2 AND ua.is_deleted = FALSE AND l.is_deleted = FALSE;

-- Location-level: assigned locations only
SELECT DISTINCT l.*
FROM iam.locations l
JOIN iam.user_assignments ua ON ua.context_type = 'location' AND ua.context_id = l.id
WHERE ua.user_id = $2 AND ua.is_deleted = FALSE AND l.is_deleted = FALSE;

-- Project-level: parent locations of assigned projects
SELECT DISTINCT l.*
FROM iam.locations l
JOIN project.projects p ON p.location_id = l.id
JOIN iam.user_assignments ua ON ua.context_type = 'project' AND ua.context_id = p.id
WHERE ua.user_id = $2 AND ua.is_deleted = FALSE AND l.is_deleted = FALSE AND p.is_deleted = FALSE;
```

### Reporting Queries

**Issue statistics by project:**
```sql
SELECT
    p.id AS project_id,
    p.name AS project_name,
    COUNT(*) FILTER (WHERE i.status = 'open') AS open_issues,
    COUNT(*) FILTER (WHERE i.status = 'in_progress') AS in_progress_issues,
    COUNT(*) FILTER (WHERE i.status = 'closed') AS closed_issues,
    COUNT(*) AS total_issues
FROM project.projects p
LEFT JOIN project.issues i ON i.project_id = p.id AND i.is_deleted = FALSE
WHERE p.org_id = $1 AND p.is_deleted = FALSE
GROUP BY p.id, p.name
ORDER BY total_issues DESC;
```

**User workload (assigned issues + RFIs):**
```sql
SELECT
    u.id,
    u.first_name || ' ' || u.last_name AS name,
    COUNT(DISTINCT i.id) AS assigned_issues,
    COUNT(DISTINCT r.id) AS assigned_rfis
FROM iam.users u
LEFT JOIN project.issues i ON i.assigned_to = u.id AND i.is_deleted = FALSE AND i.status NOT IN ('closed')
LEFT JOIN project.rfis r ON r.assigned_to = u.id AND r.is_deleted = FALSE AND r.status NOT IN ('closed', 'responded')
WHERE u.org_id = $1 AND u.is_deleted = FALSE
GROUP BY u.id, u.first_name, u.last_name
ORDER BY (COUNT(DISTINCT i.id) + COUNT(DISTINCT r.id)) DESC;
```

---

## Complete DDL Reference

For the complete, executable SQL schema definition, see:
**`/Users/mayur/git_personal/infrastructure/docs/sql/database-schema.sql`**

This document contains:
- All CREATE TABLE statements
- All indexes
- All foreign keys
- All triggers
- All constraints
- Grant statements for app_user

---

## Related Documentation

- [System Overview](./system-overview.md) - High-level architecture
- [APPLICATION-ARCHITECTURE.md](../APPLICATION-ARCHITECTURE.md) - Complete implementation guide
- [assignment-architecture.md](../assignment-architecture.md) - Deep dive on user assignments
- [CLAUDE.md](../../CLAUDE.md) - Development guidelines

---

**Document Maintainers:** Development Team
**Review Cycle:** After schema changes
**Questions:** Contact database administrator or technical lead