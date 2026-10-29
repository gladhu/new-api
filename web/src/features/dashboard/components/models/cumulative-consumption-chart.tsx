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
import { LineChart, X } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { DatePicker } from '@/components/date-picker'
import { Button } from '@/components/ui/button'
import { Combobox } from '@/components/ui/combobox'
import { IconBadge } from '@/components/ui/icon-badge'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { getCumulativeQuota } from '@/features/dashboard/api'
import {
  buildCumulativeConsumptionSeries,
  buildCumulativeConsumptionSpec,
  formatCumulativeDayLabel,
  getCumulativeCompareBounds,
  getCumulativeQueryRange,
  isSelectableCompareDay,
} from '@/features/dashboard/lib/cumulative-consumption'
import { formatQuota } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { getStartOfDay } from '@/lib/time'
import { useChartTheme } from '@/lib/use-chart-theme'
import { VCHART_OPTION } from '@/lib/vchart'
import { useAuthStore } from '@/stores/auth-store'

interface CumulativeConsumptionChartProps {
  username?: string
}

export function CumulativeConsumptionChart(
  props: CumulativeConsumptionChartProps
) {
  const { t } = useTranslation()
  const { resolvedTheme, themeReady } = useChartTheme()
  const userRole = useAuthStore((state) => state.auth.user?.role)
  const isAdmin = Boolean(userRole && userRole >= ROLE.ADMIN)
  const [selectedModel, setSelectedModel] = useState('')
  const [compareDate, setCompareDate] = useState<Date | undefined>(undefined)

  const username = isAdmin ? props.username?.trim() : undefined
  const range = getCumulativeQueryRange(new Date())
  const compareBounds = getCumulativeCompareBounds(new Date())

  const query = useQuery({
    queryKey: [
      'dashboard',
      'cumulative-consumption',
      isAdmin,
      username ?? '',
      selectedModel,
      range.start_timestamp,
      range.end_timestamp,
    ],
    queryFn: () =>
      getCumulativeQuota(
        {
          start_timestamp: range.start_timestamp,
          end_timestamp: range.end_timestamp,
          ...(username ? { username } : {}),
          ...(selectedModel ? { model_name: selectedModel } : {}),
        },
        isAdmin
      ),
    staleTime: 60 * 1000,
    refetchInterval: 60 * 1000,
    retry: false,
  })

  const models = useMemo(() => query.data?.models ?? [], [query.data?.models])
  const modelOptions = useMemo(() => {
    const names = [...models]
    if (selectedModel && !names.includes(selectedModel)) {
      names.push(selectedModel)
    }
    names.sort((left, right) => left.localeCompare(right))
    return [
      { value: '', label: t('All models') },
      ...names.map((name) => ({ value: name, label: name })),
    ]
  }, [models, selectedModel, t])

  const fetchedAt = query.dataUpdatedAt
  const series = useMemo(() => {
    if (!query.isSuccess) return []
    return buildCumulativeConsumptionSeries({
      points: query.data?.points ?? [],
      now: fetchedAt > 0 ? new Date(fetchedAt) : new Date(),
      compareDate,
      labels: { today: t('Today'), yesterday: t('Yesterday') },
    })
  }, [compareDate, fetchedAt, query.data?.points, query.isSuccess, t])

  const todayTotal =
    [...series].reverse().find((point) => point.kind === 'today')?.quota ?? 0

  const spec = useMemo(() => {
    if (!query.isSuccess) return null
    return buildCumulativeConsumptionSpec({
      points: series,
      resolvedTheme,
      formatQuotaValue: (quota) => formatQuota(quota),
      formatPoint: (point) =>
        `${formatQuota(point.quota)} (${t('This hour')} ${formatQuota(point.hourly)})`,
    })
  }, [query.isSuccess, resolvedTheme, series, t])

  const compareLabel = compareDate
    ? `${t('Compare another day')}: ${formatCumulativeDayLabel(compareDate)}`
    : t('Compare another day')

  let chartBody
  if (query.isLoading) {
    chartBody = <Skeleton className='h-full w-full' />
  } else if (query.isError) {
    chartBody = (
      <div className='text-muted-foreground flex h-full items-center justify-center px-4 text-center text-xs'>
        {t('Failed to load cumulative consumption')}
      </div>
    )
  } else if (themeReady && spec) {
    chartBody = (
      <VChart
        key={`${selectedModel}-${compareDate?.getTime() ?? 'none'}-${resolvedTheme}-${series.length}`}
        spec={{
          ...spec,
          theme: resolvedTheme === 'dark' ? 'dark' : 'light',
          background: 'transparent',
        }}
        option={VCHART_OPTION}
      />
    )
  } else {
    chartBody = <Skeleton className='h-full w-full' />
  }

  return (
    <section
      className='overflow-hidden rounded-lg border'
      aria-labelledby='cumulative-consumption-title'
    >
      <div className='flex w-full flex-col gap-2 border-b px-3 py-2 sm:gap-3 sm:px-5 sm:py-3 lg:flex-row lg:items-center lg:justify-between'>
        <div className='flex min-w-0 items-center gap-2'>
          <IconBadge tone='chart-2' size='sm'>
            <LineChart />
          </IconBadge>
          <div className='min-w-0'>
            <div className='flex flex-wrap items-baseline gap-x-2'>
              <div
                id='cumulative-consumption-title'
                className='text-sm font-semibold'
              >
                {t('Cumulative consumption')}
              </div>
              {query.isSuccess ? (
                <span className='text-muted-foreground text-xs'>
                  {t('Today')} {formatQuota(todayTotal)}
                </span>
              ) : null}
            </div>
            <div className='text-muted-foreground text-xs'>
              {t('Compare cumulative quota by time of day')}
            </div>
          </div>
        </div>

        <div className='flex w-full flex-col gap-2 sm:flex-row sm:items-center lg:w-auto'>
          <div className='min-w-0 flex-1 sm:w-[220px] sm:flex-none'>
            <Label htmlFor='cumulative-consumption-model' className='sr-only'>
              {t('Model Name')}
            </Label>
            <Combobox
              id='cumulative-consumption-model'
              options={modelOptions}
              value={selectedModel}
              onValueChange={(value) => setSelectedModel(value ?? '')}
              placeholder={t('Search models')}
              searchPlaceholder={t('Search models')}
              emptyText='No model found.'
              className='w-full'
            />
          </div>
          <div className='flex min-w-0 items-center gap-1 sm:w-[220px]'>
            <DatePicker
              selected={compareDate}
              onSelect={(date) => {
                if (!date) {
                  setCompareDate(undefined)
                  return
                }
                if (!isSelectableCompareDay(date, new Date())) return
                setCompareDate(getStartOfDay(date))
              }}
              placeholder={t('Compare another day')}
              ariaLabel={compareLabel}
              disabled={(date) => !isSelectableCompareDay(date, new Date())}
              defaultMonth={compareDate ?? compareBounds.max}
              startMonth={compareBounds.min}
              endMonth={compareBounds.max}
              className='h-8 w-full min-w-0'
            />
            {compareDate ? (
              <Button
                type='button'
                variant='ghost'
                size='icon'
                className='size-8 shrink-0'
                aria-label={t('Clear comparison day')}
                onClick={() => setCompareDate(undefined)}
              >
                <X />
              </Button>
            ) : null}
          </div>
        </div>
      </div>

      <div
        className='h-[300px] p-1.5 sm:h-96 sm:p-2'
        aria-busy={query.isLoading}
      >
        {chartBody}
      </div>
    </section>
  )
}
