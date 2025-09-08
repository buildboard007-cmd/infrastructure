# Construction Issue Management System - Comprehensive Action Plan
*Inspired by industry leaders: Procore, Bluebeam, PlanGrid, Autodesk Build*

## 1. Core Issue Types in Construction

### A. Issue Categories Hierarchy
```
Quality Issues
├── Defective Work
├── Non-Conformance
├── Material Defects
└── Workmanship Issues

Safety Issues  
├── Hazard Observations
├── Near Misses
├── Incidents/Accidents
└── PPE Violations

Punch List Items
├── Architectural Finishes
├── MEP Systems
├── Exterior Items
└── Site Work

Inspections
├── Pre-Pour Inspections
├── Rough-In Inspections
├── Final Inspections
└── Code Compliance

Warranty Items
├── 30-Day Items
├── 90-Day Items
├── 1-Year Items
└── Extended Warranty

Environmental
├── Erosion Control
├── Waste Management
├── Noise Violations
└── Dust Control
```

## 2. Enhanced Database Schema

### A. Core Tables
```sql
-- Issue Types with Construction Focus
CREATE TABLE project.issue_types (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL,
    code VARCHAR(20) NOT NULL UNIQUE, -- QC001, SF002, PL003
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL CHECK (category IN (
        'quality', 'safety', 'punch_list', 'inspection', 
        'warranty', 'environmental', 'coordination'
    )),
    subcategory VARCHAR(100),
    requires_photo BOOLEAN DEFAULT TRUE,
    requires_root_cause BOOLEAN DEFAULT FALSE,
    requires_corrective_action BOOLEAN DEFAULT TRUE,
    auto_assign_trade BOOLEAN DEFAULT TRUE,
    default_priority VARCHAR(20),
    default_due_days INTEGER, -- Days from creation
    color_code VARCHAR(7), -- Hex color for visual identification
    icon VARCHAR(50), -- Icon identifier for UI
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    CONSTRAINT fk_issue_types_org FOREIGN KEY (org_id) REFERENCES iam.organizations(id)
);

-- Enhanced Issues Table for Construction
CREATE TABLE project.issues (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL,
    issue_number VARCHAR(50) NOT NULL, -- AUTO-GENERATED: PRJ-QC-0001
    issue_type_id BIGINT NOT NULL,
    
    -- Basic Information
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    
    -- Classification
    category VARCHAR(50) NOT NULL,
    subcategory VARCHAR(100),
    severity VARCHAR(20) CHECK (severity IN ('critical', 'major', 'moderate', 'minor', 'cosmetic')),
    priority VARCHAR(20) CHECK (priority IN ('emergency', 'high', 'medium', 'low', 'planned')),
    
    -- Location Information (Enhanced)
    location_type VARCHAR(50) CHECK (location_type IN ('drawing', 'model', 'site', 'gps')),
    drawing_id BIGINT, -- Reference to project drawings
    drawing_sheet_number VARCHAR(50),
    drawing_revision VARCHAR(20),
    location_x DECIMAL(10,4), -- X coordinate on drawing
    location_y DECIMAL(10,4), -- Y coordinate on drawing
    location_z DECIMAL(10,4), -- For 3D models
    gps_latitude DECIMAL(10,8),
    gps_longitude DECIMAL(11,8),
    building_name VARCHAR(100),
    floor_level VARCHAR(50),
    grid_line VARCHAR(50), -- Construction grid reference
    room_number VARCHAR(50),
    area_name VARCHAR(100),
    location_description TEXT,
    
    -- Trade & Responsibility
    responsible_trade VARCHAR(100), -- Electrical, Plumbing, HVAC, etc.
    responsible_contractor_id BIGINT, -- Subcontractor company
    assigned_to_user_id BIGINT,
    assigned_to_company_id BIGINT,
    watchers TEXT[], -- Array of user IDs
    
    -- Root Cause Analysis
    root_cause_category VARCHAR(100) CHECK (root_cause_category IN (
        'design_error', 'construction_error', 'material_defect', 
        'coordination_issue', 'weather_related', 'equipment_failure',
        'human_error', 'process_failure', 'external_factor'
    )),
    root_cause_description TEXT,
    
    -- Corrective Actions
    corrective_action_required BOOLEAN DEFAULT TRUE,
    corrective_action_description TEXT,
    prevention_measures TEXT,
    
    -- Impact Assessment
    cost_impact DECIMAL(15,2),
    schedule_impact_days INTEGER,
    safety_risk_level VARCHAR(20) CHECK (safety_risk_level IN ('high', 'medium', 'low', 'none')),
    quality_impact_level VARCHAR(20) CHECK (quality_impact_level IN ('high', 'medium', 'low', 'none')),
    
    -- Timing
    identified_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    due_date TIMESTAMP,
    started_date TIMESTAMP,
    ready_for_inspection_date TIMESTAMP,
    verified_date TIMESTAMP,
    closed_date TIMESTAMP,
    
    -- Status & Workflow
    status VARCHAR(50) NOT NULL DEFAULT 'open' CHECK (status IN (
        'draft', 'open', 'acknowledged', 'in_progress', 
        'ready_for_inspection', 'verified', 'closed', 
        'disputed', 'void', 'deferred'
    )),
    workflow_step VARCHAR(100),
    approval_status VARCHAR(50),
    
    -- Verification
    verified_by_user_id BIGINT,
    verification_method VARCHAR(50) CHECK (verification_method IN (
        'visual_inspection', 'testing', 'measurement', 'documentation_review'
    )),
    verification_notes TEXT,
    
    -- References
    rfi_reference VARCHAR(100), -- Related RFI numbers
    change_order_reference VARCHAR(100),
    specification_section VARCHAR(50),
    drawing_detail_reference VARCHAR(100),
    submittal_reference VARCHAR(100),
    
    -- Tags & Custom Fields
    tags TEXT[],
    custom_fields JSONB,
    
    -- Weather Conditions (for exterior work)
    weather_condition VARCHAR(50),
    temperature_fahrenheit INTEGER,
    
    -- Distribution & Notifications
    distribution_list TEXT[], -- Email addresses
    notification_sent_at TIMESTAMP,
    reminder_sent_count INTEGER DEFAULT 0,
    
    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    
    CONSTRAINT fk_issues_project FOREIGN KEY (project_id) REFERENCES project.projects(id),
    CONSTRAINT fk_issues_type FOREIGN KEY (issue_type_id) REFERENCES project.issue_types(id),
    CONSTRAINT fk_issues_contractor FOREIGN KEY (responsible_contractor_id) REFERENCES iam.organizations(id),
    CONSTRAINT fk_issues_assigned_user FOREIGN KEY (assigned_to_user_id) REFERENCES iam.users(id),
    CONSTRAINT fk_issues_assigned_company FOREIGN KEY (assigned_to_company_id) REFERENCES iam.organizations(id),
    CONSTRAINT fk_issues_verified_by FOREIGN KEY (verified_by_user_id) REFERENCES iam.users(id)
);

-- Issue Locations (for multiple location tracking)
CREATE TABLE project.issue_locations (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL,
    location_name VARCHAR(255),
    location_type VARCHAR(50),
    coordinates JSONB, -- Flexible coordinate storage
    metadata JSONB, -- Additional location data
    is_primary BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_issue_locations_issue FOREIGN KEY (issue_id) REFERENCES project.issues(id)
);

-- Issue Checklists (for inspection items)
CREATE TABLE project.issue_checklists (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL,
    checklist_item TEXT NOT NULL,
    is_completed BOOLEAN DEFAULT FALSE,
    completed_by BIGINT,
    completed_at TIMESTAMP,
    notes TEXT,
    display_order INTEGER,
    CONSTRAINT fk_issue_checklists_issue FOREIGN KEY (issue_id) REFERENCES project.issues(id),
    CONSTRAINT fk_issue_checklists_completed_by FOREIGN KEY (completed_by) REFERENCES iam.users(id)
);

-- Issue Attachments (Enhanced)
CREATE TABLE project.issue_attachments (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL,
    attachment_type VARCHAR(50) NOT NULL CHECK (attachment_type IN (
        'before_photo', 'progress_photo', 'after_photo', 
        'drawing_markup', 'document', 'video', 'audio_note'
    )),
    file_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT,
    mime_type VARCHAR(100),
    thumbnail_path VARCHAR(500), -- For image previews
    
    -- Photo metadata
    taken_at TIMESTAMP,
    device_info VARCHAR(255),
    gps_latitude DECIMAL(10,8),
    gps_longitude DECIMAL(11,8),
    
    -- Drawing markup data
    markup_data JSONB, -- Store annotation data
    base_drawing_version VARCHAR(50),
    
    is_primary BOOLEAN DEFAULT FALSE,
    display_order INTEGER,
    
    uploaded_by BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    
    CONSTRAINT fk_issue_attachments_issue FOREIGN KEY (issue_id) REFERENCES project.issues(id),
    CONSTRAINT fk_issue_attachments_uploaded_by FOREIGN KEY (uploaded_by) REFERENCES iam.users(id)
);

-- Issue History (Detailed audit trail)
CREATE TABLE project.issue_history (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL,
    action VARCHAR(100) NOT NULL, -- created, updated, status_changed, assigned, etc.
    field_name VARCHAR(100),
    old_value TEXT,
    new_value TEXT,
    change_reason TEXT,
    performed_by BIGINT NOT NULL,
    performed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ip_address INET,
    user_agent TEXT,
    CONSTRAINT fk_issue_history_issue FOREIGN KEY (issue_id) REFERENCES project.issues(id),
    CONSTRAINT fk_issue_history_user FOREIGN KEY (performed_by) REFERENCES iam.users(id)
);

-- Issue Templates (for recurring issues)
CREATE TABLE project.issue_templates (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL,
    project_id BIGINT, -- Null for org-wide templates
    template_name VARCHAR(255) NOT NULL,
    issue_type_id BIGINT NOT NULL,
    
    -- Pre-filled fields
    default_title VARCHAR(255),
    default_description TEXT,
    default_priority VARCHAR(20),
    default_severity VARCHAR(20),
    default_trade VARCHAR(100),
    default_due_days INTEGER,
    
    -- Checklist items
    checklist_items JSONB,
    
    -- Custom fields definition
    custom_fields_schema JSONB,
    
    usage_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMP,
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    
    CONSTRAINT fk_issue_templates_org FOREIGN KEY (org_id) REFERENCES iam.organizations(id),
    CONSTRAINT fk_issue_templates_project FOREIGN KEY (project_id) REFERENCES project.projects(id),
    CONSTRAINT fk_issue_templates_type FOREIGN KEY (issue_type_id) REFERENCES project.issue_types(id)
);

-- Issue Links (for related issues)
CREATE TABLE project.issue_links (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL,
    linked_issue_id BIGINT NOT NULL,
    link_type VARCHAR(50) NOT NULL CHECK (link_type IN (
        'relates_to', 'blocks', 'is_blocked_by', 
        'duplicates', 'is_duplicated_by', 'causes', 'is_caused_by'
    )),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    CONSTRAINT fk_issue_links_issue FOREIGN KEY (issue_id) REFERENCES project.issues(id),
    CONSTRAINT fk_issue_links_linked FOREIGN KEY (linked_issue_id) REFERENCES project.issues(id),
    CONSTRAINT fk_issue_links_created_by FOREIGN KEY (created_by) REFERENCES iam.users(id)
);

-- Ball In Court Tracking (responsibility tracking)
CREATE TABLE project.issue_ball_in_court (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL,
    responsible_party_type VARCHAR(50) CHECK (responsible_party_type IN ('user', 'company', 'trade')),
    responsible_party_id BIGINT,
    responsible_trade VARCHAR(100),
    assigned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    acknowledged_at TIMESTAMP,
    completed_at TIMESTAMP,
    escalation_level INTEGER DEFAULT 0,
    next_escalation_at TIMESTAMP,
    CONSTRAINT fk_issue_ball_in_court_issue FOREIGN KEY (issue_id) REFERENCES project.issues(id)
);
```

