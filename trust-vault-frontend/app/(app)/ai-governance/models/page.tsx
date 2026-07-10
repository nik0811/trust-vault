'use client'

import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { StatusBadge } from '@/components/base/status-badge'
import { Bot, Plus, Loader2 } from 'lucide-react'
import { useAIModels } from '@/hooks/use-datamap'
import { toast } from 'sonner'
import { useEffect } from 'react'

const riskStatus: Record<string, 'error' | 'warning' | 'active'> = {
  high: 'error',
  medium: 'warning',
  low: 'active',
}

export default function ModelRegistryPage() {
  const { data: modelsRaw, isLoading, error } = useAIModels()

  useEffect(() => {
    if (error) toast.error('Failed to load AI models')
  }, [error])

  const models = (modelsRaw || []).map((m: any) => ({
    id: m.id,
    name: m.name,
    owner: m.owner || 'Unknown',
    risk: m.risk || 'medium',
    datasets: m.datasets || 0,
    approved: m.approved ?? false,
    lastReview: m.last_review || '—',
  }))

  return (
    <div className="space-y-6">
      <Breadcrumbs items={[{ label: 'AI Governance', href: '/ai-governance' }, { label: 'Model Registry' }]} />

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Model Registry</h1>
          <p className="text-muted-foreground mt-1">AI/ML models registered for governance review</p>
        </div>
        <button className="flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:opacity-90 transition-opacity">
          <Plus className="h-4 w-4" />
          Register Model
        </button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : models.length === 0 ? (
        <div className="rounded-lg border border-border bg-card p-12 text-center text-muted-foreground">
          <Bot className="h-12 w-12 mx-auto mb-4 opacity-50" />
          <p>No AI models registered yet</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {models.map((m: any) => (
            <div key={m.id} className="rounded-lg border border-border bg-card p-6">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div className="rounded-lg bg-primary/10 p-2">
                    <Bot className="h-5 w-5 text-primary" />
                  </div>
                  <div>
                    <h3 className="font-semibold text-foreground">{m.name}</h3>
                    <p className="text-xs text-muted-foreground">{m.owner}</p>
                  </div>
                </div>
                <StatusBadge status={riskStatus[m.risk] || 'warning'} label={`${m.risk} risk`} />
              </div>

              <div className="grid grid-cols-3 gap-4 text-center">
                <div>
                  <p className="text-sm font-semibold text-foreground">{m.datasets}</p>
                  <p className="text-xs text-muted-foreground">Datasets</p>
                </div>
                <div>
                  <p className={`text-sm font-semibold ${m.approved ? 'text-emerald-500' : 'text-amber-500'}`}>
                    {m.approved ? 'Approved' : 'Pending'}
                  </p>
                  <p className="text-xs text-muted-foreground">Governance</p>
                </div>
                <div>
                  <p className="text-sm font-semibold text-foreground">{m.lastReview}</p>
                  <p className="text-xs text-muted-foreground">Last Review</p>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
