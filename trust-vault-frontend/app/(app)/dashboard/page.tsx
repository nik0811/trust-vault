'use client'

import { useMemo } from 'react'
import Link from 'next/link'
import { StatCard } from '@/components/base/stat-card'
import { DataTable, type Column } from '@/components/base/data-table'
import { StatusIndicator } from '@/components/base/status-badge'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { BarChart3, Database, Zap, CheckCircle, AlertTriangle, Shield } from 'lucide-react'
import { useDataSources } from '@/hooks/use-datasources'
import { usePolicies } from '@/hooks/use-policies'
import { useGateStats } from '@/hooks/use-gate'
import { useAlerts, useSystemHealth } from '@/hooks/use-audit'
import { useRiskScore, useComplianceGaps, useRecommendations } from '@/hooks/use-advisor'

interface Alert {
  id: string
  severity: 'critical' | 'warning' | 'info'
  message: string
  source: string
  timestamp: Date
}

const alertColumns: Column<Alert>[] = [
  {
    id: 'severity',
    header: 'Severity',
    cell: (row) => <StatusIndicator status={row.severity === 'critical' ? 'error' : row.severity === 'warning' ? 'warning' : 'info'} label={row.severity} />,
  },
  {
    id: 'message',
    header: 'Message',
    accessorKey: 'message',
  },
  {
    id: 'source',
    header: 'Source',
    accessorKey: 'source',
  },
  {
    id: 'timestamp',
    header: 'Time',
    cell: (row) => new Date(row.timestamp).toLocaleString(),
  },
]

