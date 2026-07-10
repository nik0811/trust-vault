'use client'

import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { StatusIndicator } from '@/components/base/status-badge'
import { Plus, Loader2, Calendar } from 'lucide-react'
import { useJobs, type Job } from '@/hooks/use-jobs'
import { toast } from 'sonner'
import { useEffect } from 'react'

const columns: Column<Job>[] = [
  { id: 'name', header: 'Job Name', accessorKey: 'name' },
  { id: 'type', header: 'Type', accessorKey: 'type' },
  { id: 'schedule', header: 'Schedule', accessorKey: 'schedule' },
  {
    id: 'status',
    header: 'Status',
    cell: (row) => (
      <StatusIndicator 
        status={row.status === 'completed' ? 'success' : row.status === 'running' ? 'pending' : row.status === 'failed' ? 'error' : 'info'} 
        label={row.status} 
      />
    ),
  },
  { id: 'last_run', header: 'Last Run', cell: (row) => row.last_run ? new Date(row.last_run).toLocaleString() : 'Never' },
]

export default function JobsPage() {
  const { data: jobs, isLoading, error } = useJobs()

  useEffect(() => {
    if (error) toast.error('Failed to load jobs')
  }, [error])

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs items={[{ label: 'Jobs', active: true }]} />
          <h1 className="text-3xl font-bold text-foreground mt-4">Scheduled Jobs</h1>
          <p className="text-sm text-muted-foreground mt-1">Manage and monitor scheduled tasks</p>
        </div>
        <button className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90">
          <Plus className="h-5 w-5" />
          New Job
        </button>
      </div>

      <div className="p-8">
        <div className="rounded-lg border border-border bg-card p-6">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : jobs?.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              <Calendar className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>No scheduled jobs yet</p>
            </div>
          ) : (
            <DataTable columns={columns} data={jobs || []} />
          )}
        </div>
      </div>
    </div>
  )
}
