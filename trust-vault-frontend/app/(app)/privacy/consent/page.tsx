'use client'

import { useState, useEffect } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { CheckCircle, XCircle, Plus, Settings, Code, User, Eye, Copy } from 'lucide-react'
import { useRecordConsent, useWithdrawConsent } from '@/hooks/use-privacy'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || ''

function authHeaders() {
  const token = typeof window !== 'undefined' ? localStorage.getItem('token') : ''
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

const purposes = [
  'Marketing communications',
  'Analytics and performance',
  'Personalization',
  'Third-party sharing',
  'AI/ML training',
  'Research and development',
]

function PreferenceCenterTab() {
  const [subjectId, setSubjectId] = useState('')
  const [prefs, setPrefs] = useState<any>(null)
  const [loading, setLoading] = useState(false)
  const [saved, setSaved] = useState(false)

  const lookupPrefs = async () => {
    if (!subjectId.trim()) return
    setLoading(true)
    try {
      const res = await fetch(`${API_BASE}/api/v1/consent/preferences/${encodeURIComponent(subjectId)}`, { headers: authHeaders() })
      const data = await res.json()
      setPrefs(data.preferences || {})
    } finally {
      setLoading(false)
    }
  }

  const savePrefs = async () => {
    if (!subjectId || !prefs) return
    await fetch(`${API_BASE}/api/v1/consent/preferences/${encodeURIComponent(subjectId)}`, {
      method: 'PUT',
      headers: authHeaders(),
      body: JSON.stringify(prefs),
    })
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  return (
    <div className="space-y-6">
      <div className="rounded-lg border border-border bg-card p-6">
        <div className="flex items-center gap-3 mb-4">
          <User className="h-5 w-5 text-primary" />
          <h3 className="text-lg font-semibold">Lookup Subject Preferences</h3>
        </div>
        <div className="flex gap-3">
          <input
            type="text"
            value={subjectId}
            onChange={e => setSubjectId(e.target.value)}
            placeholder="Email or Subject ID"
            className="flex-1 px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
          />
          <button
            onClick={lookupPrefs}
            disabled={loading}
            className="px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 disabled:opacity-50"
          >
            {loading ? 'Loading…' : 'Lookup'}
          </button>
        </div>
      </div>

      {prefs && (
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-base font-semibold mb-4">Preferences for: <span className="text-primary">{subjectId}</span></h3>
          <div className="space-y-3">
            {[
              { key: 'necessary', label: 'Necessary', desc: 'Required for the service to function', locked: true },
              { key: 'analytics', label: 'Analytics', desc: 'Help us understand how the service is used' },
              { key: 'marketing', label: 'Marketing', desc: 'Personalized ads and communications' },
              { key: 'personalization', label: 'Personalization', desc: 'Customized experience based on usage' },
            ].map(item => (
              <div key={item.key} className="flex items-center justify-between p-3 rounded-lg bg-muted/40">
                <div>
                  <p className="text-sm font-medium">{item.label}</p>
                  <p className="text-xs text-muted-foreground">{item.desc}</p>
                </div>
                <button
                  disabled={item.locked}
                  onClick={() => setPrefs((p: any) => ({ ...p, [item.key]: !p[item.key] }))}
                  className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${prefs[item.key] ? 'bg-primary' : 'bg-gray-300'} ${item.locked ? 'opacity-60 cursor-not-allowed' : 'cursor-pointer'}`}
                >
                  <span className={`inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform ${prefs[item.key] ? 'translate-x-6' : 'translate-x-1'}`} />
                </button>
              </div>
            ))}
          </div>
          <div className="flex justify-end mt-4">
            <button onClick={savePrefs} className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90">
              {saved ? <><CheckCircle className="h-4 w-4" /> Saved!</> : 'Save Preferences'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

function WidgetConfigTab() {
  const [config, setConfig] = useState({
    primary_color: '#6366f1',
    background_color: '#ffffff',
    text_color: '#111827',
    banner_title: 'We value your privacy',
    banner_text: 'We use cookies and similar technologies to improve your experience.',
    accept_label: 'Accept All',
    reject_label: 'Reject Non-Essential',
  })
  const [loading, setLoading] = useState(false)
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    fetch(`${API_BASE}/api/v1/consent/widget-config`, { headers: authHeaders() })
      .then(r => r.json())
      .then(data => {
        if (data && data.primary_color) setConfig(data)
      })
      .catch(() => {})
  }, [])

  const save = async () => {
    setLoading(true)
    await fetch(`${API_BASE}/api/v1/consent/widget-config`, {
      method: 'PUT',
      headers: authHeaders(),
      body: JSON.stringify(config),
    })
    setLoading(false)
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  const fields = [
    { key: 'banner_title', label: 'Banner Title', type: 'text' },
    { key: 'banner_text', label: 'Banner Text', type: 'textarea' },
    { key: 'accept_label', label: 'Accept Button Label', type: 'text' },
    { key: 'reject_label', label: 'Reject Button Label', type: 'text' },
    { key: 'primary_color', label: 'Primary Color', type: 'color' },
    { key: 'background_color', label: 'Background Color', type: 'color' },
    { key: 'text_color', label: 'Text Color', type: 'color' },
  ]

  return (
    <div className="grid grid-cols-2 gap-6">
      <div className="rounded-lg border border-border bg-card p-6">
        <div className="flex items-center gap-3 mb-4">
          <Settings className="h-5 w-5 text-primary" />
          <h3 className="text-lg font-semibold">Widget Configuration</h3>
        </div>
        <div className="space-y-4">
          {fields.map(f => (
            <div key={f.key}>
              <label className="block text-sm font-medium mb-1">{f.label}</label>
              {f.type === 'textarea' ? (
                <textarea
                  rows={3}
                  value={(config as any)[f.key]}
                  onChange={e => setConfig(c => ({ ...c, [f.key]: e.target.value }))}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary text-sm"
                />
              ) : f.type === 'color' ? (
                <div className="flex items-center gap-3">
                  <input
                    type="color"
                    value={(config as any)[f.key]}
                    onChange={e => setConfig(c => ({ ...c, [f.key]: e.target.value }))}
                    className="h-9 w-16 rounded border border-border cursor-pointer"
                  />
                  <span className="text-sm font-mono text-muted-foreground">{(config as any)[f.key]}</span>
                </div>
              ) : (
                <input
                  type="text"
                  value={(config as any)[f.key]}
                  onChange={e => setConfig(c => ({ ...c, [f.key]: e.target.value }))}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary text-sm"
                />
              )}
            </div>
          ))}
          <button
            onClick={save}
            disabled={loading}
            className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 disabled:opacity-50"
          >
            {saved ? <><CheckCircle className="h-4 w-4" /> Saved!</> : loading ? 'Saving…' : 'Save Configuration'}
          </button>
        </div>
      </div>

      <div className="rounded-lg border border-border bg-card p-6">
        <div className="flex items-center gap-3 mb-4">
          <Eye className="h-5 w-5 text-primary" />
          <h3 className="text-lg font-semibold">Preview</h3>
        </div>
        <div className="relative rounded-xl border-2 border-dashed border-border bg-muted/20 min-h-48 overflow-hidden">
          <div className="absolute inset-0 flex items-end">
            <div
              className="w-full p-4 shadow-lg rounded-t-xl"
              style={{ backgroundColor: config.background_color, color: config.text_color }}
            >
              <p className="font-semibold text-sm mb-1">{config.banner_title}</p>
              <p className="text-xs opacity-80 mb-3">{config.banner_text}</p>
              <div className="flex gap-2">
                <button
                  className="flex-1 py-2 rounded-lg text-xs font-medium text-white"
                  style={{ backgroundColor: config.primary_color }}
                >
                  {config.accept_label}
                </button>
                <button
                  className="flex-1 py-2 rounded-lg text-xs font-medium border"
                  style={{ borderColor: config.primary_color, color: config.primary_color }}
                >
                  {config.reject_label}
                </button>
              </div>
            </div>
          </div>
          <div className="p-4 text-xs text-muted-foreground">Website content area</div>
        </div>
      </div>
    </div>
  )
}

function EmbedCodeTab() {
  const [embedCode, setEmbedCode] = useState('')
  const [tenantId, setTenantId] = useState('')
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    fetch(`${API_BASE}/api/v1/consent/embed-code`, { headers: authHeaders() })
      .then(r => r.json())
      .then(data => {
        setEmbedCode(data.embed_code || '')
        setTenantId(data.tenant_id || '')
      })
      .catch(() => {})
  }, [])

  const copy = (text: string) => {
    navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const fullEmbed = embedCode || `<script src="${API_BASE}/api/v1/consent/widget.js?tenant=YOUR_TENANT_ID"></script>`

  return (
    <div className="space-y-6">
      <div className="rounded-lg border border-border bg-card p-6">
        <div className="flex items-center gap-3 mb-4">
          <Code className="h-5 w-5 text-primary" />
          <h3 className="text-lg font-semibold">Embed Code</h3>
        </div>
        <p className="text-sm text-muted-foreground mb-4">
          Add this snippet to your website's <code className="bg-muted px-1 rounded">&lt;head&gt;</code> or before the closing <code className="bg-muted px-1 rounded">&lt;/body&gt;</code> tag.
          The consent banner will appear automatically for new visitors.
        </p>
        <div className="relative bg-gray-900 rounded-xl p-5">
          <button
            onClick={() => copy(fullEmbed)}
            className="absolute top-3 right-3 flex items-center gap-1 text-gray-400 hover:text-white text-xs px-2 py-1 rounded bg-gray-800"
          >
            {copied ? <><CheckCircle className="w-3 h-3 text-green-400" /> Copied</> : <><Copy className="w-3 h-3" /> Copy</>}
          </button>
          <pre className="text-green-400 font-mono text-sm overflow-x-auto">{fullEmbed}</pre>
        </div>
      </div>

      <div className="rounded-lg border border-border bg-card p-6">
        <h3 className="text-base font-semibold mb-4">Integration Examples</h3>
        <div className="space-y-4">
          {[
            {
              title: 'React / Next.js',
              code: `import Script from 'next/script'\n\n<Script src="${API_BASE}/api/v1/consent/widget.js?tenant=${tenantId || 'YOUR_TENANT_ID'}" strategy="afterInteractive" />`,
            },
            {
              title: 'WordPress (functions.php)',
              code: `function add_consent_widget() {\n  echo '<script src="${API_BASE}/api/v1/consent/widget.js?tenant=${tenantId || 'YOUR_TENANT_ID'}"></script>';\n}\nadd_action('wp_footer', 'add_consent_widget');`,
            },
            {
              title: 'Google Tag Manager',
              code: `// Create a Custom HTML tag with:\n<script src="${API_BASE}/api/v1/consent/widget.js?tenant=${tenantId || 'YOUR_TENANT_ID'}"></script>\n// Trigger: All Pages`,
            },
          ].map(ex => (
            <div key={ex.title}>
              <p className="text-sm font-medium mb-2">{ex.title}</p>
              <div className="relative bg-gray-900 rounded-lg p-4">
                <button
                  onClick={() => copy(ex.code)}
                  className="absolute top-2 right-2 text-gray-400 hover:text-white"
                >
                  <Copy className="w-3.5 h-3.5" />
                </button>
                <pre className="text-xs text-gray-300 font-mono overflow-x-auto whitespace-pre-wrap">{ex.code}</pre>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

export default function ConsentPage() {
  const recordConsent = useRecordConsent()
  const withdrawConsent = useWithdrawConsent()
  const [formData, setFormData] = useState({ subject_id: '', purpose: '' })
  const [withdrawId, setWithdrawId] = useState('')
  const [activeTab, setActiveTab] = useState<'manage' | 'preferences' | 'widget' | 'embed'>('manage')

  const handleRecord = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await recordConsent.mutateAsync(formData)
      setFormData({ subject_id: '', purpose: '' })
    } catch {}
  }

  const handleWithdraw = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!withdrawId.trim()) return
    try {
      await withdrawConsent.mutateAsync(withdrawId)
      setWithdrawId('')
    } catch {}
  }

  const tabs = [
    { id: 'manage', label: 'Consent Management' },
    { id: 'preferences', label: 'Preference Center' },
    { id: 'widget', label: 'Widget Configuration' },
    { id: 'embed', label: 'Embed Code' },
  ] as const

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Privacy', href: '/privacy' }, { label: 'Consent', active: true }]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">Consent Management</h1>
        <p className="text-sm text-muted-foreground mt-1">Track, manage and configure user consent for data processing</p>
        <div className="flex gap-1 mt-4">
          {tabs.map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`px-4 py-2 text-sm rounded-md font-medium ${activeTab === tab.id ? 'bg-primary text-primary-foreground' : 'hover:bg-muted text-muted-foreground'}`}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      <div className="p-8">
        {activeTab === 'manage' && (
          <div className="space-y-8">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
              <div className="rounded-lg border border-border bg-card p-6">
                <div className="flex items-center gap-3 mb-4">
                  <CheckCircle className="h-6 w-6 text-green-500" />
                  <h3 className="text-lg font-semibold">Record Consent</h3>
                </div>
                <form onSubmit={handleRecord} className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium mb-1">Subject ID</label>
                    <input
                      type="text"
                      value={formData.subject_id}
                      onChange={e => setFormData({ ...formData, subject_id: e.target.value })}
                      placeholder="user@example.com or user ID"
                      className="w-full px-3 py-2 rounded-lg border border-border bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                      required
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">Purpose</label>
                    <select
                      value={formData.purpose}
                      onChange={e => setFormData({ ...formData, purpose: e.target.value })}
                      className="w-full px-3 py-2 rounded-lg border border-border bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                      required
                    >
                      <option value="">Select purpose…</option>
                      {purposes.map(p => <option key={p} value={p}>{p}</option>)}
                    </select>
                  </div>
                  <button
                    type="submit"
                    disabled={recordConsent.isPending}
                    className="w-full flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-green-600 text-white hover:bg-green-700 disabled:opacity-50"
                  >
                    <Plus className="h-4 w-4" />
                    {recordConsent.isPending ? 'Recording…' : 'Record Consent'}
                  </button>
                </form>
              </div>

              <div className="rounded-lg border border-border bg-card p-6">
                <div className="flex items-center gap-3 mb-4">
                  <XCircle className="h-6 w-6 text-red-500" />
                  <h3 className="text-lg font-semibold">Withdraw Consent</h3>
                </div>
                <form onSubmit={handleWithdraw} className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium mb-1">Subject ID</label>
                    <input
                      type="text"
                      value={withdrawId}
                      onChange={e => setWithdrawId(e.target.value)}
                      placeholder="user@example.com or user ID"
                      className="w-full px-3 py-2 rounded-lg border border-border bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                      required
                    />
                  </div>
                  <p className="text-sm text-muted-foreground">This will withdraw all consent for the specified subject.</p>
                  <button
                    type="submit"
                    disabled={withdrawConsent.isPending}
                    className="w-full flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-red-600 text-white hover:bg-red-700 disabled:opacity-50"
                  >
                    <XCircle className="h-4 w-4" />
                    {withdrawConsent.isPending ? 'Withdrawing…' : 'Withdraw Consent'}
                  </button>
                </form>
              </div>
            </div>

            <div className="rounded-lg border border-border bg-card p-6">
              <h3 className="text-lg font-semibold mb-4">Consent Purposes</h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                {purposes.map(purpose => (
                  <div key={purpose} className="p-4 rounded-lg bg-muted/50">
                    <p className="font-medium">{purpose}</p>
                    <p className="text-sm text-muted-foreground mt-1">Requires explicit user consent</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {activeTab === 'preferences' && <PreferenceCenterTab />}
        {activeTab === 'widget' && <WidgetConfigTab />}
        {activeTab === 'embed' && <EmbedCodeTab />}
      </div>
    </div>
  )
}
