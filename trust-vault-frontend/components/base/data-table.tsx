'use client'

import { useState } from 'react'
import { ChevronUp, ChevronDown, ChevronsUpDown } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface Column<T> {
  id: string
  header: string
  accessorKey?: keyof T
  cell?: (row: T, index: number) => React.ReactNode
  width?: string
  sortable?: boolean
}

interface DataTableProps<T> {
  columns: Column<T>[]
  data: T[]
  onRowClick?: (row: T) => void
  isLoading?: boolean
  emptyMessage?: string
  className?: string
}

export function DataTable<T extends { id?: string }>({
  columns,
  data,
  onRowClick,
  isLoading,
  emptyMessage = 'No data available',
  className,
}: DataTableProps<T>) {
  const [sortConfig, setSortConfig] = useState<{
    key: string
    direction: 'asc' | 'desc'
  } | null>(null)

  const handleSort = (columnId: string) => {
    setSortConfig((prev) => {
      if (prev?.key === columnId) {
        return { key: columnId, direction: prev.direction === 'asc' ? 'desc' : 'asc' }
      }
      return { key: columnId, direction: 'asc' }
    })
  }

  const sortedData = sortConfig
    ? [...data].sort((a, b) => {
        const column = columns.find((c) => c.id === sortConfig.key)
        if (!column?.accessorKey) return 0

        const aValue = a[column.accessorKey]
        const bValue = b[column.accessorKey]

        if (aValue < bValue) return sortConfig.direction === 'asc' ? -1 : 1
        if (aValue > bValue) return sortConfig.direction === 'asc' ? 1 : -1
        return 0
      })
    : data

  if (isLoading) {
    return <div className="py-8 text-center text-muted-foreground">Loading...</div>
  }

  if (data.length === 0) {
    return <div className="py-8 text-center text-muted-foreground">{emptyMessage}</div>
  }

  return (
    <div className={cn('overflow-x-auto rounded-lg border border-border', className)}>
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border bg-muted/50 hover:bg-muted/70">
            {columns.map((column) => (
              <th key={column.id} className="px-4 py-3 text-left font-medium text-foreground">
                <button
                  onClick={() => column.sortable && handleSort(column.id)}
                  className={cn(
                    'flex items-center gap-2',
                    column.sortable && 'cursor-pointer hover:text-primary',
                  )}
                >
                  {column.header}
                  {column.sortable && (
                    <span>
                      {sortConfig?.key === column.id ? (
                        sortConfig.direction === 'asc' ? (
                          <ChevronUp className="h-4 w-4" />
                        ) : (
                          <ChevronDown className="h-4 w-4" />
                        )
                      ) : (
                        <ChevronsUpDown className="h-4 w-4 text-muted-foreground" />
                      )}
                    </span>
                  )}
                </button>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {sortedData.map((row, index) => (
            <tr
              key={row.id || index}
              onClick={() => onRowClick?.(row)}
              className={cn(
                'border-b border-border',
                onRowClick && 'cursor-pointer hover:bg-muted/50',
              )}
            >
              {columns.map((column) => (
                <td key={column.id} className="px-4 py-3 text-foreground">
                  {column.cell
                    ? column.cell(row, index)
                    : column.accessorKey
                      ? String(row[column.accessorKey] || '—')
                      : '—'}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
