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

const CUSTOM_HOME_SCOPE = '.custom-home-content'

function findMatchingBrace(css: string, openIndex: number): number {
  let depth = 0
  for (let i = openIndex; i < css.length; i += 1) {
    if (css[i] === '{') depth += 1
    else if (css[i] === '}') {
      depth -= 1
      if (depth === 0) return i
    }
  }
  return css.length - 1
}

function scopeSelectorList(selectors: string, scope: string): string {
  return selectors
    .split(',')
    .map((raw) => {
      const sel = raw.trim()
      if (!sel) return sel
      if (sel === 'html' || sel === 'body' || sel === ':root') return scope
      if (
        sel.includes(scope) ||
        sel.startsWith(':root:not(.dark)') ||
        sel.startsWith('.dark .custom-home-content')
      ) {
        return sel
      }
      return `${scope} ${sel}`
    })
    .join(', ')
}

function scopeCssRules(css: string, scope: string): string {
  let result = ''
  let i = 0

  while (i < css.length) {
    const char = css[i]

    if (char === '@' && css.startsWith('@media', i)) {
      const blockStart = css.indexOf('{', i)
      if (blockStart === -1) break
      const blockEnd = findMatchingBrace(css, blockStart)
      const mediaQuery = css.slice(i, blockStart + 1)
      const inner = css.slice(blockStart + 1, blockEnd)
      result += `${mediaQuery}${scopeCssRules(inner, scope)}}`
      i = blockEnd + 1
      continue
    }

    if (char === '@') {
      const blockStart = css.indexOf('{', i)
      if (blockStart === -1) break
      const blockEnd = findMatchingBrace(css, blockStart)
      result += css.slice(i, blockEnd + 1)
      i = blockEnd + 1
      continue
    }

    if (/\s/.test(char)) {
      result += char
      i += 1
      continue
    }

    const ruleStart = i
    const braceStart = css.indexOf('{', ruleStart)
    if (braceStart === -1) {
      result += css.slice(i)
      break
    }

    const selectors = css.slice(ruleStart, braceStart).trim()
    const braceEnd = findMatchingBrace(css, braceStart)
    const declarations = css.slice(braceStart, braceEnd + 1)
    result += `${scopeSelectorList(selectors, scope)} ${declarations}`
    i = braceEnd + 1
  }

  return result
}

/** Limit custom home CSS to the inline container so it cannot override global UI. */
export function scopeCustomHomeStyles(css: string): string {
  const trimmed = css.trim()
  if (!trimmed) return trimmed

  const themeMapped = trimmed
    .replace(/html\.embedded\s+/g, `${CUSTOM_HOME_SCOPE} `)
    .replace(/html\.light\s+/g, ':root:not(.dark) .custom-home-content ')
    .replace(/html\.dark\s+/g, '.dark .custom-home-content ')

  return scopeCssRules(themeMapped, CUSTOM_HOME_SCOPE)
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
