import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { toast } from 'sonner'

export interface DSAR {
  id: string
  subject_id: string
  type: 'access' | 'delete' | 'rectify'
  status: string
  deadline: string
  results: any
  assigned_to?: string
  completed_at?: string
  created_at: string
  updated_at: string
}

interface CreateDSARRequest {
  subject_id: string
  type: 'access' | 'delete' | 'rectify'
}

export function useDSARs() {
  return useQuery({
    queryKey: ['dsars'],
    queryFn: async () => {
      const response = await api.get<DSAR[]>('/privacy/dsar')
      return response.data
    },
  })
}

export function useDSAR(id: string) {
  return useQuery({
    queryKey: ['dsars', id],
    queryFn: async () => {
      const response = await api.get<DSAR>(`/privacy/dsar/${id}`)
      return response.data
    },
    enabled: !!id,
  })
}

export function useCreateDSAR() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: CreateDSARRequest) => {
      const response = await api.post<DSAR>('/privacy/dsar', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dsars'] })
      toast.success('DSAR created successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create DSAR')
    },
  })
}

export function useDSARPackage(id: string) {
  return useQuery({
    queryKey: ['dsars', id, 'package'],
    queryFn: async () => {
      const response = await api.get(`/privacy/dsar/${id}/package`)
      return response.data
    },
    enabled: !!id,
  })
}

export function useGeneratePIA() {
  return useMutation({
    mutationFn: async (datasetId: string) => {
      const response = await api.post('/privacy/pia', { dataset_id: datasetId })
      return response.data
    },
    onSuccess: () => {
      toast.success('PIA generated successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to generate PIA')
    },
  })
}

export function usePIA(datasetId: string) {
  return useQuery({
    queryKey: ['pia', datasetId],
    queryFn: async () => {
      const response = await api.get(`/privacy/pia/${datasetId}`)
      return response.data
    },
    enabled: !!datasetId,
  })
}

export function useRoPA() {
  return useQuery({
    queryKey: ['ropa'],
    queryFn: async () => {
      const response = await api.get('/privacy/ropa')
      return response.data
    },
  })
}

export function useCreateRoPA() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { activity: string }) => {
      const response = await api.post('/privacy/ropa', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ropa'] })
      toast.success('RoPA entry created')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create RoPA entry')
    },
  })
}

export function useRecordConsent() {
  return useMutation({
    mutationFn: async (data: { subject_id: string; purpose: string }) => {
      const response = await api.post('/privacy/consent', data)
      return response.data
    },
    onSuccess: () => {
      toast.success('Consent recorded')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to record consent')
    },
  })
}

export function useWithdrawConsent() {
  return useMutation({
    mutationFn: async (subjectId: string) => {
      const response = await api.delete(`/privacy/consent/${subjectId}`)
      return response.data
    },
    onSuccess: () => {
      toast.success('Consent withdrawn')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to withdraw consent')
    },
  })
}

export function useRetentionViolations() {
  return useQuery({
    queryKey: ['retention-violations'],
    queryFn: async () => {
      const response = await api.get('/privacy/retention/violations')
      return response.data
    },
  })
}

export function useSetRetentionPolicy() {
  return useMutation({
    mutationFn: async (data: { classification: string; retention_days: number }) => {
      const response = await api.post('/privacy/retention/policies', data)
      return response.data
    },
    onSuccess: () => {
      toast.success('Retention policy set')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to set retention policy')
    },
  })
}
