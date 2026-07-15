CREATE TABLE IF NOT EXISTS endpoint_agents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  hostname TEXT NOT NULL,
  ip TEXT,
  os TEXT,
  agent_version TEXT,
  status TEXT DEFAULT 'active',
  last_seen_at TIMESTAMPTZ,
  last_scan_at TIMESTAMPTZ,
  scan_results JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_endpoint_agents_tenant ON endpoint_agents(tenant_id);

CREATE TABLE IF NOT EXISTS residency_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  name TEXT NOT NULL,
  regulation TEXT,
  allowed_regions TEXT[],
  data_types TEXT[],
  active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_residency_rules_tenant ON residency_rules(tenant_id);

ALTER TABLE datasources ADD COLUMN IF NOT EXISTS region TEXT;
ALTER TABLE datasources ADD COLUMN IF NOT EXISTS country TEXT;

CREATE TABLE IF NOT EXISTS consent_widget_configs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL UNIQUE,
  primary_color TEXT DEFAULT '#6366f1',
  background_color TEXT DEFAULT '#ffffff',
  text_color TEXT DEFAULT '#111827',
  banner_title TEXT DEFAULT 'We value your privacy',
  banner_text TEXT DEFAULT 'We use cookies and similar technologies to improve your experience.',
  accept_label TEXT DEFAULT 'Accept All',
  reject_label TEXT DEFAULT 'Reject Non-Essential',
  purposes JSONB DEFAULT '[]',
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
