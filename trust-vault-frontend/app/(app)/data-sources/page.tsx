'use client'

import Link from 'next/link'
import { DataTable, type Column } from '@/components/base/data-table'
import { StatusIndicator } from '@/components/base/status-badge'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { EmptyState } from '@/components/base/empty-state'
import { Plus, Database, RefreshCw } from 'lucide-react'
import { useDataSources, useTriggerScan, type DataSource } from '@/hooks/use-datasources'
import { useRouter } from 'next/navigation'

const columns: Column<DataSource>[] = [
  {
    id: 'name',
    header: 'Name',
    cell: (row) => (
      <Link href={`/data-sources/${row.id}`} className="text-primary hover:underline font-medium">
        {row.name}
      </Link>
    ),
    sortable: true,
  },
  {
    id: 'type',
    header: 'Type',
    cell: (row) => (
      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-muted text-foreground capitalize">
        {row.type}
      </span>
    ),
    sortable: true,
  },
  {
    id: 'status',
    header: 'Status',
    cell: (row) => (
      <StatusIndicator
        status={row.status === 'connected' || row.status === 'active' ? 'success' : row.status === 'scanning' ? 'pending' : row.status === 'inactive' ? 'warning' : 'error'}
        label={row.status}
      />
    ),
  },
  {
    id: 'last_scan',
    header: 'Last Scan',
    cell: (row) => row.last_scan && row.last_scan !== '0001-01-01T00:00:00Z' 
      ? new Date(row.last_scan).toLocaleString() 
      : 'Never',
    sortable: true,
  },
  {
    id: 'created_at',
    header: 'Created',
    cell: (row) => new Date(row.created_at).toLocaleDateString(),
    sortable: true,
  },
]

export default function DataSourcesPage() {
  const router = useRouter()
  const { data: dataSources, isLoading, error, refetch } = useDataSources()
  const triggerScan = useTriggerScan()

  const handleRowClick = (row: DataSource) => {
    router.push(`/data-sources/${row.id}`)
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs items={[{ label: 'Data Sources', active: true }]} />
          <h1 className="text-3xl font-bold text-foreground mt-4">Data Sources</h1>
          <p className="text-sm text-muted-foreground mt-1">Manage and monitor your connected data sources</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={() => refetch()}
            className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border bg-card text-foreground hover:bg-muted transition-colors"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh
          </button>
          <Link
            href="/data-sources/new"
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
          >
            <Plus className="h-5 w-5" />
            Add Source
          </Link>
        </div>
      </div>

      {/* Content */}
      <div className="p-8">
        {isLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : error ? (
          <div className="text-center py-12">
            <p className="text-destructive">Failed to load data sources</p>
            <button onClick={() => refetch()} className="mt-4 text-primary hover:underline">
              Try again
            </button>
          </div>
        ) : Array.isArray(dataSources) && dataSources.length > 0 ? (
          <DataTable 
            columns={columns} 
            data={dataSources} 
            onRowClick={handleRowClick}
          />
        ) : (
          <EmptyState
            icon={<Database className="h-12 w-12" />}
            title="No data sources"
            description="Connect your first data source to start discovering and classifying your data."
            action={
              <Link
                href="/data-sources/new"
                className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
              >
                <Plus className="h-5 w-5" />
                Add Data Source
              </Link>
            }
          />
        )}
      </div>
    </div>
  )
}
