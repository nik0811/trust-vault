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
    enabled: false, // Only fetch when explicitly triggered
  })
}

export function useUpdateDSAR() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ id, ...data }: { id: string; status?: string }) => {
      const response = await api.put<DSAR>(`/privacy/dsar/${id}`, data)
      return response.data
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['dsars'] })
      queryClient.invalidateQueries({ queryKey: ['dsars', variables.id] })
      toast.success('DSAR updated successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to update DSAR')
    },
  })
}

export function useDeleteDSAR() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      const response = await api.delete(`/privacy/dsar/${id}`)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dsars'] })
      toast.success('DSAR deleted successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to delete DSAR')
    },
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

// ── DPIA hooks ──────────────────────────────────────────────────────────────

export interface DPIAStep {
  id: string
  name: string
  status: 'pending' | 'in_progress' | 'completed' | 'skipped'
  notes: string
  completed_at: string | null
}

export interface DPIA {
  id: string
  name: string
  description: string
  data_types: string[]
  processing_purpose: string
  risk_level: 'high' | 'medium' | 'low'
  status: 'in_progress' | 'completed' | 'pending_dpo'
  steps: DPIAStep[]
  dpo_consulted: boolean
  created_at: string
  updated_at: string
}

export function useDPIAs() {
  return useQuery({
    queryKey: ['dpias'],
    queryFn: async () => {
      const response = await api.get<DPIA[]>('/privacy/dpia')
      return response.data ?? []
    },
  })
}

export function useDPIA(id: string) {
  return useQuery({
    queryKey: ['dpias', id],
    queryFn: async () => {
      const response = await api.get<DPIA>(`/privacy/dpia/${id}`)
      return response.data
    },
    enabled: !!id,
  })
}

export function useCreateDPIA() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (data: Partial<DPIA>) => {
      const response = await api.post<DPIA>('/privacy/dpia', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dpias'] })
      toast.success('DPIA created')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create DPIA')
    },
  })
}

export function useUpdateDPIAStep() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, step, data }: { id: string; step: string; data: { status: string; notes: string } }) => {
      const response = await api.put(`/privacy/dpia/${id}/step/${step}`, data)
      return response.data
    },
    onSuccess: (_data, vars) => {
      queryClient.invalidateQueries({ queryKey: ['dpias', vars.id] })
      queryClient.invalidateQueries({ queryKey: ['dpias'] })
      toast.success('Step updated')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to update step')
    },
  })
}

// ── Enhanced Consent hooks ────────────────────────────────────────────────────

export interface ConsentRecord {
  id: string
  subject_id: string
  purpose: string
  status: 'granted' | 'withdrawn'
  ip: string
  source: string
  created_at: string
}

export interface ConsentStats {
  total: number
  granted: number
  withdrawn: number
  withdrawal_rate: number
  by_purpose: Record<string, { total: number; granted: number }>
}

export function useConsentRecords(purpose?: string, status?: string) {
  return useQuery({
    queryKey: ['consent-records', purpose, status],
    queryFn: async () => {
      const params = new URLSearchParams()
      if (purpose) params.set('purpose', purpose)
      if (status) params.set('status', status)
      const response = await api.get<ConsentRecord[]>(`/privacy/consent/records?${params}`)
      return response.data ?? []
    },
  })
}

export function useConsentStats() {
  return useQuery({
    queryKey: ['consent-stats'],
    queryFn: async () => {
      const response = await api.get<ConsentStats>('/privacy/consent/stats')
      return response.data
    },
  })
}

export function useRecordConsentV2() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (data: { subject_id: string; purpose: string; status?: string; ip?: string; source?: string }) => {
      const response = await api.post<ConsentRecord>('/privacy/consent/record', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['consent-records'] })
      queryClient.invalidateQueries({ queryKey: ['consent-stats'] })
      toast.success('Consent recorded')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to record consent')
    },
  })
}

export function useWithdrawConsentV2() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (subjectId: string) => {
      const response = await api.post(`/privacy/consent/withdraw/${subjectId}`, {})
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['consent-records'] })
      queryClient.invalidateQueries({ queryKey: ['consent-stats'] })
      toast.success('Consent withdrawn')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to withdraw consent')
    },
  })
}
