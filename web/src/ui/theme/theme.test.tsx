import React from 'react'
import { Button, Typography, useTheme } from '@mui/material'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  UI_PREFERENCES_STORAGE_KEY,
  loadPrefs,
  savePrefs,
  type UiThemeMode,
} from '../preferences'
import { AppThemeProvider } from './AppThemeProvider'
import { getPaletteOptions } from './presets'
import { useUiPreferences } from './UiPreferencesProvider'

function primaryMainFor(preset: string, mode: 'light' | 'dark') {
  const palette = getPaletteOptions(preset, mode)
  const primary = palette.primary
  if (primary && typeof primary === 'object' && 'main' in primary && typeof primary.main === 'string') {
    return primary.main
  }
  return ''
}

function mockMatchMedia(prefersDark: boolean) {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: query === '(prefers-color-scheme: dark)' ? prefersDark : false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
}

function ThemeProbe() {
  const theme = useTheme()
  const { prefs, resolvedMode, setMode, setPreset } = useUiPreferences()

  return (
    <>
      <Typography data-testid="theme-mode">{theme.palette.mode}</Typography>
      <Typography data-testid="resolved-mode">{resolvedMode}</Typography>
      <Typography data-testid="theme-primary">{theme.palette.primary.main}</Typography>
      <Typography data-testid="pref-mode">{prefs.mode}</Typography>
      <Typography data-testid="pref-preset">{prefs.preset}</Typography>
      <Button onClick={() => setMode('light')}>Mode Light</Button>
      <Button onClick={() => setMode('dark')}>Mode Dark</Button>
      <Button onClick={() => setMode('system')}>Mode System</Button>
      <Button onClick={() => setPreset('forest')}>Preset Forest</Button>
    </>
  )
}

function renderThemeProbe() {
  return render(
    <AppThemeProvider>
      <ThemeProbe />
    </AppThemeProvider>,
  )
}

function expectStoredPrefs(mode: UiThemeMode, preset: string) {
  const raw = window.localStorage.getItem(UI_PREFERENCES_STORAGE_KEY)
  expect(raw).toBeTruthy()
  expect(JSON.parse(raw ?? '{}')).toEqual(
    expect.objectContaining({
      mode,
      preset,
    }),
  )
}

beforeEach(() => {
  window.localStorage.clear()
  mockMatchMedia(false)
})

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('theme system persistence', () => {
  it('mode persists and applies after reload', () => {
    const firstRender = renderThemeProbe()

    fireEvent.click(screen.getByRole('button', { name: 'Mode Dark' }))

    expect(screen.getByTestId('theme-mode').textContent).toBe('dark')
    expectStoredPrefs('dark', 'oceanic')

    firstRender.unmount()
    renderThemeProbe()

    expect(screen.getByTestId('pref-mode').textContent).toBe('dark')
    expect(screen.getByTestId('theme-mode').textContent).toBe('dark')
  })

  it('preset persists and applies after reload', () => {
    const firstRender = renderThemeProbe()

    fireEvent.click(screen.getByRole('button', { name: 'Mode Light' }))
    fireEvent.click(screen.getByRole('button', { name: 'Preset Forest' }))

    const expectedPrimary = primaryMainFor('forest', 'light')
    expect(screen.getByTestId('theme-primary').textContent).toBe(expectedPrimary)
    expectStoredPrefs('light', 'forest')

    firstRender.unmount()
    renderThemeProbe()

    expect(screen.getByTestId('pref-preset').textContent).toBe('forest')
    expect(screen.getByTestId('theme-primary').textContent).toBe(expectedPrimary)
  })

  it('system mode deterministically follows mocked matchMedia', () => {
    savePrefs({
      mode: 'system',
      preset: 'oceanic',
      collapseNavByDefault: false,
      showFooter: true,
      pinActionsColumnRight: true,
      dataGridBorderRadius: 12,
    })

    mockMatchMedia(true)
    const firstRender = renderThemeProbe()

    expect(screen.getByTestId('pref-mode').textContent).toBe('system')
    expect(screen.getByTestId('resolved-mode').textContent).toBe('dark')
    expect(screen.getByTestId('theme-mode').textContent).toBe('dark')

    firstRender.unmount()

    mockMatchMedia(false)
    renderThemeProbe()

    expect(loadPrefs().mode).toBe('system')
    expect(screen.getByTestId('resolved-mode').textContent).toBe('light')
    expect(screen.getByTestId('theme-mode').textContent).toBe('light')
  })
})
