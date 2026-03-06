import React from 'react'
import { useMediaQuery } from '@mui/material'
import {
  loadPrefs,
  savePrefs,
  setMode as persistMode,
  setPreset as persistPreset,
  setCollapseNavByDefault as persistCollapseNavByDefault,
  setShowFooter as persistShowFooter,
  setPinActionsColumnRight as persistPinActionsColumnRight,
  setDataGridBorderRadius as persistDataGridBorderRadius,
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
  setShowFooter: (showFooter: boolean) => void
  setPinActionsColumnRight: (pinActionsColumnRight: boolean) => void
  setDataGridBorderRadius: (dataGridBorderRadius: number) => void
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
      showFooter: loaded.showFooter,
      pinActionsColumnRight: loaded.pinActionsColumnRight,
      dataGridBorderRadius: loaded.dataGridBorderRadius,
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
      showFooter: next.showFooter,
      pinActionsColumnRight: next.pinActionsColumnRight,
      dataGridBorderRadius: next.dataGridBorderRadius,
    })
  }, [])

  const setPreset = React.useCallback((preset: string) => {
    const next = persistPreset(preset)
    setPrefs({
      mode: next.mode,
      preset: sanitizePreset(next.preset),
      collapseNavByDefault: next.collapseNavByDefault,
      showFooter: next.showFooter,
      pinActionsColumnRight: next.pinActionsColumnRight,
      dataGridBorderRadius: next.dataGridBorderRadius,
    })
  }, [])

  const setCollapseNavByDefault = React.useCallback((collapseNavByDefault: boolean) => {
    const next = persistCollapseNavByDefault(collapseNavByDefault)
    setPrefs({
      mode: next.mode,
      preset: sanitizePreset(next.preset),
      collapseNavByDefault: next.collapseNavByDefault,
      showFooter: next.showFooter,
      pinActionsColumnRight: next.pinActionsColumnRight,
      dataGridBorderRadius: next.dataGridBorderRadius,
    })
  }, [])

  const setShowFooter = React.useCallback((showFooter: boolean) => {
    const next = persistShowFooter(showFooter)
    setPrefs({
      mode: next.mode,
      preset: sanitizePreset(next.preset),
      collapseNavByDefault: next.collapseNavByDefault,
      showFooter: next.showFooter,
      pinActionsColumnRight: next.pinActionsColumnRight,
      dataGridBorderRadius: next.dataGridBorderRadius,
    })
  }, [])

  const setPinActionsColumnRight = React.useCallback((pinActionsColumnRight: boolean) => {
    const next = persistPinActionsColumnRight(pinActionsColumnRight)
    setPrefs({
      mode: next.mode,
      preset: sanitizePreset(next.preset),
      collapseNavByDefault: next.collapseNavByDefault,
      showFooter: next.showFooter,
      pinActionsColumnRight: next.pinActionsColumnRight,
      dataGridBorderRadius: next.dataGridBorderRadius,
    })
  }, [])

  const setDataGridBorderRadius = React.useCallback((dataGridBorderRadius: number) => {
    const next = persistDataGridBorderRadius(dataGridBorderRadius)
    setPrefs({
      mode: next.mode,
      preset: sanitizePreset(next.preset),
      collapseNavByDefault: next.collapseNavByDefault,
      showFooter: next.showFooter,
      pinActionsColumnRight: next.pinActionsColumnRight,
      dataGridBorderRadius: next.dataGridBorderRadius,
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
      setShowFooter,
      setPinActionsColumnRight,
      setDataGridBorderRadius,
    }),
    [
      prefs,
      resolvedMode,
      setMode,
      setPreset,
      setCollapseNavByDefault,
      setShowFooter,
      setPinActionsColumnRight,
      setDataGridBorderRadius,
    ],
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
