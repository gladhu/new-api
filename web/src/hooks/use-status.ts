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
import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useSystemConfigStore } from '@/stores/system-config-store'
import { getStatus } from '@/lib/api'
import { applyFaviconToDom } from '@/lib/dom-utils'
import type { SystemStatus } from '@/features/auth/types'
import { mapStatusDataToConfig } from './use-system-config'

// Get initial cache from localStorage
function getInitialStatus(): SystemStatus | undefined {
  try {
    if (typeof window !== 'undefined') {
      const saved = window.localStorage.getItem('status')
      return saved ? (JSON.parse(saved) as SystemStatus) : undefined
    }
  } catch {
    /* empty */
  }
  return undefined
}

export function getCachedStatus(): Record<string, unknown> | null {
  const cached = getInitialStatus()
  return cached ? (cached as Record<string, unknown>) : null
}

export function applyStatusBranding(
  status: Record<string, unknown> | null | undefined
): void {
  if (!status || typeof document === 'undefined') return

  const systemName = status.system_name
  if (typeof systemName === 'string' && systemName) {
    document.title = systemName
    const metaTitle = document.querySelector(
      'meta[name="title"]'
    ) as HTMLMetaElement | null
    if (metaTitle) metaTitle.setAttribute('content', systemName)
  }

  const logo = status.logo
  if (typeof logo === 'string' && logo) {
    applyFaviconToDom(logo)
  }
}

function syncStatusResponse(status: Record<string, unknown> | null): void {
  if (!status) return

  try {
    const { setConfig } = useSystemConfigStore.getState()
    setConfig(mapStatusDataToConfig(status))
  } catch (err) {
    if (import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.warn('[useStatus] Failed to sync status to system config', err)
    }
  }

  try {
    if (typeof window !== 'undefined') {
      window.localStorage.setItem('status', JSON.stringify(status))
    }
  } catch {
    /* empty */
  }

  applyStatusBranding(status)
}

let cachedStatusSynced = false

function syncCachedStatusOnce(): void {
  if (cachedStatusSynced) return
  cachedStatusSynced = true

  const cached = getCachedStatus()
  if (cached) {
    syncStatusResponse(cached)
    useSystemConfigStore.getState().setLoading(false)
  }
}

export function useStatus() {
  useEffect(() => {
    syncCachedStatusOnce()
  }, [])

  const { data, isLoading, error } = useQuery({
    queryKey: ['status'],
    queryFn: async () => {
      try {
        const status = await getStatus()
        syncStatusResponse(status)
        return status as SystemStatus | null
      } finally {
        useSystemConfigStore.getState().setLoading(false)
      }
    },
    // Use localStorage data as initial data
    placeholderData: getInitialStatus(),
    // Data becomes stale after 5 minutes
    staleTime: 5 * 60 * 1000,
    // Cache expires after 30 minutes
    gcTime: 30 * 60 * 1000,
  })

  return {
    status: data ?? null,
    loading: isLoading,
    error,
  }
}
