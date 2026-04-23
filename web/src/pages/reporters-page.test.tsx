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

function buildReporter() {
  return {
    id: 11,
    uid: 'rep-11',
    name: 'Alice Reporter',
    telephone: '+256700000001',
    whatsapp: '+256700000001',
    telegram: '@alice',
    orgUnitId: 2,
    reportingLocation: 'Kampala Health Centre',
    districtId: 9,
    totalReports: 12,
    lastReportingDate: '2026-04-10T09:00:00Z',
    smsCode: '1234',
    smsCodeExpiresAt: '2026-04-11T09:00:00Z',
    mtuuid: 'mt-uuid-11',
    synced: true,
    rapidProUuid: 'rapidpro-11',
    isActive: true,
    createdAt: '2026-04-01T08:00:00Z',
    updatedAt: '2026-04-12T08:00:00Z',
    lastLoginAt: '2026-04-13T08:00:00Z',
    groups: ['Lead'],
  }
}

function buildRapidProDetails() {
  return {
    reporter: buildReporter(),
    found: true,
    contact: {
      uuid: 'rapidpro-11',
      name: 'Alice Reporter',
      status: 'active',
      language: 'eng',
      urns: ['tel:+256700000001'],
      groups: [{ uuid: 'group-lead', name: 'Lead' }],
      fields: { Facility: 'Kampala Health Centre' },
      flow: { uuid: 'flow-1', name: 'Registration' },
      createdOn: '2026-04-01T08:00:00Z',
      modifiedOn: '2026-04-12T08:00:00Z',
      lastSeenOn: '2026-04-13T08:00:00Z',
    },
  }
}

function buildChatHistory() {
  return {
    reporter: buildReporter(),
    found: true,
    items: [
      {
        id: 1,
        direction: 'incoming',
        type: 'text',
        status: 'handled',
        visibility: 'visible',
        text: 'Hello there',
        urn: 'tel:+256700000001',
        createdOn: '2026-04-12T08:00:00Z',
        sentOn: '',
        modifiedOn: '2026-04-12T08:00:00Z',
      },
      {
        id: 2,
        direction: 'outgoing',
        type: 'text',
        status: 'sent',
        visibility: 'visible',
        text: 'Thanks for reporting',
        urn: 'tel:+256700000001',
        channel: { uuid: 'chan-1', name: 'Vonage' },
        flow: { uuid: 'flow-1', name: 'Registration' },
        createdOn: '2026-04-12T08:05:00Z',
        sentOn: '2026-04-12T08:05:30Z',
        modifiedOn: '2026-04-12T08:05:30Z',
      },
    ],
  }
}

