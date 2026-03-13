CREATE TABLE request_events (
    id BIGSERIAL PRIMARY KEY,
    uid UUID NOT NULL UNIQUE,
    request_id BIGINT NULL REFERENCES exchange_requests(id) ON DELETE CASCADE,
    delivery_attempt_id BIGINT NULL REFERENCES delivery_attempts(id) ON DELETE CASCADE,
    async_task_id BIGINT NULL REFERENCES async_tasks(id) ON DELETE CASCADE,
    worker_run_id BIGINT NULL REFERENCES worker_runs(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    event_level TEXT NOT NULL DEFAULT 'info',
    event_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    message TEXT NULL,
    correlation_id TEXT NULL,
    actor_type TEXT NOT NULL DEFAULT 'system',
    actor_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    actor_name TEXT NULL,
    source_component TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_request_events_request_id ON request_events (request_id);
CREATE INDEX idx_request_events_delivery_attempt_id ON request_events (delivery_attempt_id);
CREATE INDEX idx_request_events_async_task_id ON request_events (async_task_id);
CREATE INDEX idx_request_events_worker_run_id ON request_events (worker_run_id);
CREATE INDEX idx_request_events_event_type ON request_events (event_type);
CREATE INDEX idx_request_events_correlation_id ON request_events (correlation_id);
CREATE INDEX idx_request_events_created_at ON request_events (created_at);

CREATE INDEX idx_exchange_requests_correlation_id ON exchange_requests (correlation_id);
CREATE INDEX idx_delivery_attempts_request_id_created_at ON delivery_attempts (request_id, created_at DESC);
CREATE INDEX idx_async_tasks_delivery_attempt_id_created_at ON async_tasks (delivery_attempt_id, created_at DESC);
