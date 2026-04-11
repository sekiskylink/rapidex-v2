import React from 'react'
import DensityMediumRoundedIcon from '@mui/icons-material/DensityMediumRounded'
import DensitySmallRoundedIcon from '@mui/icons-material/DensitySmallRounded'
import DensityLargeRoundedIcon from '@mui/icons-material/ReorderRounded'
import {
  DataGrid,
  type GridColDef,
  type GridColumnOrderChangeParams,
  type GridColumnVisibilityModel,
  type GridDensity,
  type GridFilterModel,
  type GridInitialState,
  type GridPaginationModel,
  type GridSortModel,
  type GridValidRowModel,
} from '@mui/x-data-grid'
import type { Theme } from '@mui/material/styles'
import { useRouter } from '@tanstack/react-router'
import type { SettingsStore, TablePrefsV1 } from '../../settings/types'

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100]
const EMPTY_STICKY_RIGHT_FIELDS: string[] = []
const MAX_RIGHT_PINNED_COLUMNS = 2

const defaultTablePrefs: TablePrefsV1 = {
  version: 1,
  pageSize: 25,
  density: 'standard',
  columnVisibility: {},
  columnOrder: [],
  pinnedColumns: {
    left: [],
    right: [],
  },
}

export interface AppDataGridFetchParams {
  page: number
  pageSize: number
  sortModel: GridSortModel
  filterModel: GridFilterModel
}

export interface AppDataGridFetchResult<R extends GridValidRowModel = GridValidRowModel> {
  rows: R[]
  total: number
}

interface AppDataGridProps<R extends GridValidRowModel = GridValidRowModel> {
  columns: GridColDef<R>[]
  fetchData: (params: AppDataGridFetchParams) => Promise<AppDataGridFetchResult<R>>
  initialState?: GridInitialState
  storageKey: string
  defaultColumnVisibilityModel?: GridColumnVisibilityModel
  getRowId?: (row: R) => string | number
  settingsStore?: SettingsStore
  reloadToken?: number
  externalQueryKey?: string
  stickyRightFields?: string[]
  pinActionsToRight?: boolean
}

function moveField(fields: string[], field: string, targetIndex: number | undefined): string[] {
  const sourceIndex = fields.indexOf(field)
  if (sourceIndex < 0) {
    return fields
  }
  const next = [...fields]
  const [moved] = next.splice(sourceIndex, 1)
  const requestedIndex = typeof targetIndex === 'number' ? targetIndex : next.length
  const boundedIndex = Math.max(0, Math.min(requestedIndex, next.length))
  next.splice(boundedIndex, 0, moved)
  return next
}

function applyColumnOrder<R extends GridValidRowModel>(
  columns: GridColDef<R>[],
  persistedOrder: string[],
): GridColDef<R>[] {
  if (!persistedOrder.length) {
    return columns
  }
  const columnByField = new Map(columns.map((column) => [column.field, column] as const))
  const ordered: GridColDef<R>[] = []

  for (const field of persistedOrder) {
    const column = columnByField.get(field)
    if (column) {
      ordered.push(column)
      columnByField.delete(field)
    }
  }
  for (const column of columns) {
    if (columnByField.has(column.field)) {
      ordered.push(column)
    }
  }
  return ordered
}

function withStickyRightFields(pinnedColumns: TablePrefsV1['pinnedColumns'], stickyRightFields: string[]) {
  if (!stickyRightFields.length) {
    return pinnedColumns
  }
  const stickySet = new Set(stickyRightFields)
  const left = pinnedColumns.left.filter((field) => !stickySet.has(field))
  const right = [...pinnedColumns.right.filter((field) => !stickySet.has(field)), ...stickyRightFields].slice(-MAX_RIGHT_PINNED_COLUMNS)
  return {
    left: Array.from(new Set(left)),
    right: Array.from(new Set(right)),
  }
}

function resolveStickyRightFields<R extends GridValidRowModel>(
  columns: GridColDef<R>[],
  stickyRightFields: string[],
  shouldPinActions: boolean,
) {
  if (!shouldPinActions) {
    return stickyRightFields
  }
  if (stickyRightFields.length > 0) {
    return stickyRightFields
  }
  return columns.some((column) => column.field === 'actions') ? ['actions'] : stickyRightFields
}

function paginationModelsEqual(a: GridPaginationModel, b: GridPaginationModel) {
  return a.page === b.page && a.pageSize === b.pageSize
}

function sortModelsEqual(a: GridSortModel, b: GridSortModel) {
  if (a.length !== b.length) {
    return false
  }
  return a.every((item, index) => item.field === b[index]?.field && item.sort === b[index]?.sort)
}

