import React from 'react'
import { useMediaQuery } from '@mui/material'
import {
  loadPrefs,
  savePrefs,
  setMode as persistMode,
  setPreset as persistPreset,
  setCollapseNavByDefault as persistCollapseNavByDefault,
  setShowFooter as persistShowFooter,
  setShowSukumadMenu as persistShowSukumadMenu,
  setShowAdministrationMenu as persistShowAdministrationMenu,
  setPinActionsColumnRight as persistPinActionsColumnRight,
  setDataGridBorderRadius as persistDataGridBorderRadius,
  setNavLabel as persistNavLabel,
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
  setShowSukumadMenu: (showSukumadMenu: boolean) => void
  setShowAdministrationMenu: (showAdministrationMenu: boolean) => void
  setPinActionsColumnRight: (pinActionsColumnRight: boolean) => void
  setDataGridBorderRadius: (dataGridBorderRadius: number) => void
  setNavLabel: (id: string, label: string) => void
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
      showSukumadMenu: loaded.showSukumadMenu,
      showAdministrationMenu: loaded.showAdministrationMenu,
      pinActionsColumnRight: loaded.pinActionsColumnRight,
      dataGridBorderRadius: loaded.dataGridBorderRadius,
      navLabels: loaded.navLabels,
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
      showSukumadMenu: next.showSukumadMenu,
      showAdministrationMenu: next.showAdministrationMenu,
      pinActionsColumnRight: next.pinActionsColumnRight,
      dataGridBorderRadius: next.dataGridBorderRadius,
      navLabels: next.navLabels,
    })
  }, [])

  const setPreset = React.useCallback((preset: string) => {
    const next = persistPreset(preset)
    setPrefs({
      mode: next.mode,
      preset: sanitizePreset(next.preset),
      collapseNavByDefault: next.collapseNavByDefault,
      showFooter: next.showFooter,
      showSukumadMenu: next.showSukumadMenu,
      showAdministrationMenu: next.showAdministrationMenu,
      pinActionsColumnRight: next.pinActionsColumnRight,
      dataGridBorderRadius: next.dataGridBorderRadius,
      navLabels: next.navLabels,
    })
  }, [])

  const setCollapseNavByDefault = React.useCallback((collapseNavByDefault: boolean) => {
    const next = persistCollapseNavByDefault(collapseNavByDefault)
    setPrefs({
      mode: next.mode,
      preset: sanitizePreset(next.preset),
      collapseNavByDefault: next.collapseNavByDefault,
      showFooter: next.showFooter,
      showSukumadMenu: next.showSukumadMenu,
      showAdministrationMenu: next.showAdministrationMenu,
      pinActionsColumnRight: next.pinActionsColumnRight,
      dataGridBorderRadius: next.dataGridBorderRadius,
      navLabels: next.navLabels,
    })
  }, [])

  const setShowFooter = React.useCallback((showFooter: boolean) => {
    const next = persistShowFooter(showFooter)
    setPrefs({
      mode: next.mode,
      preset: sanitizePreset(next.preset),
      collapseNavByDefault: next.collapseNavByDefault,
      showFooter: next.showFooter,
      showSukumadMenu: next.showSukumadMenu,
      showAdministrationMenu: next.showAdministrationMenu,
      pinActionsColumnRight: next.pinActionsColumnRight,
      dataGridBorderRadius: next.dataGridBorderRadius,
      navLabels: next.navLabels,
    })
  }, [])

  const setShowSukumadMenu = React.useCallback((showSukumadMenu: boolean) => {
    const next = persistShowSukumadMenu(showSukumadMenu)
    setPrefs({
      mode: next.mode,
      preset: sanitizePreset(next.preset),
      collapseNavByDefault: next.collapseNavByDefault,
      showFooter: next.showFooter,
      showSukumadMenu: next.showSukumadMenu,
      showAdministrationMenu: next.showAdministrationMenu,
      pinActionsColumnRight: next.pinActionsColumnRight,
      dataGridBorderRadius: next.dataGridBorderRadius,
      navLabels: next.navLabels,
    })
  }, [])

  const setShowAdministrationMenu = React.useCallback((showAdministrationMenu: boolean) => {
    const next = persistShowAdministrationMenu(showAdministrationMenu)
    setPrefs({
      mode: next.mode,
      preset: sanitizePreset(next.preset),
      collapseNavByDefault: next.collapseNavByDefault,
      showFooter: next.showFooter,
      showSukumadMenu: next.showSukumadMenu,
      showAdministrationMenu: next.showAdministrationMenu,
      pinActionsColumnRight: next.pinActionsColumnRight,
      dataGridBorderRadius: next.dataGridBorderRadius,
      navLabels: next.navLabels,
    })
  }, [])

  const setPinActionsColumnRight = React.useCallback((pinActionsColumnRight: boolean) => {
    const next = persistPinActionsColumnRight(pinActionsColumnRight)
    setPrefs({
      mode: next.mode,
      preset: sanitizePreset(next.preset),
      collapseNavByDefault: next.collapseNavByDefault,
      showFooter: next.showFooter,
      showSukumadMenu: next.showSukumadMenu,
      showAdministrationMenu: next.showAdministrationMenu,
      pinActionsColumnRight: next.pinActionsColumnRight,
      dataGridBorderRadius: next.dataGridBorderRadius,
      navLabels: next.navLabels,
    })
  }, [])

  const setDataGridBorderRadius = React.useCallback((dataGridBorderRadius: number) => {
    const next = persistDataGridBorderRadius(dataGridBorderRadius)
    setPrefs({
      mode: next.mode,
      preset: sanitizePreset(next.preset),
      collapseNavByDefault: next.collapseNavByDefault,
      showFooter: next.showFooter,
      showSukumadMenu: next.showSukumadMenu,
      showAdministrationMenu: next.showAdministrationMenu,
      pinActionsColumnRight: next.pinActionsColumnRight,
      dataGridBorderRadius: next.dataGridBorderRadius,
      navLabels: next.navLabels,
    })
  }, [])

  const setNavLabel = React.useCallback((id: string, label: string) => {
    const next = persistNavLabel(id, label)
    setPrefs({
      mode: next.mode,
      preset: sanitizePreset(next.preset),
      collapseNavByDefault: next.collapseNavByDefault,
      showFooter: next.showFooter,
      showSukumadMenu: next.showSukumadMenu,
      showAdministrationMenu: next.showAdministrationMenu,
      pinActionsColumnRight: next.pinActionsColumnRight,
      dataGridBorderRadius: next.dataGridBorderRadius,
      navLabels: next.navLabels,
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
      setShowSukumadMenu,
      setShowAdministrationMenu,
      setPinActionsColumnRight,
      setDataGridBorderRadius,
      setNavLabel,
    }),
    [
      prefs,
      resolvedMode,
      setMode,
      setPreset,
      setCollapseNavByDefault,
      setShowFooter,
      setShowSukumadMenu,
      setShowAdministrationMenu,
      setPinActionsColumnRight,
      setDataGridBorderRadius,
      setNavLabel,
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
