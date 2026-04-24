CREATE EXTENSION IF NOT EXISTS postgis;
CREATE TABLE IF NOT EXISTS orgunitlevel
(
    id      SERIAL       NOT NULL PRIMARY KEY,
    uid     TEXT         NOT NULL UNIQUE,
    name    VARCHAR(230) NOT NULL UNIQUE,
    code    VARCHAR(50) UNIQUE,
    level   INTEGER      NOT NULL UNIQUE,
    created TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS orgunitgroup
(
    id        SERIAL       NOT NULL PRIMARY KEY,
    uid       TEXT         NOT NULL UNIQUE,
    code      VARCHAR(50) UNIQUE,
    name      VARCHAR(230) NOT NULL UNIQUE,
    shortname VARCHAR(50)  NOT NULL DEFAULT '' UNIQUE,
    created   TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP,
    updated   TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP

);
CREATE INDEX IF NOT EXISTS orgunitgroup_name_idx ON orgunitgroup (id);

CREATE TABLE IF NOT EXISTS attribute
(
    id                        SERIAL       NOT NULL PRIMARY KEY,
    uid                       TEXT         NOT NULL UNIQUE,
    code                      VARCHAR(50) UNIQUE,
    name                      VARCHAR(230) NOT NULL UNIQUE,
    shortname                 VARCHAR(50)  NOT NULL DEFAULT '',
    valuetype                 VARCHAR(50)  NOT NULL DEFAULT '',
    isunique                  BOOLEAN      NOT NULL DEFAULT FALSE,
    mandatory                 BOOLEAN      NOT NULL DEFAULT FALSE,
    organisationunitattribute BOOLEAN      NOT NULL,
    created                   TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP,
    updated                   TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS organisationunit
(
    id              BIGSERIAL NOT NULL PRIMARY KEY,
    uid             TEXT      NOT NULL UNIQUE,
    code            VARCHAR(50) UNIQUE,
    name            TEXT      NOT NULL DEFAULT '',
    shortname       TEXT      NOT NULL DEFAULT '',
    description     TEXT      NOT NULL DEFAULT '',
    parentid        BIGINT REFERENCES organisationunit (id),
    hierarchylevel  INTEGER   NOT NULL,
    path            TEXT      NOT NULL UNIQUE,
    address         TEXT      NOT NULL DEFAULT '',
    email           TEXT      NOT NULL DEFAULT '',
    url             TEXT      NOT NULL DEFAULT '',
    phonenumber     TEXT      NOT NULL DEFAULT '',
    extras          JSONB     NOT NULL DEFAULT '{}'::jsonb,
    attributevalues JSONB              DEFAULT '{}'::jsonb,
    openingdate     DATE,
    deleted         BOOLEAN   NOT NULL DEFAULT FALSE,
    geometry        geometry(Geometry, 4326),
    lastsyncdate    TIMESTAMPTZ,
    created         TIMESTAMPTZ        DEFAULT CURRENT_TIMESTAMP,
    updated         TIMESTAMPTZ        DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS organisationunit_name_idx ON organisationunit (name);
CREATE INDEX IF NOT EXISTS organisationunit_level_idx ON organisationunit (hierarchylevel);
CREATE INDEX IF NOT EXISTS organisationunit_path_idx ON organisationunit (path);
CREATE INDEX IF NOT EXISTS organisationunit_parent_idx ON organisationunit (parentid);
CREATE INDEX IF NOT EXISTS organisationunit_created_idx ON organisationunit (created);
CREATE INDEX IF NOT EXISTS organisationunit_updated_idx ON organisationunit (updated);

CREATE TABLE IF NOT EXISTS orgunitgroupmembers
(
    organisationunitid BIGSERIAL REFERENCES organisationunit (id),
    orgunitgroupid     SERIAL REFERENCES orgunitgroup (id),
    created            TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated            TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (organisationunitid, orgunitgroupid)

);

CREATE TABLE IF NOT EXISTS user_org_unit
(
    user_id     BIGINT REFERENCES users (id) ON DELETE CASCADE,
    org_unit_id BIGINT REFERENCES organisationunit (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, org_unit_id)
);
