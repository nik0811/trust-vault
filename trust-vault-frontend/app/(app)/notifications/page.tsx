'use client'

import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { StatusIndicator } from '@/components/base/status-badge'
import { Bell, Loader2, CheckCheck, Trash2, Check } from 'lucide-react'
import {
  useNotifications,
  useMarkNotificationRead,
  useMarkAllNotificationsRead,
  useDeleteNotification,
  type Notification,
} from '@/hooks/use-jobs'
import { toast } from 'sonner'
import { useEffect } from 'react'

export default function NotificationsPage() {
  const { data: notifications, isLoading, error } = useNotifications()
  const markRead = useMarkNotificationRead()
  const markAllRead = useMarkAllNotificationsRead()
  const deleteNotif = useDeleteNotification()

  useEffect(() => {
    if (error) toast.error('Failed to load notifications')
  }, [error])

  const columns: Column<Notification>[] = [
    {
      id: 'title',
      header: 'Title',
      cell: (row) => (
        <span className={row.read ? 'text-muted-foreground' : 'font-medium text-foreground'}>
          {row.title}
        </span>
      ),
    },
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
    {
      id: 'actions',
      header: '',
      cell: (row) => (
        <div className="flex items-center gap-2 justify-end">
          {!row.read && (
            <button
              onClick={() => markRead.mutate(row.id)}
              className="p-1 rounded hover:bg-muted transition-colors"
              title="Mark as read"
            >
              <Check className="h-4 w-4 text-muted-foreground" />
            </button>
          )}
          <button
            onClick={() => deleteNotif.mutate(row.id)}
            className="p-1 rounded hover:bg-destructive/10 transition-colors"
            title="Delete"
          >
            <Trash2 className="h-4 w-4 text-muted-foreground hover:text-destructive" />
          </button>
        </div>
      ),
    },
  ]

  const unreadCount = notifications?.filter(n => !n.read).length ?? 0

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs items={[{ label: 'Notifications', active: true }]} />
          <h1 className="text-3xl font-bold text-foreground mt-4">Notifications</h1>
          <p className="text-sm text-muted-foreground mt-1">
            View all system notifications
            {unreadCount > 0 && (
              <span className="ml-2 px-2 py-0.5 rounded-full bg-primary text-primary-foreground text-xs">
                {unreadCount} unread
              </span>
            )}
          </p>
        </div>
        <div className="flex gap-2">
          {unreadCount > 0 && (
            <button
              onClick={() => markAllRead.mutate()}
              disabled={markAllRead.isPending}
              className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border hover:bg-muted transition-colors text-sm"
            >
              <CheckCheck className="h-4 w-4" />
              Mark All Read
            </button>
          )}
          <button className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border hover:bg-muted transition-colors">
            <Bell className="h-5 w-5" />
            Preferences
          </button>
        </div>
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
