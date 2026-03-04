import React from 'react'
import { CssBaseline, ThemeProvider, createTheme } from '@mui/material'
import { UiPreferencesProvider, useUiPreferences } from './UiPreferencesProvider'
import { getPaletteOptions } from './presets'

export function buildAppTheme(mode: 'light' | 'dark', preset: string) {
  return createTheme({
    palette: getPaletteOptions(preset, mode),
    shape: {
      borderRadius: 10,
    },
    components: {
      MuiAppBar: {
        styleOverrides: {
          root: {
            backdropFilter: 'blur(8px)',
          },
        },
      },
      MuiDrawer: {
        styleOverrides: {
          paper: {
            borderRightWidth: 0,
          },
        },
      },
    },
  })
}

function ThemedAppContent({ children }: React.PropsWithChildren) {
  const { prefs, resolvedMode } = useUiPreferences()

  const theme = React.useMemo(() => buildAppTheme(resolvedMode, prefs.preset), [prefs.preset, resolvedMode])

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      {children}
    </ThemeProvider>
  )
}

export function AppThemeProvider({ children }: React.PropsWithChildren) {
  return (
    <UiPreferencesProvider>
      <ThemedAppContent>{children}</ThemedAppContent>
    </UiPreferencesProvider>
  )
}
