import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { toast } from 'sonner'

export interface Policy {
  id: string
  name: string
  description: string
  type: 'access' | 'redaction' | 'ai' | 'retention'
  conditions: Record<string, any> | null
  actions: Record<string, any> | null
  regulations: string[] | null
  active: boolean
  priority: number
  created_at: string
  updated_at: string
}

interface CreatePolicyRequest {
  name: string
  description?: string
  type: 'access' | 'redaction' | 'ai' | 'retention'
  conditions?: Record<string, any>
  actions?: Record<string, any>
  regulations?: string[]
  active?: boolean
  priority?: number
}

interface EvaluateRequest {
  data: string
  context?: Record<string, any>
}

interface EvaluateResponse {
  decision: string
  applied_policies: string[]
  redacted_data?: string
}

export function usePolicies() {
  return useQuery({
    queryKey: ['policies'],
    queryFn: async () => {
      const response = await api.get<Policy[]>('/governance/policies')
      return response.data
    },
  })
}

export function usePolicy(id: string) {
  return useQuery({
    queryKey: ['policies', id],
    queryFn: async () => {
      const response = await api.get<Policy>(`/governance/policies/${id}`)
      return response.data
    },
    enabled: !!id,
  })
}

export function useCreatePolicy() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: CreatePolicyRequest) => {
      const response = await api.post<Policy>('/governance/policies', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] })
      toast.success('Policy created successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create policy')
    },
  })
}

export function useUpdatePolicy() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ id, data }: { id: string; data: Partial<CreatePolicyRequest> }) => {
      const response = await api.put<Policy>(`/governance/policies/${id}`, data)
      return response.data
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['policies'] })
      queryClient.invalidateQueries({ queryKey: ['policies', variables.id] })
      toast.success('Policy updated successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to update policy')
    },
  })
}

export function useDeletePolicy() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/governance/policies/${id}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] })
      toast.success('Policy deleted successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to delete policy')
    },
  })
}

export function useEvaluatePolicy() {
  return useMutation({
    mutationFn: async (data: EvaluateRequest) => {
      const response = await api.post<EvaluateResponse>('/governance/evaluate', data)
      return response.data
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to evaluate policy')
    },
  })
}

export interface GovernanceStats {
  total_policies: number
  active_policies: number
  evaluations_24h: number
  evaluation_status: string
}

export function useGovernanceStats() {
  return useQuery({
    queryKey: ['governance-stats'],
    queryFn: async () => {
      const response = await api.get<GovernanceStats>('/governance/stats')
      return response.data
    },
  })
}