describe('reporters page', () => {
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

  it('renders reporters grid rows from mocked API', async () => {
    authenticate(['reporters.read'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path.includes('/reporters?')) {
        return { items: [buildReporter()], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path === '/reporter-groups/options') {
        return { items: [{ id: 1, name: 'Lead' }] }
      }
      if (path.includes('/orgunits?')) {
        return { items: [{ id: 2, name: 'Kampala Health Centre' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      return {}
    })

    renderRoute('/reporters')

    expect(await screen.findByRole('heading', { name: 'Reporters', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('Alice Reporter')).toBeInTheDocument()
    expect(screen.getByText('Kampala Health Centre')).toBeInTheDocument()
  })

  it('create reporter omits backend-managed payload fields and hides backend-managed inputs', async () => {
    authenticate(['reporters.read', 'reporters.write'])
    let createPayload: Record<string, unknown> | null = null
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/reporters?')) {
        return { items: [], totalCount: 0, page: 1, pageSize: 25 }
      }
      if (path === '/reporter-groups/options') {
        return { items: [{ id: 1, name: 'Lead' }] }
      }
      if (path.includes('/orgunits?') && path.includes('rootsOnly=true')) {
        return { items: [{ id: 9, name: 'Kampala District', hasChildren: true, path: '/UG/Kampala/', displayPath: 'Uganda' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path.includes('/orgunits?') && path.includes('parentId=9')) {
        return { items: [{ id: 2, name: 'Kampala Health Centre', hasChildren: false, path: '/UG/Kampala/Kampala Health Centre/', displayPath: 'Uganda / Kampala District' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path === '/orgunits?page=0&pageSize=200') {
        return { items: [{ id: 2, name: 'Kampala Health Centre' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path.includes('/orgunits?') && path.includes('search=Kampala') && path.includes('leafOnly=true')) {
        return { items: [{ id: 2, name: 'Kampala Health Centre', displayPath: 'Uganda / Kampala District' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path === '/reporters' && init?.method === 'POST') {
        createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { id: 22 }
      }
      return {}
    })

    renderRoute('/reporters')

    fireEvent.click(await screen.findByRole('button', { name: 'New Reporter' }))
    const dialog = await screen.findByRole('dialog', { name: 'New Reporter' })
    expect(within(dialog).queryByLabelText('SMS Code')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('MT UUID')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Last Reporting Date')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Created At')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Updated At')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Last Login At')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('RapidPro UUID')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Reporting Location')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('District')).not.toBeInTheDocument()

    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Name' }), { target: { value: 'Alice Reporter' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Telephone' }), { target: { value: '+256700000001' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Browse hierarchy' }))
    const browser = await screen.findByRole('dialog', { name: 'Browse Facility Hierarchy' })
    fireEvent.click(await within(browser).findByRole('button', { name: 'Kampala District Browse children' }))
    expect(await within(browser).findByText('Uganda / Kampala District')).toBeInTheDocument()
    fireEvent.click(await within(browser).findByRole('button', { name: 'Kampala Health Centre Select facility' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Create' }))

    await waitFor(() => expect(createPayload).not.toBeNull())
    expect(createPayload).toMatchObject({
      name: 'Alice Reporter',
      telephone: '+256700000001',
      orgUnitId: 2,
    })
    expect(createPayload).not.toHaveProperty('smsCode')
    expect(createPayload).not.toHaveProperty('mtuuid')
    expect(createPayload).not.toHaveProperty('rapidProUuid')
    expect(createPayload).not.toHaveProperty('uid')
    expect(createPayload).not.toHaveProperty('totalReports')
  })

  it('edit reporter shows a facility summary and keeps backend-managed fields out of the payload', async () => {
    authenticate(['reporters.read', 'reporters.write'])
    let updatePayload: Record<string, unknown> | null = null
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/reporters?')) {
        return { items: [buildReporter()], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path === '/reporter-groups/options') {
        return { items: [{ id: 1, name: 'Lead' }] }
      }
      if (path.includes('/orgunits?') && path.includes('rootsOnly=true')) {
        return { items: [{ id: 9, name: 'Kampala District', hasChildren: true, path: '/UG/Kampala/' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path.includes('/orgunits?') && path.includes('parentId=9')) {
        return { items: [{ id: 2, name: 'Kampala Health Centre', hasChildren: false, path: '/UG/Kampala/Kampala Health Centre/' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path === '/orgunits?page=0&pageSize=200') {
        return { items: [{ id: 2, name: 'Kampala Health Centre' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path === '/reporters/11' && init?.method === 'PUT') {
        updatePayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { id: 11 }
      }
      return {}
    })

    renderRoute('/reporters')

    fireEvent.click(await screen.findByLabelText('Actions for Alice Reporter'))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Edit' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit Reporter' })
    expect(within(dialog).queryByLabelText('SMS Code')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('MT UUID')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Last Reporting Date')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Created At')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Updated At')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Last Login At')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('RapidPro UUID')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Reporting Location')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('District')).not.toBeInTheDocument()
    expect(within(dialog).getByText('Facility summary')).toBeInTheDocument()
    expect(within(dialog).getByText('Facility: Kampala Health Centre')).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Browse hierarchy' }))
    const browser = await screen.findByRole('dialog', { name: 'Browse Facility Hierarchy' })
    fireEvent.click(await within(browser).findByRole('button', { name: 'Kampala District Browse children' }))
    fireEvent.click(await within(browser).findByRole('button', { name: 'Kampala Health Centre Select facility' }))
    fireEvent.change(within(dialog).getByDisplayValue('Alice Reporter'), { target: { value: 'Alice Reporter Updated' } })
    fireEvent.click(await screen.findByRole('button', { name: 'Save' }))

    await waitFor(() => expect(updatePayload).not.toBeNull())
    expect(updatePayload).toMatchObject({
      name: 'Alice Reporter Updated',
      orgUnitId: 2,
    })
    expect(updatePayload).not.toHaveProperty('smsCode')
    expect(updatePayload).not.toHaveProperty('mtuuid')
    expect(updatePayload).not.toHaveProperty('rapidProUuid')
    expect(updatePayload).not.toHaveProperty('uid')
    expect(updatePayload).not.toHaveProperty('totalReports')
    expect(updatePayload).not.toHaveProperty('lastReportingDate')
    expect(updatePayload).not.toHaveProperty('lastLoginAt')
  })

  it('supports row sync and bulk broadcast actions', async () => {
    authenticate(['reporters.read', 'reporters.write'])
    let syncPayload: Record<string, unknown> | null = null
    let broadcastPayload: Record<string, unknown> | null = null
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/reporters?')) {
        return { items: [buildReporter()], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path === '/reporter-groups/options') {
        return { items: [{ id: 1, name: 'Lead' }] }
      }
      if (path.includes('/orgunits?')) {
        return { items: [{ id: 2, name: 'Kampala Health Centre' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path === '/reporters/11/sync' && init?.method === 'POST') {
        syncPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { reporter: buildReporter(), operation: 'updated' }
      }
      if (path === '/reporters/bulk/broadcast' && init?.method === 'POST') {
        broadcastPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { reporterIds: [11], message: 'Test broadcast' }
      }
      return {}
    })

    renderRoute('/reporters')

    fireEvent.click(await screen.findByLabelText('Actions for Alice Reporter'))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Sync to RapidPro' }))
    await waitFor(() => expect(syncPayload).toEqual({}))

    fireEvent.click(screen.getByRole('checkbox', { name: 'Select all rows' }))
    fireEvent.click(screen.getByRole('button', { name: 'Broadcast to Selected' }))
    const dialog = await screen.findByRole('dialog', { name: 'Broadcast to Selected Reporters' })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Message' }), { target: { value: 'Test broadcast' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Send Broadcast' }))

    await waitFor(() => expect(broadcastPayload).not.toBeNull())
    expect(broadcastPayload).toMatchObject({
      reporterIds: [11],
      text: 'Test broadcast',
    })
  })

  it('queues jurisdiction broadcasts from the top send message dialog', async () => {
    authenticate(['reporters.read', 'reporters.write'])
    let queuePayload: Record<string, unknown> | null = null
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/reporters?')) {
        return { items: [buildReporter()], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path === '/reporter-groups/options') {
        return { items: [{ id: 1, name: 'Lead' }] }
      }
      if (path === '/orgunits?page=0&pageSize=200') {
        return { items: [{ id: 9, name: 'Kampala District', path: '/UG/Kampala/', displayPath: 'Uganda' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path.includes('/orgunits?page=0&pageSize=20&search=Kampala')) {
        return { items: [{ id: 9, name: 'Kampala District', path: '/UG/Kampala/', displayPath: 'Uganda' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path === '/reporters/broadcasts' && init?.method === 'POST') {
        queuePayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { status: 'queued', message: 'Reporter broadcast queued. Delivery will continue in the background.', broadcast: { id: 31, matchedCount: 4, status: 'queued' } }
      }
      return {}
    })

    renderRoute('/reporters')

    fireEvent.click(await screen.findByRole('button', { name: 'Send Message' }))
    const dialog = await screen.findByRole('dialog', { name: 'Send Message' })
    const orgUnitsInput = within(dialog).getByRole('combobox', { name: 'Organisation Units' })
    fireEvent.focus(orgUnitsInput)
    fireEvent.keyDown(orgUnitsInput, { key: 'ArrowDown' })
    fireEvent.keyDown(orgUnitsInput, { key: 'Enter' })
    const reporterGroupInput = within(dialog).getByRole('combobox', { name: 'Reporter Group' })
    fireEvent.focus(reporterGroupInput)
    fireEvent.keyDown(reporterGroupInput, { key: 'ArrowDown' })
    fireEvent.keyDown(reporterGroupInput, { key: 'Enter' })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Message' }), { target: { value: 'Background hello' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Queue Broadcast' }))

    await waitFor(() => expect(queuePayload).not.toBeNull())
    expect(queuePayload).toMatchObject({
      orgUnitIds: [9],
      reporterGroup: 'Lead',
      text: 'Background hello',
    })
    expect(await screen.findByText(/Reporter broadcast queued/)).toBeInTheDocument()
  })

  it('selects all reporters on the current page from the header checkbox', async () => {
    authenticate(['reporters.read', 'reporters.write'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path.includes('/reporters?')) {
        return {
          items: [
            buildReporter(),
            {
              ...buildReporter(),
              id: 12,
              uid: 'rep-12',
              name: 'Bob Reporter',
              telephone: '+256700000002',
              rapidProUuid: 'rapidpro-12',
            },
          ],
          totalCount: 2,
          page: 1,
          pageSize: 25,
        }
      }
      if (path === '/reporter-groups/options') {
        return { items: [{ id: 1, name: 'Lead' }] }
      }
      if (path.includes('/orgunits?')) {
        return { items: [{ id: 2, name: 'Kampala Health Centre' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      return {}
    })

    renderRoute('/reporters')

    fireEvent.click(await screen.findByRole('checkbox', { name: 'Select all rows' }))
    expect(await screen.findByText('2 reporters selected.')).toBeInTheDocument()
  })

  it('shows informational reporter details, RapidPro details, and chat history dialogs', async () => {
    authenticate(['reporters.read', 'reporters.write'])
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/reporters?')) {
        return { items: [buildReporter()], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path === '/reporter-groups/options') {
        return { items: [{ id: 1, name: 'Lead' }] }
      }
      if (path.includes('/orgunits?')) {
        return { items: [{ id: 2, name: 'Kampala Health Centre' }, { id: 9, name: 'Kampala District' }], totalCount: 2, page: 1, pageSize: 25 }
      }
      if (path === '/reporters/11/rapidpro-contact' && (!init?.method || init.method === 'GET')) {
        return buildRapidProDetails()
      }
      if (path === '/reporters/11/chat-history' && (!init?.method || init.method === 'GET')) {
        return buildChatHistory()
      }
      return {}
    })

    renderRoute('/reporters')

    fireEvent.click(await screen.findByLabelText('Actions for Alice Reporter'))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View details' }))
    const detailsDialog = await screen.findByRole('dialog', { name: 'Reporter Details' })
    expect(within(detailsDialog).getByText('RapidPro synced')).toBeInTheDocument()
    expect(within(detailsDialog).getByText('Contact Channels')).toBeInTheDocument()
    expect(within(detailsDialog).getAllByText('Kampala Health Centre').length).toBeGreaterThan(0)
    fireEvent.click(within(detailsDialog).getByRole('button', { name: 'Close' }))

    fireEvent.click(await screen.findByLabelText('Actions for Alice Reporter'))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'RapidPro Details' }))
    const rapidProDialog = await screen.findByRole('dialog', { name: 'RapidPro Contact Details' })
    expect(within(rapidProDialog).getByText('Language: eng')).toBeInTheDocument()
    expect(within(rapidProDialog).getByText(/Registration/)).toBeInTheDocument()
    expect(within(rapidProDialog).getByText('Facility')).toBeInTheDocument()
    fireEvent.click(within(rapidProDialog).getByRole('button', { name: 'Close' }))

    fireEvent.click(await screen.findByRole('button', { name: '+256700000001' }))
    const chatDialog = await screen.findByRole('dialog', { name: 'Reporter Chat History' })
    expect(within(chatDialog).getByText('Hello there')).toBeInTheDocument()
    expect(within(chatDialog).getByText('Thanks for reporting')).toBeInTheDocument()
    expect(within(chatDialog).getByText('Vonage')).toBeInTheDocument()
  })

  it('shows actionable sync validation detail in the error banner', async () => {
    authenticate(['reporters.read', 'reporters.write'])
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/reporters?')) {
        return { items: [buildReporter()], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path === '/reporter-groups/options') {
        return { items: [{ id: 1, name: 'Lead' }] }
      }
      if (path.includes('/orgunits?')) {
        return { items: [{ id: 2, name: 'Kampala Health Centre' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path === '/reporters/11/sync' && init?.method === 'POST') {
        throw {
          status: 400,
          code: 'VALIDATION_ERROR',
          message: 'validation failed',
          details: { telephone: ['must resolve to a RapidPro tel: URN'] },
          requestId: 'req-reporters-422',
        }
      }
      return {}
    })

    renderRoute('/reporters')

    fireEvent.click(await screen.findByLabelText('Actions for Alice Reporter'))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Sync to RapidPro' }))

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Unable to sync reporter.')
    expect(alert).toHaveTextContent('must resolve to a RapidPro tel: URN')
    expect(alert).toHaveTextContent('req-reporters-422')
  })
})
