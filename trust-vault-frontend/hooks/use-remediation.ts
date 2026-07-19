import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { toast } from 'sonner'

export interface RemediationAction {
  id: string
  type: string
  action_type: string
  dataset_id: string
  dataset_name: string
  reason: string
  status: 'pending' | 'running' | 'completed' | 'failed' | string
  approved_by?: string
  executed_at?: string
  created_at: string
  updated_at: string
}

export interface RemediationLog {
  id: string
  user_id: string
  action: string
  resource: string
  resource_id: string
  details: any
  ip: string
  created_at: string
}

export function useRemediationActions() {
  return useQuery({
    queryKey: ['remediation-actions'],
    queryFn: async () => {
      const response = await api.get<RemediationAction[]>('/remediation/actions')
      return Array.isArray(response.data) ? response.data : []
    },
  })
}

export function useRemediationLogs(actionId: string) {
  return useQuery({
    queryKey: ['remediation-logs', actionId],
    queryFn: async () => {
      const response = await api.get<RemediationLog[]>(`/remediation/actions/${actionId}/logs`)
      return Array.isArray(response.data) ? response.data : []
    },
    enabled: !!actionId,
  })
}

export function useExecuteRemediationAction() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      const response = await api.post<RemediationAction>(`/remediation/actions/${id}/execute`)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['remediation-actions'] })
      toast.success('Remediation action executed')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to execute remediation action')
    },
  })
}

export function useCreateRemediationAction() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { type: string; action_type?: string; dataset_id: string; reason?: string }) => {
      const response = await api.post<RemediationAction>('/remediation/actions', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['remediation-actions'] })
      toast.success('Remediation action created')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create remediation action')
    },
  })
}
