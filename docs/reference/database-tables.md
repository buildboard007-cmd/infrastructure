# Database Tables Reference

> Quick reference for all database tables across IAM and Project schemas

**Database:** `appdb` (PostgreSQL on AWS RDS)
**Host:** `appdb.cdwmaay8wkw4.us-east-2.rds.amazonaws.com`

---

## IAM Schema (`iam.*`)

Identity and Access Management tables for users, organizations, roles, and permissions.

| Table Name | Columns | Primary Key | Purpose |
|------------|---------|-------------|---------|
| `iam.users` | 19 | `id` (bigserial) | User accounts linked to Cognito |
| `iam.organizations` | 14 | `id` (bigserial) | Top-level tenant entities |
| `iam.locations` | 15 | `id` (bigserial) | Physical locations within organizations |
| `iam.roles` | 12 | `id` (bigserial) | Role definitions (system + custom) |
| `iam.permissions` | 13 | `id` (bigserial) | Permission definitions |
| `iam.role_permissions` | 7 | `(role_id, permission_id)` | Many-to-many role-permission mapping |
| `iam.user_assignments` | 14 | `id` (bigserial) | **CORE TABLE** - User assignments to org/location/project |

---

## Project Schema (`project.*`)

Project-specific tables for construction management entities.

| Table Name | Columns | Primary Key | Purpose |
|------------|---------|-------------|---------|
| `project.projects` | 40 | `id` (bigserial) | Construction projects |
| `project.issues` | 38 | `id` (bigserial) | Issue tracking and punch lists |
| `project.issue_comments` | 11 | `id` (bigserial) | Issue comments and activity logs |
| `project.issue_attachments` | 13 | `id` (bigserial) | Issue file attachments |
| `project.issue_comment_attachments` | 13 | `id` (bigserial) | Issue comment attachments |
| `project.issue_templates` | 13 | `id` (bigserial) | Reusable issue templates |
| `project.rfis` | 57 | `id` (bigserial) | Request for Information workflow |
| `project.rfi_comments` | 11 | `id` (bigserial) | RFI comments and activity |
| `project.rfi_attachments` | 18 | `id` (bigserial) | RFI file attachments |
| `project.submittals` | 52 | `id` (bigserial) | Submittal management workflow |
| `project.submittal_items` | 16 | `id` (bigserial) | Individual items within submittals |
| `project.submittal_reviews` | 12 | `id` (bigserial) | Submittal review history |
| `project.submittal_attachments` | 13 | `id` (bigserial) | Submittal file attachments |
| `project.project_attachments` | 13 | `id` (bigserial) | Project-level attachments |

---

## IAM Schema Details

### `iam.users`

**Purpose:** User accounts with Cognito integration

**Key Columns:**
- `id`, `org_id`, `cognito_id` (UUID from Cognito)
- `email`, `first_name`, `last_name`
- `is_super_admin` (global admin flag)
- `status` (pending, active, inactive, suspended)
- `last_selected_location_id` (UI preference)

**Indexes:** `cognito_id`, `email`, `org_id`

---

### `iam.organizations`

**Purpose:** Multi-tenant top-level entities

**Key Columns:**
- `id`, `name`, `org_type`
- `license_number`, `address`, `phone`, `email`, `website`
- `status` (pending_setup, active, inactive, suspended)

**Indexes:** `name`, `status`

---

### `iam.locations`

**Purpose:** Physical locations within organizations

**Key Columns:**
- `id`, `org_id`, `name`
- `location_type` (office, warehouse, project_site, regional_office)
- `address`, `city`, `state`, `zip_code`, `country`
- `status` (active, inactive, closed)

**Indexes:** `org_id`, `name`, `status`

---

### `iam.roles`

**Purpose:** Role definitions (system-provided and custom)

**Key Columns:**
- `id`, `org_id` (NULL for system roles)
- `name`, `description`
- `role_type` (system, custom)
- `category` (administrative, field, design, qa_qc, specialty)
- `access_level` (organization, location, project)

**System Roles:** Project Manager, Superintendent, Foreman, Inspector, Designer, Owner Representative

**Indexes:** `org_id`, `role_type`, `category`

---

### `iam.permissions`

**Purpose:** Fine-grained permission definitions

**Key Columns:**
- `id`, `code` (unique identifier)
- `name`, `description`
- `permission_type` (system, custom)
- `module` (project, issue, rfi, submittal, user, role, etc.)
- `resource_type`, `action_type`

