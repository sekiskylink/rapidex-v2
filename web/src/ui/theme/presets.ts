import type { PaletteMode, PaletteOptions } from '@mui/material'

export interface PalettePreset {
  id: string
  name: string
  palettes: {
    light: PaletteOptions
    dark: PaletteOptions
  }
}

export const customPresetId = 'custom'
export const defaultCustomAccent = '#0F4C81'

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
  {
    id: 'aurora',
    name: 'Aurora',
    palettes: {
      light: {
        primary: { main: '#6D28D9' },
        secondary: { main: '#0891B2' },
        background: { default: '#F7F4FF', paper: '#FFFFFF' },
      },
      dark: {
        primary: { main: '#A78BFA' },
        secondary: { main: '#67E8F9' },
        background: { default: '#160F2B', paper: '#20163A' },
      },
    },
  },
  {
    id: 'copper',
    name: 'Copper',
    palettes: {
      light: {
        primary: { main: '#B45309' },
        secondary: { main: '#D97706' },
        background: { default: '#FFF8F1', paper: '#FFFFFF' },
      },
      dark: {
        primary: { main: '#F59E0B' },
        secondary: { main: '#FBBF24' },
        background: { default: '#22140A', paper: '#301C0D' },
      },
    },
  },
  {
    id: 'mint',
    name: 'Mint',
    palettes: {
      light: {
        primary: { main: '#0F766E' },
        secondary: { main: '#14B8A6' },
        background: { default: '#F0FDFA', paper: '#FFFFFF' },
      },
      dark: {
        primary: { main: '#5EEAD4' },
        secondary: { main: '#99F6E4' },
        background: { default: '#0D201F', paper: '#14302E' },
      },
    },
  },
  {
    id: 'rosewood',
    name: 'Rosewood',
    palettes: {
      light: {
        primary: { main: '#9F1239' },
        secondary: { main: '#E11D48' },
        background: { default: '#FFF4F7', paper: '#FFFFFF' },
      },
      dark: {
        primary: { main: '#FB7185' },
        secondary: { main: '#FDA4AF' },
        background: { default: '#260F18', paper: '#351520' },
      },
    },
  },
]

export const defaultPresetId = palettePresets[0].id

function clampRgb(value: number) {
  return Math.min(255, Math.max(0, Math.round(value)))
}

function expandHex(input: string) {
  if (input.length === 4) {
    return `#${input[1]}${input[1]}${input[2]}${input[2]}${input[3]}${input[3]}`
  }
  return input
}

function hexToRgb(hex: string) {
  const normalized = expandHex(hex)
  const match = /^#([0-9a-f]{6})$/i.exec(normalized)
  if (!match) {
    return null
  }
  const value = match[1]
  return {
    r: Number.parseInt(value.slice(0, 2), 16),
    g: Number.parseInt(value.slice(2, 4), 16),
    b: Number.parseInt(value.slice(4, 6), 16),
  }
}

function rgbToHex(r: number, g: number, b: number) {
  return `#${[r, g, b]
    .map((value) => clampRgb(value).toString(16).padStart(2, '0'))
    .join('')}`
}

function blendHexColors(base: string, target: string, weight: number) {
  const baseRgb = hexToRgb(base)
  const targetRgb = hexToRgb(target)
  if (!baseRgb || !targetRgb) {
    return base
  }
  const clampedWeight = Math.min(1, Math.max(0, weight))
  return rgbToHex(
    baseRgb.r + (targetRgb.r - baseRgb.r) * clampedWeight,
    baseRgb.g + (targetRgb.g - baseRgb.g) * clampedWeight,
    baseRgb.b + (targetRgb.b - baseRgb.b) * clampedWeight,
  )
}

export function normalizeHexColor(value: string | undefined | null) {
  const trimmed = (value ?? '').trim()
  if (!trimmed) {
    return undefined
  }
  const normalized = trimmed.startsWith('#') ? trimmed : `#${trimmed}`
  const expanded = expandHex(normalized)
  return /^#([0-9a-f]{6})$/i.test(expanded) ? expanded.toLowerCase() : undefined
}

export function getPresetById(presetId: string): PalettePreset {
  return palettePresets.find((preset) => preset.id === presetId) ?? palettePresets[0]
}

export function isCustomPreset(presetId: string) {
  return presetId === customPresetId
}

export function getCustomPaletteOptions(accent: string, mode: PaletteMode): PaletteOptions {
  const normalizedAccent = normalizeHexColor(accent) ?? defaultCustomAccent
  if (mode === 'dark') {
    return {
      mode,
      primary: { main: normalizedAccent },
      secondary: { main: blendHexColors(normalizedAccent, '#E2E8F0', 0.32) },
      background: {
        default: blendHexColors(normalizedAccent, '#0F172A', 0.82),
        paper: blendHexColors(normalizedAccent, '#111827', 0.72),
      },
    }
  }

  return {
    mode,
    primary: { main: normalizedAccent },
    secondary: { main: blendHexColors(normalizedAccent, '#1F2937', 0.22) },
    background: {
      default: blendHexColors(normalizedAccent, '#F8FAFC', 0.92),
      paper: '#FFFFFF',
    },
  }
}

export function getPaletteOptions(presetId: string, mode: PaletteMode, customAccent?: string): PaletteOptions {
  if (isCustomPreset(presetId)) {
    return getCustomPaletteOptions(customAccent ?? defaultCustomAccent, mode)
  }
  const preset = getPresetById(presetId)
  const paletteByMode = mode === 'dark' ? preset.palettes.dark : preset.palettes.light

  return {
    mode,
    ...paletteByMode,
  }
}
