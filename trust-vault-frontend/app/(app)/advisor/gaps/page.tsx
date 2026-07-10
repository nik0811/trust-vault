'use client'

import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { StatusIndicator } from '@/components/base/status-badge'
import { AlertTriangle } from 'lucide-react'
import { useComplianceGaps, type ComplianceGap } from '@/hooks/use-advisor'

const columns: Column<ComplianceGap>[] = [
  {
    id: 'regulation',
    header: 'Regulation',
    cell: (row) => (
      <span className="font-medium text-foreground">{row.regulation}</span>
    ),
    sortable: true,
  },
  {
    id: 'requirement',
    header: 'Requirement',
    cell: (row) => (
      <span className="text-sm text-foreground">{row.requirement}</span>
    ),
  },
  {
    id: 'status',
    header: 'Status',
    cell: (row) => (
      <StatusIndicator
        status={row.status === 'resolved' ? 'success' : row.status === 'in_progress' ? 'pending' : 'error'}
        label={row.status}
      />
    ),
  },
  {
    id: 'remediation',
    header: 'Remediation',
    cell: (row) => (
      <span className="text-sm text-muted-foreground">{row.remediation}</span>
    ),
  },
]

export default function GapsPage() {
  const { data: gaps, isLoading } = useComplianceGaps()

  const gapsData = Array.isArray(gaps) ? gaps : []

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Compliance Advisor', href: '/advisor' },
            { label: 'Gaps', active: true },
          ]}
        />
        <h1 className="text-3xl font-bold text-foreground mt-4">Compliance Gaps</h1>
        <p className="text-sm text-muted-foreground mt-1">Identified gaps in regulatory compliance</p>
      </div>

      {/* Content */}
      <div className="p-8">
        {isLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : gapsData.length > 0 ? (
          <DataTable columns={columns} data={gapsData} />
        ) : (
          <div className="text-center py-12">
            <AlertTriangle className="h-12 w-12 mx-auto text-green-500 mb-4" />
            <p className="text-foreground font-medium">No compliance gaps detected</p>
            <p className="text-sm text-muted-foreground mt-1">
              Your organization is meeting all identified compliance requirements
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
