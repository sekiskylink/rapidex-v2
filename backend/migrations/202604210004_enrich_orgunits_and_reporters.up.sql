ALTER TABLE org_units
    ADD COLUMN IF NOT EXISTS short_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS hierarchy_level INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS address TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS email TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS phone_number TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS extras JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS attribute_values JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS opening_date DATE,
    ADD COLUMN IF NOT EXISTS deleted BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS last_sync_date TIMESTAMPTZ;

UPDATE org_units
SET short_name = COALESCE(NULLIF(short_name, ''), LEFT(name, 50))
WHERE COALESCE(short_name, '') = '';

CREATE TABLE IF NOT EXISTS reporter_groups (
    id BIGSERIAL PRIMARY KEY,
    reporter_id BIGINT NOT NULL REFERENCES reporters(id) ON DELETE CASCADE,
    group_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_reporter_group UNIQUE (reporter_id, group_name)
);

CREATE INDEX IF NOT EXISTS idx_reporter_groups_reporter_id ON reporter_groups (reporter_id);
CREATE INDEX IF NOT EXISTS idx_reporter_groups_group_name ON reporter_groups (group_name);

ALTER TABLE reporters
    ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS telephone TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS whatsapp TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS telegram TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS reporting_location TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS district_id BIGINT REFERENCES org_units(id),
    ADD COLUMN IF NOT EXISTS total_reports INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_reporting_date TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS sms_code TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sms_code_expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS mtuuid TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS synced BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS rapidpro_uuid TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

UPDATE reporters
SET name = COALESCE(NULLIF(name, ''), display_name),
    telephone = COALESCE(NULLIF(telephone, ''), phone_number),
    rapidpro_uuid = COALESCE(NULLIF(rapidpro_uuid, ''), contact_uuid)
WHERE COALESCE(name, '') = ''
   OR COALESCE(telephone, '') = ''
   OR COALESCE(rapidpro_uuid, '') = '';

ALTER TABLE reporters
    ALTER COLUMN name SET NOT NULL,
    ALTER COLUMN telephone SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_reporters_telephone_unique ON reporters (telephone);
CREATE UNIQUE INDEX IF NOT EXISTS idx_reporters_mtuuid_unique ON reporters (mtuuid) WHERE mtuuid <> '';

