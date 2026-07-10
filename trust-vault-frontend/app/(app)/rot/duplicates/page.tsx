'use client'

import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { Copy } from 'lucide-react'
import { useROTDuplicates, useRemediateROT } from '@/hooks/use-advisor'

export default function DuplicatesPage() {
  const { data: duplicates, isLoading } = useROTDuplicates()
  const remediate = useRemediateROT()

  const duplicatesData = Array.isArray(duplicates) ? duplicates : []

  const handleDeduplicate = async (ids: string[]) => {
    await remediate.mutateAsync({ dataset_ids: ids, action: 'deduplicate' })
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'ROT Data', href: '/rot' },
            { label: 'Duplicates', active: true },
          ]}
        />
        <h1 className="text-3xl font-bold text-foreground mt-4">Duplicate Data</h1>
        <p className="text-sm text-muted-foreground mt-1">Identify and deduplicate redundant data</p>
      </div>

      {/* Content */}
      <div className="p-8">
        {isLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-24 w-full" />
            <Skeleton className="h-24 w-full" />
          </div>
        ) : duplicatesData.length > 0 ? (
          <div className="space-y-4">
            {duplicatesData.map((group: any, i: number) => (
              <div key={i} className="rounded-lg border border-border bg-card p-6">
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <Copy className="h-5 w-5 text-blue-500" />
                    <span className="font-medium text-foreground">
                      {group.count || group.datasets?.length || 0} duplicates found
                    </span>
                  </div>
                  <button
                    onClick={() => handleDeduplicate(group.dataset_ids || [])}
                    disabled={remediate.isPending}
                    className="px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
                  >
                    Deduplicate
                  </button>
                </div>
                <pre className="p-4 rounded-lg bg-muted text-sm overflow-auto">
                  {JSON.stringify(group, null, 2)}
                </pre>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <Copy className="h-12 w-12 mx-auto text-green-500 mb-4" />
            <p className="text-foreground font-medium">No duplicates found</p>
            <p className="text-sm text-muted-foreground mt-1">
              Your data appears to be unique
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
