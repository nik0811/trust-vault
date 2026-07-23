'use client'

import { useParams, useRouter } from 'next/navigation'
import { useState, useEffect } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { StatusIndicator } from '@/components/base/status-badge'
import { ArrowLeft, Trash2, Play, TestTube, Save, CheckCircle, XCircle, Eye, EyeOff, Edit2 } from 'lucide-react'
import { useIntegration, useDeleteIntegration, useSyncIntegration, useTestIntegration, useUpdateIntegration } from '@/hooks/use-jobs'
import Link from 'next/link'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'

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
    { name: 'auth_type', label: 'Authentication', type: 'select', options: [{ value: 'none', label: 'None' }, { value: 'bearer', label: 'Bearer Token' }, { value: 'basic', label: 'Basic Auth' }] },
    { name: 'token', label: 'Token / Password', type: 'password', placeholder: 'Bearer token or password' },
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
    { name: 'default_model', label: 'Default Model', type: 'select', options: [{ value: 'gpt-4o', label: 'GPT-4o' }, { value: 'gpt-4o-mini', label: 'GPT-4o Mini' }, { value: 'gpt-4-turbo', label: 'GPT-4 Turbo' }] },
  ],
  anthropic: [
    { name: 'api_key', label: 'API Key', type: 'password', placeholder: 'sk-ant-...', required: true },
    { name: 'default_model', label: 'Default Model', type: 'select', options: [{ value: 'claude-sonnet-4-20250514', label: 'Claude Sonnet 4' }, { value: 'claude-3-5-sonnet-20241022', label: 'Claude 3.5 Sonnet' }] },
  ],
  azure_openai: [
    { name: 'endpoint', label: 'Azure Endpoint', type: 'text', placeholder: 'https://your-resource.openai.azure.com', required: true },
    { name: 'api_key', label: 'API Key', type: 'password', placeholder: 'Azure OpenAI API key', required: true },
    { name: 'deployment_name', label: 'Deployment Name', type: 'text', placeholder: 'gpt-4', required: true },
  ],
  aws_bedrock: [
    { name: 'region', label: 'AWS Region', type: 'text', placeholder: 'us-east-1', required: true },
    { name: 'access_key_id', label: 'Access Key ID', type: 'text', placeholder: 'AKIA...', required: true },
    { name: 'secret_access_key', label: 'Secret Access Key', type: 'password', placeholder: 'Your AWS secret key', required: true },
  ],
  ollama: [
    { name: 'url', label: 'Ollama URL', type: 'text', placeholder: 'http://localhost:11434', required: true },
    { name: 'default_model', label: 'Default Model', type: 'text', placeholder: 'llama3.1', required: true },
  ],
}

