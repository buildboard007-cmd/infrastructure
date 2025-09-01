-- Construction Management Database Creation Scripts for PostgreSQL
-- Execute these scripts in order

CREATE SCHEMA iam;
CREATE SCHEMA project;

-- Set search path to include both schemas
SET search_path TO iam, project, public;

-- 2. IAM Schema Tables (Identity and Access Management)

-- Organizations Table
CREATE TABLE iam.organizations (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    company_type VARCHAR(50) NOT NULL CHECK (company_type IN ('general_contractor', 'subcontractor', 'architect', 'owner', 'consultant')),
    license_number VARCHAR(100),
    address TEXT,
    phone VARCHAR(20),
    email VARCHAR(255),
    website VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'pending_setup' CHECK (status IN ('active', 'inactive', 'pending_setup', 'suspended')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_organizations_company_type ON iam.organizations (company_type);
CREATE INDEX idx_organizations_status ON iam.organizations (status);
CREATE INDEX idx_organizations_is_deleted ON iam.organizations (is_deleted);

-- Locations Table
CREATE TABLE iam.locations (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    location_type VARCHAR(50) NOT NULL DEFAULT 'office' CHECK (location_type IN ('office', 'warehouse', 'job_site', 'yard')),
    address TEXT,
    city VARCHAR(100),
    state VARCHAR(50),
    zip_code VARCHAR(20),
    country VARCHAR(100) DEFAULT 'USA',
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'under_construction', 'closed')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_locations_org FOREIGN KEY (org_id) REFERENCES iam.organizations(id)
);

CREATE INDEX idx_locations_org_id ON iam.locations (org_id);
CREATE INDEX idx_locations_type_status ON iam.locations (location_type, status);
CREATE INDEX idx_locations_is_deleted ON iam.locations (is_deleted);

-- Users Table
CREATE TABLE iam.users (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL,
    cognito_id VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    phone VARCHAR(20),
    mobile VARCHAR(20),
    job_title VARCHAR(100),
    employee_id VARCHAR(50),
    avatar_url VARCHAR(500),
    current_location_id BIGINT,
    is_super_admin BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('active', 'inactive', 'pending', 'pending_org_setup', 'suspended')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_users_org FOREIGN KEY (org_id) REFERENCES iam.organizations(id),
    CONSTRAINT fk_users_current_location FOREIGN KEY (current_location_id) REFERENCES iam.locations(id)
);

CREATE INDEX idx_users_org_id ON iam.users (org_id);
CREATE INDEX idx_users_cognito_id ON iam.users (cognito_id);
CREATE INDEX idx_users_email ON iam.users (email);
CREATE INDEX idx_users_employee_id ON iam.users (employee_id);
CREATE INDEX idx_users_status ON iam.users (status);
CREATE INDEX idx_users_current_location ON iam.users (current_location_id);
CREATE INDEX idx_users_is_deleted ON iam.users (is_deleted);

-- User Location Access Table
CREATE TABLE iam.user_location_access (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    location_id BIGINT NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_user_location_access_user FOREIGN KEY (user_id) REFERENCES iam.users(id),
    CONSTRAINT fk_user_location_access_location FOREIGN KEY (location_id) REFERENCES iam.locations(id)
);

CREATE INDEX idx_user_location_access_user_id ON iam.user_location_access (user_id);
CREATE INDEX idx_user_location_access_location_id ON iam.user_location_access (location_id);
CREATE INDEX idx_user_location_access_is_deleted ON iam.user_location_access (is_deleted);

-- Roles Table
CREATE TABLE iam.roles (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NULL, -- NULL for system/standard roles
    name VARCHAR(100) NOT NULL,
    description TEXT,
    role_type VARCHAR(50) NOT NULL DEFAULT 'custom' CHECK (role_type IN ('system', 'custom')),
    construction_role_category VARCHAR(50) NOT NULL CHECK (construction_role_category IN ('management', 'field', 'office', 'external', 'admin')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_roles_org FOREIGN KEY (org_id) REFERENCES iam.organizations(id)
);

CREATE INDEX idx_roles_org_id ON iam.roles (org_id);
CREATE INDEX idx_roles_type_category ON iam.roles (role_type, construction_role_category);
CREATE INDEX idx_roles_is_deleted ON iam.roles (is_deleted);

-- Permissions Table
CREATE TABLE iam.permissions (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(100) NOT NULL,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    permission_type VARCHAR(50) NOT NULL DEFAULT 'system' CHECK (permission_type IN ('system', 'custom')),
    module VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50),
    action_type VARCHAR(50),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_permissions_code ON iam.permissions (code);
CREATE INDEX idx_permissions_module ON iam.permissions (module);
CREATE INDEX idx_permissions_resource_action ON iam.permissions (resource_type, action_type);
CREATE INDEX idx_permissions_type ON iam.permissions (permission_type);
CREATE INDEX idx_permissions_is_deleted ON iam.permissions (is_deleted);

-- Role Permissions Table
CREATE TABLE iam.role_permissions (
    role_id BIGINT NOT NULL,
    permission_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (role_id, permission_id),
    CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id) REFERENCES iam.roles(id),
    CONSTRAINT fk_role_permissions_permission FOREIGN KEY (permission_id) REFERENCES iam.permissions(id)
);

CREATE INDEX idx_role_permissions_is_deleted ON iam.role_permissions (is_deleted);

-- User Organization Roles Table
CREATE TABLE iam.user_organization_roles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_user_org_roles_user FOREIGN KEY (user_id) REFERENCES iam.users(id),
    CONSTRAINT fk_user_org_roles_role FOREIGN KEY (role_id) REFERENCES iam.roles(id)
);

