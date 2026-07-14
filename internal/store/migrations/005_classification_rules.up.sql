-- Classification rules for enterprise classification pipeline
CREATE TABLE IF NOT EXISTS classification_rules (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('override', 'pattern', 'whitelist', 'threshold')),
    column_pattern TEXT,
    value_pattern TEXT,
    entity_type VARCHAR(100),
    confidence DECIMAL(5,4) DEFAULT 0.95,
    priority INTEGER DEFAULT 0,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_classification_rules_tenant ON classification_rules(tenant_id);
CREATE INDEX idx_classification_rules_active ON classification_rules(tenant_id, active, priority DESC);
CREATE INDEX idx_classification_rules_type ON classification_rules(tenant_id, type);

-- Add label_id to classifications for tracking which label was applied
ALTER TABLE classifications ADD COLUMN IF NOT EXISTS label_id VARCHAR(36) REFERENCES labels(id);
ALTER TABLE classifications ADD COLUMN IF NOT EXISTS rule_id VARCHAR(36) REFERENCES classification_rules(id);
ALTER TABLE classifications ADD COLUMN IF NOT EXISTS classification_source VARCHAR(50) DEFAULT 'pattern_matching';

-- Add overall_label to datasources for quick access
ALTER TABLE datasources ADD COLUMN IF NOT EXISTS sensitivity_label VARCHAR(50);
