'use client'

import { useState, useEffect, useRef } from 'react'
import { StatCard } from '@/components/base/stat-card'
import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import Link from 'next/link'
import { Trash2, HardDrive, Copy, Clock, Play, ChevronDown, ChevronUp, Loader2, X, AlertCircle } from 'lucide-react'
import { useROTSummary, useROTDatasets, useTriggerROTScan, useRemediateROT, type ROTDataset } from '@/hooks/use-advisor'
import { api } from '@/lib/api'
import { toast } from 'sonner'

// ─── helpers ────────────────────────────────────────────────────────────────

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return 'N/A'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
}

function formatLastAccess(value: string | null | undefined): string {
  if (!value) return '—'
  const d = new Date(value)
  if (isNaN(d.getTime()) || d.getFullYear() <= 1970) return '—'
  return d.toLocaleDateString()
}

function categoryLabel(category: string): string {
  return category.charAt(0).toUpperCase() + category.slice(1)
}

const categoryColors: Record<string, string> = {
  redundant: 'bg-blue-500/10 text-blue-600',
  obsolete: 'bg-yellow-500/10 text-yellow-600',
  trivial: 'bg-gray-500/10 text-gray-600',
}

// ─── table columns ───────────────────────────────────────────────────────────

function buildColumns(onRowClick: (row: ROTDataset) => void): Column<ROTDataset>[] {
  return [
    {
      id: 'dataset_id',
      header: 'Dataset',
      cell: (row) => (
        <button
          onClick={() => onRowClick(row)}
          className="text-left font-medium text-foreground hover:text-primary transition-colors"
        >
          {row.dataset_name && row.dataset_name !== row.dataset_id
            ? row.dataset_name
            : row.dataset_id}
        </button>
      ),
      sortable: true,
    },
    {
      id: 'category',
      header: 'Category',
      cell: (row) => (
        <span className={`px-2 py-0.5 rounded text-sm ${categoryColors[row.category] ?? 'bg-gray-500/10 text-gray-600'}`}>
          {categoryLabel(row.category)}
        </span>
      ),
      sortable: true,
    },
    {
      id: 'score',
      header: 'ROT Score',
      cell: (row) => (
        <div className="flex items-center gap-2">
          <div className="w-24 h-1.5 rounded-full bg-muted overflow-hidden">
            <div
              className="h-full rounded-full bg-primary"
              style={{ width: `${Math.min(row.score * 100, 100)}%` }}
            />
          </div>
          <span className="text-sm text-foreground tabular-nums">
            {(row.score * 100).toFixed(0)}%
          </span>
        </div>
      ),
      sortable: true,
    },
    {
      id: 'size_bytes',
      header: 'Size',
      cell: (row) => (
        <span className="text-sm text-muted-foreground">{formatBytes(row.size_bytes)}</span>
      ),
      sortable: true,
    },
    {
      id: 'last_access',
      header: 'Last Access',
      cell: (row) => (
        <span className="text-sm text-muted-foreground">{formatLastAccess(row.last_access)}</span>
      ),
      sortable: true,
    },
  ]
}

// ─── detail panel ────────────────────────────────────────────────────────────

interface DetailPanelProps {
  row: ROTDataset
  onClose: () => void
  onDismiss: (id: string) => void
}

