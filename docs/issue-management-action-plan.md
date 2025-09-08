# Issue Management Implementation Action Plan

## 1. Database Schema Updates Required

### A. Create Template System Tables
```sql
-- Issue Templates Table
CREATE TABLE project.issue_templates (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100) NOT NULL,
    detail_category VARCHAR(100),
    default_priority VARCHAR(50),
    default_severity VARCHAR(50),
    default_trade VARCHAR(100),
    default_discipline VARCHAR(100),
    template_fields JSONB, -- Store custom field definitions
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_issue_templates_org FOREIGN KEY (org_id) REFERENCES iam.organizations(id)
);

-- Issue Categories Table
CREATE TABLE project.issue_categories (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL,
    category VARCHAR(100) NOT NULL,
    detail_category VARCHAR(100),
    parent_category_id BIGINT, -- For hierarchical categories
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_issue_categories_org FOREIGN KEY (org_id) REFERENCES iam.organizations(id),
    CONSTRAINT fk_issue_categories_parent FOREIGN KEY (parent_category_id) REFERENCES project.issue_categories(id)
);
```

### B. Update Issues Table
```sql
ALTER TABLE project.issues 
ADD COLUMN template_id BIGINT,
ADD COLUMN category VARCHAR(100),
ADD COLUMN detail_category VARCHAR(100),
ADD COLUMN root_cause TEXT,
ADD COLUMN discipline VARCHAR(100),
ADD COLUMN location_building VARCHAR(100),
ADD COLUMN location_level VARCHAR(50),
ADD COLUMN location_room VARCHAR(100),
ADD COLUMN location_x_coordinate DECIMAL(10,4),
ADD COLUMN location_y_coordinate DECIMAL(10,4),
ADD COLUMN distribution_list TEXT[], -- Array of emails
ADD CONSTRAINT fk_issues_template FOREIGN KEY (template_id) REFERENCES project.issue_templates(id);

-- Update priority constraint to include 'planned'
ALTER TABLE project.issues 
DROP CONSTRAINT issues_priority_check,
ADD CONSTRAINT issues_priority_check 
CHECK (priority IN ('critical', 'high', 'medium', 'low', 'planned'));

-- Update severity constraint to match API
ALTER TABLE project.issues 
DROP CONSTRAINT issues_severity_check,
ADD CONSTRAINT issues_severity_check 
CHECK (severity IN ('blocking', 'major', 'minor', 'cosmetic'));
```

## 2. API Request Body Mapping

### Field Mapping Strategy:
```
API Field                 → Database Field
----------------------------------------------
project_id               → project_id
location_id              → (from project.location_id)
org_id                   → (from project.org_id)
template_id              → template_id
category                 → category
detail_category          → detail_category
title                    → title
description              → description
priority                 → priority
severity                 → severity
root_cause               → root_cause
location.description     → location_description
location.coordinates.x   → location_x_coordinate
location.coordinates.y   → location_y_coordinate
location.building        → location_building
location.level           → location_level
location.room            → location_room
discipline               → discipline
trade                    → trade_type
assigned_to              → assigned_to
due_date                 → due_date
distribution_list        → distribution_list
```

## 3. Implementation Steps

### Phase 1: Database Schema Updates
1. Create migration script for new tables (issue_templates, issue_categories)
2. Create migration script to alter issues table with new columns
3. Test migrations on development database
4. Create rollback scripts

### Phase 2: Models & Types
1. Create Go models for IssueTemplate and IssueCategory
2. Update Issue model with new fields
3. Create request/response DTOs:
   - CreateIssueRequest
   - UpdateIssueRequest
   - IssueResponse
   - IssueTemplateResponse

### Phase 3: Repository Layer
1. Create IssueTemplateRepository interface & implementation
2. Create IssueCategoryRepository interface & implementation
3. Update IssueRepository with new CRUD operations
4. Implement template-based issue creation

### Phase 4: Lambda Functions
1. Create issue-template-management Lambda
2. Create issue-management Lambda
3. Implement handlers for:
   - CRUD operations for templates
   - CRUD operations for categories
   - CRUD operations for issues
   - Template-based issue creation
   - Distribution list notifications

### Phase 5: Infrastructure
1. Update CDK stack with new Lambda functions
2. Configure API Gateway routes:
   - /templates (GET, POST, PUT, DELETE)
   - /categories (GET, POST, PUT, DELETE)
   - /issues (GET, POST, PUT, DELETE)
   - /issues/{id}/comments (GET, POST)
   - /issues/{id}/attachments (POST, DELETE)
3. Set up IAM roles and permissions
4. Configure SES for distribution list emails

### Phase 6: Testing & Validation
1. Unit tests for repositories
2. Integration tests for Lambda handlers
3. End-to-end API testing
4. Load testing for performance

## 4. API Endpoints Design

### Template Management
- `GET /templates` - List all templates for org
- `GET /templates/{id}` - Get specific template
- `POST /templates` - Create new template
- `PUT /templates/{id}` - Update template
- `DELETE /templates/{id}` - Soft delete template

### Category Management
- `GET /categories` - List all categories for org
- `GET /categories/{id}` - Get specific category
- `POST /categories` - Create new category
- `PUT /categories/{id}` - Update category
- `DELETE /categories/{id}` - Soft delete category

### Issue Management
- `GET /projects/{projectId}/issues` - List issues for project
- `GET /issues/{id}` - Get specific issue
- `POST /projects/{projectId}/issues` - Create new issue
- `PUT /issues/{id}` - Update issue
- `DELETE /issues/{id}` - Soft delete issue
- `PATCH /issues/{id}/status` - Update issue status
- `POST /issues/{id}/comments` - Add comment
- `POST /issues/{id}/attachments` - Upload attachment

## 5. Key Considerations

### A. Template System
- Templates should be org-specific
- Templates can define default values for fields
- Templates can have custom fields (stored in JSONB)
- Categories can be hierarchical (parent-child)

### B. Location Handling
- Support both coordinate systems (x,y for floor plans, lat/long for maps)
- Building/Level/Room provides hierarchical location context
- Location description remains as free text field

### C. Distribution List
- Store as array of emails in database
- Implement async notification system
- Track notification status separately

### D. Backward Compatibility
- Keep existing fields for gradual migration
- Support both old and new field names initially
- Provide data migration scripts

## 6. Priority Order

1. **High Priority (Week 1)**
   - Database schema updates
   - Basic issue CRUD operations
   - Template system implementation

2. **Medium Priority (Week 2)**
   - Category management
   - Distribution list notifications
   - Comment system

3. **Low Priority (Week 3)**
   - Attachment handling
   - Advanced filtering/search
   - Reporting features

## 7. Risk Mitigation

- **Data Migration**: Create comprehensive backup before schema changes
- **Performance**: Index new columns appropriately
- **Validation**: Implement strict input validation at API layer
- **Security**: Ensure proper authorization for all operations
- **Monitoring**: Add CloudWatch metrics for all new endpoints