'use client'

import { useEffect, use, useState, useRef } from 'react'
import { StatCard } from '@/components/base/stat-card'
import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { StatusIndicator } from '@/components/base/status-badge'
import { ArrowLeft, Zap, TrendingUp, Settings, RefreshCw, Loader2, Table, Play, CheckCircle2, XCircle, Clock } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useDatasetClassification, useDatasetColumns, useClassificationRules, useClassificationModels, useReclassifyDataset, type ColumnClassification } from '@/hooks/use-classification'
import { toast } from 'sonner'

const getSensitivityColor = (level: string) => {
  switch (level) {
    case 'critical':
      return 'bg-red-500/20 text-red-700 dark:text-red-400'
    case 'high':
      return 'bg-orange-500/20 text-orange-700 dark:text-orange-400'
    case 'medium':
      return 'bg-yellow-500/20 text-yellow-700 dark:text-yellow-400'
    case 'low':
      return 'bg-green-500/20 text-green-700 dark:text-green-400'
    default:
      return 'bg-gray-500/20 text-gray-700 dark:text-gray-400'
  }
}

const columns: Column<ColumnClassification>[] = [
  {
    id: 'column_name',
    header: 'Column Name',
    cell: (row) => <span className="font-medium">{row.column_name}</span>,
  },
  {
    id: 'data_type',
    header: 'Data Type',
    cell: (row) => <code className="text-xs bg-muted px-2 py-1 rounded">{row.data_type}</code>,
  },
  {
    id: 'sensitivity_level',
    header: 'Sensitivity',
    cell: (row) => (
      <span className={`inline-block px-3 py-1 rounded-full text-xs font-medium ${getSensitivityColor(row.sensitivity_level)}`}>
        {row.sensitivity_level.charAt(0).toUpperCase() + row.sensitivity_level.slice(1)}
      </span>
    ),
  },
  {
    id: 'classification_tag',
    header: 'Classification',
    cell: (row) => <span className="text-sm">{row.classification_tag}</span>,
  },
  {
    id: 'confidence',
    header: 'Confidence',
    cell: (row) => (
      <div className="flex items-center gap-2">
        <div className="w-16 h-2 bg-muted rounded-full overflow-hidden">
          <div
            className="h-full bg-primary transition-all"
            style={{ width: `${row.confidence}%` }}
          />
        </div>
        <span className="text-sm font-medium">{row.confidence}%</span>
      </div>
    ),
  },
  {
    id: 'status',
    header: 'Status',
    cell: (row) => (
      <StatusIndicator status={row.status === 'classified' ? 'active' : row.status === 'pending' ? 'pending' : 'warning'} label={row.status} />
    ),
  },
]

