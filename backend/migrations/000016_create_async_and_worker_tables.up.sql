CREATE TABLE async_tasks (
    id BIGSERIAL PRIMARY KEY,
    uid UUID NOT NULL UNIQUE,
    delivery_attempt_id BIGINT NOT NULL REFERENCES delivery_attempts (id) ON DELETE CASCADE,
    remote_job_id TEXT NULL,
    poll_url TEXT NULL,
    remote_status TEXT NULL,
    terminal_state TEXT NULL,
    next_poll_at TIMESTAMPTZ NULL,
    completed_at TIMESTAMPTZ NULL,
    remote_response JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_async_tasks_delivery_attempt_id ON async_tasks (delivery_attempt_id);
CREATE INDEX idx_async_tasks_remote_job_id ON async_tasks (remote_job_id);
CREATE INDEX idx_async_tasks_next_poll_at ON async_tasks (next_poll_at);
CREATE INDEX idx_async_tasks_terminal_state ON async_tasks (terminal_state);

CREATE TABLE async_task_polls (
    id BIGSERIAL PRIMARY KEY,
    async_task_id BIGINT NOT NULL REFERENCES async_tasks (id) ON DELETE CASCADE,
    polled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status_code INTEGER NULL,
    remote_status TEXT NULL,
    response_body TEXT NULL,
    error_message TEXT NULL,
    duration_ms INTEGER NULL
);

CREATE INDEX idx_async_task_polls_async_task_id ON async_task_polls (async_task_id);
CREATE INDEX idx_async_task_polls_polled_at ON async_task_polls (polled_at);

CREATE TABLE rate_limit_policies (
    id BIGSERIAL PRIMARY KEY,
    uid UUID NOT NULL UNIQUE,
    name TEXT NOT NULL,
    scope_type TEXT NOT NULL,
    scope_ref TEXT NULL,
    rps INTEGER NOT NULL,
    burst INTEGER NOT NULL,
    max_concurrency INTEGER NOT NULL,
    timeout_ms INTEGER NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rate_limit_policies_scope_type ON rate_limit_policies (scope_type);
CREATE INDEX idx_rate_limit_policies_scope_ref ON rate_limit_policies (scope_ref);
CREATE INDEX idx_rate_limit_policies_is_active ON rate_limit_policies (is_active);

CREATE TABLE worker_runs (
    id BIGSERIAL PRIMARY KEY,
    uid UUID NOT NULL UNIQUE,
    worker_type TEXT NOT NULL,
    worker_name TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    stopped_at TIMESTAMPTZ NULL,
    last_heartbeat_at TIMESTAMPTZ NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_worker_runs_worker_type ON worker_runs (worker_type);
CREATE INDEX idx_worker_runs_status ON worker_runs (status);
CREATE INDEX idx_worker_runs_last_heartbeat_at ON worker_runs (last_heartbeat_at);
