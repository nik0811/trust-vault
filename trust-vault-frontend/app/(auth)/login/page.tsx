'use client'

import { useRouter } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { loginSchema, type LoginInput } from '@/lib/schemas'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import { toast } from 'sonner'
import Link from 'next/link'
import Cookies from 'js-cookie'
import { useState } from 'react'

export default function LoginPage() {
  const router = useRouter()
  const { login } = useAuthStore()
  const [isLoading, setIsLoading] = useState(false)

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginInput>({
    resolver: zodResolver(loginSchema),
  })

  const onSubmit = async (data: LoginInput) => {
    setIsLoading(true)
    try {
      const response = await api.post('/auth/login', {
        email: data.email,
        password: data.password,
      })

      const { access_token, refresh_token, expires_in } = response.data

      Cookies.set('accessToken', access_token, { expires: 7 })
      if (refresh_token) {
        Cookies.set('refreshToken', refresh_token, { expires: 30 })
      }

      login(access_token, { 
        id: '', 
        email: data.email, 
        name: data.email.split('@')[0], 
        role: 'admin', 
        tenantId: '' 
      })

      toast.success('Logged in successfully')
      router.push('/dashboard')
    } catch (apiError: any) {
      toast.error(apiError.response?.data?.error || 'Invalid email or password')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm">
        <div className="rounded-lg border border-border bg-card p-8 shadow-lg">
          {/* Header */}
          <div className="text-center mb-8">
            <h1 className="text-3xl font-bold text-foreground">TrustVault</h1>
            <p className="text-sm text-muted-foreground mt-2">Enterprise Data Governance</p>
          </div>

          {/* Form */}
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            {/* Email field */}
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Email</label>
              <input
                type="email"
                {...register('email')}
                className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
                placeholder="you@example.com"
              />
              {errors.email && (
                <p className="text-xs text-destructive mt-1">{errors.email.message}</p>
              )}
            </div>

            {/* Password field */}
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Password</label>
              <input
                type="password"
                {...register('password')}
                className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
                placeholder="••••••••"
              />
              {errors.password && (
                <p className="text-xs text-destructive mt-1">{errors.password.message}</p>
              )}
            </div>

            {/* Remember me */}
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                {...register('rememberMe')}
                className="w-4 h-4 rounded border-border"
              />
              <span className="text-sm text-foreground">Remember me</span>
            </label>

            {/* Submit button */}
            <button
              type="submit"
              disabled={isLoading}
              className="w-full py-2 rounded-lg bg-primary text-primary-foreground font-medium hover:bg-primary/90 disabled:opacity-50 transition-colors mt-6"
            >
              {isLoading ? 'Logging in...' : 'Sign In'}
            </button>
          </form>

          {/* Footer */}
          <div className="mt-6 pt-6 border-t border-border">
            <p className="text-center text-sm text-muted-foreground">
              Forgot your password?{' '}
              <Link href="/forgot-password" className="text-primary hover:underline">
                Reset it here
              </Link>
            </p>
          </div>
        </div>
        <p className="text-center text-[11px] text-muted-foreground/60 mt-6">
          Powered by{' '}
          <a href="https://plainsurf.com/" target="_blank" rel="noopener noreferrer" className="hover:text-foreground transition-colors underline">
            Plainsurf LLC FZ
          </a>
          {' '}Dubai, UAE © 2026
        </p>
      </div>
    </div>
  )
}
