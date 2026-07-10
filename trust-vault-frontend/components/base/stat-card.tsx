'use client'

import { ArrowUp, ArrowDown } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

interface StatCardProps {
  /** Preferred: label. `title` is supported as an alias. */
  label?: string
  title?: string
  value: string | number
  change?: number
  changeLabel?: string
  /** Accepts either a rendered node or a Lucide icon component */
  icon?: React.ReactNode | LucideIcon
  /** Shorthand trend API: { value: string, positive: boolean } */
  trend?: { value: string; positive: boolean }
  onClick?: () => void
}

function isIconComponent(icon: StatCardProps['icon']): icon is LucideIcon {
  return typeof icon === 'function' || (typeof icon === 'object' && icon !== null && '$$typeof' in (icon as object) && !('props' in (icon as object)))
}

export function StatCard({
  label,
  title,
  value,
  change,
  changeLabel,
  icon,
  trend,
  onClick,
}: StatCardProps) {
  const displayLabel = label ?? title ?? ''
  const isPositive = trend ? trend.positive : change ? change > 0 : false

  let iconNode: React.ReactNode = null
  if (icon) {
    if (isIconComponent(icon)) {
      const Icon = icon
      iconNode = <Icon className="h-5 w-5" />
    } else {
      iconNode = icon as React.ReactNode
    }
  }

  return (
    <div
      onClick={onClick}
      className="rounded-lg border border-border bg-card p-6 hover:bg-card/80 transition-colors"
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <p className="text-sm font-medium text-muted-foreground">{displayLabel}</p>
          <p className="mt-2 text-2xl font-semibold text-foreground">{value}</p>
          {trend && (
            <div className="mt-2 flex items-center gap-1">
              {isPositive ? (
                <ArrowUp className="h-4 w-4 text-green-500" />
              ) : (
                <ArrowDown className="h-4 w-4 text-red-500" />
              )}
              <span className={`text-xs ${isPositive ? 'text-green-500' : 'text-red-500'}`}>
                {trend.value}
              </span>
            </div>
          )}
          {!trend && change !== undefined && (
            <div className="mt-2 flex items-center gap-1">
              {isPositive ? (
                <ArrowUp className="h-4 w-4 text-green-500" />
              ) : (
                <ArrowDown className="h-4 w-4 text-red-500" />
              )}
              <span className={isPositive ? 'text-green-500' : 'text-red-500'}>
                {Math.abs(change)}%
              </span>
              {changeLabel && <span className="text-xs text-muted-foreground">{changeLabel}</span>}
            </div>
          )}
        </div>
        {iconNode && <div className="rounded-lg bg-primary/10 p-2.5 text-primary">{iconNode}</div>}
      </div>
    </div>
  )
}
