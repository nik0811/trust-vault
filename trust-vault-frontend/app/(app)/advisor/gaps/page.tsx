'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { StatusIndicator } from '@/components/base/status-badge'
import {
  AlertTriangle, ChevronDown, ChevronRight, Database, Clock,
  FileText, Shield, Search, RefreshCw
} from 'lucide-react'
import {
  useComplianceGaps, useRunComplianceAssessment,
  type ComplianceGap, type EvidenceItem, type AffectedAsset
} from '@/hooks/use-advisor'

function SeverityBadge({ severity }: { severity: string }) {
  const colors: Record<string, string> = {
    CRITICAL: 'bg-red-500/10 text-red-600 border-red-500/20',
    HIGH: 'bg-orange-500/10 text-orange-600 border-orange-500/20',
    MEDIUM: 'bg-yellow-500/10 text-yellow-600 border-yellow-500/20',
    LOW: 'bg-green-500/10 text-green-600 border-green-500/20',
  }
  return (
    <span className={`px-2 py-0.5 rounded border text-xs font-medium ${colors[severity] || colors.MEDIUM}`}>
      {severity}
    </span>
  )
}

function RegulationBadge({ article }: { article: string }) {
  return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-primary/10 text-primary border border-primary/20 text-xs font-mono">
      <FileText className="h-3 w-3" />
      {article}
    </span>
  )
}

