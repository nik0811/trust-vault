'use client'

import { useState } from 'react'
import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { StatusIndicator } from '@/components/base/status-badge'
import { Plus, Trash2, RefreshCw, Mail, X, Send, Clock } from 'lucide-react'
import { useUsers, useCreateUser, useDeleteUser, useInvitations, useCreateInvitation, useCancelInvitation, useResendInvitation, type Invitation } from '@/hooks/use-auth'
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

interface User {
  id: string
  email: string
  name: string
  status: string
  is_super_admin: boolean
  created_at: string
}

const columns: Column<User>[] = [
  {
    id: 'name',
    header: 'Name',
    cell: (row) => <span className="font-medium text-foreground">{row.name}</span>,
    sortable: true,
  },
  {
    id: 'email',
    header: 'Email',
    accessorKey: 'email',
    sortable: true,
  },
  {
    id: 'status',
    header: 'Status',
    cell: (row) => (
      <StatusIndicator
        status={row.status === 'active' ? 'success' : 'inactive'}
        label={row.status}
      />
    ),
  },
  {
    id: 'role',
    header: 'Role',
    cell: (row) => (
      <span className={`px-2 py-0.5 rounded text-sm ${
        row.is_super_admin ? 'bg-purple-500/10 text-purple-600' : 'bg-muted text-foreground'
      }`}>
        {row.is_super_admin ? 'Super Admin' : 'User'}
      </span>
    ),
  },
  {
    id: 'created_at',
    header: 'Created',
    cell: (row) => new Date(row.created_at).toLocaleDateString(),
    sortable: true,
  },
]

const invitationColumns: Column<Invitation>[] = [
  {
    id: 'email',
    header: 'Email',
    cell: (row) => <span className="font-medium text-foreground">{row.email}</span>,
    sortable: true,
  },
  {
    id: 'role',
    header: 'Role',
    cell: (row) => (
      <span className="px-2 py-0.5 rounded text-sm bg-muted text-foreground capitalize">
        {row.role.replace('_', ' ')}
      </span>
    ),
  },
  {
    id: 'status',
    header: 'Status',
    cell: (row) => {
      const isExpired = new Date(row.expires_at) < new Date()
      const isAccepted = !!row.accepted_at
      return (
        <StatusIndicator
          status={isAccepted ? 'success' : isExpired ? 'error' : 'warning'}
          label={isAccepted ? 'Accepted' : isExpired ? 'Expired' : 'Pending'}
        />
      )
    },
  },
  {
    id: 'expires_at',
    header: 'Expires',
    cell: (row) => new Date(row.expires_at).toLocaleDateString(),
    sortable: true,
  },
]

