'use client'

import { useMemo, useState } from 'react'
import { StatCard } from '@/components/base/stat-card'
import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { EmptyState } from '@/components/base/empty-state'
import Link from 'next/link'
import { Zap, TrendingUp, Settings, Search, FileText } from 'lucide-react'
import { useClassifyText, useClassificationModels, useClassificationRules, useClassifyStats } from '@/hooks/use-classification'
import { useDataSources } from '@/hooks/use-datasources'
import { useFeedbackStats } from '@/hooks/use-advisor'

interface ClassificationEntry {
  id: string
  dataset: string
  classifiedColumns: number
  totalColumns: number
  confidence: number
  lastClassified: Date
}

const columns: Column<ClassificationEntry>[] = [
  {
    id: 'dataset',
    header: 'Dataset',
    cell: (row) => (
      <Link href={`/classification/${row.id}`} className="text-primary hover:underline font-medium">
        {row.dataset}
      </Link>
    ),
  },
  {
    id: 'progress',
    header: 'Classified Columns',
    cell: (row) => (
      <div className="flex items-center gap-2">
        <div className="w-32 h-2 bg-muted rounded-full overflow-hidden">
          <div
            className="h-full bg-primary transition-all"
            style={{ width: row.totalColumns > 0 ? `${(row.classifiedColumns / row.totalColumns) * 100}%` : '0%' }}
          />
        </div>
        <span className="text-sm">
          {row.classifiedColumns}/{row.totalColumns}
        </span>
      </div>
    ),
  },
  {
    id: 'confidence',
    header: 'Avg Confidence',
    cell: (row) => (
      <div className="flex items-center gap-2">
        <span className="font-medium">{row.confidence}%</span>
      </div>
    ),
  },
  {
    id: 'lastClassified',
    header: 'Last Classified',
    cell: (row) => new Date(row.lastClassified).toLocaleDateString(),
  },
]

export default function ClassificationPage() {
  const [testText, setTestText] = useState('')
  const { data: dataSources, isLoading: dsLoading } = useDataSources()
  const { data: models, isLoading: modelsLoading } = useClassificationModels()
  const { data: rules, isLoading: rulesLoading } = useClassificationRules()
  const { data: feedbackStats } = useFeedbackStats()
  const { data: classifyStats } = useClassifyStats()
  const classifyText = useClassifyText()

  const classifications = useMemo(() => {
    if (!Array.isArray(dataSources)) return []
    const perDataset = classifyStats?.per_dataset ?? []
    return dataSources.map(ds => {
      const dsData = perDataset.find(p => p.dataset_id === ds.id)
      return {
        id: ds.id,
        dataset: ds.name,
        classifiedColumns: dsData?.classified_columns ?? 0,
        totalColumns: dsData?.classified_columns ?? 0, // we only know classified count
        confidence: dsData ? Math.round(dsData.avg_confidence * 100) : 0,
        lastClassified: new Date(ds.last_scan || ds.created_at),
      }
    }).filter(c => c.classifiedColumns > 0 || dsLoading)
  }, [dataSources, classifyStats, dsLoading])

  const stats = useMemo(() => {
    const totalClassified = classifyStats?.total_classified ?? 0
    const avgConfidence = classifyStats?.avg_confidence != null
      ? (classifyStats.avg_confidence * 100).toFixed(1)
      : '0'
    const pendingReview = (feedbackStats as any)?.total_corrections || 0
    const modelCount = Array.isArray(models) ? models.length : 0

    return { totalClassified, avgConfidence, pendingReview, modelCount }
  }, [classifyStats, feedbackStats, models])

  const handleTestClassify = async () => {
    if (!testText.trim()) return
    await classifyText.mutateAsync({ text: testText })
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Classification', active: true }]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">Classification</h1>
        <p className="text-sm text-muted-foreground mt-1">Automatic and manual data classification</p>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {dsLoading ? (
            <>
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
            </>
          ) : (
            <>
              <StatCard label="Classified Columns" value={stats.totalClassified.toString()} icon={<Zap className="h-6 w-6" />} />
              <StatCard label="Pending Review" value={stats.pendingReview.toString()} icon={<TrendingUp className="h-6 w-6" />} />
              <StatCard label="Classification Models" value={stats.modelCount.toString()} />
              <StatCard label="Avg Confidence" value={`${stats.avgConfidence}%`} icon={<Settings className="h-6 w-6" />} />
            </>
          )}
        </div>

        {/* Quick Test */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Quick Classification Test</h3>
          <div className="flex gap-4">
            <input
              type="text"
              value={testText}
              onChange={(e) => setTestText(e.target.value)}
              placeholder="Enter text to classify (e.g., john.doe@email.com, 555-123-4567)"
              className="flex-1 px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            />
            <button
              onClick={handleTestClassify}
              disabled={classifyText.isPending || !testText.trim()}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Search className="h-4 w-4" />
              {classifyText.isPending ? 'Classifying...' : 'Classify'}
            </button>
          </div>
          {classifyText.data && (
            <div className="mt-4 p-4 rounded-lg bg-muted">
              <p className="text-sm font-medium text-foreground mb-2">
                Sensitivity: <span className="text-primary">{classifyText.data.sensitivity_label}</span>
              </p>
              {classifyText.data.entities.length > 0 ? (
                <div className="space-y-2">
                  {classifyText.data.entities.map((entity, i) => (
                    <div key={i} className="flex items-center gap-2 text-sm">
                      <span className="px-2 py-0.5 rounded bg-primary/10 text-primary">{entity.type}</span>
                      <span className="text-foreground">{entity.value}</span>
                      <span className="text-muted-foreground">({(entity.confidence * 100).toFixed(0)}%)</span>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No sensitive data detected</p>
              )}
            </div>
          )}
        </div>

        {/* Navigation Tabs */}
        <div className="rounded-lg border border-border bg-card p-6">
          <div className="flex gap-4 border-b border-border pb-4 mb-6">
            <Link href="/classification" className="text-foreground font-medium border-b-2 border-primary pb-2">
              Datasets
            </Link>
            <Link href="/classification/rules" className="text-muted-foreground hover:text-foreground transition-colors pb-2">
              Rules
            </Link>
            <Link href="/classification/models" className="text-muted-foreground hover:text-foreground transition-colors pb-2">
              Models
            </Link>
          </div>

          {/* Datasets table */}
          {dsLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : classifications.length > 0 ? (
            <DataTable columns={columns} data={classifications} />
          ) : (
            <EmptyState
              icon={<FileText className="h-12 w-12" />}
              title="No classifications yet"
              description="Connect a data source and run a scan to start classifying your data."
              action={
                <Link
                  href="/data-sources/new"
                  className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
                >
                  Add Data Source
                </Link>
              }
            />
          )}
        </div>
      </div>
    </div>
  )
}
