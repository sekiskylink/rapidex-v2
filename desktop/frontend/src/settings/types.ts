export const AUTH_MODES = ['password', 'api_token'] as const
export type AuthMode = (typeof AUTH_MODES)[number]

export interface AppSettings {
  apiBaseUrl: string
  authMode: AuthMode
  apiToken?: string
  refreshToken?: string
  requestTimeoutSeconds: number
}

export interface SaveSettingsPatch {
  apiBaseUrl?: string
  authMode?: AuthMode
  apiToken?: string
  refreshToken?: string
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