export default function DashboardPage() {
  const { data: dataSources, isLoading: dsLoading } = useDataSources()
  const { data: policies, isLoading: policiesLoading } = usePolicies()
  const { data: gateStats, isLoading: gateLoading } = useGateStats()
  const { data: alerts, isLoading: alertsLoading } = useAlerts()
  const { data: riskScore, isLoading: riskLoading } = useRiskScore()
  const { data: gaps, isLoading: gapsLoading } = useComplianceGaps()
  const { data: recommendations, isLoading: recsLoading } = useRecommendations()
  const { data: health } = useSystemHealth()

  const stats = useMemo(() => {
    const dsArray = Array.isArray(dataSources) ? dataSources : []
    const policiesArray = Array.isArray(policies) ? policies : []
    const connectedSources = dsArray.filter(ds => ds.status === 'connected').length
    const totalSources = dsArray.length
    const activePolicies = policiesArray.filter(p => p.active).length
    const complianceScore = riskScore?.overall_score ? Math.round(riskScore.overall_score * 100) : 0

    return [
      { 
        label: 'Data Sources', 
        value: `${connectedSources}/${totalSources}`, 
        change: connectedSources, 
        changeLabel: 'connected', 
        icon: <Database className="h-6 w-6" />,
        loading: dsLoading
      },
      { 
        label: 'AI Gate Queries', 
        value: gateStats?.total_queries?.toLocaleString() || '0', 
        change: gateStats?.blocked_queries || 0, 
        changeLabel: 'blocked', 
        icon: <Zap className="h-6 w-6" />,
        loading: gateLoading
      },
      { 
        label: 'Active Policies', 
        value: activePolicies.toString(), 
        change: policiesArray.length, 
        changeLabel: 'total', 
        icon: <CheckCircle className="h-6 w-6" />,
        loading: policiesLoading
      },
      { 
        label: 'Compliance Score', 
        value: `${complianceScore}%`, 
        change: complianceScore >= 80 ? 1 : -1, 
        changeLabel: complianceScore >= 80 ? 'healthy' : 'needs attention', 
        icon: <BarChart3 className="h-6 w-6" />,
        loading: riskLoading
      },
    ]
  }, [dataSources, policies, gateStats, riskScore, dsLoading, policiesLoading, gateLoading, riskLoading])

  const formattedAlerts: Alert[] = useMemo(() => {
    if (!alerts || !Array.isArray(alerts)) return []
    return alerts.slice(0, 5).map((alert: any) => ({
      id: alert.id,
      severity: alert.severity || 'info',
      message: alert.title || alert.message,
      source: alert.resource || 'system',
      timestamp: new Date(alert.created_at),
    }))
  }, [alerts])

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Dashboard', active: true }]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">Dashboard</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Welcome back! Here&apos;s your data governance overview.
          {health?.status === 'healthy' && (
            <span className="ml-2 inline-flex items-center gap-1 text-green-600">
              <span className="h-2 w-2 rounded-full bg-green-500" />
              All systems healthy
            </span>
          )}
        </p>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Stats grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {stats.map((stat) => (
            stat.loading ? (
              <div key={stat.label} className="rounded-lg border border-border bg-card p-6">
                <Skeleton className="h-4 w-24 mb-2" />
                <Skeleton className="h-8 w-16" />
              </div>
            ) : (
              <StatCard
                key={stat.label}
                label={stat.label}
                value={stat.value}
                change={stat.change}
                changeLabel={stat.changeLabel}
                icon={stat.icon}
              />
            )
          ))}
        </div>

        {/* Two column layout */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Left column - Main content */}
          <div className="lg:col-span-2 space-y-8">
            {/* Recent Alerts */}
            <div className="rounded-lg border border-border bg-card p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-foreground">Recent Alerts</h3>
                <Link href="/observability/alerts" className="text-sm text-primary hover:underline">
                  View all
                </Link>
              </div>
              {alertsLoading ? (
                <div className="space-y-3">
                  {[1, 2, 3].map(i => (
                    <Skeleton key={i} className="h-12 w-full" />
                  ))}
                </div>
              ) : formattedAlerts.length > 0 ? (
                <DataTable columns={alertColumns} data={formattedAlerts} />
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  <CheckCircle className="h-12 w-12 mx-auto mb-2 opacity-50" />
                  <p>No alerts - everything looks good!</p>
                </div>
              )}
            </div>

            {/* AI Gate Activity */}
            <div className="rounded-lg border border-border bg-card p-6">
              <h3 className="text-lg font-semibold text-foreground mb-4">AI Gate Activity</h3>
              {gateLoading ? (
                <Skeleton className="h-32 w-full" />
              ) : (
                <div className="grid grid-cols-3 gap-4">
                  <div className="text-center p-4 rounded-lg bg-muted/50">
                    <p className="text-2xl font-bold text-foreground">{gateStats?.total_queries || 0}</p>
                    <p className="text-sm text-muted-foreground">Total Queries</p>
                  </div>
                  <div className="text-center p-4 rounded-lg bg-green-500/10">
                    <p className="text-2xl font-bold text-green-600">{gateStats?.allowed_queries || 0}</p>
                    <p className="text-sm text-muted-foreground">Allowed</p>
                  </div>
                  <div className="text-center p-4 rounded-lg bg-red-500/10">
                    <p className="text-2xl font-bold text-red-600">{gateStats?.blocked_queries || 0}</p>
                    <p className="text-sm text-muted-foreground">Blocked</p>
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Right column - Sidebar metrics */}
          <div className="space-y-6">
            {/* Compliance Overview */}
            <div className="rounded-lg border border-border bg-card p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-foreground">Compliance</h3>
                <Link href="/governance" className="text-sm text-primary hover:underline">
                  View details
                </Link>
              </div>
              {riskLoading || gapsLoading ? (
                <div className="space-y-3">
                  <Skeleton className="h-16 w-full" />
                  <Skeleton className="h-8 w-full" />
                </div>
              ) : (
                <div className="space-y-4">
                  <div className="flex items-center gap-4">
                    <div className="relative h-16 w-16">
                      <svg className="h-16 w-16 -rotate-90" viewBox="0 0 36 36">
                        <path
                          d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                          fill="none"
                          stroke="currentColor"
                          strokeWidth="3"
                          className="text-muted"
                        />
                        <path
                          d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                          fill="none"
                          stroke="currentColor"
                          strokeWidth="3"
                          strokeDasharray={`${(riskScore?.overall_score || 0) * 100}, 100`}
                          className={riskScore?.overall_score >= 0.8 ? 'text-green-500' : riskScore?.overall_score >= 0.6 ? 'text-yellow-500' : 'text-red-500'}
                        />
                      </svg>
                      <span className="absolute inset-0 flex items-center justify-center text-sm font-bold">
                        {Math.round((riskScore?.overall_score || 0) * 100)}%
                      </span>
                    </div>
                    <div>
                      <p className="font-medium text-foreground">
                        {riskScore?.risk_level === 'low' ? 'Good Standing' : 
                         riskScore?.risk_level === 'medium' ? 'Needs Attention' : 
                         riskScore?.risk_level === 'high' ? 'At Risk' : 'Critical'}
                      </p>
                      <p className="text-sm text-muted-foreground">
                        {Array.isArray(gaps) ? gaps.filter((g: any) => g.status !== 'resolved').length : 0} open gaps
                      </p>
                    </div>
                  </div>
                  {Array.isArray(recommendations) && recommendations.length > 0 && (
                    <div className="border-t border-border pt-3">
                      <p className="text-xs text-muted-foreground mb-2">Top Recommendation</p>
                      <p className="text-sm text-foreground">{recommendations[0]?.title || recommendations[0]?.description}</p>
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Data Sources Status */}
            <div className="rounded-lg border border-border bg-card p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-foreground">Data Sources</h3>
                <Link href="/data-sources" className="text-sm text-primary hover:underline">
                  Manage
                </Link>
              </div>
              {dsLoading ? (
                <div className="space-y-2">
                  {[1, 2, 3].map(i => <Skeleton key={i} className="h-8 w-full" />)}
                </div>
              ) : (
                <div className="space-y-2">
                  {Array.isArray(dataSources) && dataSources.slice(0, 5).map(ds => (
                    <div key={ds.id} className="flex items-center justify-between py-2">
                      <span className="text-sm text-foreground truncate">{ds.name}</span>
                      <StatusIndicator 
                        status={ds.status === 'connected' ? 'success' : ds.status === 'scanning' ? 'pending' : 'error'} 
                        label={ds.status} 
                      />
                    </div>
                  ))}
                  {(!Array.isArray(dataSources) || dataSources.length === 0) && (
                    <p className="text-sm text-muted-foreground text-center py-4">No data sources connected</p>
                  )}
                </div>
              )}
            </div>

            {/* Quick Actions */}
            <div className="rounded-lg border border-border bg-card p-6">
              <h3 className="text-lg font-semibold text-foreground mb-4">Quick Actions</h3>
              <div className="space-y-2">
                <Link href="/data-sources/new" className="block w-full px-4 py-2 rounded-lg bg-primary/10 text-primary hover:bg-primary/20 transition-colors text-sm font-medium text-left">
                  Add Data Source
                </Link>
                <Link href="/governance/policies/new" className="block w-full px-4 py-2 rounded-lg bg-primary/10 text-primary hover:bg-primary/20 transition-colors text-sm font-medium text-left">
                  Create Policy
                </Link>
                <Link href="/ai-gate/playground" className="block w-full px-4 py-2 rounded-lg bg-primary/10 text-primary hover:bg-primary/20 transition-colors text-sm font-medium text-left">
                  Test AI Gate
                </Link>
                <Link href="/audit/reports" className="block w-full px-4 py-2 rounded-lg bg-primary/10 text-primary hover:bg-primary/20 transition-colors text-sm font-medium text-left">
                  View Reports
                </Link>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
