'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { 
  FileText, Download, Shield, Database, Users, Clock, 
  CheckCircle, AlertTriangle, BarChart3, FileCheck
} from 'lucide-react'
import { useGenerateDefenseDocket } from '@/hooks/use-advisor'

const regulations = ['GDPR', 'CCPA', 'HIPAA', 'SOC2', 'PCI-DSS', 'DPDP', 'UAE PDPL', 'EU AI Act']

interface DocketSection {
  title: string
  type: string
  data: Record<string, any>
}

interface DocketData {
  generated_at: string
  date_range: { from: string; to: string }
  regulations: string[]
  sections: DocketSection[]
}

function SectionIcon({ type }: { type: string }) {
  const icons: Record<string, React.ReactNode> = {
    classification: <Database className="h-5 w-5 text-blue-500" />,
    policies: <Shield className="h-5 w-5 text-purple-500" />,
    audit: <FileCheck className="h-5 w-5 text-green-500" />,
    dsar: <Users className="h-5 w-5 text-orange-500" />,
    quality: <BarChart3 className="h-5 w-5 text-cyan-500" />,
    retention: <Clock className="h-5 w-5 text-yellow-500" />,
    ropa: <FileText className="h-5 w-5 text-indigo-500" />,
  }
  return icons[type] || <FileText className="h-5 w-5 text-muted-foreground" />
}

