'use client'

import { useState } from 'react'
import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { StatusIndicator } from '@/components/base/status-badge'
import { Modal, ConfirmModal } from '@/components/base/modal'
import { Plus, Loader2, Calendar, Play, Pencil, Trash2, History, MoreHorizontal, Clock, CheckCircle, XCircle, AlertCircle } from 'lucide-react'
import { useJobs, useCreateJob, useUpdateJob, useDeleteJob, useRunJobNow, useJobHistory, type Job, type JobExecution } from '@/hooks/use-jobs'
import { toast } from 'sonner'
import { useEffect } from 'react'

const JOB_TYPES = [
  { value: 'scan', label: 'Data Scan' },
  { value: 'quality_assessment', label: 'Quality Assessment' },
  { value: 'rot_detection', label: 'ROT Detection' },
  { value: 'compliance_check', label: 'Compliance Check' },
  { value: 'classification', label: 'Classification' },
  { value: 'privacy_scan', label: 'Privacy Scan' },
]

const SCHEDULE_PRESETS = [
  { value: '@hourly', label: 'Every hour' },
  { value: '@daily', label: 'Daily (midnight)' },
  { value: '@weekly', label: 'Weekly (Sunday midnight)' },
  { value: '0 2 * * *', label: 'Daily at 2 AM' },
  { value: '0 0 * * 1', label: 'Weekly (Monday midnight)' },
  { value: '0 0 1 * *', label: 'Monthly (1st of month)' },
]

function JobActionsMenu({ job, onEdit, onDelete, onRunNow, onViewHistory }: {
  job: Job
  onEdit: () => void
  onDelete: () => void
  onRunNow: () => void
  onViewHistory: () => void
}) {
  const [open, setOpen] = useState(false)

  return (
    <div className="relative">
      <button
        onClick={(e) => { e.stopPropagation(); setOpen(!open) }}
        className="p-1 rounded hover:bg-muted"
      >
        <MoreHorizontal className="h-4 w-4" />
      </button>
      {open && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} />
          <div className="absolute right-0 top-8 z-20 w-48 rounded-md border border-border bg-card shadow-lg py-1">
            <button
              onClick={(e) => { e.stopPropagation(); onRunNow(); setOpen(false) }}
              disabled={job.status === 'running'}
              className="flex items-center gap-2 w-full px-4 py-2 text-sm hover:bg-muted disabled:opacity-50"
            >
              <Play className="h-4 w-4" />
              Run Now
            </button>
            <button
              onClick={(e) => { e.stopPropagation(); onEdit(); setOpen(false) }}
              className="flex items-center gap-2 w-full px-4 py-2 text-sm hover:bg-muted"
            >
              <Pencil className="h-4 w-4" />
              Edit
            </button>
            <button
              onClick={(e) => { e.stopPropagation(); onViewHistory(); setOpen(false) }}
              className="flex items-center gap-2 w-full px-4 py-2 text-sm hover:bg-muted"
            >
              <History className="h-4 w-4" />
              View History
            </button>
            <hr className="my-1 border-border" />
            <button
              onClick={(e) => { e.stopPropagation(); onDelete(); setOpen(false) }}
              className="flex items-center gap-2 w-full px-4 py-2 text-sm text-destructive hover:bg-muted"
            >
              <Trash2 className="h-4 w-4" />
              Delete
            </button>
          </div>
        </>
      )}
    </div>
  )
}

