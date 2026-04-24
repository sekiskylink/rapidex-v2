CREATE TABLE IF NOT EXISTS org_unit_levels (
    id BIGSERIAL PRIMARY KEY,
    uid TEXT NOT NULL UNIQUE,
    code VARCHAR(64) NOT NULL DEFAULT '',
    name VARCHAR(230) NOT NULL,
    level INTEGER NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_org_unit_levels_name ON org_unit_levels (name);
CREATE INDEX IF NOT EXISTS idx_org_unit_levels_code ON org_unit_levels (code);

CREATE TABLE IF NOT EXISTS org_unit_groups (
    id BIGSERIAL PRIMARY KEY,
    uid TEXT NOT NULL UNIQUE,
    code VARCHAR(64) NOT NULL DEFAULT '',
    name VARCHAR(230) NOT NULL,
    short_name VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_org_unit_groups_name ON org_unit_groups (name);
CREATE INDEX IF NOT EXISTS idx_org_unit_groups_code ON org_unit_groups (code);

CREATE TABLE IF NOT EXISTS org_unit_attributes (
    id BIGSERIAL PRIMARY KEY,
    uid TEXT NOT NULL UNIQUE,
    code VARCHAR(64) NOT NULL DEFAULT '',
    name VARCHAR(230) NOT NULL,
    short_name VARCHAR(64) NOT NULL DEFAULT '',
    value_type VARCHAR(64) NOT NULL DEFAULT '',
    is_unique BOOLEAN NOT NULL DEFAULT FALSE,
    mandatory BOOLEAN NOT NULL DEFAULT FALSE,
    organisation_unit_attribute BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_org_unit_attributes_name ON org_unit_attributes (name);
CREATE INDEX IF NOT EXISTS idx_org_unit_attributes_code ON org_unit_attributes (code);

CREATE TABLE IF NOT EXISTS org_unit_group_members (
    org_unit_id BIGINT NOT NULL REFERENCES org_units(id) ON DELETE CASCADE,
    org_unit_group_id BIGINT NOT NULL REFERENCES org_unit_groups(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (org_unit_id, org_unit_group_id)
);

ALTER TABLE org_unit_sync_state
    ADD COLUMN IF NOT EXISTS last_started_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_completed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_status TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_error TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS source_server_code TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS district_level_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS district_level_code TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_counts JSONB NOT NULL DEFAULT '{}'::jsonb;
