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

export function useQualityTrends() {
  return useQuery({
    queryKey: ['quality-trends'],
    queryFn: async () => {
      const response = await api.get('/quality/trends')
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
