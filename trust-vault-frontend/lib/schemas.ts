import { z } from 'zod'

// Auth schemas
export const loginSchema = z.object({
  email: z.string().email('Invalid email'),
  password: z.string().min(6, 'Password must be at least 6 characters'),
  rememberMe: z.boolean().optional(),
})

export const forgotPasswordSchema = z.object({
  email: z.string().email('Invalid email'),
})

export const resetPasswordSchema = z.object({
  password: z.string().min(8, 'Password must be at least 8 characters'),
  confirmPassword: z.string(),
}).refine((data) => data.password === data.confirmPassword, {
  message: "Passwords don't match",
  path: ['confirmPassword'],
})

// Data source schemas
export const dataSourceSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  type: z.enum(['database', 'datalake', 'api', 'file', 'streaming']),
  connectionString: z.string().optional(),
  host: z.string().optional(),
  port: z.number().optional(),
  username: z.string().optional(),
  password: z.string().optional(),
  database: z.string().optional(),
})

// Policy schemas
export const policySchema = z.object({
  name: z.string().min(1, 'Policy name is required'),
  description: z.string().optional(),
  type: z.enum(['access_control', 'redaction', 'masking', 'anonymization']),
  regulations: z.array(z.string()).optional(),
  conditions: z.array(z.object({
    field: z.string(),
    operator: z.string(),
    value: z.any(),
  })).optional(),
  actions: z.array(z.object({
    type: z.string(),
    config: z.record(z.any()).optional(),
  })).optional(),
  enabled: z.boolean().default(true),
})

// Classification rule schemas
export const classificationRuleSchema = z.object({
  name: z.string().min(1, 'Rule name is required'),
  pattern: z.string().optional(),
  keywords: z.array(z.string()).optional(),
  sensitivity: z.enum(['public', 'internal', 'confidential', 'restricted']),
  enabled: z.boolean().default(true),
})

// DSAR schemas
export const dsarSchema = z.object({
  dataSubjectId: z.string().min(1, 'Data subject ID is required'),
  requestType: z.enum(['access', 'rectification', 'deletion', 'portability']),
  description: z.string().optional(),
  dueDate: z.date(),
})

// Settings schemas
export const tenantSettingsSchema = z.object({
  name: z.string().min(1, 'Tenant name is required'),
  logo: z.string().optional(),
  governanceMode: z.enum(['strict', 'balanced', 'permissive']),
  timezone: z.string(),
})

export const userInviteSchema = z.object({
  email: z.string().email('Invalid email'),
  role: z.enum(['admin', 'editor', 'viewer']),
})

export const webhookSchema = z.object({
  name: z.string().min(1, 'Webhook name is required'),
  url: z.string().url('Invalid webhook URL'),
  events: z.array(z.string()).min(1, 'At least one event is required'),
  active: z.boolean().default(true),
})

// Type exports
export type LoginInput = z.infer<typeof loginSchema>
export type ForgotPasswordInput = z.infer<typeof forgotPasswordSchema>
export type ResetPasswordInput = z.infer<typeof resetPasswordSchema>
export type DataSourceInput = z.infer<typeof dataSourceSchema>
export type PolicyInput = z.infer<typeof policySchema>
export type ClassificationRuleInput = z.infer<typeof classificationRuleSchema>
export type DSARInput = z.infer<typeof dsarSchema>
export type TenantSettingsInput = z.infer<typeof tenantSettingsSchema>
export type UserInviteInput = z.infer<typeof userInviteSchema>
export type WebhookInput = z.infer<typeof webhookSchema>