function filterModelsEqual(a: GridFilterModel, b: GridFilterModel) {
  if (a.items.length !== b.items.length) {
    return false
  }

  const itemsEqual = a.items.every((item, index) => {
    const candidate = b.items[index]
    return (
      item.field === candidate?.field &&
      item.operator === candidate?.operator &&
      String(item.value ?? '') === String(candidate?.value ?? '')
    )
  })
  if (!itemsEqual) {
    return false
  }

  if ((a.logicOperator ?? 'or') !== (b.logicOperator ?? 'or')) {
    return false
  }

  const aQuick = a.quickFilterValues ?? []
  const bQuick = b.quickFilterValues ?? []
  if (aQuick.length !== bQuick.length) {
    return false
  }
  return aQuick.every((value, index) => String(value ?? '') === String(bQuick[index] ?? ''))
}

export function AppDataGrid<R extends GridValidRowModel = GridValidRowModel>({
  columns,
  fetchData,
  initialState,
  storageKey,
  defaultColumnVisibilityModel,
  getRowId,
  settingsStore,
  reloadToken,
  externalQueryKey,
  stickyRightFields,
  pinActionsToRight,
}: AppDataGridProps<R>) {
  const stickyFields = stickyRightFields ?? EMPTY_STICKY_RIGHT_FIELDS
  const [enforcedStickyFields, setEnforcedStickyFields] = React.useState(stickyFields)
  const stickyFieldsKey = stickyFields.join('\u0000')
  const router = useRouter()
  const store = settingsStore ?? router.options.context.settingsStore

  const [rows, setRows] = React.useState<R[]>([])
  const [rowCount, setRowCount] = React.useState(0)
  const [loading, setLoading] = React.useState(false)
  const [paginationModel, setPaginationModel] = React.useState<GridPaginationModel>({
    page: 0,
    pageSize: defaultTablePrefs.pageSize,
  })
  const [sortModel, setSortModel] = React.useState<GridSortModel>([])
  const [filterModel, setFilterModel] = React.useState<GridFilterModel>({ items: [] })
  const [columnVisibilityModel, setColumnVisibilityModel] = React.useState<GridColumnVisibilityModel>(
    defaultTablePrefs.columnVisibility,
  )
  const [columnOrder, setColumnOrder] = React.useState<string[]>([])
  const [density, setDensity] = React.useState<GridDensity>(defaultTablePrefs.density)
  const [pinnedColumns, setPinnedColumns] = React.useState(defaultTablePrefs.pinnedColumns)
  const [gridBorderRadius, setGridBorderRadius] = React.useState(12)
  const [hydrated, setHydrated] = React.useState(false)
  const requestIdRef = React.useRef(0)
  const lastFetchKeyRef = React.useRef('')

  React.useEffect(() => {
    let active = true
    setHydrated(false)

    store.loadSettings().then((settings) => {
      if (!active) {
        return
      }
      const persisted = settings.tablePrefs[storageKey] ?? defaultTablePrefs
      const shouldPinActions = pinActionsToRight ?? settings.uiPrefs.pinActionsColumnRight
      const resolvedStickyFields = resolveStickyRightFields(columns, shouldPinActions ? stickyFields : EMPTY_STICKY_RIGHT_FIELDS, shouldPinActions)
      setEnforcedStickyFields(resolvedStickyFields)
      setPaginationModel((current) => ({
        page: current.page,
        pageSize: PAGE_SIZE_OPTIONS.includes(persisted.pageSize) ? persisted.pageSize : defaultTablePrefs.pageSize,
      }))
      setColumnVisibilityModel({
        ...(defaultColumnVisibilityModel ?? {}),
        ...persisted.columnVisibility,
      })
      setColumnOrder(persisted.columnOrder)
      setDensity(persisted.density)
      setPinnedColumns(withStickyRightFields(persisted.pinnedColumns, resolvedStickyFields))
      setGridBorderRadius(settings.uiPrefs.dataGridBorderRadius)
      setHydrated(true)
    })

    return () => {
      active = false
    }
  }, [columns, defaultColumnVisibilityModel, pinActionsToRight, storageKey, store, stickyFieldsKey])

  const persistTablePrefs = React.useCallback(
    async (updater: (current: TablePrefsV1) => TablePrefsV1) => {
      const settings = await store.loadSettings()
      const current = settings.tablePrefs[storageKey] ?? defaultTablePrefs
      const next = updater(current)
      await store.saveSettings({
        tablePrefs: {
          ...settings.tablePrefs,
          [storageKey]: { ...next, version: 1 },
        },
      })
    },
    [storageKey, store],
  )

  React.useEffect(() => {
    if (!hydrated) {
      return
    }
    void persistTablePrefs(() => ({
      version: 1,
      pageSize: paginationModel.pageSize,
      density,
      columnVisibility: columnVisibilityModel,
      columnOrder,
      pinnedColumns,
    }))
  }, [
    hydrated,
    persistTablePrefs,
    paginationModel.pageSize,
    density,
    columnVisibilityModel,
    columnOrder,
    pinnedColumns,
  ])

  React.useEffect(() => {
    if (!hydrated) {
      return
    }
    const fetchKey = JSON.stringify({
      page: paginationModel.page + 1,
      pageSize: paginationModel.pageSize,
      sortModel,
      filterModel,
      reloadToken: reloadToken ?? null,
      externalQueryKey: externalQueryKey ?? null,
    })
    if (fetchKey === lastFetchKeyRef.current) {
      return
    }
    lastFetchKeyRef.current = fetchKey

    const requestId = ++requestIdRef.current
    setLoading(true)
    void fetchData({
      page: paginationModel.page + 1,
      pageSize: paginationModel.pageSize,
      sortModel,
      filterModel,
    })
      .then((result) => {
        if (requestId !== requestIdRef.current) {
          return
        }
        if (!Array.isArray(result.rows) || typeof result.total !== 'number') {
          return
        }
        setRows(result.rows)
        setRowCount(result.total)
      })
      .finally(() => {
        if (requestId === requestIdRef.current) {
          setLoading(false)
        }
      })
  }, [externalQueryKey, fetchData, filterModel, hydrated, paginationModel.page, paginationModel.pageSize, reloadToken, sortModel])

  React.useEffect(() => {
    if (!hydrated) {
      return
    }
    setPaginationModel((current) => (current.page === 0 ? current : { ...current, page: 0 }))
  }, [externalQueryKey, hydrated])

  const orderedColumns = React.useMemo(() => applyColumnOrder(columns, columnOrder), [columns, columnOrder])

  const DataGridAny = DataGrid as unknown as React.ComponentType<Record<string, unknown>>

  return (
    <DataGridAny
      columns={orderedColumns}
      rows={rows}
      loading={loading}
      rowCount={rowCount}
      getRowId={getRowId}
      pagination
      paginationMode="server"
      paginationModel={paginationModel}
      onPaginationModelChange={(model: GridPaginationModel) =>
        setPaginationModel((current) => (paginationModelsEqual(current, model) ? current : model))
      }
      pageSizeOptions={PAGE_SIZE_OPTIONS}
      sortingMode="server"
      sortModel={sortModel}
      onSortModelChange={(model: GridSortModel) =>
        setSortModel((current) => (sortModelsEqual(current, model) ? current : model))
      }
      filterMode="server"
      filterModel={filterModel}
      onFilterModelChange={(model: GridFilterModel) =>
        setFilterModel((current) => (filterModelsEqual(current, model) ? current : model))
      }
      density={density}
      onDensityChange={(value: GridDensity) => setDensity(value)}
      columnVisibilityModel={columnVisibilityModel}
      onColumnVisibilityModelChange={(model: GridColumnVisibilityModel) => setColumnVisibilityModel(model)}
      onColumnOrderChange={(params: GridColumnOrderChangeParams) =>
        setColumnOrder((current) => {
          const currentOrder = current.length
            ? current
            : orderedColumns.map((column) => column.field)
          return moveField(currentOrder, params.column.field, params.targetIndex)
        })
      }
      showToolbar
      slots={{
        densityCompactIcon: DensitySmallRoundedIcon,
        densityStandardIcon: DensityMediumRoundedIcon,
        densityComfortableIcon: DensityLargeRoundedIcon,
      }}
      slotProps={{
        toolbar: {
          csvOptions: {
            fileName: storageKey.replace(/[^a-z0-9_-]/gi, '_'),
          },
          printOptions: {
            disableToolbarButton: true,
          },
        },
      }}
      initialState={initialState}
      disableRowSelectionOnClick
      sx={{
        borderRadius: `${gridBorderRadius}px`,
        '& .MuiDataGrid-columnHeaderTitle': {
          fontWeight: 700,
        },
        '& .MuiDataGrid-main': {
          minWidth: 0,
        },
        '& .MuiDataGrid-virtualScroller': {
          overflowX: 'auto',
          overflowY: 'auto',
        },
        '& .MuiDataGrid-menu': {
          zIndex: (theme: Theme) => theme.zIndex.modal + 1,
        },
        '& .MuiDataGrid-pinnedColumns, & .MuiDataGrid-pinnedColumnHeaders': {
          bgcolor: 'background.paper',
        },
        '& .MuiDataGrid-cell--pinnedRight, & .MuiDataGrid-columnHeader--pinnedRight': {
          boxShadow: (theme: Theme) => `-1px 0 0 ${theme.palette.divider}`,
        },
      }}
      pinnedColumns={pinnedColumns}
      onPinnedColumnsChange={(model: { left?: string[]; right?: string[] }) =>
        setPinnedColumns(
          withStickyRightFields(
            {
              left: model.left ?? [],
              right: model.right ?? [],
            },
            enforcedStickyFields,
          ),
        )
      }
    />
  )
}
