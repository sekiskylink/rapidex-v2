CREATE TABLE org_unit_sync_state (
    id INTEGER PRIMARY KEY,
    last_synced_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

INSERT INTO org_unit_sync_state (id, last_synced_at)
VALUES (1, NULL)
ON CONFLICT (id) DO NOTHING;
