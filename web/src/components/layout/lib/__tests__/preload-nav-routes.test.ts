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
import { describe, expect, test, vi } from 'vitest'

import type { NavGroup } from '../../types'
import {
  collectSidebarNavHrefs,
  navHrefToPreloadOptions,
  preloadNavHrefs,
} from '../preload-nav-routes'

describe('navHrefToPreloadOptions', () => {
  test('maps the console index and overview paths onto the dashboard section route', () => {
    expect(navHrefToPreloadOptions('/dashboard')).toEqual({
      to: '/dashboard/$section',
      params: { section: 'overview' },
    })
    expect(navHrefToPreloadOptions('/dashboard/overview')).toEqual({
      to: '/dashboard/$section',
      params: { section: 'overview' },
    })
    expect(navHrefToPreloadOptions('/dashboard/models?tab=1')).toEqual({
      to: '/dashboard/$section',
      params: { section: 'models' },
    })
  })

  test('keeps ordinary in-app paths and ignores external hrefs', () => {
    expect(navHrefToPreloadOptions('/pricing')).toEqual({ to: '/pricing' })
    expect(navHrefToPreloadOptions('https://example.com/docs')).toBeNull()
    expect(navHrefToPreloadOptions('//cdn.example.com/x')).toBeNull()
  })
})

describe('preloadNavHrefs', () => {
  test('preloads unique in-app destinations when a mobile menu is opened', () => {
    const preloadRoute = vi.fn(() => Promise.resolve())

    preloadNavHrefs(preloadRoute, [
      '/dashboard',
      '/dashboard/overview',
      '/pricing',
      'https://skip.me',
    ])

    expect(preloadRoute).toHaveBeenCalledTimes(2)
    expect(preloadRoute).toHaveBeenCalledWith({
      to: '/dashboard/$section',
      params: { section: 'overview' },
    })
    expect(preloadRoute).toHaveBeenCalledWith({ to: '/pricing' })
  })
})

describe('collectSidebarNavHrefs', () => {
  test('collects top-level and nested urls and skips chat presets', () => {
    const groups: NavGroup[] = [
      {
        title: 'Workspace',
        items: [
          { title: 'Keys', url: '/keys' },
          {
            title: 'Logs',
            items: [{ title: 'Usage', url: '/usage-logs' }],
          },
          { title: 'Chats', type: 'chat-presets' },
        ],
      },
    ]

    expect(collectSidebarNavHrefs(groups)).toEqual(['/keys', '/usage-logs'])
  })
})
