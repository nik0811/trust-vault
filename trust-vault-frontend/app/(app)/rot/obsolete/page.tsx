'use client'

import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { Clock, Archive } from 'lucide-react'
import { useROTDatasets, useRemediateROT } from '@/hooks/use-advisor'

export default function ObsoletePage() {
  const { data: datasets, isLoading } = useROTDatasets()
  const remediate = useRemediateROT()

  const obsoleteData = Array.isArray(datasets) 
    ? datasets.filter((d: any) => d.category === 'obsolete') 
    : []

  const handleArchive = async (ids: string[]) => {
    await remediate.mutateAsync({ dataset_ids: ids, action: 'archive' })
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'ROT Data', href: '/rot' },
            { label: 'Obsolete', active: true },
          ]}
        />
        <h1 className="text-3xl font-bold text-foreground mt-4">Obsolete Data</h1>
        <p className="text-sm text-muted-foreground mt-1">Data that hasn&apos;t been accessed in a long time</p>
      </div>

      {/* Content */}
      <div className="p-8">
        {isLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
          </div>
        ) : obsoleteData.length > 0 ? (
          <div className="space-y-4">
            {obsoleteData.map((item: any) => (
              <div key={item.id} className="rounded-lg border border-border bg-card p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4">
                    <Clock className="h-5 w-5 text-yellow-500" />
                    <div>
                      <p className="font-medium text-foreground">{item.dataset_id}</p>
                      <p className="text-sm text-muted-foreground">
                        Last accessed: {new Date(item.last_access).toLocaleDateString()}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className="text-sm text-muted-foreground">
                      {(item.size_bytes / 1024 / 1024).toFixed(2)} MB
                    </span>
                    <button
                      onClick={() => handleArchive([item.dataset_id])}
                      disabled={remediate.isPending}
                      className="flex items-center gap-2 px-3 py-1 rounded-lg border border-border text-foreground hover:bg-muted transition-colors disabled:opacity-50"
                    >
                      <Archive className="h-4 w-4" />
                      Archive
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <Clock className="h-12 w-12 mx-auto text-green-500 mb-4" />
            <p className="text-foreground font-medium">No obsolete data found</p>
            <p className="text-sm text-muted-foreground mt-1">
              All your data is being actively used
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
