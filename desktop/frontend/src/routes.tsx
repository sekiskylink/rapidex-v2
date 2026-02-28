import React from 'react'
import {
  createMemoryHistory,
  createRootRouteWithContext,
  createRoute,
  createRouter,
  useNavigate,
  useRouter,
} from '@tanstack/react-router'
import App from './App'
import { LoginPage } from './pages/LoginPage'
import { NotFoundPage } from './pages/NotFoundPage'
import { SetupPage } from './pages/SetupPage'
import { settingsStore } from './settings/store'
import type { SettingsStore } from './settings/types'

interface RouterContext {
  settingsStore: SettingsStore
}

const rootRoute = createRootRouteWithContext<RouterContext>()({
  component: App,
  notFoundComponent: NotFoundPage,
})

function IndexGatePage() {
  const router = useRouter()
  const navigate = useNavigate()

  React.useEffect(() => {
    let active = true
    router.options.context.settingsStore.loadSettings().then((settings) => {
      if (!active) {
        return
      }
      navigate({ to: settings.apiBaseUrl.trim() ? '/login' : '/setup', replace: true })
    })
    return () => {
      active = false
    }
  }, [navigate, router.options.context.settingsStore])

  return null
}

function LoginGatePage() {
  const router = useRouter()
  const navigate = useNavigate()
  const [ready, setReady] = React.useState(false)

  React.useEffect(() => {
    let active = true
    router.options.context.settingsStore.loadSettings().then((settings) => {
      if (!active) {
        return
      }
      if (!settings.apiBaseUrl.trim()) {
        navigate({ to: '/setup', replace: true })
        return
      }
      setReady(true)
    })
    return () => {
      active = false
    }
  }, [navigate, router.options.context.settingsStore])

  if (!ready) {
    return null
  }
  return <LoginPage />
}

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: IndexGatePage,
})

const setupRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/setup',
  component: SetupPage,
})

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  component: LoginGatePage,
})

const routeTree = rootRoute.addChildren([indexRoute, setupRoute, loginRoute])

export function createAppRouter(
  initialEntries: string[] = ['/'],
  routerSettingsStore: SettingsStore = settingsStore,
) {
  return createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries }),
    context: {
      settingsStore: routerSettingsStore,
    },
  })
}

export const router = createAppRouter()

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
