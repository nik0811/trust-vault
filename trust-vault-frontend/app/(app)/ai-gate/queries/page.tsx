'use client'

import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { StatusIndicator } from '@/components/base/status-badge'
import { RefreshCw } from 'lucide-react'
import { useGateQueries } from '@/hooks/use-gate'

interface GateQuery {
  id: string
  query: string
  decision: string
  policies_applied: string[]
  latency_ms: number
  created_at: string
}

const columns: Column<GateQuery>[] = [
  {
    id: 'query',
    header: 'Query',
    cell: (row) => (
      <span className="text-foreground truncate max-w-md block">{row.query || 'N/A'}</span>
    ),
  },
  {
    id: 'decision',
    header: 'Decision',
    cell: (row) => (
      <StatusIndicator
        status={row.decision === 'allow' ? 'success' : 'error'}
        label={row.decision}
      />
    ),
  },
  {
    id: 'policies',
    header: 'Policies Applied',
    cell: (row) => (
      <span className="text-sm text-muted-foreground">
        {row.policies_applied?.length || 0} policies
      </span>
    ),
  },
  {
    id: 'latency',
    header: 'Latency',
    cell: (row) => <span className="text-sm text-muted-foreground">{row.latency_ms}ms</span>,
    sortable: true,
  },
  {
    id: 'created_at',
    header: 'Time',
    cell: (row) => new Date(row.created_at).toLocaleString(),
    sortable: true,
  },
]

export default function AIGateQueriesPage() {
  const { data: queries, isLoading, refetch } = useGateQueries(100)

  const queriesData = Array.isArray(queries) ? queries : []

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs
            items={[
              { label: 'AI Gate', href: '/ai-gate' },
              { label: 'Query History', active: true },
            ]}
          />
          <h1 className="text-3xl font-bold text-foreground mt-4">Query History</h1>
          <p className="text-sm text-muted-foreground mt-1">View all AI Gate queries and decisions</p>
        </div>
        <button
          onClick={() => refetch()}
          className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border bg-card text-foreground hover:bg-muted transition-colors"
        >
          <RefreshCw className="h-4 w-4" />
          Refresh
        </button>
      </div>

      {/* Content */}
      <div className="p-8">
        {isLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : queriesData.length > 0 ? (
          <DataTable columns={columns} data={queriesData} />
        ) : (
          <div className="text-center py-12">
            <p className="text-muted-foreground">No queries recorded yet</p>
            <p className="text-sm text-muted-foreground mt-1">
              Queries will appear here when AI systems interact with the gate
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
