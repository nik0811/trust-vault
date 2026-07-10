'use client'

import { useEffect, use } from 'react'
import { StatCard } from '@/components/base/stat-card'
import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { StatusIndicator } from '@/components/base/status-badge'
import { ArrowLeft, Zap, TrendingUp, Settings, RefreshCw, Loader2, Table } from 'lucide-react'
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
  const { data: dataset, isLoading: datasetLoading, error: datasetError } = useDatasetClassification(id)
  const { data: columnsData, isLoading: columnsLoading, error: columnsError } = useDatasetColumns(id)
  const { data: rulesData, isLoading: rulesLoading } = useClassificationRules()
  const { data: modelsData, isLoading: modelsLoading } = useClassificationModels()
  const reclassify = useReclassifyDataset()

  useEffect(() => {
    if (datasetError) toast.error('Failed to load dataset classification')
    if (columnsError) toast.error('Failed to load column data')
  }, [datasetError, columnsError])

  const isLoading = datasetLoading || columnsLoading
  const columnsList: ColumnClassification[] = columnsData || dataset?.columns || []
  const rules: string[] = (rulesData || []).map((r: any) => r.name)
  const models: string[] = (modelsData || []).map((m: any) => `${m.name} ${m.version || ''}`.trim())

  const datasetName = dataset?.name || `Dataset ${id}`
  const totalColumns = dataset?.total_columns || columnsList.length
  const classifiedColumns = dataset?.classified_columns || columnsList.filter(c => c.status === 'classified').length
  const pendingColumns = dataset?.pending_columns || columnsList.filter(c => c.status === 'pending').length
  const avgConfidence = dataset?.avg_confidence || (columnsList.length > 0 
    ? Math.round(columnsList.reduce((a, b) => a + b.confidence, 0) / columnsList.length) 
    : 0)

  const handleReclassify = async () => {
    try {
      await reclassify.mutateAsync(id)
    } catch {
      // Error handled by hook
    }
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
              onClick={handleReclassify}
              disabled={reclassify.isPending}
              className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors text-sm font-medium disabled:opacity-50"
            >
              {reclassify.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <RefreshCw className="h-4 w-4" />
              )}
              Re-classify
            </button>
          </div>

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
