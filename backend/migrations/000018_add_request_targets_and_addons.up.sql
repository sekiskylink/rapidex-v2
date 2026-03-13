CREATE TABLE request_targets (
    id BIGSERIAL PRIMARY KEY,
    uid UUID NOT NULL UNIQUE,
    request_id BIGINT NOT NULL REFERENCES exchange_requests(id) ON DELETE CASCADE,
    server_id BIGINT NOT NULL REFERENCES integration_servers(id),
    target_kind TEXT NOT NULL DEFAULT 'primary',
    priority INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'blocked', 'processing', 'succeeded', 'failed')),
    blocked_reason TEXT NOT NULL DEFAULT '',
    deferred_until TIMESTAMPTZ NULL,
    last_released_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT request_targets_request_server_unique UNIQUE (request_id, server_id)
);

CREATE INDEX request_targets_request_id_idx ON request_targets (request_id);
CREATE INDEX request_targets_server_id_idx ON request_targets (server_id);
CREATE INDEX request_targets_status_idx ON request_targets (status);
CREATE INDEX request_targets_deferred_until_idx ON request_targets (deferred_until);

INSERT INTO request_targets (uid, request_id, server_id, target_kind, priority, status, created_at, updated_at)
SELECT gen_random_uuid(), r.id, r.destination_server_id, 'primary', 1,
       CASE
           WHEN r.status = 'completed' THEN 'succeeded'
           WHEN r.status = 'failed' THEN 'failed'
           WHEN r.status = 'processing' THEN 'processing'
           ELSE 'pending'
       END,
       r.created_at,
       r.updated_at
FROM exchange_requests r;

CREATE TABLE request_dependencies (
    request_id BIGINT NOT NULL REFERENCES exchange_requests(id) ON DELETE CASCADE,
    depends_on_request_id BIGINT NOT NULL REFERENCES exchange_requests(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (request_id, depends_on_request_id),
    CONSTRAINT request_dependencies_not_self CHECK (request_id <> depends_on_request_id)
);

CREATE INDEX request_dependencies_depends_on_idx ON request_dependencies (depends_on_request_id);

ALTER TABLE exchange_requests
    ADD COLUMN status_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN deferred_until TIMESTAMPTZ NULL;

ALTER TABLE delivery_attempts
    ADD COLUMN submission_hold_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN next_eligible_at TIMESTAMPTZ NULL,
    ADD COLUMN hold_policy_source TEXT NOT NULL DEFAULT '',
    ADD COLUMN response_content_type TEXT NOT NULL DEFAULT '',
    ADD COLUMN response_body_filtered BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN response_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN terminal_reason TEXT NOT NULL DEFAULT '';

ALTER TABLE delivery_attempts DROP CONSTRAINT delivery_attempts_request_attempt_unique;
ALTER TABLE delivery_attempts
    ADD CONSTRAINT delivery_attempts_request_server_attempt_unique UNIQUE (request_id, server_id, attempt_number);

CREATE INDEX idx_delivery_attempts_next_eligible_at ON delivery_attempts (next_eligible_at);
CREATE INDEX idx_delivery_attempts_submission_hold_reason ON delivery_attempts (submission_hold_reason);

ALTER TABLE async_task_polls
    ADD COLUMN response_content_type TEXT NOT NULL DEFAULT '',
    ADD COLUMN response_body_filtered BOOLEAN NOT NULL DEFAULT FALSE;
