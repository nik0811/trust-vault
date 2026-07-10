'use client'

import { useState } from 'react'
import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { EmptyState } from '@/components/base/empty-state'
import Link from 'next/link'
import { Plus, Trash2, FileCode } from 'lucide-react'
import { useClassificationRules, useCreateClassificationRule, useDeleteClassificationRule } from '@/hooks/use-classification'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'

interface Rule {
  id: string
  name: string
  pattern: string
  entity_type: string
  priority: number
  created_at: string
}

const columns: Column<Rule>[] = [
  {
    id: 'name',
    header: 'Name',
    accessorKey: 'name',
    sortable: true,
  },
  {
    id: 'entity_type',
    header: 'Entity Type',
    cell: (row) => (
      <span className="px-2 py-0.5 rounded bg-primary/10 text-primary text-sm">{row.entity_type}</span>
    ),
  },
  {
    id: 'pattern',
    header: 'Pattern',
    cell: (row) => (
      <code className="text-sm bg-muted px-2 py-1 rounded">{row.pattern}</code>
    ),
  },
  {
    id: 'priority',
    header: 'Priority',
    accessorKey: 'priority',
    sortable: true,
  },
]

export default function ClassificationRulesPage() {
  const { data: rules, isLoading, refetch } = useClassificationRules()
  const createRule = useCreateClassificationRule()
  const deleteRule = useDeleteClassificationRule()

  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    pattern: '',
    entity_type: '',
    priority: 100,
  })
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [ruleToDelete, setRuleToDelete] = useState<string | null>(null)

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await createRule.mutateAsync(formData)
      setShowForm(false)
      setFormData({ name: '', pattern: '', entity_type: '', priority: 100 })
    } catch (error) {
      // Error handled by hook
    }
  }

  const handleDelete = async (id: string) => {
    await deleteRule.mutateAsync(id)
    setDeleteDialogOpen(false)
    setRuleToDelete(null)
  }

  const openDeleteDialog = (id: string) => {
    setRuleToDelete(id)
    setDeleteDialogOpen(true)
  }

  const rulesData = Array.isArray(rules) ? rules : []

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs
            items={[
              { label: 'Classification', href: '/classification' },
              { label: 'Rules', active: true },
            ]}
          />
          <h1 className="text-3xl font-bold text-foreground mt-4">Classification Rules</h1>
          <p className="text-sm text-muted-foreground mt-1">Custom regex patterns for data classification</p>
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
        {/* Navigation Tabs */}
        <div className="flex gap-4 border-b border-border pb-4">
          <Link href="/classification" className="text-muted-foreground hover:text-foreground transition-colors pb-2">
            Datasets
          </Link>
          <Link href="/classification/rules" className="text-foreground font-medium border-b-2 border-primary pb-2">
            Rules
          </Link>
          <Link href="/classification/models" className="text-muted-foreground hover:text-foreground transition-colors pb-2">
            Models
          </Link>
        </div>

        {/* Create Form */}
        {showForm && (
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="text-lg font-semibold text-foreground mb-4">Create Classification Rule</h3>
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Name</label>
                  <input
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="Custom SSN Pattern"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Entity Type</label>
                  <input
                    type="text"
                    value={formData.entity_type}
                    onChange={(e) => setFormData({ ...formData, entity_type: e.target.value })}
                    placeholder="SSN"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    required
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">Regex Pattern</label>
                <input
                  type="text"
                  value={formData.pattern}
                  onChange={(e) => setFormData({ ...formData, pattern: e.target.value })}
                  placeholder="\d{3}-\d{2}-\d{4}"
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground font-mono focus:outline-none focus:ring-2 focus:ring-primary"
                  required
                />
              </div>
              <div className="w-32">
                <label className="block text-sm font-medium text-foreground mb-1">Priority</label>
                <input
                  type="number"
                  value={formData.priority}
                  onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) })}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
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

        {/* Rules Table */}
        <div className="rounded-lg border border-border bg-card p-6">
          {isLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : rulesData.length > 0 ? (
            <DataTable
              columns={[
                ...columns,
                {
                  id: 'actions',
                  header: '',
                  cell: (row) => (
                    <button
                      onClick={() => openDeleteDialog(row.id)}
                      className="p-2 text-destructive hover:bg-destructive/10 rounded-lg transition-colors"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  ),
                },
              ]}
              data={rulesData}
            />
          ) : (
            <EmptyState
              icon={<FileCode className="h-12 w-12" />}
              title="No custom rules"
              description="Create custom regex patterns to enhance classification accuracy."
              action={
                <button
                  onClick={() => setShowForm(true)}
                  className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
                >
                  <Plus className="h-5 w-5" />
                  Add Rule
                </button>
              }
            />
          )}
        </div>
      </div>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Classification Rule</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this classification rule? Future classifications will no longer use this pattern. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setRuleToDelete(null)}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => ruleToDelete && handleDelete(ruleToDelete)}
              disabled={deleteRule.isPending}
            >
              {deleteRule.isPending ? 'Deleting...' : 'Delete Rule'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
