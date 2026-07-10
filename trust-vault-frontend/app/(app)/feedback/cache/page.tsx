'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { StatCard } from '@/components/base/stat-card'
import { DataTable, type Column } from '@/components/base/data-table'
import { Database, Trash2 } from 'lucide-react'
import { toast } from 'sonner'

interface CacheEntry {
  id: string
  pattern: string
  entity: string
  hits: number
  lastHit: string
}

const initialEntries: CacheEntry[] = [
  { id: '1', pattern: 'INV-####-###', entity: 'INVOICE_ID', hits: 84200, lastHit: '2 min ago' },
  { id: '2', pattern: 'EMP-#####', entity: 'EMPLOYEE_ID', hits: 31500, lastHit: '14 min ago' },
  { id: '3', pattern: '###-##-#### (SSN format)', entity: 'US_SSN', hits: 12800, lastHit: '1h ago' },
  { id: '4', pattern: 'name@domain email', entity: 'EMAIL_ADDRESS', hits: 240100, lastHit: 'just now' },
  { id: '5', pattern: 'PRJ-*-#', entity: 'PROJECT_CODE', hits: 4100, lastHit: '3h ago' },
]

export default function KnowledgeCachePage() {
  const [entries, setEntries] = useState(initialEntries)

  const clearEntry = (id: string, pattern: string) => {
    setEntries((prev) => prev.filter((e) => e.id !== id))
    toast.success(`Cache entry cleared: ${pattern}`)
  }

  const columns: Column<CacheEntry>[] = [
    { id: 'pattern', header: 'Pattern', cell: (row) => <span className="font-mono text-sm">{row.pattern}</span> },
    { id: 'entity', header: 'Entity Type', cell: (row) => <span className="font-mono text-xs font-semibold">{row.entity}</span> },
    { id: 'hits', header: 'Cache Hits', cell: (row) => row.hits.toLocaleString() },
    { id: 'lastHit', header: 'Last Hit', accessorKey: 'lastHit' },
    {
      id: 'actions',
      header: '',
      cell: (row) => (
        <button
          onClick={() => clearEntry(row.id, row.pattern)}
          aria-label={`Clear cache entry ${row.pattern}`}
          className="flex items-center gap-1.5 rounded border border-border px-2 py-1 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        >
          <Trash2 className="h-3 w-3" />
          Clear
        </button>
      ),
    },
  ]

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Feedback', href: '/feedback' },
            { label: 'Knowledge Cache', active: true },
          ]}
        />
        <h1 className="mt-4 text-3xl font-bold text-foreground">Knowledge Cache</h1>
        <p className="mt-1 text-sm text-muted-foreground">Cached classification lookups for instant results</p>
      </div>

      <div className="space-y-8 p-8">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
          <StatCard label="Cache Entries" value="14.2K" icon={<Database className="h-6 w-6" />} />
          <StatCard label="Hit Rate" value="68%" change={12} changeLabel="vs last month" />
          <StatCard label="Avg Lookup Time" value="0.4ms" />
          <StatCard label="Compute Saved" value="~$860/mo" />
        </div>

        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="mb-4 text-lg font-semibold text-foreground">Top Cache Entries</h3>
          <DataTable columns={columns} data={entries} />
        </div>
      </div>
    </div>
  )
}
