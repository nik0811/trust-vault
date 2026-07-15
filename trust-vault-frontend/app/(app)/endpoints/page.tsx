'use client'

import { useState, useEffect } from 'react'
import {
  Globe, RefreshCw, Plus, Scan, Trash2, ChevronDown, ChevronUp,
  CheckCircle, Clock, AlertCircle, Shield, AlertTriangle, X, Monitor, Copy
} from 'lucide-react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || ''

function authHeaders() {
  const token = typeof window !== 'undefined' ? localStorage.getItem('token') : ''
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

// ── Risk / Status badges ──────────────────────────────────────────────────

function RiskBadge({ level }: { level: string }) {
  const map: Record<string, string> = {
    critical: 'bg-red-100 text-red-800 border border-red-200',
    high: 'bg-orange-100 text-orange-800 border border-orange-200',
    medium: 'bg-yellow-100 text-yellow-800 border border-yellow-200',
    low: 'bg-green-100 text-green-800 border border-green-200',
    unknown: 'bg-gray-100 text-gray-600 border border-gray-200',
  }
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${map[level] || map.unknown}`}>
      {level.charAt(0).toUpperCase() + level.slice(1)}
    </span>
  )
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { color: string; icon: React.ReactNode }> = {
    pending: { color: 'bg-gray-100 text-gray-600', icon: <Clock className="w-3 h-3" /> },
    scanned: { color: 'bg-green-100 text-green-700', icon: <CheckCircle className="w-3 h-3" /> },
    scanning: { color: 'bg-blue-100 text-blue-700', icon: <RefreshCw className="w-3 h-3 animate-spin" /> },
    error: { color: 'bg-red-100 text-red-700', icon: <AlertCircle className="w-3 h-3" /> },
  }
  const s = map[status] || map.pending
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${s.color}`}>
      {s.icon} {status}
    </span>
  )
}

// ── Add Endpoint Modal ────────────────────────────────────────────────────

function AddEndpointModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const [form, setForm] = useState({ name: '', url: '', method: 'GET', auth_type: 'none', token: '', username: '', password: '', api_key: '' })
  const [loading, setLoading] = useState(false)
  const [err, setErr] = useState('')

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setErr('')
    try {
      const auth_config: Record<string, string> = {}
      if (form.auth_type === 'bearer') auth_config.token = form.token
      if (form.auth_type === 'basic') { auth_config.username = form.username; auth_config.password = form.password }
      if (form.auth_type === 'api_key') auth_config.key = form.api_key

      const res = await fetch(`${API_BASE}/endpoints`, {
        method: 'POST',
        headers: authHeaders(),
        body: JSON.stringify({ name: form.name, url: form.url, method: form.method, auth_type: form.auth_type, auth_config }),
      })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      onCreated()
      onClose()
    } catch (e: any) {
      setErr(e.message || 'Failed to create endpoint')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-card border border-border rounded-xl shadow-xl w-full max-w-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">Add API Endpoint</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground"><X className="w-5 h-5" /></button>
        </div>
        <form onSubmit={submit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Name <span className="text-red-500">*</span></label>
            <input required value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
              placeholder="e.g. User Profile API"
              className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">URL <span className="text-red-500">*</span></label>
            <input required value={form.url} onChange={e => setForm(f => ({ ...f, url: e.target.value }))}
              placeholder="https://api.example.com/users"
              className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary" />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium mb-1">Method</label>
              <select value={form.method} onChange={e => setForm(f => ({ ...f, method: e.target.value }))}
                className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary">
                {['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].map(m => <option key={m}>{m}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Auth Type</label>
              <select value={form.auth_type} onChange={e => setForm(f => ({ ...f, auth_type: e.target.value }))}
                className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary">
                <option value="none">None</option>
                <option value="bearer">Bearer Token</option>
                <option value="basic">Basic Auth</option>
                <option value="api_key">API Key</option>
              </select>
            </div>
          </div>

          {form.auth_type === 'bearer' && (
            <div>
              <label className="block text-sm font-medium mb-1">Bearer Token</label>
              <input value={form.token} onChange={e => setForm(f => ({ ...f, token: e.target.value }))}
                placeholder="eyJhbGci..."
                className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary font-mono" />
            </div>
          )}
          {form.auth_type === 'basic' && (
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm font-medium mb-1">Username</label>
                <input value={form.username} onChange={e => setForm(f => ({ ...f, username: e.target.value }))}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary" />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Password</label>
                <input type="password" value={form.password} onChange={e => setForm(f => ({ ...f, password: e.target.value }))}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary" />
              </div>
            </div>
          )}
          {form.auth_type === 'api_key' && (
            <div>
              <label className="block text-sm font-medium mb-1">API Key</label>
              <input value={form.api_key} onChange={e => setForm(f => ({ ...f, api_key: e.target.value }))}
                placeholder="sk-..."
                className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary font-mono" />
            </div>
          )}

          {err && <p className="text-sm text-red-600">{err}</p>}
          <div className="flex justify-end gap-3 pt-2">
            <button type="button" onClick={onClose} className="px-4 py-2 border border-border rounded-lg text-sm hover:bg-muted">Cancel</button>
            <button type="submit" disabled={loading}
              className="px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 disabled:opacity-50">
              {loading ? 'Adding…' : 'Add Endpoint'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

// ── Findings panel ────────────────────────────────────────────────────────

function FindingsPanel({ findings }: { findings: any[] }) {
  if (!findings || findings.length === 0) {
    return <p className="text-xs text-muted-foreground italic">No PII findings detected.</p>
  }
  return (
    <div className="space-y-2">
      <p className="text-xs font-semibold text-muted-foreground mb-2">PII FINDINGS ({findings.length})</p>
      {findings.map((f, i) => (
        <div key={i} className="flex items-center justify-between bg-background border border-border rounded-lg px-3 py-2 text-xs">
          <div className="flex items-center gap-2">
            <span className="font-semibold text-orange-700 bg-orange-50 px-1.5 py-0.5 rounded">{f.entity_type}</span>
            <span className="font-mono text-muted-foreground">{f.field}</span>
          </div>
          <div className="flex items-center gap-3">
            <span className="font-mono text-xs">{f.value_masked}</span>
            <span className="text-muted-foreground">{Math.round((f.confidence || 0) * 100)}%</span>
          </div>
        </div>
      ))}
    </div>
  )
}

// ── Endpoint row ──────────────────────────────────────────────────────────

function EndpointRow({ ep, onScan, onDelete, scanning }: { ep: any; onScan: (id: string) => void; onDelete: (id: string) => void; scanning: boolean }) {
  const [expanded, setExpanded] = useState(false)
  const findings = Array.isArray(ep.findings) ? ep.findings : (() => { try { return JSON.parse(ep.findings || '[]') } catch { return [] } })()

  return (
    <div className="border border-border rounded-xl bg-card overflow-hidden">
      <div className="flex items-center justify-between p-4 cursor-pointer hover:bg-muted/30" onClick={() => setExpanded(!expanded)}>
        <div className="flex items-center gap-3 min-w-0">
          <Globe className="w-5 h-5 text-muted-foreground shrink-0" />
          <div className="min-w-0">
            <p className="font-medium text-sm truncate">{ep.name}</p>
            <p className="text-xs text-muted-foreground font-mono truncate">{ep.method} {ep.url}</p>
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0 ml-4">
          <StatusBadge status={ep.status} />
          <RiskBadge level={ep.risk_level || 'unknown'} />
          <span className="text-xs text-muted-foreground hidden md:block">
            {ep.last_scan ? new Date(ep.last_scan).toLocaleDateString() : 'Never scanned'}
          </span>
          <button
            onClick={e => { e.stopPropagation(); onScan(ep.id) }}
            disabled={scanning}
            className="flex items-center gap-1 px-3 py-1.5 bg-primary text-primary-foreground rounded-lg text-xs font-medium hover:bg-primary/90 disabled:opacity-50"
          >
            <Scan className="w-3 h-3" /> {scanning ? 'Scanning…' : 'Scan'}
          </button>
          <button
            onClick={e => { e.stopPropagation(); onDelete(ep.id) }}
            className="p-1.5 text-muted-foreground hover:text-red-600 rounded-lg hover:bg-red-50"
          >
            <Trash2 className="w-4 h-4" />
          </button>
          {expanded ? <ChevronUp className="w-4 h-4 text-muted-foreground" /> : <ChevronDown className="w-4 h-4 text-muted-foreground" />}
        </div>
      </div>
      {expanded && (
        <div className="border-t border-border p-4 bg-muted/10">
          <div className="grid grid-cols-3 gap-4 mb-4 text-sm">
            <div>
              <p className="text-xs text-muted-foreground">Auth Type</p>
              <p className="font-medium capitalize">{ep.auth_type || 'none'}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Last Scan</p>
              <p className="font-medium">{ep.last_scan ? new Date(ep.last_scan).toLocaleString() : '—'}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Findings</p>
              <p className="font-medium">{findings.length}</p>
            </div>
          </div>
          <FindingsPanel findings={findings} />
        </div>
      )}
    </div>
  )
}

// ── Device agents tab ─────────────────────────────────────────────────────

function DeviceAgentsTab() {
  const [endpoints, setEndpoints] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [showSnippet, setShowSnippet] = useState(false)
  const [copied, setCopied] = useState(false)

  const fetch_ = async () => {
    setLoading(true)
    try {
      const res = await fetch(`${API_BASE}/endpoints/agents`, { headers: authHeaders() })
      const data = await res.json()
      setEndpoints(data.endpoints || [])
    } catch { } finally { setLoading(false) }
  }

  useEffect(() => { fetch_() }, [])

  const snippet = `curl -X POST ${API_BASE}/endpoints/agents/register \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"hostname":"my-server","ip":"10.0.0.1","os":"Ubuntu 22.04","agent_version":"1.0.0"}'`

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">Register device agents that report local file-system scans.</p>
        <div className="flex gap-2">
          <button onClick={() => setShowSnippet(!showSnippet)}
            className="flex items-center gap-2 px-3 py-1.5 border border-border rounded-lg text-xs hover:bg-muted">
            <Monitor className="w-3.5 h-3.5" /> Register Snippet
          </button>
          <button onClick={fetch_} className="flex items-center gap-2 px-3 py-1.5 border border-border rounded-lg text-xs hover:bg-muted">
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} /> Refresh
          </button>
        </div>
      </div>

      {showSnippet && (
        <div className="relative bg-gray-900 rounded-xl p-4">
          <button onClick={() => { navigator.clipboard.writeText(snippet); setCopied(true); setTimeout(() => setCopied(false), 2000) }}
            className="absolute top-3 right-3 flex items-center gap-1 text-gray-400 hover:text-white text-xs px-2 py-1 rounded bg-gray-800">
            {copied ? <><CheckCircle className="w-3 h-3 text-green-400" /> Copied</> : <><Copy className="w-3 h-3" /> Copy</>}
          </button>
          <pre className="text-xs text-green-400 font-mono overflow-x-auto whitespace-pre-wrap">{snippet}</pre>
        </div>
      )}

      {loading ? (
        <div className="space-y-2">{[1, 2].map(i => <div key={i} className="h-14 bg-muted animate-pulse rounded-lg" />)}</div>
      ) : endpoints.length === 0 ? (
        <div className="text-center py-10 border border-dashed border-border rounded-xl">
          <Monitor className="w-8 h-8 text-muted-foreground mx-auto mb-2" />
          <p className="text-sm text-muted-foreground">No device agents registered yet</p>
        </div>
      ) : (
        <div className="space-y-2">
          {endpoints.map(ep => (
            <div key={ep.id} className="flex items-center justify-between border border-border rounded-lg bg-card px-4 py-3">
              <div className="flex items-center gap-3">
                <Monitor className="w-4 h-4 text-muted-foreground" />
                <div>
                  <p className="text-sm font-medium">{ep.hostname}</p>
                  <p className="text-xs text-muted-foreground">{ep.ip} · {ep.os}</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${ep.status === 'active' ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'}`}>{ep.status}</span>
                <span className="text-xs text-muted-foreground">{ep.last_scan_at ? new Date(ep.last_scan_at).toLocaleString() : 'Never'}</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// ── Main page ─────────────────────────────────────────────────────────────

export default function EndpointsPage() {
  const [tab, setTab] = useState<'api' | 'devices'>('api')
  const [endpoints, setEndpoints] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [showAdd, setShowAdd] = useState(false)
  const [scanningId, setScanningId] = useState<string | null>(null)
  const [notification, setNotification] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  const notify = (msg: string, type: 'success' | 'error' = 'success') => {
    setNotification({ msg, type })
    setTimeout(() => setNotification(null), 3500)
  }

  const fetchEndpoints = async () => {
    setLoading(true)
    try {
      const res = await fetch(`${API_BASE}/endpoints`, { headers: authHeaders() })
      const data = await res.json()
      setEndpoints(data.endpoints || [])
    } catch { notify('Failed to load endpoints', 'error') }
    finally { setLoading(false) }
  }

  useEffect(() => { if (tab === 'api') fetchEndpoints() }, [tab])

  const handleScan = async (id: string) => {
    setScanningId(id)
    try {
      const res = await fetch(`${API_BASE}/endpoints/${id}/scan`, { method: 'POST', headers: authHeaders() })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const updated = await res.json()
      setEndpoints(prev => prev.map(ep => ep.id === id ? updated : ep))
      notify('Scan complete')
    } catch (e: any) {
      notify(e.message || 'Scan failed', 'error')
    } finally {
      setScanningId(null)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this endpoint?')) return
    try {
      const res = await fetch(`${API_BASE}/endpoints/${id}`, { method: 'DELETE', headers: authHeaders() })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      setEndpoints(prev => prev.filter(ep => ep.id !== id))
      notify('Endpoint deleted')
    } catch (e: any) {
      notify(e.message || 'Delete failed', 'error')
    }
  }

  // Stats
  const critical = endpoints.filter(e => e.risk_level === 'critical').length
  const high = endpoints.filter(e => e.risk_level === 'high').length
  const scanned = endpoints.filter(e => e.status === 'scanned').length

  return (
    <div className="min-h-screen bg-background">
      {showAdd && <AddEndpointModal onClose={() => setShowAdd(false)} onCreated={fetchEndpoints} />}

      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'System' }, { label: 'Endpoints' }]} />
        <div className="flex items-center justify-between mt-2">
          <div>
            <h1 className="text-2xl font-bold">Endpoint Scanning</h1>
            <p className="text-muted-foreground text-sm mt-0.5">Scan API endpoints and devices for PII/sensitive data exposure</p>
          </div>
          {tab === 'api' && (
            <div className="flex gap-2">
              <button onClick={fetchEndpoints}
                className="flex items-center gap-2 px-3 py-2 border border-border rounded-lg text-sm hover:bg-muted">
                <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} /> Refresh
              </button>
              <button onClick={() => setShowAdd(true)}
                className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90">
                <Plus className="w-4 h-4" /> Add Endpoint
              </button>
            </div>
          )}
        </div>

        <div className="flex gap-1 mt-4">
          {[{ id: 'api', label: 'API Endpoint Scanner' }, { id: 'devices', label: 'Device Agents' }].map(t => (
            <button key={t.id} onClick={() => setTab(t.id as any)}
              className={`px-4 py-2 text-sm rounded-md font-medium transition-colors ${tab === t.id ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-muted'}`}>
              {t.label}
            </button>
          ))}
        </div>
      </div>

      {notification && (
        <div className={`mx-8 mt-4 p-3 rounded-lg text-sm flex items-center gap-2 ${notification.type === 'success' ? 'bg-green-50 border border-green-200 text-green-800' : 'bg-red-50 border border-red-200 text-red-800'}`}>
          {notification.type === 'success' ? <CheckCircle className="w-4 h-4" /> : <AlertCircle className="w-4 h-4" />}
          {notification.msg}
        </div>
      )}

      <div className="px-8 py-6">
        {tab === 'api' && (
          <div className="space-y-6">
            {/* Stats */}
            <div className="grid grid-cols-4 gap-4">
              {[
                { label: 'Total Endpoints', value: endpoints.length, color: 'text-foreground', icon: <Globe className="w-5 h-5 text-muted-foreground" /> },
                { label: 'Scanned', value: scanned, color: 'text-green-700', icon: <CheckCircle className="w-5 h-5 text-green-500" /> },
                { label: 'Critical Risk', value: critical, color: 'text-red-700', icon: <Shield className="w-5 h-5 text-red-500" /> },
                { label: 'High Risk', value: high, color: 'text-orange-700', icon: <AlertTriangle className="w-5 h-5 text-orange-500" /> },
              ].map(stat => (
                <div key={stat.label} className="border border-border rounded-xl bg-card p-4">
                  <div className="flex items-center justify-between mb-1">
                    <p className="text-xs text-muted-foreground">{stat.label}</p>
                    {stat.icon}
                  </div>
                  <p className={`text-2xl font-bold ${stat.color}`}>{stat.value}</p>
                </div>
              ))}
            </div>

            {/* Endpoint list */}
            {loading ? (
              <div className="space-y-3">{[1, 2, 3].map(i => <div key={i} className="h-16 bg-muted animate-pulse rounded-xl" />)}</div>
            ) : endpoints.length === 0 ? (
              <div className="text-center py-20 border border-dashed border-border rounded-xl">
                <Globe className="w-10 h-10 text-muted-foreground mx-auto mb-3" />
                <p className="text-sm font-medium">No endpoints registered</p>
                <p className="text-xs text-muted-foreground mt-1">Add your first API endpoint to scan it for PII exposure</p>
                <button onClick={() => setShowAdd(true)}
                  className="mt-4 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90">
                  Add Endpoint
                </button>
              </div>
            ) : (
              <div className="space-y-3">
                {endpoints.map(ep => (
                  <EndpointRow key={ep.id} ep={ep} onScan={handleScan} onDelete={handleDelete}
                    scanning={scanningId === ep.id} />
                ))}
              </div>
            )}
          </div>
        )}

        {tab === 'devices' && <DeviceAgentsTab />}
      </div>
    </div>
  )
}
