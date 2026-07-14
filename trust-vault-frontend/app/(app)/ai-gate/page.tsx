'use client'

import { useMemo } from 'react'
import { StatCard } from '@/components/base/stat-card'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import Link from 'next/link'
import { Zap, Shield, Clock, Activity } from 'lucide-react'
import { useGateStats, useGateQueries } from '@/hooks/use-gate'

export default function AIGatePage() {
  const { data: stats, isLoading: statsLoading } = useGateStats()
  const { data: queries, isLoading: queriesLoading } = useGateQueries(10)

  const recentQueries = useMemo(() => {
    if (!queries || !Array.isArray(queries)) return []
    return queries.slice(0, 5)
  }, [queries])

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'AI Gate', active: true }]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">AI Gate</h1>
        <p className="text-sm text-muted-foreground mt-1">Secure gateway between your data and AI systems</p>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {statsLoading ? (
            <>
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
            </>
          ) : (
            <>
              <StatCard
                label="Total Queries"
                value={(stats?.total_queries || 0).toLocaleString()}
                icon={<Zap className="h-6 w-6" />}
              />
              <StatCard
                label="Allowed"
                value={(stats?.allowed_queries || 0).toLocaleString()}
                change={1}
                changeLabel="passed"
                icon={<Shield className="h-6 w-6" />}
              />
              <StatCard
                label="Blocked"
                value={(stats?.blocked_queries || 0).toLocaleString()}
                change={-1}
                changeLabel="denied"
                icon={<Shield className="h-6 w-6" />}
              />
              <StatCard
                label="Avg Latency"
                value={`${Math.round(stats?.avg_latency_ms || 0)}ms`}
                icon={<Clock className="h-6 w-6" />}
              />
            </>
          )}
        </div>

        {/* Quick Links */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <Link
            href="/ai-gate/playground"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <Zap className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Playground</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Test AI Gate queries interactively with real-time policy evaluation
            </p>
          </Link>

          <Link
            href="/ai-gate/queries"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <Activity className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Query History</h3>
            <p className="text-sm text-muted-foreground mt-1">
              View all AI Gate queries, decisions, and applied policies
            </p>
          </Link>
        </div>

        {/* Recent Queries */}
        <div className="rounded-lg border border-border bg-card p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-foreground">Recent Queries</h3>
            <Link href="/ai-gate/queries" className="text-sm text-primary hover:underline">
              View all
            </Link>
          </div>
          {queriesLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : recentQueries.length > 0 ? (
            <div className="space-y-3">
              {recentQueries.map((query: any) => (
                <div
                  key={query.id}
                  className="flex items-center justify-between p-3 rounded-lg bg-muted/50"
                >
                  <div className="flex items-center gap-3">
                    <div
                      className={`w-2 h-2 rounded-full ${
                        query.decision === 'allow' ? 'bg-green-500' : 'bg-red-500'
                      }`}
                    />
                    <span className="text-sm text-foreground truncate max-w-md">
                      {query.query || 'Query'}
                    </span>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className="text-xs text-muted-foreground">
                      {query.latency_ms}ms
                    </span>
                    <span className="text-xs text-muted-foreground">
                      {new Date(query.created_at).toLocaleTimeString()}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <p className="text-muted-foreground">No queries yet</p>
              <Link href="/ai-gate/playground" className="text-primary hover:underline text-sm mt-2 inline-block">
                Try the playground
              </Link>
            </div>
          )}
        </div>

        {/* How it Works */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">How AI Gate Works</h3>
          <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
            <div>
              <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-3">
                <span className="text-primary font-bold">1</span>
              </div>
              <h4 className="font-medium text-foreground">Query</h4>
              <p className="text-sm text-muted-foreground mt-1">
                RAG/LLM sends a query through AI Gate
              </p>
            </div>
            <div>
              <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-3">
                <span className="text-primary font-bold">2</span>
              </div>
              <h4 className="font-medium text-foreground">Retrieve</h4>
              <p className="text-sm text-muted-foreground mt-1">
                Relevant data chunks are retrieved from vector DB
              </p>
            </div>
            <div>
              <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-3">
                <span className="text-primary font-bold">3</span>
              </div>
              <h4 className="font-medium text-foreground">Govern</h4>
              <p className="text-sm text-muted-foreground mt-1">
                Policies are applied, data is classified and redacted
              </p>
            </div>
            <div>
              <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-3">
                <span className="text-primary font-bold">4</span>
              </div>
              <h4 className="font-medium text-foreground">Return</h4>
              <p className="text-sm text-muted-foreground mt-1">
                Safe, governed data is returned to the AI system
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
