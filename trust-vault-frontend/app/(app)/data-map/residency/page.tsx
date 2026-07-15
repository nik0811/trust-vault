'use client'

import { useState, useEffect } from 'react'
import { Globe, Plus, Trash2, AlertTriangle, CheckCircle, RefreshCw, MapPin, ShieldCheck } from 'lucide-react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || ''

function authHeaders() {
  const token = typeof window !== 'undefined' ? localStorage.getItem('token') : ''
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

const REGIONS = [
  { id: 'EU', label: 'European Union', flag: '🇪🇺', countries: ['DE', 'FR', 'NL', 'SE', 'IT'] },
  { id: 'UK', label: 'United Kingdom', flag: '🇬🇧', countries: ['GB'] },
  { id: 'US-EAST', label: 'US East', flag: '🇺🇸', countries: ['USA'] },
  { id: 'US-WEST', label: 'US West', flag: '🇺🇸', countries: ['USA'] },
  { id: 'APAC', label: 'Asia Pacific', flag: '🌏', countries: ['JP', 'AU', 'SG', 'IN'] },
  { id: 'CA', label: 'Canada', flag: '🇨🇦', countries: ['CAN'] },
  { id: 'LATAM', label: 'Latin America', flag: '🌎', countries: ['BR', 'MX', 'AR'] },
  { id: 'MEA', label: 'Middle East & Africa', flag: '🌍', countries: ['AE', 'ZA', 'SA'] },
]

export default function DataResidencyPage() {
  const [rules, setRules] = useState<any[]>([])
  const [violations, setViolations] = useState<any[]>([])
  const [datasources, setDatasources] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [activeTab, setActiveTab] = useState<'map' | 'rules' | 'violations'>('map')
  const [showCreateRule, setShowCreateRule] = useState(false)
  const [notification, setNotification] = useState<{ type: 'success' | 'error'; msg: string } | null>(null)
  const [newRule, setNewRule] = useState({ name: '', regulation: 'GDPR', allowed_regions: [] as string[], data_types: [] as string[] })
  const [tagTarget, setTagTarget] = useState<{ id: string; name: string } | null>(null)
  const [tagRegion, setTagRegion] = useState('')
  const [tagCountry, setTagCountry] = useState('')

  const showNote = (type: 'success' | 'error', msg: string) => {
    setNotification({ type, msg })
    setTimeout(() => setNotification(null), 3000)
  }

  const loadAll = async () => {
    setLoading(true)
    try {
      const [rulesRes, vioRes, dsRes] = await Promise.all([
        fetch(`${API_BASE}/residency/rules`, { headers: authHeaders() }),
        fetch(`${API_BASE}/residency/violations`, { headers: authHeaders() }),
        fetch(`${API_BASE}/datasources`, { headers: authHeaders() }),
      ])
      if (!rulesRes.ok) throw new Error(`HTTP ${rulesRes.status}`)
      if (!vioRes.ok) throw new Error(`HTTP ${vioRes.status}`)
      if (!dsRes.ok) throw new Error(`HTTP ${dsRes.status}`)
      const rulesData = await rulesRes.json()
      const vioData = await vioRes.json()
      const dsData = await dsRes.json()
      setRules(rulesData.rules || [])
      setViolations(vioData.violations || [])
      setDatasources(Array.isArray(dsData) ? dsData : dsData.datasources || [])
    } catch (err) {
      console.error('Failed to load residency data:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { loadAll() }, [])

  const createRule = async () => {
    const res = await fetch(`${API_BASE}/residency/rules`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify(newRule),
    })
    if (res.ok) {
      showNote('success', 'Residency rule created')
      setShowCreateRule(false)
      setNewRule({ name: '', regulation: 'GDPR', allowed_regions: [], data_types: [] })
      loadAll()
    } else {
      showNote('error', 'Failed to create rule')
    }
  }

  const deleteRule = async (id: string) => {
    const res = await fetch(`${API_BASE}/residency/rules/${id}`, {
      method: 'DELETE',
      headers: authHeaders(),
    })
    if (res.ok) {
      showNote('success', 'Rule deleted')
      loadAll()
    }
  }

  const tagRegionFn = async () => {
    if (!tagTarget) return
    const res = await fetch(`${API_BASE}/residency/datasources/${tagTarget.id}/tag-region`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({ region: tagRegion, country: tagCountry }),
    })
    if (res.ok) {
      showNote('success', `Tagged "${tagTarget.name}" with region ${tagRegion}`)
      setTagTarget(null)
      loadAll()
    } else {
      showNote('error', 'Failed to tag region')
    }
  }

  const toggleRegion = (rid: string) => {
    setNewRule(r => ({
      ...r,
      allowed_regions: r.allowed_regions.includes(rid)
        ? r.allowed_regions.filter(x => x !== rid)
        : [...r.allowed_regions, rid],
    }))
  }

  const regionStats = REGIONS.map(region => {
    const regionDatasources = datasources.filter((ds: any) => ds.region === region.id)
    const count = regionDatasources.length
    const violationCount = violations.filter((v: any) => v.region === region.id).length
    return { ...region, count, violationCount, compliant: violationCount === 0, datasources: regionDatasources }
  })

  const untaggedCount = datasources.filter((ds: any) => !ds.region).length

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6">
        <div className="flex items-center justify-between">
          <div>
            <Breadcrumbs items={[{ label: 'Discover' }, { label: 'Data Residency' }]} />
            <h1 className="text-2xl font-bold mt-1">Data Residency Management</h1>
            <p className="text-muted-foreground text-sm mt-1">Track where your data lives and enforce geographic compliance rules</p>
          </div>
          <button onClick={loadAll} className="flex items-center gap-2 px-4 py-2 border border-border rounded-lg text-sm hover:bg-muted">
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} /> Refresh
          </button>
        </div>

        <div className="flex gap-1 mt-4">
          {(['map', 'rules', 'violations'] as const).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-4 py-2 text-sm rounded-md font-medium capitalize ${activeTab === tab ? 'bg-primary text-primary-foreground' : 'hover:bg-muted text-muted-foreground'}`}
            >
              {tab === 'violations' ? `Violations (${violations.length})` : tab === 'rules' ? `Rules (${rules.length})` : 'Region Map'}
            </button>
          ))}
        </div>
      </div>

      {notification && (
        <div className={`mx-8 mt-4 p-3 rounded-lg text-sm flex items-center gap-2 ${notification.type === 'success' ? 'bg-green-50 border border-green-200 text-green-800' : 'bg-red-50 border border-red-200 text-red-800'}`}>
          {notification.type === 'success' ? <CheckCircle className="w-4 h-4" /> : <AlertTriangle className="w-4 h-4" />}
          {notification.msg}
        </div>
      )}

      <div className="px-8 py-6">
        {/* Stats bar */}
        <div className="grid grid-cols-4 gap-4 mb-6">
          {[
            { label: 'Total Datasources', value: datasources.length, icon: <Globe className="w-5 h-5 text-blue-500" /> },
            { label: 'Violations', value: violations.length, icon: <AlertTriangle className="w-5 h-5 text-orange-500" />, alert: violations.length > 0 },
            { label: 'Residency Rules', value: rules.length, icon: <ShieldCheck className="w-5 h-5 text-green-500" /> },
            { label: 'Untagged Sources', value: untaggedCount, icon: <MapPin className="w-5 h-5 text-gray-400" />, alert: untaggedCount > 0 },
          ].map(stat => (
            <div key={stat.label} className={`border rounded-xl p-4 bg-card flex items-center gap-3 ${stat.alert ? 'border-orange-200' : 'border-border'}`}>
              <div className="w-10 h-10 bg-muted rounded-lg flex items-center justify-center">{stat.icon}</div>
              <div>
                <p className="text-2xl font-bold">{stat.value}</p>
                <p className="text-xs text-muted-foreground">{stat.label}</p>
              </div>
            </div>
          ))}
        </div>

        {activeTab === 'map' && (
          <div>
            <h2 className="text-base font-semibold mb-4">Geographic Data Distribution</h2>
            <div className="grid grid-cols-4 gap-4">
              {regionStats.map(region => (
                <div
                  key={region.id}
                  className={`border rounded-xl p-4 bg-card hover:shadow-sm transition-shadow ${region.violationCount > 0 ? 'border-orange-200' : region.count > 0 ? 'border-green-200' : 'border-border'}`}
                >
                  <div className="flex items-start justify-between mb-2">
                    <span className="text-2xl">{region.flag}</span>
                    {region.count > 0 && (
                      region.violationCount > 0
                        ? <span className="flex items-center gap-1 text-xs text-orange-600 font-medium"><AlertTriangle className="w-3 h-3" /> At Risk</span>
                        : <span className="flex items-center gap-1 text-xs text-green-600 font-medium"><CheckCircle className="w-3 h-3" /> Compliant</span>
                    )}
                  </div>
                  <p className="font-semibold text-sm">{region.label}</p>
                  <p className="text-xs text-muted-foreground mt-0.5">{region.id}</p>
                  <div className="mt-3 flex items-end justify-between">
                    <div>
                      <p className="text-xl font-bold">{region.count}</p>
                      <p className="text-xs text-muted-foreground">datasources</p>
                    </div>
                    {region.violationCount > 0 && (
                      <span className="text-xs bg-orange-100 text-orange-700 px-2 py-0.5 rounded-full">
                        {region.violationCount} violation{region.violationCount !== 1 ? 's' : ''}
                      </span>
                    )}
                  </div>
                  {region.datasources.length > 0 && (
                    <div className="mt-2 pt-2 border-t border-border/50">
                      {region.datasources.slice(0, 3).map((ds: any) => (
                        <p key={ds.id} className="text-xs text-muted-foreground truncate">{ds.name}</p>
                      ))}
                      {region.datasources.length > 3 && (
                        <p className="text-xs text-primary font-medium mt-0.5">+{region.datasources.length - 3} more</p>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>

            {untaggedCount > 0 && (
              <div className="mt-6">
                <h2 className="text-base font-semibold mb-3 flex items-center gap-2">
                  <MapPin className="w-4 h-4 text-gray-500" /> Untagged Datasources
                  <span className="text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded-full">{untaggedCount}</span>
                </h2>
                <div className="grid grid-cols-2 gap-3">
                  {datasources.filter((ds: any) => !ds.region).map((ds: any) => (
                    <div key={ds.id} className="border border-dashed border-border rounded-lg p-3 flex items-center justify-between bg-card">
                      <div>
                        <p className="text-sm font-medium">{ds.name}</p>
                        <p className="text-xs text-muted-foreground">{ds.type}</p>
                      </div>
                      <button
                        onClick={() => setTagTarget({ id: ds.id, name: ds.name })}
                        className="flex items-center gap-1 px-3 py-1.5 text-xs bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
                      >
                        <MapPin className="w-3 h-3" /> Tag Region
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {activeTab === 'rules' && (
          <div>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-base font-semibold">Residency Rules</h2>
              <button
                onClick={() => setShowCreateRule(!showCreateRule)}
                className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90"
              >
                <Plus className="w-4 h-4" /> New Rule
              </button>
            </div>

            {showCreateRule && (
              <div className="border border-border rounded-xl p-5 bg-card mb-4">
                <h3 className="font-medium mb-4">Create Residency Rule</h3>
                <div className="grid grid-cols-2 gap-4 mb-4">
                  <div>
                    <label className="text-xs text-muted-foreground font-medium">Rule Name *</label>
                    <input
                      className="w-full mt-1 px-3 py-2 border border-border rounded-lg text-sm bg-background"
                      placeholder="e.g. GDPR EU Data Residency"
                      value={newRule.name}
                      onChange={e => setNewRule(r => ({ ...r, name: e.target.value }))}
                    />
                  </div>
                  <div>
                    <label className="text-xs text-muted-foreground font-medium">Regulation</label>
                    <select
                      className="w-full mt-1 px-3 py-2 border border-border rounded-lg text-sm bg-background"
                      value={newRule.regulation}
                      onChange={e => setNewRule(r => ({ ...r, regulation: e.target.value }))}
                    >
                      {['GDPR', 'CCPA', 'HIPAA', 'PIPEDA', 'LGPD', 'PDPA', 'Custom'].map(r => (
                        <option key={r} value={r}>{r}</option>
                      ))}
                    </select>
                  </div>
                </div>
                <div className="mb-4">
                  <label className="text-xs text-muted-foreground font-medium">Allowed Regions</label>
                  <div className="grid grid-cols-4 gap-2 mt-2">
                    {REGIONS.map(r => (
                      <button
                        key={r.id}
                        onClick={() => toggleRegion(r.id)}
                        className={`flex items-center gap-2 px-3 py-2 rounded-lg border text-xs font-medium transition-colors ${newRule.allowed_regions.includes(r.id) ? 'bg-primary/10 border-primary text-primary' : 'border-border hover:bg-muted'}`}
                      >
                        <span>{r.flag}</span> {r.id}
                      </button>
                    ))}
                  </div>
                </div>
                <div className="mb-4">
                  <label className="text-xs text-muted-foreground font-medium">Data Types (comma-separated)</label>
                  <input
                    className="w-full mt-1 px-3 py-2 border border-border rounded-lg text-sm bg-background"
                    placeholder="PII, EMAIL, HEALTH_DATA"
                    onChange={e => setNewRule(r => ({ ...r, data_types: e.target.value.split(',').map(s => s.trim()).filter(Boolean) }))}
                  />
                </div>
                <div className="flex justify-end gap-2">
                  <button onClick={() => setShowCreateRule(false)} className="px-4 py-2 border border-border rounded-lg text-sm hover:bg-muted">Cancel</button>
                  <button onClick={createRule} className="px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90">Create Rule</button>
                </div>
              </div>
            )}

            <div className="space-y-3">
              {rules.length === 0 ? (
                <div className="text-center py-12 border border-dashed border-border rounded-xl">
                  <ShieldCheck className="w-8 h-8 text-muted-foreground mx-auto mb-2" />
                  <p className="text-sm text-muted-foreground">No residency rules defined yet</p>
                </div>
              ) : rules.map((rule: any) => (
                <div key={rule.id} className="border border-border rounded-xl p-4 bg-card flex items-start justify-between">
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="font-medium text-sm">{rule.name}</p>
                      <span className="px-2 py-0.5 bg-blue-100 text-blue-700 rounded-full text-xs">{rule.regulation}</span>
                      <span className={`px-2 py-0.5 rounded-full text-xs ${rule.active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
                        {rule.active ? 'Active' : 'Inactive'}
                      </span>
                    </div>
                    <div className="mt-2 flex flex-wrap gap-1">
                      {(() => { try { const arr = typeof rule.allowed_regions === 'string' ? JSON.parse(rule.allowed_regions) : rule.allowed_regions; return Array.isArray(arr) ? arr.map((ar: string) => <span key={ar} className="px-2 py-0.5 bg-muted text-muted-foreground rounded text-xs">{ar}</span>) : null } catch { return null } })()}
                    </div>
                  </div>
                  <button onClick={() => deleteRule(rule.id)} className="text-red-500 hover:text-red-700 p-1">
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        {activeTab === 'violations' && (
          <div>
            <h2 className="text-base font-semibold mb-4 flex items-center gap-2">
              <AlertTriangle className="w-5 h-5 text-orange-500" /> Residency Violations
            </h2>
            {violations.length === 0 ? (
              <div className="text-center py-12 border border-dashed border-green-200 rounded-xl bg-green-50">
                <CheckCircle className="w-8 h-8 text-green-500 mx-auto mb-2" />
                <p className="text-sm font-medium text-green-800">No violations detected</p>
                <p className="text-xs text-green-600 mt-1">All datasources comply with residency rules</p>
              </div>
            ) : (
              <div className="space-y-3">
                {violations.map((v: any, i: number) => (
                  <div key={i} className="border border-orange-200 rounded-xl p-4 bg-orange-50">
                    <div className="flex items-start justify-between">
                      <div>
                        <div className="flex items-center gap-2 mb-1">
                          <AlertTriangle className="w-4 h-4 text-orange-500" />
                          <p className="font-medium text-sm">{v.datasource_name}</p>
                          <span className="px-2 py-0.5 bg-orange-100 text-orange-700 rounded text-xs">{v.region || 'untagged'}</span>
                        </div>
                        <p className="text-xs text-orange-700">{v.reason}</p>
                        {v.rule_name && (
                          <p className="text-xs text-muted-foreground mt-1">Rule: {v.rule_name} · {v.regulation}</p>
                        )}
                      </div>
                      <button
                        onClick={() => setTagTarget({ id: v.datasource_id, name: v.datasource_name })}
                        className="text-xs px-3 py-1.5 bg-white border border-orange-300 text-orange-700 rounded-md hover:bg-orange-50"
                      >
                        Tag Region
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Tag Region Modal */}
      {tagTarget && (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center">
          <div className="bg-background border border-border rounded-xl p-6 w-[440px] shadow-xl">
            <h3 className="font-semibold mb-1">Tag Region for "{tagTarget.name}"</h3>
            <p className="text-sm text-muted-foreground mb-4">Select the geographic region where this datasource resides.</p>
            <div className="mb-4">
              <label className="text-xs text-muted-foreground font-medium">Region *</label>
              <div className="grid grid-cols-2 gap-2 mt-2">
                {REGIONS.map(r => (
                  <button
                    key={r.id}
                    onClick={() => setTagRegion(r.id)}
                    className={`flex items-center gap-2 px-3 py-2 rounded-lg border text-sm transition-colors ${tagRegion === r.id ? 'bg-primary/10 border-primary text-primary' : 'border-border hover:bg-muted'}`}
                  >
                    <span>{r.flag}</span> {r.id}
                  </button>
                ))}
              </div>
            </div>
            <div className="mb-4">
              <label className="text-xs text-muted-foreground font-medium">Country (optional)</label>
              <input
                className="w-full mt-1 px-3 py-2 border border-border rounded-lg text-sm bg-background"
                placeholder="e.g. Germany"
                value={tagCountry}
                onChange={e => setTagCountry(e.target.value)}
              />
            </div>
            <div className="flex justify-end gap-2">
              <button onClick={() => setTagTarget(null)} className="px-4 py-2 border border-border rounded-lg text-sm hover:bg-muted">Cancel</button>
              <button onClick={tagRegionFn} disabled={!tagRegion} className="px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 disabled:opacity-50">
                Tag Region
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
