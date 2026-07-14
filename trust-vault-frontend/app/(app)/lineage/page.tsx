'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { GitBranch, Search, ArrowRight, Database, Shield, Cpu, AlertCircle } from 'lucide-react'
import { useDatasetLineage } from '@/hooks/use-datamap'
import { useDataSources } from '@/hooks/use-datasources'

export default function LineagePage() {
  const [selectedDataset, setSelectedDataset] = useState('')
  const { data: dataSources, isLoading: dsLoading } = useDataSources()
  const { data: lineage, isLoading: lineageLoading } = useDatasetLineage(selectedDataset)

  const hasUpstream = lineage?.upstream && Array.isArray(lineage.upstream) && lineage.upstream.length > 0
  const hasDownstream = lineage?.downstream && Array.isArray(lineage.downstream) && lineage.downstream.length > 0
  const hasAiUsage = lineage?.ai_usage && Array.isArray(lineage.ai_usage) && lineage.ai_usage.length > 0
  const hasAnyLineage = hasUpstream || hasDownstream || hasAiUsage

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Data Lineage', active: true }]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">Data Lineage</h1>
        <p className="text-sm text-muted-foreground mt-1">Track data flow from source to AI consumption</p>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Dataset Selector */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Select Dataset</h3>
          <div className="flex gap-4">
            <select
              value={selectedDataset}
              onChange={(e) => setSelectedDataset(e.target.value)}
              className="flex-1 px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            >
              <option value="">Select a dataset...</option>
              {Array.isArray(dataSources) && dataSources.map((ds) => (
                <option key={ds.id} value={ds.id}>{ds.name}</option>
              ))}
            </select>
            <button
              disabled={!selectedDataset}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Search className="h-4 w-4" />
              View Lineage
            </button>
          </div>
        </div>

        {/* Lineage Visualization */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Lineage Graph</h3>
          {!selectedDataset ? (
            <div className="text-center py-12">
              <GitBranch className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <p className="text-muted-foreground">Select a dataset to view its lineage</p>
            </div>
          ) : lineageLoading ? (
            <Skeleton className="h-64 w-full" />
          ) : lineage ? (
            <div className="space-y-6">
              {/* Visual lineage flow */}
              <div className="flex items-center justify-center gap-4 py-8 overflow-x-auto">
                {/* Upstream sources */}
                <div className="flex flex-col gap-2 items-center">
                  <div className="p-4 rounded-lg bg-blue-500/10 border border-blue-500/20 min-w-[140px]">
                    <Database className="h-6 w-6 text-blue-600 mx-auto mb-2" />
                    <p className="text-sm font-medium text-blue-600 text-center">Source</p>
                    <p className="text-xs text-muted-foreground text-center">
                      {hasUpstream ? `${lineage.upstream.length} upstream` : 'Data Origin'}
                    </p>
                  </div>
                </div>
                
                <ArrowRight className="h-6 w-6 text-muted-foreground flex-shrink-0" />
                
                {/* TrustVault */}
                <div className="p-4 rounded-lg bg-primary/10 border border-primary/20 min-w-[140px]">
                  <Shield className="h-6 w-6 text-primary mx-auto mb-2" />
                  <p className="text-sm font-medium text-primary text-center">TrustVault</p>
                  <p className="text-xs text-muted-foreground text-center">Classification & Governance</p>
                </div>
                
                <ArrowRight className="h-6 w-6 text-muted-foreground flex-shrink-0" />
                
                {/* AI Gate */}
                <div className="p-4 rounded-lg bg-green-500/10 border border-green-500/20 min-w-[140px]">
                  <Shield className="h-6 w-6 text-green-600 mx-auto mb-2" />
                  <p className="text-sm font-medium text-green-600 text-center">AI Gate</p>
                  <p className="text-xs text-muted-foreground text-center">Governed Access</p>
                </div>
                
                <ArrowRight className="h-6 w-6 text-muted-foreground flex-shrink-0" />
                
                {/* AI/LLM */}
                <div className="p-4 rounded-lg bg-purple-500/10 border border-purple-500/20 min-w-[140px]">
                  <Cpu className="h-6 w-6 text-purple-600 mx-auto mb-2" />
                  <p className="text-sm font-medium text-purple-600 text-center">AI/LLM</p>
                  <p className="text-xs text-muted-foreground text-center">
                    {hasAiUsage ? `${lineage.ai_usage.length} models` : 'Consumption'}
                  </p>
                </div>
              </div>
              
              {/* Lineage details */}
              {hasAnyLineage ? (
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  {/* Upstream */}
                  <div className="rounded-lg border border-border p-4">
                    <h4 className="font-medium text-foreground mb-3 flex items-center gap-2">
                      <Database className="h-4 w-4 text-blue-600" />
                      Upstream Sources
                    </h4>
                    {hasUpstream ? (
                      <ul className="space-y-2">
                        {lineage.upstream.map((flow: any, i: number) => (
                          <li key={i} className="text-sm text-muted-foreground flex items-center gap-2">
                            <span className="w-2 h-2 rounded-full bg-blue-500" />
                            {flow.source_dataset_id}
                          </li>
                        ))}
                      </ul>
                    ) : (
                      <p className="text-sm text-muted-foreground">No upstream sources recorded</p>
                    )}
                  </div>
                  
                  {/* Downstream */}
                  <div className="rounded-lg border border-border p-4">
                    <h4 className="font-medium text-foreground mb-3 flex items-center gap-2">
                      <ArrowRight className="h-4 w-4 text-green-600" />
                      Downstream Flows
                    </h4>
                    {hasDownstream ? (
                      <ul className="space-y-2">
                        {lineage.downstream.map((flow: any, i: number) => (
                          <li key={i} className="text-sm text-muted-foreground flex items-center gap-2">
                            <span className="w-2 h-2 rounded-full bg-green-500" />
                            {flow.target_dataset_id}
                          </li>
                        ))}
                      </ul>
                    ) : (
                      <p className="text-sm text-muted-foreground">No downstream flows recorded</p>
                    )}
                  </div>
                  
                  {/* AI Usage */}
                  <div className="rounded-lg border border-border p-4">
                    <h4 className="font-medium text-foreground mb-3 flex items-center gap-2">
                      <Cpu className="h-4 w-4 text-purple-600" />
                      AI Model Usage
                    </h4>
                    {hasAiUsage ? (
                      <ul className="space-y-2">
                        {lineage.ai_usage.map((usage: any, i: number) => (
                          <li key={i} className="text-sm text-muted-foreground flex items-center gap-2">
                            <span className="w-2 h-2 rounded-full bg-purple-500" />
                            {usage.model_id || usage.model_name || 'AI Model'}
                          </li>
                        ))}
                      </ul>
                    ) : (
                      <p className="text-sm text-muted-foreground">No AI usage recorded</p>
                    )}
                  </div>
                </div>
              ) : (
                <div className="rounded-lg border border-yellow-500/20 bg-yellow-500/5 p-4 flex items-start gap-3">
                  <AlertCircle className="h-5 w-5 text-yellow-600 flex-shrink-0 mt-0.5" />
                  <div>
                    <p className="font-medium text-foreground">No lineage data yet</p>
                    <p className="text-sm text-muted-foreground mt-1">
                      Lineage data is automatically captured when you scan data sources and use the AI Gate.
                      Run a scan or make AI Gate queries to start building lineage.
                    </p>
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="text-center py-12">
              <p className="text-muted-foreground">No lineage data available for this dataset</p>
            </div>
          )}
        </div>

        {/* Info */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">About Data Lineage</h3>
          <p className="text-sm text-muted-foreground">
            Data lineage tracks the complete journey of your data from its source through TrustVault&apos;s
            governance layer to AI consumption. This provides full auditability and helps ensure compliance
            with regulations like GDPR and the EU AI Act.
          </p>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-4">
            <div className="p-4 rounded-lg bg-muted/50">
              <p className="font-medium text-foreground">Source Tracking</p>
              <p className="text-sm text-muted-foreground">Know where your data comes from</p>
            </div>
            <div className="p-4 rounded-lg bg-muted/50">
              <p className="font-medium text-foreground">Transformation History</p>
              <p className="text-sm text-muted-foreground">Track all data modifications</p>
            </div>
            <div className="p-4 rounded-lg bg-muted/50">
              <p className="font-medium text-foreground">AI Usage Audit</p>
              <p className="text-sm text-muted-foreground">See how AI systems use your data</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
