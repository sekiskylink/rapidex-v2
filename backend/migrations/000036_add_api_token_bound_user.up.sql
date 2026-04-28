ALTER TABLE api_tokens
    ADD COLUMN IF NOT EXISTS bound_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_api_tokens_bound_user_id
    ON api_tokens (bound_user_id);
