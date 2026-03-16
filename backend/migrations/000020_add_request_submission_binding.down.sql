ALTER TABLE exchange_requests
    DROP CONSTRAINT IF EXISTS exchange_requests_submission_binding_check;

ALTER TABLE exchange_requests
    DROP CONSTRAINT IF EXISTS exchange_requests_payload_format_check;

ALTER TABLE exchange_requests
    DROP COLUMN IF EXISTS submission_binding;
