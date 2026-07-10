'use client'

import Link from 'next/link'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { StatCard } from '@/components/base/stat-card'
import { Users, Server, Shield, Activity, Settings, KeyRound, Database, ScrollText } from 'lucide-react'

const adminSections = [
  { title: 'User Management', description: 'Invite, deactivate, and assign roles', href: '/settings/users', icon: Users },
  { title: 'API Keys', description: 'Programmatic access management', href: '/settings/api-keys', icon: KeyRound },
  { title: 'Data Sources', description: 'Connections and sync schedules', href: '/data-sources', icon: Database },
  { title: 'Audit Trail', description: 'Full platform activity log', href: '/audit', icon: ScrollText },
  { title: 'System Health', description: 'Service status and alerts', href: '/observability', icon: Activity },
  { title: 'Workspace Settings', description: 'Organization-wide configuration', href: '/settings', icon: Settings },
]

export default function AdminPage() {
  return (
    <div className="space-y-6">
      <Breadcrumbs items={[{ label: 'Admin' }]} />

      <div>
        <h1 className="text-2xl font-bold text-foreground">Admin Panel</h1>
        <p className="text-muted-foreground mt-1">Platform administration and system oversight</p>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard title="Active Users" value="42" icon={Users} trend={{ value: '+3 this month', positive: true }} />
        <StatCard title="Services Healthy" value="11/12" icon={Server} trend={{ value: '1 degraded', positive: false }} />
        <StatCard title="Policies Enforced" value="28" icon={Shield} />
        <StatCard title="Events (24h)" value="18,204" icon={Activity} trend={{ value: '+12% vs yesterday', positive: true }} />
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {adminSections.map((s) => (
          <Link
            key={s.href}
            href={s.href}
            className="group rounded-lg border border-border bg-card p-6 hover:border-primary/40 transition-colors"
          >
            <div className="rounded-lg bg-primary/10 p-2.5 w-fit mb-4">
              <s.icon className="h-5 w-5 text-primary" />
            </div>
            <h3 className="font-semibold text-foreground group-hover:text-primary transition-colors">{s.title}</h3>
            <p className="text-sm text-muted-foreground mt-1">{s.description}</p>
          </Link>
        ))}
      </div>
    </div>
  )
}
