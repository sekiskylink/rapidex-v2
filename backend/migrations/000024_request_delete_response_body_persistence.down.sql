ALTER TABLE exchange_requests
    DROP CONSTRAINT IF EXISTS exchange_requests_response_body_persistence_check;

ALTER TABLE exchange_requests
    DROP COLUMN IF EXISTS response_body_persistence;

ALTER TABLE integration_servers
    DROP CONSTRAINT IF EXISTS integration_servers_response_body_persistence_check;

ALTER TABLE integration_servers
    DROP COLUMN IF EXISTS response_body_persistence;