CREATE INDEX idx_user_org_roles_user_id ON iam.user_organization_roles (user_id);
CREATE INDEX idx_user_org_roles_role_id ON iam.user_organization_roles (role_id);
CREATE INDEX idx_user_org_roles_is_deleted ON iam.user_organization_roles (is_deleted);

-- 3. PROJECT Schema Tables (Construction Management)

-- Projects Table
CREATE TABLE project.projects (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL,
    location_id BIGINT NOT NULL,
    project_number VARCHAR(50),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    project_type VARCHAR(50) NOT NULL CHECK (project_type IN ('commercial', 'residential', 'industrial', 'infrastructure', 'renovation')),
    project_phase VARCHAR(50) NOT NULL DEFAULT 'pre_construction' CHECK (project_phase IN ('pre_construction', 'design', 'permitting', 'construction', 'closeout', 'warranty')),
    start_date DATE,
    end_date DATE,
    actual_start_date DATE,
    actual_end_date DATE,
    budget DECIMAL(15,2),
    contract_value DECIMAL(15,2),
    address TEXT,
    city VARCHAR(100),
    state VARCHAR(50),
    zip_code VARCHAR(20),
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'on_hold', 'completed', 'cancelled')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_projects_org FOREIGN KEY (org_id) REFERENCES iam.organizations(id),
    CONSTRAINT fk_projects_location FOREIGN KEY (location_id) REFERENCES iam.locations(id)
);

CREATE INDEX idx_projects_org_id ON project.projects (org_id);
CREATE INDEX idx_projects_location_id ON project.projects (location_id);
CREATE INDEX idx_projects_number ON project.projects (project_number);
CREATE INDEX idx_projects_type_phase ON project.projects (project_type, project_phase);
CREATE INDEX idx_projects_dates ON project.projects (start_date, end_date);
CREATE INDEX idx_projects_status ON project.projects (status);
CREATE INDEX idx_projects_is_deleted ON project.projects (is_deleted);

