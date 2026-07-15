'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import {
  FileText,
  Plus,
  X,
  ChevronDown,
  ChevronUp,
  AlertTriangle,
  ShieldCheck,
  BarChart3,
  Bot,
  ClipboardList,
  Printer,
  Loader2,
  CheckCircle2,
  Clock,
  AlertCircle,
} from 'lucide-react'
import { useReports, useGenerateReport, Report, ReportType } from '@/hooks/use-reports'
import { cn } from '@/lib/utils'

// ─── Types ───────────────────────────────────────────────────────────────────

interface ComplianceContent {
  title: string
  generated_at: string
  executive_summary: {
    overall_score: number
    status: string
    total_findings: number
    critical: number
    high: number
    medium: number
    low: number
    data_sources_total: number
    data_sources_scanned: number
    classifications_total: number
    active_policies: number
  }
  regulations: Array<{
    name: string
    score: number
    status: string
    findings_count: number
    articles_assessed: string[]
  }>
  findings: Array<{
    id: string
    severity: string
    category: string
    title: string
    description: string
    action: string
    regulation: string
    regulation_article: string
    affected_count: number
    evidence_summary?: string
    severity_reason?: string
  }>
  methodology: string
  assessor: string
}

// ─── Constants ───────────────────────────────────────────────────────────────

const REPORT_TYPES: { value: ReportType; label: string; description: string; icon: typeof FileText }[] = [
  {
    value: 'compliance',
    label: 'Compliance Report',
    description: 'GDPR, CCPA, HIPAA, PCI-DSS, DPDP Act, UAE PDPL & EU AI Act compliance status and gaps',
    icon: ShieldCheck,
  },
  {
    value: 'quality',
    label: 'Data Quality Report',
    description: 'Quality scores, trends, and issues across datasets',
    icon: BarChart3,
  },
  {
    value: 'ai_usage',
    label: 'AI Usage Report',
    description: 'AI Gate activity, blocked queries, and data lineage',
    icon: Bot,
  },
  {
    value: 'audit',
    label: 'Audit Report',
    description: 'Full history of user actions and system events',
    icon: ClipboardList,
  },
]

const SEVERITY_COLORS: Record<string, string> = {
  CRITICAL: 'bg-red-500/10 text-red-500 border-red-500/20',
  HIGH: 'bg-orange-500/10 text-orange-500 border-orange-500/20',
  MEDIUM: 'bg-yellow-500/10 text-yellow-600 border-yellow-500/20',
  LOW: 'bg-blue-500/10 text-blue-500 border-blue-500/20',
}

// ─── Generate Report Modal ────────────────────────────────────────────────────

function GenerateModal({ onClose }: { onClose: () => void }) {
  const [selected, setSelected] = useState<ReportType>('compliance')
  const { mutate: generate, isPending } = useGenerateReport()

  const handleGenerate = () => {
    generate({ type: selected }, { onSuccess: () => onClose() })
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="w-full max-w-lg rounded-xl border border-border bg-card shadow-2xl mx-4">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-border">
          <div>
            <h2 className="text-lg font-semibold text-foreground">Generate Report</h2>
            <p className="text-sm text-muted-foreground mt-0.5">Select the type of report to generate</p>
          </div>
          <button
            onClick={onClose}
            className="rounded-lg p-2 hover:bg-muted transition-colors text-muted-foreground"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Report type grid */}
        <div className="p-6 grid grid-cols-1 sm:grid-cols-2 gap-3">
          {REPORT_TYPES.map(({ value, label, description, icon: Icon }) => (
            <button
              key={value}
              onClick={() => setSelected(value)}
              className={cn(
                'flex items-start gap-3 rounded-lg border p-4 text-left transition-all',
                selected === value
                  ? 'border-primary bg-primary/5 ring-1 ring-primary'
                  : 'border-border hover:bg-muted',
              )}
            >
              <Icon className={cn('h-5 w-5 mt-0.5 flex-shrink-0', selected === value ? 'text-primary' : 'text-muted-foreground')} />
              <div>
                <p className={cn('font-medium text-sm', selected === value ? 'text-primary' : 'text-foreground')}>
                  {label}
                </p>
                <p className="text-xs text-muted-foreground mt-0.5">{description}</p>
              </div>
            </button>
          ))}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-6 pb-6">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded-lg border border-border text-foreground hover:bg-muted transition-colors text-sm"
          >
            Cancel
          </button>
          <button
            onClick={handleGenerate}
            disabled={isPending}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors text-sm disabled:opacity-60 disabled:cursor-not-allowed"
          >
            {isPending ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Generating…
              </>
            ) : (
              <>
                <Plus className="h-4 w-4" />
                Generate
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  )
}

