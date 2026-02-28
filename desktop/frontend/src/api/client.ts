import type { AppSettings } from '../settings/types'

interface ApiClientDeps {
  getSettings: () => Promise<AppSettings>
}

export function createApiClient(deps: ApiClientDeps) {
  return {
    async healthCheck() {
      const settings = await deps.getSettings()
      const baseUrl = settings.apiBaseUrl.trim()
      if (!baseUrl) {
        throw new Error('API base URL is required')
      }

      const controller = new AbortController()
      const timeoutMs = Math.max(1, settings.requestTimeoutSeconds) * 1000
      const timeout = window.setTimeout(() => controller.abort(), timeoutMs)

      try {
        const response = await fetch(`${baseUrl.replace(/\/$/, '')}/api/v1/health`, {
          method: 'GET',
          signal: controller.signal,
        })
        if (!response.ok) {
          throw new Error(`Health check failed: HTTP ${response.status}`)
        }
      } catch (error) {
        if (error instanceof DOMException && error.name === 'AbortError') {
          throw new Error('Health check timed out')
        }
        throw error
      } finally {
        window.clearTimeout(timeout)
      }
    },
  }
}
