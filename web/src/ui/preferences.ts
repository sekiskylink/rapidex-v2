export type UiThemeMode = 'light' | 'dark' | 'system'

export interface UiPreferences {
  mode: UiThemeMode
  preset: string
  collapseNavByDefault: boolean
  showFooter: boolean
  pinActionsColumnRight: boolean
  dataGridBorderRadius: number
}

export const UI_PREFERENCES_STORAGE_KEY = 'basepro.web.ui_preferences'

const DEFAULT_PREFERENCES: UiPreferences = {
  mode: 'system',
  preset: 'oceanic',
  collapseNavByDefault: false,
  showFooter: true,
  pinActionsColumnRight: true,
  dataGridBorderRadius: 12,
}

function isValidMode(value: unknown): value is UiThemeMode {
  return value === 'light' || value === 'dark' || value === 'system'
}

function sanitizeBorderRadius(value: unknown) {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return DEFAULT_PREFERENCES.dataGridBorderRadius
  }
  const rounded = Math.round(value)
  if (rounded < 4 || rounded > 32) {
    return DEFAULT_PREFERENCES.dataGridBorderRadius
  }
  return rounded
}

export function getDefaultPreferences(): UiPreferences {
  return { ...DEFAULT_PREFERENCES }
}

export function loadPrefs(): UiPreferences {
  if (typeof window === 'undefined') {
    return getDefaultPreferences()
  }

  const raw = window.localStorage.getItem(UI_PREFERENCES_STORAGE_KEY)
  if (!raw) {
    return getDefaultPreferences()
  }

  try {
    const parsed = JSON.parse(raw) as Partial<UiPreferences>
    return {
      mode: isValidMode(parsed.mode) ? parsed.mode : DEFAULT_PREFERENCES.mode,
      preset: typeof parsed.preset === 'string' && parsed.preset.trim() ? parsed.preset : DEFAULT_PREFERENCES.preset,
      collapseNavByDefault:
        typeof parsed.collapseNavByDefault === 'boolean'
          ? parsed.collapseNavByDefault
          : DEFAULT_PREFERENCES.collapseNavByDefault,
      showFooter: typeof parsed.showFooter === 'boolean' ? parsed.showFooter : DEFAULT_PREFERENCES.showFooter,
      pinActionsColumnRight:
        typeof parsed.pinActionsColumnRight === 'boolean'
          ? parsed.pinActionsColumnRight
          : DEFAULT_PREFERENCES.pinActionsColumnRight,
      dataGridBorderRadius: sanitizeBorderRadius(parsed.dataGridBorderRadius),
    }
  } catch {
    return getDefaultPreferences()
  }
}

export function savePrefs(next: UiPreferences) {
  if (typeof window === 'undefined') {
    return
  }

  window.localStorage.setItem(UI_PREFERENCES_STORAGE_KEY, JSON.stringify(next))
}

export function setMode(mode: UiThemeMode) {
  const next = {
    ...loadPrefs(),
    mode,
  }
  savePrefs(next)
  return next
}

export function setPreset(preset: string) {
  const sanitized = preset.trim()
  const next = {
    ...loadPrefs(),
    preset: sanitized || DEFAULT_PREFERENCES.preset,
  }
  savePrefs(next)
  return next
}

export function setCollapseNavByDefault(collapseNavByDefault: boolean) {
  const next = {
    ...loadPrefs(),
    collapseNavByDefault,
  }
  savePrefs(next)
  return next
}

export function setShowFooter(showFooter: boolean) {
  const next = {
    ...loadPrefs(),
    showFooter,
  }
  savePrefs(next)
  return next
}

export function setPinActionsColumnRight(pinActionsColumnRight: boolean) {
  const next = {
    ...loadPrefs(),
    pinActionsColumnRight,
  }
  savePrefs(next)
  return next
}

export function setDataGridBorderRadius(dataGridBorderRadius: number) {
  const next = {
    ...loadPrefs(),
    dataGridBorderRadius: sanitizeBorderRadius(dataGridBorderRadius),
  }
  savePrefs(next)
  return next
}