### B. Supporting Tables
```sql
-- Trade Responsibility Matrix
CREATE TABLE project.trade_responsibility_matrix (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL,
    trade_name VARCHAR(100) NOT NULL,
    contractor_id BIGINT NOT NULL,
    is_prime BOOLEAN DEFAULT FALSE,
    contact_user_id BIGINT,
    escalation_user_id BIGINT,
    auto_assign_categories TEXT[],
    CONSTRAINT fk_trade_matrix_project FOREIGN KEY (project_id) REFERENCES project.projects(id),
    CONSTRAINT fk_trade_matrix_contractor FOREIGN KEY (contractor_id) REFERENCES iam.organizations(id)
);

-- Issue Cost Tracking
CREATE TABLE project.issue_costs (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL,
    cost_type VARCHAR(50) CHECK (cost_type IN ('labor', 'material', 'equipment', 'other')),
    description TEXT,
    quantity DECIMAL(10,2),
    unit_cost DECIMAL(15,2),
    total_cost DECIMAL(15,2),
    is_back_charge BOOLEAN DEFAULT FALSE,
    back_charge_to BIGINT, -- Company ID
    approved_by BIGINT,
    approved_at TIMESTAMP,
    CONSTRAINT fk_issue_costs_issue FOREIGN KEY (issue_id) REFERENCES project.issues(id),
    CONSTRAINT fk_issue_costs_company FOREIGN KEY (back_charge_to) REFERENCES iam.organizations(id)
);
```

