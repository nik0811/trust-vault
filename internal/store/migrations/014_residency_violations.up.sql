CREATE TABLE IF NOT EXISTS residency_violations (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL,
    rule_id     TEXT NOT NULL,
    datasource_id   TEXT NOT NULL,
    datasource_name TEXT,
    datasource_region TEXT,
    rule_name       TEXT,
    regulation      TEXT,
    allowed_regions JSONB,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (rule_id, datasource_id)
);

CREATE INDEX IF NOT EXISTS idx_residency_violations_tenant ON residency_violations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_residency_violations_rule   ON residency_violations(rule_id);
