ALTER TABLE org_unit_sync_state
    DROP COLUMN IF EXISTS last_counts,
    DROP COLUMN IF EXISTS district_level_code,
    DROP COLUMN IF EXISTS district_level_name,
    DROP COLUMN IF EXISTS source_server_code,
    DROP COLUMN IF EXISTS last_error,
    DROP COLUMN IF EXISTS last_status,
    DROP COLUMN IF EXISTS last_completed_at,
    DROP COLUMN IF EXISTS last_started_at;

DROP TABLE IF EXISTS org_unit_group_members;
DROP TABLE IF EXISTS org_unit_attributes;
DROP TABLE IF EXISTS org_unit_groups;
DROP TABLE IF EXISTS org_unit_levels;
