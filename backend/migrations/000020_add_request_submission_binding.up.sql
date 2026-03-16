ALTER TABLE exchange_requests
    ADD COLUMN submission_binding TEXT NOT NULL DEFAULT 'body';

ALTER TABLE exchange_requests
    ADD CONSTRAINT exchange_requests_payload_format_check
        CHECK (payload_format IN ('json', 'text'));

ALTER TABLE exchange_requests
    ADD CONSTRAINT exchange_requests_submission_binding_check
        CHECK (submission_binding IN ('body', 'query'));
