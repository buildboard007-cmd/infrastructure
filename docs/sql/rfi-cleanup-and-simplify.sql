-- Migration: Clean up and simplify RFI table structure
-- Date: 2025-10-30
-- Description: Remove unnecessary fields, add received_from, change drawing/spec references to arrays, make category NOT NULL

-- Step 1: Add received_from field
ALTER TABLE project.rfis
ADD COLUMN received_from BIGINT REFERENCES iam.users(id);

-- Step 2: Add new array columns for drawing_numbers and specification_sections
ALTER TABLE project.rfis
ADD COLUMN drawing_numbers TEXT[] DEFAULT '{}',
ADD COLUMN specification_sections TEXT[] DEFAULT '{}';

-- Step 3: Migrate existing data from text fields to arrays (if there's data)
-- This assumes comma-separated values in the old fields
UPDATE project.rfis
SET drawing_numbers = string_to_array(drawing_references, ',')
WHERE drawing_references IS NOT NULL AND drawing_references != '';

UPDATE project.rfis
SET specification_sections = string_to_array(specification_references, ',')
WHERE specification_references IS NOT NULL AND specification_references != '';

-- Step 4: Make category NOT NULL (set default for existing NULL values)
UPDATE project.rfis
SET category = 'GENERAL'
WHERE category IS NULL;

ALTER TABLE project.rfis
ALTER COLUMN category SET NOT NULL;

-- Step 5: Drop unnecessary columns
ALTER TABLE project.rfis
DROP COLUMN IF EXISTS submitted_by,
DROP COLUMN IF EXISTS reviewer_email,
DROP COLUMN IF EXISTS approver_email,
DROP COLUMN IF EXISTS response_by,
DROP COLUMN IF EXISTS cc_list,
DROP COLUMN IF EXISTS submitted_date,
DROP COLUMN IF EXISTS response_date,
DROP COLUMN IF EXISTS response,
DROP COLUMN IF EXISTS response_status,
DROP COLUMN IF EXISTS cost_impact_details,
DROP COLUMN IF EXISTS schedule_impact_details,
DROP COLUMN IF EXISTS drawing_references,
DROP COLUMN IF EXISTS specification_references,
DROP COLUMN IF EXISTS related_submittals,
DROP COLUMN IF EXISTS related_change_events,
DROP COLUMN IF EXISTS workflow_type,
DROP COLUMN IF EXISTS requires_approval,
DROP COLUMN IF EXISTS approval_status,
DROP COLUMN IF EXISTS approved_by,
DROP COLUMN IF EXISTS approval_date,
DROP COLUMN IF EXISTS approval_comments,
DROP COLUMN IF EXISTS urgency_justification,
DROP COLUMN IF EXISTS business_justification,
DROP COLUMN IF EXISTS days_open,
DROP COLUMN IF EXISTS is_overdue,
DROP COLUMN IF EXISTS trade_type;

-- Step 6: Update status values to new simplified statuses
-- Map old statuses to new ones: DRAFT, OPEN, CLOSED
UPDATE project.rfis
SET status = CASE
    WHEN UPPER(status) = 'DRAFT' THEN 'DRAFT'
    WHEN UPPER(status) IN ('SUBMITTED', 'UNDER_REVIEW', 'ANSWERED', 'REQUIRES_REVISION') THEN 'OPEN'
    WHEN UPPER(status) IN ('CLOSED', 'VOID') THEN 'CLOSED'
    ELSE 'DRAFT'
END;

-- Step 7: Create indexes for new fields
CREATE INDEX IF NOT EXISTS idx_rfis_received_from ON project.rfis(received_from) WHERE received_from IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rfis_drawing_numbers ON project.rfis USING GIN (drawing_numbers);
CREATE INDEX IF NOT EXISTS idx_rfis_specification_sections ON project.rfis USING GIN (specification_sections);

-- Step 8: Add comments for documentation
COMMENT ON COLUMN project.rfis.received_from IS 'User ID who sent/created this RFI';
COMMENT ON COLUMN project.rfis.drawing_numbers IS 'Array of drawing reference numbers';
COMMENT ON COLUMN project.rfis.specification_sections IS 'Array of specification section numbers';
