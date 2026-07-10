'use client'

import Link from 'next/link'
import { useState, useEffect } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { StatCard } from '@/components/base/stat-card'
import { Search, Globe, PieChart, Database, Cloud, HardDrive, Server, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useDataMapSources } from '@/hooks/use-datamap'
import { toast } from 'sonner'

type Sensitivity = 'public' | 'internal' | 'confidential' | 'restricted'

interface MapSource {
  id: string
  name: string
  type: string
  datasets: number
  records: string
  sensitivity: Sensitivity
  region: string
}

const sensitivityStyles: Record<Sensitivity, { dot: string; label: string; badge: string }> = {
  public: { dot: 'bg-green-500', label: 'Public', badge: 'bg-green-500/10 text-green-600 dark:text-green-400' },
  internal: { dot: 'bg-yellow-500', label: 'Internal', badge: 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400' },
  confidential: { dot: 'bg-orange-500', label: 'Confidential', badge: 'bg-orange-500/10 text-orange-600 dark:text-orange-400' },
  restricted: { dot: 'bg-red-500', label: 'Restricted', badge: 'bg-red-500/10 text-red-600 dark:text-red-400' },
}

const typeIcons: Record<string, typeof Database> = {
  PostgreSQL: Database,
  MySQL: Database,
  Snowflake: Cloud,
  BigQuery: Cloud,
  S3: HardDrive,
  Files: Server,
}

const filters: Array<{ id: Sensitivity | 'all'; label: string }> = [
  { id: 'all', label: 'All' },
  { id: 'restricted', label: 'Restricted' },
  { id: 'confidential', label: 'Confidential' },
  { id: 'internal', label: 'Internal' },
  { id: 'public', label: 'Public' },
]

export default function DataMapPage() {
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<Sensitivity | 'all'>('all')
  const { data: sourcesRaw, isLoading, error } = useDataMapSources()

  useEffect(() => {
    if (error) toast.error('Failed to load data sources')
  }, [error])

  const sources: MapSource[] = (sourcesRaw || []).map((s: any) => ({
    id: s.id,
    name: s.name,
    type: s.type || 'Database',
    datasets: s.datasets || s.dataset_count || 0,
    records: s.records || s.record_count || '0',
    sensitivity: s.sensitivity || 'internal',
    region: s.region || 'unknown',
  }))

  const filtered = sources.filter(
    (s) =>
      (filter === 'all' || s.sensitivity === filter) &&
      s.name.toLowerCase().includes(search.toLowerCase()),
  )

  const totalDatasets = sources.reduce((a, b) => a + b.datasets, 0)
  const totalRecords = sources.reduce((a, b) => {
    const num = parseFloat(b.records.replace(/[^0-9.]/g, ''))
    return a + (isNaN(num) ? 0 : num)
  }, 0)

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Data Map', active: true }]} />
        <div className="mt-4 flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-3xl font-bold text-foreground">Data Map</h1>
            <p className="mt-1 text-sm text-muted-foreground">Where is all your data? The foundation of governance.</p>
          </div>
          <div className="flex gap-2">
            <Link href="/data-map/geography" className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted">
              <Globe className="h-4 w-4" />
              Geographic View
            </Link>
            <Link href="/data-map/coverage" className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted">
              <PieChart className="h-4 w-4" />
              Coverage
            </Link>
          </div>
        </div>
      </div>

      <div className="space-y-8 p-8">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
          <StatCard label="Total Sources" value={String(sources.length)} />
          <StatCard label="Total Datasets" value={String(totalDatasets)} />
          <StatCard label="Total Records" value={`${totalRecords.toFixed(1)}M`} />
          <StatCard label="Governed" value={sources.length > 0 ? '84%' : '—'} />
        </div>

        <div className="flex flex-col gap-4 md:flex-row md:items-center">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <input
              type="search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Find any dataset across the entire estate..."
              className="w-full rounded-lg border border-border bg-card py-2 pl-10 pr-4 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
          <div className="flex flex-wrap gap-2" role="group" aria-label="Filter by sensitivity">
            {filters.map((f) => (
              <button
                key={f.id}
                onClick={() => setFilter(f.id)}
                className={cn(
                  'rounded-full px-3 py-1.5 text-xs font-medium transition-colors',
                  filter === f.id
                    ? 'bg-primary text-primary-foreground'
                    : 'border border-border text-muted-foreground hover:bg-muted',
                )}
              >
                {f.label}
              </button>
            ))}
          </div>
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="rounded-lg border border-border bg-card p-12 text-center text-muted-foreground">
            <Database className="h-12 w-12 mx-auto mb-4 opacity-50" />
            <p>{sources.length === 0 ? 'No data sources connected' : 'No sources match your filter'}</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {filtered.map((source) => {
              const Icon = typeIcons[source.type] || Database
              const style = sensitivityStyles[source.sensitivity] || sensitivityStyles.internal
              return (
                <div
                  key={source.id}
                  className="group rounded-lg border border-border bg-card p-5 transition-all hover:border-primary/50 hover:shadow-md"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
                        <Icon className="h-5 w-5 text-foreground" />
                      </div>
                      <div>
                        <p className="font-semibold text-foreground">{source.name}</p>
                        <p className="text-xs text-muted-foreground">{source.type} · {source.region}</p>
                      </div>
                    </div>
                    <span className={cn('rounded-full px-2 py-0.5 text-[11px] font-medium', style.badge)}>
                      {style.label}
                    </span>
                  </div>
                  <div className="mt-4 flex items-center justify-between border-t border-border pt-4">
                    <div>
                      <p className="text-lg font-bold text-foreground">{source.datasets}</p>
                      <p className="text-xs text-muted-foreground">Datasets</p>
                    </div>
                    <div>
                      <p className="text-lg font-bold text-foreground">{source.records}</p>
                      <p className="text-xs text-muted-foreground">Records</p>
                    </div>
                    <Link
                      href={`/data-sources/${source.id}`}
                      className="text-sm font-medium text-primary opacity-0 transition-opacity group-hover:opacity-100"
                    >
                      Explore →
                    </Link>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
