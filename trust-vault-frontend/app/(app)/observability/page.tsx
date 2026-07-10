'use client'

import { useMemo } from 'react'
import { StatCard } from '@/components/base/stat-card'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { StatusIndicator } from '@/components/base/status-badge'
import Link from 'next/link'
import { Activity, Server, AlertTriangle, Clock } from 'lucide-react'
import { useSystemHealth, useSystemMetrics, useAlerts } from '@/hooks/use-audit'

export default function ObservabilityPage() {
  const { data: health, isLoading: healthLoading } = useSystemHealth()
  const { data: metrics, isLoading: metricsLoading } = useSystemMetrics()
  const { data: alerts, isLoading: alertsLoading } = useAlerts()

  const alertsData = Array.isArray(alerts) ? alerts : []
  const activeAlerts = alertsData.filter((a: any) => !a.resolved).length

  const isLoading = healthLoading || metricsLoading || alertsLoading

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
                label="CPU Usage"
                value={`${metrics?.cpu_usage || 0}%`}
                icon={<Activity className="h-6 w-6" />}
              />
              <StatCard
                label="Memory Usage"
                value={`${metrics?.memory_usage || 0}%`}
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

        {/* Component Health */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Component Health</h3>
          {healthLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : health?.components ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {Object.entries(health.components).map(([name, component]: [string, any]) => (
                <div key={name} className="flex items-center justify-between p-4 rounded-lg bg-muted/50">
                  <div className="flex items-center gap-3">
                    <div className={`w-3 h-3 rounded-full ${
                      component.status === 'healthy' ? 'bg-green-500' : 'bg-red-500'
                    }`} />
                    <span className="font-medium text-foreground capitalize">{name}</span>
                  </div>
                  {component.latency_ms && (
                    <span className="text-sm text-muted-foreground">{component.latency_ms}ms</span>
                  )}
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
                ? `${Math.floor(health.uptime_seconds / 3600)} hours`
                : 'N/A'}
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
