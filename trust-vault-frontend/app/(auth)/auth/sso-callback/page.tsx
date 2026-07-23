'use client'

import { useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import Cookies from 'js-cookie'
import { toast } from 'sonner'

export default function SSOCallbackPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { login } = useAuthStore()

  useEffect(() => {
    const token = searchParams.get('token')
    const error = searchParams.get('error')

    if (error) {
      toast.error(error)
      router.push('/login')
      return
    }

    if (token) {
      // Store the token
      Cookies.set('accessToken', token, { expires: 7 })
      
      // Parse JWT to get user info (basic decode, not verification)
      try {
        const payload = JSON.parse(atob(token.split('.')[1]))
        login(token, {
          id: payload.sub || '',
          email: payload.email || '',
          name: payload.name || payload.email?.split('@')[0] || '',
          role: 'user',
          tenantId: payload.tenant_id || ''
        })
        toast.success('Logged in successfully via SSO')
        router.push('/dashboard')
      } catch (e) {
        toast.error('Invalid token received')
        router.push('/login')
      }
    } else {
      router.push('/login')
    }
  }, [searchParams, router, login])

  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
        <p className="text-muted-foreground">Completing SSO login...</p>
      </div>
    </div>
  )
}
