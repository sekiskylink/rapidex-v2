export interface PalettePreset {
  id: string
  label: string
  primary: string
  secondary: string
  lightBackground: string
  lightPaper: string
  darkBackground: string
  darkPaper: string
}

export const customPresetId = 'custom'
export const defaultCustomAccent = '#0b7285'

export const palettePresets: PalettePreset[] = [
  {
    id: 'ocean',
    label: 'Ocean',
    primary: '#0B7285',
    secondary: '#0CA678',
    lightBackground: '#F3F9FA',
    lightPaper: '#FFFFFF',
    darkBackground: '#0F1722',
    darkPaper: '#141E2C',
  },
  {
    id: 'ember',
    label: 'Ember',
    primary: '#C44536',
    secondary: '#F39C12',
    lightBackground: '#FFF7F3',
    lightPaper: '#FFFFFF',
    darkBackground: '#1C1211',
    darkPaper: '#271916',
  },
  {
    id: 'forest',
    label: 'Forest',
    primary: '#2B8A3E',
    secondary: '#5C940D',
    lightBackground: '#F4FAF4',
    lightPaper: '#FFFFFF',
    darkBackground: '#111A13',
    darkPaper: '#18241B',
  },
  {
    id: 'graphite',
    label: 'Graphite',
    primary: '#364FC7',
    secondary: '#495057',
    lightBackground: '#F5F7FA',
    lightPaper: '#FFFFFF',
    darkBackground: '#11141B',
    darkPaper: '#171C27',
  },
  {
    id: 'ruby',
    label: 'Ruby',
    primary: '#A61E4D',
    secondary: '#D6336C',
    lightBackground: '#FFF4F8',
    lightPaper: '#FFFFFF',
    darkBackground: '#1E1118',
    darkPaper: '#2A1620',
  },
  {
    id: 'sunset',
    label: 'Sunset',
    primary: '#E8590C',
    secondary: '#FAB005',
    lightBackground: '#FFF8F2',
    lightPaper: '#FFFFFF',
    darkBackground: '#1D140F',
    darkPaper: '#281C14',
  },
  {
    id: 'berry',
    label: 'Berry',
    primary: '#7B2CBF',
    secondary: '#9D4EDD',
    lightBackground: '#FAF5FF',
    lightPaper: '#FFFFFF',
    darkBackground: '#190F24',
    darkPaper: '#231534',
  },
  {
    id: 'slate',
    label: 'Slate',
    primary: '#1D4ED8',
    secondary: '#0891B2',
    lightBackground: '#F3F6FB',
    lightPaper: '#FFFFFF',
    darkBackground: '#0E1621',
    darkPaper: '#15202E',
  },
  {
    id: 'orchard',
    label: 'Orchard',
    primary: '#6C584C',
    secondary: '#A98467',
    lightBackground: '#F8F5F1',
    lightPaper: '#FFFFFF',
    darkBackground: '#1D1712',
    darkPaper: '#2A201A',
  },
  {
    id: 'lagoon',
    label: 'Lagoon',
    primary: '#15616D',
    secondary: '#00A6A6',
    lightBackground: '#EFF8F9',
    lightPaper: '#FFFFFF',
    darkBackground: '#0E2124',
    darkPaper: '#173035',
  },
  {
    id: 'aurora',
    label: 'Aurora',
    primary: '#6D28D9',
    secondary: '#0891B2',
    lightBackground: '#F7F4FF',
    lightPaper: '#FFFFFF',
    darkBackground: '#160F2B',
    darkPaper: '#20163A',
  },
  {
    id: 'mint',
    label: 'Mint',
    primary: '#0F766E',
    secondary: '#14B8A6',
    lightBackground: '#F0FDFA',
    lightPaper: '#FFFFFF',
    darkBackground: '#0D201F',
    darkPaper: '#14302E',
  },
]

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

export function getPalettePreset(id: string) {
  return palettePresets.find((preset) => preset.id === id) ?? palettePresets[0]
}

export function getCustomPalettePreset(accent: string) {
  const normalizedAccent = normalizeHexColor(accent) ?? defaultCustomAccent
  return {
    id: customPresetId,
    label: 'Custom',
    primary: normalizedAccent,
    secondary: blendHexColors(normalizedAccent, '#1F2937', 0.22),
    lightBackground: blendHexColors(normalizedAccent, '#F8FAFC', 0.92),
    lightPaper: '#FFFFFF',
    darkBackground: blendHexColors(normalizedAccent, '#0F172A', 0.82),
    darkPaper: blendHexColors(normalizedAccent, '#111827', 0.72),
  }
}
