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
import type {
  PerformanceGroup,
  PerfModelSummary,
} from '@/features/performance-metrics/types'
import dayjs from '@/lib/dayjs'
import { formatChartTime } from '@/lib/time'

export type WeightedSuccessRate = {
  rate: number
  requests: number
}

export type MergedSuccessPoint = {
  ts: number
  success_rate: number
  request_count: number
}

export type FilledSuccessPoint = {
  ts: number
  success_rate: number | null
  request_count: number
}

const DEFAULT_BUCKET_SECONDS = 3600
const KNOWN_BUCKET_SECONDS = [3600, 300, 60] as const
const UPTIME_AXIS_MAX = 100
const UPTIME_FOCUSED_AXIS_MIN = 95
const UPTIME_MINOR_OUTAGE_AXIS_MIN = 90

export function weightedSuccessRate(
  rows: Array<
    Pick<PerfModelSummary, 'success_rate' | 'request_count' | 'success_count'>
  >
): WeightedSuccessRate {
  let requests = 0
  let successes = 0

  for (const row of rows) {
    const count = Number(row.request_count)
    if (!Number.isFinite(count) || count <= 0) continue
    requests += count
    if (Number.isFinite(row.success_count) && Number(row.success_count) >= 0) {
      successes += Number(row.success_count)
    } else if (Number.isFinite(row.success_rate)) {
      successes += (Number(row.success_rate) / 100) * count
    }
  }

  return {
    requests,
    rate: requests > 0 ? (successes / requests) * 100 : Number.NaN,
  }
}

export function mergeGroupSuccessSeries(
  groups: PerformanceGroup[]
): MergedSuccessPoint[] {
  const byTs = new Map<
    number,
    { request_count: number; success_count: number }
  >()

  for (const group of groups) {
    for (const point of group.series) {
      const requestCount = Number(point.request_count)
      const successCount = Number(point.success_count)
      const hasCounts =
        Number.isFinite(requestCount) &&
        requestCount > 0 &&
        Number.isFinite(successCount) &&
        successCount >= 0
      const current = byTs.get(point.ts) ?? {
        request_count: 0,
        success_count: 0,
      }
      if (hasCounts) {
        current.request_count += requestCount
        current.success_count += successCount
      } else if (Number.isFinite(point.success_rate)) {
        const inferredRequests =
          Number.isFinite(requestCount) && requestCount > 0 ? requestCount : 1
        current.request_count += inferredRequests
        current.success_count += (point.success_rate / 100) * inferredRequests
      }
      byTs.set(point.ts, current)
    }
  }

  return [...byTs.entries()]
    .sort(([left], [right]) => left - right)
    .filter(([, value]) => value.request_count > 0)
    .map(([ts, value]) => ({
      ts,
      request_count: value.request_count,
      success_rate: (value.success_count / value.request_count) * 100,
    }))
}

export function inferBucketSeconds(timestamps: number[]): number {
  if (timestamps.length === 0) return DEFAULT_BUCKET_SECONDS
  let minDelta = Number.POSITIVE_INFINITY
  for (let index = 1; index < timestamps.length; index++) {
    const delta = timestamps[index] - timestamps[index - 1]
    if (delta > 0 && delta < minDelta) minDelta = delta
  }
  for (const size of KNOWN_BUCKET_SECONDS) {
    const aligned = timestamps.every((ts) => ts % size === 0)
    const fitsDelta = !Number.isFinite(minDelta) || minDelta % size === 0
    if (aligned && fitsDelta) return size
  }
  return DEFAULT_BUCKET_SECONDS
}

export function formatSuccessRateBucketTime(
  ts: number,
  bucketSeconds: number
): string {
  if (bucketSeconds < DEFAULT_BUCKET_SECONDS) {
    return dayjs(ts * 1000).format('MM-DD HH:mm')
  }
  return formatChartTime(ts, 'hour')
}

export function fillSuccessRateBuckets(
  points: MergedSuccessPoint[],
  hours: number,
  nowSec = Math.floor(Date.now() / 1000)
): FilledSuccessPoint[] {
  const endTs = nowSec
  const startTs = endTs - hours * 3600
  const timestamps = points.map((point) => point.ts)
  const bucketSeconds = inferBucketSeconds(timestamps)
  const byTs = new Map(points.map((point) => [point.ts, point]))

  let cursor = startTs - (startTs % bucketSeconds)

  const filled: FilledSuccessPoint[] = []
  for (; cursor <= endTs; cursor += bucketSeconds) {
    const hit = byTs.get(cursor)
    filled.push(
      hit
        ? {
            ts: cursor,
            success_rate: hit.success_rate,
            request_count: hit.request_count,
          }
        : { ts: cursor, success_rate: null, request_count: 0 }
    )
  }
  return filled
}

export function getSuccessRateAxisMin(values: number[]): number {
  const finiteValues = values.filter((value) => Number.isFinite(value))
  if (finiteValues.length === 0) return UPTIME_FOCUSED_AXIS_MIN

  const minValue = Math.max(0, Math.min(...finiteValues))
  if (minValue >= UPTIME_FOCUSED_AXIS_MIN) return UPTIME_FOCUSED_AXIS_MIN
  if (minValue >= UPTIME_MINOR_OUTAGE_AXIS_MIN) {
    return UPTIME_MINOR_OUTAGE_AXIS_MIN
  }
  return Math.max(0, Math.floor((minValue - 5) / 10) * 10)
}

export const SUCCESS_RATE_AXIS_MAX = UPTIME_AXIS_MAX
