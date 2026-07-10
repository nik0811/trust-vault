'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { Trash2 } from 'lucide-react'
import { useROTDatasets, useRemediateROT } from '@/hooks/use-advisor'
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

export default function TrivialPage() {
  const { data: datasets, isLoading } = useROTDatasets()
  const remediate = useRemediateROT()
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [dataToDelete, setDataToDelete] = useState<string[] | null>(null)

  const trivialData = Array.isArray(datasets) 
    ? datasets.filter((d: any) => d.category === 'trivial') 
    : []

  const handleDelete = async (ids: string[]) => {
    await remediate.mutateAsync({ dataset_ids: ids, action: 'delete' })
    setDeleteDialogOpen(false)
    setDataToDelete(null)
  }

  const openDeleteDialog = (ids: string[]) => {
    setDataToDelete(ids)
    setDeleteDialogOpen(true)
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'ROT Data', href: '/rot' },
            { label: 'Trivial', active: true },
          ]}
        />
        <h1 className="text-3xl font-bold text-foreground mt-4">Trivial Data</h1>
        <p className="text-sm text-muted-foreground mt-1">Low-value data that can be safely removed</p>
      </div>

      {/* Content */}
      <div className="p-8">
        {isLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
          </div>
        ) : trivialData.length > 0 ? (
          <div className="space-y-4">
            {trivialData.map((item: any) => (
              <div key={item.id} className="rounded-lg border border-border bg-card p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4">
                    <Trash2 className="h-5 w-5 text-gray-500" />
                    <div>
                      <p className="font-medium text-foreground">{item.dataset_id}</p>
                      <p className="text-sm text-muted-foreground">{item.reason}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className="text-sm text-muted-foreground">
                      {(item.size_bytes / 1024 / 1024).toFixed(2)} MB
                    </span>
                    <button
                      onClick={() => openDeleteDialog([item.dataset_id])}
                      disabled={remediate.isPending}
                      className="flex items-center gap-2 px-3 py-1 rounded-lg border border-destructive text-destructive hover:bg-destructive/10 transition-colors disabled:opacity-50"
                    >
                      <Trash2 className="h-4 w-4" />
                      Delete
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <Trash2 className="h-12 w-12 mx-auto text-green-500 mb-4" />
            <p className="text-foreground font-medium">No trivial data found</p>
            <p className="text-sm text-muted-foreground mt-1">
              All your data appears to have value
            </p>
          </div>
        )}
      </div>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Trivial Data</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to permanently delete this data? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setDataToDelete(null)}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => dataToDelete && handleDelete(dataToDelete)}
              disabled={remediate.isPending}
            >
              {remediate.isPending ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
