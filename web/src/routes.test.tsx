import React from 'react'
import { cleanup, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { afterEach, describe, expect, it } from 'vitest'
import { createAppRouter } from './routes'

function renderWithRouter(initialPath: string) {
  const router = createAppRouter([initialPath])
  const queryClient = new QueryClient()

  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
}

afterEach(() => {
  cleanup()
})

describe('web routes baseline', () => {
  it('renders login route', async () => {
    renderWithRouter('/login')
    expect(await screen.findByRole('heading', { name: 'BasePro Web', level: 1 })).toBeInTheDocument()
  })

  it('renders dashboard route', async () => {
    renderWithRouter('/dashboard')
    expect(await screen.findByRole('heading', { name: 'Dashboard', level: 1 })).toBeInTheDocument()
  })

  it('renders NotFound for unknown routes', async () => {
    renderWithRouter('/missing')
    expect(await screen.findByRole('heading', { name: 'Not Found', level: 1 })).toBeInTheDocument()
  })
})
