'use client'

import { useRouter } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { ArrowLeft, Shield } from 'lucide-react'
import { useCreatePolicy } from '@/hooks/use-policies'
import Link from 'next/link'

const policySchema = z.object({
  name: z.string().min(1, 'Name is required'),
  description: z.string().optional(),
  type: z.enum(['access', 'redaction', 'ai', 'retention']),
  active: z.boolean().default(true),
  priority: z.number().min(1).max(1000).default(100),
  regulations: z.string().optional(),
})

type PolicyForm = z.infer<typeof policySchema>

const policyTypes = [
  { value: 'access', label: 'Access Control', description: 'Control who can access specific data' },
  { value: 'redaction', label: 'Redaction', description: 'Automatically redact sensitive data' },
  { value: 'ai', label: 'AI Governance', description: 'Control data eligibility for AI/ML' },
  { value: 'retention', label: 'Retention', description: 'Define data retention periods' },
]

export default function NewPolicyPage() {
  const router = useRouter()
  const createPolicy = useCreatePolicy()

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    watch,
  } = useForm<PolicyForm>({
    resolver: zodResolver(policySchema),
    defaultValues: {
      type: 'access',
      active: true,
      priority: 100,
    },
  })

  const watchType = watch('type')

  const onSubmit = async (data: PolicyForm) => {
    try {
      await createPolicy.mutateAsync({
        name: data.name,
        description: data.description,
        type: data.type,
        active: data.active,
        priority: data.priority,
        regulations: data.regulations ? data.regulations.split(',').map(r => r.trim()) : undefined,
      })
      router.push('/governance/policies')
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
            { label: 'Governance', href: '/governance' },
            { label: 'Policies', href: '/governance/policies' },
            { label: 'New Policy', active: true },
          ]}
        />
        <div className="flex items-center gap-4 mt-4">
          <Link href="/governance/policies" className="p-2 rounded-lg hover:bg-muted transition-colors">
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div>
            <h1 className="text-3xl font-bold text-foreground">Create Policy</h1>
            <p className="text-sm text-muted-foreground mt-1">Define a new governance policy</p>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="p-8 max-w-2xl">
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          {/* Name */}
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Policy Name</label>
            <input
              {...register('name')}
              type="text"
              placeholder="e.g., PII Access Restriction"
              className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            />
            {errors.name && <p className="text-sm text-destructive mt-1">{errors.name.message}</p>}
          </div>

          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Description</label>
            <textarea
              {...register('description')}
              rows={3}
              placeholder="Describe what this policy does..."
              className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary resize-none"
            />
          </div>

          {/* Type Selection */}
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Policy Type</label>
            <div className="grid grid-cols-2 gap-3">
              {policyTypes.map((type) => (
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

          {/* Priority */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Priority</label>
              <input
                {...register('priority', { valueAsNumber: true })}
                type="number"
                min={1}
                max={1000}
                className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              />
              <p className="text-xs text-muted-foreground mt-1">Lower = higher priority (1-1000)</p>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Status</label>
              <label className="flex items-center gap-3 p-4 rounded-lg border border-border cursor-pointer">
                <input
                  {...register('active')}
                  type="checkbox"
                  className="w-4 h-4 rounded border-border text-primary focus:ring-primary"
                />
                <span className="text-sm text-foreground">Active</span>
              </label>
            </div>
          </div>

          {/* Regulations */}
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Regulations (optional)</label>
            <input
              {...register('regulations')}
              type="text"
              placeholder="GDPR, CCPA, HIPAA, PCI-DSS, DPDP Act 2023, UAE PDPL, EU AI Act (comma-separated)"
              className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            />
            <p className="text-xs text-muted-foreground mt-1">Link this policy to specific regulations</p>
          </div>

          {/* Submit */}
          <div className="flex items-center gap-4 pt-4">
            <button
              type="submit"
              disabled={isSubmitting || createPolicy.isPending}
              className="flex items-center gap-2 px-6 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Shield className="h-4 w-4" />
              {isSubmitting || createPolicy.isPending ? 'Creating...' : 'Create Policy'}
            </button>
            <Link
              href="/governance/policies"
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
