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
import { getDashboardChartColors } from '@/features/dashboard/lib/charts'
import type { HourlyQuotaPoint } from '@/features/dashboard/types'
import { getEndOfDay, getStartOfDay } from '@/lib/time'

export const CUMULATIVE_COMPARE_LOOKBACK_DAYS = 30

const HOURS_IN_DAY = 24

export type CumulativeDayKind = 'today' | 'yesterday' | 'compare'

export interface CumulativeConsumptionPoint {
  hour: string
  hourIndex: number
  series: string
  kind: CumulativeDayKind
  quota: number
  hourly: number
}

export interface CumulativeConsumptionLabels {
  today: string
  yesterday: string
}

export function formatCumulativeDayLabel(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function getCumulativeCompareBounds(now: Date): {
  min: Date
  max: Date
} {
  const today = getStartOfDay(now)
  const min = new Date(today)
  min.setDate(min.getDate() - CUMULATIVE_COMPARE_LOOKBACK_DAYS)
  const max = new Date(today)
  max.setDate(max.getDate() - 2)
  return { min, max }
}

export function isSelectableCompareDay(date: Date, now: Date): boolean {
  const day = getStartOfDay(date).getTime()
  const bounds = getCumulativeCompareBounds(now)
  return day >= bounds.min.getTime() && day <= bounds.max.getTime()
}

export function getCumulativeQueryRange(now: Date): {
  start_timestamp: number
  end_timestamp: number
} {
  const start = getCumulativeCompareBounds(now).min
  const end = getEndOfDay(now)
  return {
    start_timestamp: Math.floor(start.getTime() / 1000),
    end_timestamp: Math.floor(end.getTime() / 1000),
  }
}

function localDayKey(date: Date): string {
  return `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`
}

function formatHourLabel(hour: number): string {
  return `${String(hour).padStart(2, '0')}:00`
}

function shiftLocalDays(date: Date, days: number): Date {
  const next = getStartOfDay(date)
  next.setDate(next.getDate() + days)
  return next
}

function accumulateDay(
  rows: HourlyQuotaPoint[],
  day: Date,
  capAt?: Date
): number[] {
  const buckets = Array.from({ length: HOURS_IN_DAY }, () => 0)
  const dayKey = localDayKey(day)
  for (const row of rows) {
    if (!Number.isFinite(row.created_at)) continue
    const instant = new Date(row.created_at * 1000)
    if (localDayKey(instant) !== dayKey) continue
    if (capAt && instant.getTime() > capAt.getTime()) continue
    const quota = Number.isFinite(row.quota) ? row.quota : 0
    buckets[instant.getHours()] += quota
  }
  return buckets
}

function toCumulativeSeries(
  buckets: number[],
  lastHour: number,
  series: string,
  kind: CumulativeDayKind
): CumulativeConsumptionPoint[] {
  const end = Math.min(HOURS_IN_DAY - 1, Math.max(0, lastHour))
  const points: CumulativeConsumptionPoint[] = []
  let cumulative = 0
  for (let hour = 0; hour <= end; hour += 1) {
    const hourly = buckets[hour] ?? 0
    cumulative += hourly
    points.push({
      hour: formatHourLabel(hour),
      hourIndex: hour,
      series,
      kind,
      quota: cumulative,
      hourly,
    })
  }
  return points
}

export function buildCumulativeConsumptionSeries(input: {
  points: HourlyQuotaPoint[]
  now: Date
  compareDate?: Date | null
  labels: CumulativeConsumptionLabels
}): CumulativeConsumptionPoint[] {
  const today = accumulateDay(input.points, input.now, input.now)
  const yesterday = accumulateDay(input.points, shiftLocalDays(input.now, -1))
  const series = [
    ...toCumulativeSeries(
      today,
      input.now.getHours(),
      input.labels.today,
      'today'
    ),
    ...toCumulativeSeries(
      yesterday,
      HOURS_IN_DAY - 1,
      input.labels.yesterday,
      'yesterday'
    ),
  ]

  if (
    input.compareDate &&
    isSelectableCompareDay(input.compareDate, input.now)
  ) {
    const compareDay = getStartOfDay(input.compareDate)
    series.push(
      ...toCumulativeSeries(
        accumulateDay(input.points, compareDay),
        HOURS_IN_DAY - 1,
        formatCumulativeDayLabel(compareDay),
        'compare'
      )
    )
  }

  return series
}

export function buildCumulativeConsumptionSpec(input: {
  points: CumulativeConsumptionPoint[]
  resolvedTheme: string
  formatQuotaValue: (quota: number) => string
  formatPoint: (point: CumulativeConsumptionPoint) => string
}) {
  const seriesOrder: string[] = []
  let maxQuota = 0
  for (const point of input.points) {
    if (!seriesOrder.includes(point.series)) seriesOrder.push(point.series)
    if (point.quota > maxQuota) maxQuota = point.quota
  }
  const palette = getDashboardChartColors(Math.max(seriesOrder.length, 1))
  const textColor =
    input.resolvedTheme === 'dark'
      ? 'rgba(255, 255, 255, 0.68)'
      : 'rgba(15, 23, 42, 0.58)'
  const gridColor =
    input.resolvedTheme === 'dark'
      ? 'rgba(255, 255, 255, 0.12)'
      : 'rgba(15, 23, 42, 0.12)'

  return {
    type: 'line' as const,
    data: [{ id: 'cumulative-consumption', values: input.points }],
    xField: 'hour',
    yField: 'quota',
    seriesField: 'series',
    color: palette.slice(0, Math.max(seriesOrder.length, 1)),
    legends: { visible: true, orient: 'bottom' as const },
    line: {
      style: { lineWidth: 2, curveType: 'monotone' as const },
    },
    point: {
      visible: true,
      style: { size: 3 },
    },
    tooltip: {
      dimension: {
        content: [
          {
            key: (datum: CumulativeConsumptionPoint) => datum.series,
            value: (datum: CumulativeConsumptionPoint) =>
              input.formatPoint(datum),
          },
        ],
      },
      mark: {
        content: [
          {
            key: (datum: CumulativeConsumptionPoint) => datum.series,
            value: (datum: CumulativeConsumptionPoint) =>
              input.formatPoint(datum),
          },
        ],
      },
    },
    axes: [
      {
        orient: 'bottom' as const,
        label: {
          style: { fill: textColor, fontSize: 10 },
          autoLimit: true,
        },
        tick: { visible: false },
      },
      {
        orient: 'left' as const,
        min: 0,
        ...(maxQuota === 0 ? { max: 1 } : {}),
        label: {
          formatMethod: (value: number | string) => {
            const numeric = Number(value)
            if (!Number.isFinite(numeric)) return String(value)
            return input.formatQuotaValue(numeric)
          },
          style: { fill: textColor, fontSize: 10 },
        },
        grid: {
          visible: true,
          style: { lineDash: [3, 3], stroke: gridColor },
        },
      },
    ],
    background: { fill: 'transparent' },
    animation: false,
  }
}