**Permission Pattern:** `{module}:{action}` (e.g., `project:create`, `issue:update`)

**Indexes:** `code`, `module`, `permission_type`

---

### `iam.role_permissions`

**Purpose:** Many-to-many mapping between roles and permissions

**Key Columns:**
- `role_id`, `permission_id` (composite primary key)
- Standard audit fields

**Indexes:** `role_id`, `permission_id`

---

### `iam.user_assignments` ⭐ MOST IMPORTANT

**Purpose:** Controls ALL access in the system (replaces deprecated tables)

**Key Columns:**
- `id`, `user_id`, `role_id`
- `context_type` (organization, location, project)
- `context_id` (ID of org/location/project)
- `trade_type` (optional, for trade-specific assignments)
- `is_primary` (primary assignment flag)
- `start_date`, `end_date` (optional time-bound assignments)

**Access Hierarchy:**
1. **Super Admin** → sees everything
2. **Organization** → sees all locations/projects in org
3. **Location** → sees all projects at location
4. **Project** → sees only that project

**Indexes:** `user_id`, `context_type`, `context_id`, `role_id`

**Replaces Deprecated Tables:** `org_user_roles`, `location_user_roles`, `project_user_roles`, `project_managers`

---

## Project Schema Details

### `project.projects`

**Purpose:** Construction project records

**Key Columns:**
- `id`, `org_id`, `location_id`
- `project_number` (auto: PROJ-YYYY-NNNN)
- `name`, `description`
- `project_type`, `project_stage`, `project_phase`
- `start_date`, `planned_end_date`, `actual_end_date`
- `budget`, `contract_value`, `square_footage`
- `address`, `city`, `state`, `zip_code`
- `latitude`, `longitude`
- `status` (active, inactive, completed, on_hold, cancelled)

**Indexes:** `org_id`, `location_id`, `project_number`, `status`

---

### `project.issues`

**Purpose:** Issue tracking, punch lists, deficiencies

**Key Columns:**
- `id`, `project_id`
- `issue_number` (auto: ISS-NNNN, per project)
- `title`, `description`
- `issue_type`, `category`, `detail_category`
- `priority` (critical, high, medium, low, planned)
- `severity` (blocking, major, minor, cosmetic)
- `status` (open, in_progress, ready_for_review, closed, rejected, on_hold)
- `reported_by`, `assigned_to`, `assigned_company_id`
- `location_building`, `location_level`, `location_room`
- `trade_type`, `discipline`
- `due_date`, `closed_date`
- `cost_to_fix`
- `distribution_list` (array of emails)

**Indexes:** `project_id`, `status`, `assigned_to`, `issue_number`

---

### `project.issue_comments`

**Purpose:** Comments and activity logs for issues

**Key Columns:**
- `id`, `issue_id`
- `comment`, `comment_type` (comment, activity)
- `previous_value`, `new_value` (for activity logs)

**Indexes:** `issue_id`, `created_at`

---

### `project.issue_attachments` & `project.issue_comment_attachments`

**Purpose:** File attachments for issues and comments

**Key Columns:**
- `id`, `issue_id` or `comment_id`
- `file_name`, `file_path` (S3 key)
- `file_size`, `file_type`
- `attachment_type` (before_photo, progress_photo, after_photo, issue_document, photo, document)
- `uploaded_by`

**Indexes:** `issue_id`, `comment_id`

---

### `project.rfis`

**Purpose:** Request for Information workflow

**Key Columns:**
- `id`, `project_id`, `org_id`, `location_id`
- `rfi_number` (auto: RFI-NNNN, per project)
- `subject`, `question`, `description`
- `category`, `discipline`, `trade_type`
- `priority` (critical, high, medium, low)
- `status` (draft, open, in_review, answered, closed)
- `submitted_by`, `assigned_to`, `response_by`
- `submitted_date`, `due_date`, `response_date`, `closed_date`
- `response`, `response_status`
- `cost_impact`, `schedule_impact`
- `cost_impact_amount`, `schedule_impact_days`
- `drawing_references`, `specification_references`
- `related_rfis`, `related_submittals`
- `distribution_list`, `cc_list` (arrays)

**Indexes:** `project_id`, `rfi_number`, `status`

---

### `project.rfi_comments` & `project.rfi_attachments`

**Purpose:** Comments and attachments for RFIs

**Structure:** Similar to issue comments/attachments

