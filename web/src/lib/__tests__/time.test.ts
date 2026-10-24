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
import { describe, expect, test } from 'vitest'

import { getCalendarMonthRange, isCurrentCalendarMonthRange } from '@/lib/time'

describe('getCalendarMonthRange', () => {
  test('starts at the first local day of the month and ends today', () => {
    const fromDate = new Date(2026, 9, 24, 20, 15, 30)
    const range = getCalendarMonthRange(fromDate)

    expect(range.start.getTime()).toBe(
      new Date(2026, 9, 1, 0, 0, 0, 0).getTime()
    )
    expect(range.end.getTime()).toBe(
      new Date(2026, 9, 24, 23, 59, 59, 999).getTime()
    )
  })

  test('matches only the current month bounds', () => {
    const fromDate = new Date(2026, 9, 24, 8, 0, 0)
    const range = getCalendarMonthRange(fromDate)
    expect(isCurrentCalendarMonthRange(range.start, range.end, fromDate)).toBe(
      true
    )
    expect(
      isCurrentCalendarMonthRange(
        new Date(2026, 8, 1, 0, 0, 0, 0),
        range.end,
        fromDate
      )
    ).toBe(false)
  })
})
