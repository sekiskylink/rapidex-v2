import React from 'react'
import { getSessionPrincipal, subscribeAuthState } from './session'

export function useSessionPrincipal() {
  return React.useSyncExternalStore(subscribeAuthState, getSessionPrincipal, getSessionPrincipal)
}

