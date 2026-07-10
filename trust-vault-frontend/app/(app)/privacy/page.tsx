'use client'

import { useMemo } from 'react'
import { StatCard } from '@/components/base/stat-card'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import Link from 'next/link'
import { Shield, FileText, Users, Clock } from 'lucide-react'
import { useDSARs, useRoPA, useRetentionViolations } from '@/hooks/use-privacy'

export default function PrivacyPage() {
  const { data: dsars, isLoading: dsarsLoading } = useDSARs()
  const { data: ropa, isLoading: ropaLoading } = useRoPA()
  const { data: violations, isLoading: violationsLoading } = useRetentionViolations()

  const stats = useMemo(() => {
    const openDsars = Array.isArray(dsars) ? dsars.filter((d: any) => d.status !== 'completed').length : 0
    const totalDsars = Array.isArray(dsars) ? dsars.length : 0
    const ropaEntries = Array.isArray(ropa) ? ropa.length : 0
    const retentionViolations = Array.isArray(violations) ? violations.length : 0

    return { openDsars, totalDsars, ropaEntries, retentionViolations }
  }, [dsars, ropa, violations])

  const isLoading = dsarsLoading || ropaLoading || violationsLoading

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Privacy', active: true }]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">Privacy Compliance</h1>
        <p className="text-sm text-muted-foreground mt-1">Manage GDPR, CCPA, and other privacy requirements</p>
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
                label="Open DSARs"
                value={`${stats.openDsars}/${stats.totalDsars}`}
                icon={<FileText className="h-6 w-6" />}
              />
              <StatCard
                label="RoPA Entries"
                value={stats.ropaEntries.toString()}
                icon={<Shield className="h-6 w-6" />}
              />
              <StatCard
                label="Retention Violations"
                value={stats.retentionViolations.toString()}
                change={stats.retentionViolations > 0 ? -1 : 1}
                changeLabel={stats.retentionViolations > 0 ? 'needs attention' : 'compliant'}
                icon={<Clock className="h-6 w-6" />}
              />
              <StatCard
                label="Consent Records"
                value="Active"
                icon={<Users className="h-6 w-6" />}
              />
            </>
          )}
        </div>

        {/* Quick Links */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <Link
            href="/privacy/dsar"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <FileText className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">DSAR Management</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Handle data subject access, deletion, and rectification requests
            </p>
            <p className="text-sm text-primary mt-4">{stats.openDsars} open requests</p>
          </Link>

          <Link
            href="/privacy/consent"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <Users className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Consent Management</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Track and manage user consent for data processing
            </p>
            <p className="text-sm text-primary mt-4">View consent records</p>
          </Link>

          <Link
            href="/governance/policies?type=retention"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <Clock className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Retention Policies</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Define and enforce data retention periods
            </p>
            <p className="text-sm text-primary mt-4">{stats.retentionViolations} violations</p>
          </Link>
        </div>

        {/* Recent DSARs */}
        <div className="rounded-lg border border-border bg-card p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-foreground">Recent DSARs</h3>
            <Link href="/privacy/dsar" className="text-sm text-primary hover:underline">
              View all
            </Link>
          </div>
          {dsarsLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : dsars && Array.isArray(dsars) && dsars.length > 0 ? (
            <div className="space-y-3">
              {dsars.slice(0, 5).map((dsar: any) => (
                <Link
                  key={dsar.id}
                  href={`/privacy/dsar/${dsar.id}`}
                  className="flex items-center justify-between p-3 rounded-lg hover:bg-muted transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <div className={`w-2 h-2 rounded-full ${
                      dsar.status === 'completed' ? 'bg-green-500' : 
                      dsar.status === 'in_progress' ? 'bg-yellow-500' : 'bg-gray-400'
                    }`} />
                    <span className="font-medium text-foreground">{dsar.subject_id}</span>
                    <span className="px-2 py-0.5 rounded bg-muted text-xs text-muted-foreground capitalize">
                      {dsar.type}
                    </span>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className="text-sm text-muted-foreground">
                      Due: {new Date(dsar.deadline).toLocaleDateString()}
                    </span>
                    <span className={`text-sm ${
                      dsar.status === 'completed' ? 'text-green-600' : 
                      dsar.status === 'in_progress' ? 'text-yellow-600' : 'text-muted-foreground'
                    }`}>
                      {dsar.status}
                    </span>
                  </div>
                </Link>
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <p className="text-muted-foreground">No DSARs recorded</p>
              <Link href="/privacy/dsar" className="text-primary hover:underline text-sm mt-2 inline-block">
                Create a DSAR
              </Link>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
