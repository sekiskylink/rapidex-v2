ALTER TABLE integration_servers
    ADD COLUMN response_body_persistence TEXT NOT NULL DEFAULT 'filter';

ALTER TABLE integration_servers
    ADD CONSTRAINT integration_servers_response_body_persistence_check
    CHECK (response_body_persistence IN ('filter', 'save', 'discard'));

ALTER TABLE exchange_requests
    ADD COLUMN response_body_persistence TEXT NOT NULL DEFAULT '';

ALTER TABLE exchange_requests
    ADD CONSTRAINT exchange_requests_response_body_persistence_check
    CHECK (response_body_persistence IN ('', 'filter', 'save', 'discard'));
