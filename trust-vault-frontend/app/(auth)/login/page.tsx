'use client'

import { useRouter, useSearchParams } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { loginSchema, type LoginInput } from '@/lib/schemas'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import { toast } from 'sonner'
import Link from 'next/link'
import Cookies from 'js-cookie'
import { useState, useEffect } from 'react'

interface SSOProvider {
  id: string
  name: string
  type: 'oidc' | 'saml'
}

export default function LoginPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { login } = useAuthStore()
  const [isLoading, setIsLoading] = useState(false)
  const [ssoProviders, setSSOProviders] = useState<SSOProvider[]>([])
  const [tenantSlug, setTenantSlug] = useState('')

  // Show error from SSO callback if present
  useEffect(() => {
    const error = searchParams.get('error')
    if (error) {
      toast.error(error)
    }
  }, [searchParams])

  // Fetch SSO providers when tenant slug changes
  useEffect(() => {
    if (tenantSlug.length >= 2) {
      const timer = setTimeout(() => {
        fetchSSOProviders(tenantSlug)
      }, 500)
      return () => clearTimeout(timer)
    } else {
      setSSOProviders([])
    }
  }, [tenantSlug])

  const fetchSSOProviders = async (slug: string) => {
    try {
      const response = await api.get(`/auth/sso/providers?tenant=${slug}`)
      setSSOProviders(response.data || [])
    } catch {
      setSSOProviders([])
    }
  }

  const handleSSOLogin = (provider: SSOProvider) => {
    const baseUrl = process.env.NEXT_PUBLIC_API_URL || 'https://api.securelens.ai'
    const ssoUrl = provider.type === 'oidc'
      ? `${baseUrl}/api/v1/auth/sso/oidc/${provider.id}?tenant=${tenantSlug}`
      : `${baseUrl}/api/v1/auth/sso/saml/${provider.id}?tenant=${tenantSlug}`
    window.location.href = ssoUrl
  }

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
            <h1 className="text-3xl font-bold text-foreground">SecureLens</h1>
            <p className="text-sm text-muted-foreground mt-2">Enterprise Data Governance</p>
          </div>

          {/* SSO Section */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-foreground mb-2">
              Organization (for SSO)
            </label>
            <input
              type="text"
              value={tenantSlug}
              onChange={(e) => setTenantSlug(e.target.value.toLowerCase())}
              className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
              placeholder="your-organization"
            />
            {ssoProviders.length > 0 && (
              <div className="mt-3 space-y-2">
                {ssoProviders.map((provider) => (
                  <button
                    key={provider.id}
                    type="button"
                    onClick={() => handleSSOLogin(provider)}
                    className="w-full py-2 px-4 rounded-lg border border-border bg-background text-foreground hover:bg-muted transition-colors flex items-center justify-center gap-2"
                  >
                    {provider.type === 'oidc' ? (
                      <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
                        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
                      </svg>
                    ) : (
                      <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
                        <path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm0 10.99h7c-.53 4.12-3.28 7.79-7 8.94V12H5V6.3l7-3.11v8.8z"/>
                      </svg>
                    )}
                    Sign in with {provider.name}
                  </button>
                ))}
              </div>
            )}
          </div>

          {ssoProviders.length > 0 && (
            <div className="relative mb-6">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-border"></div>
              </div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="bg-card px-2 text-muted-foreground">Or continue with email</span>
              </div>
            </div>
          )}

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
          {' '}Dubai, UAE · © 2026
        </p>
      </div>
    </div>
  )
}
