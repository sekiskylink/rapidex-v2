ALTER TABLE async_tasks
    ADD COLUMN poll_claimed_at TIMESTAMPTZ NULL,
    ADD COLUMN poll_claimed_by_worker_run_id BIGINT NULL REFERENCES worker_runs(id) ON DELETE SET NULL;

CREATE INDEX idx_async_tasks_poll_claimed_at ON async_tasks (poll_claimed_at);
CREATE INDEX idx_async_tasks_poll_claimed_by_worker_run_id ON async_tasks (poll_claimed_by_worker_run_id);
