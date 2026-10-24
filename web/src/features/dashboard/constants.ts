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
  DashboardChartPreferences,
  DashboardFilters,
  TimeRangePresetId,
} from './types'

export const TIME_GRANULARITY_STORAGE_KEY = 'data_export_default_time'
export const DASHBOARD_CHART_PREFERENCES_STORAGE_KEY =
  'dashboard_models_chart_preferences'
export const DEFAULT_TIME_GRANULARITY = 'day' as const
export const MAX_CHART_TREND_POINTS = 7

export const DEFAULT_DASHBOARD_CHART_PREFERENCES: DashboardChartPreferences = {
  consumptionDistributionChart: 'bar',
  modelAnalyticsChart: 'trend',
  defaultTimeRange: '1',
  defaultTimeGranularity: DEFAULT_TIME_GRANULARITY,
}

export const TIME_RANGE_BY_GRANULARITY = {
  hour: 1,
  day: 7,
  week: 30,
} as const

export const TIME_GRANULARITY_OPTIONS = [
  { label: 'Hour', value: 'hour' },
  { label: 'Day', value: 'day' },
  { label: 'Week', value: 'week' },
] as const

export const TIME_RANGE_PRESETS: readonly {
  id: TimeRangePresetId
  label: string
  days?: number
}[] = [
  { id: '1', label: '1 Day', days: 1 },
  { id: '3', label: '3 Days', days: 3 },
  { id: '7', label: '7 Days', days: 7 },
  { id: '14', label: '14 Days', days: 14 },
  { id: '29', label: '29 Days', days: 29 },
  { id: 'month', label: 'This Month' },
]

export const DAY_TIME_RANGE_PRESETS = TIME_RANGE_PRESETS.filter(
  (preset): preset is { id: TimeRangePresetId; label: string; days: number } =>
    preset.days != null
)

export const HEALTH_WINDOW_OPTIONS = [
  { hours: 1, tabKey: '1 Hour', titleKey: 'Last 1 hour' },
  { hours: 6, tabKey: '6 Hours', titleKey: 'Last 6 hours' },
  { hours: 24, tabKey: '24 Hours', titleKey: 'Last 24 hours' },
  { hours: 168, tabKey: '7 Days', titleKey: 'Last 7 days' },
] as const

export type HealthWindowHours = (typeof HEALTH_WINDOW_OPTIONS)[number]['hours']

export const DEFAULT_HEALTH_WINDOW_HOURS: HealthWindowHours = 24
export const HEALTH_WINDOW_STORAGE_KEY = 'dashboard_models_health_window_hours'

export const CONSUMPTION_DISTRIBUTION_CHART_OPTIONS = [
  { value: 'bar', labelKey: 'Bar Chart' },
  { value: 'area', labelKey: 'Area Chart' },
] as const

export const MODEL_ANALYTICS_CHART_OPTIONS = [
  { value: 'trend', labelKey: 'Call Trend' },
  { value: 'proportion', labelKey: 'Call Count Distribution' },
  { value: 'top', labelKey: 'Call Count Ranking' },
] as const

export const EMPTY_DASHBOARD_FILTERS: DashboardFilters = {
  start_timestamp: undefined,
  end_timestamp: undefined,
  time_granularity: 'day',
  username: '',
  time_range_preset: null,
}
