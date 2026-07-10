'use client'

import { useState, useEffect } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { CheckCircle2, XCircle, Scale, Loader2 } from 'lucide-react'
import { useEligibleDatasets, useCheckEligibility, type EligibilityResult } from '@/hooks/use-datamap'
import { toast } from 'sonner'

export default function EligibilityPage() {
  const { data: datasetsRaw, isLoading: datasetsLoading, error: datasetsError } = useEligibleDatasets()
  const checkEligibility = useCheckEligibility()
  
  const [dataset, setDataset] = useState('')
  const [useCase, setUseCase] = useState('training')
  const [result, setResult] = useState<EligibilityResult | null>(null)

  const datasets: string[] = datasetsRaw || []

  useEffect(() => {
    if (datasetsError) toast.error('Failed to load datasets')
  }, [datasetsError])

  useEffect(() => {
    if (datasets.length > 0 && !dataset) {
      setDataset(datasets[0])
    }
  }, [datasets, dataset])

  const runCheck = async () => {
    if (!dataset) {
      toast.error('Please select a dataset')
      return
    }
    
    try {
      const res = await checkEligibility.mutateAsync({ dataset, use_case: useCase })
      setResult(res)
    } catch {
      toast.error('Failed to check eligibility')
    }
  }

  return (
    <div className="space-y-6">
      <Breadcrumbs items={[{ label: 'AI Governance', href: '/ai-governance' }, { label: 'Eligibility' }]} />

      <div>
        <h1 className="text-2xl font-bold text-foreground">AI Eligibility Checker</h1>
        <p className="text-muted-foreground mt-1">Verify whether a dataset can be used for a given AI use case</p>
      </div>

      <div className="rounded-lg border border-border bg-card p-6 space-y-4 max-w-2xl">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="space-y-2">
            <label htmlFor="elig-dataset" className="text-sm font-medium text-foreground">Dataset</label>
            {datasetsLoading ? (
              <div className="flex items-center gap-2 text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                <span className="text-sm">Loading datasets...</span>
              </div>
            ) : datasets.length === 0 ? (
              <p className="text-sm text-muted-foreground">No datasets available</p>
            ) : (
              <select
                id="elig-dataset"
                value={dataset}
                onChange={(e) => { setDataset(e.target.value); setResult(null) }}
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground"
              >
                {datasets.map((d) => <option key={d} value={d}>{d}</option>)}
              </select>
            )}
          </div>
          <div className="space-y-2">
            <label htmlFor="elig-usecase" className="text-sm font-medium text-foreground">Use Case</label>
            <select
              id="elig-usecase"
              value={useCase}
              onChange={(e) => { setUseCase(e.target.value); setResult(null) }}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground"
            >
              <option value="training">Model Training</option>
              <option value="inference">Inference</option>
              <option value="rag">RAG / Retrieval</option>
              <option value="finetuning">Fine-tuning</option>
            </select>
          </div>
        </div>
        <button
          onClick={runCheck}
          disabled={checkEligibility.isPending || !dataset}
          className="flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:opacity-90 transition-opacity disabled:opacity-50"
        >
          {checkEligibility.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Scale className="h-4 w-4" />
          )}
          Check Eligibility
        </button>
      </div>

      {result && (
        <div className="rounded-lg border border-border bg-card p-6 max-w-2xl space-y-4">
          <div className={`flex items-center gap-3 rounded-lg p-4 ${result.eligible ? 'bg-emerald-500/10' : 'bg-destructive/10'}`}>
            {result.eligible ? (
              <CheckCircle2 className="h-6 w-6 text-emerald-500" />
            ) : (
              <XCircle className="h-6 w-6 text-destructive" />
            )}
            <p className="font-semibold text-foreground">
              {result.eligible ? 'Eligible for this use case' : 'Not eligible for this use case'}
            </p>
          </div>
          <div className="space-y-2">
            {result.checks.map((c) => (
              <div key={c.name} className="flex items-center justify-between rounded-md border border-border px-4 py-3">
                <div className="flex items-center gap-3">
                  {c.passed ? (
                    <CheckCircle2 className="h-4 w-4 text-emerald-500 shrink-0" />
                  ) : (
                    <XCircle className="h-4 w-4 text-destructive shrink-0" />
                  )}
                  <span className="text-sm font-medium text-foreground">{c.name}</span>
                </div>
                <span className="text-xs text-muted-foreground">{c.note}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
