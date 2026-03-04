import { resolveApiBaseUrl } from './apiBaseUrl'

export function getApiBaseUrl() {
  return resolveApiBaseUrl()
}

export const apiBaseUrl = getApiBaseUrl()
export const appName = import.meta.env.VITE_APP_NAME || 'BasePro Web'
