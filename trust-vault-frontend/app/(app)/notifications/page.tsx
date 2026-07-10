'use client'

import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { StatusIndicator } from '@/components/base/status-badge'
import { Bell, Loader2 } from 'lucide-react'
import { useNotifications, type Notification } from '@/hooks/use-jobs'
import { toast } from 'sonner'
import { useEffect } from 'react'

const columns: Column<Notification>[] = [
  { id: 'title', header: 'Title', accessorKey: 'title' },
  { id: 'message', header: 'Message', accessorKey: 'message' },
  {
    id: 'severity',
    header: 'Type',
    cell: (row) => (
      <StatusIndicator 
        status={row.severity === 'success' ? 'success' : row.severity === 'error' ? 'error' : row.severity === 'warning' ? 'warning' : 'info'} 
        label={row.severity} 
      />
    ),
  },
  { id: 'created_at', header: 'Time', cell: (row) => new Date(row.created_at).toLocaleString() },
]

export default function NotificationsPage() {
  const { data: notifications, isLoading, error } = useNotifications()

  useEffect(() => {
    if (error) toast.error('Failed to load notifications')
  }, [error])

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs items={[{ label: 'Notifications', active: true }]} />
          <h1 className="text-3xl font-bold text-foreground mt-4">Notifications</h1>
          <p className="text-sm text-muted-foreground mt-1">View all system notifications</p>
        </div>
        <button className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border hover:bg-muted transition-colors">
          <Bell className="h-5 w-5" />
          Preferences
        </button>
      </div>

      <div className="p-8">
        <div className="rounded-lg border border-border bg-card p-6">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : notifications?.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              <Bell className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>No notifications yet</p>
            </div>
          ) : (
            <DataTable columns={columns} data={notifications || []} />
          )}
        </div>
      </div>
    </div>
  )
}
