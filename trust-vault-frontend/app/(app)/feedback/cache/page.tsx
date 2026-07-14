'use client'

import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { StatCard } from '@/components/base/stat-card'
import { Skeleton } from '@/components/base/skeleton'
import { Database, Zap, Clock, TrendingUp } from 'lucide-react'
import { useKnowledgeCache } from '@/hooks/use-advisor'

export default function KnowledgeCachePage() {
  const { data: cache, isLoading, error } = useKnowledgeCache()

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Feedback', href: '/feedback' },
            { label: 'Knowledge Cache', active: true },
          ]}
        />
        <h1 className="mt-4 text-3xl font-bold text-foreground">Knowledge Cache</h1>
        <p className="mt-1 text-sm text-muted-foreground">Cached classification lookups for instant results</p>
      </div>

      <div className="space-y-8 p-8">
        {isLoading ? (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
            <Skeleton className="h-24" />
            <Skeleton className="h-24" />
            <Skeleton className="h-24" />
            <Skeleton className="h-24" />
          </div>
        ) : error ? (
          <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-6 text-center">
            <p className="text-destructive">Failed to load cache data</p>
          </div>
        ) : cache?.cache_size === 0 && cache?.total_classifications === 0 ? (
          <>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
              <StatCard label="Entity Types" value="0" icon={<Database className="h-6 w-6" />} />
              <StatCard label="Classifications" value="0" icon={<Zap className="h-6 w-6" />} />
              <StatCard label="Avg Lookup Time" value="-" icon={<Clock className="h-6 w-6" />} />
              <StatCard label="Cache Status" value="Empty" icon={<TrendingUp className="h-6 w-6" />} />
            </div>
            <div className="rounded-lg border border-border bg-card p-12 text-center">
              <Database className="mx-auto h-12 w-12 text-muted-foreground/50" />
              <h3 className="mt-4 text-lg font-semibold text-foreground">No Cache Data Yet</h3>
              <p className="mt-2 text-sm text-muted-foreground">
                Classification cache will populate as you classify data sources and documents.
                Run a classification job to start building the knowledge cache.
              </p>
            </div>
          </>
        ) : (
          <>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
              <StatCard 
                label="Entity Types" 
                value={cache?.cache_size?.toLocaleString() || '0'} 
                icon={<Database className="h-6 w-6" />} 
              />
              <StatCard 
                label="Classifications" 
                value={cache?.total_classifications?.toLocaleString() || '0'} 
                icon={<Zap className="h-6 w-6" />} 
              />
              <StatCard 
                label="Avg Lookup Time" 
                value="<1ms" 
                icon={<Clock className="h-6 w-6" />} 
              />
              <StatCard 
                label="Last Updated" 
                value={cache?.last_updated ? new Date(cache.last_updated).toLocaleTimeString() : '-'} 
                icon={<TrendingUp className="h-6 w-6" />} 
              />
            </div>

            <div className="rounded-lg border border-border bg-card p-6">
              <h3 className="mb-4 text-lg font-semibold text-foreground">Cache Information</h3>
              <div className="space-y-4">
                <div className="flex items-center justify-between border-b border-border pb-4">
                  <span className="text-sm text-muted-foreground">Total Entity Types Cached</span>
                  <span className="font-semibold text-foreground">{cache?.cache_size || 0}</span>
                </div>
                <div className="flex items-center justify-between border-b border-border pb-4">
                  <span className="text-sm text-muted-foreground">Total Classifications</span>
                  <span className="font-semibold text-foreground">{cache?.total_classifications?.toLocaleString() || 0}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Cache Status</span>
                  <span className="inline-flex items-center rounded-full bg-green-500/10 px-2.5 py-0.5 text-xs font-medium text-green-500">
                    Active
                  </span>
                </div>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
