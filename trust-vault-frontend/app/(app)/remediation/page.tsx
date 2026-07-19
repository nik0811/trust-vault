'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { DataTable, type Column } from '@/components/base/data-table'
import {
  Wrench,
  X,
  Play,
  FileText,
  Loader2,
  CheckCircle2,
  Clock,
  AlertCircle,
  XCircle,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import {
  useRemediationActions,
  useRemediationLogs,
  useExecuteRemediationAction,
  type RemediationAction,
  type RemediationLog,
} from '@/hooks/use-remediation'
import { cn } from '@/lib/utils'

// ─── helpers ─────────────────────────────────────────────────────────────────

const ACTION_TYPE_LABELS: Record<string, string> = {
  quarantine: 'Quarantine',
  flag: 'Flag',
  delete: 'Delete',
  encrypt: 'Encrypt',
  redact: 'Redact',
  archive: 'Archive',
  deduplicate: 'Deduplicate',
  label: 'Label',
}

const ACTION_TYPE_COLORS: Record<string, string> = {
  quarantine: 'bg-orange-500/10 text-orange-600',
  flag: 'bg-yellow-500/10 text-yellow-600',
  delete: 'bg-red-500/10 text-red-600',
  encrypt: 'bg-blue-500/10 text-blue-600',
  redact: 'bg-purple-500/10 text-purple-600',
  archive: 'bg-gray-500/10 text-gray-600',
  deduplicate: 'bg-teal-500/10 text-teal-600',
  label: 'bg-indigo-500/10 text-indigo-600',
}

const STATUS_CONFIG: Record<string, { label: string; color: string; icon: React.ReactNode }> = {
  pending: {
    label: 'Pending',
    color: 'bg-yellow-500/10 text-yellow-600',
    icon: <Clock className="h-3.5 w-3.5" />,
  },
  running: {
    label: 'Running',
    color: 'bg-blue-500/10 text-blue-600',
    icon: <Loader2 className="h-3.5 w-3.5 animate-spin" />,
  },
  completed: {
    label: 'Completed',
    color: 'bg-green-500/10 text-green-600',
    icon: <CheckCircle2 className="h-3.5 w-3.5" />,
  },
  failed: {
    label: 'Failed',
    color: 'bg-red-500/10 text-red-600',
    icon: <XCircle className="h-3.5 w-3.5" />,
  },
}

function resolvedDatasetName(action: RemediationAction): string {
  if (action.dataset_name && action.dataset_name !== action.dataset_id) {
    return action.dataset_name
  }
  return action.dataset_id || '—'
}

function formatDate(value: string | undefined | null): string {
  if (!value) return '—'
  const d = new Date(value)
  if (isNaN(d.getTime())) return '—'
  return d.toLocaleString()
}

// ─── status badge ─────────────────────────────────────────────────────────────

function StatusBadge({ status }: { status: string }) {
  const cfg = STATUS_CONFIG[status] ?? {
    label: status,
    color: 'bg-gray-500/10 text-gray-600',
    icon: null,
  }
  return (
    <span className={cn('inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-sm font-medium', cfg.color)}>
      {cfg.icon}
      {cfg.label}
    </span>
  )
}

// ─── action-type badge ────────────────────────────────────────────────────────

function ActionTypeBadge({ action }: { action: RemediationAction }) {
  const key = (action.action_type || '').toLowerCase()
  const label = ACTION_TYPE_LABELS[key] || (key ? key.charAt(0).toUpperCase() + key.slice(1) : 'Pending Review')
  const color = ACTION_TYPE_COLORS[key] ?? 'bg-muted text-muted-foreground'
  return (
    <span className={cn('inline-flex px-2 py-0.5 rounded text-sm font-medium', color)}>
      {label}
    </span>
  )
}

// ─── log viewer ──────────────────────────────────────────────────────────────

interface LogViewerProps {
  actionId: string
}

function LogViewer({ actionId }: LogViewerProps) {
  const [open, setOpen] = useState(false)
  const { data: logs, isLoading, refetch } = useRemediationLogs(open ? actionId : '')

  function handleToggle() {
    const next = !open
    setOpen(next)
    if (next) refetch()
  }

  return (
    <div className="rounded-lg border border-border overflow-hidden">
      <button
        onClick={handleToggle}
        className="w-full flex items-center justify-between px-4 py-3 bg-muted/40 hover:bg-muted/60 transition-colors"
      >
        <div className="flex items-center gap-2">
          <FileText className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm font-medium text-foreground">Execution Logs</span>
        </div>
        {open ? (
          <ChevronUp className="h-4 w-4 text-muted-foreground" />
        ) : (
          <ChevronDown className="h-4 w-4 text-muted-foreground" />
        )}
      </button>

      {open && (
        <div className="border-t border-border bg-background p-4 max-h-72 overflow-y-auto font-mono text-xs">
          {isLoading && (
            <div className="text-muted-foreground">Loading logs…</div>
          )}
          {!isLoading && (!logs || logs.length === 0) && (
            <div className="text-muted-foreground">No execution logs yet.</div>
          )}
          {!isLoading && logs && logs.length > 0 && (
            <div className="space-y-1">
              {logs.map((log: RemediationLog) => (
                <div key={log.id} className="flex gap-3 text-muted-foreground">
                  <span className="shrink-0 text-muted-foreground/60">{formatDate(log.created_at)}</span>
                  <span className="font-semibold text-foreground">{log.action}</span>
                  <span className="truncate">{JSON.stringify(log.details)}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

// ─── detail panel ────────────────────────────────────────────────────────────

interface DetailPanelProps {
  action: RemediationAction
  onClose: () => void
  onExecuted: (updated: RemediationAction) => void
}

function DetailPanel({ action, onClose, onExecuted }: DetailPanelProps) {
  const execute = useExecuteRemediationAction()

  function handleExecute() {
    execute.mutate(action.id, {
      onSuccess: (updated) => {
        onExecuted(updated)
      },
    })
  }

  const canExecute = action.status === 'pending' || action.status === 'failed'

  return (
    <div className="fixed inset-0 z-50 flex">
      {/* Backdrop */}
      <div className="flex-1 bg-black/40" onClick={onClose} />

      {/* Slide-over panel */}
      <div className="w-[440px] max-w-full h-full bg-card border-l border-border flex flex-col shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border">
          <h2 className="text-base font-semibold text-foreground">Remediation Details</h2>
          <button onClick={onClose} className="rounded p-1 hover:bg-muted transition-colors">
            <X className="h-4 w-4 text-muted-foreground" />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-5 space-y-5">
          {/* Dataset */}
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Dataset</p>
            <p className="font-medium text-foreground break-all">{resolvedDatasetName(action)}</p>
            {action.dataset_name && action.dataset_name !== action.dataset_id && (
              <p className="text-xs text-muted-foreground font-mono mt-0.5 break-all">{action.dataset_id}</p>
            )}
          </div>

          {/* Action Type */}
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Action Type</p>
            <ActionTypeBadge action={action} />
          </div>

          {/* Status */}
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Status</p>
            <StatusBadge status={action.status} />
          </div>

          {/* Reason */}
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Reason</p>
            <p className="text-sm text-foreground leading-relaxed">{action.reason || '—'}</p>
          </div>

          {/* Created */}
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Created</p>
            <p className="text-sm text-foreground">{formatDate(action.created_at)}</p>
          </div>

          {/* Executed */}
          {action.executed_at && (
            <div>
              <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Executed</p>
              <p className="text-sm text-foreground">{formatDate(action.executed_at)}</p>
            </div>
          )}

          {/* Logs */}
          <LogViewer actionId={action.id} />
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-border flex gap-3">
          <button
            onClick={handleExecute}
            disabled={!canExecute || execute.isPending}
            className="flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-50 transition-colors"
          >
            {execute.isPending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Play className="h-3.5 w-3.5" />
            )}
            Execute
          </button>
          <button
            onClick={onClose}
            className="flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-lg border border-border text-sm font-medium text-foreground hover:bg-muted transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  )
}

// ─── table columns ────────────────────────────────────────────────────────────

function buildColumns(onRowClick: (row: RemediationAction) => void): Column<RemediationAction>[] {
  return [
    {
      id: 'dataset_name',
      header: 'Dataset',
      cell: (row) => (
        <button
          onClick={() => onRowClick(row)}
          className="text-left font-medium text-foreground hover:text-primary transition-colors"
        >
          {resolvedDatasetName(row)}
        </button>
      ),
      sortable: true,
    },
    {
      id: 'action_type',
      header: 'Action Type',
      cell: (row) => <ActionTypeBadge action={row} />,
      sortable: true,
    },
    {
      id: 'status',
      header: 'Status',
      cell: (row) => <StatusBadge status={row.status} />,
      sortable: true,
    },
    {
      id: 'reason',
      header: 'Reason',
      cell: (row) => (
        <span className="text-sm text-muted-foreground truncate max-w-[200px] block">
          {row.reason || '—'}
        </span>
      ),
    },
    {
      id: 'created_at',
      header: 'Created',
      cell: (row) => (
        <span className="text-sm text-muted-foreground">{formatDate(row.created_at)}</span>
      ),
      sortable: true,
    },
  ]
}

// ─── page ─────────────────────────────────────────────────────────────────────

export default function RemediationPage() {
  const { data: actions, isLoading, refetch } = useRemediationActions()
  const [selectedAction, setSelectedAction] = useState<RemediationAction | null>(null)

  const rows = Array.isArray(actions) ? actions : []

  const pendingCount = rows.filter((a) => a.status === 'pending').length
  const runningCount = rows.filter((a) => a.status === 'running').length
  const completedCount = rows.filter((a) => a.status === 'completed').length
  const failedCount = rows.filter((a) => a.status === 'failed').length

  const columns = buildColumns(setSelectedAction)

  function handleExecuted(updated: RemediationAction) {
    if (selectedAction?.id === updated.id) {
      setSelectedAction(updated)
    }
    refetch()
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Remediation', active: true }]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">Data Remediation</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Review and execute remediation actions for sensitive or ROT data
        </p>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Stats row */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="rounded-lg border border-border bg-card p-4">
            <div className="flex items-center gap-2 mb-1">
              <Clock className="h-4 w-4 text-yellow-500" />
              <span className="text-xs text-muted-foreground uppercase tracking-wide">Pending</span>
            </div>
            <p className="text-2xl font-bold text-foreground">{pendingCount}</p>
          </div>
          <div className="rounded-lg border border-border bg-card p-4">
            <div className="flex items-center gap-2 mb-1">
              <Loader2 className="h-4 w-4 text-blue-500" />
              <span className="text-xs text-muted-foreground uppercase tracking-wide">Running</span>
            </div>
            <p className="text-2xl font-bold text-foreground">{runningCount}</p>
          </div>
          <div className="rounded-lg border border-border bg-card p-4">
            <div className="flex items-center gap-2 mb-1">
              <CheckCircle2 className="h-4 w-4 text-green-500" />
              <span className="text-xs text-muted-foreground uppercase tracking-wide">Completed</span>
            </div>
            <p className="text-2xl font-bold text-foreground">{completedCount}</p>
          </div>
          <div className="rounded-lg border border-border bg-card p-4">
            <div className="flex items-center gap-2 mb-1">
              <XCircle className="h-4 w-4 text-red-500" />
              <span className="text-xs text-muted-foreground uppercase tracking-wide">Failed</span>
            </div>
            <p className="text-2xl font-bold text-foreground">{failedCount}</p>
          </div>
        </div>

        {/* Actions table */}
        <div className="rounded-lg border border-border bg-card p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-foreground">Remediation Actions</h2>
            {rows.length > 0 && (
              <span className="text-sm text-muted-foreground">{rows.length} action{rows.length !== 1 ? 's' : ''}</span>
            )}
          </div>

          {isLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : rows.length > 0 ? (
            <>
              {pendingCount > 0 && (
                <div className="mb-4 flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 dark:border-amber-900/40 dark:bg-amber-950/20 px-4 py-3">
                  <AlertCircle className="h-4 w-4 text-amber-500 mt-0.5 shrink-0" />
                  <p className="text-sm text-foreground">
                    <span className="font-medium">{pendingCount} action{pendingCount !== 1 ? 's' : ''}</span> pending review — click a row to open details and execute.
                  </p>
                </div>
              )}
              <DataTable columns={columns} data={rows} />
            </>
          ) : (
            <div className="text-center py-12">
              <Wrench className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <p className="text-foreground font-medium">No remediation actions</p>
              <p className="text-sm text-muted-foreground mt-1">
                Actions are created automatically when sensitive data is detected by policies or ROT scans.
              </p>
            </div>
          )}
        </div>
      </div>

      {/* Detail slide-over */}
      {selectedAction && (
        <DetailPanel
          action={selectedAction}
          onClose={() => setSelectedAction(null)}
          onExecuted={handleExecuted}
        />
      )}
    </div>
  )
}
