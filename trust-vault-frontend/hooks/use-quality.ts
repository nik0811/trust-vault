import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { toast } from 'sonner'

export interface QualityScore {
  id: string
  dataset_id: string
  overall: number
  completeness: number
  accuracy: number
  consistency: number
  timeliness: number
  uniqueness: number
  issues: any[]
  created_at: string
}

export function useQualityScore(datasetId: string) {
  return useQuery({
    queryKey: ['quality', datasetId],
    queryFn: async () => {
      const response = await api.get<QualityScore>(`/quality/datasets/${datasetId}`)
      return response.data
    },
    enabled: !!datasetId,
  })
}

export interface QualityTrends {
  overall: number
  completeness: number
  accuracy: number
  consistency: number
  timeliness: number
  uniqueness: number
  total_datasets: number
  issues_found: number
}

export function useQualityTrends() {
  return useQuery({
    queryKey: ['quality-trends'],
    queryFn: async () => {
      // Use /quality/summary which computes metrics from classification data
      // even when no explicit quality assessments have been run
      const response = await api.get<{
        overall_score: number
        completeness: number
        accuracy: number
        consistency: number
        timeliness: number
        uniqueness: number
        total_datasets: number
        issues_found: number
      }>('/quality/summary')
      
      // Transform to expected format (overall_score -> overall)
      const data = response.data
      return {
        overall: data.overall_score,
        completeness: data.completeness,
        accuracy: data.accuracy,
        consistency: data.consistency,
        timeliness: data.timeliness,
        uniqueness: data.uniqueness,
        total_datasets: data.total_datasets,
        issues_found: data.issues_found,
      } as QualityTrends
    },
  })
}

export interface QualityDimensions {
  overall_score: number
  completeness: number
  accuracy: number
  consistency: number
  timeliness: number
  uniqueness: number
  total_datasets: number
  issues_found: number
}

export function useQualityDimensions() {
  return useQuery({
    queryKey: ['quality-dimensions'],
    queryFn: async () => {
      const response = await api.get<QualityDimensions>('/quality/summary')
      return response.data
    },
  })
}

export function useAssessQuality() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (datasetId: string) => {
      const response = await api.post('/quality/assess', { dataset_id: datasetId })
      return response.data
    },
    onSuccess: (_, datasetId) => {
      queryClient.invalidateQueries({ queryKey: ['quality', datasetId] })
      queryClient.invalidateQueries({ queryKey: ['quality-trends'] })
      toast.success('Quality assessment started')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to assess quality')
    },
  })
}

export function useSetQualityThreshold() {
  return useMutation({
    mutationFn: async (data: { dimension: string; minimum: number; severity: string }) => {
      const response = await api.post('/quality/thresholds', data)
      return response.data
    },
    onSuccess: () => {
      toast.success('Threshold updated')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to set threshold')
    },
  })
}

// ── Data Profiling ───────────────────────────────────────────────────────────

export interface ColumnProfile {
  column_name: string
  inferred_type: string
  entity_type: string
  confidence: number
  null_rate: number
  distinct_count: number
  sample_values: string[]
  is_pii: boolean
}

export interface DataProfile {
  datasource_id: string
  datasource_name: string
  datasource_type: string
  columns: ColumnProfile[]
  total_columns: number
  pii_columns: number
  profiled_at: string
  status?: string
}

export function useDataProfile(datasourceId: string) {
  return useQuery({
    queryKey: ['data-profile', datasourceId],
    queryFn: async () => {
      const response = await api.get<DataProfile>(`/quality/profile/${datasourceId}`)
      return response.data
    },
    enabled: !!datasourceId,
  })
}

export function useAutoProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (datasourceId: string) => {
      const response = await api.post<DataProfile>(`/quality/profile/${datasourceId}`, {})
      return response.data
    },
    onSuccess: (_, datasourceId) => {
      queryClient.invalidateQueries({ queryKey: ['data-profile', datasourceId] })
      toast.success('Data profiling completed')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to profile datasource')
    },
  })
}

// ── CDE (Critical Data Elements) ─────────────────────────────────────────────

export interface CDE {
  id: string
  datasource_id: string
  datasource_name?: string
  datasource_type?: string
  column_name: string
  table_name: string
  business_definition: string
  data_owner: string
  criticality: 'high' | 'medium' | 'low'
  quality_score: number
  created_at: string
}

export function useCDEs() {
  return useQuery({
    queryKey: ['cdes'],
    queryFn: async () => {
      const response = await api.get<CDE[]>('/cde')
      return response.data ?? []
    },
  })
}

export function useCreateCDE() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (data: Partial<CDE>) => {
      const response = await api.post<CDE>('/cde', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cdes'] })
      toast.success('CDE designated')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create CDE')
    },
  })
}

export function useDeleteCDE() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const response = await api.delete(`/cde/${id}`)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cdes'] })
      toast.success('CDE removed')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to delete CDE')
    },
  })
}
