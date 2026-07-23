-- SSO Providers table for OIDC and SAML configuration
CREATE TABLE IF NOT EXISTS sso_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('oidc', 'saml')),
    enabled BOOLEAN NOT NULL DEFAULT true,
    
    -- OIDC fields
    issuer_url TEXT,
    client_id TEXT,
    client_secret_encrypted TEXT,
    scopes TEXT[] DEFAULT ARRAY['openid', 'email', 'profile'],
    
    -- SAML fields
    idp_metadata_url TEXT,
    idp_entity_id TEXT,
    idp_sso_url TEXT,
    idp_certificate TEXT,
    sp_entity_id TEXT,
    
    -- Common fields
    attribute_mapping JSONB DEFAULT '{}',
    default_role VARCHAR(100) DEFAULT 'user',
    auto_create_users BOOLEAN DEFAULT true,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_provider_name_per_tenant UNIQUE (tenant_id, name)
);

-- Index for quick lookups
CREATE INDEX idx_sso_providers_tenant ON sso_providers(tenant_id);
CREATE INDEX idx_sso_providers_enabled ON sso_providers(tenant_id, enabled) WHERE enabled = true;

-- SSO sessions for tracking SSO login state
CREATE TABLE IF NOT EXISTS sso_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES sso_providers(id) ON DELETE CASCADE,
    state VARCHAR(255) NOT NULL UNIQUE,
    nonce VARCHAR(255),
    redirect_uri TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '10 minutes'
);

-- Index for state lookups and cleanup
CREATE INDEX idx_sso_sessions_state ON sso_sessions(state);
CREATE INDEX idx_sso_sessions_expires ON sso_sessions(expires_at);

-- Add sso_provider_id to users table to track SSO-created users
ALTER TABLE users ADD COLUMN IF NOT EXISTS sso_provider_id UUID REFERENCES sso_providers(id) ON DELETE SET NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS sso_subject_id VARCHAR(255);

-- Index for SSO user lookups
CREATE INDEX IF NOT EXISTS idx_users_sso ON users(tenant_id, sso_provider_id, sso_subject_id) WHERE sso_provider_id IS NOT NULL;
