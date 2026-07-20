-- Custom entities: tenant-defined entity types the classifier learns to detect
CREATE TABLE IF NOT EXISTS custom_entities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    pattern TEXT NOT NULL,
    description TEXT,
    detections INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_custom_entities_tenant ON custom_entities(tenant_id);

-- Knowledge cache: records corrections that teach the classifier
CREATE TABLE IF NOT EXISTS knowledge_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    entity_type VARCHAR(100) NOT NULL,
    pattern TEXT NOT NULL,
    correction TEXT NOT NULL,
    hit_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_knowledge_cache_tenant ON knowledge_cache(tenant_id);
