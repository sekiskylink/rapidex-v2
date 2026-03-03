import React from 'react'
import { useRouter } from '@tanstack/react-router'
import { createApiClient } from './client'

export function useApiClient() {
  const router = useRouter()
  return React.useMemo(
    () =>
      createApiClient({
        getSettings: () => router.options.context.settingsStore.loadSettings(),
      }),
    [router.options.context.settingsStore],
  )
}
