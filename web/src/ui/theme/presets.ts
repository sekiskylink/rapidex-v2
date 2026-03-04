import type { PaletteMode, PaletteOptions } from '@mui/material'

export interface PalettePreset {
  id: string
  name: string
  palettes: {
    light: PaletteOptions
    dark: PaletteOptions
  }
}

export const palettePresets: PalettePreset[] = [
  {
    id: 'oceanic',
    name: 'Oceanic',
    palettes: {
      light: {
        primary: { main: '#0F4C81' },
        secondary: { main: '#2A9D8F' },
        background: { default: '#F4F8FC', paper: '#FFFFFF' },
      },
      dark: {
        primary: { main: '#4EA8DE' },
        secondary: { main: '#52B69A' },
        background: { default: '#0D1B2A', paper: '#14263A' },
      },
    },
  },
  {
    id: 'ember',
    name: 'Ember',
    palettes: {
      light: {
        primary: { main: '#B23A48' },
        secondary: { main: '#F4A261' },
        background: { default: '#FFF7F5', paper: '#FFFFFF' },
      },
      dark: {
        primary: { main: '#E76F51' },
        secondary: { main: '#F6BD60' },
        background: { default: '#24100F', paper: '#341816' },
      },
    },
  },
  {
    id: 'forest',
    name: 'Forest',
    palettes: {
      light: {
        primary: { main: '#2D6A4F' },
        secondary: { main: '#40916C' },
        background: { default: '#F2F7F3', paper: '#FFFFFF' },
      },
      dark: {
        primary: { main: '#74C69D' },
        secondary: { main: '#95D5B2' },
        background: { default: '#10231A', paper: '#163126' },
      },
    },
  },
  {
    id: 'sunset',
    name: 'Sunset',
    palettes: {
      light: {
        primary: { main: '#9C6644' },
        secondary: { main: '#E09F3E' },
        background: { default: '#FFF8EF', paper: '#FFFFFF' },
      },
      dark: {
        primary: { main: '#D4A373' },
        secondary: { main: '#F4D58D' },
        background: { default: '#23170F', paper: '#322117' },
      },
    },
  },
  {
    id: 'slate',
    name: 'Slate',
    palettes: {
      light: {
        primary: { main: '#334155' },
        secondary: { main: '#64748B' },
        background: { default: '#F5F7FA', paper: '#FFFFFF' },
      },
      dark: {
        primary: { main: '#94A3B8' },
        secondary: { main: '#CBD5E1' },
        background: { default: '#0F172A', paper: '#1E293B' },
      },
    },
  },
  {
    id: 'orchard',
    name: 'Orchard',
    palettes: {
      light: {
        primary: { main: '#6C584C' },
        secondary: { main: '#A98467' },
        background: { default: '#F8F5F1', paper: '#FFFFFF' },
      },
      dark: {
        primary: { main: '#B08968' },
        secondary: { main: '#DDB892' },
        background: { default: '#1D1712', paper: '#2A201A' },
      },
    },
  },
  {
    id: 'graphite',
    name: 'Graphite',
    palettes: {
      light: {
        primary: { main: '#1F2937' },
        secondary: { main: '#4B5563' },
        background: { default: '#F9FAFB', paper: '#FFFFFF' },
      },
      dark: {
        primary: { main: '#9CA3AF' },
        secondary: { main: '#D1D5DB' },
        background: { default: '#111827', paper: '#1F2937' },
      },
    },
  },
  {
    id: 'lagoon',
    name: 'Lagoon',
    palettes: {
      light: {
        primary: { main: '#15616D' },
        secondary: { main: '#00A6A6' },
        background: { default: '#EFF8F9', paper: '#FFFFFF' },
      },
      dark: {
        primary: { main: '#5EC2C5' },
        secondary: { main: '#84DCCF' },
        background: { default: '#0E2124', paper: '#173035' },
      },
    },
  },
]

export const defaultPresetId = palettePresets[0].id

export function getPresetById(presetId: string): PalettePreset {
  return palettePresets.find((preset) => preset.id === presetId) ?? palettePresets[0]
}

export function getPaletteOptions(presetId: string, mode: PaletteMode): PaletteOptions {
  const preset = getPresetById(presetId)
  const paletteByMode = mode === 'dark' ? preset.palettes.dark : preset.palettes.light

  return {
    mode,
    ...paletteByMode,
  }
}
