'use client'

import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { StatCard } from '@/components/base/stat-card'
import { DataTable, type Column } from '@/components/base/data-table'
import { Globe, AlertTriangle, Loader2 } from 'lucide-react'
import { useDataMapGeography } from '@/hooks/use-datamap'
import { toast } from 'sonner'
import { useEffect } from 'react'

interface RegionData {
  id: string
  region: string
  location: string
  sources: number
  datasets: number
  volume: string
  cross_border: boolean
}

const columns: Column<RegionData>[] = [
  { id: 'region', header: 'Region', accessorKey: 'region' },
  { id: 'location', header: 'Location', accessorKey: 'location' },
  { id: 'sources', header: 'Sources', accessorKey: 'sources' },
  { id: 'datasets', header: 'Datasets', accessorKey: 'datasets' },
  { id: 'volume', header: 'Data Volume', accessorKey: 'volume' },
  {
    id: 'cross_border',
    header: 'Cross-Border Flows',
    cell: (row) =>
      row.cross_border ? (
        <span className="flex items-center gap-1.5 text-sm text-orange-600 dark:text-orange-400">
          <AlertTriangle className="h-3.5 w-3.5" />
          EU data transfers out
        </span>
      ) : (
        <span className="text-sm text-green-600 dark:text-green-400">None</span>
      ),
  },
]

export default function GeographyPage() {
  const { data, isLoading, error } = useDataMapGeography()

  useEffect(() => {
    if (error) toast.error('Failed to load geography data')
  }, [error])

  const regions: RegionData[] = data?.regions?.map((r: any, i: number) => ({
    id: String(i),
    region: r.name,
    location: r.location || r.name,
    sources: r.sources || 0,
    datasets: r.count || 0,
    volume: r.volume || '0 records',
    cross_border: r.cross_border || false,
  })) || []

  const crossBorderCount = regions.filter(r => r.cross_border).length

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Data Map', href: '/data-map' },
            { label: 'Geography', active: true },
          ]}
        />
        <h1 className="mt-4 text-3xl font-bold text-foreground">Geographic View</h1>
        <p className="mt-1 text-sm text-muted-foreground">Data center locations and cross-border transfer compliance</p>
      </div>

      <div className="space-y-8 p-8">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
          <StatCard label="Regions" value={String(regions.length)} icon={<Globe className="h-6 w-6" />} />
          <StatCard label="Cross-Border Flows" value={String(crossBorderCount)} />
          <StatCard label="GDPR Transfer Risks" value={String(crossBorderCount)} />
          <StatCard label="Data Residency Compliant" value={regions.length > 0 ? `${Math.round((1 - crossBorderCount / regions.length) * 100)}%` : '—'} />
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : regions.length === 0 ? (
          <div className="rounded-lg border border-border bg-card p-12 text-center text-muted-foreground">
            <Globe className="h-12 w-12 mx-auto mb-4 opacity-50" />
            <p>No geographic data available</p>
          </div>
        ) : (
          <>
            <div className="rounded-lg border border-border bg-card p-6">
              <h3 className="mb-6 text-lg font-semibold text-foreground">Data Volume by Region</h3>
              <div className="grid grid-cols-1 gap-6 md:grid-cols-4">
                {regions.map((region) => {
                  const size = Math.min(100, Math.max(40, region.datasets / 2.5))
                  return (
                    <div key={region.id} className="flex flex-col items-center gap-3 rounded-lg border border-border bg-background p-6">
                      <div
                        className="flex items-center justify-center rounded-full bg-primary/15 text-primary"
                        style={{ width: size, height: size }}
                        aria-hidden="true"
                      >
                        <Globe className="h-5 w-5" />
                      </div>
                      <div className="text-center">
                        <p className="font-semibold text-foreground">{region.region}</p>
                        <p className="text-xs text-muted-foreground">{region.volume}</p>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>

            <div className="rounded-lg border border-border bg-card p-6">
              <h3 className="mb-4 text-lg font-semibold text-foreground">Region Details</h3>
              <DataTable columns={columns} data={regions} />
            </div>

            {crossBorderCount > 0 && (
              <div className="flex items-start gap-3 rounded-lg border border-orange-500/20 bg-orange-500/10 p-4">
                <AlertTriangle className="mt-0.5 h-5 w-5 flex-shrink-0 text-orange-600 dark:text-orange-400" />
                <div>
                  <p className="text-sm font-medium text-foreground">GDPR Transfer Compliance</p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {crossBorderCount} cross-border data flows detected from EU regions. Verify Standard Contractual Clauses (SCCs) are in place.
                  </p>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
