'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { CheckCircle, XCircle, Plus, Loader2, BarChart3, Filter } from 'lucide-react'
import {
  useRecordConsent, useWithdrawConsent,
  useConsentRecords, useConsentStats, useRecordConsentV2, useWithdrawConsentV2,
} from '@/hooks/use-privacy'
import { cn } from '@/lib/utils'

const purposes = [
  'Marketing communications',
  'Analytics and performance',
  'Personalization',
  'Third-party sharing',
  'AI/ML training',
  'Research and development',
]

function maskSubjectId(id: string) {
  if (id.includes('@')) {
    const [local, domain] = id.split('@')
    return `${local.slice(0, 2)}***@${domain}`
  }
  if (id.length > 8) return `${id.slice(0, 4)}****${id.slice(-2)}`
  return `${id.slice(0, 2)}****`
}

export default function ConsentPage() {
  const recordConsent = useRecordConsent()
  const withdrawConsent = useWithdrawConsent()
  const recordConsentV2 = useRecordConsentV2()
  const withdrawConsentV2 = useWithdrawConsentV2()

  const [formData, setFormData] = useState({ subject_id: '', purpose: '', source: '', ip: '' })
  const [withdrawId, setWithdrawId] = useState('')
  const [filterPurpose, setFilterPurpose] = useState('')
  const [filterStatus, setFilterStatus] = useState('')

  const { data: stats } = useConsentStats()
  const { data: records = [], isLoading: recordsLoading } = useConsentRecords(filterPurpose, filterStatus)

  const handleRecord = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await recordConsentV2.mutateAsync(formData)
      setFormData({ subject_id: '', purpose: '', source: '', ip: '' })
    } catch {}
  }

  const handleWithdraw = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!withdrawId.trim()) return
    try {
      await withdrawConsentV2.mutateAsync(withdrawId)
      setWithdrawId('')
    } catch {}
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[
          { label: 'Privacy', href: '/privacy' },
          { label: 'Consent', active: true },
        ]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">Consent Management</h1>
        <p className="text-sm text-muted-foreground mt-1">Track and manage user consent for data processing</p>
      </div>

      <div className="p-8 space-y-8">
        {/* Stats Row */}
        {stats && (
          <div className="grid grid-cols-4 gap-4">
            {[
              { label: 'Total Consents', value: stats.total, color: 'text-foreground' },
              { label: 'Granted', value: stats.granted, color: 'text-green-500' },
              { label: 'Withdrawn', value: stats.withdrawn, color: 'text-red-500' },
              {
                label: 'Withdrawal Rate',
                value: `${(stats.withdrawal_rate ?? 0).toFixed(1)}%`,
                color: (stats.withdrawal_rate ?? 0) > 20 ? 'text-red-500' : 'text-yellow-500',
              },
            ].map(stat => (
              <div key={stat.label} className="rounded-lg border border-border bg-card p-5">
                <p className="text-sm text-muted-foreground">{stat.label}</p>
                <p className={cn('text-3xl font-bold mt-2', stat.color)}>{stat.value}</p>
              </div>
            ))}
          </div>
        )}

        {/* Record / Withdraw */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="rounded-lg border border-border bg-card p-6">
            <div className="flex items-center gap-3 mb-4">
              <CheckCircle className="h-6 w-6 text-green-500" />
              <h3 className="text-lg font-semibold text-foreground">Record Consent</h3>
            </div>
            <form onSubmit={handleRecord} className="space-y-3">
              <div>
                <label className="block text-sm font-medium mb-1">Subject ID *</label>
                <input
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  placeholder="user@example.com or user ID"
                  value={formData.subject_id}
                  onChange={e => setFormData(f => ({ ...f, subject_id: e.target.value }))}
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Purpose *</label>
                <select
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  value={formData.purpose}
                  onChange={e => setFormData(f => ({ ...f, purpose: e.target.value }))}
                  required
                >
                  <option value="">Select purpose...</option>
                  {purposes.map(p => <option key={p} value={p}>{p}</option>)}
                </select>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium mb-1">Source</label>
                  <input
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                    placeholder="website, app, form..."
                    value={formData.source}
                    onChange={e => setFormData(f => ({ ...f, source: e.target.value }))}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">IP Address</label>
                  <input
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                    placeholder="1.2.3.4"
                    value={formData.ip}
                    onChange={e => setFormData(f => ({ ...f, ip: e.target.value }))}
                  />
                </div>
              </div>
              <button
                type="submit"
                disabled={recordConsentV2.isPending}
                className="w-full flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-green-600 text-white hover:bg-green-700 transition-colors disabled:opacity-50 text-sm"
              >
                {recordConsentV2.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                Record Consent
              </button>
            </form>
          </div>

          <div className="rounded-lg border border-border bg-card p-6">
            <div className="flex items-center gap-3 mb-4">
              <XCircle className="h-6 w-6 text-red-500" />
              <h3 className="text-lg font-semibold text-foreground">Withdraw All Consent</h3>
            </div>
            <form onSubmit={handleWithdraw} className="space-y-3">
              <div>
                <label className="block text-sm font-medium mb-1">Subject ID *</label>
                <input
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  placeholder="user@example.com or user ID"
                  value={withdrawId}
                  onChange={e => setWithdrawId(e.target.value)}
                  required
                />
              </div>
              <p className="text-sm text-muted-foreground">
                This withdraws all consent records for the specified subject across all purposes.
              </p>
              <button
                type="submit"
                disabled={withdrawConsentV2.isPending}
                className="w-full flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-red-600 text-white hover:bg-red-700 transition-colors disabled:opacity-50 text-sm"
              >
                {withdrawConsentV2.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <XCircle className="h-4 w-4" />}
                Withdraw Consent
              </button>
            </form>
          </div>
        </div>

        {/* Records Table */}
        <div className="rounded-lg border border-border bg-card">
          <div className="p-5 border-b border-border flex items-center justify-between flex-wrap gap-3">
            <h3 className="font-semibold text-foreground">Consent Records</h3>
            <div className="flex items-center gap-3">
              <Filter className="h-4 w-4 text-muted-foreground" />
              <select
                className="border border-border rounded-md px-2 py-1 text-sm bg-background focus:outline-none"
                value={filterPurpose}
                onChange={e => setFilterPurpose(e.target.value)}
              >
                <option value="">All Purposes</option>
                {purposes.map(p => <option key={p} value={p}>{p}</option>)}
              </select>
              <select
                className="border border-border rounded-md px-2 py-1 text-sm bg-background focus:outline-none"
                value={filterStatus}
                onChange={e => setFilterStatus(e.target.value)}
              >
                <option value="">All Statuses</option>
                <option value="granted">Granted</option>
                <option value="withdrawn">Withdrawn</option>
              </select>
            </div>
          </div>
          {recordsLoading ? (
            <div className="flex items-center justify-center py-12 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin mr-2" /> Loading...
            </div>
          ) : records.length === 0 ? (
            <div className="flex items-center justify-center py-12 text-muted-foreground">
              <BarChart3 className="h-5 w-5 mr-2" /> No consent records found
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted/50">
                  <tr>
                    {['Subject ID', 'Purpose', 'Status', 'Source', 'Date', 'Action'].map(h => (
                      <th key={h} className="text-left px-4 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wide">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {records.map(rec => (
                    <tr key={rec.id} className="hover:bg-muted/30 transition-colors">
                      <td className="px-4 py-3 font-mono text-xs">{maskSubjectId(rec.subject_id)}</td>
                      <td className="px-4 py-3">{rec.purpose}</td>
                      <td className="px-4 py-3">
                        <span className={cn(
                          'text-xs px-2 py-0.5 rounded-full font-medium',
                          rec.status === 'granted'
                            ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                            : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                        )}>
                          {rec.status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">{rec.source || '—'}</td>
                      <td className="px-4 py-3 text-muted-foreground text-xs">
                        {new Date(rec.created_at).toLocaleDateString()}
                      </td>
                      <td className="px-4 py-3">
                        {rec.status === 'granted' && (
                          <button
                            onClick={() => withdrawConsentV2.mutate(rec.subject_id)}
                            className="text-xs text-red-500 hover:text-red-600 font-medium"
                          >
                            Withdraw
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Purpose breakdown */}
        {stats?.by_purpose && Object.keys(stats.by_purpose).length > 0 && (
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="font-semibold text-foreground mb-4">Consent by Purpose</h3>
            <div className="space-y-3">
              {Object.entries(stats.by_purpose).map(([purpose, data]: [string, any]) => {
                const rate = data.total > 0 ? Math.round((data.granted / data.total) * 100) : 0
                return (
                  <div key={purpose}>
                    <div className="flex justify-between text-sm mb-1">
                      <span className="font-medium">{purpose}</span>
                      <span className="text-muted-foreground">{data.granted}/{data.total} granted ({rate}%)</span>
                    </div>
                    <div className="h-2 bg-muted rounded-full overflow-hidden">
                      <div className="h-full bg-primary rounded-full" style={{ width: `${rate}%` }} />
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
