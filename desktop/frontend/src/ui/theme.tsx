import React from 'react'
import { CssBaseline, ThemeProvider, createTheme } from '@mui/material'
import { settingsStore } from '../settings/store'
import { defaultUiPrefs, type ThemeMode, type UiPrefs } from '../settings/types'
import type { SettingsStore } from '../settings/types'
import { getPalettePreset, palettePresets } from './palettePresets'

interface ThemePreferencesContextValue {
  prefs: UiPrefs
  resolvedMode: 'light' | 'dark'
  setThemeMode: (mode: ThemeMode) => Promise<void>
  setPalettePreset: (preset: string) => Promise<void>
  setNavCollapsed: (collapsed: boolean) => Promise<void>
  setPinActionsColumnRight: (enabled: boolean) => Promise<void>
  setDataGridBorderRadius: (radius: number) => Promise<void>
  presets: typeof palettePresets
}

const ThemePreferencesContext = React.createContext<ThemePreferencesContextValue | null>(null)

function getSystemMode() {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return 'light' as const
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function useSystemColorMode() {
  const [mode, setMode] = React.useState<'light' | 'dark'>(() => getSystemMode())

  React.useEffect(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
      return
    }

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = (event: MediaQueryListEvent) => {
      setMode(event.matches ? 'dark' : 'light')
    }

    setMode(mediaQuery.matches ? 'dark' : 'light')
    mediaQuery.addEventListener('change', onChange)
    return () => mediaQuery.removeEventListener('change', onChange)
  }, [])

  return mode
}

function createAppTheme(mode: 'light' | 'dark', prefs: UiPrefs) {
  const preset = getPalettePreset(prefs.palettePreset)

  return createTheme({
    palette: {
      mode,
      primary: {
        main: preset.primary,
      },
      secondary: {
        main: preset.secondary,
      },
      background: {
        default: mode === 'light' ? preset.lightBackground : preset.darkBackground,
        paper: mode === 'light' ? preset.lightPaper : preset.darkPaper,
      },
    },
    shape: {
      borderRadius: 12,
    },
    typography: {
      fontFamily: 'Nunito, "Segoe UI", sans-serif',
      h5: {
        fontWeight: 700,
      },
      h6: {
        fontWeight: 700,
      },
    },
    components: {
      MuiCard: {
        styleOverrides: {
          root: {
            borderRadius: 16,
          },
        },
      },
      MuiDrawer: {
        styleOverrides: {
          paper: {
            borderRight: 0,
          },
        },
      },
      MuiAppBar: {
        styleOverrides: {
          root: {
            boxShadow: 'none',
            borderBottom: '1px solid',
            borderColor: mode === 'light' ? 'rgba(15, 23, 42, 0.08)' : 'rgba(148, 163, 184, 0.2)',
          },
        },
      },
    },
  })
}

export function AppThemeProvider({
  children,
  store = settingsStore,
}: {
  children: React.ReactNode
  store?: SettingsStore
}) {
  const [prefs, setPrefs] = React.useState<UiPrefs>(defaultUiPrefs)
  const systemMode = useSystemColorMode()

  React.useEffect(() => {
    let active = true
    store.loadSettings().then((settings) => {
      if (!active) {
        return
      }
      setPrefs(settings.uiPrefs)
    })
    return () => {
      active = false
    }
  }, [store])

  const persistPrefs = React.useCallback(async (nextPrefs: UiPrefs) => {
    setPrefs(nextPrefs)
    const saved = await store.saveSettings({ uiPrefs: nextPrefs })
    setPrefs(saved.uiPrefs)
  }, [store])

  const setThemeMode = React.useCallback(
    async (mode: ThemeMode) => {
      await persistPrefs({ ...prefs, themeMode: mode })
    },
    [persistPrefs, prefs],
  )

  const setPalettePreset = React.useCallback(
    async (preset: string) => {
      await persistPrefs({ ...prefs, palettePreset: preset })
    },
    [persistPrefs, prefs],
  )

  const setNavCollapsed = React.useCallback(
    async (collapsed: boolean) => {
      await persistPrefs({ ...prefs, navCollapsed: collapsed })
    },
    [persistPrefs, prefs],
  )

  const setPinActionsColumnRight = React.useCallback(
    async (enabled: boolean) => {
      await persistPrefs({ ...prefs, pinActionsColumnRight: enabled })
    },
    [persistPrefs, prefs],
  )

  const setDataGridBorderRadius = React.useCallback(
    async (radius: number) => {
      await persistPrefs({ ...prefs, dataGridBorderRadius: radius })
    },
    [persistPrefs, prefs],
  )

  const resolvedMode = prefs.themeMode === 'system' ? systemMode : prefs.themeMode

  const theme = React.useMemo(() => createAppTheme(resolvedMode, prefs), [resolvedMode, prefs])

  React.useEffect(() => {
    document.documentElement.setAttribute('data-theme-mode', resolvedMode)
    document.documentElement.setAttribute('data-theme-pref', prefs.themeMode)
    document.documentElement.setAttribute('data-palette-preset', prefs.palettePreset)
  }, [resolvedMode, prefs.palettePreset, prefs.themeMode])

  const contextValue = React.useMemo(
    () => ({
      prefs,
      resolvedMode,
      setThemeMode,
      setPalettePreset,
      setNavCollapsed,
      setPinActionsColumnRight,
      setDataGridBorderRadius,
      presets: palettePresets,
    }),
    [
      prefs,
      resolvedMode,
      setThemeMode,
      setPalettePreset,
      setNavCollapsed,
      setPinActionsColumnRight,
      setDataGridBorderRadius,
    ],
  )

  return (
    <ThemePreferencesContext.Provider value={contextValue}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        {children}
      </ThemeProvider>
    </ThemePreferencesContext.Provider>
  )
}

export function useThemePreferences() {
  const context = React.useContext(ThemePreferencesContext)
  if (!context) {
    throw new Error('useThemePreferences must be used within AppThemeProvider')
  }
  return context
}
