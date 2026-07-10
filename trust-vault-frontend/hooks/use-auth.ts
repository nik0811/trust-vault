import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { useAuthStore } from '@/store/auth'
import { useRouter } from 'next/navigation'
import Cookies from 'js-cookie'
import { toast } from 'sonner'

interface LoginRequest {
  email: string
  password: string
}

interface LoginResponse {
  access_token: string
  refresh_token?: string
  expires_in: number
}

interface User {
  id: string
  email: string
  name: string
  status: string
  is_super_admin: boolean
  created_at: string
}

export function useLogin() {
  const router = useRouter()
  const { login } = useAuthStore()

  return useMutation({
    mutationFn: async (data: LoginRequest) => {
      const response = await api.post<LoginResponse>('/auth/login', data)
      return response.data
    },
    onSuccess: (data) => {
      Cookies.set('accessToken', data.access_token, { expires: 7 })
      if (data.refresh_token) {
        Cookies.set('refreshToken', data.refresh_token, { expires: 30 })
      }
      login(data.access_token, { id: '', email: '', name: '', role: 'admin', tenantId: '' })
      toast.success('Logged in successfully')
      router.push('/dashboard')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Login failed')
    },
  })
}

export function useLogout() {
  const router = useRouter()
  const { logout } = useAuthStore()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async () => {
      await api.post('/auth/logout')
    },
    onSuccess: () => {
      Cookies.remove('accessToken')
      Cookies.remove('refreshToken')
      logout()
      queryClient.clear()
      router.push('/login')
      toast.success('Logged out successfully')
    },
    onSettled: () => {
      Cookies.remove('accessToken')
      Cookies.remove('refreshToken')
      logout()
    },
  })
}

export function useCurrentUser() {
  return useQuery({
    queryKey: ['currentUser'],
    queryFn: async () => {
      const response = await api.get<User>('/auth/me')
      return response.data
    },
    retry: false,
  })
}

export function useUsers() {
  return useQuery({
    queryKey: ['users'],
    queryFn: async () => {
      const response = await api.get<User[]>('/users')
      return response.data
    },
  })
}

export function useCreateUser() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { email: string; password: string; name: string }) => {
      const response = await api.post<User>('/users', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      toast.success('User created successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create user')
    },
  })
}

export function useDeleteUser() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/users/${id}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      toast.success('User deleted successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to delete user')
    },
  })
}

export function useUpdateProfile() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { name?: string; phone?: string; location?: string; department?: string; bio?: string }) => {
      const response = await api.put('/auth/me', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['currentUser'] })
      toast.success('Profile updated')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to update profile')
    },
  })
}

export interface APIKey {
  id: string
  name: string
  prefix: string
  created_at: string
  last_used?: string
}

export function useAPIKeys() {
  return useQuery({
    queryKey: ['api-keys'],
    queryFn: async () => {
      const response = await api.get<APIKey[]>('/auth/api-keys')
      return response.data
    },
  })
}

export function useCreateAPIKey() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { name: string }) => {
      const response = await api.post<{ key: string; id: string }>('/auth/api-keys', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] })
      toast.success('API key created')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create API key')
    },
  })
}

export function useDeleteAPIKey() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/auth/api-keys/${id}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] })
      toast.success('API key deleted')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to delete API key')
    },
  })
}

// Invitation hooks

export interface Invitation {
  id: string
  tenant_id: string
  email: string
  role: string
  invited_by?: string
  expires_at: string
  accepted_at?: string
  created_at: string
  invite_url?: string
}

export function useInvitations() {
  return useQuery({
    queryKey: ['invitations'],
    queryFn: async () => {
      const response = await api.get<Invitation[]>('/invitations')
      return response.data
    },
  })
}

export function useCreateInvitation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { email: string; role: string }) => {
      const response = await api.post<Invitation>('/invitations', data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['invitations'] })
      toast.success('Invitation sent successfully')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to send invitation')
    },
  })
}

export function useCancelInvitation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/invitations/${id}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['invitations'] })
      toast.success('Invitation cancelled')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to cancel invitation')
    },
  })
}

export function useResendInvitation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      const response = await api.post<{ status: string; invite_url: string }>(`/invitations/${id}/resend`)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['invitations'] })
      toast.success('Invitation resent')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to resend invitation')
    },
  })
}
