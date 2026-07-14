'use client'

import { useMemo } from 'react'
import { StatCard } from '@/components/base/stat-card'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import Link from 'next/link'
import { Shield, FileCheck, AlertTriangle, Zap } from 'lucide-react'
import { usePolicies } from '@/hooks/use-policies'
import { useRiskScore, useComplianceGaps } from '@/hooks/use-advisor'

export default function GovernancePage() {
  const { data: policies, isLoading: policiesLoading } = usePolicies()
  const { data: riskScore, isLoading: riskLoading } = useRiskScore()
  const { data: gaps, isLoading: gapsLoading } = useComplianceGaps()

  const stats = useMemo(() => {
    const policiesArray = Array.isArray(policies) ? policies : []
    const activePolicies = policiesArray.filter(p => p.active).length
    const totalPolicies = policiesArray.length
    const complianceScore = riskScore?.overall_score ? Math.round(riskScore.overall_score * 100) : 0
    const openGaps = Array.isArray(gaps) ? gaps.filter((g: any) => g.status !== 'resolved').length : 0

    return { activePolicies, totalPolicies, complianceScore, openGaps }
  }, [policies, riskScore, gaps])

  const isLoading = policiesLoading || riskLoading || gapsLoading

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Governance', active: true }]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">Governance</h1>
        <p className="text-sm text-muted-foreground mt-1">Manage policies, compliance, and data governance</p>
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
                label="Active Policies"
                value={`${stats.activePolicies}/${stats.totalPolicies}`}
                icon={<Shield className="h-6 w-6" />}
              />
              <StatCard
                label="Compliance Score"
                value={`${stats.complianceScore}%`}
                change={stats.complianceScore >= 80 ? 1 : -1}
                changeLabel={stats.complianceScore >= 80 ? 'healthy' : 'needs attention'}
                icon={<FileCheck className="h-6 w-6" />}
              />
              <StatCard
                label="Open Gaps"
                value={stats.openGaps.toString()}
                icon={<AlertTriangle className="h-6 w-6" />}
              />
              <StatCard
                label="Policy Evaluations"
                value="24/7"
                icon={<Zap className="h-6 w-6" />}
              />
            </>
          )}
        </div>

        {/* Quick Links */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <Link
            href="/governance/policies"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <Shield className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Policies</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Create and manage access, redaction, AI, and retention policies
            </p>
            <p className="text-sm text-primary mt-4">{stats.totalPolicies} policies configured</p>
          </Link>

          <Link
            href="/governance/evaluate"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <Zap className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Policy Evaluation</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Test and evaluate policies against sample data
            </p>
            <p className="text-sm text-primary mt-4">Real-time evaluation</p>
          </Link>

          <Link
            href="/advisor"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <FileCheck className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Compliance Advisor</h3>
            <p className="text-sm text-muted-foreground mt-1">
              AI-powered compliance recommendations and gap analysis
            </p>
            <p className="text-sm text-primary mt-4">{stats.openGaps} gaps to address</p>
          </Link>
        </div>

        {/* Recent Policy Activity */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Recent Policies</h3>
          {policiesLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : policies && policies.length > 0 ? (
            <div className="space-y-3">
              {policies.slice(0, 5).map((policy) => (
                <Link
                  key={policy.id}
                  href={`/governance/policies/${policy.id}`}
                  className="flex items-center justify-between p-3 rounded-lg hover:bg-muted transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <div className={`w-2 h-2 rounded-full ${policy.active ? 'bg-green-500' : 'bg-gray-400'}`} />
                    <span className="font-medium text-foreground">{policy.name}</span>
                    <span className="px-2 py-0.5 rounded bg-muted text-xs text-muted-foreground capitalize">
                      {policy.type}
                    </span>
                  </div>
                  <span className="text-sm text-muted-foreground">
                    {new Date(policy.updated_at).toLocaleDateString()}
                  </span>
                </Link>
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <p className="text-muted-foreground">No policies configured</p>
              <Link href="/governance/policies/new" className="text-primary hover:underline text-sm mt-2 inline-block">
                Create your first policy
              </Link>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
