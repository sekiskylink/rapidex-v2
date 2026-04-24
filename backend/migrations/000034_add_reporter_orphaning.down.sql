DROP INDEX IF EXISTS idx_reporters_orphaned_at;

ALTER TABLE reporters
    DROP COLUMN IF EXISTS last_known_org_unit_name,
    DROP COLUMN IF EXISTS last_known_org_unit_uid,
    DROP COLUMN IF EXISTS orphan_reason,
    DROP COLUMN IF EXISTS orphaned_at;

ALTER TABLE reporters
    DROP CONSTRAINT IF EXISTS reporters_org_unit_id_fkey;

ALTER TABLE reporters
    ADD CONSTRAINT reporters_org_unit_id_fkey
        FOREIGN KEY (org_unit_id) REFERENCES org_units(id) ON DELETE RESTRICT;

UPDATE reporters
SET org_unit_id = (
    SELECT id
    FROM org_units
    ORDER BY id ASC
    LIMIT 1
)
WHERE org_unit_id IS NULL;

ALTER TABLE reporters
    ALTER COLUMN org_unit_id SET NOT NULL;
