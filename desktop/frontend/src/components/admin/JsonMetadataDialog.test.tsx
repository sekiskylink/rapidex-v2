import React from 'react'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { JsonMetadataDialog } from './JsonMetadataDialog'

describe('JsonMetadataDialog', () => {
  it('handles empty metadata gracefully', () => {
    render(<JsonMetadataDialog open metadata={null} onClose={vi.fn()} />)
    expect(screen.getByText('No metadata available.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Copy' })).toBeDisabled()
  })

  it('handles invalid JSON metadata gracefully', () => {
    render(<JsonMetadataDialog open metadata="{bad-json" onClose={vi.fn()} />)
    expect(screen.getByText('Metadata is not valid JSON.')).toBeInTheDocument()
    expect(screen.getByText('{bad-json')).toBeInTheDocument()
  })

  it('copies pretty-printed metadata', async () => {
    const writeText = vi.fn(async () => undefined)
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      configurable: true,
    })

    render(<JsonMetadataDialog open metadata={{ user: 'alice' }} onClose={vi.fn()} />)
    fireEvent.click(screen.getByRole('button', { name: 'Copy' }))

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith(expect.stringContaining('"user": "alice"'))
    })
  })
})
