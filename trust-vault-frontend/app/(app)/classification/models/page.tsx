'use client'

import { DataTable, type Column } from '@/components/base/data-table'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { StatusIndicator } from '@/components/base/status-badge'
import Link from 'next/link'
import { useClassificationModels, type ClassificationModel } from '@/hooks/use-classification'

const columns: Column<ClassificationModel>[] = [
  {
    id: 'name',
    header: 'Model Name',
    cell: (row) => <span className="font-medium text-foreground">{row.name}</span>,
    sortable: true,
  },
  {
    id: 'type',
    header: 'Type',
    cell: (row) => (
      <span className="px-2 py-0.5 rounded bg-muted text-foreground text-sm capitalize">{row.type}</span>
    ),
  },
  {
    id: 'version',
    header: 'Version',
    accessorKey: 'version',
  },
  {
    id: 'status',
    header: 'Status',
    cell: (row) => (
      <StatusIndicator
        status={row.status === 'active' ? 'success' : row.status === 'loading' ? 'pending' : 'error'}
        label={row.status}
      />
    ),
  },
]

export default function ClassificationModelsPage() {
  const { data: models, isLoading } = useClassificationModels()

  const modelsData = Array.isArray(models) ? models : []

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Classification', href: '/classification' },
            { label: 'Models', active: true },
          ]}
        />
        <h1 className="text-3xl font-bold text-foreground mt-4">Classification Models</h1>
        <p className="text-sm text-muted-foreground mt-1">ML models used for automatic data classification</p>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Navigation Tabs */}
        <div className="flex gap-4 border-b border-border pb-4">
          <Link href="/classification" className="text-muted-foreground hover:text-foreground transition-colors pb-2">
            Datasets
          </Link>
          <Link href="/classification/rules" className="text-muted-foreground hover:text-foreground transition-colors pb-2">
            Rules
          </Link>
          <Link href="/classification/models" className="text-foreground font-medium border-b-2 border-primary pb-2">
            Models
          </Link>
        </div>

        {/* Model Info */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Active Models</h3>
          <p className="text-sm text-muted-foreground mb-6">
            TrustVault uses proprietary AI models for zero-shot NER classification, capable of detecting 60+ PII types
            at 4M+ characters per second on CPU.
          </p>

          {isLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : modelsData.length > 0 ? (
            <DataTable columns={columns} data={modelsData} />
          ) : (
            <div className="text-center py-8">
              <p className="text-muted-foreground">No models loaded</p>
              <p className="text-sm text-muted-foreground mt-1">
                Models are automatically loaded when the classification service starts.
              </p>
            </div>
          )}
        </div>

        {/* Model Capabilities */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Supported Entity Types</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            {[
              'EMAIL', 'PHONE', 'SSN', 'CREDIT_CARD', 'IBAN', 'IP_ADDRESS',
              'DATE_OF_BIRTH', 'ADDRESS', 'NAME', 'PASSPORT', 'DRIVER_LICENSE',
              'MEDICAL_RECORD', 'BANK_ACCOUNT', 'TAX_ID', 'NATIONAL_ID', 'BIOMETRIC',
            ].map((type) => (
              <div key={type} className="px-3 py-2 rounded-lg bg-muted text-sm text-foreground">
                {type}
              </div>
            ))}
          </div>
          <p className="text-sm text-muted-foreground mt-4">
            + 44 more entity types supported. Custom entities can be added via classification rules.
          </p>
        </div>
      </div>
    </div>
  )
}