function DetailPanel({ row, onClose, onDismiss }: DetailPanelProps) {
  const remediate = useRemediateROT()

  function handleRemediate() {
    remediate.mutate(
      { dataset_ids: [row.dataset_id], action: 'archive' },
      { onSuccess: onClose },
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex">
      {/* backdrop */}
      <div className="flex-1 bg-black/40" onClick={onClose} />

      {/* panel */}
      <div className="w-[420px] max-w-full h-full bg-card border-l border-border flex flex-col shadow-2xl">
        {/* header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border">
          <h2 className="text-base font-semibold text-foreground">ROT Dataset Details</h2>
          <button onClick={onClose} className="rounded p-1 hover:bg-muted transition-colors">
            <X className="h-4 w-4 text-muted-foreground" />
          </button>
        </div>

        {/* body */}
        <div className="flex-1 overflow-y-auto px-6 py-5 space-y-6">
          {/* Dataset */}
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Dataset</p>
            <p className="font-medium text-foreground break-all">
              {row.dataset_name && row.dataset_name !== row.dataset_id
                ? row.dataset_name
                : row.dataset_id}
            </p>
            {row.dataset_name && row.dataset_name !== row.dataset_id && (
              <p className="text-xs text-muted-foreground font-mono mt-0.5 break-all">{row.dataset_id}</p>
            )}
          </div>

          {/* Category */}
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Category</p>
            <span className={`inline-flex px-2.5 py-1 rounded text-sm font-medium ${categoryColors[row.category] ?? 'bg-gray-500/10 text-gray-600'}`}>
              {categoryLabel(row.category)}
            </span>
          </div>

          {/* ROT Score */}
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide mb-2">ROT Score</p>
            <div className="flex items-center gap-3">
              <div className="flex-1 h-2 rounded-full bg-muted overflow-hidden">
                <div
                  className="h-full rounded-full bg-primary transition-all"
                  style={{ width: `${Math.min(row.score * 100, 100)}%` }}
                />
              </div>
              <span className="text-sm font-medium text-foreground tabular-nums w-10 text-right">
                {(row.score * 100).toFixed(0)}%
              </span>
            </div>
          </div>

          {/* Reason */}
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Reason</p>
            <p className="text-sm text-foreground leading-relaxed">{row.reason || '—'}</p>
          </div>

          {/* Size */}
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Size</p>
            <p className="text-sm text-foreground">{formatBytes(row.size_bytes)}</p>
          </div>

          {/* Last Access */}
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Last Access</p>
            <p className="text-sm text-foreground">{formatLastAccess(row.last_access)}</p>
          </div>
        </div>

        {/* actions */}
        <div className="px-6 py-4 border-t border-border flex gap-3">
          <button
            onClick={handleRemediate}
            disabled={remediate.isPending}
            className="flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-50 transition-colors"
          >
            {remediate.isPending && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
            Remediate
          </button>
          <button
            onClick={() => onDismiss(row.id)}
            className="flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-lg border border-border text-sm font-medium text-foreground hover:bg-muted transition-colors"
          >
            Dismiss
          </button>
        </div>
      </div>
    </div>
  )
}

// ─── ROT summary blurb ───────────────────────────────────────────────────────

interface ROTSummaryBlurbProps {
  datasets: ROTDataset[]
}

function ROTSummaryBlurb({ datasets }: ROTSummaryBlurbProps) {
  if (datasets.length === 0) return null

  const redundant = datasets.filter((d) => d.category === 'redundant').length
  const obsolete = datasets.filter((d) => d.category === 'obsolete').length
  const trivial = datasets.filter((d) => d.category === 'trivial').length

  const lines = [
    { count: redundant, label: 'Redundant', description: 'data appears in multiple scan results' },
    { count: obsolete, label: 'Obsolete', description: 'data not accessed in 90+ days' },
    { count: trivial, label: 'Trivial', description: 'low-confidence classifications' },
  ]

  return (
    <div className="rounded-lg border border-amber-200 bg-amber-50 dark:border-amber-900/40 dark:bg-amber-950/20 px-5 py-4">
      <div className="flex items-start gap-3">
        <AlertCircle className="h-5 w-5 text-amber-500 mt-0.5 shrink-0" />
        <div>
          <p className="text-sm font-medium text-foreground">
            Found {datasets.length} ROT dataset{datasets.length !== 1 ? 's' : ''}:
          </p>
          <ul className="mt-2 space-y-0.5">
            {lines.map(({ count, label, description }) => (
              <li key={label} className="text-sm text-muted-foreground">
                <span className="font-medium text-foreground">{count} {label}</span>
                {' '}— {description}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  )
}

// ─── page ─────────────────────────────────────────────────────────────────────

export default function ROTPage() {
  const { data: summary, isLoading: summaryLoading, refetch: refetchSummary } = useROTSummary()
  const { data: datasets, isLoading: datasetsLoading, refetch: refetchDatasets } = useROTDatasets()
  const triggerScan = useTriggerROTScan()

  const [isScanning, setIsScanning] = useState(false)
  const [scanLogs, setScanLogs] = useState<string[]>([])
  const [isLogsExpanded, setIsLogsExpanded] = useState(false)
  const [selectedRow, setSelectedRow] = useState<ROTDataset | null>(null)
  const [dismissedIds, setDismissedIds] = useState<Set<string>>(new Set())

  const eventSourceRef = useRef<EventSource | null>(null)
  const logsEndRef = useRef<HTMLDivElement>(null)

  const datasetsData = (Array.isArray(datasets) ? datasets : []).filter(
    (d) => !dismissedIds.has(d.id),
  )

  // Auto-scroll logs
  useEffect(() => {
    if (logsEndRef.current && isLogsExpanded) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [scanLogs, isLogsExpanded])

  // Connect to SSE while scanning
  useEffect(() => {
    if (!isScanning || eventSourceRef.current) return

    const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
    const token = document.cookie
      .split('; ')
      .find((row) => row.startsWith('accessToken='))
      ?.split('=')[1]

    try {
      const baseUrl = apiUrl.replace(/\/api\/v1\/?$/, '')
      const es = new EventSource(`${baseUrl}/api/v1/notifications/events?token=${token}`)
      eventSourceRef.current = es

      es.onopen = () => {
        setScanLogs((prev) => [...prev, `[${new Date().toLocaleTimeString()}] Connected to scan stream`])
      }

      es.addEventListener('rot.scan.started', (event) => {
        const data = JSON.parse(event.data)
        setScanLogs((prev) => [...prev, `[${new Date().toLocaleTimeString()}] ${data.message || 'ROT scan started'}`])
      })

      es.addEventListener('rot.scan.progress', (event) => {
        const data = JSON.parse(event.data)
        setScanLogs((prev) => [...prev, `[${new Date().toLocaleTimeString()}] ${data.message}`])
      })

      es.addEventListener('rot.scan.completed', (event) => {
        const data = JSON.parse(event.data)
        setScanLogs((prev) => [...prev, `[${new Date().toLocaleTimeString()}] ${data.message || 'ROT scan completed'}`])
        setIsScanning(false)
        refetchSummary()
        refetchDatasets()
        if (eventSourceRef.current) {
          eventSourceRef.current.close()
          eventSourceRef.current = null
        }
      })

      es.onerror = () => {
        setScanLogs((prev) => [...prev, `[${new Date().toLocaleTimeString()}] Connection error - retrying...`])
      }
    } catch (error) {
      console.error('Failed to connect to SSE:', error)
    }

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
        eventSourceRef.current = null
      }
    }
  }, [isScanning, refetchSummary, refetchDatasets])

  function handleStartScan() {
    setIsScanning(true)
    setIsLogsExpanded(true)
    setScanLogs([`[${new Date().toLocaleTimeString()}] Initiating ROT scan...`])
    triggerScan.mutate()
  }

  function handleDismiss(id: string) {
    setDismissedIds((prev) => new Set([...prev, id]))
    setSelectedRow(null)
    toast.success('Dataset dismissed from list')
  }

  const columns = buildColumns(setSelectedRow)

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs items={[{ label: 'ROT Data', active: true }]} />
          <h1 className="text-3xl font-bold text-foreground mt-4">ROT Data Detection</h1>
          <p className="text-sm text-muted-foreground mt-1">Identify Redundant, Obsolete, and Trivial data</p>
        </div>
        <button
          onClick={handleStartScan}
          disabled={isScanning || triggerScan.isPending}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
        >
          {isScanning ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              Scanning...
            </>
          ) : (
            <>
              <Play className="h-4 w-4" />
              Run Scan
            </>
          )}
        </button>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Scan Logs - Collapsible */}
        {(isScanning || scanLogs.length > 0) && (
          <div className="rounded-lg border border-border bg-card overflow-hidden">
            <button
              onClick={() => setIsLogsExpanded(!isLogsExpanded)}
              className="w-full flex items-center justify-between p-4 hover:bg-muted/50 transition-colors"
            >
              <div className="flex items-center gap-3">
                {isScanning && <Loader2 className="h-4 w-4 animate-spin text-primary" />}
                <span className="font-medium text-foreground">
                  {isScanning ? 'Scan in Progress' : 'Scan Logs'}
                </span>
                <span className="text-sm text-muted-foreground">({scanLogs.length} entries)</span>
              </div>
              {isLogsExpanded ? (
                <ChevronUp className="h-4 w-4 text-muted-foreground" />
              ) : (
                <ChevronDown className="h-4 w-4 text-muted-foreground" />
              )}
            </button>

            {isLogsExpanded && (
              <div className="border-t border-border bg-muted/30 p-4 max-h-64 overflow-y-auto font-mono text-sm">
                {scanLogs.map((log, i) => (
                  <div key={i} className="text-muted-foreground py-0.5">{log}</div>
                ))}
                <div ref={logsEndRef} />
              </div>
            )}
          </div>
        )}

        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {summaryLoading ? (
            <>
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
            </>
          ) : (
            <>
              <StatCard
                label="Total ROT Data"
                value={(summary?.total_rot_data || 0).toString()}
                icon={<Trash2 className="h-6 w-6" />}
              />
              <StatCard
                label="Redundant"
                value={(summary?.redundant_count || 0).toString()}
                icon={<Copy className="h-6 w-6" />}
              />
              <StatCard
                label="Obsolete"
                value={(summary?.obsolete_count || 0).toString()}
                icon={<Clock className="h-6 w-6" />}
              />
              <StatCard
                label="Potential Savings"
                value={`${(summary?.potential_savings_gb || 0).toFixed(1)} GB`}
                icon={<HardDrive className="h-6 w-6" />}
              />
            </>
          )}
        </div>

        {/* Quick Links */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <Link
            href="/rot/duplicates"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <Copy className="h-8 w-8 text-blue-500 mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Duplicates</h3>
            <p className="text-sm text-muted-foreground mt-1">Find and deduplicate redundant data</p>
          </Link>

          <Link
            href="/rot/obsolete"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <Clock className="h-8 w-8 text-yellow-500 mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Obsolete</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Data that hasn&apos;t been accessed in a long time
            </p>
          </Link>

          <Link
            href="/rot/trivial"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <Trash2 className="h-8 w-8 text-gray-500 mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Trivial</h3>
            <p className="text-sm text-muted-foreground mt-1">Low-value data that can be safely removed</p>
          </Link>
        </div>

        {/* ROT Datasets */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Detected ROT Data</h3>

          {datasetsLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : datasetsData.length > 0 ? (
            <>
              <ROTSummaryBlurb datasets={datasetsData} />
              <div className="mt-4">
                <DataTable columns={columns} data={datasetsData} />
              </div>
            </>
          ) : (
            <div className="text-center py-8">
              <Trash2 className="h-12 w-12 mx-auto text-green-500 mb-4" />
              <p className="text-foreground font-medium">No ROT data detected</p>
              <p className="text-sm text-muted-foreground mt-1">
                Run a scan to identify redundant, obsolete, and trivial data
              </p>
            </div>
          )}
        </div>
      </div>

      {/* Detail slide-over panel */}
      {selectedRow && (
        <DetailPanel
          row={selectedRow}
          onClose={() => setSelectedRow(null)}
          onDismiss={handleDismiss}
        />
      )}
    </div>
  )
}
