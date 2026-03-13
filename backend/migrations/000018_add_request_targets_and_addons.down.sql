ALTER TABLE async_task_polls
    DROP COLUMN response_body_filtered,
    DROP COLUMN response_content_type;

DROP INDEX IF EXISTS idx_delivery_attempts_submission_hold_reason;
DROP INDEX IF EXISTS idx_delivery_attempts_next_eligible_at;

ALTER TABLE delivery_attempts DROP CONSTRAINT IF EXISTS delivery_attempts_request_server_attempt_unique;
ALTER TABLE delivery_attempts
    ADD CONSTRAINT delivery_attempts_request_attempt_unique UNIQUE (request_id, attempt_number);

ALTER TABLE delivery_attempts
    DROP COLUMN terminal_reason,
    DROP COLUMN response_summary,
    DROP COLUMN response_body_filtered,
    DROP COLUMN response_content_type,
    DROP COLUMN hold_policy_source,
    DROP COLUMN next_eligible_at,
    DROP COLUMN submission_hold_reason;

ALTER TABLE exchange_requests
    DROP COLUMN deferred_until,
    DROP COLUMN status_reason;

DROP INDEX IF EXISTS request_dependencies_depends_on_idx;
DROP TABLE IF EXISTS request_dependencies;

DROP INDEX IF EXISTS request_targets_deferred_until_idx;
DROP INDEX IF EXISTS request_targets_status_idx;
DROP INDEX IF EXISTS request_targets_server_id_idx;
DROP INDEX IF EXISTS request_targets_request_id_idx;
DROP TABLE IF EXISTS request_targets;
