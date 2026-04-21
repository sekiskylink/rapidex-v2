DROP INDEX IF EXISTS idx_reporters_mtuuid_unique;
DROP INDEX IF EXISTS idx_reporters_telephone_unique;

ALTER TABLE reporters
    DROP COLUMN IF EXISTS last_login_at,
    DROP COLUMN IF EXISTS rapidpro_uuid,
    DROP COLUMN IF EXISTS synced,
    DROP COLUMN IF EXISTS mtuuid,
    DROP COLUMN IF EXISTS sms_code_expires_at,
    DROP COLUMN IF EXISTS sms_code,
    DROP COLUMN IF EXISTS last_reporting_date,
    DROP COLUMN IF EXISTS total_reports,
    DROP COLUMN IF EXISTS district_id,
    DROP COLUMN IF EXISTS reporting_location,
    DROP COLUMN IF EXISTS telegram,
    DROP COLUMN IF EXISTS whatsapp,
    DROP COLUMN IF EXISTS telephone,
    DROP COLUMN IF EXISTS name;

DROP INDEX IF EXISTS idx_reporter_groups_group_name;
DROP INDEX IF EXISTS idx_reporter_groups_reporter_id;
DROP TABLE IF EXISTS reporter_groups;

ALTER TABLE org_units
    DROP COLUMN IF EXISTS last_sync_date,
    DROP COLUMN IF EXISTS deleted,
    DROP COLUMN IF EXISTS opening_date,
    DROP COLUMN IF EXISTS attribute_values,
    DROP COLUMN IF EXISTS extras,
    DROP COLUMN IF EXISTS phone_number,
    DROP COLUMN IF EXISTS url,
    DROP COLUMN IF EXISTS email,
    DROP COLUMN IF EXISTS address,
    DROP COLUMN IF EXISTS hierarchy_level,
    DROP COLUMN IF EXISTS short_name;
