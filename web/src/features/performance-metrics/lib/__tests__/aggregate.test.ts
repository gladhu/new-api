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
  fillSuccessRateBuckets,
  formatSuccessRateBucketTime,
  inferBucketSeconds,
  mergeGroupSuccessSeries,
  weightedSuccessRate,
} from '@/features/performance-metrics/lib/aggregate'
import type { PerformanceGroup } from '@/features/performance-metrics/types'

describe('weightedSuccessRate', () => {
  test('weights models by request count instead of averaging rates', () => {
    const result = weightedSuccessRate([
      { success_rate: 50, request_count: 10, success_count: 5 },
      { success_rate: 100, request_count: 90, success_count: 90 },
    ])
    expect(result.requests).toBe(100)
    expect(result.rate).toBe(95)
  })

  test('returns a non-finite rate when there are no requests', () => {
    const result = weightedSuccessRate([
      { success_rate: 100, request_count: 0, success_count: 0 },
    ])
    expect(result.requests).toBe(0)
    expect(Number.isFinite(result.rate)).toBe(false)
  })

  test('reconstructs successes from the rate when counts are missing', () => {
    const result = weightedSuccessRate([
      { success_rate: 80, request_count: 25 },
    ])
    expect(result.requests).toBe(25)
    expect(result.rate).toBe(80)
  })
})

describe('mergeGroupSuccessSeries', () => {
  test('merges groups in the same bucket by request count', () => {
    const groups: PerformanceGroup[] = [
      {
        group: 'default',
        avg_ttft_ms: 0,
        avg_latency_ms: 0,
        success_rate: 80,
        avg_tps: 0,
        series: [
          {
            ts: 100,
            avg_ttft_ms: 0,
            avg_latency_ms: 0,
            success_rate: 80,
            avg_tps: 0,
            request_count: 10,
            success_count: 8,
          },
        ],
      },
      {
        group: 'vip',
        avg_ttft_ms: 0,
        avg_latency_ms: 0,
        success_rate: 50,
        avg_tps: 0,
        series: [
          {
            ts: 100,
            avg_ttft_ms: 0,
            avg_latency_ms: 0,
            success_rate: 50,
            avg_tps: 0,
            request_count: 10,
            success_count: 5,
          },
        ],
      },
    ]

    expect(mergeGroupSuccessSeries(groups)).toEqual([
      { ts: 100, request_count: 20, success_rate: 65 },
    ])
  })
})

describe('fillSuccessRateBuckets', () => {
  test('infers bucket size from alignment and smallest positive gap', () => {
    expect(inferBucketSeconds([0, 3600, 7200])).toBe(3600)
    expect(inferBucketSeconds([0, 7200])).toBe(3600)
    expect(inferBucketSeconds([0, 300, 600])).toBe(300)
    expect(inferBucketSeconds([300, 3900])).toBe(300)
    expect(inferBucketSeconds([300])).toBe(300)
    expect(inferBucketSeconds([60])).toBe(60)
    expect(inferBucketSeconds([100])).toBe(3600)
  })

  test('formats bucket timestamps with unique precision for hourly and sub-hourly buckets', () => {
    const baseTs = Math.floor(new Date(2026, 9, 24, 14, 5, 0).getTime() / 1000)
    expect(formatSuccessRateBucketTime(baseTs, 300)).toBe('10-24 14:05')
    expect(formatSuccessRateBucketTime(baseTs, 3600)).toBe('10-24 14:00')
  })

  test('leaves empty buckets as null instead of zero percent', () => {
    const filled = fillSuccessRateBuckets(
      [
        { ts: 3600, success_rate: 100, request_count: 4 },
        { ts: 10800, success_rate: 50, request_count: 2 },
      ],
      3,
      10800
    )

    expect(filled).toEqual([
      { ts: 0, success_rate: null, request_count: 0 },
      { ts: 3600, success_rate: 100, request_count: 4 },
      { ts: 7200, success_rate: null, request_count: 0 },
      { ts: 10800, success_rate: 50, request_count: 2 },
    ])
  })
})
