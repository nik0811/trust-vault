-- DPIA (Data Protection Impact Assessment) table
CREATE TABLE IF NOT EXISTS dpias (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  name TEXT NOT NULL,
  description TEXT,
  data_types JSONB DEFAULT '[]',
  processing_purpose TEXT,
  risk_level TEXT DEFAULT 'medium',
  status TEXT DEFAULT 'in_progress',
  steps JSONB DEFAULT '[]',
  dpo_consulted BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dpias_tenant ON dpias(tenant_id);

-- Consent records table for proper tracking
CREATE TABLE IF NOT EXISTS consent_records (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  subject_id TEXT NOT NULL,
  purpose TEXT NOT NULL,
  status TEXT DEFAULT 'granted',
  ip TEXT,
  source TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_consent_records_tenant ON consent_records(tenant_id);
CREATE INDEX IF NOT EXISTS idx_consent_records_subject ON consent_records(subject_id);

-- Critical Data Elements table
CREATE TABLE IF NOT EXISTS critical_data_elements (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID,
  column_name TEXT NOT NULL,
  table_name TEXT NOT NULL,
  business_definition TEXT,
  data_owner TEXT,
  criticality TEXT DEFAULT 'medium',
  quality_score FLOAT DEFAULT 0,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_cde_tenant ON critical_data_elements(tenant_id);

-- Data profiles table for ruleless profiling
CREATE TABLE IF NOT EXISTS data_profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  profile_data JSONB DEFAULT '{}',
  status TEXT DEFAULT 'pending',
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_data_profiles_tenant ON data_profiles(tenant_id);
CREATE INDEX IF NOT EXISTS idx_data_profiles_datasource ON data_profiles(datasource_id);

-- Document classifications linking table
CREATE TABLE IF NOT EXISTS document_classifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  document_id TEXT NOT NULL,
  document_name TEXT,
  entity_types JSONB DEFAULT '[]',
  findings JSONB DEFAULT '[]',
  governed BOOLEAN DEFAULT false,
  label_applied TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_doc_class_tenant ON document_classifications(tenant_id);
CREATE INDEX IF NOT EXISTS idx_doc_class_document ON document_classifications(document_id);
