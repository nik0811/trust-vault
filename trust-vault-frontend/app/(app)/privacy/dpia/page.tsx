'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import {
  CheckCircle2, Clock, AlertTriangle, ChevronDown, ChevronRight,
  Plus, Shield, FileText, X, Loader2
} from 'lucide-react'
import {
  useDPIAs, useCreateDPIA, useDPIA, useUpdateDPIAStep,
  type DPIA, type DPIAStep
} from '@/hooks/use-privacy'
import { cn } from '@/lib/utils'

const RISK_COLORS: Record<string, string> = {
  high: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  medium: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400',
  low: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
}

const STATUS_COLORS: Record<string, string> = {
  in_progress: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  completed: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  pending_dpo: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
}

const STATUS_LABELS: Record<string, string> = {
  in_progress: 'In Progress',
  completed: 'Completed',
  pending_dpo: 'Pending DPO',
}

const STEP_LABELS: Record<string, string> = {
  identify_processing: 'Identify Processing',
  assess_necessity: 'Assess Necessity',
  identify_risks: 'Identify Risks',
  identify_mitigation: 'Identify Mitigation',
  dpo_consultation: 'DPO Consultation',
  sign_off: 'Sign Off',
}

function StepIcon({ status }: { status: string }) {
  if (status === 'completed') return <CheckCircle2 className="h-5 w-5 text-green-500" />
  if (status === 'in_progress') return <Loader2 className="h-5 w-5 text-blue-500 animate-spin" />
  if (status === 'skipped') return <X className="h-5 w-5 text-muted-foreground" />
  return <Clock className="h-5 w-5 text-muted-foreground" />
}

function StepProgress({ steps }: { steps: DPIAStep[] }) {
  const completed = steps.filter(s => s.status === 'completed' || s.status === 'skipped').length
  const pct = steps.length > 0 ? Math.round((completed / steps.length) * 100) : 0
  return (
    <div className="space-y-1">
      <div className="flex justify-between text-xs text-muted-foreground">
        <span>{completed}/{steps.length} steps</span>
        <span>{pct}%</span>
      </div>
      <div className="h-1.5 bg-muted rounded-full overflow-hidden">
        <div className="h-full bg-primary rounded-full transition-all" style={{ width: `${pct}%` }} />
      </div>
    </div>
  )
}

