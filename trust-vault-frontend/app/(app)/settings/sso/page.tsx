'use client'

import { useState, useEffect } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { api } from '@/lib/api'
import { toast } from 'sonner'
import { Plus, Trash2, Edit2, Shield, Globe, Key, Copy, Check } from 'lucide-react'

interface SSOProvider {
  id: string
  name: string
  type: 'oidc' | 'saml'
  enabled: boolean
  issuer_url?: string
  client_id?: string
  scopes?: string[]
  idp_metadata_url?: string
  idp_entity_id?: string
  idp_sso_url?: string
  sp_entity_id?: string
  default_role: string
  auto_create_users: boolean
  created_at: string
}

export default function SSOSettingsPage() {
  const [providers, setProviders] = useState<SSOProvider[]>([])
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [editingProvider, setEditingProvider] = useState<SSOProvider | null>(null)
  const [copiedId, setCopiedId] = useState<string | null>(null)

  useEffect(() => {
    fetchProviders()
  }, [])

  const fetchProviders = async () => {
    try {
      const response = await api.get('/admin/sso/providers')
      setProviders(response.data || [])
    } catch (error) {
      toast.error('Failed to load SSO providers')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this SSO provider?')) return
    try {
      await api.delete(`/admin/sso/providers/${id}`)
      toast.success('SSO provider deleted')
      fetchProviders()
    } catch (error) {
      toast.error('Failed to delete SSO provider')
    }
  }

  const handleToggle = async (provider: SSOProvider) => {
    try {
      await api.put(`/admin/sso/providers/${provider.id}`, {
        enabled: !provider.enabled
      })
      toast.success(`SSO provider ${provider.enabled ? 'disabled' : 'enabled'}`)
      fetchProviders()
    } catch (error) {
      toast.error('Failed to update SSO provider')
    }
  }

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text)
    setCopiedId(id)
    setTimeout(() => setCopiedId(null), 2000)
  }

  const baseUrl = typeof window !== 'undefined' 
    ? `${window.location.protocol}//${window.location.host}`.replace(':3000', ':8080')
    : 'https://api.securelens.ai'

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[
          { label: 'Settings', href: '/settings' },
          { label: 'Single Sign-On', active: true }
        ]} />
        <div className="flex items-center justify-between mt-4">
          <div>
            <h1 className="text-3xl font-bold text-foreground">Single Sign-On (SSO)</h1>
            <p className="text-sm text-muted-foreground mt-1">
              Configure OIDC and SAML identity providers for your organization
            </p>
          </div>
          <button
            onClick={() => { setEditingProvider(null); setShowModal(true) }}
            className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
          >
            <Plus className="h-4 w-4" />
            Add Provider
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="p-8">
        {loading ? (
          <div className="text-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          </div>
        ) : providers.length === 0 ? (
          <div className="text-center py-12 border border-dashed border-border rounded-lg">
            <Shield className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <h3 className="text-lg font-semibold text-foreground">No SSO Providers</h3>
            <p className="text-sm text-muted-foreground mt-1 mb-4">
              Add an OIDC or SAML provider to enable single sign-on
            </p>
            <button
              onClick={() => setShowModal(true)}
              className="px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
            >
              Add Your First Provider
            </button>
          </div>
        ) : (
          <div className="space-y-4">
            {providers.map((provider) => (
              <div
                key={provider.id}
                className="border border-border rounded-lg bg-card p-6"
              >
                <div className="flex items-start justify-between">
                  <div className="flex items-start gap-4">
                    <div className={`p-3 rounded-lg ${provider.type === 'oidc' ? 'bg-blue-500/10' : 'bg-green-500/10'}`}>
                      {provider.type === 'oidc' ? (
                        <Globe className={`h-6 w-6 ${provider.type === 'oidc' ? 'text-blue-500' : 'text-green-500'}`} />
                      ) : (
                        <Key className="h-6 w-6 text-green-500" />
                      )}
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <h3 className="text-lg font-semibold text-foreground">{provider.name}</h3>
                        <span className={`px-2 py-0.5 text-xs rounded-full ${
                          provider.enabled 
                            ? 'bg-green-500/10 text-green-500' 
                            : 'bg-muted text-muted-foreground'
                        }`}>
                          {provider.enabled ? 'Enabled' : 'Disabled'}
                        </span>
                        <span className="px-2 py-0.5 text-xs rounded-full bg-muted text-muted-foreground uppercase">
                          {provider.type}
                        </span>
                      </div>
                      <div className="mt-2 space-y-1 text-sm text-muted-foreground">
                        {provider.type === 'oidc' && provider.issuer_url && (
                          <p>Issuer: {provider.issuer_url}</p>
                        )}
                        {provider.type === 'saml' && provider.idp_entity_id && (
                          <p>IdP Entity ID: {provider.idp_entity_id}</p>
                        )}
                        <p>Default Role: {provider.default_role}</p>
                        <p>Auto-create Users: {provider.auto_create_users ? 'Yes' : 'No'}</p>
                      </div>
                      
                      {/* Configuration URLs */}
                      <div className="mt-4 p-3 bg-muted/50 rounded-lg">
                        <p className="text-xs font-medium text-foreground mb-2">Configuration URLs</p>
                        {provider.type === 'oidc' && (
                          <div className="flex items-center gap-2 text-xs">
                            <span className="text-muted-foreground">Callback URL:</span>
                            <code className="bg-background px-2 py-1 rounded">{baseUrl}/api/v1/auth/sso/oidc/callback</code>
                            <button
                              onClick={() => copyToClipboard(`${baseUrl}/api/v1/auth/sso/oidc/callback`, `callback-${provider.id}`)}
                              className="p-1 hover:bg-muted rounded"
                            >
                              {copiedId === `callback-${provider.id}` ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
                            </button>
                          </div>
                        )}
                        {provider.type === 'saml' && (
                          <>
                            <div className="flex items-center gap-2 text-xs mb-1">
                              <span className="text-muted-foreground">ACS URL:</span>
                              <code className="bg-background px-2 py-1 rounded">{baseUrl}/api/v1/auth/sso/saml/acs</code>
                              <button
                                onClick={() => copyToClipboard(`${baseUrl}/api/v1/auth/sso/saml/acs`, `acs-${provider.id}`)}
                                className="p-1 hover:bg-muted rounded"
                              >
                                {copiedId === `acs-${provider.id}` ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
                              </button>
                            </div>
                            {provider.sp_entity_id && (
                              <div className="flex items-center gap-2 text-xs">
                                <span className="text-muted-foreground">SP Entity ID:</span>
                                <code className="bg-background px-2 py-1 rounded">{provider.sp_entity_id}</code>
                                <button
                                  onClick={() => copyToClipboard(provider.sp_entity_id!, `sp-${provider.id}`)}
                                  className="p-1 hover:bg-muted rounded"
                                >
                                  {copiedId === `sp-${provider.id}` ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
                                </button>
                              </div>
                            )}
                          </>
                        )}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handleToggle(provider)}
                      className={`px-3 py-1.5 text-sm rounded-lg transition-colors ${
                        provider.enabled
                          ? 'bg-muted text-foreground hover:bg-muted/80'
                          : 'bg-primary text-primary-foreground hover:bg-primary/90'
                      }`}
                    >
                      {provider.enabled ? 'Disable' : 'Enable'}
                    </button>
                    <button
                      onClick={() => { setEditingProvider(provider); setShowModal(true) }}
                      className="p-2 text-muted-foreground hover:text-foreground hover:bg-muted rounded-lg transition-colors"
                    >
                      <Edit2 className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => handleDelete(provider.id)}
                      className="p-2 text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-lg transition-colors"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Modal */}
      {showModal && (
        <SSOProviderModal
          provider={editingProvider}
          onClose={() => { setShowModal(false); setEditingProvider(null) }}
          onSave={() => { setShowModal(false); setEditingProvider(null); fetchProviders() }}
        />
      )}
    </div>
  )
}

interface SSOProviderModalProps {
  provider: SSOProvider | null
  onClose: () => void
  onSave: () => void
}

function SSOProviderModal({ provider, onClose, onSave }: SSOProviderModalProps) {
  const [loading, setLoading] = useState(false)
  const [formData, setFormData] = useState({
    name: provider?.name || '',
    type: provider?.type || 'oidc',
    enabled: provider?.enabled ?? true,
    issuer_url: provider?.issuer_url || '',
    client_id: provider?.client_id || '',
    client_secret: '',
    scopes: provider?.scopes?.join(', ') || 'openid, email, profile',
    idp_metadata_url: provider?.idp_metadata_url || '',
    idp_entity_id: provider?.idp_entity_id || '',
    idp_sso_url: provider?.idp_sso_url || '',
    idp_certificate: '',
    default_role: provider?.default_role || 'user',
    auto_create_users: provider?.auto_create_users ?? true,
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)

    try {
      const payload = {
        ...formData,
        scopes: formData.scopes.split(',').map(s => s.trim()).filter(Boolean),
      }

      if (provider) {
        await api.put(`/admin/sso/providers/${provider.id}`, payload)
        toast.success('SSO provider updated')
      } else {
        await api.post('/admin/sso/providers', payload)
        toast.success('SSO provider created')
      }
      onSave()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Failed to save SSO provider')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-card border border-border rounded-lg w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <div className="p-6 border-b border-border">
          <h2 className="text-xl font-semibold text-foreground">
            {provider ? 'Edit SSO Provider' : 'Add SSO Provider'}
          </h2>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-6">
          {/* Basic Info */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Name</label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground"
                placeholder="e.g., Okta, Azure AD"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Type</label>
              <select
                value={formData.type}
                onChange={(e) => setFormData({ ...formData, type: e.target.value as 'oidc' | 'saml' })}
                className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground"
                disabled={!!provider}
              >
                <option value="oidc">OIDC (OpenID Connect)</option>
                <option value="saml">SAML 2.0</option>
              </select>
            </div>
          </div>

          {/* OIDC Fields */}
          {formData.type === 'oidc' && (
            <>
              <div>
                <label className="block text-sm font-medium text-foreground mb-2">Issuer URL</label>
                <input
                  type="url"
                  value={formData.issuer_url}
                  onChange={(e) => setFormData({ ...formData, issuer_url: e.target.value })}
                  className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground"
                  placeholder="https://your-idp.com"
                  required={formData.type === 'oidc'}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-foreground mb-2">Client ID</label>
                  <input
                    type="text"
                    value={formData.client_id}
                    onChange={(e) => setFormData({ ...formData, client_id: e.target.value })}
                    className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground"
                    required={formData.type === 'oidc'}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-foreground mb-2">
                    Client Secret {provider && '(leave blank to keep existing)'}
                  </label>
                  <input
                    type="password"
                    value={formData.client_secret}
                    onChange={(e) => setFormData({ ...formData, client_secret: e.target.value })}
                    className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground"
                    required={formData.type === 'oidc' && !provider}
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-2">Scopes</label>
                <input
                  type="text"
                  value={formData.scopes}
                  onChange={(e) => setFormData({ ...formData, scopes: e.target.value })}
                  className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground"
                  placeholder="openid, email, profile"
                />
              </div>
            </>
          )}

          {/* SAML Fields */}
          {formData.type === 'saml' && (
            <>
              <div>
                <label className="block text-sm font-medium text-foreground mb-2">IdP Entity ID</label>
                <input
                  type="text"
                  value={formData.idp_entity_id}
                  onChange={(e) => setFormData({ ...formData, idp_entity_id: e.target.value })}
                  className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground"
                  required={formData.type === 'saml'}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-2">IdP SSO URL</label>
                <input
                  type="url"
                  value={formData.idp_sso_url}
                  onChange={(e) => setFormData({ ...formData, idp_sso_url: e.target.value })}
                  className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground"
                  placeholder="https://your-idp.com/sso"
                  required={formData.type === 'saml'}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-2">
                  IdP Certificate (PEM format) {provider && '(leave blank to keep existing)'}
                </label>
                <textarea
                  value={formData.idp_certificate}
                  onChange={(e) => setFormData({ ...formData, idp_certificate: e.target.value })}
                  className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground font-mono text-xs"
                  rows={4}
                  placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
                />
              </div>
            </>
          )}

          {/* Common Fields */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Default Role</label>
              <input
                type="text"
                value={formData.default_role}
                onChange={(e) => setFormData({ ...formData, default_role: e.target.value })}
                className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground"
                placeholder="user"
              />
            </div>
            <div className="flex items-center gap-4 pt-8">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={formData.auto_create_users}
                  onChange={(e) => setFormData({ ...formData, auto_create_users: e.target.checked })}
                  className="w-4 h-4 rounded border-border"
                />
                <span className="text-sm text-foreground">Auto-create users</span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={formData.enabled}
                  onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                  className="w-4 h-4 rounded border-border"
                />
                <span className="text-sm text-foreground">Enabled</span>
              </label>
            </div>
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-3 pt-4 border-t border-border">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-foreground bg-muted rounded-lg hover:bg-muted/80 transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 disabled:opacity-50 transition-colors"
            >
              {loading ? 'Saving...' : (provider ? 'Update' : 'Create')}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
