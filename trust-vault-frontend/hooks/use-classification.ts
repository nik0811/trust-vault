import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { toast } from 'sonner'

export interface ClassificationEntity {
  type: string
  value: string
  start: number
  end: number
  confidence: number
}

export interface ClassificationResult {
  text: string
  entities: ClassificationEntity[]
  sensitivity_label: string
}

export interface ClassificationModel {
  id: string
  name: string
  version: string
  type: string
  status: string
}

interface ClassifyTextRequest {
  text: string
}

interface ClassifyDatasetRequest {
  dataset_id: string
  async?: boolean
}

export interface ClassifyStats {
  total_classified: number
  avg_confidence: number
  high_risk: number
  last_run: string | null
  per_dataset: Array<{ dataset_id: string; classified_columns: number; avg_confidence: number }>
}

export function useClassifyStats() {
  return useQuery({
    queryKey: ['classify-stats'],
    queryFn: async () => {
      const response = await api.get<ClassifyStats>('/classify/stats')
      return response.data
    },
  })
}

export function useClassifyText() {
  return useMutation({
    mutationFn: async (data: ClassifyTextRequest) => {
      const response = await api.post<ClassificationResult>('/classify/text', data)
      return response.data
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to classify text')
    },
  })
}

export function useClassifyDataset() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: ClassifyDatasetRequest) => {
      const response = await api.post('/classify/dataset', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['classifications'] })
      toast.success('Classification job started')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to start classification')
    },
  })
}

export function useClassificationModels() {
  return useQuery({
    queryKey: ['classification-models'],
    queryFn: async () => {
      const response = await api.get<ClassificationModel[]>('/classify/models')
      return response.data
    },
  })
}

export function useClassificationRules() {
  return useQuery({
    queryKey: ['classification-rules'],
    queryFn: async () => {
      const response = await api.get('/classify/rules')
      return response.data
    },
  })
}

export function useCreateClassificationRule() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { name: string; pattern: string; entity_type: string; priority?: number }) => {
      const response = await api.post('/classify/rules', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['classification-rules'] })
      toast.success('Classification rule created')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create rule')
    },
  })
}

export function useDeleteClassificationRule() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/classify/rules/${id}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['classification-rules'] })
      toast.success('Classification rule deleted')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to delete rule')
    },
  })
}

export interface ColumnClassification {
  id: string
  column_name: string
  data_type: string
  sensitivity_level: 'low' | 'medium' | 'high' | 'critical'
  confidence: number
  classification_tag: string
  status: 'classified' | 'pending' | 'review'
  value_sample?: string | null
}

export interface DatasetClassification {
  id: string
  name: string
  source_id: string
  total_columns: number
  classified_columns: number
  pending_columns: number
  avg_confidence: number
  columns: ColumnClassification[]
}

export function useDatasetClassification(datasetId: string) {
  return useQuery({
    queryKey: ['classification', datasetId],
    queryFn: async () => {
      const response = await api.get<DatasetClassification>(`/classify/datasets/${datasetId}`)
      return response.data
    },
    enabled: !!datasetId,
  })
}

export function useDatasetColumns(datasetId: string) {
  return useQuery({
    queryKey: ['classification', datasetId, 'columns'],
    queryFn: async () => {
      const response = await api.get<ColumnClassification[]>(`/classify/datasets/${datasetId}/columns`)
      return response.data
    },
    enabled: !!datasetId,
  })
}

export function useReclassifyDataset() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (datasetId: string) => {
      const response = await api.post(`/classify/datasets/${datasetId}/reclassify`)
      return response.data
    },
    onSuccess: (_, datasetId) => {
      queryClient.invalidateQueries({ queryKey: ['classification', datasetId] })
      toast.success('Re-classification started')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to start re-classification')
    },
  })
}

// ── Document Classification hooks ─────────────────────────────────────────────

export interface DocumentClassification {
  id: string
  document_id: string
  document_name: string
  entity_types: string[]
  findings: any[]
  governed: boolean
  label_applied: string
  created_at: string
}

export function useDocumentClassifications(documentId: string) {
  return useQuery({
    queryKey: ['doc-classifications', documentId],
    queryFn: async () => {
      const response = await api.get<DocumentClassification[]>(`/documents/${documentId}/classifications`)
      return response.data ?? []
    },
    enabled: !!documentId,
  })
}

export function useClassifyDocument() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (data: { document_id: string; document_name?: string; text: string }) => {
      const response = await api.post('/documents/classify', data)
      return response.data
    },
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['doc-classifications', vars.document_id] })
      queryClient.invalidateQueries({ queryKey: ['review-queue'] })
      toast.success('Document classified')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to classify document')
    },
  })
}