## 3. API Structure

### A. Core Endpoints
```
/api/v1/projects/{projectId}/issues
  GET    - List issues with advanced filtering
  POST   - Create new issue
  
/api/v1/issues/{issueId}
  GET    - Get issue details
  PUT    - Update issue
  DELETE - Soft delete issue
  
/api/v1/issues/{issueId}/status
  PATCH  - Update status with workflow validation
  
/api/v1/issues/{issueId}/assign
  POST   - Assign/reassign issue
  
/api/v1/issues/{issueId}/verify
  POST   - Verify and close issue
  
/api/v1/issues/{issueId}/attachments
  GET    - List attachments
  POST   - Upload attachment
  DELETE - Remove attachment
  
/api/v1/issues/{issueId}/comments
  GET    - Get comments/history
  POST   - Add comment
  
/api/v1/issues/{issueId}/checklist
  GET    - Get checklist items
  PUT    - Update checklist
  
/api/v1/issues/{issueId}/costs
  GET    - Get cost impacts
  POST   - Add cost item
  
/api/v1/issues/{issueId}/links
  GET    - Get linked issues
  POST   - Link issues
  DELETE - Unlink issues
```

### B. Bulk Operations
```
/api/v1/issues/bulk/create
  POST   - Create multiple issues

/api/v1/issues/bulk/update
  PATCH  - Update multiple issues

/api/v1/issues/bulk/assign
  POST   - Bulk assignment
```

