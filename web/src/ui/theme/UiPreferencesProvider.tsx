import React from 'react'
import { useMediaQuery } from '@mui/material'
import {
  loadPrefs,
  savePrefs,
  setMode as persistMode,
  setPreset as persistPreset,
  setCollapseNavByDefault as persistCollapseNavByDefault,
  type UiPreferences,
  type UiThemeMode,
} from '../preferences'
import { defaultPresetId, getPresetById } from './presets'

interface UiPreferencesContextValue {
  prefs: UiPreferences
  resolvedMode: 'light' | 'dark'
  setMode: (mode: UiThemeMode) => void
  setPreset: (preset: string) => void
  setCollapseNavByDefault: (collapseNavByDefault: boolean) => void
}

const UiPreferencesContext = React.createContext<UiPreferencesContextValue | undefined>(undefined)

function sanitizePreset(preset: string) {
  return getPresetById(preset).id || defaultPresetId
}

export function UiPreferencesProvider({ children }: React.PropsWithChildren) {
  const [prefs, setPrefs] = React.useState<UiPreferences>(() => {
    const loaded = loadPrefs()
    return {
      mode: loaded.mode,
      preset: sanitizePreset(loaded.preset),
      collapseNavByDefault: loaded.collapseNavByDefault,
    }
  })

  const prefersDark = useMediaQuery('(prefers-color-scheme: dark)', { noSsr: true })

  const resolvedMode = prefs.mode === 'system' ? (prefersDark ? 'dark' : 'light') : prefs.mode

  const setMode = React.useCallback((mode: UiThemeMode) => {
    const next = persistMode(mode)
    setPrefs({
      mode: next.mode,
      preset: sanitizePreset(next.preset),
      collapseNavByDefault: next.collapseNavByDefault,
    })
  }, [])

  const setPreset = React.useCallback((preset: string) => {
    const next = persistPreset(preset)
    setPrefs({
      mode: next.mode,
      preset: sanitizePreset(next.preset),
      collapseNavByDefault: next.collapseNavByDefault,
    })
  }, [])

  const setCollapseNavByDefault = React.useCallback((collapseNavByDefault: boolean) => {
    const next = persistCollapseNavByDefault(collapseNavByDefault)
    setPrefs({
      mode: next.mode,
      preset: sanitizePreset(next.preset),
      collapseNavByDefault: next.collapseNavByDefault,
    })
  }, [])

  React.useEffect(() => {
    savePrefs(prefs)
  }, [prefs])

  const value = React.useMemo<UiPreferencesContextValue>(
    () => ({
      prefs,
      resolvedMode,
      setMode,
      setPreset,
      setCollapseNavByDefault,
    }),
    [prefs, resolvedMode, setMode, setPreset, setCollapseNavByDefault],
  )

  return <UiPreferencesContext.Provider value={value}>{children}</UiPreferencesContext.Provider>
}

export function useUiPreferences() {
  const context = React.useContext(UiPreferencesContext)
  if (!context) {
    throw new Error('useUiPreferences must be used inside UiPreferencesProvider')
  }
  return context
}
