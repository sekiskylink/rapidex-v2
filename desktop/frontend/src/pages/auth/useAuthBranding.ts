import React from 'react'
import { createApiClient } from '../../api/client'
import type { SettingsStore } from '../../settings/types'
import type { AuthBranding } from './AuthSplitLayout'

const defaultBranding: AuthBranding = {
  appDisplayName: 'RapidEx',
  loginImageUrl: null,
}

function normalizeBranding(payload: {
  appDisplayName?: string
  applicationDisplayName?: string
  loginImageUrl?: string | null
}): AuthBranding {
  const appDisplayName = (payload.appDisplayName ?? payload.applicationDisplayName ?? '').trim() || defaultBranding.appDisplayName
  const loginImageUrl = typeof payload.loginImageUrl === 'string' && payload.loginImageUrl.trim() ? payload.loginImageUrl.trim() : null
  return {
    appDisplayName,
    loginImageUrl,
  }
}

export function useAuthBranding(settingsStore: SettingsStore) {
  const [branding, setBranding] = React.useState<AuthBranding>(defaultBranding)

  const apiClient = React.useMemo(
    () =>
      createApiClient({
        getSettings: () => settingsStore.loadSettings(),
      }),
    [settingsStore],
  )

  React.useEffect(() => {
    let active = true
    apiClient
      .getPublicLoginBranding()
      .then((payload) => {
        if (!active) {
          return
        }
        setBranding(normalizeBranding(payload))
      })
      .catch(() => {
        if (active) {
          setBranding(defaultBranding)
        }
      })

    return () => {
      active = false
    }
  }, [apiClient])

  return branding
}
