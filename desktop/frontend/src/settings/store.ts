import { defaultSettings, type AppSettings, type SaveSettingsPatch, type SettingsStore } from './types'
import { LoadSettings, ResetSettings, SaveSettings } from '../../wailsjs/go/main/App'

const hasWailsBindings = () =>
  typeof window !== 'undefined' &&
  typeof window.go !== 'undefined' &&
  typeof window.go.main !== 'undefined' &&
  typeof window.go.main.App !== 'undefined'

function normalizeSettings(input: AppSettings): AppSettings {
  return {
    apiBaseUrl: (input.apiBaseUrl ?? '').trim(),
    authMode: input.authMode === 'api_token' ? 'api_token' : 'password',
    apiToken: input.authMode === 'api_token' ? input.apiToken?.trim() || undefined : undefined,
    requestTimeoutSeconds:
      typeof input.requestTimeoutSeconds === 'number' && input.requestTimeoutSeconds > 0
        ? Math.floor(input.requestTimeoutSeconds)
        : defaultSettings.requestTimeoutSeconds,
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
