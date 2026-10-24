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
import { useCallback, useSyncExternalStore } from 'react'

import {
  DEFAULT_HEALTH_WINDOW_HOURS,
  HEALTH_WINDOW_OPTIONS,
  HEALTH_WINDOW_STORAGE_KEY,
  type HealthWindowHours,
} from '@/features/dashboard/constants'

const listeners = new Set<() => void>()

function isHealthWindowHours(value: unknown): value is HealthWindowHours {
  return HEALTH_WINDOW_OPTIONS.some((option) => option.hours === value)
}

function readStoredWindow(): HealthWindowHours {
  if (typeof window === 'undefined') return DEFAULT_HEALTH_WINDOW_HOURS
  const parsed = Number(window.localStorage.getItem(HEALTH_WINDOW_STORAGE_KEY))
  return isHealthWindowHours(parsed) ? parsed : DEFAULT_HEALTH_WINDOW_HOURS
}

let cachedWindow = readStoredWindow()

function subscribe(onStoreChange: () => void) {
  listeners.add(onStoreChange)
  return () => {
    listeners.delete(onStoreChange)
  }
}

function getSnapshot() {
  return cachedWindow
}

function getServerSnapshot() {
  return DEFAULT_HEALTH_WINDOW_HOURS
}

export function useHealthWindowHours(): [
  HealthWindowHours,
  (hours: HealthWindowHours) => void,
] {
  const hours = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot)

  const setHours = useCallback((next: HealthWindowHours) => {
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(HEALTH_WINDOW_STORAGE_KEY, String(next))
    }
    cachedWindow = next
    listeners.forEach((listener) => listener())
  }, [])

  return [hours, setHours]
}

export function healthWindowTitleKey(hours: HealthWindowHours): string {
  return (
    HEALTH_WINDOW_OPTIONS.find((option) => option.hours === hours)?.titleKey ??
    'Last 24 hours'
  )
}
