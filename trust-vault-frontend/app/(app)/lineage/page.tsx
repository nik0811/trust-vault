'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { GitBranch, Search } from 'lucide-react'
import { useDatasetLineage } from '@/hooks/use-datamap'
import { useDataSources } from '@/hooks/use-datasources'

export default function LineagePage() {
  const [selectedDataset, setSelectedDataset] = useState('')
  const { data: dataSources, isLoading: dsLoading } = useDataSources()
  const { data: lineage, isLoading: lineageLoading } = useDatasetLineage(selectedDataset)

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
            <div className="space-y-4">
              {/* Simple lineage visualization */}
              <div className="flex items-center justify-center gap-4 py-8">
                <div className="p-4 rounded-lg bg-blue-500/10 border border-blue-500/20">
                  <p className="text-sm font-medium text-blue-600">Source</p>
                  <p className="text-xs text-muted-foreground">Data Origin</p>
                </div>
                <div className="w-16 h-0.5 bg-border" />
                <div className="p-4 rounded-lg bg-primary/10 border border-primary/20">
                  <p className="text-sm font-medium text-primary">TrustVault</p>
                  <p className="text-xs text-muted-foreground">Classification & Governance</p>
                </div>
                <div className="w-16 h-0.5 bg-border" />
                <div className="p-4 rounded-lg bg-green-500/10 border border-green-500/20">
                  <p className="text-sm font-medium text-green-600">AI Gate</p>
                  <p className="text-xs text-muted-foreground">Governed Access</p>
                </div>
                <div className="w-16 h-0.5 bg-border" />
                <div className="p-4 rounded-lg bg-purple-500/10 border border-purple-500/20">
                  <p className="text-sm font-medium text-purple-600">AI/LLM</p>
                  <p className="text-xs text-muted-foreground">Consumption</p>
                </div>
              </div>
              
              {/* Lineage details */}
              <pre className="p-4 rounded-lg bg-muted text-sm overflow-auto max-h-64">
                {JSON.stringify(lineage, null, 2)}
              </pre>
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
