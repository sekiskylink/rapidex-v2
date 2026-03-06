import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { createMemoryHistory, createRootRouteWithContext, createRoute, createRouter, RouterProvider } from '@tanstack/react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AppDataGrid, type AppDataGridFetchParams } from './AppDataGrid'
import { defaultSettings, type AppSettings, type SaveSettingsPatch, type SettingsStore } from '../../settings/types'

vi.mock('@mui/x-data-grid', () => {
  return {
    DataGrid: (props: Record<string, any>) => (
      <div>
        <div data-testid="page-size">{String(props.paginationModel.pageSize)}</div>
        <div data-testid="username-visibility">{String(props.columnVisibilityModel?.username ?? true)}</div>
        <div data-testid="pinned-right">{String((props.pinnedColumns?.right ?? []).join(','))}</div>
        <button
          type="button"
          onClick={() =>
            props.onPaginationModelChange({
              page: props.paginationModel.page + 1,
              pageSize: props.paginationModel.pageSize,
            })
          }
        >
          next-page
        </button>
        <button type="button" onClick={() => props.onSortModelChange([{ field: 'username', sort: 'desc' }])}>
          sort-username
        </button>
        <button
          type="button"
          onClick={() =>
            props.onFilterModelChange({
              items: [{ field: 'username', operator: 'contains', value: 'john' }],
            })
          }
        >
          filter-username
        </button>
        <button
          type="button"
          onClick={() =>
            props.onPaginationModelChange({
              page: props.paginationModel.page,
              pageSize: 50,
            })
          }
        >
          set-page-size
        </button>
        <button type="button" onClick={() => props.onColumnVisibilityModelChange({ username: false })}>
          hide-username
        </button>
      </div>
    ),
  }
})

function createMockSettingsStore(seed: AppSettings): SettingsStore & {
  saveSettingsMock: ReturnType<typeof vi.fn>
} {
  let state = JSON.parse(JSON.stringify(seed)) as AppSettings

  const saveSettingsMock = vi.fn(async (patch: SaveSettingsPatch) => {
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
  })

  return {
    loadSettings: vi.fn(async () => state),
    saveSettings: saveSettingsMock,
    resetSettings: vi.fn(async () => defaultSettings),
    saveSettingsMock,
  }
}

function renderGrid(store: SettingsStore, fetchData: (params: AppDataGridFetchParams) => Promise<any>) {
  interface RouterContext {
    settingsStore: SettingsStore
  }

  const rootRoute = createRootRouteWithContext<RouterContext>()({
    component: () => (
      <AppDataGrid
        storageKey="users-table"
        columns={[{ field: 'id' }, { field: 'username' }, { field: 'actions' }]}
        fetchData={fetchData}
      />
    ),
  })
  const indexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
    component: () => null,
  })
  const router = createRouter({
    routeTree: rootRoute.addChildren([indexRoute]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
    context: { settingsStore: store },
  })

  return render(<RouterProvider router={router} />)
}

describe('AppDataGrid', () => {
  afterEach(() => {
    cleanup()
  })

  it('calls fetchData when page, sort, and filter change', async () => {
    const store = createMockSettingsStore(defaultSettings)
    const fetchData = vi.fn(async () => ({ rows: [], total: 0 }))
    renderGrid(store, fetchData)

    await waitFor(() => {
      expect(fetchData).toHaveBeenCalledWith(
        expect.objectContaining({
          page: 1,
          pageSize: 25,
        }),
      )
    })

    fireEvent.click(screen.getByRole('button', { name: 'next-page' }))
    await waitFor(() => expect(fetchData).toHaveBeenCalledWith(expect.objectContaining({ page: 2 })))

    fireEvent.click(screen.getByRole('button', { name: 'sort-username' }))
    await waitFor(() =>
      expect(fetchData).toHaveBeenCalledWith(
        expect.objectContaining({
          sortModel: [{ field: 'username', sort: 'desc' }],
        }),
      ),
    )

    fireEvent.click(screen.getByRole('button', { name: 'filter-username' }))
    await waitFor(() =>
      expect(fetchData).toHaveBeenCalledWith(
        expect.objectContaining({
          filterModel: expect.objectContaining({
            items: [expect.objectContaining({ field: 'username', value: 'john' })],
          }),
        }),
      ),
    )
  })

  it('persists page size and column visibility per storage key', async () => {
    const store = createMockSettingsStore(defaultSettings)
    const fetchData = vi.fn(async () => ({ rows: [], total: 0 }))
    const view = renderGrid(store, fetchData)

    await waitFor(() => expect(fetchData).toHaveBeenCalled())

    fireEvent.click(screen.getByRole('button', { name: 'set-page-size' }))
    fireEvent.click(screen.getByRole('button', { name: 'hide-username' }))

    await waitFor(() =>
      expect(store.saveSettingsMock).toHaveBeenCalledWith(
        expect.objectContaining({
          tablePrefs: expect.objectContaining({
            'users-table': expect.objectContaining({
              pageSize: 50,
              columnVisibility: expect.objectContaining({ username: false }),
            }),
          }),
        }),
      ),
    )

    view.unmount()
    renderGrid(store, fetchData)

    await waitFor(() => {
      expect(screen.getByTestId('page-size')).toHaveTextContent('50')
      expect(screen.getByTestId('username-visibility')).toHaveTextContent('false')
    })
  })

  it('pins actions column to the right by default when available', async () => {
    const store = createMockSettingsStore(defaultSettings)
    const fetchData = vi.fn(async () => ({ rows: [], total: 0 }))
    renderGrid(store, fetchData)

    await waitFor(() => {
      expect(fetchData).toHaveBeenCalled()
      expect(screen.getByTestId('pinned-right')).toHaveTextContent('actions')
    })
  })
})
