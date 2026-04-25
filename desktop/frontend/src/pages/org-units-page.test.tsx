import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearSession, configureSessionStorage, setSession } from '../auth/session'
import { createAppRouter } from '../routes'
import { AppThemeProvider } from '../ui/theme'
import { defaultSettings, type AppSettings, type SaveSettingsPatch, type SettingsStore } from '../settings/types'

function renderCellValue(column: Record<string, any>, row: Record<string, any>) {
  if (typeof column.renderCell === 'function') {
    return column.renderCell({
      row,
      field: column.field,
      value: row[column.field],
      colDef: column,
      id: row.id,
    })
  }
  if (typeof column.valueGetter === 'function') {
    return column.valueGetter(row[column.field], row, column, null)
  }
  const value = row[column.field]
  return value === undefined || value === null ? '' : String(value)
}

vi.mock('@mui/x-data-grid', () => ({
  DataGrid: (props: Record<string, any>) => {
    const columns = Array.isArray(props.columns) ? props.columns : []
    const rows = Array.isArray(props.rows) ? props.rows : []
    return (
      <div>
        <div>
          {columns.map((column: Record<string, any>) => (
            <span key={column.field}>{typeof column.renderHeader === 'function' ? column.renderHeader({ colDef: column }) : column.headerName}</span>
          ))}
        </div>
        {rows.map((row: Record<string, any>) => (
          <div key={String(row.id)}>
            {columns.map((column: Record<string, any>) => (
              <div key={column.field}>{renderCellValue(column, row)}</div>
            ))}
          </div>
        ))}
      </div>
    )
  },
}))

function createMockSettingsStore(seed: AppSettings): SettingsStore {
  let state = {
    ...seed,
    uiPrefs: {
      ...seed.uiPrefs,
    },
  }

  return {
    loadSettings: async () => state,
    saveSettings: async (patch: SaveSettingsPatch) => {
      state = {
        ...state,
        ...patch,
        uiPrefs: {
          ...state.uiPrefs,
          ...(patch.uiPrefs ?? {}),
        },
        tablePrefs: patch.tablePrefs ?? state.tablePrefs,
      }
      return state
    },
    resetSettings: async () => state,
  }
}

function renderRoute(initialPath: string, store: SettingsStore) {
  const router = createAppRouter([initialPath], store)
  const queryClient = new QueryClient()
  return render(
    <QueryClientProvider client={queryClient}>
      <AppThemeProvider store={store}>
        <RouterProvider router={router} />
      </AppThemeProvider>
    </QueryClientProvider>,
  )
}

function buildOrgUnit(overrides: Record<string, unknown> = {}) {
  return {
    id: 11,
    uid: 'ou-11',
    code: 'FAC-11',
    name: 'Alpha Health Centre',
    shortName: 'Alpha HC',
    description: 'Alpha facility',
    parentId: 9,
    hierarchyLevel: 2,
    path: '/UG/Kampala/Alpha',
    displayPath: 'Uganda / Kampala District',
    address: '',
    email: '',
    url: '',
    phoneNumber: '',
    extras: {},
    attributeValues: {},
    deleted: false,
    hasChildren: false,
    ...overrides,
  }
}

