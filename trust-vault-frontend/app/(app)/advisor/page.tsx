'use client'

import { useMemo } from 'react'
import { StatCard } from '@/components/base/stat-card'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import Link from 'next/link'
import { Lightbulb, AlertTriangle, FileText, TrendingUp } from 'lucide-react'
import { useRecommendations, useComplianceGaps, useRiskScore } from '@/hooks/use-advisor'

export default function AdvisorPage() {
  const { data: recommendations, isLoading: recsLoading } = useRecommendations()
  const { data: gaps, isLoading: gapsLoading } = useComplianceGaps()
  const { data: riskScore, isLoading: riskLoading } = useRiskScore()

  const stats = useMemo(() => {
    const recsCount = Array.isArray(recommendations) ? recommendations.length : 0
    const gapsCount = Array.isArray(gaps) ? gaps.filter((g: any) => g.status !== 'resolved').length : 0
    const score = riskScore?.overall_score ? Math.round((1 - riskScore.overall_score) * 100) : 0

    return { recsCount, gapsCount, score }
  }, [recommendations, gaps, riskScore])

  const isLoading = recsLoading || gapsLoading || riskLoading

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Compliance Advisor', active: true }]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">Compliance Advisor</h1>
        <p className="text-sm text-muted-foreground mt-1">AI-powered compliance recommendations and gap analysis</p>
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
                label="Compliance Score"
                value={`${stats.score}%`}
                change={stats.score >= 80 ? 1 : -1}
                changeLabel={stats.score >= 80 ? 'healthy' : 'needs attention'}
                icon={<TrendingUp className="h-6 w-6" />}
              />
              <StatCard
                label="Recommendations"
                value={stats.recsCount.toString()}
                icon={<Lightbulb className="h-6 w-6" />}
              />
              <StatCard
                label="Open Gaps"
                value={stats.gapsCount.toString()}
                icon={<AlertTriangle className="h-6 w-6" />}
              />
              <StatCard
                label="Defense Dockets"
                value="Ready"
                icon={<FileText className="h-6 w-6" />}
              />
            </>
          )}
        </div>

        {/* Quick Links */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <Link
            href="/advisor/gaps"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <AlertTriangle className="h-8 w-8 text-yellow-500 mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Compliance Gaps</h3>
            <p className="text-sm text-muted-foreground mt-1">
              View and address compliance gaps across regulations
            </p>
            <p className="text-sm text-primary mt-4">{stats.gapsCount} gaps to address</p>
          </Link>

          <Link
            href="/advisor/defense-docket"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <FileText className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Defense Docket</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Generate audit-ready compliance documentation
            </p>
            <p className="text-sm text-primary mt-4">Generate report</p>
          </Link>

          <Link
            href="/advisor/playbooks"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <Lightbulb className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Playbooks</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Step-by-step guides for common compliance issues
            </p>
            <p className="text-sm text-primary mt-4">View playbooks</p>
          </Link>
        </div>

        {/* Recommendations */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Top Recommendations</h3>
          {recsLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-16 w-full" />
              <Skeleton className="h-16 w-full" />
            </div>
          ) : recommendations && Array.isArray(recommendations) && recommendations.length > 0 ? (
            <div className="space-y-3">
              {recommendations.slice(0, 5).map((rec: any) => (
                <div
                  key={rec.id}
                  className="flex items-start gap-4 p-4 rounded-lg bg-muted/50"
                >
                  <div className={`w-2 h-2 rounded-full mt-2 ${
                    rec.priority === 'high' ? 'bg-red-500' : 
                    rec.priority === 'medium' ? 'bg-yellow-500' : 'bg-green-500'
                  }`} />
                  <div className="flex-1">
                    <p className="font-medium text-foreground">{rec.title}</p>
                    <p className="text-sm text-muted-foreground mt-1">{rec.description}</p>
                  </div>
                  <span className={`px-2 py-0.5 rounded text-xs ${
                    rec.priority === 'high' ? 'bg-red-500/10 text-red-600' : 
                    rec.priority === 'medium' ? 'bg-yellow-500/10 text-yellow-600' : 'bg-green-500/10 text-green-600'
                  }`}>
                    {rec.priority}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <Lightbulb className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <p className="text-muted-foreground">No recommendations at this time</p>
              <p className="text-sm text-muted-foreground mt-1">
                Recommendations will appear as the system analyzes your data
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
