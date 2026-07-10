'use client'

import { useState } from 'react'
import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { StatusIndicator } from '@/components/base/status-badge'
import { Plus, RefreshCw, AlertTriangle } from 'lucide-react'
import { useAlerts, useCreateAlertRule } from '@/hooks/use-audit'

interface Alert {
  id: string
  type: string
  severity: string
  title: string
  message: string
  resource: string
  resolved: boolean
  created_at: string
}

const columns: Column<Alert>[] = [
  {
    id: 'severity',
    header: 'Severity',
    cell: (row) => (
      <StatusIndicator
        status={row.severity === 'critical' ? 'error' : row.severity === 'warning' ? 'warning' : 'info'}
        label={row.severity}
      />
    ),
    sortable: true,
  },
  {
    id: 'title',
    header: 'Title',
    cell: (row) => <span className="font-medium text-foreground">{row.title}</span>,
  },
  {
    id: 'message',
    header: 'Message',
    cell: (row) => <span className="text-sm text-muted-foreground">{row.message}</span>,
  },
  {
    id: 'resource',
    header: 'Resource',
    accessorKey: 'resource',
  },
  {
    id: 'created_at',
    header: 'Time',
    cell: (row) => new Date(row.created_at).toLocaleString(),
    sortable: true,
  },
]

export default function AlertsPage() {
  const { data: alerts, isLoading, refetch } = useAlerts()
  const createRule = useCreateAlertRule()
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState({ name: '', condition: '', severity: 'warning' })

  const alertsData = Array.isArray(alerts) ? alerts : []

  const handleCreateRule = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await createRule.mutateAsync(formData)
      setShowForm(false)
      setFormData({ name: '', condition: '', severity: 'warning' })
    } catch (error) {
      // Error handled by hook
    }
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs
            items={[
              { label: 'Observability', href: '/observability' },
              { label: 'Alerts', active: true },
            ]}
          />
          <h1 className="text-3xl font-bold text-foreground mt-4">Alerts</h1>
          <p className="text-sm text-muted-foreground mt-1">View and manage system alerts</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={() => refetch()}
            className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border bg-card text-foreground hover:bg-muted transition-colors"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh
          </button>
          <button
            onClick={() => setShowForm(true)}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
          >
            <Plus className="h-5 w-5" />
            Create Rule
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Create Rule Form */}
        {showForm && (
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="text-lg font-semibold text-foreground mb-4">Create Alert Rule</h3>
            <form onSubmit={handleCreateRule} className="space-y-4">
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Name</label>
                  <input
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="High CPU Alert"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Condition</label>
                  <input
                    type="text"
                    value={formData.condition}
                    onChange={(e) => setFormData({ ...formData, condition: e.target.value })}
                    placeholder="cpu_usage > 90"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Severity</label>
                  <select
                    value={formData.severity}
                    onChange={(e) => setFormData({ ...formData, severity: e.target.value })}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  >
                    <option value="info">Info</option>
                    <option value="warning">Warning</option>
                    <option value="critical">Critical</option>
                  </select>
                </div>
              </div>
              <div className="flex gap-3">
                <button
                  type="submit"
                  disabled={createRule.isPending}
                  className="px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                  {createRule.isPending ? 'Creating...' : 'Create Rule'}
                </button>
                <button
                  type="button"
                  onClick={() => setShowForm(false)}
                  className="px-4 py-2 rounded-lg border border-border text-foreground hover:bg-muted transition-colors"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        )}

        {/* Alerts Table */}
        {isLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : alertsData.length > 0 ? (
          <DataTable columns={columns} data={alertsData} />
        ) : (
          <div className="text-center py-12">
            <AlertTriangle className="h-12 w-12 mx-auto text-green-500 mb-4" />
            <p className="text-foreground font-medium">No alerts</p>
            <p className="text-sm text-muted-foreground mt-1">
              Everything is running smoothly
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