export default function UsersPage() {
  const { data: users, isLoading, refetch } = useUsers()
  const { data: invitations, isLoading: invitationsLoading, refetch: refetchInvitations } = useInvitations()
  const createUser = useCreateUser()
  const deleteUser = useDeleteUser()
  const createInvitation = useCreateInvitation()
  const cancelInvitation = useCancelInvitation()
  const resendInvitation = useResendInvitation()
  
  const [showForm, setShowForm] = useState(false)
  const [showInviteModal, setShowInviteModal] = useState(false)
  const [formData, setFormData] = useState({ email: '', password: '', name: '' })
  const [inviteData, setInviteData] = useState({ email: '', role: 'user' })
  const [activeTab, setActiveTab] = useState<'users' | 'invitations'>('users')
  const [deleteUserDialogOpen, setDeleteUserDialogOpen] = useState(false)
  const [userToDelete, setUserToDelete] = useState<string | null>(null)
  const [cancelInviteDialogOpen, setCancelInviteDialogOpen] = useState(false)
  const [inviteToCancel, setInviteToCancel] = useState<string | null>(null)

  const usersData = Array.isArray(users) ? users : []
  const invitationsData = Array.isArray(invitations) ? invitations : []
  const pendingInvitations = invitationsData.filter(i => !i.accepted_at && new Date(i.expires_at) > new Date())

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await createUser.mutateAsync(formData)
      setShowForm(false)
      setFormData({ email: '', password: '', name: '' })
    } catch (error) {
      // Error handled by hook
    }
  }

  const handleInvite = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await createInvitation.mutateAsync(inviteData)
      setShowInviteModal(false)
      setInviteData({ email: '', role: 'user' })
    } catch (error) {
      // Error handled by hook
    }
  }

  const handleDelete = async (id: string) => {
    await deleteUser.mutateAsync(id)
    setDeleteUserDialogOpen(false)
    setUserToDelete(null)
  }

  const handleCancelInvitation = async (id: string) => {
    await cancelInvitation.mutateAsync(id)
    setCancelInviteDialogOpen(false)
    setInviteToCancel(null)
  }

  const openDeleteUserDialog = (id: string) => {
    setUserToDelete(id)
    setDeleteUserDialogOpen(true)
  }

  const openCancelInviteDialog = (id: string) => {
    setInviteToCancel(id)
    setCancelInviteDialogOpen(true)
  }

  const handleResendInvitation = async (id: string) => {
    await resendInvitation.mutateAsync(id)
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs
            items={[
              { label: 'Settings', href: '/settings' },
              { label: 'Users', active: true },
            ]}
          />
          <h1 className="text-3xl font-bold text-foreground mt-4">Users</h1>
          <p className="text-sm text-muted-foreground mt-1">Manage user accounts and invitations</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={() => { refetch(); refetchInvitations(); }}
            className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border bg-card text-foreground hover:bg-muted transition-colors"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh
          </button>
          <button
            onClick={() => setShowInviteModal(true)}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
          >
            <Mail className="h-5 w-5" />
            Invite User
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Tabs */}
        <div className="flex gap-4 border-b border-border">
          <button
            onClick={() => setActiveTab('users')}
            className={`pb-3 px-1 text-sm font-medium transition-colors ${
              activeTab === 'users'
                ? 'text-primary border-b-2 border-primary'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            Users ({usersData.length})
          </button>
          <button
            onClick={() => setActiveTab('invitations')}
            className={`pb-3 px-1 text-sm font-medium transition-colors ${
              activeTab === 'invitations'
                ? 'text-primary border-b-2 border-primary'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            Pending Invitations ({pendingInvitations.length})
          </button>
        </div>

        {/* Create Form (legacy - keeping for direct user creation) */}
        {showForm && (
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="text-lg font-semibold text-foreground mb-4">Create User</h3>
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Name</label>
                  <input
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="John Doe"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Email</label>
                  <input
                    type="email"
                    value={formData.email}
                    onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                    placeholder="john@example.com"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Password</label>
                  <input
                    type="password"
                    value={formData.password}
                    onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                    placeholder="••••••••"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    required
                  />
                </div>
              </div>
              <div className="flex gap-3">
                <button
                  type="submit"
                  disabled={createUser.isPending}
                  className="px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                  {createUser.isPending ? 'Creating...' : 'Create User'}
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

        {/* Users Table */}
        {activeTab === 'users' && (
          <>
            {isLoading ? (
              <div className="space-y-4">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </div>
            ) : usersData.length > 0 ? (
              <DataTable
                columns={[
                  ...columns,
                  {
                    id: 'actions',
                    header: '',
                    cell: (row) => (
                      <button
                        onClick={() => openDeleteUserDialog(row.id)}
                        disabled={row.is_super_admin}
                        className="p-2 text-destructive hover:bg-destructive/10 rounded-lg transition-colors disabled:opacity-50"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    ),
                  },
                ]}
                data={usersData}
              />
            ) : (
              <div className="text-center py-12">
                <p className="text-muted-foreground">No users found</p>
              </div>
            )}
          </>
        )}

        {/* Invitations Table */}
        {activeTab === 'invitations' && (
          <>
            {invitationsLoading ? (
              <div className="space-y-4">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </div>
            ) : invitationsData.length > 0 ? (
              <DataTable
                columns={[
                  ...invitationColumns,
                  {
                    id: 'actions',
                    header: '',
                    cell: (row) => {
                      const isExpired = new Date(row.expires_at) < new Date()
                      const isAccepted = !!row.accepted_at
                      if (isAccepted) return null
                      return (
                        <div className="flex gap-2">
                          {isExpired && (
                            <button
                              onClick={() => handleResendInvitation(row.id)}
                              className="p-2 text-primary hover:bg-primary/10 rounded-lg transition-colors"
                              title="Resend invitation"
                            >
                              <Send className="h-4 w-4" />
                            </button>
                          )}
                          <button
                            onClick={() => openCancelInviteDialog(row.id)}
                            className="p-2 text-destructive hover:bg-destructive/10 rounded-lg transition-colors"
                            title="Cancel invitation"
                          >
                            <Trash2 className="h-4 w-4" />
                          </button>
                        </div>
                      )
                    },
                  },
                ]}
                data={invitationsData}
              />
            ) : (
              <div className="text-center py-12">
                <Clock className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                <p className="text-muted-foreground">No pending invitations</p>
                <button
                  onClick={() => setShowInviteModal(true)}
                  className="mt-4 text-primary hover:underline"
                >
                  Invite your first user
                </button>
              </div>
            )}
          </>
        )}
      </div>

      {/* Invite Modal */}
      {showInviteModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-card rounded-lg border border-border shadow-xl w-full max-w-md mx-4">
            <div className="flex items-center justify-between p-6 border-b border-border">
              <h2 className="text-lg font-semibold text-foreground">Invite User</h2>
              <button
                onClick={() => setShowInviteModal(false)}
                className="p-2 hover:bg-muted rounded-lg transition-colors"
              >
                <X className="h-5 w-5 text-muted-foreground" />
              </button>
            </div>
            <form onSubmit={handleInvite} className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-foreground mb-2">Email Address</label>
                <input
                  type="email"
                  value={inviteData.email}
                  onChange={(e) => setInviteData({ ...inviteData, email: e.target.value })}
                  placeholder="colleague@company.com"
                  className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-2">Role</label>
                <select
                  value={inviteData.role}
                  onChange={(e) => setInviteData({ ...inviteData, role: e.target.value })}
                  className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                >
                  <option value="user">User</option>
                  <option value="analyst">Analyst</option>
                  <option value="admin">Admin</option>
                  <option value="tenant_admin">Tenant Admin</option>
                </select>
                <p className="text-xs text-muted-foreground mt-1">
                  Only superadmins can invite tenant admins
                </p>
              </div>
              <div className="flex gap-3 pt-4">
                <button
                  type="submit"
                  disabled={createInvitation.isPending}
                  className="flex-1 py-2 rounded-lg bg-primary text-primary-foreground font-medium hover:bg-primary/90 disabled:opacity-50 transition-colors"
                >
                  {createInvitation.isPending ? 'Sending...' : 'Send Invitation'}
                </button>
                <button
                  type="button"
                  onClick={() => setShowInviteModal(false)}
                  className="px-4 py-2 rounded-lg border border-border text-foreground hover:bg-muted transition-colors"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Delete User Dialog */}
      <AlertDialog open={deleteUserDialogOpen} onOpenChange={setDeleteUserDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete User</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this user? They will lose access to the platform immediately. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setUserToDelete(null)}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => userToDelete && handleDelete(userToDelete)}
              disabled={deleteUser.isPending}
            >
              {deleteUser.isPending ? 'Deleting...' : 'Delete User'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Cancel Invitation Dialog */}
      <AlertDialog open={cancelInviteDialogOpen} onOpenChange={setCancelInviteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Cancel Invitation</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to cancel this invitation? The invitation link will no longer work.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setInviteToCancel(null)}>Keep Invitation</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => inviteToCancel && handleCancelInvitation(inviteToCancel)}
              disabled={cancelInvitation.isPending}
            >
              {cancelInvitation.isPending ? 'Cancelling...' : 'Cancel Invitation'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
