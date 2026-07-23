'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { TopBar } from '@/components/layout/top-bar'
import { Sidebar } from '@/components/layout/sidebar'
import { useAuthStore } from '@/store/auth'
import { useUIStore } from '@/store/ui'
import Cookies from 'js-cookie'

function decodeJWT(token: string): { sub?: string; tenant_id?: string; is_super_admin?: boolean } | null {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) return null
    const payload = JSON.parse(atob(parts[1]))
    return payload
  } catch {
    return null
  }
}

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const { isAuthenticated, login } = useAuthStore()
  const { sidebarOpen, toggleSidebar } = useUIStore()
  const [isMounted, setIsMounted] = useState(false)
  const [isChecking, setIsChecking] = useState(true)

  useEffect(() => {
    setIsMounted(true)
    
    // Check for existing token in cookies and restore session
    const token = Cookies.get('accessToken')
    if (token && !isAuthenticated) {
      // Decode JWT to get user info
      const payload = decodeJWT(token)
      const userEmail = payload?.sub || 'user'
      const isSuperAdmin = payload?.is_super_admin || false
      
      // Restore session from cookie with decoded user info
      login(token, { 
        id: payload?.sub || '', 
        email: userEmail, 
        name: isSuperAdmin ? 'Super Admin' : userEmail.split('@')[0], 
        role: isSuperAdmin ? 'superadmin' : 'admin', 
        tenantId: payload?.tenant_id || '' 
      })
    }
    
    setIsChecking(false)
  }, [])

  useEffect(() => {
    // Only redirect after initial check is complete
    if (!isChecking && !isAuthenticated) {
      const token = Cookies.get('accessToken')
      if (!token) {
        router.push('/login')
      }
    }
  }, [isChecking, isAuthenticated, router])

  if (!isMounted || isChecking) {
    return null
  }

  return (
    <div className="h-screen overflow-hidden bg-background">
      {/* Top bar - fixed at top */}
      <TopBar onMenuClick={toggleSidebar} />

      {/* Main container below top bar */}
      <div className="flex h-[calc(100vh-64px)]">
        {/* Sidebar - fixed position, never scrolls with content */}
        <Sidebar isOpen={sidebarOpen} onClose={toggleSidebar} />

        {/* Main content - this is the ONLY scrollable area */}
        <main className="flex-1 overflow-y-auto">
          <div className="p-6">{children}</div>
        </main>
      </div>
    </div>
  )
}
