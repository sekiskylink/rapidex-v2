import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { afterEach, beforeEach, describe, expect, it, vi, type MockInstance } from 'vitest'
import { clearAuthSnapshot, setAuthSnapshot } from '../auth/state'
import { API_BASE_URL_OVERRIDE_STORAGE_KEY } from '../lib/apiBaseUrl'
import * as api from '../lib/api'
import { createAppRouter } from '../routes'
import { SnackbarProvider } from '../ui/snackbar'

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

function renderRoute(path: string) {
  const router = createAppRouter([path])
  const queryClient = new QueryClient()
  return render(
    <QueryClientProvider client={queryClient}>
      <SnackbarProvider>
        <RouterProvider router={router} />
      </SnackbarProvider>
    </QueryClientProvider>,
  )
}

function authenticate(permissions: string[]) {
  setAuthSnapshot({
    isAuthenticated: true,
    accessToken: 'access-token',
    refreshToken: 'refresh-token',
    user: {
      id: 7,
      username: 'operator',
      roles: ['Staff'],
      permissions,
    },
  })
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

describe('org units page', () => {
  let apiRequestSpy: MockInstance

  beforeEach(() => {
    window.localStorage.clear()
    clearAuthSnapshot()
    vi.stubEnv('VITE_API_BASE_URL', 'http://localhost:8080/api/v1')
    window.localStorage.setItem(API_BASE_URL_OVERRIDE_STORAGE_KEY, 'http://localhost:8080/api/v1')
    apiRequestSpy = vi.spyOn(api, 'apiRequest')
  })

  afterEach(() => {
    cleanup()
    clearAuthSnapshot()
    vi.unstubAllEnvs()
    apiRequestSpy.mockRestore()
  })

  it('issues an org-unit search request and shows the matching option text', async () => {
    authenticate(['orgunits.read'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path === '/orgunits?page=0&pageSize=200') {
        return { items: [], totalCount: 0, page: 0, pageSize: 200 }
      }
      if (path === '/orgunits/sync-state') {
        return {}
      }
      if (path.includes('/orgunits?page=0&pageSize=20&search=Alpha')) {
        return { items: [buildOrgUnit()], totalCount: 1, page: 0, pageSize: 20 }
      }
      return {}
    })

    renderRoute('/orgunits')

    const searchBox = await screen.findByRole('combobox', { name: 'Find org unit' })
    fireEvent.focus(searchBox)
    fireEvent.change(searchBox, { target: { value: 'Alpha' } })

    await waitFor(() => expect(apiRequestSpy).toHaveBeenCalledWith('/orgunits?page=0&pageSize=20&search=Alpha'))
    expect(await screen.findByText('Alpha Health Centre')).toBeInTheDocument()
  })

  it('opens facility details from the browse hierarchy', async () => {
    authenticate(['orgunits.read'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path === '/orgunits?page=0&pageSize=200') {
        return { items: [], totalCount: 0, page: 0, pageSize: 200 }
      }
      if (path === '/orgunits/sync-state') {
        return {}
      }
      if (path.includes('/orgunits?page=0&pageSize=200&rootsOnly=true')) {
        return {
          items: [
            buildOrgUnit({ id: 2, name: 'Beta District', parentId: null, hierarchyLevel: 1, path: '/UG/Beta', displayPath: 'Uganda', hasChildren: true }),
            buildOrgUnit({ id: 1, name: 'Alpha District', parentId: null, hierarchyLevel: 1, path: '/UG/Alpha', displayPath: 'Uganda', hasChildren: true }),
          ],
          totalCount: 2,
          page: 0,
          pageSize: 200,
        }
      }
      return {}
    })

    renderRoute('/orgunits')

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
    authenticate(['orgunits.read'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path === '/orgunits?page=0&pageSize=200') {
        return { items: [], totalCount: 0, page: 0, pageSize: 200 }
      }
      if (path === '/orgunits/sync-state') {
        return {}
      }
      if (path.includes('/orgunits?page=0&pageSize=200&rootsOnly=true')) {
        return {
          items: [
            buildOrgUnit({ id: 2, name: 'Beta District', parentId: null, hierarchyLevel: 1, path: '/UG/Beta', displayPath: 'Uganda', hasChildren: true }),
            buildOrgUnit({ id: 1, name: 'Alpha District', parentId: null, hierarchyLevel: 1, path: '/UG/Alpha', displayPath: 'Uganda', hasChildren: true }),
          ],
          totalCount: 2,
          page: 0,
          pageSize: 200,
        }
      }
      if (path.includes('/orgunits?page=0&pageSize=200&parentId=1')) {
        return {
          items: [
            buildOrgUnit({ id: 4, name: 'Zulu Health Centre', parentId: 1, displayPath: 'Uganda / Alpha District' }),
            buildOrgUnit({ id: 3, name: 'Alpha Health Centre', parentId: 1, displayPath: 'Uganda / Alpha District' }),
          ],
          totalCount: 2,
          page: 0,
          pageSize: 200,
        }
      }
      return {}
    })

    renderRoute('/orgunits')

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
