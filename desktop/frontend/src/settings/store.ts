import {
  AUTH_MODES,
  defaultSettings,
  type AppSettings,
  type AuthMode,
  type SaveSettingsPatch,
  type SettingsStore,
} from './types'
import { LoadSettings, ResetSettings, SaveSettings } from '../../wailsjs/go/main/App'

const hasWailsBindings = () =>
  typeof window !== 'undefined' &&
  typeof window.go !== 'undefined' &&
  typeof window.go.main !== 'undefined' &&
  typeof window.go.main.App !== 'undefined'

function isObjectRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isAuthMode(value: unknown): value is AuthMode {
  return typeof value === 'string' && AUTH_MODES.some((mode) => mode === value)
}

function readString(value: unknown, fallback = ''): string {
  return typeof value === 'string' ? value : fallback
}

function readPositiveInteger(value: unknown, fallback: number): number {
  return typeof value === 'number' && Number.isFinite(value) && value > 0
    ? Math.floor(value)
    : fallback
}

function normalizeSettings(input: unknown): AppSettings {
  const record = isObjectRecord(input) ? input : {}
  const authMode = isAuthMode(record.authMode) ? record.authMode : 'password'
  const apiToken = readString(record.apiToken).trim()

  return {
    apiBaseUrl: readString(record.apiBaseUrl).trim(),
    authMode,
    apiToken: authMode === 'api_token' && apiToken ? apiToken : undefined,
    requestTimeoutSeconds: readPositiveInteger(
      record.requestTimeoutSeconds,
      defaultSettings.requestTimeoutSeconds,
    ),
  }
}

export const settingsStore: SettingsStore = {
  async loadSettings() {
    if (!hasWailsBindings()) {
      return defaultSettings
    }
    const settings = await LoadSettings()
    return normalizeSettings(settings)
  },
  async saveSettings(patch: SaveSettingsPatch) {
    if (!hasWailsBindings()) {
      return normalizeSettings({ ...defaultSettings, ...patch })
    }
    const settings = await SaveSettings(patch)
    return normalizeSettings(settings)
  },
  async resetSettings() {
    if (!hasWailsBindings()) {
      return defaultSettings
    }
    const settings = await ResetSettings()
    return normalizeSettings(settings)
  },
}
