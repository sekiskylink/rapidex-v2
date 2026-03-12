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
            <span key={column.field}>{column.headerName}</span>
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

describe('desktop servers page', () => {
  beforeEach(async () => {
    await clearSession()
  })

  afterEach(async () => {
    cleanup()
    await clearSession()
  })

  it('renders servers route and grouped navigation', async () => {
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
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/api/v1/auth/me')) {
          return new Response(
            JSON.stringify({
              id: 5,
              username: 'alice',
              roles: ['Staff'],
              permissions: ['servers.read', 'servers.write'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/servers?')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 1,
                  uid: 'uid-1',
                  name: 'DHIS2',
                  code: 'dhis2',
                  systemType: 'dhis2',
                  baseUrl: 'https://dhis.example.com',
                  endpointType: 'http',
                  httpMethod: 'POST',
                  useAsync: true,
                  parseResponses: true,
                  headers: {},
                  urlParams: {},
                  suspended: false,
                  createdAt: '2026-03-10T09:00:00Z',
                  updatedAt: '2026-03-10T10:00:00Z',
                },
              ],
              totalCount: 1,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/bootstrap')) {
          return new Response(JSON.stringify({}), { status: 200, headers: { 'Content-Type': 'application/json' } })
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderRoute('/servers', store)

    expect(await screen.findByRole('heading', { name: 'Servers', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('DHIS2')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Toggle Sukumad menu' })).toBeInTheDocument()
  })

  it('create and edit server flows call backend APIs', async () => {
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

    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('/api/v1/auth/me')) {
        return new Response(
          JSON.stringify({
            id: 5,
            username: 'alice',
            roles: ['Staff'],
            permissions: ['servers.read', 'servers.write'],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.includes('/api/v1/servers?')) {
        return new Response(
          JSON.stringify({
            items: [
              {
                id: 4,
                uid: 'uid-4',
                name: 'OpenHIM',
                code: 'openhim',
                systemType: 'api',
                baseUrl: 'https://openhim.example.com',
                endpointType: 'http',
                httpMethod: 'POST',
                useAsync: false,
                parseResponses: true,
                headers: {},
                urlParams: {},
                suspended: false,
                createdAt: '2026-03-10T09:00:00Z',
                updatedAt: '2026-03-10T10:00:00Z',
              },
            ],
            totalCount: 1,
            page: 1,
            pageSize: 25,
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.endsWith('/api/v1/servers/4') && (!init?.method || init.method === 'GET')) {
        return new Response(
          JSON.stringify({
            id: 4,
            uid: 'uid-4',
            name: 'OpenHIM',
            code: 'openhim',
            systemType: 'api',
            baseUrl: 'https://openhim.example.com',
            endpointType: 'http',
            httpMethod: 'POST',
            useAsync: false,
            parseResponses: true,
            headers: {},
            urlParams: {},
            suspended: false,
            createdAt: '2026-03-10T09:00:00Z',
            updatedAt: '2026-03-10T10:00:00Z',
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.endsWith('/api/v1/servers') && init?.method === 'POST') {
        return new Response(JSON.stringify({ id: 9 }), { status: 201, headers: { 'Content-Type': 'application/json' } })
      }
      if (url.endsWith('/api/v1/servers/4') && init?.method === 'PUT') {
        return new Response(JSON.stringify({ id: 4 }), { status: 200, headers: { 'Content-Type': 'application/json' } })
      }
      if (url.includes('/api/v1/bootstrap')) {
        return new Response(JSON.stringify({}), { status: 200, headers: { 'Content-Type': 'application/json' } })
      }
      return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderRoute('/servers', store)

    fireEvent.click(await screen.findByRole('button', { name: 'Create Server' }))
    const createDialog = await screen.findByRole('dialog', { name: 'Create Server' })
    expect(within(createDialog).getByTestId('desktop-server-create-form-grid')).toBeInTheDocument()
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Server Name' }), { target: { value: 'RapidPro' } })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Code' }), { target: { value: 'rapidpro' } })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Base URL' }), { target: { value: 'https://rapidpro.example.com' } })
    fireEvent.click(within(createDialog).getByRole('button', { name: 'Create' }))

    await waitFor(() => {
      expect(fetchMock.mock.calls.some((call) => String(call[0]).endsWith('/api/v1/servers'))).toBe(true)
    })

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for openhim' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Edit' }))
    const editDialog = await screen.findByRole('dialog', { name: 'Edit Server' })
    expect(within(editDialog).getByTestId('desktop-server-edit-form-grid')).toBeInTheDocument()
    fireEvent.change(within(editDialog).getByRole('textbox', { name: 'Base URL' }), { target: { value: 'https://openhim.example.com/v2' } })
    fireEvent.click(within(editDialog).getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      expect(
        fetchMock.mock.calls.some(
          (call) => String(call[0]).endsWith('/api/v1/servers/4') && (call[1] as RequestInit | undefined)?.method === 'PUT',
        ),
      ).toBe(true)
    })
  })
})
