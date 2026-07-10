'use client'

import { useMemo } from 'react'
import { StatCard } from '@/components/base/stat-card'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import Link from 'next/link'
import { Tag, Shield, FileText, Settings } from 'lucide-react'
import { useLabelSummary } from '@/hooks/use-advisor'

export default function LabelsPage() {
  const { data: summary, isLoading } = useLabelSummary()

  const stats = useMemo(() => {
    if (!summary) return { total: 0, public: 0, internal: 0, confidential: 0, restricted: 0 }
    return {
      total: summary.total || 0,
      public: summary.public || 0,
      internal: summary.internal || 0,
      confidential: summary.confidential || 0,
      restricted: summary.restricted || 0,
    }
  }, [summary])

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Sensitivity Labels', active: true }]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">Sensitivity Labels</h1>
        <p className="text-sm text-muted-foreground mt-1">Automatic and manual data sensitivity labeling</p>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-5 gap-4">
          {isLoading ? (
            <>
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
            </>
          ) : (
            <>
              <StatCard label="Total Labeled" value={stats.total.toString()} icon={<Tag className="h-6 w-6" />} />
              <StatCard label="Public" value={stats.public.toString()} />
              <StatCard label="Internal" value={stats.internal.toString()} />
              <StatCard label="Confidential" value={stats.confidential.toString()} />
              <StatCard label="Restricted" value={stats.restricted.toString()} />
            </>
          )}
        </div>

        {/* Quick Links */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <Link
            href="/labels/datasets"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <FileText className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Dataset Labels</h3>
            <p className="text-sm text-muted-foreground mt-1">
              View and manage labels assigned to datasets
            </p>
          </Link>

          <Link
            href="/labels/rules"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <Settings className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Label Rules</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Configure automatic label assignment rules
            </p>
          </Link>

          <div className="rounded-lg border border-border bg-card p-6">
            <Shield className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Label Hierarchy</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Understanding sensitivity levels
            </p>
            <div className="mt-4 space-y-2">
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-green-500" />
                <span className="text-sm text-foreground">Public</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-blue-500" />
                <span className="text-sm text-foreground">Internal</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-yellow-500" />
                <span className="text-sm text-foreground">Confidential</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-red-500" />
                <span className="text-sm text-foreground">Restricted</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
