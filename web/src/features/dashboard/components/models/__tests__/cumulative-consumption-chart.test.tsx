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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { formatCumulativeDayLabel } from '@/features/dashboard/lib/cumulative-consumption'
import { useAuthStore } from '@/stores/auth-store'

import { CumulativeConsumptionChart } from '../cumulative-consumption-chart'

const { getCumulativeQuota } = vi.hoisted(() => ({
  getCumulativeQuota: vi.fn(),
}))

vi.mock('@/features/dashboard/api', () => ({
  getCumulativeQuota: (...args: unknown[]) => getCumulativeQuota(...args),
}))

vi.mock('@visactor/react-vchart', () => ({
  VChart: () => null,
}))

vi.mock('@/lib/use-chart-theme', () => ({
  useChartTheme: () => ({ resolvedTheme: 'light', themeReady: true }),
}))

function renderChart(username?: string) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <CumulativeConsumptionChart username={username} />
    </QueryClientProvider>
  )
}

describe('cumulative consumption chart controls', () => {
  afterEach(() => {
    getCumulativeQuota.mockReset()
    useAuthStore.getState().auth.setUser(null)
  })

  test('defaults to all models and can switch to one model', async () => {
    const user = userEvent.setup()
    getCumulativeQuota.mockResolvedValue({
      models: ['claude', 'gpt-4o'],
      points: [],
    })

    renderChart()

    expect(
      screen.getByRole('region', { name: 'Cumulative consumption' })
    ).toBeVisible()
    await waitFor(() => expect(getCumulativeQuota).toHaveBeenCalled())
    expect(getCumulativeQuota.mock.calls[0]?.[0]).not.toHaveProperty(
      'model_name'
    )
    expect(getCumulativeQuota.mock.calls[0]?.[1]).toBe(false)
    expect(screen.getByRole('combobox')).toHaveValue('All models')

    await user.click(screen.getByRole('combobox'))
    await user.click(await screen.findByRole('option', { name: 'gpt-4o' }))

    await waitFor(() => {
      expect(getCumulativeQuota).toHaveBeenCalledWith(
        expect.objectContaining({ model_name: 'gpt-4o' }),
        false
      )
    })
    expect(screen.getByRole('combobox')).toHaveValue('gpt-4o')
  })

  test('sends the dashboard username only for an admin', async () => {
    getCumulativeQuota.mockResolvedValue({ models: [], points: [] })
    useAuthStore.getState().auth.setUser({
      id: 1,
      username: 'root',
      role: 10,
    })

    renderChart('  alice  ')

    await waitFor(() => expect(getCumulativeQuota).toHaveBeenCalled())
    expect(getCumulativeQuota).toHaveBeenCalledWith(
      expect.objectContaining({ username: 'alice' }),
      true
    )
    expect(
      screen.getByRole('button', { name: 'Compare another day' })
    ).toBeVisible()
  })

  test('shows the load error instead of an empty curve', async () => {
    getCumulativeQuota.mockRejectedValue(new Error('down'))

    renderChart('alice')

    expect(
      await screen.findByText('Failed to load cumulative consumption')
    ).toBeVisible()
    expect(getCumulativeQuota.mock.calls[0]?.[0]).not.toHaveProperty('username')
  })

  test('lets you compare a day in the past month and clear it', async () => {
    const user = userEvent.setup()
    getCumulativeQuota.mockResolvedValue({ models: [], points: [] })
    renderChart()

    await user.click(
      screen.getByRole('button', { name: 'Compare another day' })
    )

    const now = new Date()
    const yesterday = new Date(now)
    yesterday.setDate(now.getDate() - 1)
    const earlier = new Date(now)
    earlier.setDate(now.getDate() - 2)

    const dayCell = (date: Date) =>
      document.querySelector<HTMLElement>(
        `[data-day="${formatCumulativeDayLabel(date)}"]`
      )

    const yesterdayCell = dayCell(yesterday)
    if (yesterdayCell) {
      expect(yesterdayCell).toHaveAttribute('data-disabled', 'true')
    }
    const earlierCell = dayCell(earlier)
    expect(earlierCell).not.toHaveAttribute('data-disabled', 'true')
    const earlierButton = earlierCell?.querySelector('button')
    if (!earlierButton) throw new Error('missing comparison day')

    await user.click(earlierButton)

    expect(
      screen.getByRole('button', {
        name: `Compare another day: ${formatCumulativeDayLabel(earlier)}`,
      })
    ).toBeVisible()

    await user.click(
      screen.getByRole('button', { name: 'Clear comparison day' })
    )
    expect(
      screen.getByRole('button', { name: 'Compare another day' })
    ).toBeVisible()
  })
})
