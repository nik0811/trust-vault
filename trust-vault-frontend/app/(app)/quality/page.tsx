'use client'

import { useMemo, useState } from 'react'
import { StatCard } from '@/components/base/stat-card'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import Link from 'next/link'
import { BarChart3, CheckCircle, AlertTriangle, TrendingUp, Database, Zap, Eye, Shield, X, Loader2 } from 'lucide-react'
import { useQualityTrends, useQualityDimensions, useAutoProfile, useDataProfile, type ColumnProfile } from '@/hooks/use-quality'
import { useDataSources } from '@/hooks/use-datasources'
import { cn } from '@/lib/utils'

function ProfileModal({ datasourceId, datasourceName, onClose }: { datasourceId: string; datasourceName: string; onClose: () => void }) {
  const { data: profile, isLoading } = useDataProfile(datasourceId)
  const autoProfile = useAutoProfile()

  const columns: ColumnProfile[] = profile?.columns ?? []

  return (
    <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4">
      <div className="bg-card rounded-xl border border-border w-full max-w-3xl max-h-[85vh] flex flex-col">
        <div className="sticky top-0 bg-card border-b border-border p-5 flex items-start justify-between">
          <div>
            <h2 className="text-lg font-bold">{datasourceName} — Data Profile</h2>
            {profile && (
              <p className="text-xs text-muted-foreground mt-1">
                {profile.total_columns} columns · {profile.pii_columns} PII columns · profiled {new Date(profile.profiled_at).toLocaleString()}
              </p>
            )}
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => autoProfile.mutate(datasourceId)}
              disabled={autoProfile.isPending}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs bg-primary text-primary-foreground rounded-md hover:opacity-90 disabled:opacity-50"
            >
              {autoProfile.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Zap className="h-3.5 w-3.5" />}
              Re-profile
            </button>
            <button onClick={onClose}><X className="h-5 w-5 text-muted-foreground" /></button>
          </div>
        </div>
        <div className="flex-1 overflow-y-auto p-5">
          {isLoading ? (
            <div className="flex items-center justify-center py-12 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin mr-2" /> Loading profile...
            </div>
          ) : !profile || profile.status === 'not_profiled' ? (
            <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
              <Database className="h-12 w-12 mb-3 opacity-30" />
              <p className="mb-4">No profile yet for this datasource</p>
              <button
                onClick={() => autoProfile.mutate(datasourceId)}
                disabled={autoProfile.isPending}
                className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md text-sm hover:opacity-90 disabled:opacity-50"
              >
                {autoProfile.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />}
                Auto-Profile Now
              </button>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted/50 sticky top-0">
                  <tr>
                    {['Column', 'Type', 'Null %', 'Distinct', 'Sample Values', 'PII'].map(h => (
                      <th key={h} className="text-left px-3 py-2 text-xs font-semibold text-muted-foreground uppercase tracking-wide">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {columns.map((col, i) => (
                    <tr key={i} className={cn('hover:bg-muted/30', col.is_pii && 'bg-yellow-50/30 dark:bg-yellow-950/10')}>
                      <td className="px-3 py-2.5 font-mono font-medium">
                        <div className="flex items-center gap-1.5">
                          {col.is_pii && <Shield className="h-3.5 w-3.5 text-yellow-500 shrink-0" />}
                          {col.column_name}
                        </div>
                      </td>
                      <td className="px-3 py-2.5">
                        <span className="text-xs bg-muted px-2 py-0.5 rounded">{col.inferred_type}</span>
                      </td>
                      <td className="px-3 py-2.5 text-muted-foreground text-xs">
                        {col.null_rate.toFixed(1)}%
                      </td>
                      <td className="px-3 py-2.5 text-muted-foreground text-xs">
                        {col.distinct_count.toLocaleString()}
                      </td>
                      <td className="px-3 py-2.5">
                        <div className="flex flex-wrap gap-1">
                          {(col.sample_values ?? []).slice(0, 2).map((sv, j) => (
                            <span key={j} className="text-xs font-mono bg-muted px-1.5 py-0.5 rounded max-w-[120px] truncate">
                              {col.is_pii ? '***' : sv}
                            </span>
                          ))}
                        </div>
                      </td>
                      <td className="px-3 py-2.5">
                        {col.is_pii ? (
                          <span className="text-xs bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400 px-1.5 py-0.5 rounded">PII</span>
                        ) : (
                          <span className="text-xs text-muted-foreground">—</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default function QualityPage() {
  const { data: trends, isLoading: trendsLoading } = useQualityTrends()
  const { data: dimensions, isLoading: dimensionsLoading } = useQualityDimensions()
  const { data: dataSources, isLoading: dsLoading } = useDataSources()
  const autoProfile = useAutoProfile()
  const [profileDsId, setProfileDsId] = useState<string | null>(null)
  const profileDs = dataSources?.find((ds: any) => ds.id === profileDsId)

  const hasQualityData = trends && Array.isArray(trends) && trends.length > 0

  const stats = useMemo(() => {
    const datasetsAssessed = Array.isArray(dataSources) ? dataSources.length : 0
    if (dimensions) {
      const avgScore = Math.round((dimensions.overall_score ?? 0) * 100)
      const issuesFound = dimensions.issues_found ?? 0
      return { avgScore, datasetsAssessed, issuesFound, improvement: 0 }
    }
    if (!hasQualityData) {
      return { avgScore: 0, datasetsAssessed, issuesFound: 0, improvement: 0 }
    }
    const avgScore = Math.round(trends.reduce((sum: number, t: any) => sum + (t.overall || 0), 0) / trends.length * 100)
    return { avgScore, datasetsAssessed, issuesFound: 0, improvement: 0 }
  }, [trends, dimensions, dataSources, hasQualityData])

  // Per-dimension scores (0-100), each derived independently from real data
  const dimScores = useMemo(() => {
    if (!dimensions) return null
    return {
      Completeness: Math.round((dimensions.completeness ?? 0) * 100),
      Accuracy: Math.round((dimensions.accuracy ?? 0) * 100),
      Consistency: Math.round((dimensions.consistency ?? 0) * 100),
      Timeliness: Math.round((dimensions.timeliness ?? 0) * 100),
      Uniqueness: Math.round((dimensions.uniqueness ?? 0) * 100),
    }
  }, [dimensions])

  const hasDimensions = dimScores !== null && Object.values(dimScores).some(v => v > 0)

  const isLoading = trendsLoading || dsLoading || dimensionsLoading

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
                value={(hasDimensions || hasQualityData) ? `${stats.avgScore}%` : '-'}
                change={(hasDimensions || hasQualityData) && stats.avgScore >= 80 ? 1 : (hasDimensions || hasQualityData) ? -1 : undefined}
                changeLabel={(hasDimensions || hasQualityData) ? (stats.avgScore >= 80 ? 'healthy' : 'needs attention') : undefined}
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
                change={stats.issuesFound > 0 ? -1 : undefined}
                changeLabel={stats.issuesFound > 0 ? 'need attention' : undefined}
                icon={<AlertTriangle className="h-6 w-6" />}
              />
              <StatCard
                label="Improvement"
                value={(hasDimensions || hasQualityData) ? `+${stats.improvement}%` : '-'}
                change={hasQualityData ? 1 : undefined}
                changeLabel={hasQualityData ? 'this month' : undefined}
                icon={<TrendingUp className="h-6 w-6" />}
              />
            </>
          )}
        </div>

        {!isLoading && !hasQualityData && !hasDimensions ? (
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
            {hasDimensions && dimScores && (
              <div className="rounded-lg border border-border bg-card p-6">
                <div className="flex items-center justify-between mb-6">
                  <h3 className="text-lg font-semibold text-foreground">Quality Dimensions</h3>
                  <Link href="/quality/rules" className="text-sm text-primary hover:underline">
                    Configure Rules
                  </Link>
                </div>
                <div className="space-y-4">
                  {(Object.entries(dimScores) as [string, number][]).map(([dim, score]) => (
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
                  ))}
                </div>
              </div>
            )}

            {/* Dataset Quality */}
            <div className="rounded-lg border border-border bg-card p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-foreground">Dataset Quality Scores</h3>
                <Link href="/quality/cde" className="text-sm text-primary hover:underline flex items-center gap-1">
                  <Shield className="h-4 w-4" />
                  Critical Elements
                </Link>
              </div>
              {dsLoading ? (
                <div className="space-y-3">
                  <Skeleton className="h-12 w-full" />
                  <Skeleton className="h-12 w-full" />
                </div>
              ) : dataSources && dataSources.length > 0 ? (
                <div className="space-y-2">
                  {dataSources.map((ds: any) => (
                    <div
                      key={ds.id}
                      className="flex items-center justify-between p-3 rounded-lg hover:bg-muted transition-colors"
                    >
                      <Link href={`/data-sources/${ds.id}`} className="flex items-center gap-3 flex-1 min-w-0">
                        <div className="w-2.5 h-2.5 rounded-full bg-muted-foreground/30 shrink-0" />
                        <span className="font-medium text-foreground truncate">{ds.name}</span>
                      </Link>
                      <div className="flex items-center gap-2 shrink-0">
                        <button
                          onClick={() => setProfileDsId(ds.id)}
                          className="flex items-center gap-1.5 px-3 py-1.5 text-xs border border-border rounded-md hover:bg-muted transition-colors"
                        >
                          <Eye className="h-3.5 w-3.5" />
                          Profile
                        </button>
                        <button
                          onClick={() => { autoProfile.mutate(ds.id); setProfileDsId(ds.id) }}
                          disabled={autoProfile.isPending}
                          className="flex items-center gap-1.5 px-3 py-1.5 text-xs bg-primary text-primary-foreground rounded-md hover:opacity-90 disabled:opacity-50"
                        >
                          {autoProfile.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Zap className="h-3.5 w-3.5" />}
                          Auto-Profile
                        </button>
                        <Link href={`/data-sources/${ds.id}`} className="text-sm text-muted-foreground hover:text-foreground">→</Link>
                      </div>
                    </div>
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

      {profileDsId && profileDs && (
        <ProfileModal
          datasourceId={profileDsId}
          datasourceName={(profileDs as any).name}
          onClose={() => setProfileDsId(null)}
        />
      )}
    </div>
  )
}
