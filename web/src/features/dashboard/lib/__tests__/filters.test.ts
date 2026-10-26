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
import { afterEach, describe, expect, test } from 'vitest'

import {
  DEFAULT_DASHBOARD_CHART_PREFERENCES,
  DASHBOARD_CHART_PREFERENCES_STORAGE_KEY,
} from '@/features/dashboard/constants'
import {
  buildQueryParams,
  detectQuickRangePreset,
  getRangeForTimeRangePreset,
  getSavedChartPreferences,
  granularityForTimeRangePreset,
} from '@/features/dashboard/lib/filters'
import { getCalendarMonthRange } from '@/lib/time'

const fromDate = new Date(2026, 9, 24, 15, 30, 0)

describe('buildQueryParams', () => {
  const timeRange = { start_timestamp: 1000, end_timestamp: 2000 }

  test('includes an exact model name for the flow query', () => {
    expect(
      buildQueryParams(timeRange, {
        time_granularity: 'day',
        username: 'alice',
        model_name: 'gpt-4o',
      })
    ).toMatchObject({
      start_timestamp: 1000,
      end_timestamp: 2000,
      default_time: 'day',
      username: 'alice',
      model_name: 'gpt-4o',
    })
  })

  test('omits a blank model name', () => {
    expect(
      buildQueryParams(timeRange, {
        time_granularity: 'day',
        model_name: '',
      }).model_name
    ).toBeUndefined()
  })
})

describe('granularityForTimeRangePreset', () => {
  test('maps one day to hour buckets', () => {
    expect(granularityForTimeRangePreset('1')).toBe('hour')
  })

  test('maps three and seven days to day buckets', () => {
    expect(granularityForTimeRangePreset('3')).toBe('day')
    expect(granularityForTimeRangePreset('7')).toBe('day')
  })

  test('maps twenty-nine days to week buckets', () => {
    expect(granularityForTimeRangePreset('29')).toBe('week')
  })

  test('maps this month to day buckets', () => {
    expect(granularityForTimeRangePreset('month')).toBe('day')
  })
})

describe('getRangeForTimeRangePreset', () => {
  test('returns an inclusive three-day calendar range', () => {
    const range = getRangeForTimeRangePreset('3', fromDate)
    expect(range.start.getTime()).toBe(
      new Date(2026, 9, 22, 0, 0, 0, 0).getTime()
    )
    expect(range.end.getTime()).toBe(
      new Date(2026, 9, 24, 23, 59, 59, 999).getTime()
    )
  })

  test('returns the current local calendar month', () => {
    const range = getRangeForTimeRangePreset('month', fromDate)
    const expected = getCalendarMonthRange(fromDate)
    expect(range.start.getTime()).toBe(expected.start.getTime())
    expect(range.end.getTime()).toBe(expected.end.getTime())
    expect(range.start.getTime()).toBe(
      new Date(2026, 9, 1, 0, 0, 0, 0).getTime()
    )
  })
})

describe('detectQuickRangePreset', () => {
  test('prefers an explicit preset over date inference', () => {
    const month = getCalendarMonthRange(fromDate)
    expect(
      detectQuickRangePreset(
        {
          start_timestamp: month.start,
          end_timestamp: month.end,
          time_range_preset: '1',
        },
        fromDate
      )
    ).toBe('1')
  })

  test('detects this month from calendar bounds', () => {
    const month = getCalendarMonthRange(fromDate)
    expect(
      detectQuickRangePreset(
        {
          start_timestamp: month.start,
          end_timestamp: month.end,
        },
        fromDate
      )
    ).toBe('month')
  })

  test('detects a three-day inclusive range', () => {
    const range = getRangeForTimeRangePreset('3', fromDate)
    expect(
      detectQuickRangePreset(
        {
          start_timestamp: range.start,
          end_timestamp: range.end,
        },
        fromDate
      )
    ).toBe('3')
  })

  test('returns null for a custom range or historical three-day window', () => {
    expect(
      detectQuickRangePreset(
        {
          start_timestamp: new Date(2026, 8, 1, 0, 0, 0, 0),
          end_timestamp: new Date(2026, 8, 10, 23, 59, 59, 999),
        },
        fromDate
      )
    ).toBeNull()
    expect(
      detectQuickRangePreset(
        {
          start_timestamp: new Date(2026, 8, 1, 0, 0, 0, 0),
          end_timestamp: new Date(2026, 8, 3, 23, 59, 59, 999),
        },
        fromDate
      )
    ).toBeNull()
    const range = getRangeForTimeRangePreset('3', fromDate)
    expect(
      detectQuickRangePreset(
        {
          start_timestamp: range.start,
          end_timestamp: range.end,
          time_range_preset: null,
        },
        fromDate
      )
    ).toBeNull()
  })
})

describe('getSavedChartPreferences', () => {
  afterEach(() => {
    window.localStorage.removeItem(DASHBOARD_CHART_PREFERENCES_STORAGE_KEY)
  })

  test('migrates a saved day count to the named preset', () => {
    window.localStorage.setItem(
      DASHBOARD_CHART_PREFERENCES_STORAGE_KEY,
      JSON.stringify({ defaultTimeRangeDays: 29 })
    )
    expect(getSavedChartPreferences().defaultTimeRange).toBe('29')
  })

  test('keeps a saved month preset', () => {
    window.localStorage.setItem(
      DASHBOARD_CHART_PREFERENCES_STORAGE_KEY,
      JSON.stringify({ defaultTimeRange: 'month' })
    )
    expect(getSavedChartPreferences().defaultTimeRange).toBe('month')
  })

  test('falls back when the saved range is unknown', () => {
    window.localStorage.setItem(
      DASHBOARD_CHART_PREFERENCES_STORAGE_KEY,
      JSON.stringify({ defaultTimeRangeDays: 12 })
    )
    expect(getSavedChartPreferences().defaultTimeRange).toBe(
      DEFAULT_DASHBOARD_CHART_PREFERENCES.defaultTimeRange
    )
  })
})