**Indexes:** `rfi_id`

---

### `project.submittals`

**Purpose:** Submittal management and workflow

**Key Columns:**
- `id`, `project_id`, `org_id`, `location_id`
- `submittal_number` (auto: SUB-NNNN, per project)
- `title`, `description`
- `submittal_type` (shop_drawing, product_data, sample, material, equipment, other)
- `specification_section`, `drawing_reference`
- `trade_type`, `priority`
- `status` (draft, submitted, under_review, approved, approved_as_noted, rejected, revise_and_resubmit)
- `revision_number`
- `submitted_by`, `reviewed_by`, `assigned_to`, `reviewer`, `approver`
- `submitted_date`, `due_date`, `reviewed_date`, `approval_date`
- `current_phase`, `ball_in_court`, `workflow_status`
- `linked_drawings`, `submittal_references` (JSONB)
- `distribution_list`, `notification_settings` (JSONB)

**Indexes:** `project_id`, `submittal_number`, `status`

---

### `project.submittal_items`

**Purpose:** Line items within submittals

**Key Columns:**
- `id`, `submittal_id`
- `item_number`, `item_description`
- `manufacturer`, `model_number`
- `quantity`, `unit_price`, `total_price`
- `status` (pending, approved, rejected)

**Indexes:** `submittal_id`

---

### `project.submittal_reviews`

**Purpose:** Submittal review history and approvals

**Key Columns:**
- `id`, `submittal_id`, `revision_number`
- `reviewer_id`, `review_status`
- `review_comments`, `review_date`

**Indexes:** `submittal_id`, `revision_number`

---

### `project.submittal_attachments`

**Purpose:** File attachments for submittals

**Structure:** Similar to other attachment tables

**Indexes:** `submittal_id`

---

### `project.project_attachments`

**Purpose:** Project-level attachments (plans, specs, contracts)

**Structure:** Similar to other attachment tables

**Indexes:** `project_id`

---

### `project.issue_templates`

**Purpose:** Reusable issue templates

**Key Columns:**
- `id`, `org_id`
- `name`, `category`, `detail_category`
- `default_priority`, `default_severity`, `default_description`
- `is_active`

**Indexes:** `org_id`, `is_active`

---

## Common Audit Fields

All tables include these standard audit columns:

| Column | Type | Purpose |
|--------|------|---------|
| `created_at` | TIMESTAMP | Record creation timestamp |
| `created_by` | BIGINT | User ID who created record |
| `updated_at` | TIMESTAMP | Last update timestamp |
| `updated_by` | BIGINT | User ID who last updated record |
| `is_deleted` | BOOLEAN | Soft delete flag (default: false) |

---

## Key Relationships

```
iam.organizations (1) ──→ (N) iam.locations
                   (1) ──→ (N) iam.users
                   (1) ──→ (N) iam.roles (custom)
                   (1) ──→ (N) project.projects

iam.locations (1) ──→ (N) project.projects

iam.users (1) ──→ (N) iam.user_assignments
          (1) ──→ (N) project.issues (created/assigned)

iam.roles (1) ──→ (N) iam.role_permissions
          (1) ──→ (N) iam.user_assignments

iam.permissions (1) ──→ (N) iam.role_permissions

project.projects (1) ──→ (N) project.issues
                 (1) ──→ (N) project.rfis
                 (1) ──→ (N) project.submittals
                 (1) ──→ (N) iam.user_assignments

project.issues (1) ──→ (N) project.issue_comments
               (1) ──→ (N) project.issue_attachments

project.submittals (1) ──→ (N) project.submittal_items
                   (1) ──→ (N) project.submittal_reviews
                   (1) ──→ (N) project.submittal_attachments
```

---

## Deprecated Tables (DO NOT USE)

- `iam.org_user_roles` → Use `iam.user_assignments`
- `iam.location_user_roles` → Use `iam.user_assignments`
- `project.project_user_roles` → Use `iam.user_assignments`
- `project.project_managers` → Use `iam.user_assignments`

---

## Database Access

**Via MCP (Recommended):**
```
"Show me all assignments for user 19"
"List projects at location 6"
"What roles does user 19 have?"
```

**Direct Connection (Read-only):**
- Host: `appdb.cdwmaay8wkw4.us-east-2.rds.amazonaws.com`
- Database: `appdb`
- Port: 5432
- SSL: Required

---

**Last Updated:** 2025-10-27