function DPIADetail({ id, onClose }: { id: string; onClose: () => void }) {
  const { data: dpia, isLoading } = useDPIA(id)
  const updateStep = useUpdateDPIAStep()
  const [expandedStep, setExpandedStep] = useState<string | null>(null)
  const [notes, setNotes] = useState<Record<string, string>>({})

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }
  if (!dpia) return null

  const steps: DPIAStep[] = Array.isArray(dpia.steps)
    ? dpia.steps
    : (typeof dpia.steps === 'string' ? JSON.parse(dpia.steps) : [])

  const handleComplete = (stepId: string) => {
    updateStep.mutate({
      id,
      step: stepId,
      data: { status: 'completed', notes: notes[stepId] ?? '' },
    })
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4">
      <div className="bg-card rounded-xl border border-border w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <div className="sticky top-0 bg-card border-b border-border p-6 flex items-start justify-between">
          <div>
            <div className="flex items-center gap-2 mb-1">
              <span className={cn('text-xs px-2 py-0.5 rounded-full font-medium', RISK_COLORS[dpia.risk_level] ?? RISK_COLORS.medium)}>
                {dpia.risk_level?.toUpperCase()} Risk
              </span>
              <span className={cn('text-xs px-2 py-0.5 rounded-full font-medium', STATUS_COLORS[dpia.status] ?? STATUS_COLORS.in_progress)}>
                {STATUS_LABELS[dpia.status] ?? dpia.status}
              </span>
            </div>
            <h2 className="text-xl font-bold text-foreground">{dpia.name}</h2>
            {dpia.description && <p className="text-sm text-muted-foreground mt-1">{dpia.description}</p>}
          </div>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="p-6 space-y-6">
          {/* Meta */}
          <div className="grid grid-cols-2 gap-4 text-sm">
            {dpia.processing_purpose && (
              <div>
                <p className="text-muted-foreground">Processing Purpose</p>
                <p className="font-medium">{dpia.processing_purpose}</p>
              </div>
            )}
            {Array.isArray(dpia.data_types) && dpia.data_types.length > 0 && (
              <div>
                <p className="text-muted-foreground">Data Types</p>
                <div className="flex flex-wrap gap-1 mt-1">
                  {dpia.data_types.map(dt => (
                    <span key={dt} className="text-xs bg-muted px-2 py-0.5 rounded-full">{dt}</span>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Progress */}
          <div>
            <h3 className="text-sm font-semibold text-foreground mb-3">Workflow Progress</h3>
            <StepProgress steps={steps} />
          </div>

          {/* Steps */}
          <div className="space-y-2">
            {steps.map((step, idx) => (
              <div key={step.id} className="border border-border rounded-lg overflow-hidden">
                <button
                  className="w-full flex items-center gap-3 p-4 hover:bg-muted/50 transition-colors text-left"
                  onClick={() => setExpandedStep(expandedStep === step.id ? null : step.id)}
                >
                  <span className="text-sm text-muted-foreground w-5">{idx + 1}</span>
                  <StepIcon status={step.status} />
                  <span className="flex-1 font-medium text-sm">{STEP_LABELS[step.id] ?? step.name}</span>
                  <span className={cn(
                    'text-xs px-2 py-0.5 rounded-full',
                    step.status === 'completed' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' :
                    step.status === 'in_progress' ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' :
                    step.status === 'skipped' ? 'bg-muted text-muted-foreground' :
                    'bg-muted text-muted-foreground'
                  )}>
                    {step.status.replace('_', ' ')}
                  </span>
                  {expandedStep === step.id ? <ChevronDown className="h-4 w-4 text-muted-foreground" /> : <ChevronRight className="h-4 w-4 text-muted-foreground" />}
                </button>
                {expandedStep === step.id && (
                  <div className="border-t border-border p-4 bg-muted/30 space-y-3">
                    {step.notes && (
                      <p className="text-sm text-muted-foreground italic">"{step.notes}"</p>
                    )}
                    {step.status !== 'completed' && (
                      <>
                        <textarea
                          className="w-full text-sm border border-border rounded-md p-2 bg-background resize-none focus:outline-none focus:ring-1 focus:ring-primary"
                          rows={3}
                          placeholder="Add notes for this step..."
                          value={notes[step.id] ?? ''}
                          onChange={e => setNotes(prev => ({ ...prev, [step.id]: e.target.value }))}
                        />
                        <button
                          className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground text-sm rounded-md hover:opacity-90 transition-opacity disabled:opacity-50"
                          disabled={updateStep.isPending}
                          onClick={() => handleComplete(step.id)}
                        >
                          {updateStep.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <CheckCircle2 className="h-4 w-4" />}
                          Mark Complete
                        </button>
                      </>
                    )}
                    {step.completed_at && (
                      <p className="text-xs text-muted-foreground">
                        Completed: {new Date(step.completed_at).toLocaleString()}
                      </p>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}

function CreateDPIAModal({ onClose }: { onClose: () => void }) {
  const createDPIA = useCreateDPIA()
  const [form, setForm] = useState({
    name: '',
    description: '',
    processing_purpose: '',
    risk_level: 'medium',
    data_types: [] as string[],
  })
  const [dtInput, setDtInput] = useState('')

  const addDT = () => {
    const v = dtInput.trim().toUpperCase()
    if (v && !form.data_types.includes(v)) {
      setForm(f => ({ ...f, data_types: [...f.data_types, v] }))
    }
    setDtInput('')
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    await createDPIA.mutateAsync(form)
    onClose()
  }

  const commonTypes = ['EMAIL', 'SSN', 'PHONE', 'DATE_OF_BIRTH', 'ADDRESS', 'CREDIT_CARD', 'MEDICAL_RECORD']

  return (
    <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4">
      <div className="bg-card rounded-xl border border-border w-full max-w-lg">
        <div className="flex items-center justify-between border-b border-border p-5">
          <h2 className="text-lg font-bold">New DPIA</h2>
          <button onClick={onClose}><X className="h-5 w-5 text-muted-foreground" /></button>
        </div>
        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Name *</label>
            <input
              className="w-full border border-border rounded-md px-3 py-2 text-sm bg-background focus:outline-none focus:ring-1 focus:ring-primary"
              placeholder="e.g. Customer data processing DPIA"
              value={form.name}
              onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Description</label>
            <textarea
              className="w-full border border-border rounded-md px-3 py-2 text-sm bg-background resize-none focus:outline-none focus:ring-1 focus:ring-primary"
              rows={3}
              value={form.description}
              onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Processing Purpose</label>
            <input
              className="w-full border border-border rounded-md px-3 py-2 text-sm bg-background focus:outline-none focus:ring-1 focus:ring-primary"
              placeholder="e.g. Analytics and reporting"
              value={form.processing_purpose}
              onChange={e => setForm(f => ({ ...f, processing_purpose: e.target.value }))}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Risk Level</label>
            <select
              className="w-full border border-border rounded-md px-3 py-2 text-sm bg-background focus:outline-none focus:ring-1 focus:ring-primary"
              value={form.risk_level}
              onChange={e => setForm(f => ({ ...f, risk_level: e.target.value }))}
            >
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">Data Types</label>
            <div className="flex flex-wrap gap-1 mb-2">
              {commonTypes.map(dt => (
                <button
                  key={dt}
                  type="button"
                  onClick={() => setForm(f => ({
                    ...f,
                    data_types: f.data_types.includes(dt)
                      ? f.data_types.filter(x => x !== dt)
                      : [...f.data_types, dt]
                  }))}
                  className={cn(
                    'text-xs px-2 py-1 rounded-full border transition-colors',
                    form.data_types.includes(dt)
                      ? 'bg-primary text-primary-foreground border-primary'
                      : 'border-border hover:bg-muted'
                  )}
                >
                  {dt}
                </button>
              ))}
            </div>
            <div className="flex gap-2">
              <input
                className="flex-1 border border-border rounded-md px-3 py-1.5 text-sm bg-background focus:outline-none focus:ring-1 focus:ring-primary"
                placeholder="Add custom type..."
                value={dtInput}
                onChange={e => setDtInput(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && (e.preventDefault(), addDT())}
              />
              <button type="button" onClick={addDT} className="px-3 py-1.5 bg-muted rounded-md text-sm hover:bg-muted/80">Add</button>
            </div>
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={onClose} className="px-4 py-2 text-sm border border-border rounded-md hover:bg-muted">Cancel</button>
            <button
              type="submit"
              disabled={createDPIA.isPending || !form.name}
              className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-md hover:opacity-90 disabled:opacity-50 flex items-center gap-2"
            >
              {createDPIA.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
              Create DPIA
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default function DPIAPage() {
  const { data: dpias = [], isLoading } = useDPIAs()
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)

  const stats = {
    total: dpias.length,
    completed: dpias.filter(d => d.status === 'completed').length,
    in_progress: dpias.filter(d => d.status === 'in_progress').length,
    high_risk: dpias.filter(d => d.risk_level === 'high').length,
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[
          { label: 'Privacy', href: '/privacy' },
          { label: 'DPIA', active: true },
        ]} />
        <div className="flex items-center justify-between mt-4">
          <div>
            <h1 className="text-3xl font-bold text-foreground">DPIA Management</h1>
            <p className="text-sm text-muted-foreground mt-1">Data Protection Impact Assessments — track risk and compliance</p>
          </div>
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:opacity-90"
          >
            <Plus className="h-4 w-4" />
            New DPIA
          </button>
        </div>
      </div>

      <div className="p-8 space-y-6">
        {/* Stats */}
        <div className="grid grid-cols-4 gap-4">
          {[
            { label: 'Total DPIAs', value: stats.total, icon: FileText, color: 'text-foreground' },
            { label: 'Completed', value: stats.completed, icon: CheckCircle2, color: 'text-green-500' },
            { label: 'In Progress', value: stats.in_progress, icon: Clock, color: 'text-blue-500' },
            { label: 'High Risk', value: stats.high_risk, icon: AlertTriangle, color: 'text-red-500' },
          ].map(stat => (
            <div key={stat.label} className="rounded-lg border border-border bg-card p-5">
              <div className="flex items-center justify-between">
                <p className="text-sm text-muted-foreground">{stat.label}</p>
                <stat.icon className={cn('h-5 w-5', stat.color)} />
              </div>
              <p className="text-3xl font-bold mt-2">{stat.value}</p>
            </div>
          ))}
        </div>

        {/* List */}
        <div className="rounded-lg border border-border bg-card">
          <div className="p-5 border-b border-border">
            <h2 className="font-semibold text-foreground">All DPIAs</h2>
          </div>
          {isLoading ? (
            <div className="flex items-center justify-center py-16 text-muted-foreground">
              <Loader2 className="h-6 w-6 animate-spin mr-2" />
              Loading DPIAs...
            </div>
          ) : dpias.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
              <Shield className="h-12 w-12 mb-3 opacity-30" />
              <p>No DPIAs yet. Create one to start assessing data processing risks.</p>
            </div>
          ) : (
            <div className="divide-y divide-border">
              {dpias.map(dpia => {
                const steps: DPIAStep[] = Array.isArray(dpia.steps)
                  ? dpia.steps
                  : (typeof dpia.steps === 'string' ? JSON.parse(dpia.steps || '[]') : [])
                return (
                  <div
                    key={dpia.id}
                    className="p-5 hover:bg-muted/30 transition-colors cursor-pointer"
                    onClick={() => setSelectedId(dpia.id)}
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <h3 className="font-medium text-foreground truncate">{dpia.name}</h3>
                          <span className={cn('text-xs px-2 py-0.5 rounded-full shrink-0', RISK_COLORS[dpia.risk_level] ?? RISK_COLORS.medium)}>
                            {dpia.risk_level?.toUpperCase()}
                          </span>
                          <span className={cn('text-xs px-2 py-0.5 rounded-full shrink-0', STATUS_COLORS[dpia.status] ?? STATUS_COLORS.in_progress)}>
                            {STATUS_LABELS[dpia.status] ?? dpia.status}
                          </span>
                        </div>
                        {dpia.description && (
                          <p className="text-sm text-muted-foreground truncate">{dpia.description}</p>
                        )}
                        <div className="mt-3 max-w-sm">
                          <StepProgress steps={steps} />
                        </div>
                      </div>
                      <div className="text-right ml-4 shrink-0">
                        <p className="text-xs text-muted-foreground">
                          {new Date(dpia.created_at).toLocaleDateString()}
                        </p>
                        {dpia.dpo_consulted && (
                          <span className="text-xs text-green-600 dark:text-green-400">DPO Consulted ✓</span>
                        )}
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </div>

      {showCreate && <CreateDPIAModal onClose={() => setShowCreate(false)} />}
      {selectedId && <DPIADetail id={selectedId} onClose={() => setSelectedId(null)} />}
    </div>
  )
}