function EvidencePanel({ evidence, assets }: { evidence: EvidenceItem[]; assets: AffectedAsset[] }) {
  if (evidence.length === 0 && assets.length === 0) return null

  return (
    <div className="mt-3 space-y-3">
      {evidence.length > 0 && (
        <div className="rounded-md border border-border bg-muted/30 p-3">
          <h5 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-2 flex items-center gap-1">
            <Search className="h-3 w-3" /> Evidence ({evidence.length})
          </h5>
          <div className="space-y-2">
            {evidence.map((ev) => (
              <div key={ev.id} className="flex items-start gap-2 text-xs">
                <div className="w-1.5 h-1.5 rounded-full bg-primary mt-1.5 shrink-0" />
                <div className="flex-1">
                  <p className="text-foreground">{ev.description}</p>
                  <div className="flex items-center gap-3 mt-1 text-muted-foreground">
                    <span className="flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      {new Date(ev.detected_at).toLocaleDateString()}
                    </span>
                    <span className="px-1.5 py-0 rounded bg-muted text-muted-foreground">
                      {ev.source}
                    </span>
                    {ev.type && (
                      <span className="px-1.5 py-0 rounded bg-muted text-muted-foreground">
                        {ev.type.replace(/_/g, ' ')}
                      </span>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {assets.length > 0 && (
        <div className="rounded-md border border-border bg-muted/30 p-3">
          <h5 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-2 flex items-center gap-1">
            <Database className="h-3 w-3" /> Affected Assets ({assets.length})
          </h5>
          <div className="flex flex-wrap gap-2">
            {assets.map((asset) => (
              <span
                key={asset.id}
                className="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-background border border-border text-xs"
              >
                <Database className="h-3 w-3 text-muted-foreground" />
                {asset.name}
                <span className="text-muted-foreground">({asset.type})</span>
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function GapRow({ gap }: { gap: ComplianceGap }) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="border border-border rounded-lg overflow-hidden">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-4 p-4 hover:bg-muted/50 transition-colors text-left"
      >
        <div className="shrink-0">
          {expanded ? (
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          ) : (
            <ChevronRight className="h-4 w-4 text-muted-foreground" />
          )}
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="font-semibold text-foreground">{gap.regulation}</span>
            <span className="text-muted-foreground">-</span>
            <span className="text-sm text-foreground">{gap.requirement}</span>
          </div>
          <p className="text-xs text-muted-foreground truncate">{gap.remediation}</p>
        </div>

        <div className="flex items-center gap-3 shrink-0">
          {gap.evidence_count > 0 && (
            <span className="flex items-center gap-1 text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded">
              <FileText className="h-3 w-3" />
              {gap.evidence_count} evidence
            </span>
          )}
          <SeverityBadge severity={gap.severity} />
          <StatusIndicator
            status={gap.status === 'resolved' ? 'success' : gap.status === 'in_progress' ? 'pending' : 'error'}
            label={gap.status}
          />
        </div>
      </button>

      {expanded && (
        <div className="border-t border-border bg-muted/20 p-4 space-y-3">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <h5 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1">
                Regulation Reference
              </h5>
              {gap.regulation_article && <RegulationBadge article={gap.regulation_article.split(' - ')[0]} />}
              {gap.regulation_article && (
                <p className="text-xs text-muted-foreground mt-1">
                  {gap.regulation_article}
                </p>
              )}
            </div>
            <div>
              <h5 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1">
                Assessment Details
              </h5>
              <div className="flex items-center gap-4 text-xs text-muted-foreground">
                <span className="flex items-center gap-1">
                  <Clock className="h-3 w-3" />
                  Detected: {new Date(gap.detected_at).toLocaleString()}
                </span>
                <span className="flex items-center gap-1">
                  <RefreshCw className="h-3 w-3" />
                  Last assessed: {new Date(gap.last_assessed).toLocaleString()}
                </span>
              </div>
            </div>
          </div>

          {gap.severity_reason && (
            <div className="rounded-md border border-border bg-background p-3">
              <h5 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1 flex items-center gap-1">
                <Shield className="h-3 w-3" /> Severity Justification
              </h5>
              <p className="text-xs text-foreground">{gap.severity_reason}</p>
            </div>
          )}

          <div className="rounded-md border border-border bg-background p-3">
            <h5 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1">
              Recommended Remediation
            </h5>
            <p className="text-sm text-foreground">{gap.remediation}</p>
          </div>

          <EvidencePanel evidence={gap.evidence || []} assets={gap.affected_assets || []} />
        </div>
      )}
    </div>
  )
}

export default function GapsPage() {
  const { data: gaps, isLoading } = useComplianceGaps()
  const assessment = useRunComplianceAssessment()

  const gapsData: ComplianceGap[] = Array.isArray(gaps) ? gaps : []

  const groupedGaps = gapsData.reduce<Record<string, ComplianceGap[]>>((acc, gap) => {
    if (!acc[gap.regulation]) acc[gap.regulation] = []
    acc[gap.regulation].push(gap)
    return acc
  }, {})

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Compliance Advisor', href: '/advisor' },
            { label: 'Gaps', active: true },
          ]}
        />
        <div className="flex items-center justify-between mt-4">
          <div>
            <h1 className="text-3xl font-bold text-foreground">Compliance Gaps</h1>
            <p className="text-sm text-muted-foreground mt-1">
              Evidence-backed compliance gap analysis across regulations
            </p>
          </div>
          <button
            onClick={() => assessment.mutate()}
            disabled={assessment.isPending}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`h-4 w-4 ${assessment.isPending ? 'animate-spin' : ''}`} />
            {assessment.isPending ? 'Assessing...' : 'Run Assessment'}
          </button>
        </div>

        {/* Assessment result banner */}
        {assessment.data && (
          <div className="mt-4 rounded-lg border border-primary/20 bg-primary/5 p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className="text-center">
                  <p className="text-2xl font-bold text-foreground">
                    {Math.round(assessment.data.compliance_score * 100)}%
                  </p>
                  <p className="text-xs text-muted-foreground">Score</p>
                </div>
                <div className="h-8 w-px bg-border" />
                <div className="flex gap-3 text-xs">
                  {assessment.data.critical_findings > 0 && (
                    <span className="px-2 py-1 rounded bg-red-500/10 text-red-600">
                      {assessment.data.critical_findings} Critical
                    </span>
                  )}
                  {assessment.data.high_findings > 0 && (
                    <span className="px-2 py-1 rounded bg-orange-500/10 text-orange-600">
                      {assessment.data.high_findings} High
                    </span>
                  )}
                  {assessment.data.medium_findings > 0 && (
                    <span className="px-2 py-1 rounded bg-yellow-500/10 text-yellow-600">
                      {assessment.data.medium_findings} Medium
                    </span>
                  )}
                  {assessment.data.low_findings > 0 && (
                    <span className="px-2 py-1 rounded bg-green-500/10 text-green-600">
                      {assessment.data.low_findings} Low
                    </span>
                  )}
                </div>
              </div>
              <div className="text-xs text-muted-foreground text-right">
                <p>{assessment.data.total_evidence} evidence items collected</p>
                <p>Assessed: {new Date(assessment.data.assessed_at).toLocaleString()}</p>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Content */}
      <div className="p-8">
        {isLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
          </div>
        ) : gapsData.length > 0 ? (
          <div className="space-y-8">
            {Object.entries(groupedGaps).map(([regulation, regGaps]) => (
              <div key={regulation}>
                <div className="flex items-center gap-2 mb-3">
                  <h2 className="text-lg font-semibold text-foreground">{regulation}</h2>
                  <span className="text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded">
                    {regGaps.length} gap{regGaps.length !== 1 ? 's' : ''}
                  </span>
                  <span className="text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded">
                    {regGaps.reduce((sum, g) => sum + (g.evidence_count || 0), 0)} evidence items
                  </span>
                </div>
                <div className="space-y-3">
                  {regGaps.map((gap, idx) => (
                    <GapRow key={`${regulation}-${idx}`} gap={gap} />
                  ))}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <AlertTriangle className="h-12 w-12 mx-auto text-green-500 mb-4" />
            <p className="text-foreground font-medium">No compliance gaps detected</p>
            <p className="text-sm text-muted-foreground mt-1">
              Your organization is meeting all identified compliance requirements
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
