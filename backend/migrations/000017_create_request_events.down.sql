DROP INDEX IF EXISTS idx_async_tasks_delivery_attempt_id_created_at;
DROP INDEX IF EXISTS idx_delivery_attempts_request_id_created_at;
DROP INDEX IF EXISTS idx_exchange_requests_correlation_id;

DROP INDEX IF EXISTS idx_request_events_created_at;
DROP INDEX IF EXISTS idx_request_events_correlation_id;
DROP INDEX IF EXISTS idx_request_events_event_type;
DROP INDEX IF EXISTS idx_request_events_worker_run_id;
DROP INDEX IF EXISTS idx_request_events_async_task_id;
DROP INDEX IF EXISTS idx_request_events_delivery_attempt_id;
DROP INDEX IF EXISTS idx_request_events_request_id;

DROP TABLE IF EXISTS request_events;