-- Project User Roles Table
CREATE TABLE project.project_user_roles (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    trade_type VARCHAR(100),
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    start_date DATE,
    end_date DATE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_project_user_roles_project FOREIGN KEY (project_id) REFERENCES project.projects(id),
    CONSTRAINT fk_project_user_roles_user FOREIGN KEY (user_id) REFERENCES iam.users(id),
    CONSTRAINT fk_project_user_roles_role FOREIGN KEY (role_id) REFERENCES iam.roles(id)
);

CREATE INDEX idx_project_user_roles_project_id ON project.project_user_roles (project_id);
CREATE INDEX idx_project_user_roles_user_id ON project.project_user_roles (user_id);
CREATE INDEX idx_project_user_roles_role_id ON project.project_user_roles (role_id);
CREATE INDEX idx_project_user_roles_trade ON project.project_user_roles (trade_type);
CREATE INDEX idx_project_user_roles_primary ON project.project_user_roles (is_primary);
CREATE INDEX idx_project_user_roles_is_deleted ON project.project_user_roles (is_deleted);

-- RFIs Table
CREATE TABLE project.rfis (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL,
    rfi_number VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    question TEXT NOT NULL,
    location_description VARCHAR(255),
    drawing_reference VARCHAR(255),
    specification_reference VARCHAR(255),
    priority VARCHAR(50) NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    status VARCHAR(50) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'in_review', 'responded', 'closed', 'cancelled')),
    submitted_by BIGINT NOT NULL,
    assigned_to BIGINT,
    submitted_date TIMESTAMP NULL,
    due_date TIMESTAMP NULL,
    response_date TIMESTAMP NULL,
    response TEXT,
    response_by BIGINT,
    cost_impact DECIMAL(15,2) DEFAULT 0.00,
    schedule_impact_days INTEGER DEFAULT 0,
    trade_type VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_rfis_project FOREIGN KEY (project_id) REFERENCES project.projects(id),
    CONSTRAINT fk_rfis_submitted_by FOREIGN KEY (submitted_by) REFERENCES iam.users(id),
    CONSTRAINT fk_rfis_assigned_to FOREIGN KEY (assigned_to) REFERENCES iam.users(id),
    CONSTRAINT fk_rfis_response_by FOREIGN KEY (response_by) REFERENCES iam.users(id)
);

CREATE INDEX idx_rfis_project_id ON project.rfis (project_id);
CREATE INDEX idx_rfis_number ON project.rfis (rfi_number);
CREATE INDEX idx_rfis_status ON project.rfis (status);
CREATE INDEX idx_rfis_priority ON project.rfis (priority);
CREATE INDEX idx_rfis_submitted_by ON project.rfis (submitted_by);
CREATE INDEX idx_rfis_assigned_to ON project.rfis (assigned_to);
CREATE INDEX idx_rfis_due_date ON project.rfis (due_date);
CREATE INDEX idx_rfis_trade_type ON project.rfis (trade_type);
CREATE INDEX idx_rfis_is_deleted ON project.rfis (is_deleted);

-- RFI Attachments Table
CREATE TABLE project.rfi_attachments (
    id BIGSERIAL PRIMARY KEY,
    rfi_id BIGINT NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT,
    file_type VARCHAR(50),
    uploaded_by BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_rfi_attachments_rfi FOREIGN KEY (rfi_id) REFERENCES project.rfis(id),
    CONSTRAINT fk_rfi_attachments_uploaded_by FOREIGN KEY (uploaded_by) REFERENCES iam.users(id)
);

CREATE INDEX idx_rfi_attachments_rfi_id ON project.rfi_attachments (rfi_id);
CREATE INDEX idx_rfi_attachments_is_deleted ON project.rfi_attachments (is_deleted);

