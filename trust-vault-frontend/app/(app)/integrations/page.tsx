'use client'

import Link from 'next/link'
import { DataTable, type Column } from '@/components/base/data-table'
import { StatusIndicator } from '@/components/base/status-badge'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { EmptyState } from '@/components/base/empty-state'
import { Plus, Plug, RefreshCw } from 'lucide-react'
import { useIntegrations, type Integration } from '@/hooks/use-jobs'
import { useRouter } from 'next/navigation'

const columns: Column<Integration>[] = [
  {
    id: 'name',
    header: 'Name',
    cell: (row) => (
      <Link href={`/integrations/${row.id}`} className="text-primary hover:underline font-medium">
        {row.name}
      </Link>
    ),
    sortable: true,
  },
  {
    id: 'type',
    header: 'Type',
    cell: (row) => (
      <span className="px-2 py-0.5 rounded bg-muted text-foreground text-sm capitalize">
        {row.type.replace('_', ' ')}
      </span>
    ),
    sortable: true,
  },
  {
    id: 'provider',
    header: 'Provider',
    accessorKey: 'provider',
  },
  {
    id: 'status',
    header: 'Status',
    cell: (row) => (
      <StatusIndicator
        status={row.status === 'connected' ? 'success' : row.status === 'syncing' ? 'pending' : 'error'}
        label={row.status}
      />
    ),
  },
  {
    id: 'last_sync',
    header: 'Last Sync',
    cell: (row) => row.last_sync && row.last_sync !== '0001-01-01T00:00:00Z'
      ? new Date(row.last_sync).toLocaleString()
      : 'Never',
    sortable: true,
  },
]

export default function IntegrationsPage() {
  const router = useRouter()
  const { data: integrations, isLoading, refetch } = useIntegrations()

  const integrationsData = Array.isArray(integrations) ? integrations : []

  const handleRowClick = (row: Integration) => {
    router.push(`/integrations/${row.id}`)
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs items={[{ label: 'Integrations', active: true }]} />
          <h1 className="text-3xl font-bold text-foreground mt-4">Integrations</h1>
          <p className="text-sm text-muted-foreground mt-1">Connect TrustVault with external systems</p>
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
            href="/integrations/new"
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
          >
            <Plus className="h-5 w-5" />
            Add Integration
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
        ) : integrationsData.length > 0 ? (
          <DataTable columns={columns} data={integrationsData} onRowClick={handleRowClick} />
        ) : (
          <EmptyState
            icon={<Plug className="h-12 w-12" />}
            title="No integrations"
            description="Connect TrustVault with DLP, privacy platforms, data catalogs, and more."
            action={
              <Link
                href="/integrations/new"
                className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
              >
                <Plus className="h-5 w-5" />
                Add Integration
              </Link>
            }
          />
        )}
      </div>
    </div>
  )
}
