import { useMutation, useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'

// Data Map hooks
export interface DataMapNode {
  id: string
  type: string
  name: string
  metadata: any
}

export interface DataMapEdge {
  source: string
  target: string
  type: string
}

export interface DataMap {
  nodes: DataMapNode[]
  edges: DataMapEdge[]
}

export interface DataMapCoverage {
  total_sources: number
  scanned_sources: number
  coverage_percentage: number
}

export interface DataMapGeography {
  regions: { name: string; count: number }[]
}

export function useDataMap() {
  return useQuery({
    queryKey: ['datamap'],
    queryFn: async () => {
      const response = await api.get<DataMap>('/datamap')
      return response.data
    },
  })
}

export function useDataMapSources() {
  return useQuery({
    queryKey: ['datamap-sources'],
    queryFn: async () => {
      const response = await api.get('/datamap/sources')
      return response.data
    },
  })
}

export function useDataMapFlows() {
  return useQuery({
    queryKey: ['datamap-flows'],
    queryFn: async () => {
      const response = await api.get('/datamap/flows')
      return response.data
    },
  })
}

export function useDataMapCoverage() {
  return useQuery({
    queryKey: ['datamap-coverage'],
    queryFn: async () => {
      const response = await api.get<DataMapCoverage>('/datamap/coverage')
      return response.data
    },
  })
}

export function useDataMapGeography() {
  return useQuery({
    queryKey: ['datamap-geography'],
    queryFn: async () => {
      const response = await api.get<DataMapGeography>('/datamap/geography')
      return response.data
    },
  })
}

export function useDarkData() {
  return useQuery({
    queryKey: ['dark-data'],
    queryFn: async () => {
      const response = await api.get('/datamap/dark-data')
      return response.data
    },
  })
}

export function useShadowIT() {
  return useQuery({
    queryKey: ['shadow-it'],
    queryFn: async () => {
      const response = await api.get('/datamap/shadow-it')
      return response.data
    },
  })
}

export interface AIModel {
  id: string
  name: string
  owner: string
  risk: 'low' | 'medium' | 'high'
  datasets: number
  approved: boolean
  last_review: string
}

export function useAIModels() {
  return useQuery({
    queryKey: ['ai-models'],
    queryFn: async () => {
      const response = await api.get<AIModel[]>('/ai-governance/models')
      return response.data
    },
  })
}

// Lineage hooks
export function useDatasetLineage(datasetId: string) {
  return useQuery({
    queryKey: ['lineage', datasetId],
    queryFn: async () => {
      const response = await api.get(`/audit/lineage/${datasetId}`)
      return response.data
    },
    enabled: !!datasetId,
  })
}

// AI Governance hooks
export function useAIGovPolicies() {
  return useQuery({
    queryKey: ['ai-gov-policies'],
    queryFn: async () => {
      const response = await api.get('/ai-governance/policies')
      return response.data
    },
  })
}

export interface EligibilityCheck {
  name: string
  passed: boolean
  note: string
}

export interface EligibilityResult {
  eligible: boolean
  checks: EligibilityCheck[]
}

export function useAIEligibility(datasetId: string, useCase?: string) {
  return useQuery({
    queryKey: ['ai-eligibility', datasetId, useCase],
    queryFn: async () => {
      const params = useCase ? `?use_case=${useCase}` : ''
      const response = await api.get<EligibilityResult>(`/ai-governance/eligible/${datasetId}${params}`)
      return response.data
    },
    enabled: !!datasetId,
  })
}

export function useCheckEligibility() {
  return useMutation({
    mutationFn: async (data: { dataset: string; use_case: string }) => {
      const response = await api.post<EligibilityResult>('/ai-governance/check-eligibility', data)
      return response.data
    },
  })
}

export function useEligibleDatasets() {
  return useQuery({
    queryKey: ['eligible-datasets'],
    queryFn: async () => {
      const response = await api.get<string[]>('/ai-governance/datasets')
      return response.data
    },
  })
}

export function useModelLineage(modelId: string) {
  return useQuery({
    queryKey: ['model-lineage', modelId],
    queryFn: async () => {
      const response = await api.get(`/ai-governance/lineage/${modelId}`)
      return response.data
    },
    enabled: !!modelId,
  })
}

// Reports hooks
export interface Report {
  id: string
  type: string
  status: string
  generated_at: string
}

export function useReports() {
  return useQuery({
    queryKey: ['reports'],
    queryFn: async () => {
      const response = await api.get<Report[]>('/reports')
      return response.data
    },
  })
}

export function useAnalyticsSummary() {
  return useQuery({
    queryKey: ['analytics-summary'],
    queryFn: async () => {
      const response = await api.get('/analytics/summary')
      return response.data
    },
  })
}

export function useAnalyticsTrends() {
  return useQuery({
    queryKey: ['analytics-trends'],
    queryFn: async () => {
      const response = await api.get('/analytics/trends')
      return response.data
    },
  })
}

// Documents hooks
export function useExtractDocument() {
  return useQuery({
    queryKey: ['documents'],
    queryFn: async () => {
      const response = await api.get('/documents/review-queue')
      return response.data
    },
  })
}