### C. Templates & Types
```
/api/v1/issue-types
  GET    - List issue types
  POST   - Create custom type
  
/api/v1/issue-templates
  GET    - List templates
  POST   - Create template
  PUT    - Update template
  
/api/v1/issue-templates/{templateId}/create-issue
  POST   - Create issue from template
```

### D. Reports & Analytics
```
/api/v1/issues/reports/summary
  GET    - Dashboard summary
  
/api/v1/issues/reports/by-trade
  GET    - Issues by trade
  
/api/v1/issues/reports/by-location
  GET    - Issues by location
  
/api/v1/issues/reports/aging
  GET    - Aging report
  
/api/v1/issues/reports/cost-impact
  GET    - Cost impact analysis
```

## 4. Key Features Implementation

### A. Auto-Assignment Logic
```go
// Based on:
1. Issue category → Trade mapping
2. Trade → Contractor mapping  
3. Location → Responsible party
4. Previous assignment patterns
5. Workload balancing
```

### B. Notification System
```go
// Triggers:
1. New issue created → Assigned party
2. Status change → Watchers + Creator
3. Due date approaching → Assigned party
4. Overdue → Escalation chain
5. Verified/Closed → All stakeholders
```

### C. Mobile App Considerations
```go
// Offline Support:
1. Queue issue creation
2. Cache photos locally
3. Sync when connected
4. Conflict resolution

// Quick Actions:
1. Photo + Quick issue
2. Voice-to-text notes
3. Barcode/QR scanning
4. GPS auto-location
```

