CREATE TABLE IF NOT EXISTS reporter_group_catalog (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_reporter_group_catalog_name_unique
    ON reporter_group_catalog (LOWER(name));

CREATE INDEX IF NOT EXISTS idx_reporter_group_catalog_is_active
    ON reporter_group_catalog (is_active);

INSERT INTO reporter_group_catalog (name, is_active, created_at, updated_at)
SELECT DISTINCT TRIM(group_name), TRUE, NOW(), NOW()
FROM reporter_groups
WHERE TRIM(group_name) <> ''
ON CONFLICT (LOWER(name)) DO NOTHING;
