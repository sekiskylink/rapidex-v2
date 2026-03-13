DROP INDEX IF EXISTS idx_async_tasks_poll_claimed_by_worker_run_id;
DROP INDEX IF EXISTS idx_async_tasks_poll_claimed_at;

ALTER TABLE async_tasks
    DROP COLUMN IF EXISTS poll_claimed_by_worker_run_id,
    DROP COLUMN IF EXISTS poll_claimed_at;
