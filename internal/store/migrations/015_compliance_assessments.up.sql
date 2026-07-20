CREATE TABLE IF NOT EXISTS compliance_assessments (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    assessed_by            UUID NOT NULL,
    compliance_score       FLOAT NOT NULL DEFAULT 0,
    total_findings         INT NOT NULL DEFAULT 0,
    critical_findings      INT NOT NULL DEFAULT 0,
    high_findings          INT NOT NULL DEFAULT 0,
    medium_findings        INT NOT NULL DEFAULT 0,
    low_findings           INT NOT NULL DEFAULT 0,
    total_evidence         INT NOT NULL DEFAULT 0,
    data_sources_checked   INT NOT NULL DEFAULT 0,
    classifications_checked INT NOT NULL DEFAULT 0,
    policies_evaluated     INT NOT NULL DEFAULT 0,
    regulations_covered    JSONB NOT NULL DEFAULT '[]',
    summary                JSONB NOT NULL DEFAULT '{}',
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_compliance_assessments_tenant ON compliance_assessments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_compliance_assessments_created ON compliance_assessments(tenant_id, created_at DESC);
