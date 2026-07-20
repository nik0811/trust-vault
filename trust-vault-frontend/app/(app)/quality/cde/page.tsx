'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Database, Plus, Trash2, Shield, AlertTriangle, Loader2, X, Star } from 'lucide-react'
import { useCDEs, useCreateCDE, useDeleteCDE, type CDE } from '@/hooks/use-quality'
import { useDataSources } from '@/hooks/use-datasources'
import { cn } from '@/lib/utils'

const CRITICALITY_STYLES: Record<string, string> = {
  high: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400 border border-red-200 dark:border-red-800',
  medium: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400 border border-yellow-200 dark:border-yellow-800',
  low: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400 border border-green-200 dark:border-green-800',
}

function AddCDEModal({ onClose }: { onClose: () => void }) {
  const createCDE = useCreateCDE()
  const { data: datasources = [] } = useDataSources()
  const [form, setForm] = useState<{
    datasource_id: string
    column_name: string
    table_name: string
    business_definition: string
    data_owner: string
    criticality: 'high' | 'medium' | 'low'
  }>({
    datasource_id: '',
    column_name: '',
    table_name: '',
    business_definition: '',
    data_owner: '',
    criticality: 'medium',
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    await createCDE.mutateAsync(form)
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4">
      <div className="bg-card rounded-xl border border-border w-full max-w-lg">
        <div className="flex items-center justify-between border-b border-border p-5">
          <h2 className="text-lg font-bold">Designate Critical Data Element</h2>
          <button onClick={onClose}><X className="h-5 w-5 text-muted-foreground" /></button>
        </div>
        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Data Source</label>
            <select
              className="w-full border border-border rounded-md px-3 py-2 text-sm bg-background focus:outline-none focus:ring-1 focus:ring-primary"
              value={form.datasource_id}
              onChange={e => setForm(f => ({ ...f, datasource_id: e.target.value }))}
            >
              <option value="">Select datasource (optional)</option>
              {datasources.map((ds: any) => (
                <option key={ds.id} value={ds.id}>{ds.name}</option>
              ))}
            </select>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium mb-1">Table Name *</label>
              <input
                className="w-full border border-border rounded-md px-3 py-2 text-sm bg-background focus:outline-none focus:ring-1 focus:ring-primary"
                placeholder="e.g. customers"
                value={form.table_name}
                onChange={e => setForm(f => ({ ...f, table_name: e.target.value }))}
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Column Name *</label>
              <input
                className="w-full border border-border rounded-md px-3 py-2 text-sm bg-background focus:outline-none focus:ring-1 focus:ring-primary"
                placeholder="e.g. email"
                value={form.column_name}
                onChange={e => setForm(f => ({ ...f, column_name: e.target.value }))}
                required
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Business Definition</label>
            <textarea
              className="w-full border border-border rounded-md px-3 py-2 text-sm bg-background resize-none focus:outline-none focus:ring-1 focus:ring-primary"
              rows={3}
              placeholder="Describe what this data element represents..."
              value={form.business_definition}
              onChange={e => setForm(f => ({ ...f, business_definition: e.target.value }))}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium mb-1">Data Owner</label>
              <input
                className="w-full border border-border rounded-md px-3 py-2 text-sm bg-background focus:outline-none focus:ring-1 focus:ring-primary"
                placeholder="e.g. john@company.com"
                value={form.data_owner}
                onChange={e => setForm(f => ({ ...f, data_owner: e.target.value }))}
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Criticality</label>
              <select
                className="w-full border border-border rounded-md px-3 py-2 text-sm bg-background focus:outline-none focus:ring-1 focus:ring-primary"
                value={form.criticality}
                onChange={e => setForm(f => ({ ...f, criticality: e.target.value as 'high' | 'medium' | 'low' }))}
              >
                <option value="low">Low</option>
                <option value="medium">Medium</option>
                <option value="high">High</option>
              </select>
            </div>
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={onClose} className="px-4 py-2 text-sm border border-border rounded-md hover:bg-muted">Cancel</button>
            <button
              type="submit"
              disabled={createCDE.isPending || !form.column_name || !form.table_name}
              className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-md hover:opacity-90 disabled:opacity-50 flex items-center gap-2"
            >
              {createCDE.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
              Add CDE
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default function CDEPage() {
  const { data: cdes = [], isLoading } = useCDEs()
  const deleteCDE = useDeleteCDE()
  const [showAdd, setShowAdd] = useState(false)

  const stats = {
    total: cdes.length,
    high: cdes.filter(c => c.criticality === 'high').length,
    medium: cdes.filter(c => c.criticality === 'medium').length,
    low: cdes.filter(c => c.criticality === 'low').length,
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[
          { label: 'Quality', href: '/quality' },
          { label: 'Critical Elements', active: true },
        ]} />
        <div className="flex items-center justify-between mt-4">
          <div>
            <h1 className="text-3xl font-bold text-foreground">Critical Data Elements</h1>
            <p className="text-sm text-muted-foreground mt-1">Designate and track the most important data elements in your organization</p>
          </div>
          <button
            onClick={() => setShowAdd(true)}
            className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:opacity-90"
          >
            <Plus className="h-4 w-4" />
            Add CDE
          </button>
        </div>
      </div>

      <div className="p-8 space-y-6">
        {/* Stats */}
        <div className="grid grid-cols-4 gap-4">
          {[
            { label: 'Total CDEs', value: stats.total, color: 'text-foreground', icon: Database },
            { label: 'High Criticality', value: stats.high, color: 'text-red-500', icon: AlertTriangle },
            { label: 'Medium Criticality', value: stats.medium, color: 'text-yellow-500', icon: Shield },
            { label: 'Low Criticality', value: stats.low, color: 'text-green-500', icon: Star },
          ].map(stat => (
            <div key={stat.label} className="rounded-lg border border-border bg-card p-5">
              <div className="flex items-center justify-between">
                <p className="text-sm text-muted-foreground">{stat.label}</p>
                <stat.icon className={cn('h-5 w-5', stat.color)} />
              </div>
              <p className="text-3xl font-bold mt-2">{stat.value}</p>
            </div>
          ))}
        </div>

        {/* CDE List */}
        <div className="rounded-lg border border-border bg-card">
          <div className="p-5 border-b border-border">
            <h2 className="font-semibold">All Critical Data Elements</h2>
          </div>
          {isLoading ? (
            <div className="flex items-center justify-center py-12 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin mr-2" /> Loading...
            </div>
          ) : cdes.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
              <Database className="h-12 w-12 mb-3 opacity-30" />
              <p>No CDEs yet. Designate your first critical data element.</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted/50">
                  <tr>
                    {['Column', 'Table', 'Data Source', 'Criticality', 'Data Owner', 'Business Definition', ''].map(h => (
                      <th key={h} className="text-left px-4 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wide">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {cdes.map(cde => (
                    <tr
                      key={cde.id}
                      className={cn(
                        'hover:bg-muted/30 transition-colors',
                        cde.criticality === 'high' && 'bg-red-50/30 dark:bg-red-950/10'
                      )}
                    >
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          {cde.criticality === 'high' && <AlertTriangle className="h-4 w-4 text-red-500 shrink-0" />}
                          <span className="font-mono font-medium">{cde.column_name}</span>
                        </div>
                      </td>
                      <td className="px-4 py-3 font-mono text-muted-foreground">{cde.table_name}</td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {(cde as any).datasource_name ?? cde.datasource_id ? (
                          <span className="text-xs bg-muted px-2 py-0.5 rounded">
                            {(cde as any).datasource_name ?? cde.datasource_id}
                          </span>
                        ) : '—'}
                      </td>
                      <td className="px-4 py-3">
                        <span className={cn('text-xs px-2 py-0.5 rounded-full font-medium', CRITICALITY_STYLES[cde.criticality] ?? CRITICALITY_STYLES.medium)}>
                          {cde.criticality}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground text-xs">{cde.data_owner || '—'}</td>
                      <td className="px-4 py-3 text-muted-foreground text-xs max-w-xs truncate">
                        {cde.business_definition || '—'}
                      </td>
                      <td className="px-4 py-3">
                        <button
                          onClick={() => deleteCDE.mutate(cde.id)}
                          className="text-muted-foreground hover:text-red-500 transition-colors"
                          title="Remove CDE"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>

      {showAdd && <AddCDEModal onClose={() => setShowAdd(false)} />}
    </div>
  )
}