const ALL_FRAMEWORKS = [
  'GDPR', 'CCPA', 'HIPAA', 'PCI-DSS', 'DPDP Act 2023', 'UAE PDPL', 'EU AI Act',
]

// ─── Compliance Report Viewer ─────────────────────────────────────────────────

function ComplianceViewer({ content }: { content: ComplianceContent }) {
  const [openRegulations, setOpenRegulations] = useState<Set<string>>(new Set())
  const [openSeverities, setOpenSeverities] = useState<Set<string>>(new Set(['CRITICAL', 'HIGH']))

  const toggleRegulation = (name: string) => {
    setOpenRegulations((prev) => {
      const next = new Set(prev)
      next.has(name) ? next.delete(name) : next.add(name)
      return next
    })
  }

  const toggleSeverity = (sev: string) => {
    setOpenSeverities((prev) => {
      const next = new Set(prev)
      next.has(sev) ? next.delete(sev) : next.add(sev)
      return next
    })
  }

  const { executive_summary: es, findings } = content
  const score = Math.round(es.overall_score)

  // Merge API-provided regulations with the full list of 7 frameworks
  const apiRegMap = new Map((content.regulations || []).map((r) => [r.name, r]))
  const regulations = ALL_FRAMEWORKS.map((name) =>
    apiRegMap.get(name) ?? {
      name,
      score: 0,
      status: 'Not Assessed',
      findings_count: 0,
      articles_assessed: [] as string[],
    }
  )

  const scoreColor =
    score >= 90 ? 'text-green-500' : score >= 75 ? 'text-yellow-500' : score >= 50 ? 'text-orange-500' : 'text-red-500'

  const findingsBySeverity = ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW'].reduce<Record<string, typeof findings>>(
    (acc, sev) => {
      acc[sev] = findings.filter((f) => f.severity === sev)
      return acc
    },
    {},
  )

  return (
    <div className="space-y-6">
      {/* Executive Summary */}
      <div className="rounded-lg border border-border bg-card p-6">
        <h3 className="text-base font-semibold text-foreground mb-4">Executive Summary</h3>
        <div className="flex items-start gap-6 flex-wrap">
          {/* Score */}
          <div className="flex flex-col items-center justify-center w-28 h-28 rounded-full border-4 border-border flex-shrink-0">
            <span className={cn('text-3xl font-bold', scoreColor)}>{score}</span>
            <span className="text-xs text-muted-foreground mt-0.5">/ 100</span>
          </div>

          {/* Stats grid */}
          <div className="flex-1 grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-lg bg-red-500/10 border border-red-500/20 p-3">
              <p className="text-2xl font-bold text-red-500">{es.critical}</p>
              <p className="text-xs text-muted-foreground mt-0.5">Critical</p>
            </div>
            <div className="rounded-lg bg-orange-500/10 border border-orange-500/20 p-3">
              <p className="text-2xl font-bold text-orange-500">{es.high}</p>
              <p className="text-xs text-muted-foreground mt-0.5">High</p>
            </div>
            <div className="rounded-lg bg-yellow-500/10 border border-yellow-500/20 p-3">
              <p className="text-2xl font-bold text-yellow-600">{es.medium}</p>
              <p className="text-xs text-muted-foreground mt-0.5">Medium</p>
            </div>
            <div className="rounded-lg bg-blue-500/10 border border-blue-500/20 p-3">
              <p className="text-2xl font-bold text-blue-500">{es.low}</p>
              <p className="text-xs text-muted-foreground mt-0.5">Low</p>
            </div>
          </div>
        </div>

        {/* Status row */}
        <div className="mt-4 flex items-center gap-2">
          <span className="text-sm text-muted-foreground">Overall Status:</span>
          <span
            className={cn(
              'text-sm font-medium px-2.5 py-0.5 rounded-full',
              es.status === 'Compliant'
                ? 'bg-green-500/10 text-green-500'
                : es.status === 'Non-Compliant'
                ? 'bg-red-500/10 text-red-500'
                : 'bg-yellow-500/10 text-yellow-600',
            )}
          >
            {es.status}
          </span>
        </div>

        {/* Quick stats */}
        <div className="mt-4 grid grid-cols-2 md:grid-cols-4 gap-3 text-sm">
          <div className="text-muted-foreground">
            Data Sources: <span className="text-foreground font-medium">{es.data_sources_scanned}/{es.data_sources_total} scanned</span>
          </div>
          <div className="text-muted-foreground">
            Classifications: <span className="text-foreground font-medium">{es.classifications_total}</span>
          </div>
          <div className="text-muted-foreground">
            Active Policies: <span className="text-foreground font-medium">{es.active_policies}</span>
          </div>
          <div className="text-muted-foreground">
            Total Findings: <span className="text-foreground font-medium">{es.total_findings}</span>
          </div>
        </div>
      </div>

      {/* Regulations - always show all 7 frameworks */}
      {regulations && regulations.length > 0 && (
        <div className="rounded-lg border border-border bg-card">
          <div className="p-4 border-b border-border">
            <h3 className="text-base font-semibold text-foreground">Regulations Coverage</h3>
            <p className="text-xs text-muted-foreground mt-0.5">All 7 supported frameworks — &ldquo;Not Assessed&rdquo; means no data collected yet</p>
          </div>
          <div className="divide-y divide-border">
            {regulations.map((reg) => (
              <div key={reg.name}>
                <button
                  onClick={() => toggleRegulation(reg.name)}
                  className="w-full flex items-center justify-between p-4 hover:bg-muted transition-colors text-left"
                >
                  <div className="flex items-center gap-3">
                    <span className="font-medium text-sm text-foreground">{reg.name}</span>
                    <span
                      className={cn(
                        'text-xs px-2 py-0.5 rounded-full',
                        reg.status === 'Compliant'
                          ? 'bg-green-500/10 text-green-500'
                          : reg.status === 'Non-Compliant'
                          ? 'bg-red-500/10 text-red-500'
                          : reg.status === 'Not Assessed'
                          ? 'bg-muted text-muted-foreground'
                          : 'bg-yellow-500/10 text-yellow-600',
                      )}
                    >
                      {reg.status}
                    </span>
                  </div>
                  <div className="flex items-center gap-4">
                    <div className="text-right">
                      <p className="text-sm font-semibold text-foreground">{Math.round(reg.score)}/100</p>
                      <p className="text-xs text-muted-foreground">{reg.findings_count} findings</p>
                    </div>
                    {openRegulations.has(reg.name) ? (
                      <ChevronUp className="h-4 w-4 text-muted-foreground" />
                    ) : (
                      <ChevronDown className="h-4 w-4 text-muted-foreground" />
                    )}
                  </div>
                </button>
                {openRegulations.has(reg.name) && reg.articles_assessed && reg.articles_assessed.length > 0 && (
                  <div className="px-4 pb-4 bg-muted/30">
                    <p className="text-xs text-muted-foreground mb-2 font-medium">Articles Assessed</p>
                    <div className="flex flex-wrap gap-1.5">
                      {reg.articles_assessed.map((a) => (
                        <span key={a} className="text-xs px-2 py-0.5 rounded-full bg-muted border border-border text-foreground">
                          {a}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Findings by Severity */}
      {findings && findings.length > 0 && (
        <div className="rounded-lg border border-border bg-card">
          <div className="p-4 border-b border-border">
            <h3 className="text-base font-semibold text-foreground">Findings by Severity</h3>
          </div>
          <div className="divide-y divide-border">
            {['CRITICAL', 'HIGH', 'MEDIUM', 'LOW'].map((sev) => {
              const sevFindings = findingsBySeverity[sev]
              if (!sevFindings?.length) return null
              const isOpen = openSeverities.has(sev)
              return (
                <div key={sev}>
                  <button
                    onClick={() => toggleSeverity(sev)}
                    className="w-full flex items-center justify-between p-4 hover:bg-muted transition-colors"
                  >
                    <div className="flex items-center gap-2">
                      <span className={cn('text-xs font-semibold px-2.5 py-0.5 rounded-full border', SEVERITY_COLORS[sev])}>
                        {sev}
                      </span>
                      <span className="text-sm text-muted-foreground">{sevFindings.length} findings</span>
                    </div>
                    {isOpen ? <ChevronUp className="h-4 w-4 text-muted-foreground" /> : <ChevronDown className="h-4 w-4 text-muted-foreground" />}
                  </button>
                  {isOpen && (
                    <div className="px-4 pb-4 space-y-3">
                      {sevFindings.map((f) => (
                        <div key={f.id} className="rounded-lg border border-border bg-background p-4">
                          <div className="flex items-start justify-between gap-2 flex-wrap">
                            <p className="font-medium text-sm text-foreground">{f.title}</p>
                            {f.regulation && (
                              <span className="text-xs px-2 py-0.5 rounded-full bg-muted border border-border text-muted-foreground flex-shrink-0">
                                {f.regulation}
                              </span>
                            )}
                          </div>
                          {f.regulation_article && (
                            <p className="text-xs text-muted-foreground/70 mt-1 truncate" title={f.regulation_article}>
                              📌 {f.regulation_article}
                            </p>
                          )}
                          <p className="text-sm text-muted-foreground mt-1">{f.description}</p>
                          {f.action && (
                            <p className="text-xs text-primary mt-2">
                              <span className="font-medium">Recommended action:</span> {f.action}
                            </p>
                          )}
                          {f.evidence_summary && (
                            <p className="text-xs text-muted-foreground mt-1">
                              <span className="font-medium">Evidence:</span> {f.evidence_summary}
                            </p>
                          )}
                          {f.affected_count > 0 && (
                            <p className="text-xs text-muted-foreground mt-1">
                              Affected records: <span className="text-foreground font-medium">{f.affected_count}</span>
                            </p>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* Methodology */}
      {content.methodology && (
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground">
            <span className="font-medium text-foreground">Methodology:</span> {content.methodology}
          </p>
          {content.assessor && (
            <p className="text-xs text-muted-foreground mt-1">
              <span className="font-medium text-foreground">Assessor:</span> {content.assessor}
            </p>
          )}
        </div>
      )}

      {/* Print hint */}
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <Printer className="h-3.5 w-3.5" />
        <span>Use browser print (Ctrl/Cmd + P) to export this report as PDF.</span>
      </div>
    </div>
  )
}

// ─── Generic JSON Viewer ─────────────────────────────────────────────────────

function GenericContentViewer({ content }: { content: any }) {
  return (
    <div className="space-y-4">
      <div className="rounded-lg border border-border bg-muted/30 p-4 overflow-x-auto">
        <pre className="text-xs text-foreground whitespace-pre-wrap break-words font-mono">
          {JSON.stringify(content, null, 2)}
        </pre>
      </div>
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <Printer className="h-3.5 w-3.5" />
        <span>Use browser print (Ctrl/Cmd + P) to export this report as PDF.</span>
      </div>
    </div>
  )
}

// ─── Report Detail Panel ─────────────────────────────────────────────────────

function ReportDetailPanel({ report, onClose }: { report: Report; onClose: () => void }) {
  const typeLabel = REPORT_TYPES.find((t) => t.value === report.type)?.label ?? `${report.type} Report`
  const content = report.content

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center md:items-center bg-black/60 backdrop-blur-sm">
      <div className="w-full max-w-3xl max-h-[90vh] flex flex-col rounded-t-xl md:rounded-xl border border-border bg-background shadow-2xl mx-0 md:mx-4">
        {/* Header */}
        <div className="flex items-start justify-between p-6 border-b border-border flex-shrink-0">
          <div>
            <h2 className="text-lg font-semibold text-foreground">{typeLabel}</h2>
            <p className="text-sm text-muted-foreground mt-0.5">
              Generated {new Date(report.generated_at || report.created_at).toLocaleString()}
            </p>
          </div>
          <button
            onClick={onClose}
            className="rounded-lg p-2 hover:bg-muted transition-colors text-muted-foreground flex-shrink-0"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {!content ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground gap-3">
              <AlertCircle className="h-10 w-10" />
              <p className="text-sm">Report content not yet available.</p>
            </div>
          ) : report.type === 'compliance' ? (
            <ComplianceViewer content={content as ComplianceContent} />
          ) : (
            <GenericContentViewer content={content} />
          )}
        </div>
      </div>
    </div>
  )
}

// ─── Status badge ─────────────────────────────────────────────────────────────

function StatusBadge({ status }: { status: Report['status'] }) {
  if (status === 'completed') {
    return (
      <span className="flex items-center gap-1 text-xs text-green-500">
        <CheckCircle2 className="h-3.5 w-3.5" />
        Completed
      </span>
    )
  }
  if (status === 'generating') {
    return (
      <span className="flex items-center gap-1 text-xs text-yellow-600">
        <Clock className="h-3.5 w-3.5" />
        Generating
      </span>
    )
  }
  return (
    <span className="flex items-center gap-1 text-xs text-red-500">
      <AlertTriangle className="h-3.5 w-3.5" />
      Failed
    </span>
  )
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function ReportsPage() {
  const { data: reports, isLoading } = useReports()
  const [showGenerateModal, setShowGenerateModal] = useState(false)
  const [viewingReport, setViewingReport] = useState<Report | null>(null)

  const reportsData: Report[] = Array.isArray(reports) ? reports : []

  return (
    <div className="min-h-screen bg-background">
      {/* Modals */}
      {showGenerateModal && <GenerateModal onClose={() => setShowGenerateModal(false)} />}
      {viewingReport && <ReportDetailPanel report={viewingReport} onClose={() => setViewingReport(null)} />}

      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs
            items={[
              { label: 'Audit', href: '/audit' },
              { label: 'Reports', active: true },
            ]}
          />
          <h1 className="text-3xl font-bold text-foreground mt-4">Reports</h1>
          <p className="text-sm text-muted-foreground mt-1">Generate and view compliance and analytics reports</p>
        </div>
        <button
          onClick={() => setShowGenerateModal(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
        >
          <Plus className="h-5 w-5" />
          Generate Report
        </button>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Report Types quick-access */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {REPORT_TYPES.map(({ value, label, description, icon: Icon }) => (
            <button
              key={value}
              onClick={() => setShowGenerateModal(true)}
              className="rounded-lg border border-border bg-card p-5 text-left hover:border-primary hover:bg-primary/5 transition-all group"
            >
              <Icon className="h-7 w-7 text-primary mb-3 group-hover:scale-110 transition-transform" />
              <h3 className="text-sm font-semibold text-foreground">{label}</h3>
              <p className="text-xs text-muted-foreground mt-1">{description}</p>
            </button>
          ))}
        </div>

        {/* Recent Reports */}
        <div className="rounded-lg border border-border bg-card">
          <div className="p-5 border-b border-border flex items-center justify-between">
            <h3 className="text-base font-semibold text-foreground">Generated Reports</h3>
            {reportsData.length > 0 && (
              <span className="text-xs text-muted-foreground">{reportsData.length} report{reportsData.length !== 1 ? 's' : ''}</span>
            )}
          </div>

          {isLoading ? (
            <div className="p-5 space-y-3">
              <Skeleton className="h-14 w-full" />
              <Skeleton className="h-14 w-full" />
              <Skeleton className="h-14 w-full" />
            </div>
          ) : reportsData.length > 0 ? (
            <div className="divide-y divide-border">
              {reportsData.map((report) => {
                const typeInfo = REPORT_TYPES.find((t) => t.value === report.type)
                const Icon = typeInfo?.icon ?? FileText
                return (
                  <div
                    key={report.id}
                    className="flex items-center justify-between px-5 py-4 hover:bg-muted/40 transition-colors"
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <div className="rounded-lg bg-primary/10 p-2 flex-shrink-0">
                        <Icon className="h-4 w-4 text-primary" />
                      </div>
                      <div className="min-w-0">
                        <p className="font-medium text-sm text-foreground truncate">
                          {typeInfo?.label ?? `${report.type} Report`}
                        </p>
                        <p className="text-xs text-muted-foreground mt-0.5">
                          {new Date(report.generated_at || report.created_at).toLocaleString()}
                        </p>
                      </div>
                    </div>

                    <div className="flex items-center gap-4 flex-shrink-0">
                      <StatusBadge status={report.status} />
                      {report.status === 'completed' && (
                        <button
                          onClick={() => setViewingReport(report)}
                          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-border text-sm text-foreground hover:bg-muted transition-colors"
                        >
                          <FileText className="h-3.5 w-3.5" />
                          View
                        </button>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center py-14 text-center px-6">
              <FileText className="h-12 w-12 text-muted-foreground/30 mb-3" />
              <p className="text-sm font-medium text-foreground">No reports yet</p>
              <p className="text-sm text-muted-foreground mt-1">Click "Generate Report" to create your first compliance report.</p>
              <button
                onClick={() => setShowGenerateModal(true)}
                className="mt-4 flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors text-sm"
              >
                <Plus className="h-4 w-4" />
                Generate Report
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
