ALTER TABLE reporters
    ALTER COLUMN org_unit_id DROP NOT NULL;

ALTER TABLE reporters
    DROP CONSTRAINT IF EXISTS reporters_org_unit_id_fkey;

ALTER TABLE reporters
    ADD CONSTRAINT reporters_org_unit_id_fkey
        FOREIGN KEY (org_unit_id) REFERENCES org_units(id) ON DELETE SET NULL;

ALTER TABLE reporters
    ADD COLUMN IF NOT EXISTS orphaned_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS orphan_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_known_org_unit_uid TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_known_org_unit_name TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_reporters_orphaned_at ON reporters (orphaned_at);
