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
import { Coins, LineChart, Activity, X, type LucideIcon } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { DatePicker } from '@/components/date-picker'
import { Button } from '@/components/ui/button'
import { Combobox } from '@/components/ui/combobox'
import { IconBadge } from '@/components/ui/icon-badge'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getCumulativeQuota } from '@/features/dashboard/api'
import {
  buildCumulativeConsumptionSeries,
  buildCumulativeConsumptionSpec,
  formatCumulativeDayLabel,
  getCumulativeCompareBounds,
  getCumulativeQueryRange,
  isSelectableCompareDay,
  type HourlyMetric,
} from '@/features/dashboard/lib/cumulative-consumption'
import { formatCompactNumber, formatNumber, formatQuota } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { getStartOfDay } from '@/lib/time'
import { useChartTheme } from '@/lib/use-chart-theme'
import { VCHART_OPTION } from '@/lib/vchart'
import { useAuthStore } from '@/stores/auth-store'

export type HourlyComparisonKind = 'quota' | 'tokens' | 'calls'

interface HourlyComparisonConfig {
  titleKey: string
  descriptionKey: string
  errorKey: string
  icon: LucideIcon
  metrics: { value: HourlyMetric; labelKey: string }[]
  formatAxis: (value: number) => string
  formatValue: (value: number) => string
}

const HOURLY_COMPARISON_CONFIGS: Record<
  HourlyComparisonKind,
  HourlyComparisonConfig
> = {
  quota: {
    titleKey: 'Cumulative consumption',
    descriptionKey: 'Compare cumulative quota by time of day',
    errorKey: 'Failed to load cumulative consumption',
    icon: LineChart,
    metrics: [{ value: 'quota', labelKey: 'Cumulative consumption' }],
    formatAxis: (value) => formatQuota(value),
    formatValue: (value) => formatQuota(value),
  },
  tokens: {
    titleKey: 'Token usage',
    descriptionKey: 'Compare cumulative token usage by time of day',
    errorKey: 'Failed to load token usage',
    icon: Coins,
    metrics: [
      { value: 'tokens', labelKey: 'All tokens' },
      { value: 'inputTokens', labelKey: 'Input tokens' },
      { value: 'outputTokens', labelKey: 'Output tokens' },
    ],
    formatAxis: (value) => formatCompactNumber(value),
    formatValue: (value) => formatNumber(value),
  },
  calls: {
    titleKey: 'Call Trend',
    descriptionKey: 'Compare cumulative calls by time of day',
    errorKey: 'Failed to load call trend',
    icon: Activity,
    metrics: [{ value: 'count', labelKey: 'Call Trend' }],
    formatAxis: (value) => formatCompactNumber(value),
    formatValue: (value) => formatNumber(value),
  },
}

interface HourlyComparisonChartProps {
  kind: HourlyComparisonKind
  username?: string
  // Render only the controls and the chart so a parent card can supply the title.
  embedded?: boolean
}

export function HourlyComparisonChart(props: HourlyComparisonChartProps) {
  const { t } = useTranslation()
  const config = HOURLY_COMPARISON_CONFIGS[props.kind]
  const { resolvedTheme, themeReady } = useChartTheme()
  const userRole = useAuthStore((state) => state.auth.user?.role)
  const isAdmin = Boolean(userRole && userRole >= ROLE.ADMIN)
  const [selectedModel, setSelectedModel] = useState('')
  const [compareDate, setCompareDate] = useState<Date | undefined>(undefined)
  const [metric, setMetric] = useState<HourlyMetric>(
    config.metrics[0]?.value ?? 'quota'
  )

  const username = isAdmin ? props.username?.trim() : undefined
  const range = getCumulativeQueryRange(new Date())
  const compareBounds = getCumulativeCompareBounds(new Date())

  const query = useQuery({
    queryKey: [
      'dashboard',
      'hourly-usage',
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
      metric,
      now: fetchedAt > 0 ? new Date(fetchedAt) : new Date(),
      compareDate,
      labels: {
        today: t('Today'),
        yesterday: t('Yesterday'),
        lastWeek: t('Same day last week'),
      },
    })
  }, [compareDate, fetchedAt, metric, query.data?.points, query.isSuccess, t])

  const todayTotal =
    [...series].reverse().find((point) => point.kind === 'today')?.value ?? 0

  const spec = useMemo(() => {
    if (!query.isSuccess) return null
    return buildCumulativeConsumptionSpec({
      points: series,
      resolvedTheme,
      formatValue: config.formatAxis,
      formatPoint: (point) =>
        `${config.formatValue(point.value)} (${t('This hour')} ${config.formatValue(point.hourly)})`,
    })
  }, [config, query.isSuccess, resolvedTheme, series, t])

  const compareLabel = compareDate
    ? `${t('Compare another day')}: ${formatCumulativeDayLabel(compareDate)}`
    : t('Compare another day')

  let chartBody
  if (query.isLoading) {
    chartBody = <Skeleton className='h-full w-full' />
  } else if (query.isError) {
    chartBody = (
      <div className='text-muted-foreground flex h-full items-center justify-center px-4 text-center text-xs'>
        {t(config.errorKey)}
      </div>
    )
  } else if (themeReady && spec) {
    chartBody = (
      <VChart
        key={`${metric}-${selectedModel}-${compareDate?.getTime() ?? 'none'}-${resolvedTheme}-${series.length}`}
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

  const todaySummary = query.isSuccess ? (
    <span className='text-muted-foreground text-xs'>
      {t('Today')} {config.formatValue(todayTotal)}
    </span>
  ) : null

  const titleId = `hourly-comparison-${props.kind}-title`
  const modelId = `hourly-comparison-${props.kind}-model`
  const Icon = config.icon

  return (
    <section
      className={
        props.embedded ? undefined : 'overflow-hidden rounded-lg border'
      }
      aria-label={t(config.titleKey)}
    >
      <div className='flex w-full flex-col gap-2 border-b px-3 py-2 sm:gap-3 sm:px-5 sm:py-3 lg:flex-row lg:items-center lg:justify-between'>
        {props.embedded ? (
          <div className='flex flex-wrap items-baseline gap-x-2'>
            {todaySummary}
          </div>
        ) : (
          <div className='flex min-w-0 items-center gap-2'>
            <IconBadge tone='chart-2' size='sm'>
              <Icon />
            </IconBadge>
            <div className='min-w-0'>
              <div className='flex flex-wrap items-baseline gap-x-2'>
                <div id={titleId} className='text-sm font-semibold'>
                  {t(config.titleKey)}
                </div>
                {todaySummary}
              </div>
              <div className='text-muted-foreground text-xs'>
                {t(config.descriptionKey)}
              </div>
            </div>
          </div>
        )}

        <div className='flex w-full flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center lg:w-auto'>
          {config.metrics.length > 1 ? (
            <Tabs
              value={metric}
              onValueChange={(value) => setMetric(value as HourlyMetric)}
            >
              <TabsList>
                {config.metrics.map((item) => (
                  <TabsTrigger key={item.value} value={item.value}>
                    {t(item.labelKey)}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          ) : null}
          <div className='min-w-0 flex-1 sm:w-[220px] sm:flex-none'>
            <Label htmlFor={modelId} className='sr-only'>
              {t('Model Name')}
            </Label>
            <Combobox
              id={modelId}
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
