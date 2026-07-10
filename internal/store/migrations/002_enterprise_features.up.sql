-- TrustVault Enterprise Features
-- Migration: 002_enterprise_features.up.sql

-- Remediation Actions
CREATE TABLE remediation_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    dataset_id VARCHAR(255),
    reason TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    executed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_remediation_actions_tenant ON remediation_actions(tenant_id, status);

-- Reports
CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    type VARCHAR(100) NOT NULL,
    status VARCHAR(50) DEFAULT 'generating',
    date_from TIMESTAMPTZ,
    date_to TIMESTAMPTZ,
    file_path VARCHAR(500),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_reports_tenant ON reports(tenant_id, created_at DESC);

-- Label Rules (auto-assign labels based on classification)
CREATE TABLE label_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    classification VARCHAR(100) NOT NULL,
    label VARCHAR(50) NOT NULL,
    priority INT DEFAULT 0,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_label_rules_tenant ON label_rules(tenant_id, active);

-- RoPA (Records of Processing Activities)
CREATE TABLE ropa (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    purpose TEXT,
    legal_basis VARCHAR(100),
    data_categories JSONB DEFAULT '[]',
    recipients JSONB DEFAULT '[]',
    retention_period VARCHAR(100),
    security_measures TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ropa_tenant ON ropa(tenant_id);

-- Playbooks (remediation playbooks)
CREATE TABLE playbooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    issue_type VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    steps JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_playbooks_tenant ON playbooks(tenant_id, issue_type);

-- Model Lineage (AI model training data lineage)
CREATE TABLE model_lineage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    model_id VARCHAR(255) NOT NULL,
    dataset_id VARCHAR(255) NOT NULL,
    usage_type VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_model_lineage_model ON model_lineage(tenant_id, model_id);
CREATE INDEX idx_model_lineage_dataset ON model_lineage(tenant_id, dataset_id);

-- Integration Logs
CREATE TABLE integration_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    integration_id UUID REFERENCES integrations(id) ON DELETE CASCADE,
    level VARCHAR(20) NOT NULL,
    message TEXT,
    details JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_integration_logs_integration ON integration_logs(integration_id, created_at DESC);

-- Data Flows (lineage between datasets)
CREATE TABLE data_flows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    source_dataset_id VARCHAR(255) NOT NULL,
    target_dataset_id VARCHAR(255) NOT NULL,
    flow_type VARCHAR(50),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_data_flows_tenant ON data_flows(tenant_id);
CREATE INDEX idx_data_flows_source ON data_flows(tenant_id, source_dataset_id);
CREATE INDEX idx_data_flows_target ON data_flows(tenant_id, target_dataset_id);

-- Duplicate Groups (for ROT detection)
CREATE TABLE duplicate_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    hash VARCHAR(64) NOT NULL,
    dataset_ids JSONB DEFAULT '[]',
    total_size_bytes BIGINT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_duplicate_groups_tenant ON duplicate_groups(tenant_id);

-- Review Queue (documents pending review)
CREATE TABLE review_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    document_id VARCHAR(255) NOT NULL,
    document_name VARCHAR(255),
    status VARCHAR(50) DEFAULT 'pending',
    classification_results JSONB DEFAULT '{}',
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_review_queue_tenant ON review_queue(tenant_id, status);

-- Retention Policies
CREATE TABLE retention_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    classification VARCHAR(100),
    retention_days INT NOT NULL,
    action VARCHAR(50) DEFAULT 'archive',
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_retention_policies_tenant ON retention_policies(tenant_id, active);

-- Retention Violations
CREATE TABLE retention_violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    dataset_id VARCHAR(255) NOT NULL,
    policy_id UUID REFERENCES retention_policies(id) ON DELETE SET NULL,
    violation_type VARCHAR(50) NOT NULL,
    days_overdue INT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_retention_violations_tenant ON retention_violations(tenant_id);

-- Classification Models (available models)
CREATE TABLE classification_models (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    size VARCHAR(50),
    accuracy FLOAT,
    speed VARCHAR(50),
    is_default BOOLEAN DEFAULT FALSE,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default models
INSERT INTO classification_models (id, name, size, accuracy, speed, is_default) VALUES
    ('gliner-pii-edge-int8', 'GLiNER PII Edge (INT8)', '197MB', 0.96, '4M chars/sec', TRUE),
    ('gliner-pii-base-fp16', 'GLiNER PII Base (FP16)', '330MB', 0.98, '2M chars/sec', FALSE);

-- Insert default label rules
INSERT INTO label_rules (tenant_id, classification, label, priority) VALUES
    ('00000000-0000-0000-0000-000000000001', 'PII', 'CONFIDENTIAL', 10),
    ('00000000-0000-0000-0000-000000000001', 'PHI', 'RESTRICTED', 20),
    ('00000000-0000-0000-0000-000000000001', 'PCI', 'HIGHLY_CONFIDENTIAL', 30);

-- Insert default playbooks
INSERT INTO playbooks (tenant_id, issue_type, name, steps) VALUES
    ('00000000-0000-0000-0000-000000000001', 'pii_exposure', 'PII Exposure Remediation', 
     '["Identify affected datasets", "Assess exposure scope", "Apply redaction policies", "Notify stakeholders", "Document remediation"]'),
    ('00000000-0000-0000-0000-000000000001', 'retention_violation', 'Retention Violation Remediation',
     '["Review retention policy", "Identify overdue data", "Archive or delete as required", "Update retention schedules", "Verify compliance"]'),
    ('00000000-0000-0000-0000-000000000001', 'compliance_gap', 'Compliance Gap Remediation',
     '["Identify compliance requirement", "Assess current state", "Create remediation plan", "Implement controls", "Verify compliance"]');
