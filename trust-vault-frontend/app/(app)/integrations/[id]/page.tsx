'use client'

import { useParams, useRouter } from 'next/navigation'
import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { StatusIndicator } from '@/components/base/status-badge'
import { ArrowLeft, RefreshCw, Trash2, Play, TestTube } from 'lucide-react'
import { useIntegration, useDeleteIntegration, useSyncIntegration, useTestIntegration } from '@/hooks/use-jobs'
import Link from 'next/link'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'

export default function IntegrationDetailPage() {
  const params = useParams()
  const router = useRouter()
  const id = params.id as string
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  const { data: integration, isLoading, refetch } = useIntegration(id)
  const deleteIntegration = useDeleteIntegration()
  const syncIntegration = useSyncIntegration()
  const testIntegration = useTestIntegration()

  const handleDelete = async () => {
    try {
      await deleteIntegration.mutateAsync(id)
      setDeleteDialogOpen(false)
      router.push('/integrations')
    } catch (error) {
      // Error handled by hook
    }
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background p-8">
        <Skeleton className="h-8 w-48 mb-4" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (!integration) {
    return (
      <div className="min-h-screen bg-background p-8">
        <div className="text-center py-12">
          <p className="text-destructive">Integration not found</p>
          <Link href="/integrations" className="mt-4 text-primary hover:underline">
            Back to Integrations
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Integrations', href: '/integrations' },
            { label: integration.name, active: true },
          ]}
        />
        <div className="flex items-center justify-between mt-4">
          <div className="flex items-center gap-4">
            <Link href="/integrations" className="p-2 rounded-lg hover:bg-muted transition-colors">
              <ArrowLeft className="h-5 w-5" />
            </Link>
            <div>
              <h1 className="text-3xl font-bold text-foreground">{integration.name}</h1>
              <div className="flex items-center gap-3 mt-1">
                <span className="text-sm text-muted-foreground capitalize">
                  {integration.type.replace('_', ' ')}
                </span>
                <StatusIndicator
                  status={integration.status === 'connected' ? 'success' : integration.status === 'syncing' ? 'pending' : 'error'}
                  label={integration.status}
                />
              </div>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={() => testIntegration.mutate(id)}
              disabled={testIntegration.isPending}
              className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border bg-card text-foreground hover:bg-muted transition-colors disabled:opacity-50"
            >
              <TestTube className="h-4 w-4" />
              Test
            </button>
            <button
              onClick={() => syncIntegration.mutate(id)}
              disabled={syncIntegration.isPending}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Play className="h-4 w-4" />
              Sync Now
            </button>
            <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
              <AlertDialogTrigger asChild>
                <button
                  disabled={deleteIntegration.isPending}
                  className="flex items-center gap-2 px-4 py-2 rounded-lg border border-destructive text-destructive hover:bg-destructive/10 transition-colors disabled:opacity-50"
                >
                  <Trash2 className="h-4 w-4" />
                  Delete
                </button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Delete Integration</AlertDialogTitle>
                  <AlertDialogDescription>
                    Are you sure you want to delete &quot;{integration.name}&quot;? This will stop all syncing and remove the integration configuration. This action cannot be undone.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    variant="destructive"
                    onClick={handleDelete}
                    disabled={deleteIntegration.isPending}
                  >
                    {deleteIntegration.isPending ? 'Deleting...' : 'Delete'}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Details */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Details</h3>
          <div className="grid grid-cols-2 gap-6">
            <div>
              <p className="text-sm text-muted-foreground">Provider</p>
              <p className="text-sm font-medium text-foreground">{integration.provider}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Sync Frequency</p>
              <p className="text-sm font-medium text-foreground">{integration.sync_freq || 'Manual'}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Last Sync</p>
              <p className="text-sm font-medium text-foreground">
                {integration.last_sync && integration.last_sync !== '0001-01-01T00:00:00Z'
                  ? new Date(integration.last_sync).toLocaleString()
                  : 'Never'}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Created</p>
              <p className="text-sm font-medium text-foreground">
                {new Date(integration.created_at).toLocaleString()}
              </p>
            </div>
          </div>
        </div>

        {/* Configuration */}
        {integration.config && Object.keys(integration.config).length > 0 && (
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="text-lg font-semibold text-foreground mb-4">Configuration</h3>
            <pre className="text-sm bg-muted p-4 rounded-lg overflow-auto">
              {JSON.stringify(integration.config, null, 2)}
            </pre>
          </div>
        )}
      </div>
    </div>
  )
}
