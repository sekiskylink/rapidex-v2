import React from 'react'
import {
  createBrowserHistory,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
  type RouterHistory,
} from '@tanstack/react-router'
import App from './App'
import { getAuthSnapshot } from './auth/state'
import { AppShell } from './components/AppShell'
import { AuditPage } from './pages/AuditPage'
import { DashboardPage } from './pages/DashboardPage'
import { DocumentationPage } from './pages/DocumentationPage'
import { ForgotPasswordPage } from './pages/ForgotPasswordPage'
import { LoginPage } from './pages/LoginPage'
import { ModuleDisabledPage } from './pages/ModuleDisabledPage'
import { NotAuthorizedPage } from './pages/NotAuthorizedPage'
import { NotFoundPage } from './pages/NotFoundPage'
import { ObservabilityPage } from './pages/ObservabilityPage'
import { OrgUnitsPage } from './pages/OrgUnitsPage'
import { PermissionsPage } from './pages/PermissionsPage'
import { DeliveriesPage } from './pages/DeliveriesPage'
import { JobsPage } from './pages/JobsPage'
import { RequestsPage } from './pages/RequestsPage'
import { ReportersPage } from './pages/ReportersPage'
import { SchedulerJobFormPage } from './pages/SchedulerJobFormPage'
import { SchedulerJobsPage } from './pages/SchedulerJobsPage'
import { SchedulerJobRunsPage } from './pages/SchedulerJobRunsPage'
import { ResetPasswordPage } from './pages/ResetPasswordPage'
import { RolesPage } from './pages/RolesPage'
import { RouteErrorPage } from './pages/RouteErrorPage'
import { ServersPage } from './pages/ServersPage'
import { SettingsPage } from './pages/SettingsPage'
import { UsersPage } from './pages/UsersPage'
import {
  normalizeRequestsRouteSearch,
  normalizeDeliveriesRouteSearch,
  normalizeJobsRouteSearch,
  normalizeSchedulerRouteSearch,
  normalizeObservabilityRouteSearch,
} from './pages/listRouteSearch'
import { getModuleLabelForPath } from './registry/moduleEnablement'
import { getRouteAccessState } from './registry/navigation'

const rootRoute = createRootRoute({
  component: App,
  notFoundComponent: NotFoundPage,
  errorComponent: RouteErrorPage,
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  beforeLoad: () => {
    if (getAuthSnapshot().isAuthenticated) {
      throw redirect({ to: '/dashboard' })
    }
    throw redirect({ to: '/login' })
  },
  component: () => null,
})

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  beforeLoad: () => {
    if (getAuthSnapshot().isAuthenticated) {
      throw redirect({ to: '/dashboard' })
    }
  },
  component: LoginPage,
})

const forgotPasswordRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/forgot-password',
  beforeLoad: () => {
    if (getAuthSnapshot().isAuthenticated) {
      throw redirect({ to: '/dashboard' })
    }
  },
  component: ForgotPasswordPage,
})

const resetPasswordRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/reset-password',
  beforeLoad: () => {
    if (getAuthSnapshot().isAuthenticated) {
      throw redirect({ to: '/dashboard' })
    }
  },
  component: ResetPasswordPage,
})

const authenticatedRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'authenticated',
  beforeLoad: () => {
    if (!getAuthSnapshot().isAuthenticated) {
      throw redirect({ to: '/login' })
    }
  },
  component: AppShell,
})

const dashboardRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/dashboard',
  component: () => {
    const state = getRouteAccessState('/dashboard', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <DashboardPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/dashboard') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const usersRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/users',
  component: () => {
    const state = getRouteAccessState('/users', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <UsersPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/users') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const rolesRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/roles',
  component: () => {
    const state = getRouteAccessState('/roles', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <RolesPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/roles') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const permissionsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/permissions',
  component: () => {
    const state = getRouteAccessState('/permissions', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <PermissionsPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/permissions') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const auditRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/audit',
  component: () => {
    const state = getRouteAccessState('/audit', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <AuditPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/audit') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const settingsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/settings',
  beforeLoad: () => {
    throw redirect({ to: '/settings/general' })
  },
  component: () => null,
})

const settingsGeneralRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/settings/general',
  component: () => {
    const state = getRouteAccessState('/settings/general', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <SettingsPage section="general" />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/settings/general') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const settingsBrandingRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/settings/branding',
  component: () => {
    const state = getRouteAccessState('/settings/branding', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <SettingsPage section="branding" />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/settings/branding') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const settingsModulesRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/settings/modules',
  component: () => {
    const state = getRouteAccessState('/settings/modules', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <SettingsPage section="modules" />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/settings/modules') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const settingsIntegrationsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/settings/integrations',
  component: () => {
    const state = getRouteAccessState('/settings/integrations', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <SettingsPage section="integrations" />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/settings/integrations') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const settingsAboutRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/settings/about',
  component: () => {
    const state = getRouteAccessState('/settings/about', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <SettingsPage section="about" />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/settings/about') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const serversRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/servers',
  component: () => {
    const state = getRouteAccessState('/servers', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <ServersPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/servers') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const requestsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/requests',
  validateSearch: (search: Record<string, unknown>) =>
    normalizeRequestsRouteSearch(search),
  component: () => {
    const state = getRouteAccessState('/requests', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <RequestsPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/requests') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const deliveriesRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/deliveries',
  validateSearch: (search: Record<string, unknown>) =>
    normalizeDeliveriesRouteSearch(search),
  component: () => {
    const state = getRouteAccessState('/deliveries', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <DeliveriesPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/deliveries') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const jobsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/jobs',
  validateSearch: (search: Record<string, unknown>) =>
    normalizeJobsRouteSearch(search),
  component: () => {
    const state = getRouteAccessState('/jobs', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <JobsPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/jobs') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const schedulerRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/scheduler',
  validateSearch: (search: Record<string, unknown>) =>
    normalizeSchedulerRouteSearch(search),
  component: () => {
    const state = getRouteAccessState('/scheduler', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <SchedulerJobsPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/scheduler') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const schedulerCreateRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/scheduler/new',
  component: () => {
    const state = getRouteAccessState('/scheduler', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <SchedulerJobFormPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/scheduler') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const schedulerEditRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/scheduler/$jobId',
  component: () => {
    const state = getRouteAccessState('/scheduler', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <SchedulerJobFormPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/scheduler') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const schedulerRunsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/scheduler/$jobId/runs',
  component: () => {
    const state = getRouteAccessState('/scheduler', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <SchedulerJobRunsPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/scheduler') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const observabilityRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/observability',
  validateSearch: (search: Record<string, unknown>) =>
    normalizeObservabilityRouteSearch(search),
  component: () => {
    const state = getRouteAccessState('/observability', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <ObservabilityPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/observability') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const documentationRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/documentation',
  component: () => {
    const state = getRouteAccessState('/documentation', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <DocumentationPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/documentation') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const orgUnitsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/orgunits',
  component: () => {
    const state = getRouteAccessState('/orgunits', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <OrgUnitsPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/orgunits') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const reportersRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/reporters',
  component: () => {
    const state = getRouteAccessState('/reporters', getAuthSnapshot().user)
    if (state === 'allowed') {
      return <ReportersPage />
    }
    if (state === 'module-disabled') {
      return <ModuleDisabledPage moduleLabel={getModuleLabelForPath('/reporters') ?? undefined} />
    }
    return <NotAuthorizedPage />
  },
})

const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  forgotPasswordRoute,
  resetPasswordRoute,
  authenticatedRoute.addChildren([
    dashboardRoute,
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
    orgUnitsRoute,
    reportersRoute,
    settingsRoute,
    settingsGeneralRoute,
    settingsBrandingRoute,
    settingsModulesRoute,
    settingsIntegrationsRoute,
    settingsAboutRoute,
  ]),
])

export function createAppRouter(
  initialEntries: string[] = ['/'],
  history: RouterHistory = createMemoryHistory({ initialEntries }),
) {
  return createRouter({
    routeTree,
    history,
  })
}

export const router = createAppRouter(['/'], createBrowserHistory())

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