### D. Drawing Integration
```go
// Features:
1. Pin issues on drawings
2. Markup tools
3. Measurement tools
4. Layer management
5. Version comparison
```

## 5. Workflow States

### Standard Issue Workflow
```
Draft → Open → Acknowledged → In Progress → Ready for Inspection → Verified → Closed
         ↓                         ↓                                    ↓
      Disputed                  On Hold                             Reopened
```

### Punch List Workflow
```
Identified → Assigned → In Progress → Completed → Verified → Accepted
                ↓            ↓                         ↓
            Not Started   Rejected                 Disputed
```

### Safety Issue Workflow
```
Reported → Under Investigation → Corrective Action → Verification → Closed
     ↓              ↓                    ↓                ↓
  Critical      Root Cause           Training         Follow-up
  (Immediate)    Analysis            Required         Required
```

## 6. Integration Points

### A. External Systems
- Document Management (S3, SharePoint)
- BIM Models (Autodesk, Revit)
- Scheduling (P6, MS Project)
- Accounting (Sage, QuickBooks)
- Email (SES, SendGrid)
- SMS (Twilio)

### B. Data Exchange
- Import from Excel/CSV
- Export to PDF reports
- API webhooks for third-party
- Drawing PDF overlay generation

## 7. Performance Optimization

### A. Database
```sql
-- Indexes for common queries
CREATE INDEX idx_issues_project_status ON project.issues (project_id, status);
CREATE INDEX idx_issues_assigned ON project.issues (assigned_to_user_id, status);
CREATE INDEX idx_issues_trade ON project.issues (responsible_trade, status);
CREATE INDEX idx_issues_due_date ON project.issues (due_date) WHERE status != 'closed';
CREATE INDEX idx_issues_location ON project.issues USING GIST (point(gps_latitude, gps_longitude));
CREATE INDEX idx_issues_created ON project.issues (created_at DESC);

-- Materialized view for dashboard
CREATE MATERIALIZED VIEW project.issue_summary AS
SELECT 
    project_id,
    COUNT(*) FILTER (WHERE status = 'open') as open_count,
    COUNT(*) FILTER (WHERE status = 'in_progress') as in_progress_count,
    COUNT(*) FILTER (WHERE status = 'closed') as closed_count,
    AVG(EXTRACT(epoch FROM (closed_date - identified_date))/86400)::INT as avg_resolution_days
FROM project.issues
GROUP BY project_id;
```

### B. Caching Strategy
- Redis for frequently accessed templates
- CloudFront for attachment delivery
- API Gateway caching for reports
- ElasticSearch for full-text search

## 8. Security & Compliance

### A. Access Control
```go
// Permission Levels:
1. View own issues
2. View project issues  
3. Create issues
4. Edit own issues
5. Edit all issues
6. Delete issues
7. Verify issues
8. Manage templates
```

### B. Audit Requirements
- Complete history tracking
- IP address logging
- File integrity checksums
- Data retention policies
- GDPR compliance

## 9. Implementation Phases

### Phase 1: Foundation (Week 1)
- [ ] Database schema creation
- [ ] Basic CRUD operations
- [ ] Simple issue creation/update
- [ ] File upload support

### Phase 2: Core Features (Week 2)
- [ ] Template system
- [ ] Auto-assignment logic
- [ ] Status workflow
- [ ] Email notifications
- [ ] Basic reporting

### Phase 3: Advanced Features (Week 3)
- [ ] Drawing integration
- [ ] Cost tracking
- [ ] Ball-in-court tracking
- [ ] Bulk operations
- [ ] Mobile API optimization

### Phase 4: Intelligence (Week 4)
- [ ] ML-based categorization
- [ ] Predictive due dates
- [ ] Risk scoring
- [ ] Pattern detection
- [ ] Smart notifications

### Phase 5: Polish (Week 5)
- [ ] Performance optimization
- [ ] Advanced reports
- [ ] Dashboard widgets
- [ ] Export capabilities
- [ ] Integration testing

