'use client'

import { useRouter } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { ArrowLeft, Plug, Eye, EyeOff } from 'lucide-react'
import { useCreateIntegration } from '@/hooks/use-jobs'
import Link from 'next/link'
import { useState } from 'react'

const integrationSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  type: z.string().min(1, 'Type is required'),
  provider: z.string().optional(),
  sync_freq: z.string().optional(),
  config: z.record(z.string(), z.any()).optional(),
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

interface ConfigField {
  name: string
  label: string
  type: 'text' | 'password' | 'number' | 'select' | 'textarea'
  placeholder?: string
  required?: boolean
  options?: { value: string; label: string }[]
}

const configFieldsByType: Record<string, ConfigField[]> = {
  slack: [
    { name: 'webhook_url', label: 'Webhook URL', type: 'text', placeholder: 'https://hooks.slack.com/services/...', required: true },
    { name: 'channel', label: 'Channel (optional)', type: 'text', placeholder: '#security-alerts' },
  ],
  teams: [
    { name: 'webhook_url', label: 'Webhook URL', type: 'text', placeholder: 'https://outlook.office.com/webhook/...', required: true },
  ],
  email: [
    { name: 'smtp_host', label: 'SMTP Host', type: 'text', placeholder: 'smtp.example.com', required: true },
    { name: 'smtp_port', label: 'SMTP Port', type: 'number', placeholder: '587', required: true },
    { name: 'smtp_user', label: 'SMTP Username', type: 'text', placeholder: 'user@example.com' },
    { name: 'smtp_password', label: 'SMTP Password', type: 'password', placeholder: '••••••••' },
    { name: 'from_address', label: 'From Address', type: 'text', placeholder: 'noreply@example.com', required: true },
    { name: 'to_addresses', label: 'To Addresses (comma-separated)', type: 'text', placeholder: 'admin@example.com, security@example.com', required: true },
  ],
  webhook: [
    { name: 'url', label: 'Webhook URL', type: 'text', placeholder: 'https://api.example.com/webhook', required: true },
    { name: 'method', label: 'HTTP Method', type: 'select', options: [{ value: 'POST', label: 'POST' }, { value: 'PUT', label: 'PUT' }], required: true },
    { name: 'auth_type', label: 'Authentication', type: 'select', options: [{ value: 'none', label: 'None' }, { value: 'bearer', label: 'Bearer Token' }, { value: 'basic', label: 'Basic Auth' }] },
    { name: 'token', label: 'Token / Password', type: 'password', placeholder: 'Bearer token or password' },
    { name: 'headers', label: 'Custom Headers (JSON)', type: 'textarea', placeholder: '{"X-Custom-Header": "value"}' },
  ],
  jira: [
    { name: 'url', label: 'Jira URL', type: 'text', placeholder: 'https://yourcompany.atlassian.net', required: true },
    { name: 'email', label: 'Email', type: 'text', placeholder: 'user@example.com', required: true },
    { name: 'api_token', label: 'API Token', type: 'password', placeholder: 'Your Jira API token', required: true },
    { name: 'project_key', label: 'Project Key', type: 'text', placeholder: 'SEC', required: true },
    { name: 'issue_type', label: 'Issue Type', type: 'text', placeholder: 'Task' },
  ],
  servicenow: [
    { name: 'instance', label: 'Instance Name', type: 'text', placeholder: 'yourcompany (without .service-now.com)', required: true },
    { name: 'username', label: 'Username', type: 'text', placeholder: 'admin', required: true },
    { name: 'password', label: 'Password', type: 'password', placeholder: '••••••••', required: true },
    { name: 'table_name', label: 'Table Name', type: 'text', placeholder: 'incident' },
  ],
  pagerduty: [
    { name: 'routing_key', label: 'Routing Key', type: 'password', placeholder: 'Events API v2 routing key', required: true },
  ],
  splunk: [
    { name: 'url', label: 'HEC URL', type: 'text', placeholder: 'https://splunk.example.com:8088/services/collector', required: true },
    { name: 'token', label: 'HEC Token', type: 'password', placeholder: 'Your Splunk HEC token', required: true },
    { name: 'index', label: 'Index (optional)', type: 'text', placeholder: 'main' },
  ],
  sentinel: [
    { name: 'workspace_id', label: 'Workspace ID', type: 'text', placeholder: 'Log Analytics Workspace ID', required: true },
    { name: 'shared_key', label: 'Shared Key', type: 'password', placeholder: 'Primary or Secondary key', required: true },
    { name: 'log_type', label: 'Log Type', type: 'text', placeholder: 'SecureLens' },
  ],
  datadog: [
    { name: 'api_key', label: 'API Key', type: 'password', placeholder: 'Your Datadog API key', required: true },
    { name: 'site', label: 'Site', type: 'select', options: [{ value: 'datadoghq.com', label: 'US1 (datadoghq.com)' }, { value: 'datadoghq.eu', label: 'EU (datadoghq.eu)' }, { value: 'us3.datadoghq.com', label: 'US3' }, { value: 'us5.datadoghq.com', label: 'US5' }] },
  ],
  collibra: [
    { name: 'url', label: 'Collibra URL', type: 'text', placeholder: 'https://yourcompany.collibra.com', required: true },
    { name: 'username', label: 'Username', type: 'text', placeholder: 'api-user', required: true },
    { name: 'password', label: 'Password', type: 'password', placeholder: '••••••••', required: true },
  ],
  alation: [
    { name: 'url', label: 'Alation URL', type: 'text', placeholder: 'https://yourcompany.alationcloud.com', required: true },
    { name: 'api_token', label: 'API Token', type: 'password', placeholder: 'Your Alation API token', required: true },
  ],
  onetrust: [
    { name: 'url', label: 'OneTrust URL', type: 'text', placeholder: 'https://yourcompany.onetrust.com', required: true },
    { name: 'client_id', label: 'Client ID', type: 'text', placeholder: 'OAuth Client ID', required: true },
    { name: 'client_secret', label: 'Client Secret', type: 'password', placeholder: 'OAuth Client Secret', required: true },
  ],
  rest_api: [
    { name: 'url', label: 'API URL', type: 'text', placeholder: 'https://api.example.com/endpoint', required: true },
    { name: 'method', label: 'HTTP Method', type: 'select', options: [{ value: 'GET', label: 'GET' }, { value: 'POST', label: 'POST' }, { value: 'PUT', label: 'PUT' }] },
    { name: 'auth_type', label: 'Authentication', type: 'select', options: [{ value: 'none', label: 'None' }, { value: 'bearer', label: 'Bearer Token' }, { value: 'api_key', label: 'API Key' }, { value: 'basic', label: 'Basic Auth' }] },
    { name: 'token', label: 'Token / API Key', type: 'password', placeholder: 'Authentication credential' },
    { name: 'headers', label: 'Custom Headers (JSON)', type: 'textarea', placeholder: '{"X-Custom-Header": "value"}' },
  ],
  custom: [
    { name: 'url', label: 'Endpoint URL', type: 'text', placeholder: 'https://...' },
    { name: 'api_key', label: 'API Key', type: 'password', placeholder: 'Optional API key' },
    { name: 'custom_config', label: 'Custom Configuration (JSON)', type: 'textarea', placeholder: '{"key": "value"}' },
  ],
}

