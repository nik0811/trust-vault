'use client'

import { useState } from 'react'
import { Monitor, RefreshCw, Plus, Scan, ChevronDown, ChevronUp, Copy, CheckCircle, Clock, AlertCircle } from 'lucide-react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || ''

function useEndpoints() {
  const [endpoints, setEndpoints] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchEndpoints = async () => {
    setLoading(true)
    setError(null)
    try {
      const token = localStorage.getItem('token')
      const res = await fetch(`${API_BASE}/api/v1/endpoints`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      const data = await res.json()
      setEndpoints(data.endpoints || [])
    } catch {
      setError('Failed to load endpoints')
    } finally {
      setLoading(false)
    }
  }

  return { endpoints, loading, error, fetchEndpoints }
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { label: string; color: string; icon: React.ReactNode }> = {
    active: { label: 'Active', color: 'bg-green-100 text-green-800', icon: <CheckCircle className="w-3 h-3" /> },
    scanning: { label: 'Scanning', color: 'bg-blue-100 text-blue-800', icon: <RefreshCw className="w-3 h-3 animate-spin" /> },
    inactive: { label: 'Inactive', color: 'bg-gray-100 text-gray-600', icon: <Clock className="w-3 h-3" /> },
    error: { label: 'Error', color: 'bg-red-100 text-red-700', icon: <AlertCircle className="w-3 h-3" /> },
  }
  const s = map[status] || map.inactive
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${s.color}`}>
      {s.icon} {s.label}
    </span>
  )
}

function RegisterSnippet() {
  const [copied, setCopied] = useState(false)
  const token = typeof window !== 'undefined' ? localStorage.getItem('token') || 'YOUR_TOKEN' : 'YOUR_TOKEN'
  const snippet = `curl -X POST ${API_BASE}/api/v1/endpoints/register \\
  -H "Authorization: Bearer ${token}" \\
  -H "Content-Type: application/json" \\
  -d '{"hostname":"my-laptop","ip":"192.168.1.10","os":"macOS 14","agent_version":"1.0.0"}'`

  const copy = () => {
    navigator.clipboard.writeText(snippet)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="relative bg-gray-900 rounded-lg p-4 mt-3">
      <button
        onClick={copy}
        className="absolute top-3 right-3 text-gray-400 hover:text-white"
        title="Copy"
      >
        {copied ? <CheckCircle className="w-4 h-4 text-green-400" /> : <Copy className="w-4 h-4" />}
      </button>
      <pre className="text-xs text-green-400 font-mono overflow-x-auto whitespace-pre-wrap">{snippet}</pre>
    </div>
  )
}

function EndpointRow({ endpoint, onScan }: { endpoint: any; onScan: (id: string) => void }) {
  const [expanded, setExpanded] = useState(false)
  const results = endpoint.scan_results
    ? (() => { try { return typeof endpoint.scan_results === 'string' ? JSON.parse(endpoint.scan_results) : endpoint.scan_results } catch { return null } })()
    : null

  return (
    <div className="border border-border rounded-lg bg-card overflow-hidden">
      <div className="flex items-center justify-between p-4 cursor-pointer hover:bg-muted/40" onClick={() => setExpanded(!expanded)}>
        <div className="flex items-center gap-3">
          <Monitor className="w-5 h-5 text-muted-foreground" />
          <div>
            <p className="font-medium text-sm">{endpoint.hostname}</p>
            <p className="text-xs text-muted-foreground">{endpoint.ip || 'No IP'} · {endpoint.os || 'Unknown OS'}</p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <StatusBadge status={endpoint.status} />
          <span className="text-xs text-muted-foreground">
            {endpoint.last_scan_at ? `Last scan: ${new Date(endpoint.last_scan_at).toLocaleString()}` : 'Never scanned'}
          </span>
          <button
            onClick={(e) => { e.stopPropagation(); onScan(endpoint.id) }}
            className="flex items-center gap-1 px-3 py-1.5 bg-primary text-primary-foreground rounded-md text-xs font-medium hover:bg-primary/90"
          >
            <Scan className="w-3 h-3" /> Scan
          </button>
          {expanded ? <ChevronUp className="w-4 h-4 text-muted-foreground" /> : <ChevronDown className="w-4 h-4 text-muted-foreground" />}
        </div>
      </div>
      {expanded && (
        <div className="border-t border-border p-4 bg-muted/20">
          <div className="grid grid-cols-2 gap-4 mb-3">
            <div>
              <p className="text-xs text-muted-foreground">Agent Version</p>
              <p className="text-sm font-medium">{endpoint.agent_version || '—'}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Registered</p>
              <p className="text-sm font-medium">{new Date(endpoint.created_at).toLocaleString()}</p>
            </div>
          </div>
          {results ? (
            <div className="bg-background rounded-lg p-3 border border-border">
              <p className="text-xs font-semibold text-muted-foreground mb-2">LAST SCAN RESULTS</p>
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <p className="text-xl font-bold">{results.files_scanned ?? 0}</p>
                  <p className="text-xs text-muted-foreground">Files Scanned</p>
                </div>
                <div>
                  <p className="text-xl font-bold text-orange-600">{Array.isArray(results.pii_found) ? results.pii_found.length : 0}</p>
                  <p className="text-xs text-muted-foreground">PII Findings</p>
                </div>
                <div>
                  <p className="text-xl font-bold">{results.scan_duration_ms ? `${(results.scan_duration_ms / 1000).toFixed(1)}s` : '—'}</p>
                  <p className="text-xs text-muted-foreground">Duration</p>
                </div>
              </div>
              {Array.isArray(results.pii_found) && results.pii_found.length > 0 && (
                <div className="mt-3">
                  <p className="text-xs font-semibold text-muted-foreground mb-1">PII FINDINGS</p>
                  <div className="space-y-1 max-h-40 overflow-y-auto">
                    {results.pii_found.map((pii: any, i: number) => (
                      <div key={i} className="flex items-center justify-between text-xs bg-orange-50 border border-orange-200 rounded px-2 py-1">
                        <span className="font-medium">{pii.entity_type || pii.type || 'PII'}</span>
                        <span className="text-muted-foreground">{pii.file || pii.path || ''}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <p className="text-xs text-muted-foreground italic">No scan results yet. Click Scan to trigger a scan.</p>
          )}
        </div>
      )}
    </div>
  )
}

export default function EndpointsPage() {
  const { endpoints, loading, fetchEndpoints } = useEndpoints()
  const [showRegister, setShowRegister] = useState(false)
  const [notification, setNotification] = useState<string | null>(null)

  const showNote = (msg: string) => {
    setNotification(msg)
    setTimeout(() => setNotification(null), 3000)
  }

  const triggerScan = async (id: string) => {
    const token = localStorage.getItem('token')
    const res = await fetch(`${API_BASE}/api/v1/endpoints/${id}/scan`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    })
    if (res.ok) {
      showNote('Scan triggered successfully')
      fetchEndpoints()
    } else {
      showNote('Failed to trigger scan')
    }
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6">
        <div className="flex items-center justify-between">
          <div>
            <Breadcrumbs items={[{ label: 'System' }, { label: 'Endpoints' }]} />
            <h1 className="text-2xl font-bold mt-1">Endpoint Scanning</h1>
            <p className="text-muted-foreground text-sm mt-1">Register and scan laptops & servers for PII data</p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={fetchEndpoints}
              className="flex items-center gap-2 px-4 py-2 border border-border rounded-lg text-sm hover:bg-muted"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} /> Refresh
            </button>
            <button
              onClick={() => setShowRegister(!showRegister)}
              className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90"
            >
              <Plus className="w-4 h-4" /> Register Endpoint
            </button>
          </div>
        </div>
      </div>

      {notification && (
        <div className="mx-8 mt-4 p-3 bg-green-50 border border-green-200 text-green-800 rounded-lg text-sm flex items-center gap-2">
          <CheckCircle className="w-4 h-4" /> {notification}
        </div>
      )}

      <div className="px-8 py-6 space-y-6">
        {showRegister && (
          <div className="border border-border rounded-xl bg-card p-6">
            <h2 className="text-base font-semibold mb-1">Register an Endpoint Agent</h2>
            <p className="text-sm text-muted-foreground mb-2">
              Run this curl command on the target machine to register it with SecureLens. The agent will check in and can receive scan commands.
            </p>
            <RegisterSnippet />
            <p className="text-xs text-muted-foreground mt-3">
              After registration, click <strong>Refresh</strong> to see the new endpoint appear in the list below.
            </p>
          </div>
        )}

        <div>
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-base font-semibold">Registered Endpoints</h2>
            <span className="text-sm text-muted-foreground">{endpoints.length} endpoint{endpoints.length !== 1 ? 's' : ''}</span>
          </div>

          {loading ? (
            <div className="space-y-3">
              {[1, 2, 3].map(i => (
                <div key={i} className="h-16 bg-muted animate-pulse rounded-lg" />
              ))}
            </div>
          ) : endpoints.length === 0 ? (
            <div className="text-center py-16 border border-dashed border-border rounded-xl">
              <Monitor className="w-10 h-10 text-muted-foreground mx-auto mb-3" />
              <p className="text-sm font-medium">No endpoints registered</p>
              <p className="text-xs text-muted-foreground mt-1">Click "Register Endpoint" to add your first machine</p>
              <button
                onClick={() => { setShowRegister(true); fetchEndpoints() }}
                className="mt-4 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90"
              >
                Get Started
              </button>
            </div>
          ) : (
            <div className="space-y-3">
              {endpoints.map(ep => (
                <EndpointRow key={ep.id} endpoint={ep} onScan={triggerScan} />
              ))}
            </div>
          )}
        </div>

        <div className="border border-border rounded-xl bg-card p-6">
          <h2 className="text-base font-semibold mb-3">How It Works</h2>
          <div className="grid grid-cols-3 gap-4">
            {[
              { step: '1', title: 'Register', desc: 'Run the curl command on any machine to register it as an endpoint agent.' },
              { step: '2', title: 'Scan', desc: 'Click "Scan" to trigger a file system scan that detects PII across all files.' },
              { step: '3', title: 'Review', desc: 'Scan results show files scanned, PII types found, and their file paths.' },
            ].map(item => (
              <div key={item.step} className="flex gap-3">
                <div className="w-7 h-7 rounded-full bg-primary/10 text-primary flex items-center justify-center text-xs font-bold shrink-0">{item.step}</div>
                <div>
                  <p className="text-sm font-medium">{item.title}</p>
                  <p className="text-xs text-muted-foreground mt-0.5">{item.desc}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
