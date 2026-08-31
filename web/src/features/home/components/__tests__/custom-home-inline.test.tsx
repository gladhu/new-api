/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, test, vi } from 'vitest'

const useStatusMock = vi.fn()
const getCachedStatusMock = vi.fn()

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => useStatusMock(),
  getCachedStatus: () => getCachedStatusMock(),
}))

const { CustomHomeInline } = await import('../custom-home-inline')

const HERO_API_HTML = `
<input type="text" class="hero-api-input" id="hero-api-url" readonly value="" aria-label="API 地址" />
<button type="button" class="hero-api-copy" id="hero-api-copy" aria-label="复制 API 地址">复制</button>
`

describe('custom home inline API base URL', () => {
  beforeEach(() => {
    useStatusMock.mockReturnValue({
      status: { server_address: 'https://api.example.com' },
    })
    getCachedStatusMock.mockReturnValue(null)
  })

  test('shows a flat server_address from useStatus', () => {
    render(<CustomHomeInline html={HERO_API_HTML} />)

    expect((screen.getByLabelText('API 地址') as HTMLInputElement).value).toBe(
      'https://api.example.com'
    )
  })

  test('shows nested data.server_address like the original /api/status body', () => {
    useStatusMock.mockReturnValue({
      status: { data: { server_address: 'https://nested.example.com' } },
    })

    render(<CustomHomeInline html={HERO_API_HTML} />)

    expect((screen.getByLabelText('API 地址') as HTMLInputElement).value).toBe(
      'https://nested.example.com'
    )
  })

  test('copying writes the visible base URL', async () => {
    const user = userEvent.setup()
    render(<CustomHomeInline html={HERO_API_HTML} />)

    await user.click(screen.getByRole('button', { name: '复制 API 地址' }))

    expect(await navigator.clipboard.readText()).toBe('https://api.example.com')
  })

  test('keeps the base URL after home html is replaced on refresh', () => {
    const { rerender } = render(<CustomHomeInline html={HERO_API_HTML} />)

    expect((screen.getByLabelText('API 地址') as HTMLInputElement).value).toBe(
      'https://api.example.com'
    )

    rerender(
      <CustomHomeInline
        html={`${HERO_API_HTML}<span data-testid="home-refresh-marker"></span>`}
      />
    )

    expect((screen.getByLabelText('API 地址') as HTMLInputElement).value).toBe(
      'https://api.example.com'
    )
    expect(screen.getByTestId('home-refresh-marker')).toBeInTheDocument()
  })

  test('copy still works after home html is replaced on refresh', async () => {
    const user = userEvent.setup()
    const { rerender } = render(<CustomHomeInline html={HERO_API_HTML} />)

    rerender(
      <CustomHomeInline
        html={`${HERO_API_HTML}<span data-testid="home-refresh-marker"></span>`}
      />
    )

    await user.click(screen.getByRole('button', { name: '复制 API 地址' }))

    expect(await navigator.clipboard.readText()).toBe('https://api.example.com')
  })
})
