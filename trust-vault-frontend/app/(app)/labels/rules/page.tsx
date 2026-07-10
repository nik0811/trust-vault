'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { Plus } from 'lucide-react'
import { useLabelRules, useCreateLabelRule } from '@/hooks/use-advisor'

export default function LabelRulesPage() {
  const { data: rules, isLoading } = useLabelRules()
  const createRule = useCreateLabelRule()
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState({ classification: '', label: 'INTERNAL' })

  const rulesData = Array.isArray(rules) ? rules : []

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await createRule.mutateAsync(formData)
      setShowForm(false)
      setFormData({ classification: '', label: 'INTERNAL' })
    } catch (error) {
      // Error handled by hook
    }
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs
            items={[
              { label: 'Sensitivity Labels', href: '/labels' },
              { label: 'Rules', active: true },
            ]}
          />
          <h1 className="text-3xl font-bold text-foreground mt-4">Label Rules</h1>
          <p className="text-sm text-muted-foreground mt-1">Configure automatic label assignment</p>
        </div>
        <button
          onClick={() => setShowForm(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
        >
          <Plus className="h-5 w-5" />
          Add Rule
        </button>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Create Form */}
        {showForm && (
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="text-lg font-semibold text-foreground mb-4">Create Label Rule</h3>
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Classification</label>
                  <input
                    type="text"
                    value={formData.classification}
                    onChange={(e) => setFormData({ ...formData, classification: e.target.value })}
                    placeholder="e.g., SSN, CREDIT_CARD, PHI"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Assign Label</label>
                  <select
                    value={formData.label}
                    onChange={(e) => setFormData({ ...formData, label: e.target.value })}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  >
                    <option value="PUBLIC">Public</option>
                    <option value="INTERNAL">Internal</option>
                    <option value="CONFIDENTIAL">Confidential</option>
                    <option value="HIGHLY_CONFIDENTIAL">Highly Confidential</option>
                    <option value="RESTRICTED">Restricted</option>
                  </select>
                </div>
              </div>
              <div className="flex gap-3">
                <button
                  type="submit"
                  disabled={createRule.isPending}
                  className="px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                  {createRule.isPending ? 'Creating...' : 'Create Rule'}
                </button>
                <button
                  type="button"
                  onClick={() => setShowForm(false)}
                  className="px-4 py-2 rounded-lg border border-border text-foreground hover:bg-muted transition-colors"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        )}

        {/* Rules List */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Active Rules</h3>
          {isLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : rulesData.length > 0 ? (
            <div className="space-y-3">
              {rulesData.map((rule: any, i: number) => (
                <div key={i} className="flex items-center justify-between p-4 rounded-lg bg-muted/50">
                  <div className="flex items-center gap-4">
                    <span className="px-2 py-0.5 rounded bg-primary/10 text-primary text-sm">
                      {rule.classification}
                    </span>
                    <span className="text-muted-foreground">→</span>
                    <span className={`px-2 py-0.5 rounded text-sm ${
                      rule.label === 'RESTRICTED' ? 'bg-red-500/10 text-red-600' :
                      rule.label === 'CONFIDENTIAL' || rule.label === 'HIGHLY_CONFIDENTIAL' ? 'bg-yellow-500/10 text-yellow-600' :
                      rule.label === 'INTERNAL' ? 'bg-blue-500/10 text-blue-600' :
                      'bg-green-500/10 text-green-600'
                    }`}>
                      {rule.label}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <p className="text-muted-foreground">No label rules configured</p>
              <p className="text-sm text-muted-foreground mt-1">
                Create rules to automatically assign labels based on classification
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
