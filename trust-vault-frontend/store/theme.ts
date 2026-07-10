import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type ThemeMode = 'light' | 'dark' | 'system'

interface ThemeState {
  mode: ThemeMode
  resolved: 'light' | 'dark'

  // Actions
  setMode: (mode: ThemeMode) => void
  setResolved: (resolved: 'light' | 'dark') => void
  toggle: () => void
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      mode: 'system',
      resolved: 'dark',

      setMode: (mode) => {
        set({ mode })
        applyTheme(mode)
      },

      setResolved: (resolved) => set({ resolved }),

      toggle: () => {
        const { resolved } = get()
        const newResolved = resolved === 'light' ? 'dark' : 'light'
        set({ resolved: newResolved, mode: newResolved })
        applyTheme(newResolved)
      },
    }),
    {
      name: 'theme-storage',
      partialize: (state) => ({ mode: state.mode }),
    },
  ),
)

export const applyTheme = (mode: ThemeMode) => {
  const htmlEl = document.documentElement
  const isDark = getSystemTheme() === 'dark'
  const resolved = mode === 'system' ? (isDark ? 'dark' : 'light') : mode

  useThemeStore.setState({ resolved })

  if (resolved === 'dark') {
    htmlEl.classList.add('dark')
  } else {
    htmlEl.classList.remove('dark')
  }
}

export const getSystemTheme = (): 'light' | 'dark' => {
  if (typeof window === 'undefined') return 'dark'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

// Monitor system theme changes
if (typeof window !== 'undefined') {
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
    const { mode } = useThemeStore.getState()
    if (mode === 'system') {
      applyTheme('system')
    }
  })
}
