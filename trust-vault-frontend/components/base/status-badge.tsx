'use client'

import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const badgeVariants = cva(
  'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors',
  {
    variants: {
      variant: {
        default:
          'border border-border bg-muted text-muted-foreground hover:bg-muted/80',
        primary:
          'border border-primary/30 bg-primary/10 text-primary hover:bg-primary/20',
        success:
          'border border-green-500/30 bg-green-500/10 text-green-600 dark:text-green-400 hover:bg-green-500/20',
        warning:
          'border border-yellow-500/30 bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 hover:bg-yellow-500/20',
        destructive:
          'border border-destructive/30 bg-destructive/10 text-destructive hover:bg-destructive/20',
        info: 'border border-blue-500/30 bg-blue-500/10 text-blue-600 dark:text-blue-400 hover:bg-blue-500/20',
      },
      size: {
        sm: 'text-xs px-2 py-1',
        md: 'text-sm px-2.5 py-1.5',
        lg: 'text-base px-3 py-2',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'md',
    },
  },
)

type StatusKey =
  | 'active'
  | 'inactive'
  | 'success'
  | 'warning'
  | 'error'
  | 'pending'

const statusToVariant: Record<StatusKey, VariantProps<typeof badgeVariants>['variant']> = {
  active: 'success',
  inactive: 'default',
  success: 'success',
  warning: 'warning',
  error: 'destructive',
  pending: 'info',
}

const statusDotColors: Record<StatusKey, string> = {
  active: 'bg-green-500',
  inactive: 'bg-muted-foreground',
  success: 'bg-green-500',
  warning: 'bg-yellow-500',
  error: 'bg-red-500',
  pending: 'bg-blue-500',
}

interface StatusBadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {
  /** Shorthand API: maps a status to a variant and renders a dot + label */
  status?: StatusKey
  label?: string
}

export function StatusBadge({
  className,
  variant,
  size,
  status,
  label,
  children,
  ...props
}: StatusBadgeProps) {
  const resolvedVariant = status ? statusToVariant[status] : variant

  return (
    <div className={cn(badgeVariants({ variant: resolvedVariant, size: size ?? 'sm' }), className)} {...props}>
      {status && (
        <span
          className={`h-1.5 w-1.5 rounded-full ${statusDotColors[status]} mr-1.5`}
          aria-hidden="true"
        />
      )}
      {label ?? children}
    </div>
  )
}

interface StatusIndicatorProps {
  status: 'success' | 'warning' | 'error' | 'pending'
  label: string
  size?: 'sm' | 'md' | 'lg'
}

export function StatusIndicator({ status, label, size = 'md' }: StatusIndicatorProps) {
  return <StatusBadge status={status} label={label} size={size} />
}
