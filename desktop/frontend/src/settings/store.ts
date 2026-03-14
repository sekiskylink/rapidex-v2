import {
  AUTH_MODES,
  THEME_MODES,
  defaultSettings,
  defaultUiPrefs,
  type AppSettings,
  type AuthMode,
  type SaveSettingsPatch,
  type SettingsStore,
  type ThemeMode,
  type TablePrefsMap,
  type TablePrefsV1,
  type UiPrefs,
} from './types'

interface WailsAppBindings {
  LoadSettings: () => Promise<unknown>
  SaveSettings: (patch: unknown) => Promise<unknown>
  ResetSettings: () => Promise<unknown>
}

function getWailsBindings(): WailsAppBindings | null {
  if (typeof window === 'undefined' || typeof window.go === 'undefined') {
    return null
  }

  const app = window.go?.main?.App
  if (!app) {
    return null
  }

  if (
    typeof app.LoadSettings !== 'function' ||
    typeof app.SaveSettings !== 'function' ||
    typeof app.ResetSettings !== 'function'
  ) {
    return null
  }

  return {
    LoadSettings: app.LoadSettings,
    SaveSettings: app.SaveSettings,
    ResetSettings: app.ResetSettings,
  }
}

function isObjectRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isAuthMode(value: unknown): value is AuthMode {
  return typeof value === 'string' && AUTH_MODES.some((mode) => mode === value)
}

function isThemeMode(value: unknown): value is ThemeMode {
  return typeof value === 'string' && THEME_MODES.some((mode) => mode === value)
}

function readString(value: unknown, fallback = ''): string {
  return typeof value === 'string' ? value : fallback
}

function readBoolean(value: unknown, fallback = false): boolean {
  return typeof value === 'boolean' ? value : fallback
}

function readPositiveInteger(value: unknown, fallback: number): number {
  return typeof value === 'number' && Number.isFinite(value) && value > 0
    ? Math.floor(value)
    : fallback
}

function readBoundedInteger(value: unknown, fallback: number, min: number, max: number): number {
  const next = readPositiveInteger(value, fallback)
  if (next < min || next > max) {
    return fallback
  }
  return next
}

function normalizeStringList(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return []
  }
  return value
    .filter((entry): entry is string => typeof entry === 'string')
    .map((entry) => entry.trim())
    .filter((entry) => entry.length > 0)
}

function normalizeColumnVisibility(input: unknown): Record<string, boolean> {
  if (!isObjectRecord(input)) {
    return {}
  }
  const result: Record<string, boolean> = {}
  for (const [key, value] of Object.entries(input)) {
    if (typeof value === 'boolean') {
      result[key] = value
    }
  }
  return result
}

function normalizeNavLabels(input: unknown): Record<string, string> {
  if (!isObjectRecord(input)) {
    return {}
  }
  const result: Record<string, string> = {}
  for (const [key, value] of Object.entries(input)) {
    const nextKey = key.trim()
    const nextValue = readString(value).trim()
    if (!nextKey || !nextValue) {
      continue
    }
    result[nextKey] = nextValue
  }
  return result
}

function normalizeTablePref(input: unknown): TablePrefsV1 {
  const record = isObjectRecord(input) ? input : {}
  const density =
    record.density === 'compact' || record.density === 'standard' || record.density === 'comfortable'
      ? record.density
      : 'standard'

  const pinnedColumnsRecord = isObjectRecord(record.pinnedColumns) ? record.pinnedColumns : {}

  return {
    version: 1,
    pageSize: readPositiveInteger(record.pageSize, 25),
    density,
    columnVisibility: normalizeColumnVisibility(record.columnVisibility),
    columnOrder: normalizeStringList(record.columnOrder),
    pinnedColumns: {
      left: normalizeStringList(pinnedColumnsRecord.left),
      right: normalizeStringList(pinnedColumnsRecord.right),
    },
  }
}

