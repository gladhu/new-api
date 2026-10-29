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

import {
  buildCumulativeConsumptionSeries,
  buildCumulativeConsumptionSpec,
  getCumulativeQueryRange,
  isSelectableCompareDay,
} from '@/features/dashboard/lib/cumulative-consumption'

const labels = { today: 'Today', yesterday: 'Yesterday' }

function unix(date: Date): number {
  return Math.floor(date.getTime() / 1000)
}

describe('cumulative consumption comparison', () => {
  const now = new Date(2026, 9, 29, 15, 30, 0)

  test('accumulates today through the current hour and keeps yesterday full', () => {
    const series = buildCumulativeConsumptionSeries({
      now,
      labels,
      points: [
        { created_at: unix(new Date(2026, 9, 29, 10, 0, 0)), quota: 100 },
        { created_at: unix(new Date(2026, 9, 29, 10, 0, 0)), quota: 20 },
        { created_at: unix(new Date(2026, 9, 29, 15, 0, 0)), quota: 50 },
        { created_at: unix(new Date(2026, 9, 29, 16, 0, 0)), quota: 999 },
        { created_at: unix(new Date(2026, 9, 28, 8, 0, 0)), quota: 10 },
        { created_at: unix(new Date(2026, 9, 28, 20, 0, 0)), quota: 40 },
        { created_at: unix(new Date(2026, 9, 27, 1, 0, 0)), quota: 5 },
      ],
    })

    const today = series.filter((point) => point.kind === 'today')
    const yesterday = series.filter((point) => point.kind === 'yesterday')

    expect(today).toHaveLength(16)
    expect(today[10]).toMatchObject({ hour: '10:00', quota: 120, hourly: 120 })
    expect(today[14]).toMatchObject({ hour: '14:00', quota: 120, hourly: 0 })
    expect(today[15]).toMatchObject({ hour: '15:00', quota: 170, hourly: 50 })
    expect(yesterday).toHaveLength(24)
    expect(yesterday[8]).toMatchObject({ hour: '08:00', quota: 10, hourly: 10 })
    expect(yesterday[20]).toMatchObject({
      hour: '20:00',
      quota: 50,
      hourly: 40,
    })
    expect(yesterday[23]).toMatchObject({ hour: '23:00', quota: 50, hourly: 0 })
    expect(series.some((point) => point.kind === 'compare')).toBe(false)
  })

  test('adds a zero-filled curve for another day inside the last month', () => {
    const series = buildCumulativeConsumptionSeries({
      now,
      labels,
      compareDate: new Date(2026, 9, 27, 18, 0, 0),
      points: [
        { created_at: unix(new Date(2026, 9, 27, 1, 0, 0)), quota: 5 },
        { created_at: unix(new Date(2026, 9, 26, 1, 0, 0)), quota: 80 },
      ],
    })

    const compare = series.filter((point) => point.kind === 'compare')
    expect(compare).toHaveLength(24)
    expect(compare[0]?.series).toBe('2026-10-27')
    expect(compare[1]).toMatchObject({ hour: '01:00', quota: 5, hourly: 5 })
    expect(compare[23]).toMatchObject({ hour: '23:00', quota: 5, hourly: 0 })
  })

  test('ignores a comparison day that is today or yesterday', () => {
    const series = buildCumulativeConsumptionSeries({
      now,
      labels,
      compareDate: new Date(2026, 9, 28, 12, 0, 0),
      points: [],
    })

    expect(series.some((point) => point.kind === 'compare')).toBe(false)
  })

  test('allows a comparison day from 30 days ago through the day before yesterday', () => {
    expect(isSelectableCompareDay(new Date(2026, 8, 29), now)).toBe(true)
    expect(isSelectableCompareDay(new Date(2026, 8, 28), now)).toBe(false)
    expect(isSelectableCompareDay(new Date(2026, 9, 27), now)).toBe(true)
    expect(isSelectableCompareDay(new Date(2026, 9, 28), now)).toBe(false)
    expect(isSelectableCompareDay(new Date(2026, 9, 29), now)).toBe(false)
  })

  test('keeps the query window inside the one-month server limit', () => {
    const range = getCumulativeQueryRange(now)
    expect(range.end_timestamp).toBeGreaterThan(range.start_timestamp)
    expect(range.end_timestamp - range.start_timestamp).toBeLessThanOrEqual(
      32 * 24 * 60 * 60
    )
  })

  test('formats the tooltip with cumulative and hourly quota and keeps a zero axis usable', () => {
    const series = buildCumulativeConsumptionSeries({
      now,
      labels,
      points: [],
    })
    const spec = buildCumulativeConsumptionSpec({
      points: series,
      resolvedTheme: 'light',
      formatQuotaValue: (quota) => `q:${quota}`,
      formatPoint: (point) => `${point.quota}/${point.hourly}`,
    })

    expect(spec.yField).toBe('quota')
    expect(spec.xField).toBe('hour')
    const leftAxis = spec.axes.find((axis) => axis.orient === 'left')
    if (!leftAxis || leftAxis.orient !== 'left') {
      throw new Error('missing left axis')
    }
    expect(leftAxis.min).toBe(0)
    expect(leftAxis.max).toBe(1)
    expect(leftAxis.label.formatMethod(12)).toBe('q:12')
    const first = series[0]
    const tooltipValue = spec.tooltip.dimension.content[0]?.value
    if (!first || !tooltipValue) {
      throw new Error('missing tooltip point')
    }
    expect(tooltipValue(first)).toBe('0/0')
  })
})
