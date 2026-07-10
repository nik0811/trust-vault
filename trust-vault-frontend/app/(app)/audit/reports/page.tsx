'use client'

import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { FileText, Download, Plus } from 'lucide-react'
import { useReports, useAnalyticsSummary } from '@/hooks/use-datamap'
import Link from 'next/link'

export default function ReportsPage() {
  const { data: reports, isLoading: reportsLoading } = useReports()
  const { data: analytics, isLoading: analyticsLoading } = useAnalyticsSummary()

  const reportsData = Array.isArray(reports) ? reports : []

  return (
    <div className="min-h-screen bg-background">
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
        <button className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors">
          <Plus className="h-5 w-5" />
          Generate Report
        </button>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Report Types */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="rounded-lg border border-border bg-card p-6">
            <FileText className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Compliance Report</h3>
            <p className="text-sm text-muted-foreground mt-1">
              GDPR, CCPA, HIPAA compliance status and gaps
            </p>
            <button className="mt-4 text-sm text-primary hover:underline">Generate</button>
          </div>
          <div className="rounded-lg border border-border bg-card p-6">
            <FileText className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Data Quality Report</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Quality scores, trends, and issues across datasets
            </p>
            <button className="mt-4 text-sm text-primary hover:underline">Generate</button>
          </div>
          <div className="rounded-lg border border-border bg-card p-6">
            <FileText className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">AI Usage Report</h3>
            <p className="text-sm text-muted-foreground mt-1">
              AI Gate activity, blocked queries, and data lineage
            </p>
            <button className="mt-4 text-sm text-primary hover:underline">Generate</button>
          </div>
        </div>

        {/* Recent Reports */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Recent Reports</h3>
          {reportsLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : reportsData.length > 0 ? (
            <div className="space-y-3">
              {reportsData.map((report: any) => (
                <div
                  key={report.id}
                  className="flex items-center justify-between p-3 rounded-lg hover:bg-muted transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <FileText className="h-5 w-5 text-muted-foreground" />
                    <div>
                      <p className="font-medium text-foreground capitalize">{report.type} Report</p>
                      <p className="text-sm text-muted-foreground">
                        Generated {new Date(report.generated_at).toLocaleString()}
                      </p>
                    </div>
                  </div>
                  <button className="flex items-center gap-2 px-3 py-1 rounded-lg border border-border text-foreground hover:bg-muted transition-colors">
                    <Download className="h-4 w-4" />
                    Download
                  </button>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <p className="text-muted-foreground">No reports generated yet</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