function normalizeTablePrefs(input: unknown): TablePrefsMap {
  if (!isObjectRecord(input)) {
    return {}
  }
  const result: TablePrefsMap = {}
  for (const [key, value] of Object.entries(input)) {
    const storageKey = key.trim()
    if (!storageKey) {
      continue
    }
    result[storageKey] = normalizeTablePref(value)
  }
  return result
}

function normalizeUiPrefs(input: unknown): UiPrefs {
  const record = isObjectRecord(input) ? input : {}
  const themeMode = isThemeMode(record.themeMode) ? record.themeMode : defaultUiPrefs.themeMode
  const palettePreset = readString(record.palettePreset, defaultUiPrefs.palettePreset).trim()

  return {
    themeMode,
    palettePreset: palettePreset || defaultUiPrefs.palettePreset,
    navCollapsed: readBoolean(record.navCollapsed, defaultUiPrefs.navCollapsed),
    showSukumadMenu: readBoolean(record.showSukumadMenu, defaultUiPrefs.showSukumadMenu),
    showAdministrationMenu: readBoolean(record.showAdministrationMenu, defaultUiPrefs.showAdministrationMenu),
    pinActionsColumnRight: readBoolean(record.pinActionsColumnRight, defaultUiPrefs.pinActionsColumnRight),
    dataGridBorderRadius: readBoundedInteger(
      record.dataGridBorderRadius,
      defaultUiPrefs.dataGridBorderRadius,
      4,
      32,
    ),
    navLabels: normalizeNavLabels(record.navLabels),
  }
}

function normalizeSettings(input: unknown): AppSettings {
  const record = isObjectRecord(input) ? input : {}
  const authMode = isAuthMode(record.authMode) ? record.authMode : 'password'
  const apiToken = readString(record.apiToken).trim()
  const refreshToken = readString(record.refreshToken).trim()

  return {
    apiBaseUrl: readString(record.apiBaseUrl).trim(),
    authMode,
    apiToken: authMode === 'api_token' && apiToken ? apiToken : undefined,
    refreshToken: refreshToken || undefined,
    requestTimeoutSeconds: readPositiveInteger(
      record.requestTimeoutSeconds,
      defaultSettings.requestTimeoutSeconds,
    ),
    uiPrefs: normalizeUiPrefs(record.uiPrefs),
    tablePrefs: normalizeTablePrefs(record.tablePrefs),
  }
}

export const settingsStore: SettingsStore = {
  async loadSettings() {
    const bindings = getWailsBindings()
    if (!bindings) {
      return defaultSettings
    }
    try {
      const settings = await bindings.LoadSettings()
      return normalizeSettings(settings)
    } catch {
      return defaultSettings
    }
  },
  async saveSettings(patch: SaveSettingsPatch) {
    const fallback = normalizeSettings({
      ...defaultSettings,
      ...patch,
      uiPrefs: { ...defaultSettings.uiPrefs, ...patch.uiPrefs },
      tablePrefs: patch.tablePrefs ?? defaultSettings.tablePrefs,
    })

    const bindings = getWailsBindings()
    if (!bindings) {
      return fallback
    }

    try {
      const { main } = await import('../../wailsjs/go/models')
      const settings = await bindings.SaveSettings(new main.SettingsPatch(patch))
      return normalizeSettings(settings)
    } catch {
      return normalizeSettings({
        ...defaultSettings,
        ...patch,
        uiPrefs: { ...defaultSettings.uiPrefs, ...patch.uiPrefs },
        tablePrefs: patch.tablePrefs ?? defaultSettings.tablePrefs,
      })
    }
  },
  async resetSettings() {
    const bindings = getWailsBindings()
    if (!bindings) {
      return defaultSettings
    }
    try {
      const settings = await bindings.ResetSettings()
      return normalizeSettings(settings)
    } catch {
      return defaultSettings
    }
  },
}
