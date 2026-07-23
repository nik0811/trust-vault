'use client'

import { useMemo } from 'react'
import { StatCard } from '@/components/base/stat-card'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { StatusIndicator } from '@/components/base/status-badge'
import Link from 'next/link'
import { Activity, Server, AlertTriangle, Clock, Database, Cpu, HardDrive, Zap } from 'lucide-react'
import { useSystemHealth, useSystemMetrics, useAlerts } from '@/hooks/use-audit'

export default function ObservabilityPage() {
  const { data: health, isLoading: healthLoading } = useSystemHealth()
  const { data: metrics, isLoading: metricsLoading } = useSystemMetrics()
  const { data: alerts, isLoading: alertsLoading } = useAlerts()

  const alertsData = Array.isArray(alerts) ? alerts : []
  const activeAlerts = alertsData.filter((a: any) => !a.resolved).length

  const isLoading = healthLoading || metricsLoading || alertsLoading

  // Get memory usage from metrics (real data from Go runtime)
  const memoryUsage = metrics?.memory_usage || 0
  const memoryAllocMB = metrics?.memory_alloc_mb || 0
  const memorySysMB = metrics?.memory_sys_mb || 0
  
  // Get goroutines as activity indicator
  const goroutines = metrics?.goroutines || 0
  
  // Get DB connection stats
  const activeConnections = metrics?.active_connections || 0
  const openConnections = metrics?.open_connections || 0
  
  // Get requests per minute
  const requestsPerMinute = metrics?.requests_per_minute || 0

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Observability', active: true }]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">System Observability</h1>
        <p className="text-sm text-muted-foreground mt-1">Monitor system health, metrics, and alerts</p>
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
                label="System Status"
                value={health?.status === 'healthy' ? 'Healthy' : 'Degraded'}
                change={health?.status === 'healthy' ? 1 : -1}
                changeLabel={health?.status === 'healthy' ? 'all systems' : 'issues detected'}
                icon={<Server className="h-6 w-6" />}
              />
              <StatCard
                label="Memory Usage"
                value={`${memoryUsage}%`}
                changeLabel={`${memoryAllocMB}MB / ${memorySysMB}MB`}
                icon={<Cpu className="h-6 w-6" />}
              />
              <StatCard
                label="Active Goroutines"
                value={goroutines.toString()}
                changeLabel={`${requestsPerMinute} req/min`}
                icon={<Activity className="h-6 w-6" />}
              />
              <StatCard
                label="Active Alerts"
                value={activeAlerts.toString()}
                change={activeAlerts > 0 ? -1 : 1}
                changeLabel={activeAlerts > 0 ? 'needs attention' : 'all clear'}
                icon={<AlertTriangle className="h-6 w-6" />}
              />
            </>
          )}
        </div>

        {/* Database Connections */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {isLoading ? (
            <>
              <Skeleton className="h-20" />
              <Skeleton className="h-20" />
              <Skeleton className="h-20" />
            </>
          ) : (
            <>
              <div className="rounded-lg border border-border bg-card p-4">
                <div className="flex items-center gap-3">
                  <Database className="h-5 w-5 text-primary" />
                  <div>
                    <p className="text-sm text-muted-foreground">DB Connections</p>
                    <p className="text-xl font-semibold text-foreground">{activeConnections} / {openConnections}</p>
                  </div>
                </div>
              </div>
              <div className="rounded-lg border border-border bg-card p-4">
                <div className="flex items-center gap-3">
                  <Zap className="h-5 w-5 text-yellow-500" />
                  <div>
                    <p className="text-sm text-muted-foreground">Avg Query Latency</p>
                    <p className="text-xl font-semibold text-foreground">{metrics?.queries?.avg_latency?.toFixed(1) || 0}ms</p>
                  </div>
                </div>
              </div>
              <div className="rounded-lg border border-border bg-card p-4">
                <div className="flex items-center gap-3">
                  <HardDrive className="h-5 w-5 text-blue-500" />
                  <div>
                    <p className="text-sm text-muted-foreground">Total Queries (24h)</p>
                    <p className="text-xl font-semibold text-foreground">{metrics?.queries?.last_24h || 0}</p>
                  </div>
                </div>
              </div>
            </>
          )}
        </div>

        {/* Component Health */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Component Health</h3>
          {healthLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : health?.components && Object.keys(health.components).length > 0 ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {Object.entries(health.components).map(([name, component]: [string, any]) => (
                <div key={name} className="flex items-center justify-between p-4 rounded-lg bg-muted/50">
                  <div className="flex items-center gap-3">
                    <div className={`w-3 h-3 rounded-full ${
                      component.status === 'healthy' ? 'bg-green-500' : 
                      component.status === 'degraded' ? 'bg-yellow-500' : 'bg-red-500'
                    }`} />
                    <span className="font-medium text-foreground">{name}</span>
                  </div>
                  <div className="text-right">
                    {component.latency_ms !== undefined && component.latency_ms > 0 && (
                      <span className="text-sm text-muted-foreground">{component.latency_ms}ms</span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <p className="text-muted-foreground">No component data available</p>
            </div>
          )}
        </div>

        {/* Quick Links */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <Link
            href="/observability/alerts"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <AlertTriangle className="h-8 w-8 text-yellow-500 mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Alerts</h3>
            <p className="text-sm text-muted-foreground mt-1">
              View and manage system alerts and notifications
            </p>
            <p className="text-sm text-primary mt-4">{activeAlerts} active alerts</p>
          </Link>

          <div className="rounded-lg border border-border bg-card p-6">
            <Clock className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Uptime</h3>
            <p className="text-sm text-muted-foreground mt-1">
              System has been running continuously
            </p>
            <p className="text-sm text-primary mt-4">
              {health?.uptime_seconds 
                ? formatUptime(health.uptime_seconds)
                : 'N/A'}
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  
  if (days > 0) {
    return `${days}d ${hours}h ${minutes}m`
  } else if (hours > 0) {
    return `${hours}h ${minutes}m`
  } else {
    return `${minutes}m`
  }
}
