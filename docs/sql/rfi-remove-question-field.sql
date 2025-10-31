-- Migration: Remove question field from RFIs and make description required
-- Date: 2025-10-29
-- Description: This migration removes the deprecated question column and makes description NOT NULL

-- First, set description to question value for any RFIs where description is NULL but question has content
UPDATE project.rfis
SET description = question
WHERE description IS NULL AND question IS NOT NULL AND question != '';

-- Set a default value for any remaining NULL descriptions
UPDATE project.rfis
SET description = 'No description provided'
WHERE description IS NULL OR description = '';

-- Now make description NOT NULL
ALTER TABLE project.rfis ALTER COLUMN description SET NOT NULL;

-- Drop the question column
ALTER TABLE project.rfis DROP COLUMN question;
