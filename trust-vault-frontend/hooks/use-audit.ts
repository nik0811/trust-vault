import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { toast } from 'sonner'

export interface AuditLog {
  id: string
  user_id: string
  action: string
  resource: string
  resource_id: string
  details: any
  ip_address: string
  created_at: string
}

export function useAuditTrail(params?: { limit?: number; offset?: number }) {
  return useQuery({
    queryKey: ['audit-trail', params],
    queryFn: async () => {
      const response = await api.get<AuditLog[]>('/audit/trail', { params })
      return response.data
    },
  })
}

export function useAIUsage(datasetId: string) {
  return useQuery({
    queryKey: ['ai-usage', datasetId],
    queryFn: async () => {
      const response = await api.get(`/audit/datasets/${datasetId}/ai-usage`)
      return response.data
    },
    enabled: !!datasetId,
  })
}

export function useComplianceReport() {
  return useQuery({
    queryKey: ['compliance-report'],
    queryFn: async () => {
      const response = await api.get('/audit/compliance-report')
      return response.data
    },
  })
}

export function useLineage(datasetId: string) {
  return useQuery({
    queryKey: ['lineage', datasetId],
    queryFn: async () => {
      const response = await api.get(`/audit/lineage/${datasetId}`)
      return response.data
    },
    enabled: !!datasetId,
  })
}

// Observability hooks
export interface SystemHealth {
  status: string
  components: Record<string, { status: string; latency_ms?: number }>
  uptime_seconds: number
}

export interface SystemMetrics {
  cpu_usage: number
  memory_usage: number
  disk_usage: number
  active_connections: number
  requests_per_minute: number
}

export function useSystemHealth() {
  return useQuery({
    queryKey: ['system-health'],
    queryFn: async () => {
      const response = await api.get<SystemHealth>('/observability/health')
      return response.data
    },
    refetchInterval: 30000,
  })
}

export function useSystemMetrics() {
  return useQuery({
    queryKey: ['system-metrics'],
    queryFn: async () => {
      const response = await api.get<SystemMetrics>('/observability/metrics')
      return response.data
    },
    refetchInterval: 10000,
  })
}

export function useAlerts() {
  return useQuery({
    queryKey: ['alerts'],
    queryFn: async () => {
      const response = await api.get('/observability/alerts')
      return response.data
    },
  })
}

export function useCreateAlertRule() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { name: string; condition: string; severity: string }) => {
      const response = await api.post('/observability/alerts/rules', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
      toast.success('Alert rule created')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create alert rule')
    },
  })
}
