'use client'

import { Breadcrumbs } from '@/components/base/breadcrumbs'
import Link from 'next/link'
import { Users, Key, Bell, Palette, Shield } from 'lucide-react'

export default function SettingsPage() {
  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Settings', active: true }]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">Settings</h1>
        <p className="text-sm text-muted-foreground mt-1">Manage your SecureLens configuration</p>
      </div>

      {/* Content */}
      <div className="p-8">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 max-w-4xl">
          <Link
            href="/settings/users"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <Users className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Users & Roles</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Manage users, roles, and permissions
            </p>
          </Link>

          <Link
            href="/settings/api-keys"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <Key className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">API Keys</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Manage API keys for service-to-service authentication
            </p>
          </Link>

          <Link
            href="/settings/sso"
            className="rounded-lg border border-border bg-card p-6 hover:border-primary/50 transition-colors"
          >
            <Shield className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Single Sign-On (SSO)</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Configure OIDC and SAML identity providers
            </p>
          </Link>

          <div className="rounded-lg border border-border bg-card p-6">
            <Bell className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Notifications</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Configure email and webhook notifications
            </p>
            <p className="text-xs text-muted-foreground mt-4">Coming soon</p>
          </div>

          <div className="rounded-lg border border-border bg-card p-6">
            <Palette className="h-8 w-8 text-primary mb-4" />
            <h3 className="text-lg font-semibold text-foreground">Appearance</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Customize theme and branding
            </p>
            <p className="text-xs text-muted-foreground mt-4">Coming soon</p>
          </div>
        </div>
      </div>
    </div>
  )
}
