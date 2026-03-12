CREATE TABLE delivery_attempts (
    id BIGSERIAL PRIMARY KEY,
    uid UUID NOT NULL UNIQUE,
    request_id BIGINT NOT NULL REFERENCES exchange_requests(id) ON DELETE CASCADE,
    server_id BIGINT NOT NULL REFERENCES integration_servers(id),
    attempt_number INTEGER NOT NULL CHECK (attempt_number > 0),
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'retrying')),
    http_status INTEGER,
    response_body TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    retry_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT delivery_attempts_request_attempt_unique UNIQUE (request_id, attempt_number)
);

CREATE INDEX idx_delivery_attempts_request_id ON delivery_attempts (request_id);
CREATE INDEX idx_delivery_attempts_server_id ON delivery_attempts (server_id);
CREATE INDEX idx_delivery_attempts_status ON delivery_attempts (status);
CREATE INDEX idx_delivery_attempts_retry_at ON delivery_attempts (retry_at);
CREATE INDEX idx_delivery_attempts_created_at ON delivery_attempts (created_at);
