'use client'

import { useMemo, useState } from 'react'
import { StatCard } from '@/components/base/stat-card'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import Link from 'next/link'
import {
  Lightbulb, AlertTriangle, FileText, TrendingUp, Shield,
  ChevronDown, ChevronRight, RefreshCw, Clock, Database, History, CheckCircle
} from 'lucide-react'
import {
  useRecommendations, useComplianceGaps, useRiskScore,
  useRunComplianceAssessment, useAssessmentLogs, type Recommendation, type AssessmentLog
} from '@/hooks/use-advisor'

function RecommendationCard({ rec }: { rec: Recommendation }) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="rounded-lg border border-border bg-muted/30 overflow-hidden">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-start gap-4 p-4 hover:bg-muted/50 transition-colors text-left"
      >
        <div className={`w-2 h-2 rounded-full mt-2 shrink-0 ${
          rec.priority === 'high' ? 'bg-red-500' :
          rec.priority === 'medium' ? 'bg-yellow-500' : 'bg-green-500'
        }`} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <p className="font-medium text-foreground">{rec.title}</p>
            {rec.evidence_count > 0 && (
              <span className="flex items-center gap-1 px-1.5 py-0 rounded bg-primary/10 text-primary text-xs">
                <FileText className="h-3 w-3" />
                {rec.evidence_count}
              </span>
            )}
          </div>
          <p className="text-sm text-muted-foreground mt-1">{rec.description}</p>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <span className={`px-2 py-0.5 rounded text-xs ${
            rec.priority === 'high' ? 'bg-red-500/10 text-red-600' :
            rec.priority === 'medium' ? 'bg-yellow-500/10 text-yellow-600' : 'bg-green-500/10 text-green-600'
          }`}>
            {rec.priority}
          </span>
          {expanded ? <ChevronDown className="h-4 w-4 text-muted-foreground" /> : <ChevronRight className="h-4 w-4 text-muted-foreground" />}
        </div>
      </button>

      {expanded && (
        <div className="border-t border-border bg-background p-4 space-y-3">
          {rec.regulation_article && (
            <div>
              <h5 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1">
                Regulation Reference
              </h5>
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-primary/10 text-primary border border-primary/20 text-xs font-mono">
                <FileText className="h-3 w-3" />
                {rec.regulation_article.split(' - ')[0]}
              </span>
              <p className="text-xs text-muted-foreground mt-1">{rec.regulation_article}</p>
            </div>
          )}

          {rec.severity_reason && (
            <div className="rounded-md border border-border bg-muted/30 p-3">
              <h5 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1 flex items-center gap-1">
                <Shield className="h-3 w-3" /> Severity Justification
              </h5>
              <p className="text-xs text-foreground">{rec.severity_reason}</p>
            </div>
          )}

          {rec.evidence_summary && (
            <div className="rounded-md border border-border bg-muted/30 p-3">
              <h5 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1">
                Evidence Summary
              </h5>
              <p className="text-xs text-foreground">{rec.evidence_summary}</p>
            </div>
          )}

          {rec.evidence && rec.evidence.length > 0 && (
            <div className="rounded-md border border-border bg-muted/30 p-3">
              <h5 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-2">
                Supporting Evidence ({rec.evidence.length})
              </h5>
              <div className="space-y-2">
                {rec.evidence.slice(0, 3).map((ev) => (
                  <div key={ev.id} className="flex items-start gap-2 text-xs">
                    <div className="w-1.5 h-1.5 rounded-full bg-primary mt-1.5 shrink-0" />
                    <div className="flex-1">
                      <p className="text-foreground">{ev.description}</p>
                      <span className="text-muted-foreground flex items-center gap-1 mt-0.5">
                        <Clock className="h-3 w-3" />
                        {new Date(ev.detected_at).toLocaleDateString()} via {ev.source}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {rec.affected_assets && rec.affected_assets.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {rec.affected_assets.slice(0, 5).map((asset) => (
                <span
                  key={asset.id}
                  className="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-muted border border-border text-xs"
                >
                  <Database className="h-3 w-3 text-muted-foreground" />
                  {asset.name}
                </span>
              ))}
              {rec.affected_assets.length > 5 && (
                <span className="text-xs text-muted-foreground">
                  +{rec.affected_assets.length - 5} more
                </span>
              )}
            </div>
          )}

          <div className="rounded-md border border-primary/20 bg-primary/5 p-3">
            <h5 className="text-xs font-semibold text-primary uppercase tracking-wide mb-1">
              Recommended Action
            </h5>
            <p className="text-sm text-foreground">{rec.action}</p>
          </div>
        </div>
      )}
    </div>
  )
}

export default function AdvisorPage() {
  const { data: recommendations, isLoading: recsLoading } = useRecommendations()
  const { data: gaps, isLoading: gapsLoading } = useComplianceGaps()
  const { data: riskScore, isLoading: riskLoading } = useRiskScore()
  const assessment = useRunComplianceAssessment()
  const { data: assessmentLogs, isLoading: logsLoading } = useAssessmentLogs()
  const [showLogs, setShowLogs] = useState(true)

  const stats = useMemo(() => {
    const recsCount = Array.isArray(recommendations) ? recommendations.length : 0
    const gapsCount = Array.isArray(gaps) ? gaps.filter((g: any) => g.status !== 'resolved').length : 0
    const score = riskScore?.overall_score ? Math.round(riskScore.overall_score * 100) : 0
    const totalEvidence = Array.isArray(recommendations)
      ? recommendations.reduce((sum: number, r: any) => sum + (r.evidence_count || 0), 0)
      : 0

    return { recsCount, gapsCount, score, totalEvidence }
  }, [recommendations, gaps, riskScore])

  const isLoading = recsLoading || gapsLoading || riskLoading

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Compliance Advisor', active: true }]} />
        <div className="flex items-center justify-between mt-4">
          <div>
            <h1 className="text-3xl font-bold text-foreground">Compliance Advisor</h1>
            <p className="text-sm text-muted-foreground mt-1">
              Evidence-backed compliance recommendations and audit-grade gap analysis
            </p>
          </div>
          <button
            onClick={() => assessment.mutate()}
            disabled={assessment.isPending}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`h-4 w-4 ${assessment.isPending ? 'animate-spin' : ''}`} />
            {assessment.isPending ? 'Running Assessment...' : 'Run Assessment'}
          </button>
        </div>

        {/* Assessment result banner */}
        {assessment.isPending && (
          <div className="mt-4 rounded-lg border border-primary/20 bg-primary/5 p-4">
            <div className="flex items-center gap-3 text-sm text-muted-foreground">
              <RefreshCw className="h-4 w-4 animate-spin text-primary" />
              <span>Running compliance assessment — this may take a moment…</span>
            </div>
          </div>
        )}
        {assessment.isSuccess && assessmentLogs && assessmentLogs.length > 0 && (() => {
          const log = assessmentLogs[0]
          const scorePercent = Math.round((log.compliance_score ?? 0) * 100)
          const assessedAt = log.created_at ? new Date(log.created_at).toLocaleString() : '—'
          return (
            <div className="mt-4 rounded-lg border border-primary/20 bg-primary/5 p-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <div className="text-center">
                    <p className="text-2xl font-bold text-foreground">{scorePercent}%</p>
                    <p className="text-xs text-muted-foreground">Compliance Score</p>
                  </div>
                  <div className="h-8 w-px bg-border" />
                  <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-muted-foreground">
                    <span>{log.data_sources_checked} sources checked</span>
                    <span>{log.classifications_checked} classifications</span>
                    <span>{log.policies_evaluated} policies evaluated</span>
                    <span>{log.total_evidence} evidence items</span>
                  </div>
                </div>
                <div className="text-xs text-muted-foreground text-right">
                  <p>Assessed: {assessedAt}</p>
                  <p>Regulations: {log.regulations_covered?.join(', ') ?? '—'}</p>
                </div>
              </div>
            </div>
          )
        })()}
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
                label="Evidence Items"
                value={stats.totalEvidence.toString()}
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
              View evidence-backed gaps across regulations
            </p>
            <p className="text-sm text-primary mt-4">{stats.gapsCount} gaps with evidence trails</p>
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

        {/* Recommendations with Evidence */}
        <div className="rounded-lg border border-border bg-card p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-foreground">Top Recommendations</h3>
            {stats.totalEvidence > 0 && (
              <span className="text-xs text-muted-foreground bg-muted px-2 py-1 rounded">
                {stats.totalEvidence} evidence items supporting {stats.recsCount} findings
              </span>
            )}
          </div>
          {recsLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-16 w-full" />
              <Skeleton className="h-16 w-full" />
            </div>
          ) : recommendations && Array.isArray(recommendations) && recommendations.length > 0 ? (
            <div className="space-y-3">
              {recommendations.slice(0, 7).map((rec: Recommendation) => (
                <RecommendationCard key={rec.id} rec={rec} />
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <Lightbulb className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <p className="text-muted-foreground">No recommendations at this time</p>
              <p className="text-sm text-muted-foreground mt-1">
                Click "Run Assessment" to analyze your compliance posture
              </p>
            </div>
          )}
        </div>

        {/* Assessment History Logs */}
        <div className="rounded-lg border border-border bg-card p-6">
          <button
            onClick={() => setShowLogs(!showLogs)}
            className="w-full flex items-center justify-between"
          >
            <div className="flex items-center gap-2">
              <History className="h-5 w-5 text-muted-foreground" />
              <h3 className="text-lg font-semibold text-foreground">Assessment History</h3>
              {Array.isArray(assessmentLogs) && assessmentLogs.length > 0 && (
                <span className="text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded">
                  {assessmentLogs.length} runs
                </span>
              )}
            </div>
            {showLogs ? <ChevronDown className="h-4 w-4 text-muted-foreground" /> : <ChevronRight className="h-4 w-4 text-muted-foreground" />}
          </button>

          {showLogs && (
            <div className="mt-4">
              {logsLoading ? (
                <div className="space-y-2">
                  <Skeleton className="h-14 w-full" />
                  <Skeleton className="h-14 w-full" />
                </div>
              ) : Array.isArray(assessmentLogs) && assessmentLogs.length > 0 ? (
                <div className="space-y-2">
                  {assessmentLogs.map((log: AssessmentLog) => {
                    const score = log.compliance_score ?? 0
                    const scorePercent = Math.round(score * 100)
                    const logDate = log.created_at ? new Date(log.created_at).toLocaleString() : '—'
                    return (
                    <div key={log.id} className="flex items-center justify-between p-3 rounded-lg border border-border bg-muted/30">
                      <div className="flex items-center gap-3">
                        <CheckCircle className={`h-4 w-4 ${score >= 0.8 ? 'text-green-500' : score >= 0.6 ? 'text-yellow-500' : 'text-red-500'}`} />
                        <div>
                          <p className="text-sm font-medium text-foreground">
                            {scorePercent}% compliance — {log.total_findings} finding{log.total_findings !== 1 ? 's' : ''}
                          </p>
                          <p className="text-xs text-muted-foreground flex items-center gap-1 mt-0.5">
                            <Clock className="h-3 w-3" />
                            {logDate}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-4 text-xs text-muted-foreground">
                        <span>{log.data_sources_checked} sources</span>
                        <span>{log.classifications_checked} classifications</span>
                        <span>{log.policies_evaluated} policies</span>
                        {log.critical_findings > 0 && (
                          <span className="text-red-500 font-medium">{log.critical_findings} critical</span>
                        )}
                      </div>
                    </div>
                  )})}
                </div>
              ) : (
                <div className="text-center py-6">
                  <History className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
                  <p className="text-sm text-muted-foreground">No assessment runs yet</p>
                  <p className="text-xs text-muted-foreground mt-1">Run an assessment to start tracking history</p>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
