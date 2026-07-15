import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { toast } from 'sonner'

export type ReportType = 'compliance' | 'quality' | 'ai_usage' | 'audit'

export interface Report {
  id: string
  tenant_id: string
  type: ReportType
  status: 'generating' | 'completed' | 'failed'
  content: any | null
  generated_at: string
  created_at: string
  updated_at: string
}

export interface GenerateReportRequest {
  type: ReportType
  date_from?: string
  date_to?: string
}

export function useReports() {
  return useQuery({
    queryKey: ['reports'],
    queryFn: async () => {
      const response = await api.get<Report[]>('/reports')
      return response.data ?? []
    },
  })
}

export function useReport(id: string) {
  return useQuery({
    queryKey: ['reports', id],
    queryFn: async () => {
      const response = await api.get<Report>(`/reports/${id}`)
      return response.data
    },
    enabled: !!id,
  })
}

export function useGenerateReport() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: GenerateReportRequest) => {
      const response = await api.post<Report>('/reports', data)
      return response.data
    },
    onSuccess: (report) => {
      queryClient.invalidateQueries({ queryKey: ['reports'] })
      toast.success(`${report.type.replace('_', ' ')} report generated successfully`)
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to generate report')
    },
  })
}