-- RFI Comments Table
CREATE TABLE project.rfi_comments (
    id BIGSERIAL PRIMARY KEY,
    rfi_id BIGINT NOT NULL,
    comment TEXT NOT NULL,
    comment_type VARCHAR(50) NOT NULL DEFAULT 'comment' CHECK (comment_type IN ('comment', 'status_change', 'assignment', 'response')),
    previous_value VARCHAR(255),
    new_value VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_rfi_comments_rfi FOREIGN KEY (rfi_id) REFERENCES project.rfis(id),
    CONSTRAINT fk_rfi_comments_created_by FOREIGN KEY (created_by) REFERENCES iam.users(id)
);

CREATE INDEX idx_rfi_comments_rfi_id ON project.rfi_comments (rfi_id);
CREATE INDEX idx_rfi_comments_type ON project.rfi_comments (comment_type);
CREATE INDEX idx_rfi_comments_is_deleted ON project.rfi_comments (is_deleted);

-- Issues Table
CREATE TABLE project.issues (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL,
    issue_number VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    issue_type VARCHAR(50) NOT NULL DEFAULT 'general' CHECK (issue_type IN ('quality', 'safety', 'deficiency', 'punch_item', 'code_violation', 'general')),
    severity VARCHAR(50) NOT NULL DEFAULT 'minor' CHECK (severity IN ('minor', 'major', 'critical')),
    priority VARCHAR(50) NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    status VARCHAR(50) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'ready_for_review', 'closed', 'rejected', 'on_hold')),
    location_description VARCHAR(255),
    room_area VARCHAR(100),
    floor_level VARCHAR(50),
    trade_type VARCHAR(100),
    reported_by BIGINT NOT NULL,
    assigned_to BIGINT,
    assigned_company_id BIGINT,
    due_date DATE,
    closed_date TIMESTAMP NULL,
    cost_to_fix DECIMAL(15,2) DEFAULT 0.00,
    drawing_reference VARCHAR(255),
    specification_reference VARCHAR(255),
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_issues_project FOREIGN KEY (project_id) REFERENCES project.projects(id),
    CONSTRAINT fk_issues_reported_by FOREIGN KEY (reported_by) REFERENCES iam.users(id),
    CONSTRAINT fk_issues_assigned_to FOREIGN KEY (assigned_to) REFERENCES iam.users(id),
    CONSTRAINT fk_issues_assigned_company FOREIGN KEY (assigned_company_id) REFERENCES iam.organizations(id)
);

CREATE INDEX idx_issues_project_id ON project.issues (project_id);
CREATE INDEX idx_issues_number ON project.issues (issue_number);
CREATE INDEX idx_issues_type_severity ON project.issues (issue_type, severity);
CREATE INDEX idx_issues_status ON project.issues (status);
CREATE INDEX idx_issues_priority ON project.issues (priority);
CREATE INDEX idx_issues_reported_by ON project.issues (reported_by);
CREATE INDEX idx_issues_assigned_to ON project.issues (assigned_to);
CREATE INDEX idx_issues_trade_type ON project.issues (trade_type);
CREATE INDEX idx_issues_due_date ON project.issues (due_date);
CREATE INDEX idx_issues_is_deleted ON project.issues (is_deleted);

-- Issue Attachments Table
CREATE TABLE project.issue_attachments (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT,
    file_type VARCHAR(50),
    attachment_type VARCHAR(50) NOT NULL DEFAULT 'before_photo' CHECK (attachment_type IN ('before_photo', 'after_photo', 'document', 'drawing_markup')),
    uploaded_by BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_issue_attachments_issue FOREIGN KEY (issue_id) REFERENCES project.issues(id),
    CONSTRAINT fk_issue_attachments_uploaded_by FOREIGN KEY (uploaded_by) REFERENCES iam.users(id)
);

CREATE INDEX idx_issue_attachments_issue_id ON project.issue_attachments (issue_id);
CREATE INDEX idx_issue_attachments_type ON project.issue_attachments (attachment_type);
CREATE INDEX idx_issue_attachments_is_deleted ON project.issue_attachments (is_deleted);

