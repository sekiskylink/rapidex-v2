CREATE TABLE integration_servers (
    id BIGSERIAL PRIMARY KEY,
    uid UUID NOT NULL,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    system_type TEXT NOT NULL,
    base_url TEXT NOT NULL,
    endpoint_type TEXT NOT NULL,
    http_method TEXT NOT NULL,
    use_async BOOLEAN NOT NULL DEFAULT FALSE,
    parse_responses BOOLEAN NOT NULL DEFAULT TRUE,
    headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    url_params JSONB NOT NULL DEFAULT '{}'::jsonb,
    suspended BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX integration_servers_uid_key ON integration_servers (uid);
CREATE UNIQUE INDEX integration_servers_code_key ON integration_servers (code);
CREATE INDEX integration_servers_system_type_idx ON integration_servers (system_type);
CREATE INDEX integration_servers_suspended_idx ON integration_servers (suspended);
CREATE INDEX integration_servers_created_at_idx ON integration_servers (created_at DESC);
CREATE INDEX integration_servers_updated_at_idx ON integration_servers (updated_at DESC);