export default function NewIntegrationPage() {
  const router = useRouter()
  const createIntegration = useCreateIntegration()
  const [showPasswords, setShowPasswords] = useState<Record<string, boolean>>({})

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    watch,
    setValue,
  } = useForm<IntegrationForm>({
    resolver: zodResolver(integrationSchema),
    defaultValues: {
      type: 'slack',
      config: {},
    },
  })

  const watchType = watch('type')
  const watchConfig = watch('config') || {}

  const configFields = configFieldsByType[watchType] || []

  const togglePasswordVisibility = (fieldName: string) => {
    setShowPasswords(prev => ({ ...prev, [fieldName]: !prev[fieldName] }))
  }

  const handleConfigChange = (fieldName: string, value: string | number) => {
    setValue('config', { ...watchConfig, [fieldName]: value })
  }

  const onSubmit = async (data: IntegrationForm) => {
    try {
      const selectedType = integrationTypes.find(t => t.value === data.type)
      const payload = {
        ...data,
        provider: data.provider || selectedType?.label || data.type,
      }
      await createIntegration.mutateAsync(payload)
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

          {/* Configuration Fields */}
          {configFields.length > 0 && (
            <div className="space-y-4 p-4 rounded-lg border border-border bg-muted/30">
              <h3 className="text-sm font-semibold text-foreground">Configuration</h3>
              {configFields.map((field) => (
                <div key={field.name}>
                  <label className="block text-sm font-medium text-foreground mb-1">
                    {field.label}
                    {field.required && <span className="text-destructive ml-1">*</span>}
                  </label>
                  {field.type === 'select' ? (
                    <select
                      value={(watchConfig[field.name] as string) || ''}
                      onChange={(e) => handleConfigChange(field.name, e.target.value)}
                      className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    >
                      <option value="">Select...</option>
                      {field.options?.map((opt) => (
                        <option key={opt.value} value={opt.value}>{opt.label}</option>
                      ))}
                    </select>
                  ) : field.type === 'textarea' ? (
                    <textarea
                      value={(watchConfig[field.name] as string) || ''}
                      onChange={(e) => handleConfigChange(field.name, e.target.value)}
                      placeholder={field.placeholder}
                      rows={3}
                      className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary font-mono text-sm"
                    />
                  ) : field.type === 'password' ? (
                    <div className="relative">
                      <input
                        type={showPasswords[field.name] ? 'text' : 'password'}
                        value={(watchConfig[field.name] as string) || ''}
                        onChange={(e) => handleConfigChange(field.name, e.target.value)}
                        placeholder={field.placeholder}
                        className="w-full px-4 py-2 pr-10 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                      />
                      <button
                        type="button"
                        onClick={() => togglePasswordVisibility(field.name)}
                        className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                      >
                        {showPasswords[field.name] ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                      </button>
                    </div>
                  ) : (
                    <input
                      type={field.type}
                      value={(watchConfig[field.name] as string | number) || ''}
                      onChange={(e) => handleConfigChange(field.name, field.type === 'number' ? Number(e.target.value) : e.target.value)}
                      placeholder={field.placeholder}
                      className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    />
                  )}
                </div>
              ))}
            </div>
          )}

          {/* Provider (optional override) */}
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Provider Name (optional)</label>
            <input
              {...register('provider')}
              type="text"
              placeholder="Auto-detected from type"
              className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            />
            <p className="text-xs text-muted-foreground mt-1">Leave blank to use the integration type name</p>
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
