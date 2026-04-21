CREATE TABLE org_units (
    id BIGSERIAL PRIMARY KEY,
    uid VARCHAR(36) NOT NULL UNIQUE,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    parent_id BIGINT REFERENCES org_units(id) ON DELETE RESTRICT,
    path TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_org_units_parent_id ON org_units (parent_id);
CREATE INDEX idx_org_units_path ON org_units (path);

CREATE TABLE reporters (
    id BIGSERIAL PRIMARY KEY,
    uid VARCHAR(36) NOT NULL UNIQUE,
    contact_uuid VARCHAR(36) NOT NULL UNIQUE,
    phone_number VARCHAR(32) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    org_unit_id BIGINT NOT NULL REFERENCES org_units(id) ON DELETE RESTRICT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reporters_org_unit_id ON reporters (org_unit_id);
CREATE INDEX idx_reporters_is_active ON reporters (is_active);
