'use client'

import { use } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { StatusIndicator } from '@/components/base/status-badge'
import { Skeleton } from '@/components/base/skeleton'
import { 
  ArrowLeft, 
  User, 
  Calendar, 
  Clock, 
  FileText, 
  CheckCircle, 
  XCircle,
  Download,
  Play,
  Trash2
} from 'lucide-react'
import { useDSAR, useUpdateDSAR, useDeleteDSAR, useDSARPackage } from '@/hooks/use-privacy'
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

export default function DSARDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params)
  const router = useRouter()
  const { data: dsar, isLoading, error } = useDSAR(id)
  const updateDSAR = useUpdateDSAR()
  const deleteDSAR = useDeleteDSAR()
  const { data: dsarPackage, isLoading: packageLoading, refetch: fetchPackage } = useDSARPackage(id)

  const handleStatusUpdate = async (newStatus: string) => {
    try {
      await updateDSAR.mutateAsync({ id, status: newStatus })
    } catch (error) {
      // Error handled by hook
    }
  }

  const handleDelete = async () => {
    try {
      await deleteDSAR.mutateAsync(id)
      router.push('/privacy/dsar')
    } catch (error) {
      // Error handled by hook
    }
  }

  const handleGeneratePackage = () => {
    fetchPackage()
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background p-8">
        <Skeleton className="h-8 w-64 mb-4" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (error || !dsar) {
    return (
      <div className="min-h-screen bg-background p-8">
        <div className="text-center py-12">
          <XCircle className="h-12 w-12 text-destructive mx-auto mb-4" />
          <h2 className="text-xl font-semibold text-foreground mb-2">DSAR Not Found</h2>
          <p className="text-muted-foreground mb-4">The requested DSAR could not be found.</p>
          <Link
            href="/privacy/dsar"
            className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90"
          >
            <ArrowLeft className="h-4 w-4" />
            Back to DSARs
          </Link>
        </div>
      </div>
    )
  }

  const deadline = new Date(dsar.deadline)
  const isOverdue = deadline < new Date() && dsar.status !== 'completed'
  const daysRemaining = Math.ceil((deadline.getTime() - Date.now()) / (1000 * 60 * 60 * 24))

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Privacy', href: '/privacy' },
            { label: 'DSAR', href: '/privacy/dsar' },
            { label: dsar.subject_id, active: true },
          ]}
        />
        <div className="flex items-center justify-between mt-4">
          <div className="flex items-center gap-4">
            <Link
              href="/privacy/dsar"
              className="p-2 rounded-lg border border-border hover:bg-muted transition-colors"
            >
              <ArrowLeft className="h-5 w-5" />
            </Link>
            <div>
              <h1 className="text-3xl font-bold text-foreground">{dsar.subject_id}</h1>
              <p className="text-sm text-muted-foreground mt-1">
                {dsar.type.charAt(0).toUpperCase() + dsar.type.slice(1)} Request
              </p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            {dsar.status === 'pending' && (
              <button
                onClick={() => handleStatusUpdate('in_progress')}
                disabled={updateDSAR.isPending}
                className="flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700 transition-colors disabled:opacity-50"
              >
                <Play className="h-4 w-4" />
                Start Processing
              </button>
            )}
            {dsar.status === 'in_progress' && (
              <button
                onClick={() => handleStatusUpdate('completed')}
                disabled={updateDSAR.isPending}
                className="flex items-center gap-2 px-4 py-2 rounded-lg bg-green-600 text-white hover:bg-green-700 transition-colors disabled:opacity-50"
              >
                <CheckCircle className="h-4 w-4" />
                Mark Complete
              </button>
            )}
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <button
                  className="flex items-center gap-2 px-4 py-2 rounded-lg border border-destructive text-destructive hover:bg-destructive/10 transition-colors"
                >
                  <Trash2 className="h-4 w-4" />
                  Delete
                </button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Delete DSAR?</AlertDialogTitle>
                  <AlertDialogDescription>
                    This will permanently delete this DSAR request. This action cannot be undone.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={handleDelete}
                    className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                  >
                    Delete
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Status and Timeline */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <div className="rounded-lg border border-border bg-card p-6">
            <div className="flex items-center gap-3 mb-2">
              <div className="p-2 rounded-lg bg-muted">
                <FileText className="h-5 w-5 text-muted-foreground" />
              </div>
              <span className="text-sm text-muted-foreground">Status</span>
            </div>
            <StatusIndicator
              status={dsar.status === 'completed' ? 'success' : dsar.status === 'in_progress' ? 'pending' : 'inactive'}
              label={dsar.status.replace('_', ' ')}
            />
          </div>

          <div className="rounded-lg border border-border bg-card p-6">
            <div className="flex items-center gap-3 mb-2">
              <div className="p-2 rounded-lg bg-muted">
                <User className="h-5 w-5 text-muted-foreground" />
              </div>
              <span className="text-sm text-muted-foreground">Request Type</span>
            </div>
            <p className="text-lg font-semibold text-foreground capitalize">{dsar.type}</p>
          </div>

          <div className="rounded-lg border border-border bg-card p-6">
            <div className="flex items-center gap-3 mb-2">
              <div className="p-2 rounded-lg bg-muted">
                <Calendar className="h-5 w-5 text-muted-foreground" />
              </div>
              <span className="text-sm text-muted-foreground">Deadline</span>
            </div>
            <p className={`text-lg font-semibold ${isOverdue ? 'text-destructive' : 'text-foreground'}`}>
              {deadline.toLocaleDateString()}
            </p>
          </div>

          <div className="rounded-lg border border-border bg-card p-6">
            <div className="flex items-center gap-3 mb-2">
              <div className="p-2 rounded-lg bg-muted">
                <Clock className="h-5 w-5 text-muted-foreground" />
              </div>
              <span className="text-sm text-muted-foreground">Time Remaining</span>
            </div>
            <p className={`text-lg font-semibold ${isOverdue ? 'text-destructive' : daysRemaining <= 7 ? 'text-yellow-600' : 'text-foreground'}`}>
              {isOverdue ? 'Overdue' : `${daysRemaining} days`}
            </p>
          </div>
        </div>

        {/* Request Details */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Request Details</h3>
          <div className="grid grid-cols-2 gap-6">
            <div>
              <label className="text-sm text-muted-foreground">Subject ID</label>
              <p className="text-foreground font-medium">{dsar.subject_id}</p>
            </div>
            <div>
              <label className="text-sm text-muted-foreground">Request ID</label>
              <p className="text-foreground font-mono text-sm">{dsar.id}</p>
            </div>
            <div>
              <label className="text-sm text-muted-foreground">Created</label>
              <p className="text-foreground">{new Date(dsar.created_at).toLocaleString()}</p>
            </div>
            <div>
              <label className="text-sm text-muted-foreground">Last Updated</label>
              <p className="text-foreground">{new Date(dsar.updated_at).toLocaleString()}</p>
            </div>
          </div>
        </div>

        {/* Data Package */}
        <div className="rounded-lg border border-border bg-card p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-foreground">Data Package</h3>
            <button
              onClick={handleGeneratePackage}
              disabled={packageLoading}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Download className="h-4 w-4" />
              {packageLoading ? 'Generating...' : 'Generate Package'}
            </button>
          </div>
          
          {dsarPackage ? (
            <div className="space-y-4">
              <div className="p-4 rounded-lg bg-muted">
                <p className="text-sm text-muted-foreground mb-2">Package Contents</p>
                <pre className="text-sm text-foreground overflow-auto max-h-64">
                  {JSON.stringify(dsarPackage, null, 2)}
                </pre>
              </div>
            </div>
          ) : (
            <p className="text-muted-foreground">
              Click "Generate Package" to search all connected data sources for data related to this subject.
            </p>
          )}
        </div>

        {/* Audit Trail */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Activity Log</h3>
          <div className="space-y-3">
            <div className="flex items-center gap-3 text-sm">
              <div className="w-2 h-2 rounded-full bg-green-500" />
              <span className="text-muted-foreground">{new Date(dsar.created_at).toLocaleString()}</span>
              <span className="text-foreground">DSAR request created</span>
            </div>
            {dsar.status === 'in_progress' && (
              <div className="flex items-center gap-3 text-sm">
                <div className="w-2 h-2 rounded-full bg-blue-500" />
                <span className="text-muted-foreground">{new Date(dsar.updated_at).toLocaleString()}</span>
                <span className="text-foreground">Processing started</span>
              </div>
            )}
            {dsar.status === 'completed' && (
              <div className="flex items-center gap-3 text-sm">
                <div className="w-2 h-2 rounded-full bg-green-500" />
                <span className="text-muted-foreground">{new Date(dsar.updated_at).toLocaleString()}</span>
                <span className="text-foreground">Request completed</span>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
