'use client'

import { use, useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { StatusIndicator } from '@/components/base/status-badge'
import { ArrowLeft, Pencil, Trash2, Save, X } from 'lucide-react'
import { usePolicy, useUpdatePolicy, useDeletePolicy } from '@/hooks/use-policies'
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

export default function PolicyDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params)
  const router = useRouter()
  const { data: policy, isLoading, error } = usePolicy(id)
  const updatePolicy = useUpdatePolicy()
  const deletePolicy = useDeletePolicy()
  
  const [isEditing, setIsEditing] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    type: 'access' as const,
    active: true,
    priority: 1,
  })

  const startEdit = () => {
    if (policy) {
      setFormData({
        name: policy.name,
        description: policy.description || '',
        type: policy.type,
        active: policy.active,
        priority: policy.priority,
      })
      setIsEditing(true)
    }
  }

  const cancelEdit = () => {
    setIsEditing(false)
  }

  const handleSave = async () => {
    try {
      await updatePolicy.mutateAsync({ id, data: formData })
      setIsEditing(false)
    } catch (error) {
      // Error handled by hook
    }
  }

  const handleDelete = async () => {
    try {
      await deletePolicy.mutateAsync(id)
      router.push('/governance/policies')
    } catch (error) {
      // Error handled by hook
    }
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background p-8">
        <Skeleton className="h-8 w-64 mb-4" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (error || !policy) {
    return (
      <div className="min-h-screen bg-background p-8">
        <div className="text-center py-12">
          <p className="text-destructive text-lg">Policy not found</p>
          <Link href="/governance/policies" className="mt-4 text-primary hover:underline">
            Back to policies
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <div className="flex items-center justify-between">
          <div>
            <Breadcrumbs
              items={[
                { label: 'Governance', href: '/governance' },
                { label: 'Policies', href: '/governance/policies' },
                { label: policy.name, active: true },
              ]}
            />
            <div className="flex items-center gap-4 mt-4">
              <Link
                href="/governance/policies"
                className="p-2 rounded-lg hover:bg-muted transition-colors"
              >
                <ArrowLeft className="h-5 w-5 text-muted-foreground" />
              </Link>
              <div>
                <h1 className="text-3xl font-bold text-foreground">{policy.name}</h1>
                <div className="flex items-center gap-3 mt-1">
                  <span className="px-2 py-0.5 rounded bg-muted text-foreground text-sm capitalize">
                    {policy.type}
                  </span>
                  <StatusIndicator
                    status={policy.active ? 'success' : 'inactive'}
                    label={policy.active ? 'Active' : 'Inactive'}
                  />
                </div>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-3">
            {isEditing ? (
              <>
                <button
                  onClick={cancelEdit}
                  className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border text-foreground hover:bg-muted transition-colors"
                >
                  <X className="h-4 w-4" />
                  Cancel
                </button>
                <button
                  onClick={handleSave}
                  disabled={updatePolicy.isPending}
                  className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                  <Save className="h-4 w-4" />
                  {updatePolicy.isPending ? 'Saving...' : 'Save Changes'}
                </button>
              </>
            ) : (
              <>
                <button
                  onClick={startEdit}
                  className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border text-foreground hover:bg-muted transition-colors"
                >
                  <Pencil className="h-4 w-4" />
                  Edit
                </button>
                <button
                  onClick={() => setDeleteDialogOpen(true)}
                  className="flex items-center gap-2 px-4 py-2 rounded-lg border border-destructive text-destructive hover:bg-destructive/10 transition-colors"
                >
                  <Trash2 className="h-4 w-4" />
                  Delete
                </button>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {isEditing ? (
          /* Edit Form */
          <div className="rounded-lg border border-border bg-card p-6 space-y-6">
            <div className="grid grid-cols-2 gap-6">
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">Name</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">Type</label>
                <select
                  value={formData.type}
                  onChange={(e) => setFormData({ ...formData, type: e.target.value as any })}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                >
                  <option value="access">Access Control</option>
                  <option value="redaction">Redaction</option>
                  <option value="ai">AI Governance</option>
                  <option value="retention">Retention</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">Priority</label>
                <input
                  type="number"
                  value={formData.priority}
                  onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) || 1 })}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>
              <div className="flex items-center gap-3">
                <label className="block text-sm font-medium text-foreground">Active</label>
                <button
                  type="button"
                  onClick={() => setFormData({ ...formData, active: !formData.active })}
                  className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                    formData.active ? 'bg-primary' : 'bg-muted'
                  }`}
                >
                  <span
                    className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                      formData.active ? 'translate-x-6' : 'translate-x-1'
                    }`}
                  />
                </button>
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Description</label>
              <textarea
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                rows={3}
                className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>
          </div>
        ) : (
          /* View Mode */
          <>
            {/* Details Card */}
            <div className="rounded-lg border border-border bg-card p-6">
              <h3 className="text-lg font-semibold text-foreground mb-4">Details</h3>
              <div className="grid grid-cols-2 gap-6">
                <div>
                  <p className="text-sm text-muted-foreground">ID</p>
                  <p className="text-sm font-mono text-foreground">{policy.id}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Type</p>
                  <p className="text-sm text-foreground capitalize">{policy.type}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Priority</p>
                  <p className="text-sm text-foreground">{policy.priority}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Status</p>
                  <StatusIndicator
                    status={policy.active ? 'success' : 'inactive'}
                    label={policy.active ? 'Active' : 'Inactive'}
                  />
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Created</p>
                  <p className="text-sm text-foreground">{new Date(policy.created_at).toLocaleString()}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Updated</p>
                  <p className="text-sm text-foreground">{new Date(policy.updated_at).toLocaleString()}</p>
                </div>
              </div>
              {policy.description && (
                <div className="mt-6">
                  <p className="text-sm text-muted-foreground">Description</p>
                  <p className="text-sm text-foreground mt-1">{policy.description}</p>
                </div>
              )}
            </div>

            {/* Conditions Card */}
            {policy.conditions && Object.keys(policy.conditions).length > 0 && (
              <div className="rounded-lg border border-border bg-card p-6">
                <h3 className="text-lg font-semibold text-foreground mb-4">Conditions</h3>
                <pre className="text-sm bg-muted p-4 rounded-lg overflow-auto">
                  {JSON.stringify(policy.conditions, null, 2)}
                </pre>
              </div>
            )}

            {/* Actions Card */}
            {policy.actions && Object.keys(policy.actions).length > 0 && (
              <div className="rounded-lg border border-border bg-card p-6">
                <h3 className="text-lg font-semibold text-foreground mb-4">Actions</h3>
                <pre className="text-sm bg-muted p-4 rounded-lg overflow-auto">
                  {JSON.stringify(policy.actions, null, 2)}
                </pre>
              </div>
            )}

            {/* Regulations Card */}
            {policy.regulations && policy.regulations.length > 0 && (
              <div className="rounded-lg border border-border bg-card p-6">
                <h3 className="text-lg font-semibold text-foreground mb-4">Regulations</h3>
                <div className="flex flex-wrap gap-2">
                  {policy.regulations.map((reg, i) => (
                    <span key={i} className="px-3 py-1 rounded-full bg-muted text-foreground text-sm">
                      {reg}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </div>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Policy</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete &quot;{policy.name}&quot;? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={handleDelete}
              disabled={deletePolicy.isPending}
            >
              {deletePolicy.isPending ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
