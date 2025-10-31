-- Migration: Create rfi_comment_attachments table
-- Date: 2025-10-30
-- Description: Add support for attachments on RFI comments (similar to issue_comment_attachments)

-- Create rfi_comment_attachments table
CREATE TABLE IF NOT EXISTS project.rfi_comment_attachments (
    id BIGSERIAL PRIMARY KEY,
    comment_id BIGINT REFERENCES project.rfi_comments(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT,
    file_type VARCHAR(50),
    attachment_type VARCHAR(50) NOT NULL DEFAULT 'photo',
    uploaded_by BIGINT NOT NULL REFERENCES iam.users(id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL REFERENCES iam.users(id),
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT NOT NULL REFERENCES iam.users(id),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_rfi_comment_attachments_comment_id ON project.rfi_comment_attachments(comment_id) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_rfi_comment_attachments_uploaded_by ON project.rfi_comment_attachments(uploaded_by) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_rfi_comment_attachments_created_at ON project.rfi_comment_attachments(created_at);

-- Add trigger to update updated_at timestamp
CREATE TRIGGER update_rfi_comment_attachments_updated_at
    BEFORE UPDATE ON project.rfi_comment_attachments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE project.rfi_comment_attachments IS 'Attachments linked to RFI comments';
COMMENT ON COLUMN project.rfi_comment_attachments.comment_id IS 'Reference to the RFI comment';
COMMENT ON COLUMN project.rfi_comment_attachments.attachment_type IS 'Type of attachment: photo, document, etc.';
