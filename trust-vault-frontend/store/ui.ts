import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface UIState {
  sidebarOpen: boolean
  activeFilters: Record<string, any>
  currentPage: number
  pageSize: number

  // Actions
  setSidebarOpen: (open: boolean) => void
  toggleSidebar: () => void
  setActiveFilters: (filters: Record<string, any>) => void
  clearFilters: () => void
  setCurrentPage: (page: number) => void
  setPageSize: (size: number) => void
}

export const useUIStore = create<UIState>()(
  persist(
    (set, get) => ({
      sidebarOpen: true,
      activeFilters: {},
      currentPage: 1,
      pageSize: 20,

      setSidebarOpen: (open) => set({ sidebarOpen: open }),

      toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),

      setActiveFilters: (filters) => set({ activeFilters: filters }),

      clearFilters: () => set({ activeFilters: {}, currentPage: 1 }),

      setCurrentPage: (page) => set({ currentPage: page }),

      setPageSize: (size) => set({ pageSize: size }),
    }),
    {
      name: 'ui-storage',
    },
  ),
)
