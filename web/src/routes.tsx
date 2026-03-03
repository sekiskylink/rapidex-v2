import React from 'react'
import {
  createBrowserHistory,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  type RouterHistory,
  useNavigate,
} from '@tanstack/react-router'
import App from './App'
import { DashboardPage } from './pages/DashboardPage'
import { LoginPage } from './pages/LoginPage'
import { NotFoundPage } from './pages/NotFoundPage'
import { RouteErrorPage } from './pages/RouteErrorPage'

const rootRoute = createRootRoute({
  component: App,
  notFoundComponent: NotFoundPage,
  errorComponent: RouteErrorPage,
})

function IndexRedirectPage() {
  const navigate = useNavigate()

  React.useEffect(() => {
    void navigate({ to: '/login', replace: true })
  }, [navigate])

  return null
}

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: IndexRedirectPage,
})

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  component: LoginPage,
})

const dashboardRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/dashboard',
  component: DashboardPage,
})

const routeTree = rootRoute.addChildren([indexRoute, loginRoute, dashboardRoute])

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
