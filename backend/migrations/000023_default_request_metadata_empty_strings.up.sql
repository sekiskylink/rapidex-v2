UPDATE exchange_requests
SET batch_id = COALESCE(batch_id, ''),
    correlation_id = COALESCE(correlation_id, ''),
    idempotency_key = COALESCE(idempotency_key, '');

ALTER TABLE exchange_requests
    ALTER COLUMN batch_id SET DEFAULT '',
    ALTER COLUMN batch_id SET NOT NULL,
    ALTER COLUMN correlation_id SET DEFAULT '',
    ALTER COLUMN correlation_id SET NOT NULL,
    ALTER COLUMN idempotency_key SET DEFAULT '',
    ALTER COLUMN idempotency_key SET NOT NULL;
