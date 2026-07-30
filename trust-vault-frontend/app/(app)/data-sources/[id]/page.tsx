'use client'

import { useParams, useRouter } from 'next/navigation'
import { useEffect, useRef, useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { StatusIndicator } from '@/components/base/status-badge'
import { ArrowLeft, RefreshCw, Trash2, Play, Settings, Loader2, History, ChevronDown, ChevronRight, Clock, CheckCircle2, XCircle, Database, Tag, X, Info, AlertTriangle, Shield, ShieldAlert } from 'lucide-react'
import { useDataSource, useDeleteDataSource, useTriggerScan, useScanLogs, useDataSourceClassificationStats, ScanLog, ScanLogEntry } from '@/hooks/use-datasources'
import { toast } from 'sonner'
import Link from 'next/link'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'

export default function DataSourceDetailPage() {
  const params = useParams()
  const router = useRouter()
  const id = params.id as string
  const prevStatusRef = useRef<string | null>(null)
  const [localScanning, setLocalScanning] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [isRefreshing, setIsRefreshing] = useState(false)

  const { data: dataSource, isLoading, error, refetch } = useDataSource(id)
  const deleteDataSource = useDeleteDataSource()
  const triggerScan = useTriggerScan()

  // Determine if scanning - use local state OR actual status
  const isScanning = localScanning || dataSource?.status === 'scanning' || triggerScan.isPending

  // Poll for status updates when scanning
  useEffect(() => {
    if (isScanning) {
      const interval = setInterval(() => {
        refetch()
      }, 1000) // Poll every 1 second when scanning
      
      return () => clearInterval(interval)
    }
  }, [isScanning, refetch])

  // Clear local scanning state when actual status changes from scanning
  useEffect(() => {
    if (dataSource?.status && dataSource.status !== 'scanning') {
      setLocalScanning(false)
    }
  }, [dataSource?.status])

  // Show toast when scan completes (transition from scanning to another state)
  useEffect(() => {
    if (prevStatusRef.current === 'scanning' && dataSource?.status !== 'scanning') {
      if (dataSource?.status === 'connected') {
        toast.success('Scan completed successfully - connection verified')
      } else if (dataSource?.status === 'error') {
        toast.error('Scan failed - check connection settings')
      }
    }
    prevStatusRef.current = dataSource?.status || null
  }, [dataSource?.status])

  const handleDelete = async () => {
    try {
      await deleteDataSource.mutateAsync(id)
      setDeleteDialogOpen(false)
      router.push('/data-sources')
    } catch (error) {
      // Error handled by hook
    }
  }

  const handleScan = async () => {
    // Set local scanning state immediately for instant UI feedback
    setLocalScanning(true)
    try {
      await triggerScan.mutateAsync(id)
      // Immediately refetch to show scanning status, then poll rapidly
      refetch()
      setTimeout(() => refetch(), 200)
      setTimeout(() => refetch(), 500)
      setTimeout(() => refetch(), 1000)
    } catch (error) {
      setLocalScanning(false)
      // Error handled by hook
    }
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background p-8">
        <Skeleton className="h-8 w-48 mb-4" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (error || !dataSource) {
    return (
      <div className="min-h-screen bg-background p-8">
        <div className="text-center py-12">
          <p className="text-destructive">Data source not found</p>
          <Link href="/data-sources" className="mt-4 text-primary hover:underline">
            Back to Data Sources
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Data Sources', href: '/data-sources' },
            { label: dataSource.name, active: true },
          ]}
        />
        <div className="flex items-center justify-between mt-4">
          <div className="flex items-center gap-4">
            <Link href="/data-sources" className="p-2 rounded-lg hover:bg-muted transition-colors">
              <ArrowLeft className="h-5 w-5" />
            </Link>
            <div>
              <h1 className="text-3xl font-bold text-foreground">{dataSource.name}</h1>
              <div className="flex items-center gap-3 mt-1">
                <span className="text-sm text-muted-foreground capitalize">{dataSource.type}</span>
                {isScanning ? (
                  <div className="flex items-center gap-2 px-2 py-1 rounded-full bg-yellow-500/20 text-yellow-600 dark:text-yellow-400">
                    <Loader2 className="h-3 w-3 animate-spin" />
                    <span className="text-xs font-medium">Scanning...</span>
                  </div>
                ) : (
                  <StatusIndicator
                    status={dataSource.status === 'connected' ? 'success' : dataSource.status === 'error' ? 'error' : 'warning'}
                    label={dataSource.status}
                  />
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={async () => {
                setIsRefreshing(true)
                await refetch()
                setIsRefreshing(false)
              }}
              disabled={isRefreshing}
              className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border bg-card text-foreground hover:bg-muted transition-colors active:scale-95 disabled:opacity-50"
            >
              <RefreshCw className={`h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
              {isRefreshing ? 'Refreshing...' : 'Refresh'}
            </button>
            <button
              onClick={handleScan}
              disabled={isScanning}
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
            <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
              <AlertDialogTrigger
                className={`flex items-center gap-2 px-4 py-2 rounded-lg border border-destructive text-destructive hover:bg-destructive/10 transition-colors cursor-pointer ${deleteDataSource.isPending ? 'opacity-50 pointer-events-none' : ''}`}
                disabled={deleteDataSource.isPending}
              >
                <Trash2 className="h-4 w-4" />
                Delete
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Delete Data Source</AlertDialogTitle>
                  <AlertDialogDescription>
                    Are you sure you want to delete &quot;{dataSource.name}&quot;? This action cannot be undone and will remove all associated data.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    variant="destructive"
                    onClick={handleDelete}
                    disabled={deleteDataSource.isPending}
                  >
                    {deleteDataSource.isPending ? 'Deleting...' : 'Delete'}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </div>
      </div>

      {/* Scanning Progress Banner */}
      {isScanning && (
        <div className="bg-yellow-500/10 border-b border-yellow-500/20 px-8 py-3">
          <div className="flex items-center gap-3">
            <Loader2 className="h-5 w-5 animate-spin text-yellow-600 dark:text-yellow-400" />
            <div>
              <p className="text-sm font-medium text-yellow-700 dark:text-yellow-300">Scan in progress</p>
              <p className="text-xs text-yellow-600 dark:text-yellow-400">Testing connection and discovering schema...</p>
            </div>
          </div>
        </div>
      )}

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Details Card */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Details</h3>
          <div className="grid grid-cols-2 gap-6">
            <div>
              <p className="text-sm text-muted-foreground">ID</p>
              <p className="text-sm font-mono text-foreground">{dataSource.id}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Type</p>
              <p className="text-sm text-foreground capitalize">{dataSource.type}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Status</p>
              {isScanning ? (
                <div className="flex items-center gap-2 text-yellow-600 dark:text-yellow-400">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span className="text-sm">Scanning...</span>
                </div>
              ) : (
                <StatusIndicator
                  status={dataSource.status === 'connected' ? 'success' : dataSource.status === 'error' ? 'error' : 'warning'}
                  label={dataSource.status}
                />
              )}
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Last Scan</p>
              <p className="text-sm text-foreground">
                {dataSource.last_scan && dataSource.last_scan !== '0001-01-01T00:00:00Z'
                  ? new Date(dataSource.last_scan).toLocaleString()
                  : 'Never'}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Created</p>
              <p className="text-sm text-foreground">{new Date(dataSource.created_at).toLocaleString()}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Updated</p>
              <p className="text-sm text-foreground">{new Date(dataSource.updated_at).toLocaleString()}</p>
            </div>
          </div>
        </div>

        {/* Classification Overview Card */}
        <ClassificationOverviewCard dataSourceId={id} />

        {/* Configuration Card */}
        {dataSource.config && Object.keys(dataSource.config).length > 0 && (
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="text-lg font-semibold text-foreground mb-4">Configuration</h3>
            <pre className="text-sm bg-muted p-4 rounded-lg overflow-auto">
              {JSON.stringify(dataSource.config, null, 2)}
            </pre>
          </div>
        )}

        {/* Scan Logs Card */}
        <ScanLogsCard dataSourceId={id} isScanning={isScanning} status={dataSource.status} />

        {/* Actions Card */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Actions</h3>
          <div className="flex flex-wrap gap-3">
            <button
              onClick={handleScan}
              disabled={isScanning}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary/10 text-primary hover:bg-primary/20 transition-colors disabled:opacity-50"
            >
              {isScanning ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Scanning...
                </>
              ) : (
                <>
                  <Play className="h-4 w-4" />
                  Trigger Full Scan
                </>
              )}
            </button>
            <Link
              href={`/classification?source=${dataSource.id}`}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary/10 text-primary hover:bg-primary/20 transition-colors"
            >
              View Classifications
            </Link>
            <Link
              href={`/lineage?source=${dataSource.id}`}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary/10 text-primary hover:bg-primary/20 transition-colors"
            >
              View Lineage
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}

// Scan Logs Component - Collapsible, only shows when scanning or has logs
function ScanLogsCard({ dataSourceId, isScanning, status }: { dataSourceId: string; isScanning: boolean; status: string }) {
  const [logs, setLogs] = useState<string[]>([])
  const [isConnected, setIsConnected] = useState(false)
  const [isExpanded, setIsExpanded] = useState(false)
  const [activeTab, setActiveTab] = useState<'current' | 'history'>('current')
  const [expandedHistoryId, setExpandedHistoryId] = useState<string | null>(null)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const eventSourceRef = useRef<EventSource | null>(null)
  const storedLogsLoadedRef = useRef(false)
  
  const { data: scanHistory, refetch: refetchHistory } = useScanLogs(dataSourceId)

  // Load stored log entries from the latest scan log's logs array.
  // Called after scan completes or on mount when scan is already in progress.
  const loadStoredLogs = (history: ScanLog[] | undefined) => {
    if (!history || history.length === 0) return
    const latest = history[0]
    if (!latest.logs || latest.logs.length === 0) return
    const storedLines = latest.logs
      .filter((e: ScanLogEntry) => e && e.message)
      .map((e: ScanLogEntry) => `[${e.time || new Date().toLocaleTimeString()}] ${e.message}`)
    if (storedLines.length === 0) return
    setLogs(prev => {
      // Merge: use stored lines as authoritative baseline; append any live SSE lines not already in stored
      const storedSet = new Set(storedLines)
      const extraLive = prev.filter(l => !storedSet.has(l))
      return [...storedLines, ...extraLive]
    })
  }

  // Auto-expand when scanning starts
  useEffect(() => {
    if (isScanning) {
      setIsExpanded(true)
      setActiveTab('current')
    }
  }, [isScanning])

  // On mount: if a scan is already running, load whatever progress has been stored so far
  useEffect(() => {
    if ((status === 'scanning' || isScanning) && !storedLogsLoadedRef.current && scanHistory) {
      storedLogsLoadedRef.current = true
      loadStoredLogs(scanHistory)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status, isScanning, scanHistory])

  // When scan completes: refetch history, then load stored log entries into the terminal
  useEffect(() => {
    if (!isScanning && status !== 'scanning') {
      refetchHistory().then(result => {
        if (result.data) {
          loadStoredLogs(result.data)
        }
      })
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isScanning, status])

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (isExpanded && activeTab === 'current') {
      logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [logs, isExpanded, activeTab])

  // Connect to SSE when scanning starts
  useEffect(() => {
    if (isScanning && !eventSourceRef.current) {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const token = document.cookie.split('; ').find(row => row.startsWith('accessToken='))?.split('=')[1]
      
      // Add initial log only if not already added
      setLogs(prev => {
        const hasStarted = prev.some(log => log.includes('Scan started...'))
        if (hasStarted) return prev
        return [...prev, `[${new Date().toLocaleTimeString()}] Scan started...`]
      })
      
      // Try to connect to SSE endpoint
      try {
        // Remove /api/v1 suffix if present since we add it below
        const baseUrl = apiUrl.replace(/\/api\/v1\/?$/, '')
        const es = new EventSource(`${baseUrl}/api/v1/notifications/events?token=${token}`)
        eventSourceRef.current = es
        
        es.onopen = () => {
          setIsConnected(true)
        }
        
        // Handler for processing scan events
        const handleScanEvent = (event: MessageEvent) => {
          try {
            const data = JSON.parse(event.data)
            // Only process events for this datasource
            if (data.datasource_id === dataSourceId || data.dataset_id === dataSourceId) {
              // Handle log lines from ingestion
              if (data.log_lines && Array.isArray(data.log_lines)) {
                data.log_lines.forEach((line: string) => {
                  if (line && line.trim()) {
                    setLogs(prev => [...prev, `[${new Date().toLocaleTimeString()}] ${line}`])
                  }
                })
              }
              // Handle single message
              if (data.message && typeof data.message === 'string' && data.message.trim()) {
                setLogs(prev => [...prev, `[${new Date().toLocaleTimeString()}] ${data.message}`])
              }
              // Handle status updates
              if (data.status === 'completed') {
                setLogs(prev => [...prev, `[${new Date().toLocaleTimeString()}] ✓ Scan completed - ${data.columns || data.datasets_discovered || 0} columns classified`])
                // Fetch stored logs from DB to ensure terminal is complete
                refetchHistory().then(result => { if (result.data) loadStoredLogs(result.data) })
              } else if (data.status === 'failed') {
                setLogs(prev => [...prev, `[${new Date().toLocaleTimeString()}] ✗ Scan failed: ${data.message || 'Unknown error'}`])
              }
            }
          } catch {
            // Ignore parse errors for non-JSON messages
          }
        }
        
        // Listen for named SSE events (backend sends event: datasource.scan.progress, etc.)
        es.addEventListener('datasource.scan.progress', handleScanEvent)
        es.addEventListener('datasource.scan.completed', handleScanEvent)
        es.addEventListener('datasource.scan.failed', handleScanEvent)
        es.addEventListener('datasource.scan.started', handleScanEvent)
        
        // Also listen for generic message events as fallback
        es.onmessage = handleScanEvent
        
        es.onerror = () => {
          setIsConnected(false)
        }
      } catch (err) {
        console.error('SSE connection failed:', err)
      }
    }
    
    // Cleanup when scanning stops
    if (!isScanning && eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
      setIsConnected(false)
      
      // Add completion log based on final status
      if (status === 'active' || status === 'connected') {
        setLogs(prev => [...prev, `[${new Date().toLocaleTimeString()}] ✓ Scan completed successfully`])
      } else if (status === 'error') {
        setLogs(prev => [...prev, `[${new Date().toLocaleTimeString()}] ✗ Scan failed - check logs for details`])
      }
    }
    
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
        eventSourceRef.current = null
      }
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isScanning, dataSourceId, status])

  // Helper to format duration
  const formatDuration = (startedAt: string, completedAt?: string) => {
    const start = new Date(startedAt).getTime()
    const end = completedAt ? new Date(completedAt).getTime() : Date.now()
    const durationMs = end - start
    const seconds = Math.floor(durationMs / 1000)
    const minutes = Math.floor(seconds / 60)
    if (minutes > 0) {
      return `${minutes}m ${seconds % 60}s`
    }
    return `${seconds}s`
  }

  // Helper to get status icon
  const getStatusIcon = (scanStatus: string) => {
    switch (scanStatus) {
      case 'success':
        return <CheckCircle2 className="h-4 w-4 text-green-500" />
      case 'failed':
        return <XCircle className="h-4 w-4 text-red-500" />
      case 'running':
        return <Loader2 className="h-4 w-4 text-yellow-500 animate-spin" />
      default:
        return <Clock className="h-4 w-4 text-muted-foreground" />
    }
  }

  // Don't render if no logs and not scanning and no history
  const hasHistory = scanHistory && scanHistory.length > 0
  const shouldShow = isScanning || logs.length > 0 || status === 'error' || hasHistory
  
  if (!shouldShow) {
    return null
  }

  return (
    <div className="rounded-lg border border-border bg-card overflow-hidden">
      {/* Collapsible Header */}
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full flex items-center justify-between p-4 hover:bg-muted/50 transition-colors"
      >
        <div className="flex items-center gap-3">
          <h3 className="text-lg font-semibold text-foreground">Scan Logs</h3>
          {isScanning && (
            <span className="flex items-center gap-1 text-xs text-yellow-600 dark:text-yellow-400 bg-yellow-500/10 px-2 py-0.5 rounded-full">
              <span className="h-2 w-2 rounded-full bg-yellow-500 animate-pulse" />
              Live
            </span>
          )}
          {status === 'error' && !isScanning && (
            <span className="text-xs text-red-500 bg-red-500/10 px-2 py-0.5 rounded-full">Failed</span>
          )}
          {status === 'connected' && !isScanning && logs.length > 0 && (
            <span className="text-xs text-green-500 bg-green-500/10 px-2 py-0.5 rounded-full">Success</span>
          )}
          {hasHistory && (
            <span className="text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
              {scanHistory.length} scan{scanHistory.length !== 1 ? 's' : ''}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">{logs.length} entries</span>
          <svg 
            className={`h-4 w-4 text-muted-foreground transition-transform ${isExpanded ? 'rotate-180' : ''}`}
            fill="none" 
            viewBox="0 0 24 24" 
            stroke="currentColor"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </button>
      
      {/* Collapsible Content */}
      {isExpanded && (
        <div className="border-t border-border">
          {/* Tabs */}
          <div className="flex border-b border-border">
            <button
              onClick={() => setActiveTab('current')}
              className={`flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors ${
                activeTab === 'current'
                  ? 'text-primary border-b-2 border-primary bg-muted/30'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              <Play className="h-4 w-4" />
              Current Scan
              {isScanning && (
                <span className="h-2 w-2 rounded-full bg-yellow-500 animate-pulse" />
              )}
            </button>
            <button
              onClick={() => setActiveTab('history')}
              className={`flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors ${
                activeTab === 'history'
                  ? 'text-primary border-b-2 border-primary bg-muted/30'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              <History className="h-4 w-4" />
              History
              {hasHistory && (
                <span className="text-xs bg-muted px-1.5 py-0.5 rounded">{scanHistory.length}</span>
              )}
            </button>
          </div>

          {/* Current Scan Tab */}
          {activeTab === 'current' && (
            <div className="p-4">
              <div className="flex items-center justify-end gap-2 mb-2">
                {isConnected && (
                  <span className="flex items-center gap-1 text-xs text-green-600 dark:text-green-400">
                    <span className="h-2 w-2 rounded-full bg-green-500" />
                    Connected
                  </span>
                )}
                <button
                  onClick={(e) => { e.stopPropagation(); setLogs([]); storedLogsLoadedRef.current = false; }}
                  className="text-xs text-muted-foreground hover:text-foreground"
                >
                  Clear
                </button>
              </div>
              <div className="bg-zinc-950 rounded-lg p-4 h-64 overflow-auto font-mono text-xs">
                {logs.length === 0 ? (
                  <p className="text-zinc-500">No logs yet. Click &quot;Run Scan&quot; to start.</p>
                ) : (
                  logs.map((log, i) => (
                    <div 
                      key={i} 
                      className={`py-0.5 ${
                        log.includes('✓') ? 'text-green-400' : 
                        log.includes('✗') || log.includes('error') || log.includes('failed') ? 'text-red-400' : 
                        log.includes('Connected') ? 'text-blue-400' :
                        'text-zinc-300'
                      }`}
                    >
                      {log}
                    </div>
                  ))
                )}
                <div ref={logsEndRef} />
              </div>
              {status === 'error' && (
                <p className="mt-3 text-sm text-muted-foreground">
                  For detailed error logs, check: <code className="bg-muted px-1 rounded">docker logs securelens-ingestion</code>
                </p>
              )}
            </div>
          )}

          {/* History Tab */}
          {activeTab === 'history' && (
            <div className="p-4">
              {!hasHistory ? (
                <div className="text-center py-8 text-muted-foreground">
                  <History className="h-8 w-8 mx-auto mb-2 opacity-50" />
                  <p>No scan history yet</p>
                  <p className="text-sm">Run a scan to see history here</p>
                </div>
              ) : (
                <div className="space-y-2">
                  {scanHistory.map((scan: ScanLog) => (
                    <div key={scan.id} className="border border-border rounded-lg overflow-hidden">
                      <button
                        onClick={() => setExpandedHistoryId(expandedHistoryId === scan.id ? null : scan.id)}
                        className="w-full flex items-center justify-between p-3 hover:bg-muted/50 transition-colors"
                      >
                        <div className="flex items-center gap-3">
                          {getStatusIcon(scan.status)}
                          <div className="text-left">
                            <div className="flex items-center gap-2">
                              <span className="text-sm font-medium">
                                {new Date(scan.started_at).toLocaleDateString()} at {new Date(scan.started_at).toLocaleTimeString()}
                              </span>
                              <span className={`text-xs px-2 py-0.5 rounded-full ${
                                scan.status === 'success' ? 'bg-green-500/10 text-green-600 dark:text-green-400' :
                                scan.status === 'failed' ? 'bg-red-500/10 text-red-600 dark:text-red-400' :
                                'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400'
                              }`}>
                                {scan.status}
                              </span>
                            </div>
                            <div className="flex items-center gap-4 text-xs text-muted-foreground mt-0.5">
                              <span className="flex items-center gap-1">
                                <Clock className="h-3 w-3" />
                                {formatDuration(scan.started_at, scan.completed_at)}
                              </span>
                              {scan.datasets_discovered > 0 && (
                                <span className="flex items-center gap-1">
                                  <Database className="h-3 w-3" />
                                  {scan.datasets_discovered} dataset{scan.datasets_discovered !== 1 ? 's' : ''}
                                </span>
                              )}
                            </div>
                          </div>
                        </div>
                        {expandedHistoryId === scan.id ? (
                          <ChevronDown className="h-4 w-4 text-muted-foreground" />
                        ) : (
                          <ChevronRight className="h-4 w-4 text-muted-foreground" />
                        )}
                      </button>
                      
                      {expandedHistoryId === scan.id && (
                        <div className="border-t border-border p-3 bg-muted/30">
                          <div className="space-y-2 text-sm">
                            <div className="flex justify-between">
                              <span className="text-muted-foreground">Started:</span>
                              <span>{new Date(scan.started_at).toLocaleString()}</span>
                            </div>
                            {scan.completed_at && (
                              <div className="flex justify-between">
                                <span className="text-muted-foreground">Completed:</span>
                                <span>{new Date(scan.completed_at).toLocaleString()}</span>
                              </div>
                            )}
                            <div className="flex justify-between">
                              <span className="text-muted-foreground">Duration:</span>
                              <span>{formatDuration(scan.started_at, scan.completed_at)}</span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-muted-foreground">Datasets Discovered:</span>
                              <span>{scan.datasets_discovered}</span>
                            </div>
                            {scan.message && (
                              <div className="pt-2 border-t border-border">
                                <span className="text-muted-foreground">Message:</span>
                                <p className={`mt-1 ${scan.status === 'failed' ? 'text-red-500' : ''}`}>
                                  {scan.message}
                                </p>
                              </div>
                            )}
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

// Entity type sensitivity levels and display names
const ENTITY_SENSITIVITY: Record<string, { level: 'high' | 'medium' | 'low'; shortName: string }> = {
  // High sensitivity (red)
  SSN: { level: 'high', shortName: 'SSN' },
  SOCIAL_SECURITY_NUMBER: { level: 'high', shortName: 'SSN' },
  CREDIT_CARD: { level: 'high', shortName: 'Card' },
  CREDIT_CARD_NUMBER: { level: 'high', shortName: 'Card' },
  PASSPORT: { level: 'high', shortName: 'Passport' },
  PASSPORT_NUMBER: { level: 'high', shortName: 'Passport' },
  BANK_ACCOUNT: { level: 'high', shortName: 'Bank' },
  BANK_ACCOUNT_NUMBER: { level: 'high', shortName: 'Bank' },
  DRIVER_LICENSE: { level: 'high', shortName: 'License' },
  DRIVER_LICENSE_NUMBER: { level: 'high', shortName: 'License' },
  TAX_ID: { level: 'high', shortName: 'Tax ID' },
  NATIONAL_ID: { level: 'high', shortName: 'Nat. ID' },
  
  // Medium sensitivity (orange)
  EMAIL: { level: 'medium', shortName: 'Email' },
  EMAIL_ADDRESS: { level: 'medium', shortName: 'Email' },
  PHONE: { level: 'medium', shortName: 'Phone' },
  PHONE_NUMBER: { level: 'medium', shortName: 'Phone' },
  ADDRESS: { level: 'medium', shortName: 'Address' },
  STREET_ADDRESS: { level: 'medium', shortName: 'Address' },
  DATE_OF_BIRTH: { level: 'medium', shortName: 'DOB' },
  DOB: { level: 'medium', shortName: 'DOB' },
  MEDICAL_RECORD: { level: 'medium', shortName: 'Medical' },
  HEALTH_INSURANCE: { level: 'medium', shortName: 'Health ID' },
  
  // Low sensitivity (yellow)
  PERSON_NAME: { level: 'low', shortName: 'Name' },
  NAME: { level: 'low', shortName: 'Name' },
  FIRST_NAME: { level: 'low', shortName: 'Name' },
  LAST_NAME: { level: 'low', shortName: 'Name' },
  IP_ADDRESS: { level: 'low', shortName: 'IP' },
  IP: { level: 'low', shortName: 'IP' },
  DATE: { level: 'low', shortName: 'Date' },
  AGE: { level: 'low', shortName: 'Age' },
  GENDER: { level: 'low', shortName: 'Gender' },
  LOCATION: { level: 'low', shortName: 'Location' },
  CITY: { level: 'low', shortName: 'City' },
  STATE: { level: 'low', shortName: 'State' },
  COUNTRY: { level: 'low', shortName: 'Country' },
  ZIP_CODE: { level: 'low', shortName: 'ZIP' },
  POSTAL_CODE: { level: 'low', shortName: 'ZIP' },
}

function getEntityInfo(type: string): { level: 'high' | 'medium' | 'low'; shortName: string } {
  const normalized = type.toUpperCase().replace(/-/g, '_')
  return ENTITY_SENSITIVITY[normalized] || { level: 'low', shortName: type.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase()).substring(0, 12) }
}

function getSensitivityColor(level: 'high' | 'medium' | 'low'): { bg: string; text: string; border: string } {
  switch (level) {
    case 'high':
      return { bg: 'bg-red-500/10', text: 'text-red-600 dark:text-red-400', border: 'border-red-500/30' }
    case 'medium':
      return { bg: 'bg-orange-500/10', text: 'text-orange-600 dark:text-orange-400', border: 'border-orange-500/30' }
    case 'low':
      return { bg: 'bg-yellow-500/10', text: 'text-yellow-600 dark:text-yellow-400', border: 'border-yellow-500/30' }
  }
}

function getSensitivityIcon(level: 'high' | 'medium' | 'low') {
  switch (level) {
    case 'high':
      return <ShieldAlert className="h-3 w-3" />
    case 'medium':
      return <AlertTriangle className="h-3 w-3" />
    case 'low':
      return <Shield className="h-3 w-3" />
  }
}

// Entity Types Details Modal
function EntityTypesModal({ 
  entities, 
  isOpen, 
  onClose 
}: { 
  entities: Array<{ type: string; count: number }>
  isOpen: boolean
  onClose: () => void 
}) {
  if (!isOpen) return null

  const totalCount = entities.reduce((sum, e) => sum + e.count, 0)
  const sortedEntities = [...entities].sort((a, b) => b.count - a.count)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-card border border-border rounded-lg shadow-xl max-w-lg w-full mx-4 max-h-[80vh] overflow-hidden">
        <div className="flex items-center justify-between p-4 border-b border-border">
          <div className="flex items-center gap-2">
            <Tag className="h-5 w-5 text-primary" />
            <h3 className="text-lg font-semibold">Entity Types Breakdown</h3>
          </div>
          <button onClick={onClose} className="p-1 rounded hover:bg-muted transition-colors">
            <X className="h-5 w-5" />
          </button>
        </div>
        
        <div className="p-4 overflow-y-auto max-h-[60vh]">
          <div className="mb-4 p-3 rounded-lg bg-muted/50">
            <div className="flex justify-between items-center">
              <span className="text-sm text-muted-foreground">Total Entity Types</span>
              <span className="font-semibold">{entities.length}</span>
            </div>
            <div className="flex justify-between items-center mt-1">
              <span className="text-sm text-muted-foreground">Total Occurrences</span>
              <span className="font-semibold">{totalCount.toLocaleString()}</span>
            </div>
          </div>

          <div className="space-y-2">
            {sortedEntities.map((entity) => {
              const info = getEntityInfo(entity.type)
              const colors = getSensitivityColor(info.level)
              const percentage = totalCount > 0 ? ((entity.count / totalCount) * 100).toFixed(1) : '0'
              
              return (
                <div 
                  key={entity.type} 
                  className={`flex items-center justify-between p-3 rounded-lg border ${colors.bg} ${colors.border}`}
                >
                  <div className="flex items-center gap-3">
                    <span className={colors.text}>
                      {getSensitivityIcon(info.level)}
                    </span>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-sm">{info.shortName}</span>
                        <span className="text-xs text-muted-foreground">({entity.type})</span>
                      </div>
                      <div className="flex items-center gap-2 mt-0.5">
                        <span className={`text-xs capitalize ${colors.text}`}>{info.level} risk</span>
                      </div>
                    </div>
                  </div>
                  <div className="text-right">
                    <span className="font-semibold">{entity.count.toLocaleString()}</span>
                    <span className="text-xs text-muted-foreground ml-1">({percentage}%)</span>
                  </div>
                </div>
              )
            })}
          </div>
        </div>

        <div className="p-4 border-t border-border bg-muted/30">
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <div className="flex items-center gap-4">
              <span className="flex items-center gap-1">
                <span className="h-2 w-2 rounded-full bg-red-500" /> High
              </span>
              <span className="flex items-center gap-1">
                <span className="h-2 w-2 rounded-full bg-orange-500" /> Medium
              </span>
              <span className="flex items-center gap-1">
                <span className="h-2 w-2 rounded-full bg-yellow-500" /> Low
              </span>
            </div>
            <button onClick={onClose} className="text-primary hover:underline">
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

// Classification Overview Card - Shows classification stats for this datasource
function ClassificationOverviewCard({ dataSourceId }: { dataSourceId: string }) {
  const { data: stats, isLoading } = useDataSourceClassificationStats(dataSourceId)
  const [showAllEntities, setShowAllEntities] = useState(false)
  const [showDetailsModal, setShowDetailsModal] = useState(false)

  if (isLoading) {
    return (
      <div className="rounded-lg border border-border bg-card p-6">
        <div className="flex items-center justify-between mb-4">
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-4 w-16" />
        </div>
        <div className="space-y-4">
          <Skeleton className="h-10 w-32" />
          <Skeleton className="h-20 w-full" />
        </div>
      </div>
    )
  }

  const hasClassifications = stats && stats.total_classifications > 0
  const sortedEntities = stats?.top_entities 
    ? [...stats.top_entities].sort((a, b) => b.count - a.count)
    : []
  const displayedEntities = sortedEntities.slice(0, 5)
  const remainingCount = sortedEntities.length - 5

  return (
    <div className="rounded-lg border border-border bg-card p-6">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Tag className="h-5 w-5 text-primary" />
          <h3 className="text-lg font-semibold text-foreground">Classification Overview</h3>
        </div>
        <Link href={`/classification?source=${dataSourceId}`} className="text-sm text-primary hover:underline">
          View Details
        </Link>
      </div>

      {!hasClassifications ? (
        <div className="text-center py-8">
          <Tag className="h-12 w-12 mx-auto mb-3 text-muted-foreground opacity-50" />
          <p className="text-muted-foreground">No classifications yet</p>
          <p className="text-sm text-muted-foreground mt-1">Run a scan to classify data in this source</p>
        </div>
      ) : (
        <div className="space-y-4">
          <div className="flex items-baseline gap-2">
            <span className="text-3xl font-bold text-foreground">{stats.columns_classified.toLocaleString()}</span>
            <span className="text-sm text-muted-foreground">columns classified</span>
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div className="p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-foreground">{stats.total_classifications.toLocaleString()}</p>
              <p className="text-xs text-muted-foreground">Total Findings</p>
            </div>
            <div className="p-3 rounded-lg bg-red-500/10">
              <p className="text-2xl font-bold text-red-600">{stats.pii_detected.toLocaleString()}</p>
              <p className="text-xs text-muted-foreground">PII Detected</p>
            </div>
            <div className="p-3 rounded-lg bg-orange-500/10">
              <p className="text-2xl font-bold text-orange-600">{stats.high_risk.toLocaleString()}</p>
              <p className="text-xs text-muted-foreground">High Risk</p>
            </div>
          </div>

          {sortedEntities.length > 0 && (
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <p className="text-xs text-muted-foreground font-medium">Top Entity Types</p>
                {sortedEntities.length > 0 && (
                  <button 
                    onClick={() => setShowDetailsModal(true)}
                    className="text-xs text-primary hover:underline flex items-center gap-1"
                  >
                    <Info className="h-3 w-3" />
                    Details
                  </button>
                )}
              </div>
              <div className="flex flex-wrap gap-1.5">
                {displayedEntities.map((entity) => {
                  const info = getEntityInfo(entity.type)
                  const colors = getSensitivityColor(info.level)
                  
                  return (
                    <span 
                      key={entity.type} 
                      className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs font-medium border ${colors.bg} ${colors.text} ${colors.border}`}
                      title={`${entity.type} - ${info.level} sensitivity`}
                    >
                      {getSensitivityIcon(info.level)}
                      {info.shortName}
                      <span className="opacity-70">({entity.count.toLocaleString()})</span>
                    </span>
                  )
                })}
                {remainingCount > 0 && (
                  <button
                    onClick={() => setShowDetailsModal(true)}
                    className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs font-medium bg-muted hover:bg-muted/80 text-muted-foreground transition-colors"
                  >
                    +{remainingCount} more
                  </button>
                )}
              </div>
            </div>
          )}

          <div className="pt-2 border-t border-border">
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Avg Confidence</span>
              <span className="font-medium text-foreground">
                {stats.avg_confidence ? `${Math.round(stats.avg_confidence * 100)}%` : '-'}
              </span>
            </div>
          </div>
        </div>
      )}

      <EntityTypesModal 
        entities={sortedEntities}
        isOpen={showDetailsModal}
        onClose={() => setShowDetailsModal(false)}
      />
    </div>
  )
}
