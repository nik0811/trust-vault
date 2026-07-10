'use client'

import { useMemo } from 'react'
import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { RefreshCw, FileText, Download } from 'lucide-react'
import { useAuditTrail, type AuditLog } from '@/hooks/use-audit'

const columns: Column<AuditLog>[] = [
  {
    id: 'action',
    header: 'Action',
    cell: (row) => (
      <span className="font-medium text-foreground">{row.action}</span>
    ),
    sortable: true,
  },
  {
    id: 'resource',
    header: 'Resource',
    cell: (row) => (
      <span className="text-sm text-muted-foreground">{row.resource}</span>
    ),
  },
  {
    id: 'user_id',
    header: 'User',
    cell: (row) => (
      <span className="text-sm font-mono text-foreground">{row.user_id?.slice(0, 8)}...</span>
    ),
  },
  {
    id: 'ip_address',
    header: 'IP Address',
    accessorKey: 'ip_address',
  },
  {
    id: 'created_at',
    header: 'Time',
    cell: (row) => new Date(row.created_at).toLocaleString(),
    sortable: true,
  },
]

export default function AuditPage() {
  const { data: auditLogs, isLoading, refetch } = useAuditTrail({ limit: 100 })

  const logsData = useMemo(() => {
    if (!auditLogs || !Array.isArray(auditLogs)) return []
    return auditLogs
  }, [auditLogs])

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs items={[{ label: 'Audit Trail', active: true }]} />
          <h1 className="text-3xl font-bold text-foreground mt-4">Audit Trail</h1>
          <p className="text-sm text-muted-foreground mt-1">Complete audit log of all system activities</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={() => refetch()}
            className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border bg-card text-foreground hover:bg-muted transition-colors"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh
          </button>
          <button className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border bg-card text-foreground hover:bg-muted transition-colors">
            <Download className="h-4 w-4" />
            Export
          </button>
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
        ) : logsData.length > 0 ? (
          <DataTable columns={columns} data={logsData} />
        ) : (
          <div className="text-center py-12">
            <FileText className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <p className="text-muted-foreground">No audit logs recorded yet</p>
            <p className="text-sm text-muted-foreground mt-1">
              Activities will be logged as users interact with the system
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
