'use client'

import { useState } from 'react'
import Link from 'next/link'
import { DataTable, type Column } from '@/components/base/data-table'
import { StatusIndicator } from '@/components/base/status-badge'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { EmptyState } from '@/components/base/empty-state'
import { Plus, FileText, RefreshCw } from 'lucide-react'
import { useDSARs, useCreateDSAR, type DSAR } from '@/hooks/use-privacy'

const columns: Column<DSAR>[] = [
  {
    id: 'subject_id',
    header: 'Subject ID',
    cell: (row) => (
      <Link href={`/privacy/dsar/${row.id}`} className="text-primary hover:underline font-medium">
        {row.subject_id}
      </Link>
    ),
    sortable: true,
  },
  {
    id: 'type',
    header: 'Type',
    cell: (row) => (
      <span className="px-2 py-0.5 rounded bg-muted text-foreground text-sm capitalize">{row.type}</span>
    ),
    sortable: true,
  },
  {
    id: 'status',
    header: 'Status',
    cell: (row) => (
      <StatusIndicator
        status={row.status === 'completed' ? 'success' : row.status === 'in_progress' ? 'pending' : 'inactive'}
        label={row.status}
      />
    ),
  },
  {
    id: 'deadline',
    header: 'Deadline',
    cell: (row) => {
      const deadline = new Date(row.deadline)
      const isOverdue = deadline < new Date() && row.status !== 'completed'
      return (
        <span className={isOverdue ? 'text-red-600' : 'text-foreground'}>
          {deadline.toLocaleDateString()}
        </span>
      )
    },
    sortable: true,
  },
  {
    id: 'created_at',
    header: 'Created',
    cell: (row) => new Date(row.created_at).toLocaleDateString(),
    sortable: true,
  },
]

export default function DSARPage() {
  const { data: dsars, isLoading, refetch } = useDSARs()
  const createDSAR = useCreateDSAR()
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState({ subject_id: '', type: 'access' as const })

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await createDSAR.mutateAsync(formData)
      setShowForm(false)
      setFormData({ subject_id: '', type: 'access' })
    } catch (error) {
      // Error handled by hook
    }
  }

  const dsarsData = Array.isArray(dsars) ? dsars : []

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs
            items={[
              { label: 'Privacy', href: '/privacy' },
              { label: 'DSAR', active: true },
            ]}
          />
          <h1 className="text-3xl font-bold text-foreground mt-4">Data Subject Access Requests</h1>
          <p className="text-sm text-muted-foreground mt-1">Manage access, deletion, and rectification requests</p>
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
            New DSAR
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Create Form */}
        {showForm && (
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="text-lg font-semibold text-foreground mb-4">Create DSAR</h3>
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Subject ID</label>
                  <input
                    type="text"
                    value={formData.subject_id}
                    onChange={(e) => setFormData({ ...formData, subject_id: e.target.value })}
                    placeholder="user@example.com or user ID"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Request Type</label>
                  <select
                    value={formData.type}
                    onChange={(e) => setFormData({ ...formData, type: e.target.value as any })}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  >
                    <option value="access">Access (Right to Access)</option>
                    <option value="delete">Delete (Right to Erasure)</option>
                    <option value="rectify">Rectify (Right to Rectification)</option>
                  </select>
                </div>
              </div>
              <div className="flex gap-3">
                <button
                  type="submit"
                  disabled={createDSAR.isPending}
                  className="px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                  {createDSAR.isPending ? 'Creating...' : 'Create DSAR'}
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

        {/* DSARs Table */}
        {isLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : dsarsData.length > 0 ? (
          <DataTable columns={columns} data={dsarsData} />
        ) : (
          <EmptyState
            icon={<FileText className="h-12 w-12" />}
            title="No DSARs"
            description="Create a data subject access request to start processing privacy requests."
            action={
              <button
                onClick={() => setShowForm(true)}
                className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
              >
                <Plus className="h-5 w-5" />
                New DSAR
              </button>
            }
          />
        )}
      </div>
    </div>
  )
}
