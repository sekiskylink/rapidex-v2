export type AuthMode = 'password' | 'api_token'

export interface AppSettings {
  apiBaseUrl: string
  authMode: AuthMode
  apiToken?: string
  requestTimeoutSeconds: number
}

export interface SaveSettingsPatch {
  apiBaseUrl?: string
  authMode?: AuthMode
  apiToken?: string
  requestTimeoutSeconds?: number
}

export interface SettingsStore {
  loadSettings: () => Promise<AppSettings>
  saveSettings: (patch: SaveSettingsPatch) => Promise<AppSettings>
  resetSettings: () => Promise<AppSettings>
}

export const defaultSettings: AppSettings = {
  apiBaseUrl: '',
  authMode: 'password',
  requestTimeoutSeconds: 15,
}
