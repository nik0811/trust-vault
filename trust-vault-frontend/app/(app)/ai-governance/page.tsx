'use client'

import { StatCard } from '@/components/base/stat-card'
import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Brain, Loader2 } from 'lucide-react'
import { useAIGovPolicies } from '@/hooks/use-datamap'
import { toast } from 'sonner'
import { useEffect } from 'react'

interface AIPolicy {
  id: string
  name: string
  description: string
  enabled: boolean
}

const columns: Column<AIPolicy>[] = [
  { id: 'name', header: 'Policy', accessorKey: 'name' },
  { id: 'description', header: 'Description', accessorKey: 'description' },
  { id: 'enabled', header: 'Status', cell: (row) => row.enabled ? 'Enabled' : 'Disabled' },
]

export default function AIGovernancePage() {
  const { data: policiesRaw, isLoading, error } = useAIGovPolicies()

  useEffect(() => {
    if (error) toast.error('Failed to load AI governance policies')
  }, [error])

  const policies: AIPolicy[] = (policiesRaw || []).map((p: any) => ({
    id: p.id,
    name: p.name,
    description: p.description || '',
    enabled: p.enabled ?? true,
  }))

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'AI Governance', active: true }]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">AI Governance</h1>
        <p className="text-sm text-muted-foreground mt-1">Manage AI model governance and policies</p>
      </div>

      <div className="p-8 space-y-8">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <StatCard label="AI Models" value="—" icon={<Brain className="h-6 w-6" />} />
          <StatCard label="Governance Policies" value={String(policies.length)} />
          <StatCard label="Models Approved" value="—" />
          <StatCard label="Policy Compliance" value="—" />
        </div>

        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">AI Governance Policies</h3>
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : policies.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              <Brain className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>No AI governance policies configured</p>
            </div>
          ) : (
            <DataTable columns={columns} data={policies} />
          )}
        </div>
      </div>
    </div>
  )
}
