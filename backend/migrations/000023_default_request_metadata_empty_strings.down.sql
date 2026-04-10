ALTER TABLE exchange_requests
    ALTER COLUMN batch_id DROP NOT NULL,
    ALTER COLUMN batch_id DROP DEFAULT,
    ALTER COLUMN correlation_id DROP NOT NULL,
    ALTER COLUMN correlation_id DROP DEFAULT,
    ALTER COLUMN idempotency_key DROP NOT NULL,
    ALTER COLUMN idempotency_key DROP DEFAULT;
