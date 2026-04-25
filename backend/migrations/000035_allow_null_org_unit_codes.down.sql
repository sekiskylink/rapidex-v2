DROP INDEX IF EXISTS idx_org_units_code_unique;

UPDATE org_units
SET code = uid
WHERE NULLIF(BTRIM(code), '') IS NULL;

ALTER TABLE org_units
    ALTER COLUMN code SET NOT NULL;

ALTER TABLE org_units
    ADD CONSTRAINT org_units_code_key UNIQUE (code);
