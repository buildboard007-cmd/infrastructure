-- Submittal Management Schema Update
-- Enhancing existing submittal tables to match API requirements

-- Update main submittals table to align with API requirements
ALTER TABLE project.submittals
ADD COLUMN IF NOT EXISTS org_id BIGINT REFERENCES iam.organizations(id),
ADD COLUMN IF NOT EXISTS location_id BIGINT,
ADD COLUMN IF NOT EXISTS package_name VARCHAR(200),
ADD COLUMN IF NOT EXISTS csi_division VARCHAR(2),
ADD COLUMN IF NOT EXISTS csi_section VARCHAR(10),
ADD COLUMN IF NOT EXISTS current_phase VARCHAR(50) DEFAULT 'preparation',
ADD COLUMN IF NOT EXISTS ball_in_court VARCHAR(50) DEFAULT 'contractor',
ADD COLUMN IF NOT EXISTS workflow_status VARCHAR(50) DEFAULT 'pending_submission',
ADD COLUMN IF NOT EXISTS assigned_to BIGINT,
ADD COLUMN IF NOT EXISTS reviewer BIGINT,
ADD COLUMN IF NOT EXISTS approver BIGINT,
ADD COLUMN IF NOT EXISTS required_approval_date TIMESTAMP,
ADD COLUMN IF NOT EXISTS fabrication_start_date TIMESTAMP,
ADD COLUMN IF NOT EXISTS installation_date TIMESTAMP,
ADD COLUMN IF NOT EXISTS delivery_tracking JSONB DEFAULT '{}',
ADD COLUMN IF NOT EXISTS team_assignments JSONB DEFAULT '{}',
ADD COLUMN IF NOT EXISTS linked_drawings JSONB DEFAULT '{"drawing_numbers": [], "drawing_revisions": [], "detail_references": []}',
ADD COLUMN IF NOT EXISTS references JSONB DEFAULT '{"specification_sections": [], "related_submittals": [], "related_rfis": [], "related_issues": []}',
ADD COLUMN IF NOT EXISTS procurement_log JSONB DEFAULT '{}',
ADD COLUMN IF NOT EXISTS approval_actions JSONB DEFAULT '{}',
ADD COLUMN IF NOT EXISTS distribution_list JSONB DEFAULT '[]',
ADD COLUMN IF NOT EXISTS notification_settings JSONB DEFAULT '{"notify_on_status_change": true, "notify_on_delivery_update": true, "notify_on_approval": true}',
ADD COLUMN IF NOT EXISTS tags JSONB DEFAULT '[]',
ADD COLUMN IF NOT EXISTS custom_fields JSONB DEFAULT '{}';

-- Update constraints for enhanced fields
ALTER TABLE project.submittals
DROP CONSTRAINT IF EXISTS check_submittal_type,
ADD CONSTRAINT check_submittal_type CHECK (submittal_type IN ('shop_drawings', 'product_data', 'samples', 'material_certificates', 'method_statements', 'test_reports', 'other'));

ALTER TABLE project.submittals
DROP CONSTRAINT IF EXISTS check_priority,
ADD CONSTRAINT check_priority CHECK (priority IN ('critical', 'high', 'medium', 'low'));

ALTER TABLE project.submittals
DROP CONSTRAINT IF EXISTS check_current_phase,
ADD CONSTRAINT check_current_phase CHECK (current_phase IN ('preparation', 'review', 'approval', 'fabrication', 'delivery', 'installation', 'completed'));

ALTER TABLE project.submittals
DROP CONSTRAINT IF EXISTS check_ball_in_court,
ADD CONSTRAINT check_ball_in_court CHECK (ball_in_court IN ('contractor', 'architect', 'engineer', 'owner', 'subcontractor', 'vendor'));

ALTER TABLE project.submittals
DROP CONSTRAINT IF EXISTS check_workflow_status,
ADD CONSTRAINT check_workflow_status CHECK (workflow_status IN ('pending_submission', 'under_review', 'approved', 'approved_as_noted', 'revise_resubmit', 'rejected', 'for_information_only'));

-- Create new submittal_history table for audit trail
CREATE TABLE IF NOT EXISTS project.submittal_history (
    id BIGSERIAL PRIMARY KEY,
    submittal_id BIGINT NOT NULL REFERENCES project.submittals(id) ON DELETE CASCADE,
    action VARCHAR(100) NOT NULL,
    field_name VARCHAR(100),
    old_value TEXT,
    new_value TEXT,
    comment TEXT,
    created_by BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create new submittal_notifications table
CREATE TABLE IF NOT EXISTS project.submittal_notifications (
    id BIGSERIAL PRIMARY KEY,
    submittal_id BIGINT NOT NULL REFERENCES project.submittals(id) ON DELETE CASCADE,
    recipient_email VARCHAR(255) NOT NULL,
    notification_type VARCHAR(50) NOT NULL CHECK (notification_type IN ('created', 'updated', 'status_changed', 'delivery_update', 'approval_required')),
    sent_at TIMESTAMP,
    delivery_status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (delivery_status IN ('pending', 'sent', 'delivered', 'failed')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Update submittal_reviews table constraints
ALTER TABLE project.submittal_reviews
DROP CONSTRAINT IF EXISTS check_review_status,
ADD CONSTRAINT check_review_status CHECK (review_status IN ('pending', 'approved', 'approved_as_noted', 'revise_resubmit', 'rejected'));

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_submittals_project_id ON project.submittals(project_id);
CREATE INDEX IF NOT EXISTS idx_submittals_workflow_status ON project.submittals(workflow_status);
CREATE INDEX IF NOT EXISTS idx_submittals_priority ON project.submittals(priority);
CREATE INDEX IF NOT EXISTS idx_submittals_ball_in_court ON project.submittals(ball_in_court);
CREATE INDEX IF NOT EXISTS idx_submittals_assigned_to ON project.submittals(assigned_to);
CREATE INDEX IF NOT EXISTS idx_submittals_required_approval_date ON project.submittals(required_approval_date);
CREATE INDEX IF NOT EXISTS idx_submittals_created_at ON project.submittals(created_at);
CREATE INDEX IF NOT EXISTS idx_submittals_csi_division ON project.submittals(csi_division);
CREATE INDEX IF NOT EXISTS idx_submittal_history_submittal_id ON project.submittal_history(submittal_id);
CREATE INDEX IF NOT EXISTS idx_submittal_notifications_submittal_id ON project.submittal_notifications(submittal_id);

-- Full text search index
CREATE INDEX IF NOT EXISTS idx_submittals_search ON project.submittals USING GIN(
    to_tsvector('english',
        COALESCE(package_name, '') || ' ' ||
        COALESCE(title, '') || ' ' ||
        COALESCE(description, '')
    )
);

-- Update audit triggers if they exist
CREATE OR REPLACE FUNCTION project.update_submittal_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_submittal_updated_at ON project.submittals;
CREATE TRIGGER update_submittal_updated_at
    BEFORE UPDATE ON project.submittals
    FOR EACH ROW
    EXECUTE FUNCTION project.update_submittal_updated_at();