CREATE TABLE exchange_requests (
    id BIGSERIAL PRIMARY KEY,
    uid UUID NOT NULL,
    source_system TEXT NOT NULL DEFAULT '',
    destination_server_id BIGINT NOT NULL REFERENCES integration_servers(id),
    batch_id TEXT NULL,
    correlation_id TEXT NULL,
    idempotency_key TEXT NULL,
    payload_body TEXT NOT NULL,
    payload_format TEXT NOT NULL DEFAULT 'json',
    url_suffix TEXT NULL,
    status TEXT NOT NULL,
    extras JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX exchange_requests_uid_key ON exchange_requests (uid);
CREATE INDEX exchange_requests_correlation_id_idx ON exchange_requests (correlation_id);
CREATE INDEX exchange_requests_destination_server_id_idx ON exchange_requests (destination_server_id);
CREATE INDEX exchange_requests_created_at_idx ON exchange_requests (created_at DESC);
CREATE INDEX exchange_requests_status_idx ON exchange_requests (status);
