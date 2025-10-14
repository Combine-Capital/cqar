-- Drop group_members table first (due to foreign key dependency)
DROP TABLE IF EXISTS group_members CASCADE;

-- Drop asset_groups table
DROP TABLE IF EXISTS asset_groups CASCADE;
