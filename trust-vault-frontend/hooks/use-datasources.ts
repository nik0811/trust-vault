import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { toast } from 'sonner'

export interface DataSource {
  id: string
  name: string
  type: string
  config: Record<string, any> | null
  status: string
  last_scan: string
  created_at: string
  updated_at: string
}

interface CreateDataSourceRequest {
  name: string
  type: 'postgres' | 'mysql' | 's3' | 'snowflake' | 'bigquery' | 'file'
  config?: Record<string, any>
}

export function useDataSources() {
  return useQuery({
    queryKey: ['datasources'],
    queryFn: async () => {
      const response = await api.get<DataSource[]>('/datasources')
      return response.data
    },
  })
}

export function useDataSource(id: string) {
  return useQuery({
    queryKey: ['datasources', id],
    queryFn: async () => {
      const response = await api.get<DataSource>(`/datasources/${id}`)
      return response.data
    },
    enabled: !!id,
  })
}

export function useCreateDataSource() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: CreateDataSourceRequest) => {
      const response = await api.post<DataSource>('/datasources', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['datasources'] })
      toast.success('Data source created successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create data source')
    },
  })
}

export function useUpdateDataSource() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ id, data }: { id: string; data: Partial<CreateDataSourceRequest> }) => {
      const response = await api.put<DataSource>(`/datasources/${id}`, data)
      return response.data
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['datasources'] })
      queryClient.invalidateQueries({ queryKey: ['datasources', variables.id] })
      toast.success('Data source updated successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to update data source')
    },
  })
}

export function useDeleteDataSource() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/datasources/${id}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['datasources'] })
      toast.success('Data source deleted successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to delete data source')
    },
  })
}

export function useTriggerScan() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      const response = await api.post(`/datasources/${id}/scan`)
      return response.data
    },
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ['datasources', id] })
      // Don't show toast here - the UI shows scanning state
    },
    onError: (error: any) => {
      if (error.response?.status === 409) {
        toast.info('A scan is already in progress')
      } else {
        toast.error(error.response?.data?.error || 'Failed to trigger scan')
      }
    },
  })
}

export function useScanStatus(id: string) {
  return useQuery({
    queryKey: ['datasources', id, 'status'],
    queryFn: async () => {
      const response = await api.get(`/datasources/${id}/status`)
      return response.data
    },
    enabled: !!id,
    refetchInterval: 5000,
  })
}

export interface ScanLogEntry {
  time: string
  message: string
}

export interface ScanLog {
  id: string
  datasource_id: string
  status: 'running' | 'success' | 'failed' | 'completed'
  started_at: string
  completed_at?: string
  message: string
  logs: ScanLogEntry[]
  datasets_discovered: number
  created_at: string
}

export function useScanLogs(datasourceId: string) {
  return useQuery({
    queryKey: ['datasources', datasourceId, 'logs'],
    queryFn: async () => {
      const response = await api.get<ScanLog[]>(`/datasources/${datasourceId}/logs`)
      return response.data
    },
    enabled: !!datasourceId,
  })
}
