UPDATE org_units
SET code = NULL
WHERE NULLIF(BTRIM(code), '') IS NULL;

ALTER TABLE org_units
    DROP CONSTRAINT IF EXISTS org_units_code_key;

ALTER TABLE org_units
    ALTER COLUMN code DROP NOT NULL;

DROP INDEX IF EXISTS idx_org_units_code_unique;

CREATE UNIQUE INDEX IF NOT EXISTS idx_org_units_code_unique
    ON org_units (NULLIF(BTRIM(code), ''));
