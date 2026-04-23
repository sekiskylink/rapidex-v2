CREATE TABLE reporter_broadcasts (
    id BIGSERIAL PRIMARY KEY,
    uid TEXT NOT NULL UNIQUE,
    requested_by_user_id BIGINT NOT NULL,
    org_unit_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    reporter_group TEXT NOT NULL DEFAULT '',
    message_text TEXT NOT NULL DEFAULT '',
    dedupe_key TEXT NOT NULL DEFAULT '',
    matched_count INTEGER NOT NULL DEFAULT 0,
    sent_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'completed', 'failed')),
    last_error TEXT NOT NULL DEFAULT '',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    claimed_at TIMESTAMPTZ NULL,
    claimed_by_worker_run_id BIGINT NULL REFERENCES worker_runs(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reporter_broadcasts_status_requested ON reporter_broadcasts (status, requested_at);
CREATE INDEX idx_reporter_broadcasts_dedupe_requested ON reporter_broadcasts (dedupe_key, requested_at DESC);
