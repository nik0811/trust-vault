import axios, { AxiosError, AxiosInstance } from 'axios'
import Cookies from 'js-cookie'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1'

interface ApiError {
  message: string
  code?: string
  status?: number
}

const createApiClient = (): AxiosInstance => {
  const client = axios.create({
    baseURL: API_BASE_URL,
    timeout: 30000,
  })

  // Request interceptor to add JWT token
  client.interceptors.request.use((config) => {
    const token = Cookies.get('accessToken')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  })

  // Response interceptor for error handling
  client.interceptors.response.use(
    (response) => response,
    (error: AxiosError<any>) => {
      if (error.response?.status === 401) {
        // Token expired or invalid
        Cookies.remove('accessToken')
        Cookies.remove('refreshToken')
        window.location.href = '/login'
      }
      return Promise.reject(error)
    },
  )

  return client
}

export const api = createApiClient()

export const apiCall = async <T>(
  method: 'get' | 'post' | 'put' | 'delete' | 'patch',
  url: string,
  data?: any,
): Promise<T> => {
  try {
    const response = await api[method]<T>(url, data)
    return response.data
  } catch (error) {
    throw handleApiError(error)
  }
}

export const handleApiError = (error: unknown): ApiError => {
  if (axios.isAxiosError(error)) {
    return {
      message: error.response?.data?.message || error.message,
      code: error.response?.data?.code,
      status: error.response?.status,
    }
  }
  return {
    message: 'An unknown error occurred',
  }
}

export default api