describe('desktop org units page', () => {
  beforeEach(async () => {
    await clearSession()
  })

  afterEach(async () => {
    cleanup()
    await clearSession()
  })

  it('issues an org-unit search request and shows the matching option text', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
      refreshToken: 'refresh-token',
    })
    configureSessionStorage(store)
    await setSession({
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      expiresAt: Date.now() + 60_000,
    })

    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('/api/v1/auth/me')) {
        return new Response(JSON.stringify({ id: 5, username: 'alice', roles: ['Staff'], permissions: ['orgunits.read'] }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      if (url.endsWith('/api/v1/orgunits?page=0&pageSize=200')) {
        return new Response(JSON.stringify({ items: [], totalCount: 0, page: 0, pageSize: 200 }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      if (url.endsWith('/api/v1/orgunits/sync-state')) {
        return new Response(JSON.stringify({}), { status: 200, headers: { 'Content-Type': 'application/json' } })
      }
      if (url.includes('/api/v1/orgunits?page=0&pageSize=20&search=Alpha')) {
        return new Response(JSON.stringify({ items: [buildOrgUnit()], totalCount: 1, page: 0, pageSize: 20 }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderRoute('/orgunits', store)

    const searchBox = await screen.findByRole('combobox', { name: 'Find org unit' })
    fireEvent.focus(searchBox)
    fireEvent.change(searchBox, { target: { value: 'Alpha' } })

    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining('/api/v1/orgunits?page=0&pageSize=20&search=Alpha'), expect.anything()))
    expect(await screen.findByText('Alpha Health Centre')).toBeInTheDocument()
  })

  it('opens facility details from the browse hierarchy', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
      refreshToken: 'refresh-token',
    })
    configureSessionStorage(store)
    await setSession({
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      expiresAt: Date.now() + 60_000,
    })

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/api/v1/auth/me')) {
          return new Response(JSON.stringify({ id: 5, username: 'alice', roles: ['Staff'], permissions: ['orgunits.read'] }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        if (url.endsWith('/api/v1/orgunits?page=0&pageSize=200')) {
          return new Response(JSON.stringify({ items: [], totalCount: 0, page: 0, pageSize: 200 }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        if (url.endsWith('/api/v1/orgunits/sync-state')) {
          return new Response(JSON.stringify({}), { status: 200, headers: { 'Content-Type': 'application/json' } })
        }
        if (url.includes('/api/v1/orgunits?page=0&pageSize=200&rootsOnly=true')) {
          return new Response(
            JSON.stringify({
              items: [
                buildOrgUnit({ id: 2, name: 'Beta District', parentId: null, hierarchyLevel: 1, path: '/UG/Beta', displayPath: 'Uganda', hasChildren: true }),
                buildOrgUnit({ id: 1, name: 'Alpha District', parentId: null, hierarchyLevel: 1, path: '/UG/Alpha', displayPath: 'Uganda', hasChildren: true }),
              ],
              totalCount: 2,
              page: 0,
              pageSize: 200,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderRoute('/orgunits', store)

    fireEvent.click(await screen.findByRole('button', { name: 'Browse hierarchy' }))
    const dialog = await screen.findByRole('dialog', { name: 'Browse Facility Hierarchy' })
    const rootButtons = within(dialog)
      .getAllByRole('button')
      .filter((button) => button.textContent?.includes('District') && button.textContent?.includes('Uganda'))
    expect(rootButtons.map((button) => button.textContent)).toEqual(['Alpha DistrictUganda', 'Beta DistrictUganda'])

    fireEvent.click(within(dialog).getByRole('button', { name: 'Beta District Uganda' }))
    const details = await screen.findByRole('dialog', { name: 'Facility Details' })
    expect(within(details).getAllByText('Beta District').length).toBeGreaterThan(0)
    expect(within(details).getByText('Overview')).toBeInTheDocument()
    expect(within(details).getByText('Hierarchy')).toBeInTheDocument()
    expect(within(details).getByText('Active')).toBeInTheDocument()
    expect(within(details).getByText('Level 1')).toBeInTheDocument()
  })

  it('shows child hierarchy entries alphabetically when browsing deeper', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
      refreshToken: 'refresh-token',
    })
    configureSessionStorage(store)
    await setSession({
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      expiresAt: Date.now() + 60_000,
    })

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/api/v1/auth/me')) {
          return new Response(JSON.stringify({ id: 5, username: 'alice', roles: ['Staff'], permissions: ['orgunits.read'] }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        if (url.endsWith('/api/v1/orgunits?page=0&pageSize=200')) {
          return new Response(JSON.stringify({ items: [], totalCount: 0, page: 0, pageSize: 200 }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        if (url.endsWith('/api/v1/orgunits/sync-state')) {
          return new Response(JSON.stringify({}), { status: 200, headers: { 'Content-Type': 'application/json' } })
        }
        if (url.includes('/api/v1/orgunits?page=0&pageSize=200&rootsOnly=true')) {
          return new Response(
            JSON.stringify({
              items: [
                buildOrgUnit({ id: 2, name: 'Beta District', parentId: null, hierarchyLevel: 1, path: '/UG/Beta', displayPath: 'Uganda', hasChildren: true }),
                buildOrgUnit({ id: 1, name: 'Alpha District', parentId: null, hierarchyLevel: 1, path: '/UG/Alpha', displayPath: 'Uganda', hasChildren: true }),
              ],
              totalCount: 2,
              page: 0,
              pageSize: 200,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/orgunits?page=0&pageSize=200&parentId=1')) {
          return new Response(
            JSON.stringify({
              items: [
                buildOrgUnit({ id: 4, name: 'Zulu Health Centre', parentId: 1, displayPath: 'Uganda / Alpha District' }),
                buildOrgUnit({ id: 3, name: 'Alpha Health Centre', parentId: 1, displayPath: 'Uganda / Alpha District' }),
              ],
              totalCount: 2,
              page: 0,
              pageSize: 200,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderRoute('/orgunits', store)

    fireEvent.click(await screen.findByRole('button', { name: 'Browse hierarchy' }))
    const dialog = await screen.findByRole('dialog', { name: 'Browse Facility Hierarchy' })
    fireEvent.click(within(dialog).getAllByRole('button', { name: 'Browse children' })[0])
    await screen.findByRole('button', { name: 'Alpha District' })

    const childButtons = within(dialog)
      .getAllByRole('button')
      .filter((button) => button.textContent?.includes('Health Centre'))
    expect(childButtons.map((button) => button.textContent)).toEqual(['Alpha Health CentreUganda / Alpha District', 'Zulu Health CentreUganda / Alpha District'])
  })
})
