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
  // Notifications - Alert channels for policy violations, classification events, compliance issues
  { value: 'slack', label: 'Slack', description: 'Send alerts to Slack channels', category: 'Notifications' },
  { value: 'teams', label: 'Microsoft Teams', description: 'Send alerts to Teams channels', category: 'Notifications' },
  { value: 'email', label: 'Email (SMTP)', description: 'Email notifications for alerts', category: 'Notifications' },
  { value: 'webhook', label: 'Webhook', description: 'Custom HTTP webhook endpoint', category: 'Notifications' },
  // Vector Databases - For AI Gate context retrieval
  { value: 'pinecone', label: 'Pinecone', description: 'Vector DB for RAG context', category: 'Vector Databases' },
  { value: 'qdrant', label: 'Qdrant', description: 'Vector search for AI Gate', category: 'Vector Databases' },
  { value: 'weaviate', label: 'Weaviate', description: 'Vector DB with hybrid search', category: 'Vector Databases' },
  { value: 'chroma', label: 'Chroma', description: 'Open-source embedding database', category: 'Vector Databases' },
  // LLM Providers - For AI Gate LLM proxy
  { value: 'openai', label: 'OpenAI', description: 'GPT models via OpenAI API', category: 'LLM Providers' },
  { value: 'anthropic', label: 'Anthropic', description: 'Claude models via Anthropic API', category: 'LLM Providers' },
  { value: 'azure_openai', label: 'Azure OpenAI', description: 'OpenAI models on Azure', category: 'LLM Providers' },
  { value: 'aws_bedrock', label: 'AWS Bedrock', description: 'Foundation models on AWS', category: 'LLM Providers' },
  { value: 'ollama', label: 'Ollama', description: 'Local LLM inference server', category: 'LLM Providers' },
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
  // Notifications
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
    { name: 'auth_type', label: 'Authentication', type: 'select', options: [{ value: 'none', label: 'None' }, { value: 'bearer', label: 'Bearer Token' }, { value: 'basic', label: 'Basic Auth' }, { value: 'api_key', label: 'API Key Header' }] },
    { name: 'token', label: 'Token / Password', type: 'password', placeholder: 'Bearer token or password' },
    { name: 'headers', label: 'Custom Headers (JSON)', type: 'textarea', placeholder: '{"X-Custom-Header": "value"}' },
  ],
  // Vector Databases
  pinecone: [
    { name: 'api_key', label: 'API Key', type: 'password', placeholder: 'Pinecone API key', required: true },
    { name: 'environment', label: 'Environment', type: 'text', placeholder: 'us-east-1-aws', required: true },
    { name: 'index_name', label: 'Index Name', type: 'text', placeholder: 'securelens-vectors', required: true },
  ],
  qdrant: [
    { name: 'url', label: 'Qdrant URL', type: 'text', placeholder: 'https://qdrant.example.com:6333', required: true },
    { name: 'api_key', label: 'API Key', type: 'password', placeholder: 'Qdrant API key' },
    { name: 'collection', label: 'Collection Name', type: 'text', placeholder: 'securelens', required: true },
  ],
  weaviate: [
    { name: 'url', label: 'Weaviate URL', type: 'text', placeholder: 'https://weaviate.example.com', required: true },
    { name: 'api_key', label: 'API Key', type: 'password', placeholder: 'Weaviate API key' },
    { name: 'class_name', label: 'Class Name', type: 'text', placeholder: 'SecureLensDocument', required: true },
  ],
  chroma: [
    { name: 'url', label: 'Chroma URL', type: 'text', placeholder: 'http://localhost:8000', required: true },
    { name: 'collection', label: 'Collection Name', type: 'text', placeholder: 'securelens', required: true },
    { name: 'auth_token', label: 'Auth Token (optional)', type: 'password', placeholder: 'Bearer token' },
  ],
  // LLM Providers
  openai: [
    { name: 'api_key', label: 'API Key', type: 'password', placeholder: 'sk-...', required: true },
    { name: 'organization', label: 'Organization ID (optional)', type: 'text', placeholder: 'org-...' },
    { name: 'default_model', label: 'Default Model', type: 'select', options: [{ value: 'gpt-4o', label: 'GPT-4o' }, { value: 'gpt-4o-mini', label: 'GPT-4o Mini' }, { value: 'gpt-4-turbo', label: 'GPT-4 Turbo' }, { value: 'gpt-3.5-turbo', label: 'GPT-3.5 Turbo' }] },
  ],
  anthropic: [
    { name: 'api_key', label: 'API Key', type: 'password', placeholder: 'sk-ant-...', required: true },
    { name: 'default_model', label: 'Default Model', type: 'select', options: [{ value: 'claude-sonnet-4-20250514', label: 'Claude Sonnet 4' }, { value: 'claude-3-5-sonnet-20241022', label: 'Claude 3.5 Sonnet' }, { value: 'claude-3-opus-20240229', label: 'Claude 3 Opus' }, { value: 'claude-3-haiku-20240307', label: 'Claude 3 Haiku' }] },
  ],
  azure_openai: [
    { name: 'endpoint', label: 'Azure Endpoint', type: 'text', placeholder: 'https://your-resource.openai.azure.com', required: true },
    { name: 'api_key', label: 'API Key', type: 'password', placeholder: 'Azure OpenAI API key', required: true },
    { name: 'deployment_name', label: 'Deployment Name', type: 'text', placeholder: 'gpt-4', required: true },
    { name: 'api_version', label: 'API Version', type: 'text', placeholder: '2024-02-15-preview' },
  ],
  aws_bedrock: [
    { name: 'region', label: 'AWS Region', type: 'text', placeholder: 'us-east-1', required: true },
    { name: 'access_key_id', label: 'Access Key ID', type: 'text', placeholder: 'AKIA...', required: true },
    { name: 'secret_access_key', label: 'Secret Access Key', type: 'password', placeholder: 'Your AWS secret key', required: true },
    { name: 'default_model', label: 'Default Model', type: 'select', options: [{ value: 'anthropic.claude-3-sonnet-20240229-v1:0', label: 'Claude 3 Sonnet' }, { value: 'anthropic.claude-3-haiku-20240307-v1:0', label: 'Claude 3 Haiku' }, { value: 'amazon.titan-text-express-v1', label: 'Titan Text Express' }] },
  ],
  ollama: [
    { name: 'url', label: 'Ollama URL', type: 'text', placeholder: 'http://localhost:11434', required: true },
    { name: 'default_model', label: 'Default Model', type: 'text', placeholder: 'llama3.1', required: true },
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
    <div className="h-full bg-background overflow-auto">
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