-- Issue Comments Table
CREATE TABLE project.issue_comments (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL,
    comment TEXT NOT NULL,
    comment_type VARCHAR(50) NOT NULL DEFAULT 'comment' CHECK (comment_type IN ('comment', 'status_change', 'assignment', 'resolution')),
    previous_value VARCHAR(255),
    new_value VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_issue_comments_issue FOREIGN KEY (issue_id) REFERENCES project.issues(id),
    CONSTRAINT fk_issue_comments_created_by FOREIGN KEY (created_by) REFERENCES iam.users(id)
);

CREATE INDEX idx_issue_comments_issue_id ON project.issue_comments (issue_id);
CREATE INDEX idx_issue_comments_type ON project.issue_comments (comment_type);
CREATE INDEX idx_issue_comments_is_deleted ON project.issue_comments (is_deleted);

-- Submittals Table
CREATE TABLE project.submittals (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL,
    submittal_number VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    submittal_type VARCHAR(50) NOT NULL CHECK (submittal_type IN ('shop_drawings', 'product_data', 'samples', 'design_mix', 'test_reports', 'certificates', 'operation_manuals', 'warranties')),
    specification_section VARCHAR(50),
    drawing_reference VARCHAR(255),
    trade_type VARCHAR(100),
    priority VARCHAR(50) NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    status VARCHAR(50) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'under_review', 'approved', 'approved_with_comments', 'rejected', 'resubmit_required')),
    revision_number INTEGER NOT NULL DEFAULT 1,
    submitted_by BIGINT NOT NULL,
    submitted_company_id BIGINT,
    reviewed_by BIGINT,
    submitted_date TIMESTAMP NULL,
    due_date TIMESTAMP NULL,
    reviewed_date TIMESTAMP NULL,
    approval_date TIMESTAMP NULL,
    review_comments TEXT,
    lead_time_days INTEGER,
    quantity_submitted INTEGER,
    unit_of_measure VARCHAR(20),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_submittals_project FOREIGN KEY (project_id) REFERENCES project.projects(id),
    CONSTRAINT fk_submittals_submitted_by FOREIGN KEY (submitted_by) REFERENCES iam.users(id),
    CONSTRAINT fk_submittals_submitted_company FOREIGN KEY (submitted_company_id) REFERENCES iam.organizations(id),
    CONSTRAINT fk_submittals_reviewed_by FOREIGN KEY (reviewed_by) REFERENCES iam.users(id)
);

CREATE INDEX idx_submittals_project_id ON project.submittals (project_id);
CREATE INDEX idx_submittals_number ON project.submittals (submittal_number);
CREATE INDEX idx_submittals_type ON project.submittals (submittal_type);
CREATE INDEX idx_submittals_status ON project.submittals (status);
CREATE INDEX idx_submittals_priority ON project.submittals (priority);
CREATE INDEX idx_submittals_submitted_by ON project.submittals (submitted_by);
CREATE INDEX idx_submittals_reviewed_by ON project.submittals (reviewed_by);
CREATE INDEX idx_submittals_spec_section ON project.submittals (specification_section);
CREATE INDEX idx_submittals_trade_type ON project.submittals (trade_type);
CREATE INDEX idx_submittals_due_date ON project.submittals (due_date);
CREATE INDEX idx_submittals_is_deleted ON project.submittals (is_deleted);

-- Submittal Items Table
CREATE TABLE project.submittal_items (
    id BIGSERIAL PRIMARY KEY,
    submittal_id BIGINT NOT NULL,
    item_number VARCHAR(50),
    item_description TEXT NOT NULL,
    manufacturer VARCHAR(255),
    model_number VARCHAR(100),
    quantity INTEGER,
    unit_price DECIMAL(15,2),
    total_price DECIMAL(15,2),
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'approved_with_comments')),
    comments TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_submittal_items_submittal FOREIGN KEY (submittal_id) REFERENCES project.submittals(id)
);