## 10. Success Metrics

### KPIs to Track
1. Average resolution time
2. Issues per project phase
3. Cost impact per issue
4. First-time fix rate
5. Escalation frequency
6. User adoption rate
7. Mobile vs desktop usage
8. Photo attachment rate

## 11. Sample API Request/Response

### Create Issue Request
```json
{
  "project_id": 12345,
  "issue_type_id": 1,
  "title": "Concrete crack in slab at Grid B-4",
  "description": "Hairline crack observed in concrete slab...",
  "category": "quality",
  "subcategory": "concrete_defect",
  "priority": "high",
  "severity": "major",
  
  "location": {
    "type": "drawing",
    "drawing_sheet": "S-201",
    "coordinates": {"x": 450.5, "y": 320.2},
    "building": "Tower A",
    "floor": "Level 3",
    "grid_line": "B-4",
    "description": "Near column intersection"
  },
  
  "responsible_trade": "concrete",
  "assigned_to_company_id": 567,
  
  "root_cause_category": "construction_error",
  "root_cause_description": "Improper curing process",
  
  "due_date": "2024-01-15",
  "distribution_list": [
    "pm@contractor.com",
    "super@site.com"
  ],
  
  "attachments": [
    {
      "type": "before_photo",
      "file_id": "temp_upload_123"
    }
  ],
  
  "checklist": [
    "Measure crack width",
    "Document length",
    "Check for water infiltration"
  ]
}
```

### Issue Response
```json
{
  "id": 98765,
  "issue_number": "TWR-A-QC-0145",
  "project": {
    "id": 12345,
    "name": "Tower A Construction"
  },
  "status": "open",
  "created_at": "2024-01-08T10:30:00Z",
  
  "title": "Concrete crack in slab at Grid B-4",
  "description": "Hairline crack observed...",
  
  "category": "quality",
  "subcategory": "concrete_defect",
  "priority": "high",
  "severity": "major",
  
  "location": {
    "primary": {
      "type": "drawing",
      "drawing": {
        "id": 789,
        "sheet": "S-201",
        "revision": "C"
      },
      "coordinates": {"x": 450.5, "y": 320.2},
      "building": "Tower A",
      "floor": "Level 3",
      "grid_line": "B-4"
    }
  },
  
  "responsibility": {
    "trade": "concrete",
    "contractor": {
      "id": 567,
      "name": "ABC Concrete Inc"
    },
    "assigned_to": {
      "id": 234,
      "name": "John Smith",
      "email": "john@abcconcrete.com"
    }
  },
  
  "ball_in_court": {
    "current": "ABC Concrete Inc",
    "since": "2024-01-08T10:30:00Z",
    "acknowledged": false
  },
  
  "timeline": {
    "created": "2024-01-08T10:30:00Z",
    "due": "2024-01-15T17:00:00Z",
    "days_open": 0,
    "days_until_due": 7
  },
  
  "metrics": {
    "cost_impact": 5000.00,
    "schedule_impact_days": 2,
    "safety_risk": "low",
    "quality_impact": "high"
  },
  
  "attachments": [
    {
      "id": 111,
      "type": "before_photo",
      "url": "https://...",
      "thumbnail": "https://...",
      "uploaded_by": "Mike Jones",
      "uploaded_at": "2024-01-08T10:30:00Z"
    }
  ],
  
  "links": {
    "self": "/api/v1/issues/98765",
    "project": "/api/v1/projects/12345",
    "attachments": "/api/v1/issues/98765/attachments",
    "comments": "/api/v1/issues/98765/comments",
    "history": "/api/v1/issues/98765/history"
  }
}
```

## 12. Technology Stack Recommendations

### Backend
- Go + Lambda (current stack)
- PostgreSQL with PostGIS for spatial
- Redis for caching
- S3 for file storage
- ElasticSearch for search
- SQS for async processing

### Image Processing
- Lambda with ImageMagick layer
- Rekognition for AI detection
- Textract for document OCR

### Notifications
- SES for email
- SNS for push notifications
- Twilio for SMS

### Analytics
- CloudWatch for metrics
- QuickSight for dashboards
- Athena for ad-hoc queries