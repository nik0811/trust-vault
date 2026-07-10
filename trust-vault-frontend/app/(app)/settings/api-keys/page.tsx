'use client'

import { useState, useEffect } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Key, Plus, Copy, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { useAPIKeys, useCreateAPIKey, useDeleteAPIKey } from '@/hooks/use-auth'
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

export default function APIKeysPage() {
  const { data: keysRaw, isLoading, error } = useAPIKeys()
  const createKey = useCreateAPIKey()
  const deleteKey = useDeleteAPIKey()
  
  const [showForm, setShowForm] = useState(false)
  const [keyName, setKeyName] = useState('')
  const [generatedKey, setGeneratedKey] = useState<string | null>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [keyToDelete, setKeyToDelete] = useState<string | null>(null)

  useEffect(() => {
    if (error) toast.error('Failed to load API keys')
  }, [error])

  const keys = keysRaw || []

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!keyName.trim()) {
      toast.error('Please enter a key name')
      return
    }
    try {
      const result = await createKey.mutateAsync({ name: keyName })
      setGeneratedKey(result.key)
    } catch {
      // Error handled by hook
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteKey.mutateAsync(id)
      setDeleteDialogOpen(false)
      setKeyToDelete(null)
    } catch {
      // Error handled by hook
    }
  }

  const openDeleteDialog = (id: string) => {
    setKeyToDelete(id)
    setDeleteDialogOpen(true)
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    toast.success('Copied to clipboard')
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6 flex items-center justify-between">
        <div>
          <Breadcrumbs
            items={[
              { label: 'Settings', href: '/settings' },
              { label: 'API Keys', active: true },
            ]}
          />
          <h1 className="text-3xl font-bold text-foreground mt-4">API Keys</h1>
          <p className="text-sm text-muted-foreground mt-1">Manage API keys for service-to-service authentication</p>
        </div>
        <button
          onClick={() => setShowForm(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
        >
          <Plus className="h-5 w-5" />
          Create Key
        </button>
      </div>

      <div className="p-8 space-y-8">
        {showForm && (
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="text-lg font-semibold text-foreground mb-4">Create API Key</h3>
            {generatedKey ? (
              <div className="space-y-4">
                <div className="p-4 rounded-lg bg-yellow-500/10 border border-yellow-500/20">
                  <p className="text-sm text-yellow-600 mb-2">
                    Make sure to copy your API key now. You won&apos;t be able to see it again!
                  </p>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 p-2 rounded bg-muted font-mono text-sm break-all">
                      {generatedKey}
                    </code>
                    <button
                      onClick={() => copyToClipboard(generatedKey)}
                      className="p-2 rounded-lg border border-border hover:bg-muted transition-colors"
                    >
                      <Copy className="h-4 w-4" />
                    </button>
                  </div>
                </div>
                <button
                  onClick={() => {
                    setShowForm(false)
                    setGeneratedKey(null)
                    setKeyName('')
                  }}
                  className="px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
                >
                  Done
                </button>
              </div>
            ) : (
              <form onSubmit={handleCreate} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Key Name</label>
                  <input
                    type="text"
                    value={keyName}
                    onChange={(e) => setKeyName(e.target.value)}
                    placeholder="e.g., Production API"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    required
                  />
                </div>
                <div className="flex gap-3">
                  <button
                    type="submit"
                    disabled={createKey.isPending}
                    className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
                  >
                    {createKey.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                    Generate Key
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
            )}
          </div>
        )}

        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : keys.length > 0 && (
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="text-lg font-semibold text-foreground mb-4">Your API Keys</h3>
            <div className="space-y-3">
              {keys.map((key: any) => (
                <div key={key.id} className="flex items-center justify-between p-4 rounded-lg border border-border">
                  <div>
                    <p className="font-medium text-foreground">{key.name}</p>
                    <p className="text-sm text-muted-foreground">
                      {key.prefix}••••••••• · Created {new Date(key.created_at).toLocaleDateString()}
                      {key.last_used && ` · Last used ${new Date(key.last_used).toLocaleDateString()}`}
                    </p>
                  </div>
                  <button
                    onClick={() => openDeleteDialog(key.id)}
                    disabled={deleteKey.isPending}
                    className="px-3 py-1.5 text-sm text-destructive hover:bg-destructive/10 rounded-lg transition-colors disabled:opacity-50"
                  >
                    Delete
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">About API Keys</h3>
          <p className="text-sm text-muted-foreground mb-4">
            API keys are used for service-to-service authentication. They provide programmatic access
            to the TrustVault API without requiring user credentials.
          </p>
          <div className="space-y-3">
            <div className="flex items-start gap-3">
              <Key className="h-5 w-5 text-primary mt-0.5" />
              <div>
                <p className="font-medium text-foreground">Secure Storage</p>
                <p className="text-sm text-muted-foreground">
                  Store API keys securely. Never commit them to version control.
                </p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <Key className="h-5 w-5 text-primary mt-0.5" />
              <div>
                <p className="font-medium text-foreground">Rotation</p>
                <p className="text-sm text-muted-foreground">
                  Rotate keys regularly and revoke unused keys.
                </p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <Key className="h-5 w-5 text-primary mt-0.5" />
              <div>
                <p className="font-medium text-foreground">Scoped Access</p>
                <p className="text-sm text-muted-foreground">
                  API keys inherit the permissions of the creating user.
                </p>
              </div>
            </div>
          </div>
        </div>

        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Usage Example</h3>
          <pre className="p-4 rounded-lg bg-muted text-sm font-mono overflow-auto">
{`curl -X GET "https://api.trustvault.io/v1/datasources" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json"`}
          </pre>
        </div>
      </div>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete API Key</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this API key? Any applications using this key will lose access immediately. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setKeyToDelete(null)}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => keyToDelete && handleDelete(keyToDelete)}
              disabled={deleteKey.isPending}
            >
              {deleteKey.isPending ? 'Deleting...' : 'Delete Key'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
