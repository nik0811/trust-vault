'use client'

import { useState, useEffect } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { DataTable, type Column } from '@/components/base/data-table'
import { Plus, Loader2, Braces } from 'lucide-react'
import { toast } from 'sonner'
import { useCustomEntities, useCreateCustomEntity, type CustomEntity } from '@/hooks/use-advisor'

const columns: Column<CustomEntity>[] = [
  { id: 'name', header: 'Entity Type', cell: (row) => <span className="font-mono text-sm font-semibold">{row.name}</span> },
  { id: 'examples', header: 'Examples', cell: (row) => <span className="font-mono text-xs text-muted-foreground">{row.examples}</span> },
  { id: 'detections', header: 'Detections', cell: (row) => row.detections.toLocaleString() },
  {
    id: 'accuracy',
    header: 'Accuracy',
    cell: (row) => (
      <div className="flex items-center gap-2">
        <div className="h-1.5 w-16 overflow-hidden rounded-full bg-muted">
          <div
            className={row.accuracy >= 95 ? 'h-full bg-green-500' : 'h-full bg-yellow-500'}
            style={{ width: `${row.accuracy}%` }}
          />
        </div>
        <span className="text-sm">{row.accuracy}%</span>
      </div>
    ),
  },
]

export default function CustomEntitiesPage() {
  const { data: entitiesRaw, isLoading, error } = useCustomEntities()
  const createEntity = useCreateCustomEntity()
  const [name, setName] = useState('')
  const [examples, setExamples] = useState('')

  useEffect(() => {
    if (error) toast.error('Failed to load custom entities')
  }, [error])

  const entities: CustomEntity[] = entitiesRaw || []

  const addEntity = async () => {
    if (!name.trim() || !examples.trim()) {
      toast.error('Provide an entity name and at least one example')
      return
    }
    try {
      await createEntity.mutateAsync({ 
        name: name.toUpperCase().replace(/\s+/g, '_'), 
        pattern: examples.split('\n').join('|') 
      })
      setName('')
      setExamples('')
    } catch {
      // Error handled by hook
    }
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Feedback', href: '/feedback' },
            { label: 'Custom Entities', active: true },
          ]}
        />
        <h1 className="mt-4 text-3xl font-bold text-foreground">Custom Entities</h1>
        <p className="mt-1 text-sm text-muted-foreground">Tenant-specific entity types the model learns to detect</p>
      </div>

      <div className="grid grid-cols-1 gap-8 p-8 lg:grid-cols-3">
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="mb-4 text-lg font-semibold text-foreground">Define New Entity</h3>
          <div className="space-y-4">
            <div>
              <label htmlFor="entity-name" className="mb-1.5 block text-sm font-medium text-foreground">
                Entity name
              </label>
              <input
                id="entity-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. CONTRACT_NUMBER"
                className="w-full rounded-lg border border-border bg-background px-3 py-2 font-mono text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <div>
              <label htmlFor="entity-examples" className="mb-1.5 block text-sm font-medium text-foreground">
                Examples (one per line)
              </label>
              <textarea
                id="entity-examples"
                value={examples}
                onChange={(e) => setExamples(e.target.value)}
                rows={5}
                placeholder={'CTR-2026-0041\nCTR-2025-1998'}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 font-mono text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <button
              onClick={addEntity}
              disabled={createEntity.isPending}
              className="flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
            >
              {createEntity.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Plus className="h-4 w-4" />
              )}
              Add Entity
            </button>
          </div>
        </div>

        <div className="rounded-lg border border-border bg-card p-6 lg:col-span-2">
          <h3 className="mb-4 text-lg font-semibold text-foreground">Registered Entities</h3>
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : entities.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              <Braces className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>No custom entities defined yet</p>
            </div>
          ) : (
            <DataTable columns={columns} data={entities} />
          )}
        </div>
      </div>
    </div>
  )
}
