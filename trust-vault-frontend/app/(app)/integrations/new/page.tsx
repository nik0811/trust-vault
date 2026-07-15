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
  type: z.string().min(1, 'Type is required'),
  provider: z.string().min(1, 'Provider is required'),
  sync_freq: z.string().optional(),
  config: z.any().optional(),
})

type IntegrationForm = z.infer<typeof integrationSchema>

const integrationTypes = [
  // Communication
  { value: 'slack', label: 'Slack', description: 'Slack workspace notifications', category: 'Communication' },
  { value: 'teams', label: 'Microsoft Teams', description: 'Teams channel notifications', category: 'Communication' },
  { value: 'email', label: 'Email (SMTP)', description: 'Email notifications via SMTP', category: 'Communication' },
  { value: 'webhook', label: 'Webhook', description: 'Generic HTTP webhook endpoint', category: 'Communication' },
  // Ticketing
  { value: 'jira', label: 'Jira', description: 'Atlassian Jira issue tracking', category: 'Ticketing' },
  { value: 'servicenow', label: 'ServiceNow', description: 'ServiceNow ITSM platform', category: 'Ticketing' },
  { value: 'pagerduty', label: 'PagerDuty', description: 'Incident management', category: 'Ticketing' },
  // Security / DLP
  { value: 'dlp', label: 'DLP', description: 'Generic Data Loss Prevention', category: 'Security' },
  { value: 'siem', label: 'SIEM', description: 'Generic security monitoring', category: 'Security' },
  { value: 'splunk', label: 'Splunk', description: 'Splunk HEC endpoint', category: 'Security' },
  { value: 'sentinel', label: 'Azure Sentinel', description: 'Microsoft Azure Sentinel SIEM', category: 'Security' },
  // Catalog
  { value: 'catalog', label: 'Data Catalog', description: 'Generic data catalog', category: 'Catalog' },
  { value: 'collibra', label: 'Collibra', description: 'Collibra data governance', category: 'Catalog' },
  { value: 'alation', label: 'Alation', description: 'Alation data catalog', category: 'Catalog' },
  // Privacy
  { value: 'onetrust', label: 'OneTrust', description: 'OneTrust privacy platform', category: 'Privacy' },
  { value: 'privacyops', label: 'PrivacyOps', description: 'Privacy operations platform', category: 'Privacy' },
  // Custom
  { value: 'rest_api', label: 'REST API', description: 'Custom REST API endpoint', category: 'Custom' },
  { value: 'custom', label: 'Custom', description: 'Custom integration', category: 'Custom' },
]

const typeCategories = Array.from(new Set(integrationTypes.map(t => t.category)))

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
      type: 'slack',
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
            {typeCategories.map((category) => (
              <div key={category} className="mb-4">
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-2">{category}</p>
                <div className="grid grid-cols-2 gap-2">
                  {integrationTypes.filter(t => t.category === category).map((type) => (
                    <label
                      key={type.value}
                      className={`flex flex-col p-3 rounded-lg border cursor-pointer transition-colors ${
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
                      <span className="text-xs text-muted-foreground mt-0.5">{type.description}</span>
                    </label>
                  ))}
                </div>
              </div>
            ))}
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
