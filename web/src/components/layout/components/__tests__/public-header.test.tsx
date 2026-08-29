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
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, test, vi } from 'vitest'

const { preloadRoute, router } = vi.hoisted(() => {
  const preloadRoute = vi.fn(() => Promise.resolve())
  return {
    preloadRoute,
    router: { preloadRoute },
  }
})

vi.mock('@tanstack/react-router', () => ({
  Link: (props: {
    to: string
    params?: Record<string, string>
    children: ReactNode
    className?: string
    preload?: string
  }) => {
    const href = props.params
      ? props.to.replaceAll(/\$(\w+)/g, (_, key: string) => props.params?.[key] ?? '')
      : props.to
    return (
      <a href={href} className={props.className} data-preload={props.preload}>
        {props.children}
      </a>
    )
  },
  useNavigate: () => vi.fn(),
  useRouter: () => router,
  useRouterState: () => ({ location: { pathname: '/' } }),
}))

vi.mock('@/components/dialog', () => ({
  Dialog: () => null,
}))

vi.mock('@/components/language-switcher', () => ({
  LanguageSwitcher: () => null,
}))

vi.mock('@/components/notification-popover', () => ({
  NotificationPopover: () => null,
}))

vi.mock('@/components/profile-dropdown', () => ({
  ProfileDropdown: () => null,
}))

vi.mock('@/components/theme-switch', () => ({
  ThemeSwitch: () => null,
}))

vi.mock('@/hooks/use-notifications', () => ({
  useNotifications: () => ({
    popoverOpen: false,
    setPopoverOpen: vi.fn(),
    unreadCount: 0,
    activeTab: 'notice',
    setActiveTab: vi.fn(),
    notice: [],
    announcements: [],
    loading: false,
  }),
}))

vi.mock('@/hooks/use-system-config', () => ({
  useSystemConfig: () => ({
    systemName: 'FaceCloud',
    logo: '',
    loading: false,
    logoLoaded: true,
  }),
}))

vi.mock('@/hooks/use-top-nav-links', () => ({
  useTopNavLinks: () => [
    { title: 'Home', href: '/' },
    { title: 'Console', href: '/dashboard/overview' },
    { title: 'Model Square', href: '/pricing' },
  ],
}))

vi.mock('@/stores/auth-store', () => ({
  useAuthStore: () => ({ auth: { user: { id: 1 } } }),
}))

const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { PublicHeader } = await import('../public-header')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        Console: 'Console',
        'Go to Dashboard': 'Go to Dashboard',
        Home: 'Home',
        'Model Square': 'Model Square',
        'Toggle navigation menu': 'Toggle navigation menu',
      },
    },
  },
})

afterEach(() => {
  cleanup()
  preloadRoute.mockClear()
})

function renderHeader() {
  return render(
    <I18nextProvider i18n={i18n}>
      <PublicHeader />
    </I18nextProvider>
  )
}

describe('PublicHeader mobile navigation', () => {
  test('keeps the menu button readable in dark mode and expands the sheet', async () => {
    const user = userEvent.setup()
    renderHeader()

    const menuButton = screen.getByRole('button', {
      name: 'Toggle navigation menu',
    })
    expect(menuButton).toHaveAttribute('aria-expanded', 'false')
    expect(menuButton.className.split(' ')).toEqual(
      expect.arrayContaining(['text-foreground'])
    )
    expect(
      menuButton.querySelectorAll('.bg-foreground').length
    ).toBeGreaterThan(0)

    await user.click(menuButton)

    expect(menuButton).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getAllByRole('link', { name: 'Console' }).length).toBeGreaterThan(
      0
    )
  })

  test('sends Console and dashboard entry points to the overview section', () => {
    renderHeader()

    const consoleLinks = screen.getAllByRole('link', { name: 'Console' })
    expect(
      consoleLinks.every(
        (link) => link.getAttribute('href') === '/dashboard/overview'
      )
    ).toBe(true)

    expect(
      screen.getByRole('link', { name: 'Go to Dashboard' })
    ).toHaveAttribute('href', '/dashboard/overview')
  })

  test('warms the console route after mount for a signed-in user without render-preload links', async () => {
    renderHeader()

    const consoleAndDashboardLinks = [
      ...screen.getAllByRole('link', { name: 'Console' }),
      screen.getByRole('link', { name: 'Go to Dashboard' }),
    ]
    expect(
      consoleAndDashboardLinks.every(
        (link) => link.getAttribute('data-preload') !== 'render'
      )
    ).toBe(true)

    await waitFor(() => {
      expect(preloadRoute).toHaveBeenCalledWith({
        to: '/dashboard/$section',
        params: { section: 'overview' },
      })
    })
  })

  test('starts downloading other in-app destinations when the mobile menu opens', async () => {
    const user = userEvent.setup()
    renderHeader()

    await waitFor(() => {
      expect(preloadRoute).toHaveBeenCalled()
    })
    preloadRoute.mockClear()

    await user.click(
      screen.getByRole('button', { name: 'Toggle navigation menu' })
    )

    expect(preloadRoute).toHaveBeenCalledWith({ to: '/pricing' })
    expect(preloadRoute).not.toHaveBeenCalledWith({
      to: '/dashboard/$section',
      params: { section: 'overview' },
    })
  })
})