export default function ClassificationDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params)
  const router = useRouter()
  const { data: dataset, isLoading: datasetLoading, error: datasetError, refetch: refetchDataset } = useDatasetClassification(id)
  const { data: columnsData, isLoading: columnsLoading, error: columnsError, refetch: refetchColumns } = useDatasetColumns(id)
  const { data: rulesData, isLoading: rulesLoading } = useClassificationRules()
  const { data: modelsData, isLoading: modelsLoading } = useClassificationModels()
  const reclassify = useReclassifyDataset()
  const [isClassifying, setIsClassifying] = useState(false)

  useEffect(() => {
    if (datasetError) toast.error('Failed to load dataset classification')
    if (columnsError) toast.error('Failed to load column data')
  }, [datasetError, columnsError])

  const isLoading = datasetLoading || columnsLoading
  const columnsList: ColumnClassification[] = columnsData || dataset?.columns || []
  const rules: string[] = (rulesData || []).map((r: any) => r.name)
  const models: string[] = (modelsData || []).map((m: any) => m.name)

  const datasetName = dataset?.name || `Dataset ${id}`
  const totalColumns = dataset?.total_columns || columnsList.length
  const classifiedColumns = dataset?.classified_columns || columnsList.filter(c => c.status === 'classified').length
  const pendingColumns = dataset?.pending_columns || columnsList.filter(c => c.status === 'pending').length
  const avgConfidence = dataset?.avg_confidence 
    ? (dataset.avg_confidence * 100).toFixed(1)
    : (columnsList.length > 0 
      ? (columnsList.reduce((a, b) => a + (b.confidence || 0), 0) / columnsList.length * 100).toFixed(1)
      : '0')

  // Determine if dataset has been classified before
  const hasBeenClassified = classifiedColumns > 0 || columnsList.length > 0

  const handleClassify = async () => {
    try {
      setIsClassifying(true)
      await reclassify.mutateAsync(id)
    } catch {
      // Error handled by hook
    }
  }

  // Handle classification completion - refetch data and reset state
  const handleClassificationComplete = () => {
    setIsClassifying(false)
    refetchDataset()
    refetchColumns()
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6">
        <div className="flex items-center gap-4">
          <button
            onClick={() => router.back()}
            className="p-2 hover:bg-muted rounded-lg transition-colors"
          >
            <ArrowLeft className="h-5 w-5" />
          </button>
          <div>
            <Breadcrumbs
              items={[
                { label: 'Classification', href: '/classification' },
                { label: datasetName, active: true },
              ]}
            />
            <h1 className="text-3xl font-bold text-foreground mt-4">{datasetName}</h1>
            <p className="text-sm text-muted-foreground mt-1">Column-level classification and sensitivity analysis</p>
          </div>
        </div>
      </div>

      <div className="p-8 space-y-8">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <StatCard label="Total Columns" value={String(totalColumns)} icon={<Zap className="h-6 w-6" />} />
          <StatCard label="Classified" value={String(classifiedColumns)} icon={<TrendingUp className="h-6 w-6" />} />
          <StatCard label="Pending Review" value={String(pendingColumns)} />
          <StatCard label="Avg Confidence" value={`${avgConfidence}%`} icon={<Settings className="h-6 w-6" />} />
        </div>

        <div className="rounded-lg border border-border bg-card p-6">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-lg font-semibold text-foreground">Column Classifications</h2>
            <button 
              onClick={handleClassify}
              disabled={reclassify.isPending || isClassifying}
              className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors text-sm font-medium disabled:opacity-50"
            >
              {reclassify.isPending || isClassifying ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : hasBeenClassified ? (
                <RefreshCw className="h-4 w-4" />
              ) : (
                <Play className="h-4 w-4" />
              )}
              {hasBeenClassified ? 'Re-classify' : 'Classify'}
            </button>
          </div>

          {/* Classification Status Card */}
          <ClassificationStatusCard 
            datasetId={id} 
            isClassifying={isClassifying || reclassify.isPending}
            onComplete={handleClassificationComplete}
          />

          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : columnsList.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              <Table className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>No column data available</p>
            </div>
          ) : (
            <DataTable columns={columns} data={columnsList} />
          )}
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="font-semibold text-foreground mb-4">Applied Rules</h3>
            {rulesLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            ) : rules.length === 0 ? (
              <p className="text-center py-4 text-sm text-muted-foreground">No rules configured</p>
            ) : (
              <div className="space-y-3">
                {rules.map((rule) => (
                  <div key={rule} className="flex items-center justify-between p-3 bg-muted/50 rounded-lg">
                    <span className="text-sm">{rule}</span>
                    <span className="text-xs bg-primary/20 text-primary px-2 py-1 rounded">Active</span>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="font-semibold text-foreground mb-4">Classification Models</h3>
            {modelsLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            ) : models.length === 0 ? (
              <p className="text-center py-4 text-sm text-muted-foreground">No models configured</p>
            ) : (
              <div className="space-y-3">
                {models.map((model) => (
                  <div key={model} className="flex items-center justify-between p-3 bg-muted/50 rounded-lg">
                    <span className="text-sm">{model}</span>
                    <span className="text-xs bg-green-500/20 text-green-700 dark:text-green-400 px-2 py-1 rounded">Active</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

// Classification Status Card - Shows real-time job progress
function ClassificationStatusCard({ 
  datasetId, 
  isClassifying, 
  onComplete 
}: { 
  datasetId: string
  isClassifying: boolean
  onComplete: () => void 
}) {
  const [logs, setLogs] = useState<string[]>([])
  const [isConnected, setIsConnected] = useState(false)
  const [isExpanded, setIsExpanded] = useState(false)
  const [jobStatus, setJobStatus] = useState<'idle' | 'queued' | 'running' | 'completed' | 'failed'>('idle')
  const [progress, setProgress] = useState({ current: 0, total: 0 })
  const logsEndRef = useRef<HTMLDivElement>(null)
  const eventSourceRef = useRef<EventSource | null>(null)

  // Auto-expand when classification starts
  useEffect(() => {
    if (isClassifying) {
      setIsExpanded(true)
      setJobStatus('queued')
      setLogs([`[${new Date().toLocaleTimeString()}] Classification job queued...`])
    }
  }, [isClassifying])

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (isExpanded) {
      logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [logs, isExpanded])

  // Connect to SSE when classification starts
  useEffect(() => {
    if (isClassifying && !eventSourceRef.current) {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const token = document.cookie.split('; ').find(row => row.startsWith('accessToken='))?.split('=')[1]
      
      try {
        // Remove /api/v1 suffix if present since we add it below
        const baseUrl = apiUrl.replace(/\/api\/v1\/?$/, '')
        const es = new EventSource(`${baseUrl}/api/v1/notifications/events?token=${token}`)
        eventSourceRef.current = es
        
        es.onopen = () => {
          setIsConnected(true)
          setLogs(prev => {
            const connectMsg = `Connected to classification stream`
            if (prev.some(log => log.includes(connectMsg))) {
              return prev
            }
            return [...prev, `[${new Date().toLocaleTimeString()}] ${connectMsg}`]
          })
        }
        
        // Handler for processing classification events
        const handleClassificationEvent = (event: MessageEvent) => {
          try {
            const data = JSON.parse(event.data)
            // Only process classification events for this dataset
            // Events come with dataset_id in the data payload
            if (data.dataset_id === datasetId) {
              // Handle progress updates
              if (data.progress) {
                setProgress({ current: data.progress.current || 0, total: data.progress.total || 0 })
                setJobStatus('running')
              }
              // Handle log messages
              if (data.message && typeof data.message === 'string' && data.message.trim()) {
                setLogs(prev => [...prev, `[${new Date().toLocaleTimeString()}] ${data.message}`])
              }
              // Handle status updates
              if (data.status === 'completed') {
                setJobStatus('completed')
                setLogs(prev => [...prev, `[${new Date().toLocaleTimeString()}] ✓ Classification completed - ${data.columns_classified || 0} columns classified`])
                setTimeout(() => onComplete(), 1000)
              } else if (data.status === 'failed') {
                setJobStatus('failed')
                setLogs(prev => [...prev, `[${new Date().toLocaleTimeString()}] ✗ Classification failed: ${data.error || 'Unknown error'}`])
              } else if (data.status === 'running') {
                setJobStatus('running')
              }
            }
          } catch {
            // Ignore parse errors for non-JSON messages
          }
        }
        
        // Listen for named SSE events (backend sends event: classification.progress, etc.)
        es.addEventListener('classification.started', handleClassificationEvent)
        es.addEventListener('classification.progress', handleClassificationEvent)
        es.addEventListener('classification.completed', handleClassificationEvent)
        es.addEventListener('classification.failed', handleClassificationEvent)
        es.addEventListener('classification.queued', handleClassificationEvent)
        
        // Also listen for generic message events as fallback
        es.onmessage = handleClassificationEvent
        
        es.onerror = () => {
          setIsConnected(false)
        }
      } catch (err) {
        console.error('SSE connection failed:', err)
      }
    }
    
    // Cleanup when classification stops
    if (!isClassifying && eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
      setIsConnected(false)
    }
    
    // When classification stops, trigger completion after a delay to allow backend to finish
    if (!isClassifying && logs.length > 0 && jobStatus !== 'completed' && jobStatus !== 'failed') {
      const timer = setTimeout(() => {
        setLogs(prev => [...prev, `[${new Date().toLocaleTimeString()}] ✓ Classification completed`])
        setJobStatus('completed')
        onComplete()
      }, 3000)
      return () => clearTimeout(timer)
    }
    
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
        eventSourceRef.current = null
      }
    }
  }, [isClassifying, datasetId, onComplete, jobStatus, logs.length])

  // Don't render if no activity
  if (!isClassifying && logs.length === 0) {
    return null
  }

  const getStatusIcon = () => {
    switch (jobStatus) {
      case 'completed':
        return <CheckCircle2 className="h-4 w-4 text-green-500" />
      case 'failed':
        return <XCircle className="h-4 w-4 text-red-500" />
      case 'running':
        return <Loader2 className="h-4 w-4 text-yellow-500 animate-spin" />
      case 'queued':
        return <Clock className="h-4 w-4 text-blue-500" />
      default:
        return <Clock className="h-4 w-4 text-muted-foreground" />
    }
  }

  const getStatusLabel = () => {
    switch (jobStatus) {
      case 'completed':
        return 'Completed'
      case 'failed':
        return 'Failed'
      case 'running':
        return 'Running'
      case 'queued':
        return 'Queued'
      default:
        return 'Idle'
    }
  }

  return (
    <div className="rounded-lg border border-border bg-muted/30 overflow-hidden mb-6">
      {/* Collapsible Header */}
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full flex items-center justify-between p-4 hover:bg-muted/50 transition-colors"
      >
        <div className="flex items-center gap-3">
          <h3 className="text-sm font-semibold text-foreground">Classification Status</h3>
          {isClassifying && (
            <span className="flex items-center gap-1 text-xs text-yellow-600 dark:text-yellow-400 bg-yellow-500/10 px-2 py-0.5 rounded-full">
              <span className="h-2 w-2 rounded-full bg-yellow-500 animate-pulse" />
              Live
            </span>
          )}
          <span className="flex items-center gap-1 text-xs text-muted-foreground">
            {getStatusIcon()}
            {getStatusLabel()}
          </span>
          {progress.total > 0 && (
            <span className="text-xs text-muted-foreground">
              ({progress.current}/{progress.total} columns)
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
          {/* Progress Bar */}
          {progress.total > 0 && (
            <div className="px-4 py-2 border-b border-border">
              <div className="flex items-center justify-between text-xs text-muted-foreground mb-1">
                <span>Progress</span>
                <span>{Math.round((progress.current / progress.total) * 100)}%</span>
              </div>
              <div className="w-full h-2 bg-muted rounded-full overflow-hidden">
                <div 
                  className="h-full bg-primary transition-all duration-300"
                  style={{ width: `${(progress.current / progress.total) * 100}%` }}
                />
              </div>
            </div>
          )}
          
          {/* Logs */}
          <div className="max-h-48 overflow-y-auto p-4 font-mono text-xs bg-background/50">
            {logs.length === 0 ? (
              <p className="text-muted-foreground">Waiting for classification to start...</p>
            ) : (
              logs.map((log, i) => (
                <div 
                  key={i} 
                  className={`py-0.5 ${
                    log.includes('✓') ? 'text-green-600 dark:text-green-400' :
                    log.includes('✗') ? 'text-red-600 dark:text-red-400' :
                    log.includes('Connected') ? 'text-blue-600 dark:text-blue-400' :
                    'text-muted-foreground'
                  }`}
                >
                  {log}
                </div>
              ))
            )}
            <div ref={logsEndRef} />
          </div>
        </div>
      )}
    </div>
  )
}
