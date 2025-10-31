-- Migration: Make location_id required in RFIs table
-- Date: 2025-10-30
-- Description: Change location_id from nullable to NOT NULL

-- Step 1: First, check if there are any RFIs with NULL location_id
-- If there are, you'll need to update them first before running this migration

-- Step 2: Make location_id NOT NULL
ALTER TABLE project.rfis
ALTER COLUMN location_id SET NOT NULL;

-- Step 3: Add comment for documentation
COMMENT ON COLUMN project.rfis.location_id IS 'Location ID (required) - references project.locations';
