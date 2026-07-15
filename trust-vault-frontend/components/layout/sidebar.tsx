'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import {
  ChevronRight,
  LayoutDashboard,
  Map,
  GitBranch,
  Database,
  ScanSearch,
  FileStack,
  Tags,
  ShieldCheck,
  Bot,
  Wrench,
  BarChart3,
  Trash2,
  MessageSquarePlus,
  UserCheck,
  Sparkles,
  ClipboardList,
  FileText,
  Plug,
  CalendarClock,
  Activity,
  Settings,
  Monitor,
  Globe,
} from 'lucide-react'
import { cn } from '@/lib/utils'

interface SidebarProps {
  isOpen?: boolean
  onClose?: () => void
}

interface NavItem {
  icon: typeof LayoutDashboard
  label: string
  href: string
  description?: string
}

interface NavGroup {
  title: string
  emoji: string
  items: NavItem[]
}

const navGroups: NavGroup[] = [
  {
    title: 'Insights',
    emoji: '📊',
    items: [
      { icon: LayoutDashboard, label: 'Dashboard', href: '/dashboard', description: 'Overview metrics & health' },
      { icon: Map, label: 'Data Map', href: '/data-map', description: 'Where is your data?' },
      { icon: GitBranch, label: 'Lineage', href: '/lineage', description: 'How does data flow?' },
    ],
  },
  {
    title: 'Discover',
    emoji: '🔍',
    items: [
      { icon: Database, label: 'Data Sources', href: '/data-sources', description: 'Connect your systems' },
      { icon: ScanSearch, label: 'Classification', href: '/classification', description: 'Auto-detect sensitive data' },
      { icon: FileStack, label: 'Documents', href: '/documents', description: 'Process files & PDFs' },
      { icon: Tags, label: 'Labels', href: '/labels', description: 'Sensitivity tags' },
      { icon: Globe, label: 'Data Residency', href: '/data-map/residency', description: 'Geographic data controls' },
    ],
  },
  {
    title: 'Protect',
    emoji: '🛡️',
    items: [
      { icon: ShieldCheck, label: 'Policies', href: '/governance', description: 'Governance rules' },
      { icon: Bot, label: 'AI Gate', href: '/ai-gate', description: 'LLM guardrails' },
      { icon: Wrench, label: 'Remediation', href: '/ai-governance', description: 'Fix data issues' },
    ],
  },
  {
    title: 'Improve',
    emoji: '📈',
    items: [
      { icon: BarChart3, label: 'Quality', href: '/quality', description: 'Data accuracy scores' },
      { icon: Trash2, label: 'ROT Analysis', href: '/rot', description: 'Cleanup data waste' },
      { icon: MessageSquarePlus, label: 'Feedback', href: '/feedback', description: 'Train the system' },
      { icon: Database, label: 'Critical Elements', href: '/quality/cde', description: 'CDE management' },
    ],
  },
  {
    title: 'Comply',
    emoji: '✅',
    items: [
      { icon: UserCheck, label: 'Privacy Center', href: '/privacy', description: 'DSAR & consent' },
      { icon: FileText, label: 'DPIA', href: '/privacy/dpia', description: 'Impact assessments' },
      { icon: Sparkles, label: 'Advisor', href: '/advisor', description: 'AI recommendations' },
      { icon: FileText, label: 'Reports', href: '/audit/reports', description: 'Compliance & analytics reports' },
    ],
  },
  {
    title: 'System',
    emoji: '⚙️',
    items: [
      { icon: ClipboardList, label: 'Audit Trail', href: '/audit', description: 'System activity log' },
      { icon: Plug, label: 'Integrations', href: '/integrations', description: 'External connections' },
      { icon: CalendarClock, label: 'Jobs', href: '/jobs', description: 'Scheduled tasks' },
      { icon: Activity, label: 'Observability', href: '/observability', description: 'Health & metrics' },
      { icon: Monitor, label: 'Endpoints', href: '/endpoints', description: 'Endpoint scanning' },
      { icon: Settings, label: 'Settings', href: '/settings', description: 'Configuration' },
    ],
  },
]

export function Sidebar({ isOpen = true, onClose }: SidebarProps) {
  const pathname = usePathname()

  return (
    <>
      {/* Mobile overlay */}
      {!isOpen && (
        <div
          className="fixed inset-0 z-30 bg-black/50 md:hidden"
          onClick={onClose}
        />
      )}

      {/* Sidebar */}
      <aside
        className={cn(
          'fixed left-0 top-16 z-40 flex h-[calc(100vh-64px)] w-64 flex-col border-r border-border bg-card transition-transform duration-300 md:relative md:top-0 md:translate-x-0',
          !isOpen && '-translate-x-full md:translate-x-0',
        )}
      >
        <nav className="flex-1 overflow-y-auto p-4">
          {navGroups.map((group) => (
            <div key={group.title} className="mb-5">
              <p className="px-3 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground flex items-center gap-1.5">
                <span>{group.emoji}</span>
                <span>{group.title}</span>
              </p>
              <div className="space-y-0.5">
                {group.items.map((item) => {
                  const Icon = item.icon
                  const isActive =
                    pathname === item.href || pathname.startsWith(item.href + '/')

                  return (
                    <Link
                      key={item.href}
                      href={item.href}
                      onClick={onClose}
                      title={item.description}
                      className={cn(
                        'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                        isActive
                          ? 'bg-primary/10 text-primary'
                          : 'text-foreground hover:bg-muted',
                      )}
                    >
                      <Icon className="h-4 w-4 flex-shrink-0" />
                      <span className="flex-1">{item.label}</span>
                      {isActive && <ChevronRight className="h-4 w-4" />}
                    </Link>
                  )
                })}
              </div>
            </div>
          ))}
        </nav>

        {/* Footer section */}
        <div className="border-t border-border p-4">
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span className="flex items-center gap-1.5">
              <span className="h-2 w-2 rounded-full bg-green-500" aria-hidden="true" />
              All systems healthy
            </span>
            <span>v1.0.0</span>
          </div>
          <div className="mt-3 text-[10px] text-muted-foreground/70 text-center">
            Powered by{' '}
            <a href="https://plainsurf.com/" target="_blank" rel="noopener noreferrer" className="hover:text-foreground transition-colors underline">
              Plainsurf LLC FZ
            </a>
            {' '}Dubai, UAE © 2026
          </div>
        </div>
      </aside>
    </>
  )
}
