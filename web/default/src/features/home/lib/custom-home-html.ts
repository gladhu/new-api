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
import { isHttpUrl } from '@/lib/content-format'

function toRelativePath(url: URL): string {
  return `${url.pathname}${url.search}${url.hash}`
}

/** Rewrite primary-domain absolute URLs to site-relative paths for CDN mirrors. */
export function normalizeHomeContentSource(
  source: string,
  serverAddress?: string | null
): string {
  const trimmed = source.trim()
  if (!trimmed || trimmed.startsWith('/') || !isHttpUrl(trimmed)) {
    return trimmed
  }

  try {
    const url = new URL(trimmed)
    if (url.origin === window.location.origin) {
      return toRelativePath(url)
    }

    if (serverAddress) {
      const serverOrigin = new URL(serverAddress.replace(/\/$/, '')).origin
      if (url.origin === serverOrigin) {
        return toRelativePath(url)
      }
    }
  } catch {
    return trimmed
  }

  return trimmed
}

export function isSameOriginUrl(value: string): boolean {
  if (!isHttpUrl(value)) return false
  try {
    const url = new URL(value)
    return url.origin === window.location.origin
  } catch {
    return false
  }
}

/** Whether the configured home source can be loaded on the current origin. */
export function isFetchableHomeSource(source: string): boolean {
  const trimmed = source.trim()
  if (!trimmed) return false
  if (trimmed.startsWith('/')) return true
  if (isHttpUrl(trimmed)) return isSameOriginUrl(trimmed)
  return true
}

function resolveSameOriginFetchUrl(value: string): string | null {
  const trimmed = value.trim()
  if (!trimmed) return null

  try {
    if (isHttpUrl(trimmed)) {
      return isSameOriginUrl(trimmed) ? trimmed : null
    }
    if (trimmed.startsWith('/')) {
      return `${window.location.origin}${trimmed}`
    }
  } catch {
    return null
  }

  return null
}

/** Strip duplicate site header and keep styles + main body for inline SPA rendering. */
export function extractInlineHomeHtml(documentHtml: string): string {
  const parser = new DOMParser()
  const doc = parser.parseFromString(documentHtml, 'text/html')
  doc.querySelector('header')?.remove()

  const styles = Array.from(doc.querySelectorAll('style'))
    .map((el) => el.textContent ?? '')
    .filter(Boolean)
    .join('\n')

  const main = doc.querySelector('main')
  const bodyHtml = main ? main.innerHTML : doc.body.innerHTML

  if (!styles) return bodyHtml
  return `<style>${styles}</style>${bodyHtml}`
}

export async function fetchInlineHomeFromUrl(url: string): Promise<string> {
  const fetchUrl = resolveSameOriginFetchUrl(url)
  if (!fetchUrl) {
    throw new Error('Custom home URL is not same-origin')
  }

  const response = await fetch(fetchUrl, { credentials: 'same-origin' })
  if (!response.ok) {
    throw new Error(`Failed to fetch custom home (${response.status})`)
  }
  return extractInlineHomeHtml(await response.text())
}

export async function resolveHomeContentToInlineHtml(
  source: string
): Promise<string | null> {
  const trimmed = source.trim()
  if (!trimmed) return null

  if (isHttpUrl(trimmed) || trimmed.startsWith('/')) {
    if (!isFetchableHomeSource(trimmed)) return null
    return fetchInlineHomeFromUrl(trimmed)
  }

  return extractInlineHomeHtml(trimmed)
}
