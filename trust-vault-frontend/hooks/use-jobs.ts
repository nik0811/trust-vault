import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { toast } from 'sonner'

// Jobs hooks
export interface Job {
  id: string
  name: string
  type: string
  schedule: string
  config: any
  status: string
  last_run: string
  next_run: string
  created_at: string
  updated_at: string
}

interface CreateJobRequest {
  name: string
  type: string
  schedule: string
  config?: Record<string, any>
}

export function useJobs() {
  return useQuery({
    queryKey: ['jobs'],
    queryFn: async () => {
      const response = await api.get<Job[]>('/jobs')
      return response.data
    },
  })
}

export function useJob(id: string) {
  return useQuery({
    queryKey: ['jobs', id],
    queryFn: async () => {
      const response = await api.get<Job>(`/jobs/${id}`)
      return response.data
    },
    enabled: !!id,
  })
}

export function useCreateJob() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: CreateJobRequest) => {
      const response = await api.post<Job>('/jobs', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] })
      toast.success('Job created successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create job')
    },
  })
}

export function useDeleteJob() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/jobs/${id}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] })
      toast.success('Job deleted successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to delete job')
    },
  })
}

export function useRunJobNow() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      const response = await api.post(`/jobs/${id}/run-now`)
      return response.data
    },
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ['jobs', id] })
      toast.success('Job started')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to run job')
    },
  })
}

// Integrations hooks
export interface Integration {
  id: string
  name: string
  type: string  // slack, teams, email, webhook, jira, servicenow, pagerduty, dlp, siem, splunk, sentinel, catalog, collibra, alation, onetrust, privacyops, rest_api, custom, privacy_platform, ticketing, communication
  provider: string
  config: any
  sync_freq: string
  status: string
  last_sync: string | null
  created_at: string
  updated_at: string
}

interface CreateIntegrationRequest {
  name: string
  type: string
  provider: string
  config?: Record<string, any>
  sync_freq?: string
}

export function useIntegrations() {
  return useQuery({
    queryKey: ['integrations'],
    queryFn: async () => {
      const response = await api.get<Integration[]>('/integrations')
      return response.data
    },
  })
}

export function useIntegration(id: string) {
  return useQuery({
    queryKey: ['integrations', id],
    queryFn: async () => {
      const response = await api.get<Integration>(`/integrations/${id}`)
      return response.data
    },
    enabled: !!id,
  })
}

export function useCreateIntegration() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: CreateIntegrationRequest) => {
      const response = await api.post<Integration>('/integrations', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['integrations'] })
      toast.success('Integration created successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create integration')
    },
  })
}

export function useUpdateIntegration() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ id, data }: { id: string; data: Partial<CreateIntegrationRequest> }) => {
      const response = await api.put<Integration>(`/integrations/${id}`, data)
      return response.data
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['integrations'] })
      queryClient.invalidateQueries({ queryKey: ['integrations', variables.id] })
      toast.success('Integration updated successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to update integration')
    },
  })
}

export function useDeleteIntegration() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/integrations/${id}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['integrations'] })
      toast.success('Integration deleted successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to delete integration')
    },
  })
}

export function useTestIntegration() {
  return useMutation({
    mutationFn: async (id: string) => {
      const response = await api.post(`/integrations/${id}/test`)
      return response.data
    },
    onSuccess: () => {
      toast.success('Integration test successful')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Integration test failed')
    },
  })
}

export function useSyncIntegration() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      const response = await api.post(`/integrations/${id}/sync`)
      return response.data
    },
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ['integrations', id] })
      toast.success('Sync started')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to sync integration')
    },
  })
}

export function useIntegrationLogs(id: string) {
  return useQuery({
    queryKey: ['integrations', id, 'logs'],
    queryFn: async () => {
      const response = await api.get(`/integrations/${id}/logs`)
      return response.data
    },
    enabled: !!id,
  })
}

// Notifications hooks
export interface Notification {
  id: string
  type: string
  severity: string
  title: string
  message: string
  resource: string
  read: boolean
  created_at: string
}

export function useNotifications() {
  return useQuery({
    queryKey: ['notifications'],
    queryFn: async () => {
      const response = await api.get<Notification[]>('/notifications')
      return response.data
    },
  })
}

export function useMarkNotificationRead() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      const response = await api.put(`/notifications/${id}/read`)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
    },
  })
}

export function useMarkAllNotificationsRead() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async () => {
      const response = await api.put('/notifications/read-all')
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
    },
  })
}

export function useDeleteNotification() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/notifications/${id}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
    },
  })
}

// Webhooks hooks
export interface Webhook {
  id: string
  url: string
  events: string[]
  active: boolean
  created_at: string
  updated_at: string
}

export function useWebhooks() {
  return useQuery({
    queryKey: ['webhooks'],
    queryFn: async () => {
      const response = await api.get<Webhook[]>('/notifications/webhooks')
      return response.data
    },
  })
}

export function useCreateWebhook() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { url: string; events: string[] }) => {
      const response = await api.post<Webhook>('/notifications/webhooks', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['webhooks'] })
      toast.success('Webhook created successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create webhook')
    },
  })
}

export function useDeleteWebhook() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/notifications/webhooks/${id}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['webhooks'] })
      toast.success('Webhook deleted successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to delete webhook')
    },
  })
}
