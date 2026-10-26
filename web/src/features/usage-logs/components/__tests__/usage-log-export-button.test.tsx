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

import { UsageLogExportButton } from '../usage-log-export-button'

describe('usage log export button', () => {
  test('downloads the simple log immediately for a non-admin', async () => {
    const user = userEvent.setup()
    const onExport = vi.fn()
    render(
      <UsageLogExportButton
        isAdmin={false}
        disabled={false}
        exporting={false}
        onExport={onExport}
      />
    )

    await user.click(screen.getByRole('button', { name: 'Export logs' }))

    expect(onExport).toHaveBeenCalledTimes(1)
    expect(onExport).toHaveBeenCalledWith('simple')
    expect(screen.queryByRole('menuitem', { name: /Detailed logs/ })).toBeNull()
  })

  test('lets an admin choose simple or detailed logs', async () => {
    const user = userEvent.setup()
    const onExport = vi.fn()
    render(
      <UsageLogExportButton
        isAdmin
        disabled={false}
        exporting={false}
        onExport={onExport}
      />
    )

    await user.click(screen.getByRole('button', { name: 'Export logs' }))
    await user.click(screen.getByRole('menuitem', { name: /Simple logs/ }))
    expect(onExport).toHaveBeenCalledWith('simple')

    await user.click(screen.getByRole('button', { name: 'Export logs' }))
    await user.click(screen.getByRole('menuitem', { name: /Detailed logs/ }))
    expect(onExport).toHaveBeenCalledWith('full')
    expect(onExport).toHaveBeenCalledTimes(2)
  })

  test('keeps the export control disabled while a download is running', () => {
    render(
      <UsageLogExportButton
        isAdmin
        disabled={false}
        exporting
        onExport={vi.fn()}
      />
    )

    expect(screen.getByRole('button', { name: 'Export logs' })).toBeDisabled()
  })
})