function DocketSectionCard({ section }: { section: DocketSection }) {
  const { title, type, data } = section

  const renderContent = () => {
    switch (type) {
      case 'classification':
        return (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-foreground">{data.total_classifications || 0}</p>
              <p className="text-xs text-muted-foreground">Total Classifications</p>
            </div>
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-foreground">{data.pii_detections || 0}</p>
              <p className="text-xs text-muted-foreground">PII Detections</p>
            </div>
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-foreground">{data.datasets_scanned || 0}</p>
              <p className="text-xs text-muted-foreground">Datasets Scanned</p>
            </div>
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-foreground">{(data.coverage_percentage || 0).toFixed(1)}%</p>
              <p className="text-xs text-muted-foreground">Coverage</p>
            </div>
          </div>
        )

      case 'policies':
        return (
          <div className="space-y-4">
            <div className="flex gap-4">
              <div className="text-center p-3 rounded-lg bg-muted/50 flex-1">
                <p className="text-2xl font-bold text-foreground">{data.active_policies || 0}</p>
                <p className="text-xs text-muted-foreground">Active Policies</p>
              </div>
              <div className="text-center p-3 rounded-lg bg-muted/50 flex-1">
                <p className="text-2xl font-bold text-foreground">{data.policy_evaluations || 0}</p>
                <p className="text-xs text-muted-foreground">Policy Evaluations</p>
              </div>
            </div>
            {data.policies?.length > 0 && (
              <div className="space-y-2">
                <p className="text-xs font-medium text-muted-foreground uppercase">Active Policies</p>
                {data.policies.slice(0, 5).map((policy: any, i: number) => (
                  <div key={i} className="flex items-center justify-between p-2 rounded bg-muted/30">
                    <span className="text-sm text-foreground">{policy.name}</span>
                    <span className="text-xs px-2 py-0.5 rounded bg-green-500/10 text-green-600">{policy.type}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )

      case 'audit':
        return (
          <div className="space-y-4">
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-foreground">{data.total_events || 0}</p>
              <p className="text-xs text-muted-foreground">Audit Events in Period</p>
            </div>
            {data.audit_entries?.length > 0 && (
              <div className="space-y-2 max-h-48 overflow-y-auto">
                <p className="text-xs font-medium text-muted-foreground uppercase">Recent Events</p>
                {data.audit_entries.slice(0, 10).map((entry: any, i: number) => (
                  <div key={i} className="flex items-center justify-between p-2 rounded bg-muted/30 text-xs">
                    <span className="text-foreground">{entry.action}</span>
                    <span className="text-muted-foreground">{new Date(entry.timestamp).toLocaleDateString()}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )

      case 'dsar':
        return (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-foreground">{data.total_requests || 0}</p>
              <p className="text-xs text-muted-foreground">Total Requests</p>
            </div>
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-green-600">{data.completed_requests || 0}</p>
              <p className="text-xs text-muted-foreground">Completed</p>
            </div>
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-yellow-600">{data.pending_requests || 0}</p>
              <p className="text-xs text-muted-foreground">Pending</p>
            </div>
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-foreground">{(data.compliance_rate || 0).toFixed(1)}%</p>
              <p className="text-xs text-muted-foreground">Compliance Rate</p>
            </div>
          </div>
        )

      case 'quality':
        return (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-foreground">{(data.overall_score || 0).toFixed(1)}%</p>
              <p className="text-xs text-muted-foreground">Overall Score</p>
            </div>
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-foreground">{(data.average_completeness || 0).toFixed(1)}%</p>
              <p className="text-xs text-muted-foreground">Completeness</p>
            </div>
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-foreground">{(data.average_accuracy || 0).toFixed(1)}%</p>
              <p className="text-xs text-muted-foreground">Accuracy</p>
            </div>
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-foreground">{(data.average_consistency || 0).toFixed(1)}%</p>
              <p className="text-xs text-muted-foreground">Consistency</p>
            </div>
          </div>
        )

      case 'retention':
        return (
          <div className="grid grid-cols-3 gap-4">
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-foreground">{data.total_violations || 0}</p>
              <p className="text-xs text-muted-foreground">Total Violations</p>
            </div>
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-green-600">{data.resolved_violations || 0}</p>
              <p className="text-xs text-muted-foreground">Resolved</p>
            </div>
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-foreground">{(data.compliance_rate || 100).toFixed(1)}%</p>
              <p className="text-xs text-muted-foreground">Compliance Rate</p>
            </div>
          </div>
        )

      case 'ropa':
        return (
          <div className="space-y-4">
            <div className="text-center p-3 rounded-lg bg-muted/50">
              <p className="text-2xl font-bold text-foreground">{data.total_records || 0}</p>
              <p className="text-xs text-muted-foreground">Processing Activity Records</p>
            </div>
            {data.entries?.length > 0 && (
              <div className="space-y-2">
                <p className="text-xs font-medium text-muted-foreground uppercase">Recent Entries</p>
                {data.entries.slice(0, 5).map((entry: any, i: number) => (
                  <div key={i} className="p-2 rounded bg-muted/30">
                    <p className="text-sm font-medium text-foreground">{entry.name}</p>
                    <p className="text-xs text-muted-foreground">{entry.purpose}</p>
                  </div>
                ))}
              </div>
            )}
          </div>
        )

      default:
        return (
          <pre className="p-4 rounded-lg bg-muted text-xs overflow-auto">
            {JSON.stringify(data, null, 2)}
          </pre>
        )
    }
  }

  return (
    <div className="rounded-lg border border-border bg-card p-6">
      <div className="flex items-center gap-3 mb-4">
        <SectionIcon type={type} />
        <h4 className="text-lg font-semibold text-foreground">{title}</h4>
      </div>
      {renderContent()}
    </div>
  )
}

export default function DefenseDocketPage() {
  const generateDocket = useGenerateDefenseDocket()
  const [selectedRegs, setSelectedRegs] = useState<string[]>(['GDPR'])
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')

  const handleGenerate = async () => {
    if (selectedRegs.length === 0) return
    await generateDocket.mutateAsync({
      regulations: selectedRegs,
      date_from: dateFrom || new Date(Date.now() - 90 * 24 * 60 * 60 * 1000).toISOString(),
      date_to: dateTo || new Date().toISOString(),
    })
  }

  const toggleReg = (reg: string) => {
    setSelectedRegs(prev =>
      prev.includes(reg) ? prev.filter(r => r !== reg) : [...prev, reg]
    )
  }

  const docketData = generateDocket.data as DocketData | undefined

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Compliance Advisor', href: '/advisor' },
            { label: 'Defense Docket', active: true },
          ]}
        />
        <h1 className="text-3xl font-bold text-foreground mt-4">Defense Docket</h1>
        <p className="text-sm text-muted-foreground mt-1">Generate audit-ready compliance documentation</p>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Configuration */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Generate Defense Docket</h3>
          
          {/* Regulations */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-foreground mb-2">Select Regulations</label>
            <div className="flex flex-wrap gap-2">
              {regulations.map((reg) => (
                <button
                  key={reg}
                  onClick={() => toggleReg(reg)}
                  className={`px-4 py-2 rounded-lg border transition-colors ${
                    selectedRegs.includes(reg)
                      ? 'border-primary bg-primary/10 text-primary'
                      : 'border-border text-foreground hover:border-primary/50'
                  }`}
                >
                  {reg}
                </button>
              ))}
            </div>
          </div>

          {/* Date Range */}
          <div className="grid grid-cols-2 gap-4 mb-6">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">From Date</label>
              <input
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
                className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">To Date</label>
              <input
                type="date"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
                className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>
          </div>

          <button
            onClick={handleGenerate}
            disabled={generateDocket.isPending || selectedRegs.length === 0}
            className="flex items-center gap-2 px-6 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <FileText className="h-4 w-4" />
            {generateDocket.isPending ? 'Generating...' : 'Generate Docket'}
          </button>
        </div>

        {/* Loading State */}
        {generateDocket.isPending && (
          <div className="space-y-4">
            <Skeleton className="h-32 w-full" />
            <Skeleton className="h-32 w-full" />
            <Skeleton className="h-32 w-full" />
          </div>
        )}

        {/* Result */}
        {docketData && !generateDocket.isPending && (
          <div className="space-y-6">
            {/* Header */}
            <div className="rounded-lg border border-primary/20 bg-primary/5 p-6">
              <div className="flex items-center justify-between">
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <CheckCircle className="h-5 w-5 text-green-500" />
                    <h3 className="text-lg font-semibold text-foreground">Defense Docket Generated</h3>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Generated on {new Date(docketData.generated_at).toLocaleString()} for {docketData.regulations?.join(', ')}
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">
                    Date range: {new Date(docketData.date_range?.from).toLocaleDateString()} - {new Date(docketData.date_range?.to).toLocaleDateString()}
                  </p>
                </div>
                <button className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border text-foreground hover:bg-muted transition-colors">
                  <Download className="h-4 w-4" />
                  Download PDF
                </button>
              </div>
            </div>

            {/* Sections */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {docketData.sections?.map((section, i) => (
                <DocketSectionCard key={i} section={section} />
              ))}
            </div>
          </div>
        )}

        {/* Info */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">What&apos;s Included</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <h4 className="font-medium text-foreground mb-2">Compliance Evidence</h4>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• Policy configurations and enforcement logs</li>
                <li>• Data classification summaries</li>
                <li>• Access control audit trails</li>
                <li>• DSAR processing records</li>
              </ul>
            </div>
            <div>
              <h4 className="font-medium text-foreground mb-2">Risk Assessment</h4>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• Data quality scores and trends</li>
                <li>• Identified gaps and remediation status</li>
                <li>• AI governance compliance</li>
                <li>• Retention policy adherence</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
