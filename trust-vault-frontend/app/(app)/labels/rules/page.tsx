'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { Plus, Pencil, Trash2, X } from 'lucide-react'
import { useLabelRules, useCreateLabelRule, useUpdateLabelRule, useDeleteLabelRule } from '@/hooks/use-advisor'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/base/alert-dialog'

export default function LabelRulesPage() {
  const { data: rules, isLoading } = useLabelRules()
  const createRule = useCreateLabelRule()
  const updateRule = useUpdateLabelRule()
  const deleteRule = useDeleteLabelRule()
  const [showForm, setShowForm] = useState(false)
  const [editingRule, setEditingRule] = useState<any>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [ruleToDelete, setRuleToDelete] = useState<any>(null)
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

  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingRule) return
    try {
      await updateRule.mutateAsync({ id: editingRule.id, ...formData })
      setEditingRule(null)
      setFormData({ classification: '', label: 'INTERNAL' })
    } catch (error) {
      // Error handled by hook
    }
  }

  const handleDelete = async () => {
    if (!ruleToDelete) return
    try {
      await deleteRule.mutateAsync(ruleToDelete.id)
      setDeleteDialogOpen(false)
      setRuleToDelete(null)
    } catch (error) {
      // Error handled by hook
    }
  }

  const startEdit = (rule: any) => {
    setEditingRule(rule)
    setFormData({ classification: rule.classification, label: rule.label })
    setShowForm(false)
  }

  const cancelEdit = () => {
    setEditingRule(null)
    setFormData({ classification: '', label: 'INTERNAL' })
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
          onClick={() => { setShowForm(true); setEditingRule(null); setFormData({ classification: '', label: 'INTERNAL' }); }}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
        >
          <Plus className="h-5 w-5" />
          Add Rule
        </button>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Create/Edit Form */}
        {(showForm || editingRule) && (
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="text-lg font-semibold text-foreground mb-4">
              {editingRule ? 'Edit Label Rule' : 'Create Label Rule'}
            </h3>
            <form onSubmit={editingRule ? handleUpdate : handleCreate} className="space-y-4">
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
                  disabled={createRule.isPending || updateRule.isPending}
                  className="px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                  {editingRule 
                    ? (updateRule.isPending ? 'Saving...' : 'Save Changes')
                    : (createRule.isPending ? 'Creating...' : 'Create Rule')
                  }
                </button>
                <button
                  type="button"
                  onClick={() => { setShowForm(false); cancelEdit(); }}
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
              {rulesData.map((rule: any) => (
                <div key={rule.id} className="flex items-center justify-between p-4 rounded-lg bg-muted/50">
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
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => startEdit(rule)}
                      className="p-2 rounded-lg hover:bg-muted transition-colors text-muted-foreground hover:text-foreground"
                      title="Edit rule"
                    >
                      <Pencil className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => { setRuleToDelete(rule); setDeleteDialogOpen(true); }}
                      className="p-2 rounded-lg hover:bg-destructive/10 transition-colors text-muted-foreground hover:text-destructive"
                      title="Delete rule"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
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

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Label Rule</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete the rule for &quot;{ruleToDelete?.classification}&quot;? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteRule.isPending}
            >
              {deleteRule.isPending ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
