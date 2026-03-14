export const AUTH_MODES = ['password', 'api_token'] as const
export type AuthMode = (typeof AUTH_MODES)[number]

export const THEME_MODES = ['light', 'dark', 'system'] as const
export type ThemeMode = (typeof THEME_MODES)[number]

export interface UiPrefs {
  themeMode: ThemeMode
  palettePreset: string
  navCollapsed: boolean
  showSukumadMenu: boolean
  showAdministrationMenu: boolean
  pinActionsColumnRight: boolean
  dataGridBorderRadius: number
  navLabels: Record<string, string>
}

export interface TablePinnedColumns {
  left: string[]
  right: string[]
}

export interface TablePrefsV1 {
  version: 1
  pageSize: number
  density: 'compact' | 'standard' | 'comfortable'
  columnVisibility: Record<string, boolean>
  columnOrder: string[]
  pinnedColumns: TablePinnedColumns
}

export type TablePrefsMap = Record<string, TablePrefsV1>

export interface AppSettings {
  apiBaseUrl: string
  authMode: AuthMode
  apiToken?: string
  refreshToken?: string
  requestTimeoutSeconds: number
  uiPrefs: UiPrefs
  tablePrefs: TablePrefsMap
}

export interface SaveSettingsPatch {
  apiBaseUrl?: string
  authMode?: AuthMode
  apiToken?: string
  refreshToken?: string
  requestTimeoutSeconds?: number
  uiPrefs?: Partial<UiPrefs>
  tablePrefs?: TablePrefsMap
}

export interface SettingsStore {
  loadSettings: () => Promise<AppSettings>
  saveSettings: (patch: SaveSettingsPatch) => Promise<AppSettings>
  resetSettings: () => Promise<AppSettings>
}

export const defaultUiPrefs: UiPrefs = {
  themeMode: 'system',
  palettePreset: 'ocean',
  navCollapsed: false,
  showSukumadMenu: true,
  showAdministrationMenu: true,
  pinActionsColumnRight: true,
  dataGridBorderRadius: 12,
  navLabels: {},
}

export const defaultSettings: AppSettings = {
  apiBaseUrl: '',
  authMode: 'password',
  requestTimeoutSeconds: 15,
  uiPrefs: defaultUiPrefs,
  tablePrefs: {},
}
