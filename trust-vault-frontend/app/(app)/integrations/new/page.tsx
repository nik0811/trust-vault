'use client'

import { useRouter } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { ArrowLeft, Plug } from 'lucide-react'
import { useCreateIntegration } from '@/hooks/use-jobs'
import Link from 'next/link'

const integrationSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  type: z.enum(['dlp', 'privacy_platform', 'catalog', 'siem', 'ticketing', 'communication']),
  provider: z.string().min(1, 'Provider is required'),
})

type IntegrationForm = z.infer<typeof integrationSchema>

const integrationTypes = [
  { value: 'dlp', label: 'DLP', description: 'Data Loss Prevention systems' },
  { value: 'privacy_platform', label: 'Privacy Platform', description: 'OneTrust, BigID, etc.' },
  { value: 'catalog', label: 'Data Catalog', description: 'Collibra, Alation, etc.' },
  { value: 'siem', label: 'SIEM', description: 'Security monitoring' },
  { value: 'ticketing', label: 'Ticketing', description: 'Jira, ServiceNow, etc.' },
  { value: 'communication', label: 'Communication', description: 'Slack, Teams, etc.' },
]

export default function NewIntegrationPage() {
  const router = useRouter()
  const createIntegration = useCreateIntegration()

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    watch,
  } = useForm<IntegrationForm>({
    resolver: zodResolver(integrationSchema),
    defaultValues: {
      type: 'dlp',
    },
  })

  const watchType = watch('type')

  const onSubmit = async (data: IntegrationForm) => {
    try {
      await createIntegration.mutateAsync(data)
      router.push('/integrations')
    } catch (error) {
      // Error handled by hook
    }
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Integrations', href: '/integrations' },
            { label: 'New Integration', active: true },
          ]}
        />
        <div className="flex items-center gap-4 mt-4">
          <Link href="/integrations" className="p-2 rounded-lg hover:bg-muted transition-colors">
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div>
            <h1 className="text-3xl font-bold text-foreground">Add Integration</h1>
            <p className="text-sm text-muted-foreground mt-1">Connect an external system</p>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="p-8 max-w-2xl">
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          {/* Name */}
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Name</label>
            <input
              {...register('name')}
              type="text"
              placeholder="My Integration"
              className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            />
            {errors.name && <p className="text-sm text-destructive mt-1">{errors.name.message}</p>}
          </div>

          {/* Type Selection */}
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Type</label>
            <div className="grid grid-cols-2 gap-3">
              {integrationTypes.map((type) => (
                <label
                  key={type.value}
                  className={`flex flex-col p-4 rounded-lg border cursor-pointer transition-colors ${
                    watchType === type.value
                      ? 'border-primary bg-primary/10'
                      : 'border-border hover:border-primary/50'
                  }`}
                >
                  <input
                    {...register('type')}
                    type="radio"
                    value={type.value}
                    className="sr-only"
                  />
                  <span className="text-sm font-medium text-foreground">{type.label}</span>
                  <span className="text-xs text-muted-foreground mt-1">{type.description}</span>
                </label>
              ))}
            </div>
          </div>

          {/* Provider */}
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Provider</label>
            <input
              {...register('provider')}
              type="text"
              placeholder="e.g., Slack, OneTrust, Collibra"
              className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            />
            {errors.provider && <p className="text-sm text-destructive mt-1">{errors.provider.message}</p>}
          </div>

          {/* Submit */}
          <div className="flex items-center gap-4 pt-4">
            <button
              type="submit"
              disabled={isSubmitting || createIntegration.isPending}
              className="flex items-center gap-2 px-6 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Plug className="h-4 w-4" />
              {isSubmitting || createIntegration.isPending ? 'Creating...' : 'Create Integration'}
            </button>
            <Link
              href="/integrations"
              className="px-6 py-2 rounded-lg border border-border text-foreground hover:bg-muted transition-colors"
            >
              Cancel
            </Link>
          </div>
        </form>
      </div>
    </div>
  )
}