export default function IntegrationDetailPage() {
  const params = useParams()
  const router = useRouter()
  const id = params.id as string
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [testResult, setTestResult] = useState<{ success: boolean; message: string; latency_ms?: number } | null>(null)
  const [isEditing, setIsEditing] = useState(false)
  const [editConfig, setEditConfig] = useState<Record<string, any>>({})
  const [showPasswords, setShowPasswords] = useState<Record<string, boolean>>({})

  const { data: integration, isLoading, refetch } = useIntegration(id)
  const deleteIntegration = useDeleteIntegration()
  const syncIntegration = useSyncIntegration()
  const testIntegration = useTestIntegration()
  const updateIntegration = useUpdateIntegration()

  useEffect(() => {
    if (integration?.config) {
      setEditConfig(typeof integration.config === 'object' ? integration.config : {})
    }
  }, [integration])

  const handleDelete = async () => {
    try {
      await deleteIntegration.mutateAsync(id)
      setDeleteDialogOpen(false)
      router.push('/integrations')
    } catch (error) {
      // Error handled by hook
    }
  }

  const handleTest = async () => {
    setTestResult(null)
    try {
      const result = await testIntegration.mutateAsync(id)
      setTestResult({
        success: result?.success ?? true,
        message: result?.message || 'Connection successful',
        latency_ms: result?.latency_ms,
      })
      refetch()
    } catch (error: any) {
      setTestResult({
        success: false,
        message: error?.response?.data?.error || error?.message || 'Connection test failed',
      })
    }
  }

  const handleSaveConfig = async () => {
    try {
      await updateIntegration.mutateAsync({ id, data: { config: editConfig } })
      setIsEditing(false)
      refetch()
    } catch (error) {
      // Error handled by hook
    }
  }

  const togglePasswordVisibility = (fieldName: string) => {
    setShowPasswords(prev => ({ ...prev, [fieldName]: !prev[fieldName] }))
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background p-8">
        <Skeleton className="h-8 w-48 mb-4" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (!integration) {
    return (
      <div className="min-h-screen bg-background p-8">
        <div className="text-center py-12">
          <p className="text-destructive">Integration not found</p>
          <Link href="/integrations" className="mt-4 text-primary hover:underline">
            Back to Integrations
          </Link>
        </div>
      </div>
    )
  }

  const configFields = configFieldsByType[integration.type] || []
  const configData = typeof integration.config === 'object' && integration.config !== null ? integration.config : {}

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Integrations', href: '/integrations' },
            { label: integration.name, active: true },
          ]}
        />
        <div className="flex items-center justify-between mt-4">
          <div className="flex items-center gap-4">
            <Link href="/integrations" className="p-2 rounded-lg hover:bg-muted transition-colors">
              <ArrowLeft className="h-5 w-5" />
            </Link>
            <div>
              <h1 className="text-3xl font-bold text-foreground">{integration.name}</h1>
              <div className="flex items-center gap-3 mt-1">
                <span className="text-sm text-muted-foreground capitalize">
                  {integration.type.replace('_', ' ')}
                </span>
                <StatusIndicator
                  status={integration.status === 'connected' ? 'success' : integration.status === 'syncing' ? 'pending' : 'error'}
                  label={integration.status}
                />
              </div>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={handleTest}
              disabled={testIntegration.isPending}
              className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border bg-card text-foreground hover:bg-muted transition-colors disabled:opacity-50"
            >
              <TestTube className="h-4 w-4" />
              {testIntegration.isPending ? 'Testing...' : 'Test Connection'}
            </button>
            <button
              onClick={() => syncIntegration.mutate(id)}
              disabled={syncIntegration.isPending}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Play className="h-4 w-4" />
              Sync Now
            </button>
            <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
              <AlertDialogTrigger asChild>
                <button
                  disabled={deleteIntegration.isPending}
                  className="flex items-center gap-2 px-4 py-2 rounded-lg border border-destructive text-destructive hover:bg-destructive/10 transition-colors disabled:opacity-50"
                >
                  <Trash2 className="h-4 w-4" />
                  Delete
                </button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Delete Integration</AlertDialogTitle>
                  <AlertDialogDescription>
                    Are you sure you want to delete &quot;{integration.name}&quot;? This will stop all syncing and remove the integration configuration. This action cannot be undone.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    variant="destructive"
                    onClick={handleDelete}
                    disabled={deleteIntegration.isPending}
                  >
                    {deleteIntegration.isPending ? 'Deleting...' : 'Delete'}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </div>
      </div>

      {/* Test Result Banner */}
      {testResult && (
        <div className={`mx-8 mt-4 p-4 rounded-lg flex items-center gap-3 ${testResult.success ? 'bg-green-500/10 border border-green-500/30' : 'bg-destructive/10 border border-destructive/30'}`}>
          {testResult.success ? (
            <CheckCircle className="h-5 w-5 text-green-500" />
          ) : (
            <XCircle className="h-5 w-5 text-destructive" />
          )}
          <div className="flex-1">
            <p className={`text-sm font-medium ${testResult.success ? 'text-green-500' : 'text-destructive'}`}>
              {testResult.success ? 'Connection Successful' : 'Connection Failed'}
            </p>
            <p className="text-sm text-muted-foreground">{testResult.message}</p>
          </div>
          {testResult.latency_ms && (
            <span className="text-xs text-muted-foreground">{testResult.latency_ms}ms</span>
          )}
          <button onClick={() => setTestResult(null)} className="text-muted-foreground hover:text-foreground">
            <XCircle className="h-4 w-4" />
          </button>
        </div>
      )}

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Details */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Details</h3>
          <div className="grid grid-cols-2 gap-6">
            <div>
              <p className="text-sm text-muted-foreground">Provider</p>
              <p className="text-sm font-medium text-foreground capitalize">
                {integration.provider || integration.type?.replace('_', ' ') || 'Unknown'}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Sync Frequency</p>
              <p className="text-sm font-medium text-foreground">{integration.sync_freq || 'Manual'}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Last Sync</p>
              <p className="text-sm font-medium text-foreground">
                {integration.last_sync && integration.last_sync !== '0001-01-01T00:00:00Z'
                  ? new Date(integration.last_sync).toLocaleString()
                  : 'Never'}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Created</p>
              <p className="text-sm font-medium text-foreground">
                {new Date(integration.created_at).toLocaleString()}
              </p>
            </div>
          </div>
        </div>

        {/* Configuration */}
        <div className="rounded-lg border border-border bg-card p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-foreground">Configuration</h3>
            {!isEditing ? (
              <button
                onClick={() => setIsEditing(true)}
                className="flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg border border-border hover:bg-muted transition-colors"
              >
                <Edit2 className="h-4 w-4" />
                Edit
              </button>
            ) : (
              <div className="flex items-center gap-2">
                <button
                  onClick={() => {
                    setIsEditing(false)
                    setEditConfig(configData as Record<string, any>)
                  }}
                  className="px-3 py-1.5 text-sm rounded-lg border border-border hover:bg-muted transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={handleSaveConfig}
                  disabled={updateIntegration.isPending}
                  className="flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                  <Save className="h-4 w-4" />
                  {updateIntegration.isPending ? 'Saving...' : 'Save'}
                </button>
              </div>
            )}
          </div>

          {configFields.length > 0 ? (
            <div className="space-y-4">
              {configFields.map((field) => (
                <div key={field.name}>
                  <label className="block text-sm font-medium text-foreground mb-1">
                    {field.label}
                    {field.required && <span className="text-destructive ml-1">*</span>}
                  </label>
                  {isEditing ? (
                    field.type === 'select' ? (
                      <select
                        value={editConfig[field.name] || ''}
                        onChange={(e) => setEditConfig({ ...editConfig, [field.name]: e.target.value })}
                        className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                      >
                        <option value="">Select...</option>
                        {field.options?.map((opt) => (
                          <option key={opt.value} value={opt.value}>{opt.label}</option>
                        ))}
                      </select>
                    ) : field.type === 'password' ? (
                      <div className="relative">
                        <input
                          type={showPasswords[field.name] ? 'text' : 'password'}
                          value={editConfig[field.name] || ''}
                          onChange={(e) => setEditConfig({ ...editConfig, [field.name]: e.target.value })}
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
                        value={editConfig[field.name] || ''}
                        onChange={(e) => setEditConfig({ ...editConfig, [field.name]: field.type === 'number' ? Number(e.target.value) : e.target.value })}
                        placeholder={field.placeholder}
                        className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                      />
                    )
                  ) : (
                    <p className="text-sm text-muted-foreground bg-muted px-4 py-2 rounded-lg">
                      {field.type === 'password' && configData[field.name]
                        ? '••••••••'
                        : configData[field.name] || <span className="italic">Not configured</span>}
                    </p>
                  )}
                </div>
              ))}
            </div>
          ) : Object.keys(configData).length > 0 ? (
            <pre className="text-sm bg-muted p-4 rounded-lg overflow-auto">
              {JSON.stringify(configData, null, 2)}
            </pre>
          ) : (
            <p className="text-sm text-muted-foreground italic">No configuration set</p>
          )}
        </div>
      </div>
    </div>
  )
}
