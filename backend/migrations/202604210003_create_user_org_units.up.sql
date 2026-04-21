CREATE TABLE user_org_units (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_unit_id BIGINT NOT NULL REFERENCES org_units(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, org_unit_id)
);

CREATE INDEX idx_user_org_units_user_id ON user_org_units (user_id);
CREATE INDEX idx_user_org_units_org_unit_id ON user_org_units (org_unit_id);
