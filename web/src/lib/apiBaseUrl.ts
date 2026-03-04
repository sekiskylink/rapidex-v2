export const API_BASE_URL_OVERRIDE_STORAGE_KEY = 'basepro.web.api_base_url_override'

function normalizeBaseUrl(baseUrl: string) {
  return baseUrl.trim().replace(/\/+$/, '')
}

export function getApiBaseUrlOverride() {
  if (typeof window === 'undefined') {
    return ''
  }

  return normalizeBaseUrl(window.localStorage.getItem(API_BASE_URL_OVERRIDE_STORAGE_KEY) ?? '')
}

export function setApiBaseUrlOverride(value: string) {
  if (typeof window === 'undefined') {
    return
  }

  const normalized = normalizeBaseUrl(value)
  if (!normalized) {
    window.localStorage.removeItem(API_BASE_URL_OVERRIDE_STORAGE_KEY)
    return
  }

  window.localStorage.setItem(API_BASE_URL_OVERRIDE_STORAGE_KEY, normalized)
}

export function resolveApiBaseUrl() {
  const override = getApiBaseUrlOverride()
  if (override) {
    return override
  }

  const fromViteEnv = import.meta.env.VITE_API_BASE_URL
  const fromProcessEnv =
    typeof process !== 'undefined' && process.env && typeof process.env.VITE_API_BASE_URL === 'string'
      ? process.env.VITE_API_BASE_URL
      : ''
  return normalizeBaseUrl(fromViteEnv || fromProcessEnv || '')
}
