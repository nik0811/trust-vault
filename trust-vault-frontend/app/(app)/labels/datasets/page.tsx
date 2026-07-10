'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { Tag } from 'lucide-react'
import { useDataSources } from '@/hooks/use-datasources'
import { useDatasetLabel, useAssignLabel } from '@/hooks/use-advisor'

export default function DatasetLabelsPage() {
  const { data: dataSources, isLoading: dsLoading } = useDataSources()
  const [selectedDataset, setSelectedDataset] = useState('')
  const { data: label, isLoading: labelLoading } = useDatasetLabel(selectedDataset)
  const assignLabel = useAssignLabel()

  const handleAssign = async (newLabel: string) => {
    if (!selectedDataset) return
    await assignLabel.mutateAsync({ dataset_id: selectedDataset, label: newLabel })
  }

  const labels = ['PUBLIC', 'INTERNAL', 'CONFIDENTIAL', 'HIGHLY_CONFIDENTIAL', 'RESTRICTED']

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Sensitivity Labels', href: '/labels' },
            { label: 'Datasets', active: true },
          ]}
        />
        <h1 className="text-3xl font-bold text-foreground mt-4">Dataset Labels</h1>
        <p className="text-sm text-muted-foreground mt-1">View and manage labels for datasets</p>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Dataset Selector */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Select Dataset</h3>
          <select
            value={selectedDataset}
            onChange={(e) => setSelectedDataset(e.target.value)}
            className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
          >
            <option value="">Select a dataset...</option>
            {Array.isArray(dataSources) && dataSources.map((ds) => (
              <option key={ds.id} value={ds.id}>{ds.name}</option>
            ))}
          </select>
        </div>

        {/* Label Assignment */}
        {selectedDataset && (
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="text-lg font-semibold text-foreground mb-4">Current Label</h3>
            {labelLoading ? (
              <Skeleton className="h-12 w-full" />
            ) : (
              <div className="space-y-4">
                <div className="flex items-center gap-4">
                  <Tag className="h-5 w-5 text-primary" />
                  <span className={`px-3 py-1 rounded text-sm font-medium ${
                    label?.label === 'RESTRICTED' ? 'bg-red-500/10 text-red-600' :
                    label?.label === 'CONFIDENTIAL' || label?.label === 'HIGHLY_CONFIDENTIAL' ? 'bg-yellow-500/10 text-yellow-600' :
                    label?.label === 'INTERNAL' ? 'bg-blue-500/10 text-blue-600' :
                    'bg-green-500/10 text-green-600'
                  }`}>
                    {label?.label || 'No label assigned'}
                  </span>
                  {label?.auto_assigned && (
                    <span className="text-xs text-muted-foreground">(auto-assigned)</span>
                  )}
                </div>

                <div>
                  <p className="text-sm font-medium text-foreground mb-2">Assign New Label</p>
                  <div className="flex flex-wrap gap-2">
                    {labels.map((l) => (
                      <button
                        key={l}
                        onClick={() => handleAssign(l)}
                        disabled={assignLabel.isPending}
                        className={`px-4 py-2 rounded-lg border transition-colors disabled:opacity-50 ${
                          label?.label === l
                            ? 'border-primary bg-primary/10 text-primary'
                            : 'border-border text-foreground hover:border-primary/50'
                        }`}
                      >
                        {l}
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {/* All Datasets */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">All Datasets</h3>
          {dsLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : dataSources && dataSources.length > 0 ? (
            <div className="space-y-3">
              {dataSources.map((ds) => (
                <div
                  key={ds.id}
                  onClick={() => setSelectedDataset(ds.id)}
                  className={`flex items-center justify-between p-4 rounded-lg cursor-pointer transition-colors ${
                    selectedDataset === ds.id ? 'bg-primary/10 border border-primary' : 'bg-muted/50 hover:bg-muted'
                  }`}
                >
                  <span className="font-medium text-foreground">{ds.name}</span>
                  <span className="text-sm text-muted-foreground capitalize">{ds.type}</span>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <p className="text-muted-foreground">No datasets available</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
