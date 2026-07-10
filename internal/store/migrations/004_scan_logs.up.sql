-- Scan logs table for storing scan history
CREATE TABLE IF NOT EXISTS scan_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    datasource_id UUID REFERENCES datasources(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL, -- 'running', 'success', 'failed'
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    message TEXT,
    logs JSONB DEFAULT '[]', -- Array of log entries
    datasets_discovered INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_scan_logs_datasource ON scan_logs(datasource_id);
CREATE INDEX idx_scan_logs_tenant ON scan_logs(tenant_id);
CREATE INDEX idx_scan_logs_started_at ON scan_logs(started_at DESC);
