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
import type { NavGroup } from '../types'

export type NavRoutePreloadOptions = {
  to: string
  params?: Record<string, string>
}

export type PreloadNavRoute = (
  options: NavRoutePreloadOptions
) => Promise<unknown>

/**
 * Map a public/sidebar href to TanStack Router preload options.
 * `/dashboard` and `/dashboard/:section` share the `$section` route module.
 */
export function navHrefToPreloadOptions(
  href: string
): NavRoutePreloadOptions | null {
  if (!href.startsWith('/') || href.startsWith('//')) {
    return null
  }

  const pathname = href.split(/[?#]/)[0] ?? href
  if (pathname === '/dashboard' || pathname.startsWith('/dashboard/')) {
    const section = pathname.split('/')[2] || 'overview'
    return {
      to: '/dashboard/$section',
      params: { section },
    }
  }

  return { to: pathname }
}

export function collectSidebarNavHrefs(groups: NavGroup[]): string[] {
  const hrefs: string[] = []

  for (const group of groups) {
    for (const item of group.items) {
      if (item.type === 'chat-presets') {
        continue
      }
      if (typeof item.url === 'string') {
        hrefs.push(item.url)
      }
      if (item.items) {
        for (const subItem of item.items) {
          if (typeof subItem.url === 'string') {
            hrefs.push(subItem.url)
          }
        }
      }
    }
  }

  return hrefs
}

export function preloadNavHrefs(
  preloadRoute: PreloadNavRoute,
  hrefs: Iterable<string>
) {
  const seen = new Set<string>()

  for (const href of hrefs) {
    const options = navHrefToPreloadOptions(href)
    if (!options) {
      continue
    }

    const key = `${options.to}:${options.params?.section ?? ''}`
    if (seen.has(key)) {
      continue
    }
    seen.add(key)
    void preloadRoute(options).catch(() => {
      // Warming is best-effort; navigation still works without it.
    })
  }
}
