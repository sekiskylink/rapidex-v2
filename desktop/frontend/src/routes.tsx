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
import { createApiClient } from './api/client'
import { useSessionPrincipal } from './auth/hooks'
import { configureSessionStorage, getSessionPrincipal, isAuthenticated, setSessionPrincipal } from './auth/session'
import { AppShell } from './components/AppShell'
import { AuditPage } from './pages/AuditPage'
import { DashboardPage } from './pages/DashboardPage'
import { DocumentationPage } from './pages/DocumentationPage'
import { ForgotPasswordPage } from './pages/ForgotPasswordPage'
import { ForbiddenPage } from './pages/ForbiddenPage'
import { LoginPage } from './pages/LoginPage'
import { ModuleDisabledPage } from './pages/ModuleDisabledPage'
import { NotFoundPage } from './pages/NotFoundPage'
import { ObservabilityPage } from './pages/ObservabilityPage'
import { PermissionsPage } from './pages/PermissionsPage'
import { DeliveriesPage } from './pages/DeliveriesPage'
import { JobsPage } from './pages/JobsPage'
import { RequestsPage } from './pages/RequestsPage'
import { SchedulerJobFormPage } from './pages/SchedulerJobFormPage'
import { SchedulerJobsPage } from './pages/SchedulerJobsPage'
import { SchedulerJobRunsPage } from './pages/SchedulerJobRunsPage'
import { ResetPasswordPage } from './pages/ResetPasswordPage'
import { RolesPage } from './pages/RolesPage'
import { ServersPage } from './pages/ServersPage'
import { SettingsPage } from './pages/SettingsPage'
import { SetupPage } from './pages/SetupPage'
import { UsersPage } from './pages/UsersPage'
import {
  normalizeRequestsRouteSearch,
  normalizeDeliveriesRouteSearch,
  normalizeJobsRouteSearch,
  normalizeSchedulerRouteSearch,
  normalizeObservabilityRouteSearch,
} from './pages/listRouteSearch'
import { settingsStore } from './settings/store'
import type { SettingsStore } from './settings/types'
import { getModuleLabelForPath } from './registry/moduleEnablement'
import { applyBootstrap, hydrateBootstrapFromCache } from './bootstrap/state'
import { getRouteAccessState } from './registry/navigation'

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

      if (!settings.apiBaseUrl.trim()) {
        void navigate({ to: '/setup', replace: true })
        return
      }

      void navigate({ to: isAuthenticated() ? '/dashboard' : '/login', replace: true })
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
        void navigate({ to: '/setup', replace: true })
        return
      }

      if (isAuthenticated()) {
        void navigate({ to: '/dashboard', replace: true })
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

function ForgotPasswordGatePage() {
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
        void navigate({ to: '/setup', replace: true })
        return
      }

      if (isAuthenticated()) {
        void navigate({ to: '/dashboard', replace: true })
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

  return <ForgotPasswordPage />
}

function ResetPasswordGatePage() {
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
        void navigate({ to: '/setup', replace: true })
        return
      }

      if (isAuthenticated()) {
        void navigate({ to: '/dashboard', replace: true })
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

  return <ResetPasswordPage />
}

function AuthenticatedGatePage() {
  const router = useRouter()
  const navigate = useNavigate()
  const [ready, setReady] = React.useState(false)
  const settingsStore = router.options.context.settingsStore
  const apiClient = React.useMemo(
    () =>
      createApiClient({
        getSettings: () => settingsStore.loadSettings(),
      }),
    [settingsStore],
  )

  React.useEffect(() => {
    let active = true
    settingsStore.loadSettings().then(async (settings) => {
      if (!active) {
        return
      }

      if (!settings.apiBaseUrl.trim()) {
        void navigate({ to: '/setup', replace: true })
        return
      }

      if (!isAuthenticated()) {
        void navigate({ to: '/login', replace: true })
        return
      }

      hydrateBootstrapFromCache()

      try {
        const bootstrap = await apiClient.getBootstrap()
        if (!active) {
          return
        }
        applyBootstrap(bootstrap, 'live')
        if (bootstrap.principal?.type === 'user' && bootstrap.principal.userId && bootstrap.principal.username) {
          setSessionPrincipal({
            id: bootstrap.principal.userId,
            username: bootstrap.principal.username,
            roles: bootstrap.principal.roles ?? [],
            permissions: bootstrap.principal.permissions ?? [],
          })
        }
      } catch {
        // Keep cached bootstrap/default module state when refresh fails.
      }

      if (!getSessionPrincipal()) {
        try {
          const me = await apiClient.me()
          if (!active) {
            return
          }
          setSessionPrincipal({
            id: me.id,
            username: me.username,
            roles: me.roles ?? [],
            permissions: me.permissions ?? [],
          })
        } catch {
          if (!active) {
            return
          }
          void navigate({ to: '/login', replace: true })
          return
        }
      }

      setReady(true)
    })

    return () => {
      active = false
    }
  }, [apiClient, navigate, settingsStore])

  if (!ready) {
    return null
  }

  return <AppShell />
}

function UsersRoutePage() {
  const principal = useSessionPrincipal()
  const accessState = getRouteAccessState(principal, '/users')
  if (accessState === 'allowed') {
    return <UsersPage />
  }
  if (accessState === 'module-disabled') {
    return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/users') ?? undefined} />
  }
  return <ForbiddenPage />
}

function DashboardRoutePage() {
  const principal = useSessionPrincipal()
  const accessState = getRouteAccessState(principal, '/dashboard')
  if (accessState === 'allowed') {
    return <DashboardPage />
  }
  if (accessState === 'module-disabled') {
    return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/dashboard') ?? undefined} />
  }
  return <ForbiddenPage />
}

function RolesRoutePage() {
  const principal = useSessionPrincipal()
  const accessState = getRouteAccessState(principal, '/roles')
  if (accessState === 'allowed') {
    return <RolesPage />
  }
  if (accessState === 'module-disabled') {
    return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/roles') ?? undefined} />
  }
  return <ForbiddenPage />
}

function PermissionsRoutePage() {
  const principal = useSessionPrincipal()
  const accessState = getRouteAccessState(principal, '/permissions')
  if (accessState === 'allowed') {
    return <PermissionsPage />
  }
  if (accessState === 'module-disabled') {
    return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/permissions') ?? undefined} />
  }
  return <ForbiddenPage />
}

function AuditRoutePage() {
  const principal = useSessionPrincipal()
  const accessState = getRouteAccessState(principal, '/audit')
  if (accessState === 'allowed') {
    return <AuditPage />
  }
  if (accessState === 'module-disabled') {
    return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/audit') ?? undefined} />
  }
  return <ForbiddenPage />
}

function SettingsRoutePage() {
  const principal = useSessionPrincipal()
  const accessState = getRouteAccessState(principal, '/settings')
  if (accessState === 'allowed') {
    return <SettingsPage />
  }
  if (accessState === 'module-disabled') {
    return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/settings') ?? undefined} />
  }
  return <ForbiddenPage />
}

function ServersRoutePage() {
  const principal = useSessionPrincipal()
  const accessState = getRouteAccessState(principal, '/servers')
  if (accessState === 'allowed') {
    return <ServersPage />
  }
  if (accessState === 'module-disabled') {
    return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/servers') ?? undefined} />
  }
  return <ForbiddenPage />
}

function RequestsRoutePage() {
  const principal = useSessionPrincipal()
  const accessState = getRouteAccessState(principal, '/requests')
  if (accessState === 'allowed') {
    return <RequestsPage />
  }
  if (accessState === 'module-disabled') {
    return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/requests') ?? undefined} />
  }
  return <ForbiddenPage />
}

function DeliveriesRoutePage() {
  const principal = useSessionPrincipal()
  const accessState = getRouteAccessState(principal, '/deliveries')
  if (accessState === 'allowed') {
    return <DeliveriesPage />
  }
  if (accessState === 'module-disabled') {
    return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/deliveries') ?? undefined} />
  }
  return <ForbiddenPage />
}

function JobsRoutePage() {
  const principal = useSessionPrincipal()
  const accessState = getRouteAccessState(principal, '/jobs')
  if (accessState === 'allowed') {
    return <JobsPage />
  }
  if (accessState === 'module-disabled') {
    return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/jobs') ?? undefined} />
  }
  return <ForbiddenPage />
}

function SchedulerRoutePage() {
  const principal = useSessionPrincipal()
  const accessState = getRouteAccessState(principal, '/scheduler')
  if (accessState === 'allowed') {
    return <SchedulerJobsPage />
  }
  if (accessState === 'module-disabled') {
    return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/scheduler') ?? undefined} />
  }
  return <ForbiddenPage />
}

function SchedulerFormRoutePage() {
  const principal = useSessionPrincipal()
  const accessState = getRouteAccessState(principal, '/scheduler')
  if (accessState === 'allowed') {
    return <SchedulerJobFormPage />
  }
  if (accessState === 'module-disabled') {
    return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/scheduler') ?? undefined} />
  }
  return <ForbiddenPage />
}

function SchedulerRunsRoutePage() {
  const principal = useSessionPrincipal()
  const accessState = getRouteAccessState(principal, '/scheduler')
  if (accessState === 'allowed') {
    return <SchedulerJobRunsPage />
  }
  if (accessState === 'module-disabled') {
    return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/scheduler') ?? undefined} />
  }
  return <ForbiddenPage />
}

function ObservabilityRoutePage() {
  const principal = useSessionPrincipal()
  const accessState = getRouteAccessState(principal, '/observability')
  if (accessState === 'allowed') {
    return <ObservabilityPage />
  }
  if (accessState === 'module-disabled') {
    return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/observability') ?? undefined} />
  }
  return <ForbiddenPage />
}

function DocumentationRoutePage() {
  const principal = useSessionPrincipal()
  const accessState = getRouteAccessState(principal, '/documentation')
  if (accessState === 'allowed') {
    return <DocumentationPage />
  }
  if (accessState === 'module-disabled') {
    return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/documentation') ?? undefined} />
  }
  return <ForbiddenPage />
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

const forgotPasswordRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/forgot-password',
  component: ForgotPasswordGatePage,
})

const resetPasswordRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/reset-password',
  component: ResetPasswordGatePage,
})

const authenticatedRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'authenticated',
  component: AuthenticatedGatePage,
})

const dashboardRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/dashboard',
  component: DashboardRoutePage,
})

const settingsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/settings',
  component: SettingsRoutePage,
})

const usersRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/users',
  component: UsersRoutePage,
})

const rolesRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/roles',
  component: RolesRoutePage,
})

const permissionsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/permissions',
  component: PermissionsRoutePage,
})

const auditRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/audit',
  component: AuditRoutePage,
})

const serversRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/servers',
  component: ServersRoutePage,
})

const requestsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/requests',
  validateSearch: (search: Record<string, unknown>) =>
    normalizeRequestsRouteSearch(search),
  component: RequestsRoutePage,
})

const deliveriesRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/deliveries',
  validateSearch: (search: Record<string, unknown>) =>
    normalizeDeliveriesRouteSearch(search),
  component: DeliveriesRoutePage,
})

const jobsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/jobs',
  validateSearch: (search: Record<string, unknown>) =>
    normalizeJobsRouteSearch(search),
  component: JobsRoutePage,
})

const schedulerRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/scheduler',
  validateSearch: (search: Record<string, unknown>) =>
    normalizeSchedulerRouteSearch(search),
  component: SchedulerRoutePage,
})

const schedulerCreateRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/scheduler/new',
  component: SchedulerFormRoutePage,
})

const schedulerEditRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/scheduler/$jobId',
  component: SchedulerFormRoutePage,
})

const schedulerRunsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/scheduler/$jobId/runs',
  component: SchedulerRunsRoutePage,
})

const observabilityRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/observability',
  validateSearch: (search: Record<string, unknown>) =>
    normalizeObservabilityRouteSearch(search),
  component: ObservabilityRoutePage,
})

const documentationRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/documentation',
  component: DocumentationRoutePage,
})

const routeTree = rootRoute.addChildren([
  indexRoute,
  setupRoute,
  loginRoute,
  forgotPasswordRoute,
  resetPasswordRoute,
  authenticatedRoute.addChildren([
    dashboardRoute,
    settingsRoute,
    usersRoute,
    rolesRoute,
    permissionsRoute,
    auditRoute,
    serversRoute,
    requestsRoute,
    deliveriesRoute,
    jobsRoute,
    schedulerRoute,
    schedulerCreateRoute,
    schedulerEditRoute,
    schedulerRunsRoute,
    observabilityRoute,
    documentationRoute,
  ]),
])

export function createAppRouter(
  initialEntries: string[] = ['/'],
  routerSettingsStore: SettingsStore = settingsStore,
) {
  configureSessionStorage(routerSettingsStore)

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
