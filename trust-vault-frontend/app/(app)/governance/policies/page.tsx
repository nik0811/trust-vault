'use client'

import Link from 'next/link'
import { DataTable, type Column } from '@/components/base/data-table'
import { StatusIndicator } from '@/components/base/status-badge'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { EmptyState } from '@/components/base/empty-state'
import { Plus, Shield, RefreshCw } from 'lucide-react'
import { usePolicies, useDeletePolicy, type Policy } from '@/hooks/use-policies'
import { useRouter } from 'next/navigation'

const columns: Column<Policy>[] = [
  {
    id: 'name',
    header: 'Name',
    cell: (row) => (
      <Link href={`/governance/policies/${row.id}`} className="text-primary hover:underline font-medium">
        {row.name}
      </Link>
    ),
    sortable: true,
  },
  {
    id: 'type',
    header: 'Type',
    cell: (row) => (
      <span className="px-2 py-0.5 rounded bg-muted text-foreground text-sm capitalize">{row.type}</span>
    ),
    sortable: true,
  },
  {
    id: 'active',
    header: 'Status',
    cell: (row) => (
      <StatusIndicator status={row.active ? 'success' : 'inactive'} label={row.active ? 'Active' : 'Inactive'} />
    ),
  },
  {
    id: 'priority',
    header: 'Priority',
    accessorKey: 'priority',
    sortable: true,
  },
  {
    id: 'updated_at',
    header: 'Updated',
    cell: (row) => new Date(row.updated_at).toLocaleDateString(),
    sortable: true,
  },
]

export default function PoliciesPage() {
  const router = useRouter()
  const { data: policies, isLoading, error, refetch } = usePolicies()
  const deletePolicy = useDeletePolicy()

  const handleRowClick = (row: Policy) => {
    router.push(`/governance/policies/${row.id}`)
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs
            items={[
              { label: 'Governance', href: '/governance' },
              { label: 'Policies', active: true },
            ]}
          />
          <h1 className="text-3xl font-bold text-foreground mt-4">Policies</h1>
          <p className="text-sm text-muted-foreground mt-1">Manage governance policies for data access and protection</p>
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
            href="/governance/policies/new"
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
          >
            <Plus className="h-5 w-5" />
            Create Policy
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
            <p className="text-destructive">Failed to load policies</p>
            <button onClick={() => refetch()} className="mt-4 text-primary hover:underline">
              Try again
            </button>
          </div>
        ) : Array.isArray(policies) && policies.length > 0 ? (
          <DataTable columns={columns} data={policies} onRowClick={handleRowClick} />
        ) : (
          <EmptyState
            icon={<Shield className="h-12 w-12" />}
            title="No policies"
            description="Create your first governance policy to start protecting your data."
            action={
              <Link
                href="/governance/policies/new"
                className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
              >
                <Plus className="h-5 w-5" />
                Create Policy
              </Link>
            }
          />
        )}
      </div>
    </div>
  )
}
