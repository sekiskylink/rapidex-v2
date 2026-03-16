CREATE TABLE ingest_files (
    id BIGSERIAL PRIMARY KEY,
    uid UUID NOT NULL UNIQUE,
    source_kind TEXT NOT NULL,
    original_name TEXT NOT NULL,
    source_path TEXT NOT NULL,
    current_path TEXT NOT NULL,
    archived_path TEXT NULL,
    status TEXT NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    modified_at TIMESTAMPTZ NULL,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    claimed_at TIMESTAMPTZ NULL,
    claimed_by_worker_run_id BIGINT NULL REFERENCES worker_runs(id) ON DELETE SET NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NULL,
    request_id BIGINT NULL REFERENCES exchange_requests(id) ON DELETE SET NULL,
    checksum_sha256 TEXT NULL,
    idempotency_key TEXT NULL,
    last_error_code TEXT NULL,
    last_error_message TEXT NULL,
    processed_at TIMESTAMPTZ NULL,
    failed_at TIMESTAMPTZ NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ingest_files_status ON ingest_files (status);
CREATE INDEX idx_ingest_files_next_attempt_at ON ingest_files (next_attempt_at);
CREATE INDEX idx_ingest_files_claimed_at ON ingest_files (claimed_at);
CREATE INDEX idx_ingest_files_request_id ON ingest_files (request_id);
CREATE INDEX idx_ingest_files_source_path ON ingest_files (source_path);
CREATE UNIQUE INDEX idx_ingest_files_active_current_path
    ON ingest_files (current_path)
    WHERE status IN ('discovered', 'retry', 'processing');
