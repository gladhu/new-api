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
import { useQuery } from '@tanstack/react-query'
import { VChart } from '@visactor/react-vchart'
import { Activity } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Combobox } from '@/components/ui/combobox'
import { IconBadge } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'
import { useThemeCustomization } from '@/context/theme-customization-provider'
import {
  healthWindowTitleKey,
  useHealthWindowHours,
} from '@/features/dashboard/hooks/use-health-window'
import {
  getPerfMetrics,
  getPerfMetricsSummary,
} from '@/features/performance-metrics/api'
import {
  fillSuccessRateBuckets,
  formatSuccessRateBucketTime,
  getSuccessRateAxisMin,
  inferBucketSeconds,
  mergeGroupSuccessSeries,
  SUCCESS_RATE_AXIS_MAX,
} from '@/features/performance-metrics/lib/aggregate'
import {
  formatUptimePct,
  getSuccessRateColor,
} from '@/features/performance-metrics/lib/format'
import { useChartTheme } from '@/lib/use-chart-theme'
import { VCHART_OPTION } from '@/lib/vchart'

export function SuccessRateTrendChart() {
  const { t } = useTranslation()
  const { resolvedTheme, themeReady } = useChartTheme()
  const { customization } = useThemeCustomization()
  const [windowHours] = useHealthWindowHours()
  const [selectedModel, setSelectedModel] = useState('')

  const summaryQuery = useQuery({
    queryKey: ['perf-metrics-summary', windowHours],
    queryFn: () => getPerfMetricsSummary(windowHours),
    staleTime: 60 * 1000,
    retry: false,
  })

  const models = useMemo(
    () =>
      (summaryQuery.data?.data.models ?? []).filter(
        (model) => (model.request_count ?? 0) > 0
      ),
    [summaryQuery.data]
  )
  const defaultModel = models[0]?.model_name ?? ''

  useEffect(() => {
    if (
      !selectedModel ||
      !models.some((model) => model.model_name === selectedModel)
    ) {
      setSelectedModel(defaultModel)
    }
  }, [defaultModel, models, selectedModel])

  const seriesQuery = useQuery({
    queryKey: ['perf-metrics', selectedModel, windowHours],
    queryFn: () => getPerfMetrics(selectedModel, windowHours),
    enabled: selectedModel.length > 0,
    staleTime: 60 * 1000,
    retry: false,
  })

  const { filledPoints, bucketSeconds } = useMemo(() => {
    const groups = seriesQuery.data?.data.groups ?? []
    const merged = mergeGroupSuccessSeries(groups)
    return {
      filledPoints: fillSuccessRateBuckets(merged, windowHours),
      bucketSeconds: inferBucketSeconds(merged.map((point) => point.ts)),
    }
  }, [seriesQuery.data, windowHours])

  const hasSeries = filledPoints.some((point) => point.success_rate != null)
  const loading = summaryQuery.isLoading || seriesQuery.isLoading

  const spec = useMemo(() => {
    if (!hasSeries) return null

    const rawData = filledPoints.map((point) => ({
      time: formatSuccessRateBucketTime(point.ts, bucketSeconds),
      success_rate: point.success_rate,
      request_count: point.request_count,
    }))
    const data =
      rawData.filter((point) => point.success_rate != null).length === 1
        ? rawData.flatMap((point) =>
            point.success_rate == null
              ? [point]
              : [
                  { ...point, time: `${point.time}__start` },
                  { ...point, time: `${point.time}__end` },
                ]
          )
        : rawData
    const axisMin = getSuccessRateAxisMin(
      rawData
        .map((point) => point.success_rate)
        .filter((value): value is number => value != null)
    )
    const textColor =
      resolvedTheme === 'dark'
        ? 'rgba(255, 255, 255, 0.68)'
        : 'rgba(15, 23, 42, 0.58)'
    const gridColor =
      resolvedTheme === 'dark'
        ? 'rgba(255, 255, 255, 0.12)'
        : 'rgba(15, 23, 42, 0.12)'

    return {
      type: 'line' as const,
      data: [{ id: 'success-rate', values: data }],
      xField: 'time',
      yField: 'success_rate',
      invalidType: 'break' as const,
      smooth: true,
      line: {
        style: { stroke: '#10b981', lineWidth: 2 },
      },
      point: {
        visible: true,
        style: {
          size: 5,
          stroke: '#ffffff',
          lineWidth: 1.5,
          fill: (datum: { success_rate?: number | null }) =>
            getSuccessRateColor(Number(datum.success_rate)),
        },
      },
      tooltip: {
        mark: {
          title: {
            value: (datum: { time: string }) =>
              String(datum.time).replace(/__(start|end)$/, ''),
          },
          content: [
            {
              key: t('Success rate'),
              value: (datum: { success_rate?: number | null }) =>
                formatUptimePct(Number(datum.success_rate)),
            },
            {
              key: t('Requests'),
              value: (datum: { request_count?: number }) =>
                String(datum.request_count ?? 0),
            },
          ],
        },
      },
      axes: [
        {
          orient: 'bottom',
          label: {
            formatMethod: (val: number | string) => {
              const cleaned = String(val).replace(/__(start|end)$/, '')
              return windowHours >= 168 ? cleaned.split(' ')[0] : cleaned
            },
            style: { fill: textColor, fontSize: 10 },
            autoLimit: true,
          },
          tick: { visible: false },
        },
        {
          orient: 'left',
          min: axisMin,
          max: SUCCESS_RATE_AXIS_MAX,
          label: {
            formatMethod: (val: number | string) => `${val}%`,
            style: { fill: textColor, fontSize: 10 },
          },
          grid: {
            visible: true,
            style: { lineDash: [3, 3], stroke: gridColor },
          },
        },
      ],
    }
  }, [bucketSeconds, filledPoints, hasSeries, resolvedTheme, t, windowHours])

  let chartBody
  if (loading) {
    chartBody = <Skeleton className='h-full w-full' />
  } else if (themeReady && spec) {
    chartBody = (
      <VChart
        key={`${selectedModel}-${windowHours}-${resolvedTheme}-${customization.preset}`}
        spec={{
          ...spec,
          theme: resolvedTheme === 'dark' ? 'dark' : 'light',
          background: 'transparent',
        }}
        option={VCHART_OPTION}
      />
    )
  } else {
    chartBody = (
      <div className='text-muted-foreground flex h-full items-center justify-center text-xs'>
        {t('No success rate data available')}
      </div>
    )
  }

  return (
    <div className='overflow-hidden rounded-lg border'>
      <div className='flex w-full flex-col gap-1.5 border-b px-3 py-2 sm:gap-3 sm:px-5 sm:py-3 lg:flex-row lg:items-center lg:justify-between'>
        <div className='flex items-center gap-2'>
          <IconBadge tone='success' size='sm'>
            <Activity />
          </IconBadge>
          <div className='text-sm font-semibold'>
            {t('Success rate trend')} · {t(healthWindowTitleKey(windowHours))}
          </div>
        </div>
        <Combobox
          options={models.map((model) => ({
            value: model.model_name,
            label: model.model_name,
          }))}
          value={selectedModel}
          onValueChange={(value) => {
            if (value) setSelectedModel(value)
          }}
          placeholder={t('Search models')}
          searchPlaceholder={t('Search models')}
          emptyText={t('No model found.')}
          className='w-full lg:w-[240px]'
        />
      </div>

      <div className='h-[300px] p-1.5 sm:h-96 sm:p-2'>{chartBody}</div>
    </div>
  )
}
