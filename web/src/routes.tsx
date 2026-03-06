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
import { canAccessRoute } from './navigation'
import { AuditPage } from './pages/AuditPage'
import { DashboardPage } from './pages/DashboardPage'
import { LoginPage } from './pages/LoginPage'
import { NotAuthorizedPage } from './pages/NotAuthorizedPage'
import { NotFoundPage } from './pages/NotFoundPage'
import { PermissionsPage } from './pages/PermissionsPage'
import { RolesPage } from './pages/RolesPage'
import { RouteErrorPage } from './pages/RouteErrorPage'
import { SettingsPage } from './pages/SettingsPage'
import { UsersPage } from './pages/UsersPage'

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
  component: DashboardPage,
})

const usersRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/users',
  component: () => (canAccessRoute('/users') ? <UsersPage /> : <NotAuthorizedPage />),
})

const rolesRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/roles',
  component: () => (canAccessRoute('/roles') ? <RolesPage /> : <NotAuthorizedPage />),
})

const permissionsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/permissions',
  component: () => (canAccessRoute('/permissions') ? <PermissionsPage /> : <NotAuthorizedPage />),
})

const auditRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/audit',
  component: () => (canAccessRoute('/audit') ? <AuditPage /> : <NotAuthorizedPage />),
})

const settingsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/settings',
  component: () => (canAccessRoute('/settings') ? <SettingsPage /> : <NotAuthorizedPage />),
})

const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  authenticatedRoute.addChildren([
    dashboardRoute,
    usersRoute,
    rolesRoute,
    permissionsRoute,
    auditRoute,
    settingsRoute,
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
