'use client'

import { useMemo } from 'react'
import { StatCard } from '@/components/base/stat-card'
import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import Link from 'next/link'
import { Trash2, HardDrive, Copy, Clock, Play } from 'lucide-react'
import { useROTSummary, useROTDatasets, useTriggerROTScan, type ROTDataset } from '@/hooks/use-advisor'

const columns: Column<ROTDataset>[] = [
  {
    id: 'dataset_id',
    header: 'Dataset',
    cell: (row) => (
      <span className="font-medium text-foreground">{row.dataset_id}</span>
    ),
    sortable: true,
  },
  {
    id: 'category',
    header: 'Category',
    cell: (row) => (
      <span className={`px-2 py-0.5 rounded text-sm ${
        row.category === 'redundant' ? 'bg-blue-500/10 text-blue-600' :
        row.category === 'obsolete' ? 'bg-yellow-500/10 text-yellow-600' :
        'bg-gray-500/10 text-gray-600'
      }`}>
        {row.category}
      </span>
    ),
    sortable: true,
  },
  {
    id: 'score',
    header: 'ROT Score',
    cell: (row) => (
      <span className="text-sm text-foreground">{(row.score * 100).toFixed(0)}%</span>
    ),
    sortable: true,
  },
  {
    id: 'size_bytes',
    header: 'Size',
    cell: (row) => (
      <span className="text-sm text-muted-foreground">
        {(row.size_bytes / 1024 / 1024).toFixed(2)} MB
      </span>
    ),
    sortable: true,
  },
  {
    id: 'last_access',
    header: 'Last Access',
    cell: (row) => new Date(row.last_access).toLocaleDateString(),
    sortable: true,
  },
]

export default function ROTPage() {
  const { data: summary, isLoading: summaryLoading } = useROTSummary()
  const { data: datasets, isLoading: datasetsLoading } = useROTDatasets()
  const triggerScan = useTriggerROTScan()

  const datasetsData = Array.isArray(datasets) ? datasets : []

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
          onClick={() => triggerScan.mutate()}
          disabled={triggerScan.isPending}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
        >
          <Play className="h-4 w-4" />
          {triggerScan.isPending ? 'Scanning...' : 'Run Scan'}
        </button>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
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
            <p className="text-sm text-muted-foreground mt-1">
              Find and deduplicate redundant data
            </p>
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
            <p className="text-sm text-muted-foreground mt-1">
              Low-value data that can be safely removed
            </p>
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
            <DataTable columns={columns} data={datasetsData} />
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
    </div>
  )
}
