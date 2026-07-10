import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { toast } from 'sonner'

// Advisor hooks
export interface Recommendation {
  id: string
  type: string
  priority: string
  title: string
  description: string
  action: string
}

export interface ComplianceGap {
  regulation: string
  requirement: string
  status: string
  remediation: string
}

export function useRecommendations() {
  return useQuery({
    queryKey: ['recommendations'],
    queryFn: async () => {
      const response = await api.get<Recommendation[]>('/advisor/recommendations')
      return response.data
    },
  })
}

export function useComplianceGaps() {
  return useQuery({
    queryKey: ['compliance-gaps'],
    queryFn: async () => {
      const response = await api.get<ComplianceGap[]>('/advisor/gaps')
      return response.data
    },
  })
}

export function useGenerateDefenseDocket() {
  return useMutation({
    mutationFn: async (data: { regulations: string[]; date_from: string; date_to: string }) => {
      const response = await api.post('/advisor/defense-docket', data)
      return response.data
    },
    onSuccess: () => {
      toast.success('Defense docket generated')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to generate defense docket')
    },
  })
}

export function usePlaybook(issueType: string) {
  return useQuery({
    queryKey: ['playbook', issueType],
    queryFn: async () => {
      const response = await api.get(`/advisor/playbook/${issueType}`)
      return response.data
    },
    enabled: !!issueType,
  })
}

export function useRiskScore() {
  return useQuery({
    queryKey: ['risk-score'],
    queryFn: async () => {
      const response = await api.get('/advisor/risk-score')
      return response.data
    },
  })
}

// ROT Data hooks
export interface ROTSummary {
  total_rot_data: number
  redundant_count: number
  obsolete_count: number
  trivial_count: number
  potential_savings_gb: number
}

export interface ROTDataset {
  id: string
  dataset_id: string
  category: 'redundant' | 'obsolete' | 'trivial'
  score: number
  reason: string
  size_bytes: number
  last_access: string
  created_at: string
}

export function useROTSummary() {
  return useQuery({
    queryKey: ['rot-summary'],
    queryFn: async () => {
      const response = await api.get<ROTSummary>('/rot/summary')
      return response.data
    },
  })
}

export function useROTDatasets() {
  return useQuery({
    queryKey: ['rot-datasets'],
    queryFn: async () => {
      const response = await api.get<ROTDataset[]>('/rot/datasets')
      return response.data
    },
  })
}

export function useROTDuplicates() {
  return useQuery({
    queryKey: ['rot-duplicates'],
    queryFn: async () => {
      const response = await api.get('/rot/duplicates')
      return response.data
    },
  })
}

export function useTriggerROTScan() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async () => {
      const response = await api.post('/rot/scan')
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rot-summary'] })
      queryClient.invalidateQueries({ queryKey: ['rot-datasets'] })
      toast.success('ROT scan started')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to start ROT scan')
    },
  })
}

export function useRemediateROT() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { dataset_ids: string[]; action: 'archive' | 'delete' | 'deduplicate' }) => {
      const response = await api.post('/rot/remediate', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rot-summary'] })
      queryClient.invalidateQueries({ queryKey: ['rot-datasets'] })
      toast.success('ROT remediation started')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to remediate ROT data')
    },
  })
}

// Labels hooks
export interface Label {
  id: string
  dataset_id: string
  label: 'PUBLIC' | 'INTERNAL' | 'CONFIDENTIAL' | 'HIGHLY_CONFIDENTIAL' | 'RESTRICTED'
  auto_assigned: boolean
  assigned_by?: string
  created_at: string
  updated_at: string
}

export function useDatasetLabel(datasetId: string) {
  return useQuery({
    queryKey: ['labels', datasetId],
    queryFn: async () => {
      const response = await api.get<Label>(`/labels/datasets/${datasetId}`)
      return response.data
    },
    enabled: !!datasetId,
  })
}

export function useAssignLabel() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { dataset_id: string; label: string }) => {
      const response = await api.post<Label>('/labels/assign', data)
      return response.data
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['labels', variables.dataset_id] })
      toast.success('Label assigned')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to assign label')
    },
  })
}

export function useLabelRules() {
  return useQuery({
    queryKey: ['label-rules'],
    queryFn: async () => {
      const response = await api.get('/labels/rules')
      return response.data
    },
  })
}

export function useCreateLabelRule() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { classification: string; label: string }) => {
      const response = await api.post('/labels/rules', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['label-rules'] })
      toast.success('Label rule created')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create label rule')
    },
  })
}

export function useLabelSummary() {
  return useQuery({
    queryKey: ['label-summary'],
    queryFn: async () => {
      const response = await api.get('/labels/summary')
      return response.data
    },
  })
}

// Feedback hooks
export function useSubmitCorrection() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { classification_id: string; corrected_label: string }) => {
      const response = await api.post('/feedback/correction', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['feedback-stats'] })
      toast.success('Correction submitted')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to submit correction')
    },
  })
}

export function useSubmitConfirmation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { classification_id: string }) => {
      const response = await api.post('/feedback/confirmation', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['feedback-stats'] })
      toast.success('Confirmation submitted')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to submit confirmation')
    },
  })
}

export function useFeedbackStats() {
  return useQuery({
    queryKey: ['feedback-stats'],
    queryFn: async () => {
      const response = await api.get('/feedback/stats')
      return response.data
    },
  })
}

export function useCreateCustomEntity() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { name: string; pattern: string }) => {
      const response = await api.post('/feedback/custom-entity', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['custom-entities'] })
      toast.success('Custom entity created')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create custom entity')
    },
  })
}

export function useKnowledgeCache() {
  return useQuery({
    queryKey: ['knowledge-cache'],
    queryFn: async () => {
      const response = await api.get('/feedback/knowledge-cache')
      return response.data
    },
  })
}

export interface Correction {
  id: string
  text: string
  from: string
  to: string
  user: string
  created_at: string
}

export function useCorrections() {
  return useQuery({
    queryKey: ['corrections'],
    queryFn: async () => {
      const response = await api.get<Correction[]>('/feedback/corrections')
      return response.data
    },
  })
}

export function useCorrectionTrend() {
  return useQuery({
    queryKey: ['correction-trend'],
    queryFn: async () => {
      const response = await api.get<number[]>('/feedback/trend')
      return response.data
    },
  })
}

export interface CustomEntity {
  id: string
  name: string
  examples: string
  detections: number
  accuracy: number
}

export function useCustomEntities() {
  return useQuery({
    queryKey: ['custom-entities'],
    queryFn: async () => {
      const response = await api.get<CustomEntity[]>('/feedback/custom-entities')
      return response.data
    },
  })
}
