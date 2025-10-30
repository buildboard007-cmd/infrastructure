-- Update organization table to allow NULL values for initial super admin signup
-- These fields will be populated during the organization setup step

ALTER TABLE iam.organizations 
ALTER COLUMN name DROP NOT NULL;

ALTER TABLE iam.organizations 
ALTER COLUMN org_type DROP NOT NULL;

-- Also make user names nullable since they're not required during initial signup
ALTER TABLE iam.users 
ALTER COLUMN first_name DROP NOT NULL;

ALTER TABLE iam.users 
ALTER COLUMN last_name DROP NOT NULL;

-- Verify the changes
SELECT 
    column_name,
    is_nullable,
    data_type,
    column_default
FROM information_schema.columns
WHERE table_schema = 'iam' 
AND table_name = 'organizations'
AND column_name IN ('name', 'org_type')

UNION ALL

SELECT 
    column_name,
    is_nullable,
    data_type,
    column_default
FROM information_schema.columns
WHERE table_schema = 'iam' 
AND table_name = 'users'
AND column_name IN ('first_name', 'last_name');