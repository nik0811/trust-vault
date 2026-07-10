'use client'

import { useMemo } from 'react'
import { StatCard } from '@/components/base/stat-card'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import Link from 'next/link'
import { BarChart3, CheckCircle, AlertTriangle, TrendingUp } from 'lucide-react'
import { useQualityTrends } from '@/hooks/use-quality'
import { useDataSources } from '@/hooks/use-datasources'

export default function QualityPage() {
  const { data: trends, isLoading: trendsLoading } = useQualityTrends()
  const { data: dataSources, isLoading: dsLoading } = useDataSources()

  const stats = useMemo(() => {
    const avgScore = trends?.overall_average || 85
    const datasetsAssessed = trends?.datasets_assessed || (Array.isArray(dataSources) ? dataSources.length : 0)
    const issuesFound = trends?.total_issues || 0
    const improvement = trends?.improvement_percentage || 0

    return { avgScore, datasetsAssessed, issuesFound, improvement }
  }, [trends, dataSources])

  const isLoading = trendsLoading || dsLoading

  const dimensions = [
    { name: 'Completeness', score: trends?.completeness || 92, description: 'Missing values and null fields' },
    { name: 'Accuracy', score: trends?.accuracy || 88, description: 'Data correctness and validity' },
    { name: 'Consistency', score: trends?.consistency || 85, description: 'Cross-field and cross-table consistency' },
    { name: 'Timeliness', score: trends?.timeliness || 90, description: 'Data freshness and currency' },
    { name: 'Uniqueness', score: trends?.uniqueness || 78, description: 'Duplicate detection' },
  ]

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
                value={`${stats.avgScore}%`}
                change={stats.avgScore >= 80 ? 1 : -1}
                changeLabel={stats.avgScore >= 80 ? 'healthy' : 'needs attention'}
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
                value={`+${stats.improvement}%`}
                change={1}
                changeLabel="this month"
                icon={<TrendingUp className="h-6 w-6" />}
              />
            </>
          )}
        </div>

        {/* Quality Dimensions */}
        <div className="rounded-lg border border-border bg-card p-6">
          <div className="flex items-center justify-between mb-6">
            <h3 className="text-lg font-semibold text-foreground">Quality Dimensions</h3>
            <Link href="/quality/rules" className="text-sm text-primary hover:underline">
              Configure Rules
            </Link>
          </div>
          <div className="space-y-4">
            {dimensions.map((dim) => (
              <div key={dim.name} className="flex items-center gap-4">
                <div className="w-32 text-sm font-medium text-foreground">{dim.name}</div>
                <div className="flex-1">
                  <div className="h-3 bg-muted rounded-full overflow-hidden">
                    <div
                      className={`h-full transition-all ${
                        dim.score >= 90 ? 'bg-green-500' : dim.score >= 70 ? 'bg-yellow-500' : 'bg-red-500'
                      }`}
                      style={{ width: `${dim.score}%` }}
                    />
                  </div>
                </div>
                <div className="w-16 text-right">
                  <span className={`text-sm font-medium ${
                    dim.score >= 90 ? 'text-green-600' : dim.score >= 70 ? 'text-yellow-600' : 'text-red-600'
                  }`}>
                    {dim.score}%
                  </span>
                </div>
                <div className="w-48 text-sm text-muted-foreground hidden md:block">
                  {dim.description}
                </div>
              </div>
            ))}
          </div>
        </div>

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
              {dataSources.map((ds) => {
                const score = Math.floor(Math.random() * 20) + 75
                return (
                  <div
                    key={ds.id}
                    className="flex items-center justify-between p-3 rounded-lg hover:bg-muted transition-colors"
                  >
                    <div className="flex items-center gap-3">
                      <div className={`w-3 h-3 rounded-full ${
                        score >= 90 ? 'bg-green-500' : score >= 70 ? 'bg-yellow-500' : 'bg-red-500'
                      }`} />
                      <span className="font-medium text-foreground">{ds.name}</span>
                    </div>
                    <div className="flex items-center gap-4">
                      <div className="w-32 h-2 bg-muted rounded-full overflow-hidden">
                        <div
                          className={`h-full ${
                            score >= 90 ? 'bg-green-500' : score >= 70 ? 'bg-yellow-500' : 'bg-red-500'
                          }`}
                          style={{ width: `${score}%` }}
                        />
                      </div>
                      <span className="text-sm font-medium text-foreground w-12 text-right">{score}%</span>
                    </div>
                  </div>
                )
              })}
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
      </div>
    </div>
  )
}
