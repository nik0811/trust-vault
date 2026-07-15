-- Endpoint scans: API endpoint registration and PII scanning
CREATE TABLE IF NOT EXISTS endpoint_scans (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  url TEXT NOT NULL,
  method VARCHAR(10) DEFAULT 'GET',
  headers JSONB DEFAULT '{}',
  auth_type VARCHAR(50) DEFAULT 'none',
  auth_config JSONB DEFAULT '{}',
  status VARCHAR(50) DEFAULT 'pending',
  last_scan TIMESTAMPTZ,
  findings JSONB DEFAULT '[]',
  risk_level VARCHAR(20) DEFAULT 'unknown',
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_endpoint_scans_tenant ON endpoint_scans(tenant_id);

-- Consent preferences: per-subject preference storage
CREATE TABLE IF NOT EXISTS consent_preferences (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
  subject_id VARCHAR(255) NOT NULL,
  preferences JSONB DEFAULT '{}',
  ip VARCHAR(45),
  updated_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(tenant_id, subject_id)
);
CREATE INDEX IF NOT EXISTS idx_consent_preferences_tenant ON consent_preferences(tenant_id);
CREATE INDEX IF NOT EXISTS idx_consent_preferences_subject ON consent_preferences(subject_id);
