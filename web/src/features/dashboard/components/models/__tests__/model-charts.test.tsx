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
import { describe, expect, test, vi } from 'vitest'

import { ModelCharts } from '../model-charts'

vi.mock('@visactor/react-vchart', () => ({
  VChart: () => <div data-testid='range-chart' />,
}))

vi.mock('@visactor/vchart', () => ({
  ThemeManager: { setCurrentTheme: vi.fn() },
}))

vi.mock('@/context/theme-provider', () => ({
  useTheme: () => ({ resolvedTheme: 'light' }),
}))

vi.mock('@/context/theme-customization-provider', () => ({
  useThemeCustomization: () => ({
    customization: { preset: 'default', radius: 'md' },
  }),
}))

vi.mock('@/lib/theme-radius', () => ({
  useThemeRadiusPx: () => 8,
}))

vi.mock('../hourly-comparison-chart', () => ({
  HourlyComparisonChart: (props: { kind: string; username?: string }) => (
    <div data-testid='day-comparison'>
      {props.kind}:{props.username}
    </div>
  ),
}))

describe('model call analytics trend view', () => {
  test('shows the day comparison for the call trend by default and can switch to the time range chart', async () => {
    const user = userEvent.setup()
    render(<ModelCharts data={[]} username='alice' />)

    expect(await screen.findByTestId('day-comparison')).toHaveTextContent(
      'calls:alice'
    )
    expect(screen.queryByTestId('range-chart')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Time range' }))

    expect(screen.queryByTestId('day-comparison')).not.toBeInTheDocument()
    expect(await screen.findByTestId('range-chart')).toBeVisible()
  })

  test('hides the trend view switch on the other analytics tabs', async () => {
    const user = userEvent.setup()
    render(<ModelCharts data={[]} />)

    await user.click(
      screen.getByRole('button', { name: 'Call Count Distribution' })
    )

    expect(
      screen.queryByRole('button', { name: 'Day comparison' })
    ).not.toBeInTheDocument()
    expect(screen.queryByTestId('day-comparison')).not.toBeInTheDocument()
  })
})
