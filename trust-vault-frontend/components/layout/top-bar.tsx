'use client'

import { useState } from 'react'
import { Bell, Menu, Moon, Sun, LogOut, Settings, User } from 'lucide-react'
import { useThemeStore, applyTheme } from '@/store/theme'
import { useAuthStore } from '@/store/auth'
import { useLogout } from '@/hooks/use-auth'
import Link from 'next/link'

interface TopBarProps {
  onMenuClick?: () => void
}

export function TopBar({ onMenuClick }: TopBarProps) {
  const { mode, setMode, toggle } = useThemeStore()
  const { user } = useAuthStore()
  const logoutMutation = useLogout()
  const [userMenuOpen, setUserMenuOpen] = useState(false)

  const handleThemeToggle = () => {
    const newMode = mode === 'dark' ? 'light' : 'dark'
    setMode(newMode)
  }

  const handleLogout = () => {
    setUserMenuOpen(false)
    logoutMutation.mutate()
  }

  return (
    <div className="sticky top-0 z-40 border-b border-border bg-card">
      <div className="flex items-center justify-between px-6 py-4">
        {/* Left side - Menu button and logo */}
        <div className="flex items-center gap-4">
          <button
            onClick={onMenuClick}
            className="p-2 hover:bg-muted rounded-lg transition-colors md:hidden"
          >
            <Menu className="h-5 w-5 text-foreground" />
          </button>
          <Link href="/dashboard" className="font-bold text-foreground text-lg">
            SecureLens
          </Link>
        </div>

        {/* Right side - Actions */}
        <div className="flex items-center gap-4">
          {/* Notifications */}
          <button className="p-2 hover:bg-muted rounded-lg transition-colors relative">
            <Bell className="h-5 w-5 text-foreground" />
            <span className="absolute top-1 right-1 h-2 w-2 bg-destructive rounded-full" />
          </button>

          {/* Theme toggle */}
          <button
            onClick={handleThemeToggle}
            className="p-2 hover:bg-muted rounded-lg transition-colors"
          >
            {mode === 'dark' || (mode === 'system' && window?.matchMedia('(prefers-color-scheme: dark)').matches) ? (
              <Sun className="h-5 w-5 text-foreground" />
            ) : (
              <Moon className="h-5 w-5 text-foreground" />
            )}
          </button>

          {/* Logout button - always visible */}
          <button
            onClick={handleLogout}
            className="p-2 hover:bg-muted rounded-lg transition-colors text-muted-foreground hover:text-destructive"
            title="Logout"
            data-testid="logout-button"
          >
            <LogOut className="h-5 w-5" />
          </button>

          {/* User menu */}
          <div className="relative">
            <button
              onClick={() => setUserMenuOpen(!userMenuOpen)}
              className="flex items-center gap-3 p-2 hover:bg-muted rounded-lg transition-colors"
            >
              <div className="h-8 w-8 rounded-full bg-primary/20 flex items-center justify-center">
                <span className="text-xs font-semibold text-primary">
                  {user?.name?.charAt(0).toUpperCase()}
                </span>
              </div>
              <span className="text-sm font-medium text-foreground hidden sm:block">{user?.name}</span>
            </button>

            {/* User dropdown menu */}
            {userMenuOpen && (
              <div className="absolute right-0 mt-2 w-48 rounded-lg border border-border bg-card shadow-lg py-2">
                <Link href="/profile" className="block px-4 py-2 text-left text-sm text-foreground hover:bg-muted flex items-center gap-2 transition-colors">
                  <User className="h-4 w-4" />
                  Profile
                </Link>
                <Link href="/settings" className="block px-4 py-2 text-left text-sm text-foreground hover:bg-muted flex items-center gap-2 transition-colors">
                  <Settings className="h-4 w-4" />
                  Settings
                </Link>
                <hr className="my-2 border-border" />
                <button
                  onClick={handleLogout}
                  className="w-full px-4 py-2 text-left text-sm text-destructive hover:bg-muted flex items-center gap-2 transition-colors"
                >
                  <LogOut className="h-4 w-4" />
                  Logout
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
