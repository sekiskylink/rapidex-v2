CREATE UNIQUE INDEX exchange_requests_source_system_idempotency_key_unique
    ON exchange_requests (source_system, idempotency_key)
    WHERE idempotency_key IS NOT NULL AND idempotency_key <> '';

CREATE INDEX exchange_requests_batch_id_idx
    ON exchange_requests (batch_id)
    WHERE batch_id IS NOT NULL AND batch_id <> '';