CREATE INDEX idx_submittal_items_submittal_id ON project.submittal_items (submittal_id);
CREATE INDEX idx_submittal_items_status ON project.submittal_items (status);
CREATE INDEX idx_submittal_items_is_deleted ON project.submittal_items (is_deleted);

-- Submittal Attachments Table
CREATE TABLE project.submittal_attachments (
    id BIGSERIAL PRIMARY KEY,
    submittal_id BIGINT NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT,
    file_type VARCHAR(50),
    attachment_type VARCHAR(50) NOT NULL DEFAULT 'other' CHECK (attachment_type IN ('shop_drawing', 'product_data', 'specification', 'sample_photo', 'certificate', 'test_report', 'other')),
    uploaded_by BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_submittal_attachments_submittal FOREIGN KEY (submittal_id) REFERENCES project.submittals(id),
    CONSTRAINT fk_submittal_attachments_uploaded_by FOREIGN KEY (uploaded_by) REFERENCES iam.users(id)
);

CREATE INDEX idx_submittal_attachments_submittal_id ON project.submittal_attachments (submittal_id);
CREATE INDEX idx_submittal_attachments_type ON project.submittal_attachments (attachment_type);
CREATE INDEX idx_submittal_attachments_is_deleted ON project.submittal_attachments (is_deleted);

-- Submittal Reviews Table
CREATE TABLE project.submittal_reviews (
    id BIGSERIAL PRIMARY KEY,
    submittal_id BIGINT NOT NULL,
    revision_number INTEGER NOT NULL,
    reviewer_id BIGINT NOT NULL,
    review_status VARCHAR(50) NOT NULL CHECK (review_status IN ('approved', 'approved_with_comments', 'rejected', 'resubmit_required')),
    review_comments TEXT,
    review_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_submittal_reviews_submittal FOREIGN KEY (submittal_id) REFERENCES project.submittals(id),
    CONSTRAINT fk_submittal_reviews_reviewer FOREIGN KEY (reviewer_id) REFERENCES iam.users(id)
);

CREATE INDEX idx_submittal_reviews_submittal_id ON project.submittal_reviews (submittal_id);
CREATE INDEX idx_submittal_reviews_reviewer_id ON project.submittal_reviews (reviewer_id);
CREATE INDEX idx_submittal_reviews_status ON project.submittal_reviews (review_status);
CREATE INDEX idx_submittal_reviews_is_deleted ON project.submittal_reviews (is_deleted);

-- Add function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$ LANGUAGE plpgsql;

-- Add triggers to auto-update updated_at for all tables
-- IAM Schema triggers
CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON iam.organizations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_locations_updated_at BEFORE UPDATE ON iam.locations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON iam.users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_location_access_updated_at BEFORE UPDATE ON iam.user_location_access FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_roles_updated_at BEFORE UPDATE ON iam.roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_permissions_updated_at BEFORE UPDATE ON iam.permissions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_role_permissions_updated_at BEFORE UPDATE ON iam.role_permissions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_organization_roles_updated_at BEFORE UPDATE ON iam.user_organization_roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Project Schema triggers
CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON project.projects FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_project_user_roles_updated_at BEFORE UPDATE ON project.project_user_roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_rfis_updated_at BEFORE UPDATE ON project.rfis FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_rfi_attachments_updated_at BEFORE UPDATE ON project.rfi_attachments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_rfi_comments_updated_at BEFORE UPDATE ON project.rfi_comments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_issues_updated_at BEFORE UPDATE ON project.issues FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_issue_attachments_updated_at BEFORE UPDATE ON project.issue_attachments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_issue_comments_updated_at BEFORE UPDATE ON project.issue_comments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_submittals_updated_at BEFORE UPDATE ON project.submittals FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_submittal_items_updated_at BEFORE UPDATE ON project.submittal_items FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_submittal_attachments_updated_at BEFORE UPDATE ON project.submittal_attachments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_submittal_reviews_updated_at BEFORE UPDATE ON project.submittal_reviews FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();