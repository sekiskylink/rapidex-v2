import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { SnackbarProvider } from '../../ui/snackbar'
import { AppDataGrid } from './AppDataGrid'

vi.mock('@mui/x-data-grid', () => ({
  DataGrid: (props: Record<string, any>) => (
    <div>
      <div data-testid="page-size">{String(props.paginationModel.pageSize)}</div>
      <div data-testid="density">{String(props.density)}</div>
      <div data-testid="username-visible">{String(props.columnVisibilityModel?.username ?? true)}</div>
      <div data-testid="first-column">{String(props.columns?.[0]?.field ?? '')}</div>
      <div data-testid="pinned-left">{String((props.pinnedColumns?.left ?? []).join(','))}</div>
      <div data-testid="pinned-right">{String((props.pinnedColumns?.right ?? []).join(','))}</div>
      <button
        type="button"
        onClick={() => props.onPaginationModelChange({ page: props.paginationModel.page + 1, pageSize: props.paginationModel.pageSize })}
      >
        next-page
      </button>
      <button type="button" onClick={() => props.onSortModelChange([{ field: 'username', sort: 'desc' }])}>
        sort-change
      </button>
      <button
        type="button"
        onClick={() => props.onFilterModelChange({ items: [{ field: 'username', operator: 'contains', value: 'ali' }] })}
      >
        filter-change
      </button>
      <button type="button" onClick={() => props.onPaginationModelChange({ page: 0, pageSize: 50 })}>
        page-size-50
      </button>
      <button type="button" onClick={() => props.onColumnVisibilityModelChange({ username: false })}>
        hide-username
      </button>
      <button type="button" onClick={() => props.onColumnOrderChange({ column: { field: 'username' }, targetIndex: 0 })}>
        reorder-username
      </button>
      <button type="button" onClick={() => props.onDensityChange('comfortable')}>
        density-comfortable
      </button>
      <button type="button" onClick={() => props.onPinnedColumnsChange?.({ left: ['username'], right: ['id'] })}>
        pin-columns
      </button>
    </div>
  ),
}))

function renderGrid(fetchData: (params: any) => Promise<any>, storageKey = 'users-table', enablePinnedColumns = false) {
  return render(
    <SnackbarProvider>
      <AppDataGrid
        storageKey={storageKey}
        columns={[
          { field: 'id', headerName: 'ID' },
          { field: 'username', headerName: 'Username' },
          { field: 'actions', headerName: 'Actions' },
        ]}
        fetchData={fetchData}
        enablePinnedColumns={enablePinnedColumns}
      />
    </SnackbarProvider>,
  )
}

describe('AppDataGrid', () => {
  beforeEach(() => {
    window.localStorage.clear()
  })

  afterEach(() => {
    cleanup()
    window.localStorage.clear()
  })

  it('triggers fetch on page change', async () => {
    const fetchData = vi.fn(async () => ({ rows: [], total: 0 }))
    renderGrid(fetchData)

    await waitFor(() => expect(fetchData).toHaveBeenCalledWith(expect.objectContaining({ page: 1, pageSize: 25 })))

    fireEvent.click(screen.getByRole('button', { name: 'next-page' }))

    await waitFor(() => expect(fetchData).toHaveBeenCalledWith(expect.objectContaining({ page: 2 })))
  })

  it('triggers fetch on sort model change', async () => {
    const fetchData = vi.fn(async () => ({ rows: [], total: 0 }))
    renderGrid(fetchData)

    await waitFor(() => expect(fetchData).toHaveBeenCalled())
    fireEvent.click(screen.getByRole('button', { name: 'sort-change' }))

    await waitFor(() =>
      expect(fetchData).toHaveBeenCalledWith(
        expect.objectContaining({
          sortModel: [{ field: 'username', sort: 'desc' }],
        }),
      ),
    )
  })

  it('triggers fetch on filter model change', async () => {
    const fetchData = vi.fn(async () => ({ rows: [], total: 0 }))
    renderGrid(fetchData)

    await waitFor(() => expect(fetchData).toHaveBeenCalled())
    fireEvent.click(screen.getByRole('button', { name: 'filter-change' }))

    await waitFor(() =>
      expect(fetchData).toHaveBeenCalledWith(
        expect.objectContaining({
          filterModel: expect.objectContaining({
            items: [expect.objectContaining({ field: 'username', value: 'ali' })],
          }),
        }),
      ),
    )
  })

  it('restores and persists pageSize, visibility, order, density, and pinned columns', async () => {
    window.localStorage.setItem(
      'app.datagrid.users-table.v1',
      JSON.stringify({
        version: 1,
        pageSize: 50,
        columnVisibility: { username: false },
        columnOrder: ['username', 'id'],
        density: 'compact',
        pinnedColumns: { left: ['username'], right: ['id'] },
      }),
    )

    const fetchData = vi.fn(async () => ({ rows: [], total: 0 }))
    const view = renderGrid(fetchData, 'users-table', true)

    await waitFor(() => {
      expect(screen.getByTestId('page-size')).toHaveTextContent('50')
      expect(screen.getByTestId('username-visible')).toHaveTextContent('false')
      expect(screen.getByTestId('first-column')).toHaveTextContent('username')
      expect(screen.getByTestId('density')).toHaveTextContent('compact')
      expect(screen.getByTestId('pinned-left')).toHaveTextContent('username')
      expect(screen.getByTestId('pinned-right')).toHaveTextContent('id,actions')
    })

    fireEvent.click(screen.getByRole('button', { name: 'density-comfortable' }))
    fireEvent.click(screen.getByRole('button', { name: 'reorder-username' }))
    fireEvent.click(screen.getByRole('button', { name: 'pin-columns' }))
    fireEvent.click(screen.getByRole('button', { name: 'page-size-50' }))
    fireEvent.click(screen.getByRole('button', { name: 'hide-username' }))

    await waitFor(() => {
      const saved = JSON.parse(window.localStorage.getItem('app.datagrid.users-table.v1') ?? '{}') as Record<string, any>
      expect(saved.pageSize).toBe(50)
      expect(saved.columnVisibility).toEqual(expect.objectContaining({ username: false }))
      expect(saved.columnOrder).toEqual(expect.arrayContaining(['id', 'username']))
      expect(saved.density).toBe('comfortable')
      expect(saved.pinnedColumns).toEqual({ left: ['username'], right: ['id', 'actions'] })
    })

    view.unmount()
    renderGrid(fetchData, 'users-table', true)
    await waitFor(() => expect(screen.getByTestId('density')).toHaveTextContent('comfortable'))
  })

  it('pins actions column to the right by default', async () => {
    const fetchData = vi.fn(async () => ({ rows: [], total: 0 }))
    renderGrid(fetchData, 'users-table', true)

    await waitFor(() => {
      expect(fetchData).toHaveBeenCalled()
      expect(screen.getByTestId('pinned-right')).toHaveTextContent('actions')
    })
  })
})
