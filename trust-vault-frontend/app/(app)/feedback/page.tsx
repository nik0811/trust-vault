'use client'

import Link from 'next/link'
import { useEffect } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { StatCard } from '@/components/base/stat-card'
import { DataTable, type Column } from '@/components/base/data-table'
import { MessageSquare, Database, Braces, Loader2 } from 'lucide-react'
import { useCorrections, useCorrectionTrend, useFeedbackStats, type Correction } from '@/hooks/use-advisor'
import { toast } from 'sonner'

const columns: Column<Correction>[] = [
  { id: 'text', header: 'Text Sample', cell: (row) => <span className="font-mono text-sm">{row.text || '—'}</span> },
  { id: 'from', header: 'Was Classified', cell: (row) => <span className="text-red-600 dark:text-red-400">{row.from || 'Unknown'}</span> },
  { id: 'to', header: 'Corrected To', cell: (row) => <span className="text-green-600 dark:text-green-400">{row.to}</span> },
  { id: 'user', header: 'By', accessorKey: 'user' },
  { id: 'created_at', header: 'When', cell: (row) => {
    const date = new Date(row.created_at)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const hours = Math.floor(diff / (1000 * 60 * 60))
    if (hours < 24) return `${hours}h ago`
    const days = Math.floor(hours / 24)
    return `${days}d ago`
  }},
]

export default function FeedbackPage() {
  const { data: correctionsRaw, isLoading: correctionsLoading, error: correctionsError } = useCorrections()
  const { data: trendRaw, isLoading: trendLoading } = useCorrectionTrend()
  const { data: stats, isLoading: statsLoading } = useFeedbackStats()

  useEffect(() => {
    if (correctionsError) toast.error('Failed to load corrections')
  }, [correctionsError])

  const corrections: Correction[] = correctionsRaw || []
  const trend: {week: string, count: number}[] = trendRaw || []
  const isLoading = correctionsLoading || trendLoading || statsLoading

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Feedback', active: true }]} />
        <div className="mt-4 flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-3xl font-bold text-foreground">Feedback & Learning</h1>
            <p className="mt-1 text-sm text-muted-foreground">User corrections that make classification smarter over time</p>
          </div>
          <div className="flex gap-2">
            <Link href="/feedback/entities" className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted">
              <Braces className="h-4 w-4" />
              Custom Entities
            </Link>
            <Link href="/feedback/cache" className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted">
              <Database className="h-4 w-4" />
              Knowledge Cache
            </Link>
          </div>
        </div>
      </div>

      <div className="space-y-8 p-8">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
          <StatCard label="Total Corrections" value={String((stats as any)?.total_corrections || corrections.length)} icon={<MessageSquare className="h-6 w-6" />} />
          <StatCard label="Accuracy Improvement" value={(stats as any)?.accuracy_improvement || '—'} />
          <StatCard label="Knowledge Cache Size" value={(stats as any)?.cache_size || '—'} />
          <StatCard label="Cache Hit Rate" value={(stats as any)?.cache_hit_rate || '—'} />
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-8 lg:grid-cols-3">
            <div className="rounded-lg border border-border bg-card p-6">
              <h3 className="mb-2 text-lg font-semibold text-foreground">Correction Trend</h3>
              {trend.length > 0 && trend.some(t => t.count > 0) ? (
                <>
                  {(() => {
                    const firstWeek = trend[0]?.count || 0
                    const lastWeek = trend[trend.length - 1]?.count || 0
                    const isDecreasing = firstWeek > lastWeek
                    const isIncreasing = lastWeek > firstWeek
                    if (isDecreasing) {
                      return (
                        <p className="mb-6 text-sm text-green-600 dark:text-green-400">
                          Decreasing — model is improving
                        </p>
                      )
                    } else if (isIncreasing) {
                      return (
                        <p className="mb-6 text-sm text-yellow-600 dark:text-yellow-400">
                          Increasing — review corrections
                        </p>
                      )
                    } else {
                      return (
                        <p className="mb-6 text-sm text-muted-foreground">
                          Stable
                        </p>
                      )
                    }
                  })()}
                  <div className="flex h-36 items-end gap-2" role="img" aria-label="Weekly corrections trend">
                    {trend.map((t, i) => (
                      <div key={i} className="flex flex-1 flex-col items-center gap-1">
                        <div className="w-full rounded-t bg-primary/70" style={{ height: `${(t.count / Math.max(...trend.map(x => x.count), 1)) * 100}%` }} />
                        <span className="text-[10px] text-muted-foreground">{t.week}</span>
                      </div>
                    ))}
                  </div>
                </>
              ) : (
                <p className="text-center py-8 text-muted-foreground">No trend data available</p>
              )}
            </div>

            <div className="rounded-lg border border-border bg-card p-6 lg:col-span-2">
              <h3 className="mb-4 text-lg font-semibold text-foreground">Recent Corrections</h3>
              {corrections.length === 0 ? (
                <div className="text-center py-12 text-muted-foreground">
                  <MessageSquare className="h-12 w-12 mx-auto mb-4 opacity-50" />
                  <p>No corrections submitted yet</p>
                </div>
              ) : (
                <DataTable columns={columns} data={corrections} />
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
