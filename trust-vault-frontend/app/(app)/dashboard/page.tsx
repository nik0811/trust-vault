'use client'

import { useMemo } from 'react'
import Link from 'next/link'
import { StatCard } from '@/components/base/stat-card'
import { DataTable, type Column } from '@/components/base/data-table'
import { StatusIndicator } from '@/components/base/status-badge'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { BarChart3, Database, Zap, CheckCircle, AlertTriangle, Shield, FileText, Activity, TrendingUp, Eye, Lightbulb } from 'lucide-react'
import { useDataSources } from '@/hooks/use-datasources'
import { usePolicies } from '@/hooks/use-policies'
import { useGateStats } from '@/hooks/use-gate'
import { useAlerts, useSystemHealth, useAuditTrail } from '@/hooks/use-audit'
import { useRiskScore, useComplianceGaps, useRecommendations } from '@/hooks/use-advisor'
import { useQualityTrends } from '@/hooks/use-quality'
import { useAnalyticsSummary } from '@/hooks/use-datamap'

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
  const { data: auditTrail, isLoading: auditLoading } = useAuditTrail({ limit: 10 })
  const { data: qualityTrends, isLoading: qualityLoading } = useQualityTrends()
  const { data: analytics, isLoading: analyticsLoading } = useAnalyticsSummary()

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
        label: 'Records Scanned', 
        value: analytics?.total_records?.toLocaleString() || '0', 
        change: analytics?.pii_detected || 0, 
        changeLabel: 'PII detected', 
        icon: <Eye className="h-6 w-6" />,
        loading: analyticsLoading
      },
      { 
        label: 'Compliance Score', 
        value: `${complianceScore}%`, 
        change: complianceScore >= 80 ? 1 : -1, 
        changeLabel: complianceScore >= 80 ? 'healthy' : 'needs attention', 
        icon: <Shield className="h-6 w-6" />,
        loading: riskLoading
      },
    ]
  }, [dataSources, policies, gateStats, riskScore, analytics, dsLoading, policiesLoading, gateLoading, riskLoading, analyticsLoading])

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
          <div className="lg:col-span-2 space-y-6">
            {/* Classification Overview & Processing Stats Row */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {/* Classification Overview */}
              <div className="rounded-lg border border-border bg-card p-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-semibold text-foreground">Classification Overview</h3>
                  <Link href="/classification" className="text-sm text-primary hover:underline">
                    Details
                  </Link>
                </div>
                {analyticsLoading ? (
                  <div className="space-y-3">
                    <Skeleton className="h-8 w-full" />
                    <Skeleton className="h-20 w-full" />
                  </div>
                ) : (
                  <div className="space-y-4">
                    <div className="flex items-baseline gap-2">
                      <span className="text-3xl font-bold text-foreground">{analytics?.columns_classified?.toLocaleString() || 0}</span>
                      <span className="text-sm text-muted-foreground">columns classified</span>
                    </div>
                    <div className="space-y-2">
                      <p className="text-xs text-muted-foreground font-medium">Top Entity Types</p>
                      <div className="flex flex-wrap gap-2">
                        {(analytics?.top_entities || [{ type: 'EMAIL', count: 0 }, { type: 'PHONE', count: 0 }, { type: 'SSN', count: 0 }]).slice(0, 4).map((entity: any) => (
                          <span key={entity.type} className="inline-flex items-center gap-1 px-2 py-1 rounded-md bg-primary/10 text-xs font-medium text-primary">
                            {entity.type}
                            <span className="text-muted-foreground">({entity.count})</span>
                          </span>
                        ))}
                      </div>
                    </div>
                    <div className="pt-2 border-t border-border">
                      <div className="flex justify-between text-sm">
                        <span className="text-muted-foreground">Avg Confidence</span>
                        <span className="font-medium text-foreground">{analytics?.avg_confidence ? `${Math.round(analytics.avg_confidence * 100)}%` : '-'}</span>
                      </div>
                    </div>
                  </div>
                )}
              </div>

              {/* Processing Stats */}
              <div className="rounded-lg border border-border bg-card p-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-semibold text-foreground">Processing Stats</h3>
                  <Activity className="h-5 w-5 text-muted-foreground" />
                </div>
                {analyticsLoading ? (
                  <div className="space-y-3">
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-10 w-full" />)}
                  </div>
                ) : (
                  <div className="space-y-3">
                    <div className="flex items-center justify-between p-3 rounded-lg bg-muted/50">
                      <div className="flex items-center gap-2">
                        <FileText className="h-4 w-4 text-blue-500" />
                        <span className="text-sm text-muted-foreground">Documents</span>
                      </div>
                      <span className="font-semibold text-foreground">{analytics?.documents_processed?.toLocaleString() || 0}</span>
                    </div>
                    <div className="flex items-center justify-between p-3 rounded-lg bg-muted/50">
                      <div className="flex items-center gap-2">
                        <Database className="h-4 w-4 text-green-500" />
                        <span className="text-sm text-muted-foreground">Records</span>
                      </div>
                      <span className="font-semibold text-foreground">{analytics?.total_records?.toLocaleString() || 0}</span>
                    </div>
                    <div className="flex items-center justify-between p-3 rounded-lg bg-red-500/10">
                      <div className="flex items-center gap-2">
                        <AlertTriangle className="h-4 w-4 text-red-500" />
                        <span className="text-sm text-muted-foreground">PII Detected</span>
                      </div>
                      <span className="font-semibold text-red-600">{analytics?.pii_detected?.toLocaleString() || 0}</span>
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Data Quality Summary */}
            <div className="rounded-lg border border-border bg-card p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-foreground">Data Quality Summary</h3>
                <Link href="/quality" className="text-sm text-primary hover:underline">
                  View details
                </Link>
              </div>
              {qualityLoading ? (
                <div className="space-y-3">
                  <Skeleton className="h-8 w-32" />
                  <Skeleton className="h-24 w-full" />
                </div>
              ) : qualityTrends?.overall ? (
                <div className="space-y-4">
                  <div className="flex items-baseline gap-3">
                    <span className="text-4xl font-bold text-foreground">{Math.round(qualityTrends.overall * 100)}%</span>
                    <span className="text-sm text-muted-foreground">Overall Quality Score</span>
                  </div>
                  <div className="grid grid-cols-5 gap-3">
                    {[
                      { label: 'Completeness', value: qualityTrends?.completeness || 0, color: 'bg-blue-500' },
                      { label: 'Accuracy', value: qualityTrends?.accuracy || 0, color: 'bg-green-500' },
                      { label: 'Consistency', value: qualityTrends?.consistency || 0, color: 'bg-yellow-500' },
                      { label: 'Timeliness', value: qualityTrends?.timeliness || 0, color: 'bg-purple-500' },
                      { label: 'Uniqueness', value: qualityTrends?.uniqueness || 0, color: 'bg-pink-500' },
                    ].map((dim) => (
                      <div key={dim.label} className="text-center">
                        <div className="h-20 w-full bg-muted rounded-lg relative overflow-hidden">
                          <div 
                            className={`absolute bottom-0 left-0 right-0 ${dim.color} transition-all`}
                            style={{ height: `${dim.value * 100}%` }}
                          />
                        </div>
                        <p className="text-xs text-muted-foreground mt-2 truncate">{dim.label}</p>
                        <p className="text-sm font-medium">{Math.round(dim.value * 100)}%</p>
                      </div>
                    ))}
                  </div>
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center py-8 text-center">
                  <div className="h-12 w-12 rounded-full bg-muted flex items-center justify-center mb-3">
                    <BarChart3 className="h-6 w-6 text-muted-foreground" />
                  </div>
                  <p className="text-sm text-muted-foreground">No quality assessments yet</p>
                  <Link href="/data-sources" className="text-xs text-primary hover:underline mt-1">
                    Run assessment on a data source
                  </Link>
                </div>
              )}
            </div>

            {/* Recent Activity Feed */}
            <div className="rounded-lg border border-border bg-card p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-foreground">Recent Activity</h3>
                <Link href="/audit" className="text-sm text-primary hover:underline">
                  View all
                </Link>
              </div>
              {auditLoading ? (
                <div className="space-y-3">
                  {[1, 2, 3, 4].map(i => (
                    <Skeleton key={i} className="h-16 w-full" />
                  ))}
                </div>
              ) : Array.isArray(auditTrail) && auditTrail.length > 0 ? (
                <div className="space-y-2">
                  {auditTrail.slice(0, 4).map((log: any) => {
                    const details = typeof log.details === 'string' ? JSON.parse(log.details || '{}') : (log.details || {});
                    const actionParts = log.action?.split('.') || [];
                    const actionType = actionParts[1] || actionParts[0] || 'action';
                    const resourceType = actionParts[0] || log.resource || 'system';
                    
                    const getActionIcon = () => {
                      if (log.action?.includes('login')) return '🔐';
                      if (log.action?.includes('logout')) return '🚪';
                      if (log.action?.includes('create')) return '➕';
                      if (log.action?.includes('delete')) return '🗑️';
                      if (log.action?.includes('update')) return '✏️';
                      if (log.action?.includes('scan')) return '🔍';
                      if (log.action?.includes('classification')) return '🏷️';
                      if (log.action?.includes('policy')) return '📋';
                      return '📌';
                    };
                    
                    const getActionColor = () => {
                      if (log.action?.includes('delete')) return 'text-red-500';
                      if (log.action?.includes('create')) return 'text-green-500';
                      if (log.action?.includes('login')) return 'text-blue-500';
                      if (log.action?.includes('scan')) return 'text-purple-500';
                      return 'text-gray-500';
                    };
                    
                    const getDescription = () => {
                      if (log.action === 'user.login') {
                        return `${details.email || 'User'} signed in`;
                      }
                      if (log.action === 'user.logout') {
                        return `${details.email || 'User'} signed out`;
                      }
                      if (log.action?.includes('datasource.created')) {
                        return `Data source "${details.name || 'Unknown'}" created`;
                      }
                      if (log.action?.includes('datasource.deleted')) {
                        return `Data source removed`;
                      }
                      if (log.action?.includes('datasource.scan_started')) {
                        return `Scan started on "${details.name || 'data source'}"`;
                      }
                      if (log.action?.includes('classification')) {
                        return `Classification ${actionType} on dataset`;
                      }
                      if (log.action?.includes('policy')) {
                        return `Policy "${details.name || ''}" ${actionType}`;
                      }
                      return `${resourceType} ${actionType}`;
                    };

                    return (
                      <div key={log.id} className="flex items-start gap-3 p-3 rounded-lg border border-border/50 hover:bg-muted/50 transition-colors">
                        <span className="text-lg">{getActionIcon()}</span>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm text-foreground font-medium">
                            {getDescription()}
                          </p>
                          <div className="flex items-center gap-3 mt-1">
                            <span className="text-xs text-muted-foreground">
                              {new Date(log.created_at).toLocaleString('en-US', {
                                month: 'short',
                                day: 'numeric',
                                hour: '2-digit',
                                minute: '2-digit'
                              })}
                            </span>
                            {log.ip && log.ip !== '' && (
                              <span className="text-xs text-muted-foreground flex items-center gap-1">
                                <span className="inline-block w-1.5 h-1.5 rounded-full bg-green-500"></span>
                                {log.ip}
                              </span>
                            )}
                          </div>
                        </div>
                        <span className={`text-xs font-medium px-2 py-0.5 rounded-full bg-muted ${getActionColor()}`}>
                          {actionType}
                        </span>
                      </div>
                    );
                  })}
                </div>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  <Activity className="h-12 w-12 mx-auto mb-2 opacity-50" />
                  <p>No recent activity</p>
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
            {/* Compliance Score - Large Circular */}
            <div className="rounded-lg border border-border bg-card p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-foreground">Compliance Score</h3>
                <Link href="/governance" className="text-sm text-primary hover:underline">
                  Details
                </Link>
              </div>
              {riskLoading || gapsLoading ? (
                <div className="space-y-3">
                  <Skeleton className="h-32 w-32 mx-auto rounded-full" />
                  <Skeleton className="h-8 w-full" />
                </div>
              ) : (
                <div className="space-y-4">
                  <div className="flex justify-center">
                    <div className="relative h-32 w-32">
                      <svg className="h-32 w-32 -rotate-90" viewBox="0 0 36 36">
                        <path
                          d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                          fill="none"
                          stroke="currentColor"
                          strokeWidth="2.5"
                          className="text-muted"
                        />
                        <path
                          d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                          fill="none"
                          stroke="currentColor"
                          strokeWidth="2.5"
                          strokeDasharray={`${(riskScore?.overall_score || 0) * 100}, 100`}
                          strokeLinecap="round"
                          className={riskScore?.overall_score >= 0.8 ? 'text-green-500' : riskScore?.overall_score >= 0.6 ? 'text-yellow-500' : 'text-red-500'}
                        />
                      </svg>
                      <div className="absolute inset-0 flex flex-col items-center justify-center">
                        <span className="text-3xl font-bold">{Math.round((riskScore?.overall_score || 0) * 100)}%</span>
                        <span className="text-xs text-muted-foreground">Overall</span>
                      </div>
                    </div>
                  </div>
                  <div className="space-y-2 pt-2 border-t border-border">
                    {[
                      { label: 'GDPR',         key: 'gdpr_score',    color: 'bg-blue-500',    fallback: 0.75 },
                      { label: 'CCPA',         key: 'ccpa_score',    color: 'bg-purple-500',  fallback: 0.82 },
                      { label: 'HIPAA',        key: 'hipaa_score',   color: 'bg-green-500',   fallback: 0.68 },
                      { label: 'PCI-DSS',      key: 'pci_score',     color: 'bg-orange-500',  fallback: null },
                      { label: 'DPDP 2023',    key: 'dpdp_score',    color: 'bg-pink-500',    fallback: null },
                      { label: 'UAE PDPL',     key: 'uae_pdpl_score',color: 'bg-cyan-500',    fallback: null },
                      { label: 'EU AI Act',    key: 'eu_ai_act_score',color: 'bg-yellow-500', fallback: null },
                    ].map(({ label, key, color, fallback }) => {
                      const raw = riskScore?.[key as keyof typeof riskScore]
                      const pct = raw != null ? Math.round((raw as number) * 100) : (fallback != null ? Math.round(fallback * 100) : null)
                      return (
                        <div key={label} className="flex items-center justify-between">
                          <span className="text-sm text-muted-foreground w-20 shrink-0">{label}</span>
                          <div className="flex items-center gap-2">
                            <div className="w-20 h-2 bg-muted rounded-full overflow-hidden">
                              <div className={`h-full ${color}`} style={{ width: pct != null ? `${pct}%` : '0%' }} />
                            </div>
                            <span className="text-sm font-medium w-14 text-right">
                              {pct != null ? `${pct}%` : 'N/A'}
                            </span>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                  <p className="text-xs text-muted-foreground text-center">
                    {Array.isArray(gaps) ? gaps.filter((g: any) => g.status !== 'resolved').length : 0} compliance gaps to address
                  </p>
                </div>
              )}
            </div>

            {/* Alerts & Recommendations */}
            <div className="rounded-lg border border-border bg-card p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-foreground">Recommendations</h3>
                <Lightbulb className="h-5 w-5 text-yellow-500" />
              </div>
              {recsLoading ? (
                <div className="space-y-3">
                  {[1, 2, 3].map(i => <Skeleton key={i} className="h-16 w-full" />)}
                </div>
              ) : Array.isArray(recommendations) && recommendations.length > 0 ? (
                <div className="space-y-3">
                  {recommendations.slice(0, 3).map((rec: any, idx: number) => (
                    <div key={rec.id || idx} className={`p-3 rounded-lg border ${
                      rec.priority === 'high' ? 'border-red-200 bg-red-50 dark:border-red-900 dark:bg-red-950' :
                      rec.priority === 'medium' ? 'border-yellow-200 bg-yellow-50 dark:border-yellow-900 dark:bg-yellow-950' :
                      'border-blue-200 bg-blue-50 dark:border-blue-900 dark:bg-blue-950'
                    }`}>
                      <div className="flex items-start gap-2">
                        <AlertTriangle className={`h-4 w-4 mt-0.5 flex-shrink-0 ${
                          rec.priority === 'high' ? 'text-red-500' :
                          rec.priority === 'medium' ? 'text-yellow-500' :
                          'text-blue-500'
                        }`} />
                        <div>
                          <p className="text-sm font-medium text-foreground">{rec.title}</p>
                          <p className="text-xs text-muted-foreground mt-1 line-clamp-2">{rec.description}</p>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-6 text-muted-foreground">
                  <CheckCircle className="h-10 w-10 mx-auto mb-2 opacity-50" />
                  <p className="text-sm">All caught up!</p>
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
