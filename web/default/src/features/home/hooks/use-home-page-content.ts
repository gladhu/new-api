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
import { useEffect, useState } from 'react'
import i18next from 'i18next'
import { toast } from 'sonner'
import { isHttpUrl, isLikelyHtml } from '@/lib/content-format'
import { getHomePageContent } from '../api'
import {
  isFetchableHomeSource,
  resolveHomeContentToInlineHtml,
} from '../lib/custom-home-html'
import type { HomeContentMode, HomePageContentResult } from '../types'

const STORAGE_KEY = 'home_page_content'
const RESOLVED_STORAGE_KEY = 'home_page_content_resolved'
const RESOLVED_MODE_KEY = 'home_page_content_mode'

function readResolvedCache(): { content: string; mode: HomeContentMode } | null {
  try {
    const content = localStorage.getItem(RESOLVED_STORAGE_KEY)
    if (!content) return null
    const mode = localStorage.getItem(RESOLVED_MODE_KEY) as HomeContentMode | null
    if (mode === 'inline-html' || mode === 'markdown') {
      return { content, mode }
    }
    return { content, mode: 'inline-html' }
  } catch {
    return null
  }
}

function writeResolvedCache(content: string, mode: HomeContentMode): void {
  try {
    localStorage.setItem(RESOLVED_STORAGE_KEY, content)
    localStorage.setItem(RESOLVED_MODE_KEY, mode)
  } catch {
    /* empty */
  }
}

function clearResolvedCache(): void {
  try {
    localStorage.removeItem(RESOLVED_STORAGE_KEY)
    localStorage.removeItem(RESOLVED_MODE_KEY)
  } catch {
    /* empty */
  }
}

async function resolveSourceToHomeContent(source: string): Promise<{
  content: string
  mode: HomeContentMode
} | null> {
  const trimmed = source.trim()
  if (!trimmed) return null

  if (isHttpUrl(trimmed) || trimmed.startsWith('/')) {
    const html = await resolveHomeContentToInlineHtml(trimmed)
    if (!html) return null
    return { content: html, mode: 'inline-html' }
  }

  if (isLikelyHtml(trimmed)) {
    const html = await resolveHomeContentToInlineHtml(trimmed)
    return { content: html, mode: 'inline-html' }
  }

  return { content: trimmed, mode: 'markdown' }
}

function getInitialHomeState(): { content: string; mode: HomeContentMode } {
  const resolved = readResolvedCache()
  if (resolved) return resolved

  try {
    const raw = localStorage.getItem(STORAGE_KEY)?.trim()
    if (raw && isFetchableHomeSource(raw)) {
      return { content: '', mode: 'inline-html' }
    }
  } catch {
    /* empty */
  }

  return { content: '', mode: 'default' }
}

/**
 * Load and manage custom home page content.
 * Uses localStorage for instant render; refreshes from API in the background.
 */
export function useHomePageContent(): HomePageContentResult {
  const initial = getInitialHomeState()
  const [content, setContent] = useState(initial.content)
  const [mode, setMode] = useState<HomeContentMode>(initial.mode)
  const [isRefreshing, setIsRefreshing] = useState(
    () => initial.mode === 'inline-html' && !initial.content
  )

  useEffect(() => {
    let mounted = true

    const refresh = async () => {
      const hadResolvedCache = Boolean(readResolvedCache())
      setIsRefreshing(!hadResolvedCache)

      try {
        const response = await getHomePageContent()
        const source = response.success ? (response.data?.trim() ?? '') : ''

        if (!mounted) return

        if (!source) {
          setContent('')
          setMode('default')
          localStorage.removeItem(STORAGE_KEY)
          clearResolvedCache()
          return
        }

        localStorage.setItem(STORAGE_KEY, source)

        const resolved = await resolveSourceToHomeContent(source)
        if (!mounted) return

        if (!resolved) {
          // Cross-origin URL (e.g. production URL in local DB) — fall back quietly.
          if (!isFetchableHomeSource(source)) {
            setContent('')
            setMode('default')
            return
          }
          toast.error(i18next.t('Failed to load home page content'))
          setContent('')
          setMode('default')
          clearResolvedCache()
          return
        }

        setContent(resolved.content)
        setMode(resolved.mode)
        writeResolvedCache(resolved.content, resolved.mode)
      } catch (error) {
        if (!mounted) return
        // eslint-disable-next-line no-console
        console.error('Failed to load home page content:', error)
        if (!readResolvedCache()) {
          toast.error(i18next.t('Failed to load home page content'))
        }
      } finally {
        if (mounted) {
          setIsRefreshing(false)
        }
      }
    }

    void refresh()

    return () => {
      mounted = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return { content, mode, isRefreshing }
}
