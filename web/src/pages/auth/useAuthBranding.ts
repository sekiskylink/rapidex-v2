import React from 'react'
import { apiRequest } from '../../lib/api'
import { appName } from '../../lib/env'
import type { AuthBranding } from './AuthSplitLayout'

interface BrandingPayload {
  appDisplayName?: string
  applicationDisplayName?: string
  loginImageUrl?: string | null
}

function toBranding(payload?: BrandingPayload): AuthBranding {
  const appDisplayName = (payload?.appDisplayName ?? payload?.applicationDisplayName ?? '').trim() || appName
  const loginImageUrl = typeof payload?.loginImageUrl === 'string' && payload.loginImageUrl.trim() ? payload.loginImageUrl.trim() : null
  return {
    appDisplayName,
    loginImageUrl,
  }
}

export function useAuthBranding() {
  const [branding, setBranding] = React.useState<AuthBranding>(() => toBranding())

  React.useEffect(() => {
    let active = true
    apiRequest<BrandingPayload>('/settings/public/login-branding', { method: 'GET' }, { withAuth: false, retryOnUnauthorized: false })
      .then((payload) => {
        if (!active) {
          return
        }
        setBranding(toBranding(payload))
      })
      .catch(() => {
        if (active) {
          setBranding(toBranding())
        }
      })

    return () => {
      active = false
    }
  }, [])

  return branding
}
