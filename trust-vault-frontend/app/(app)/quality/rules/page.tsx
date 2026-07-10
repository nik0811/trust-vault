'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { Plus, Save } from 'lucide-react'
import { useSetQualityThreshold } from '@/hooks/use-quality'
import { toast } from 'sonner'

const dimensions = [
  { id: 'completeness', name: 'Completeness', description: 'Minimum percentage of non-null values' },
  { id: 'accuracy', name: 'Accuracy', description: 'Minimum data correctness score' },
  { id: 'consistency', name: 'Consistency', description: 'Minimum cross-field consistency' },
  { id: 'timeliness', name: 'Timeliness', description: 'Maximum data age in days' },
  { id: 'uniqueness', name: 'Uniqueness', description: 'Maximum duplicate percentage' },
]

const severities = ['info', 'warning', 'critical']

export default function QualityRulesPage() {
  const setThreshold = useSetQualityThreshold()
  const [thresholds, setThresholds] = useState<Record<string, { minimum: number; severity: string }>>({
    completeness: { minimum: 90, severity: 'warning' },
    accuracy: { minimum: 85, severity: 'warning' },
    consistency: { minimum: 80, severity: 'warning' },
    timeliness: { minimum: 7, severity: 'info' },
    uniqueness: { minimum: 95, severity: 'critical' },
  })

  const handleSave = async (dimension: string) => {
    const threshold = thresholds[dimension]
    try {
      await setThreshold.mutateAsync({
        dimension,
        minimum: threshold.minimum,
        severity: threshold.severity,
      })
    } catch (error) {
      // Error handled by hook
    }
  }

  const handleSaveAll = async () => {
    for (const dimension of Object.keys(thresholds)) {
      await handleSave(dimension)
    }
    toast.success('All thresholds saved')
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs
            items={[
              { label: 'Data Quality', href: '/quality' },
              { label: 'Rules', active: true },
            ]}
          />
          <h1 className="text-3xl font-bold text-foreground mt-4">Quality Rules</h1>
          <p className="text-sm text-muted-foreground mt-1">Configure quality thresholds and alerting rules</p>
        </div>
        <button
          onClick={handleSaveAll}
          disabled={setThreshold.isPending}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
        >
          <Save className="h-4 w-4" />
          Save All
        </button>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Threshold Configuration */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-6">Quality Thresholds</h3>
          <div className="space-y-6">
            {dimensions.map((dim) => (
              <div key={dim.id} className="flex items-center gap-6 p-4 rounded-lg bg-muted/50">
                <div className="flex-1">
                  <h4 className="font-medium text-foreground">{dim.name}</h4>
                  <p className="text-sm text-muted-foreground">{dim.description}</p>
                </div>
                <div className="flex items-center gap-4">
                  <div>
                    <label className="block text-xs text-muted-foreground mb-1">Minimum</label>
                    <input
                      type="number"
                      value={thresholds[dim.id]?.minimum || 0}
                      onChange={(e) => setThresholds({
                        ...thresholds,
                        [dim.id]: { ...thresholds[dim.id], minimum: parseInt(e.target.value) || 0 }
                      })}
                      className="w-20 px-3 py-1 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-muted-foreground mb-1">Severity</label>
                    <select
                      value={thresholds[dim.id]?.severity || 'warning'}
                      onChange={(e) => setThresholds({
                        ...thresholds,
                        [dim.id]: { ...thresholds[dim.id], severity: e.target.value }
                      })}
                      className="px-3 py-1 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    >
                      {severities.map((s) => (
                        <option key={s} value={s}>{s}</option>
                      ))}
                    </select>
                  </div>
                  <button
                    onClick={() => handleSave(dim.id)}
                    disabled={setThreshold.isPending}
                    className="px-3 py-1 rounded-lg border border-border text-foreground hover:bg-muted transition-colors disabled:opacity-50"
                  >
                    Save
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Info */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">How Quality Rules Work</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div>
              <div className="w-10 h-10 rounded-lg bg-blue-500/10 flex items-center justify-center mb-3">
                <span className="text-blue-500 font-bold">1</span>
              </div>
              <h4 className="font-medium text-foreground">Continuous Assessment</h4>
              <p className="text-sm text-muted-foreground mt-1">
                Quality is assessed automatically during data scans
              </p>
            </div>
            <div>
              <div className="w-10 h-10 rounded-lg bg-yellow-500/10 flex items-center justify-center mb-3">
                <span className="text-yellow-500 font-bold">2</span>
              </div>
              <h4 className="font-medium text-foreground">Threshold Monitoring</h4>
              <p className="text-sm text-muted-foreground mt-1">
                When scores fall below thresholds, alerts are triggered
              </p>
            </div>
            <div>
              <div className="w-10 h-10 rounded-lg bg-green-500/10 flex items-center justify-center mb-3">
                <span className="text-green-500 font-bold">3</span>
              </div>
              <h4 className="font-medium text-foreground">Remediation</h4>
              <p className="text-sm text-muted-foreground mt-1">
                Issues are flagged for review and remediation
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