function JobFormModal({ isOpen, onClose, job }: { isOpen: boolean; onClose: () => void; job?: Job }) {
  const createJob = useCreateJob()
  const updateJob = useUpdateJob()
  const [formData, setFormData] = useState({
    name: job?.name || '',
    type: job?.type || 'scan',
    schedule: job?.schedule || '@daily',
    customSchedule: '',
    config: job?.config || {},
  })

  useEffect(() => {
    if (job) {
      const isPreset = SCHEDULE_PRESETS.some(p => p.value === job.schedule)
      setFormData({
        name: job.name,
        type: job.type,
        schedule: isPreset ? job.schedule : 'custom',
        customSchedule: isPreset ? '' : job.schedule,
        config: job.config || {},
      })
    } else {
      setFormData({ name: '', type: 'scan', schedule: '@daily', customSchedule: '', config: {} })
    }
  }, [job, isOpen])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const schedule = formData.schedule === 'custom' ? formData.customSchedule : formData.schedule
    
    if (job) {
      await updateJob.mutateAsync({ id: job.id, data: { name: formData.name, type: formData.type, schedule, config: formData.config } })
    } else {
      await createJob.mutateAsync({ name: formData.name, type: formData.type, schedule, config: formData.config })
    }
    onClose()
  }

  const isLoading = createJob.isPending || updateJob.isPending

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={job ? 'Edit Job' : 'Create New Job'}
      description={job ? 'Update the scheduled job configuration' : 'Configure a new scheduled job'}
      size="lg"
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-foreground mb-1">Job Name</label>
          <input
            type="text"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            placeholder="e.g., Daily PII Scan"
            className="w-full px-3 py-2 rounded-md border border-border bg-background text-foreground"
            required
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-foreground mb-1">Job Type</label>
          <select
            value={formData.type}
            onChange={(e) => setFormData({ ...formData, type: e.target.value })}
            className="w-full px-3 py-2 rounded-md border border-border bg-background text-foreground"
          >
            {JOB_TYPES.map((type) => (
              <option key={type.value} value={type.value}>{type.label}</option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-foreground mb-1">Schedule</label>
          <select
            value={SCHEDULE_PRESETS.some(p => p.value === formData.schedule) ? formData.schedule : 'custom'}
            onChange={(e) => setFormData({ ...formData, schedule: e.target.value })}
            className="w-full px-3 py-2 rounded-md border border-border bg-background text-foreground"
          >
            {SCHEDULE_PRESETS.map((preset) => (
              <option key={preset.value} value={preset.value}>{preset.label}</option>
            ))}
            <option value="custom">Custom (cron expression)</option>
          </select>
        </div>

        {formData.schedule === 'custom' && (
          <div>
            <label className="block text-sm font-medium text-foreground mb-1">Cron Expression</label>
            <input
              type="text"
              value={formData.customSchedule}
              onChange={(e) => setFormData({ ...formData, customSchedule: e.target.value })}
              placeholder="e.g., 0 2 * * * (daily at 2 AM)"
              className="w-full px-3 py-2 rounded-md border border-border bg-background text-foreground font-mono text-sm"
              required
            />
            <p className="mt-1 text-xs text-muted-foreground">Format: minute hour day month weekday</p>
          </div>
        )}

        <div className="flex justify-end gap-3 pt-4">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 rounded-md border border-border bg-card hover:bg-muted text-sm font-medium"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isLoading}
            className="px-4 py-2 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium disabled:opacity-50"
          >
            {isLoading ? 'Saving...' : job ? 'Update Job' : 'Create Job'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

function JobHistoryModal({ isOpen, onClose, job }: { isOpen: boolean; onClose: () => void; job: Job | null }) {
  const { data: history, isLoading } = useJobHistory(job?.id || '')

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed': return <CheckCircle className="h-4 w-4 text-green-500" />
      case 'failed': return <XCircle className="h-4 w-4 text-red-500" />
      case 'running': return <Loader2 className="h-4 w-4 text-blue-500 animate-spin" />
      default: return <Clock className="h-4 w-4 text-muted-foreground" />
    }
  }

  const formatDuration = (ms: number | null) => {
    if (!ms) return '—'
    if (ms < 1000) return `${ms}ms`
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
    return `${(ms / 60000).toFixed(1)}m`
  }

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`Execution History: ${job?.name || ''}`}
      description="View past job executions and their results"
      size="xl"
    >
      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : !history || history.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          <History className="h-10 w-10 mx-auto mb-3 opacity-50" />
          <p>No execution history yet</p>
          <p className="text-sm mt-1">Run the job to see execution records here</p>
        </div>
      ) : (
        <div className="space-y-3 max-h-96 overflow-y-auto">
          {history.map((execution) => (
            <div key={execution.id} className="flex items-start gap-3 p-3 rounded-lg border border-border bg-muted/30">
              {getStatusIcon(execution.status)}
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-medium text-sm capitalize">{execution.status}</span>
                  <span className="text-xs text-muted-foreground">
                    {execution.started_at ? new Date(execution.started_at).toLocaleString() : '—'}
                  </span>
                </div>
                <div className="flex items-center gap-4 mt-1 text-xs text-muted-foreground">
                  <span>Duration: {formatDuration(execution.duration_ms)}</span>
                  {execution.completed_at && (
                    <span>Completed: {new Date(execution.completed_at).toLocaleString()}</span>
                  )}
                </div>
                {execution.error && (
                  <div className="mt-2 p-2 rounded bg-destructive/10 text-destructive text-xs">
                    {execution.error}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </Modal>
  )
}

export default function JobsPage() {
  const { data: jobs, isLoading, error } = useJobs()
  const deleteJob = useDeleteJob()
  const runJobNow = useRunJobNow()

  const [showCreateModal, setShowCreateModal] = useState(false)
  const [editingJob, setEditingJob] = useState<Job | null>(null)
  const [deletingJob, setDeletingJob] = useState<Job | null>(null)
  const [historyJob, setHistoryJob] = useState<Job | null>(null)

  useEffect(() => {
    if (error) toast.error('Failed to load jobs')
  }, [error])

  const handleDelete = async () => {
    if (deletingJob) {
      await deleteJob.mutateAsync(deletingJob.id)
      setDeletingJob(null)
    }
  }

  const handleRunNow = async (job: Job) => {
    await runJobNow.mutateAsync(job.id)
  }

  const columns: Column<Job>[] = [
    { id: 'name', header: 'Job Name', accessorKey: 'name', sortable: true },
    { 
      id: 'type', 
      header: 'Type', 
      cell: (row) => {
        const type = JOB_TYPES.find(t => t.value === row.type)
        return <span className="capitalize">{type?.label || row.type}</span>
      }
    },
    { 
      id: 'schedule', 
      header: 'Schedule', 
      cell: (row) => {
        const preset = SCHEDULE_PRESETS.find(p => p.value === row.schedule)
        return <code className="text-xs bg-muted px-2 py-1 rounded">{preset?.label || row.schedule}</code>
      }
    },
    {
      id: 'status',
      header: 'Status',
      cell: (row) => (
        <StatusIndicator 
          status={row.status === 'completed' ? 'success' : row.status === 'running' ? 'pending' : row.status === 'failed' ? 'error' : 'warning'} 
          label={row.status} 
        />
      ),
    },
    { 
      id: 'last_run', 
      header: 'Last Run', 
      cell: (row) => row.last_run ? new Date(row.last_run).toLocaleString() : '—' 
    },
    { 
      id: 'next_run', 
      header: 'Next Run', 
      cell: (row) => row.next_run ? new Date(row.next_run).toLocaleString() : '—' 
    },
    {
      id: 'actions',
      header: '',
      cell: (row) => (
        <JobActionsMenu
          job={row}
          onEdit={() => setEditingJob(row)}
          onDelete={() => setDeletingJob(row)}
          onRunNow={() => handleRunNow(row)}
          onViewHistory={() => setHistoryJob(row)}
        />
      ),
    },
  ]

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs items={[{ label: 'Jobs', active: true }]} />
          <h1 className="text-3xl font-bold text-foreground mt-4">Scheduled Jobs</h1>
          <p className="text-sm text-muted-foreground mt-1">Manage and monitor scheduled tasks</p>
        </div>
        <button 
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90"
        >
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
              <p className="text-sm mt-1">Create a job to automate scans, quality checks, and more</p>
              <button
                onClick={() => setShowCreateModal(true)}
                className="mt-4 inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90"
              >
                <Plus className="h-4 w-4" />
                Create Your First Job
              </button>
            </div>
          ) : (
            <DataTable columns={columns} data={jobs || []} />
          )}
        </div>
      </div>

      {/* Create/Edit Modal */}
      <JobFormModal
        isOpen={showCreateModal || !!editingJob}
        onClose={() => { setShowCreateModal(false); setEditingJob(null) }}
        job={editingJob || undefined}
      />

      {/* Delete Confirmation */}
      <ConfirmModal
        isOpen={!!deletingJob}
        onConfirm={handleDelete}
        onCancel={() => setDeletingJob(null)}
        title="Delete Job"
        description={`Are you sure you want to delete "${deletingJob?.name}"? This action cannot be undone.`}
        confirmText="Delete"
        isDestructive
        isLoading={deleteJob.isPending}
      />

      {/* History Modal */}
      <JobHistoryModal
        isOpen={!!historyJob}
        onClose={() => setHistoryJob(null)}
        job={historyJob}
      />
    </div>
  )
}
