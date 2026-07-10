import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { toast } from 'sonner'

export interface GateQueryResult {
  decision: string
  query: string
  chunks: any[]
  redacted_chunks: any[]
  policies_applied: string[]
  latency_ms: number
}

export interface GateStats {
  total_queries: number
  allowed_queries: number
  blocked_queries: number
  avg_latency_ms: number
}

interface GateQueryRequest {
  query: string
  max_chunks?: number
  filters?: Record<string, any>
}

interface GateRetrieveRequest {
  query: string
  max_chunks?: number
}

interface GateValidateRequest {
  response: string
  context?: Record<string, any>
}

export function useGateQuery() {
  return useMutation({
    mutationFn: async (data: GateQueryRequest) => {
      const response = await api.post<GateQueryResult>('/gate/query', data)
      return response.data
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Gate query failed')
    },
  })
}

export function useGateRetrieve() {
  return useMutation({
    mutationFn: async (data: GateRetrieveRequest) => {
      const response = await api.post('/gate/retrieve', data)
      return response.data
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to retrieve data')
    },
  })
}

export function useGateValidate() {
  return useMutation({
    mutationFn: async (data: GateValidateRequest) => {
      const response = await api.post('/gate/validate', data)
      return response.data
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Validation failed')
    },
  })
}

export function useGateStats() {
  return useQuery({
    queryKey: ['gate-stats'],
    queryFn: async () => {
      const response = await api.get<GateStats>('/gate/stats')
      return response.data
    },
    refetchInterval: 30000,
  })
}

export function useGateQueries(limit = 50) {
  return useQuery({
    queryKey: ['gate-queries', limit],
    queryFn: async () => {
      const response = await api.get(`/gate/queries?limit=${limit}`)
      return response.data
    },
  })
}
