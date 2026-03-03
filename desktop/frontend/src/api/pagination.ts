import type { GridFilterModel, GridSortModel } from '@mui/x-data-grid'

export interface PaginatedResponse<T> {
  items: T[]
  totalCount: number
  page: number
  pageSize: number
}

export function buildServerQuery(params: {
  page: number
  pageSize: number
  sortModel: GridSortModel
  filterModel: GridFilterModel
}) {
  const query = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
  })

  const firstSort = params.sortModel[0]
  if (firstSort?.field && firstSort.sort) {
    query.set('sort', `${firstSort.field}:${firstSort.sort}`)
  }

  const firstFilter = params.filterModel.items.find(
    (item) => item.field && item.value !== undefined && item.value !== null && String(item.value).trim() !== '',
  )
  if (firstFilter?.field && firstFilter.value !== undefined && firstFilter.value !== null) {
    query.set('filter', `${firstFilter.field}:${String(firstFilter.value).trim()}`)
  }

  return query.toString()
}
