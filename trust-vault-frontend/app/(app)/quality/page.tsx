'use client'

import { useMemo } from 'react'
import { StatCard } from '@/components/base/stat-card'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import Link from 'next/link'
import { BarChart3, CheckCircle, AlertTriangle, TrendingUp, Database } from 'lucide-react'
import { useQualityTrends } from '@/hooks/use-quality'
import { useDataSources } from '@/hooks/use-datasources'

export default function QualityPage() {
  const { data: trends, isLoading: trendsLoading } = useQualityTrends()
  const { data: dataSources, isLoading: dsLoading } = useDataSources()

  const hasQualityData = trends && Array.isArray(trends) && trends.length > 0

  const stats = useMemo(() => {
    if (!hasQualityData) {
      return { avgScore: 0, datasetsAssessed: 0, issuesFound: 0, improvement: 0 }
    }
    const avgScore = Math.round(trends.reduce((sum: number, t: any) => sum + (t.overall || 0), 0) / trends.length * 100)
    const datasetsAssessed = Array.isArray(dataSources) ? dataSources.length : 0
    return { avgScore, datasetsAssessed, issuesFound: 0, improvement: 0 }
  }, [trends, dataSources, hasQualityData])

  const isLoading = trendsLoading || dsLoading

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Data Quality', active: true }]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">Data Quality</h1>
        <p className="text-sm text-muted-foreground mt-1">Monitor and improve data quality across your organization</p>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {isLoading ? (
            <>
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
            </>
          ) : (
            <>
              <StatCard
                label="Overall Score"
                value={hasQualityData ? `${stats.avgScore}%` : '-'}
                change={hasQualityData && stats.avgScore >= 80 ? 1 : hasQualityData ? -1 : undefined}
                changeLabel={hasQualityData ? (stats.avgScore >= 80 ? 'healthy' : 'needs attention') : undefined}
                icon={<BarChart3 className="h-6 w-6" />}
              />
              <StatCard
                label="Datasets Assessed"
                value={stats.datasetsAssessed.toString()}
                icon={<CheckCircle className="h-6 w-6" />}
              />
              <StatCard
                label="Issues Found"
                value={stats.issuesFound.toString()}
                icon={<AlertTriangle className="h-6 w-6" />}
              />
              <StatCard
                label="Improvement"
                value={hasQualityData ? `+${stats.improvement}%` : '-'}
                change={hasQualityData ? 1 : undefined}
                changeLabel={hasQualityData ? 'this month' : undefined}
                icon={<TrendingUp className="h-6 w-6" />}
              />
            </>
          )}
        </div>

        {!isLoading && !hasQualityData ? (
          <div className="rounded-lg border border-border bg-card p-12 text-center">
            <Database className="mx-auto h-12 w-12 text-muted-foreground/50" />
            <h3 className="mt-4 text-lg font-semibold text-foreground">No Quality Assessments Yet</h3>
            <p className="mt-2 text-sm text-muted-foreground max-w-md mx-auto">
              Quality scores will appear here after you run quality assessments on your data sources.
              Go to a data source and trigger a quality assessment to get started.
            </p>
            <Link 
              href="/data-sources" 
              className="mt-4 inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
            >
              View Data Sources
            </Link>
          </div>
        ) : (
          <>
            {/* Quality Dimensions - only show if we have data */}
            {hasQualityData && (
              <div className="rounded-lg border border-border bg-card p-6">
                <div className="flex items-center justify-between mb-6">
                  <h3 className="text-lg font-semibold text-foreground">Quality Dimensions</h3>
                  <Link href="/quality/rules" className="text-sm text-primary hover:underline">
                    Configure Rules
                  </Link>
                </div>
                <div className="space-y-4">
                  {['Completeness', 'Accuracy', 'Consistency', 'Timeliness', 'Uniqueness'].map((dim) => {
                    const score = stats.avgScore
                    return (
                      <div key={dim} className="flex items-center gap-4">
                        <div className="w-32 text-sm font-medium text-foreground">{dim}</div>
                        <div className="flex-1">
                          <div className="h-3 bg-muted rounded-full overflow-hidden">
                            <div
                              className={`h-full transition-all ${
                                score >= 90 ? 'bg-green-500' : score >= 70 ? 'bg-yellow-500' : 'bg-red-500'
                              }`}
                              style={{ width: `${score}%` }}
                            />
                          </div>
                        </div>
                        <div className="w-16 text-right">
                          <span className={`text-sm font-medium ${
                            score >= 90 ? 'text-green-600' : score >= 70 ? 'text-yellow-600' : 'text-red-600'
                          }`}>
                            {score}%
                          </span>
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>
            )}

            {/* Dataset Quality */}
            <div className="rounded-lg border border-border bg-card p-6">
              <h3 className="text-lg font-semibold text-foreground mb-4">Dataset Quality Scores</h3>
              {dsLoading ? (
                <div className="space-y-3">
                  <Skeleton className="h-12 w-full" />
                  <Skeleton className="h-12 w-full" />
                </div>
              ) : dataSources && dataSources.length > 0 ? (
                <div className="space-y-3">
                  {dataSources.map((ds) => (
                    <Link
                      key={ds.id}
                      href={`/data-sources/${ds.id}`}
                      className="flex items-center justify-between p-3 rounded-lg hover:bg-muted transition-colors"
                    >
                      <div className="flex items-center gap-3">
                        <div className="w-3 h-3 rounded-full bg-muted-foreground/30" />
                        <span className="font-medium text-foreground">{ds.name}</span>
                      </div>
                      <span className="text-sm text-muted-foreground">Run assessment →</span>
                    </Link>
                  ))}
                </div>
              ) : (
                <div className="text-center py-8">
                  <p className="text-muted-foreground">No datasets to assess</p>
                  <Link href="/data-sources/new" className="text-primary hover:underline text-sm mt-2 inline-block">
                    Add a data source
                  </Link>
                </div>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  )
}
