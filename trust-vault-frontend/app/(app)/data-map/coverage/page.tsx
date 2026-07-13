'use client'

import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { StatCard } from '@/components/base/stat-card'
import { EyeOff, ShieldAlert, PieChart, Loader2 } from 'lucide-react'
import { useDataMapCoverage, useDarkData, useShadowIT } from '@/hooks/use-datamap'
import { toast } from 'sonner'
import { useEffect } from 'react'

interface SourceCoverage {
  name: string
  scanned: number
  classified: number
}

interface DarkDataItem {
  name: string
  reason: string
  risk: string
}

interface ShadowITItem {
  name: string
  reason: string
  risk: string
}

function ProgressBar({ value, colorClass }: { value: number; colorClass: string }) {
  return (
    <div className="h-2 w-full overflow-hidden rounded-full bg-muted" role="progressbar" aria-valuenow={value} aria-valuemin={0} aria-valuemax={100}>
      <div className={`h-full rounded-full ${colorClass}`} style={{ width: `${value}%` }} />
    </div>
  )
}

export default function CoveragePage() {
  const { data: coverageData, isLoading: coverageLoading, error: coverageError } = useDataMapCoverage()
  const { data: darkDataRaw, isLoading: darkLoading, error: darkError } = useDarkData()
  const { data: shadowITRaw, isLoading: shadowLoading, error: shadowError } = useShadowIT()

  useEffect(() => {
    if (coverageError) toast.error('Failed to load coverage data')
    if (darkError) toast.error('Failed to load dark data')
    if (shadowError) toast.error('Failed to load shadow IT data')
  }, [coverageError, darkError, shadowError])

  const isLoading = coverageLoading || darkLoading || shadowLoading
  const coverage: SourceCoverage[] = coverageData?.sources || []
  const darkData: DarkDataItem[] = darkDataRaw || []
  const shadowIT: ShadowITItem[] = shadowITRaw || []

  const avgScanned = coverage.length > 0 ? Math.round(coverage.reduce((a, b) => a + b.scanned, 0) / coverage.length) : 0
  const avgClassified = coverage.length > 0 ? Math.round(coverage.reduce((a, b) => a + b.classified, 0) / coverage.length) : 0

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Data Map', href: '/data-map' },
            { label: 'Coverage', active: true },
          ]}
        />
        <h1 className="mt-4 text-3xl font-bold text-foreground">Coverage Dashboard</h1>
        <p className="mt-1 text-sm text-muted-foreground">Classification coverage, dark data, and shadow IT detection</p>
      </div>

      <div className="space-y-8 p-8">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
          <StatCard label="Estate Classified" value={`${avgClassified}%`} icon={<PieChart className="h-6 w-6" />} />
          <StatCard label="Scan Coverage" value={`${avgScanned}%`} />
          <StatCard label="Dark Data Sources" value={String(darkData.length)} icon={<EyeOff className="h-6 w-6" />} />
          <StatCard label="Shadow IT Findings" value={String(shadowIT.length)} icon={<ShieldAlert className="h-6 w-6" />} />
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : (
          <>
            <div className="rounded-lg border border-border bg-card p-6">
              <h3 className="mb-6 text-lg font-semibold text-foreground">Coverage by Source</h3>
              {coverage.length === 0 ? (
                <p className="text-center py-8 text-muted-foreground">No coverage data available</p>
              ) : (
                <div className="space-y-6">
                  {coverage.map((source) => (
                    <div key={source.name}>
                      <div className="mb-2 flex items-center justify-between">
                        <p className="text-sm font-medium text-foreground">{source.name}</p>
                        <p className="text-xs text-muted-foreground">
                          Scanned {source.scanned}% · Classified {source.classified}%
                        </p>
                      </div>
                      <div className="flex flex-col gap-1.5">
                        <ProgressBar value={source.scanned} colorClass="bg-primary" />
                        <ProgressBar value={source.classified} colorClass="bg-green-500" />
                      </div>
                    </div>
                  ))}
                </div>
              )}
              <div className="mt-6 flex items-center gap-6 border-t border-border pt-4 text-xs text-muted-foreground">
                <span className="flex items-center gap-2">
                  <span className="h-2 w-4 rounded-full bg-primary" aria-hidden="true" /> Scan coverage
                </span>
                <span className="flex items-center gap-2">
                  <span className="h-2 w-4 rounded-full bg-green-500" aria-hidden="true" /> Classification coverage
                </span>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-8 lg:grid-cols-2">
              <div className="rounded-lg border border-border bg-card p-6">
                <div className="mb-4 flex items-center gap-2">
                  <EyeOff className="h-5 w-5 text-orange-600 dark:text-orange-400" />
                  <h3 className="text-lg font-semibold text-foreground">Dark Data</h3>
                </div>
                <p className="mb-4 text-sm text-muted-foreground">Datasets that exist but are not governed</p>
                {darkData.length === 0 ? (
                  <p className="text-center py-4 text-sm text-muted-foreground">No dark data detected</p>
                ) : (
                  <div className="space-y-3">
                    {darkData.map((item) => (
                      <div key={item.name} className="flex items-start justify-between rounded-lg border border-border bg-background p-4">
                        <div>
                          <p className="text-sm font-medium text-foreground">{item.name}</p>
                          <p className="mt-1 text-xs text-muted-foreground">{item.reason}</p>
                        </div>
                        <span className="rounded-full bg-orange-500/10 px-2 py-0.5 text-[11px] font-medium text-orange-600 dark:text-orange-400">
                          {item.risk}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="rounded-lg border border-border bg-card p-6">
                <div className="mb-4 flex items-center gap-2">
                  <ShieldAlert className="h-5 w-5 text-red-600 dark:text-red-400" />
                  <h3 className="text-lg font-semibold text-foreground">Shadow IT</h3>
                </div>
                <p className="mb-4 text-sm text-muted-foreground">Data in unapproved locations</p>
                {shadowIT.length === 0 ? (
                  <p className="text-center py-4 text-sm text-muted-foreground">No shadow IT detected</p>
                ) : (
                  <div className="space-y-3">
                    {shadowIT.map((item) => (
                      <div key={item.name} className="flex items-start justify-between rounded-lg border border-border bg-background p-4">
                        <div>
                          <p className="text-sm font-medium text-foreground">{item.name}</p>
                          <p className="mt-1 text-xs text-muted-foreground">{item.reason}</p>
                        </div>
                        <span className="rounded-full bg-red-500/10 px-2 py-0.5 text-[11px] font-medium text-red-600 dark:text-red-400">
                          {item.risk}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
