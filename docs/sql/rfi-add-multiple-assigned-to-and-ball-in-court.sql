-- Migration: Convert assigned_to to array and add ball_in_court field
-- Date: 2025-10-29
-- Description: This migration converts assigned_to from single user to array of users and adds ball_in_court field

-- Step 1: Add new columns
ALTER TABLE project.rfis
ADD COLUMN assigned_to_temp BIGINT[] DEFAULT '{}',
ADD COLUMN ball_in_court BIGINT REFERENCES iam.users(id);

-- Step 2: Migrate existing assigned_to data to array
UPDATE project.rfis
SET assigned_to_temp = ARRAY[assigned_to]
WHERE assigned_to IS NOT NULL;

-- Step 3: Drop old assigned_to column and rename new one
ALTER TABLE project.rfis
DROP COLUMN assigned_to;

ALTER TABLE project.rfis
RENAME COLUMN assigned_to_temp TO assigned_to;

-- Step 4: Create indexes for performance
CREATE INDEX idx_rfis_assigned_to ON project.rfis USING GIN (assigned_to);
CREATE INDEX idx_rfis_ball_in_court ON project.rfis(ball_in_court) WHERE ball_in_court IS NOT NULL;

-- Step 5: Add comments for documentation
COMMENT ON COLUMN project.rfis.assigned_to IS 'Array of user IDs assigned to this RFI';
COMMENT ON COLUMN project.rfis.ball_in_court IS 'User ID who currently needs to take action on this RFI';
