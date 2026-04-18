CREATE TABLE scheduled_jobs (
    id BIGSERIAL PRIMARY KEY,
    uid UUID NOT NULL UNIQUE,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    job_category TEXT NOT NULL CHECK (job_category IN ('integration', 'maintenance')),
    job_type TEXT NOT NULL,
    schedule_type TEXT NOT NULL CHECK (schedule_type IN ('cron', 'interval')),
    schedule_expr TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    allow_concurrent_runs BOOLEAN NOT NULL DEFAULT FALSE,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_run_at TIMESTAMPTZ NULL,
    next_run_at TIMESTAMPTZ NULL,
    last_success_at TIMESTAMPTZ NULL,
    last_failure_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_scheduled_jobs_category ON scheduled_jobs (job_category);
CREATE INDEX idx_scheduled_jobs_enabled_next_run ON scheduled_jobs (enabled, next_run_at);

CREATE TABLE scheduled_job_runs (
    id BIGSERIAL PRIMARY KEY,
    uid UUID NOT NULL UNIQUE,
    scheduled_job_id BIGINT NOT NULL REFERENCES scheduled_jobs(id) ON DELETE CASCADE,
    trigger_mode TEXT NOT NULL CHECK (trigger_mode IN ('scheduled', 'manual')),
    scheduled_for TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'cancelled', 'skipped')),
    worker_id BIGINT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    result_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_scheduled_job_runs_job_created ON scheduled_job_runs (scheduled_job_id, created_at DESC);
CREATE INDEX idx_scheduled_job_runs_status ON scheduled_job_runs (status);
