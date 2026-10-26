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
import { afterEach, describe, expect, test, vi } from 'vitest'

import { DEFAULT_DASHBOARD_CHART_PREFERENCES } from '@/features/dashboard/constants'
import { buildDefaultDashboardFilters } from '@/features/dashboard/lib/filters'
import type { DashboardFilters } from '@/features/dashboard/types'
import { useAuthStore } from '@/stores/auth-store'

import { ModelsFilter } from '../models-filter-dialog'

if (!Element.prototype.getAnimations) {
  Element.prototype.getAnimations = () => []
}

function renderFilter(props?: {
  includeModelName?: boolean
  currentFilters?: DashboardFilters
  onFilterChange?: (filters: DashboardFilters) => void
}) {
  return render(
    <ModelsFilter
      preferences={DEFAULT_DASHBOARD_CHART_PREFERENCES}
      currentFilters={
        props?.currentFilters ??
        buildDefaultDashboardFilters(DEFAULT_DASHBOARD_CHART_PREFERENCES)
      }
      onFilterChange={props?.onFilterChange ?? (() => undefined)}
      onReset={() => undefined}
      includeModelName={props?.includeModelName}
      titleKey='Flow Filters'
      descriptionKey='Filter the traffic flow view by time range, model, and user.'
    />
  )
}

describe('flow model name filter', () => {
  afterEach(() => {
    useAuthStore.getState().auth.setUser(null)
  })

  test('shows the model name field for a non-admin flow filter', async () => {
    const user = userEvent.setup()
    renderFilter({ includeModelName: true })

    await user.click(screen.getByRole('button', { name: 'Filter' }))

    expect(screen.getByRole('textbox', { name: 'Model Name' })).toBeVisible()
    expect(screen.queryByRole('textbox', { name: 'Username' })).toBeNull()
  })

  test('keeps the model name field off the model analytics filter', async () => {
    const user = userEvent.setup()
    renderFilter()

    await user.click(screen.getByRole('button', { name: 'Filter' }))

    expect(screen.queryByRole('textbox', { name: 'Model Name' })).toBeNull()
  })

  test('applies a trimmed model name together with the username', async () => {
    const user = userEvent.setup()
    const onFilterChange = vi.fn()
    useAuthStore.getState().auth.setUser({
      id: 1,
      username: 'root',
      role: 100,
    })
    renderFilter({ includeModelName: true, onFilterChange })

    await user.click(screen.getByRole('button', { name: 'Filter' }))
    await user.type(
      screen.getByRole('textbox', { name: 'Model Name' }),
      '  gpt-4o  '
    )
    await user.type(screen.getByRole('textbox', { name: 'Username' }), 'alice')
    await user.click(screen.getByRole('button', { name: 'Apply Filters' }))

    expect(onFilterChange).toHaveBeenCalledTimes(1)
    expect(onFilterChange.mock.calls[0]?.[0]).toMatchObject({
      model_name: 'gpt-4o',
      username: 'alice',
    })
  })
})
