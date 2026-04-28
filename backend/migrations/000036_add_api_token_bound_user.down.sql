DROP INDEX IF EXISTS idx_api_tokens_bound_user_id;

ALTER TABLE api_tokens
    DROP COLUMN IF EXISTS bound_user_id;
