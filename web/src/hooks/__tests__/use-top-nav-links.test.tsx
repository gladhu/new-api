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
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, test, vi } from 'vitest'

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({ status: null, loading: false }),
}))

vi.mock('@/stores/auth-store', () => ({
  useAuthStore: () => ({ auth: { user: { id: 1 } } }),
}))

const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { useTopNavLinks } = await import('../use-top-nav-links')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        About: 'About',
        Console: 'Console',
        Docs: 'Docs',
        Home: 'Home',
        'Model Square': 'Model Square',
        Rankings: 'Rankings',
      },
    },
  },
})

function NavLinksProbe() {
  const links = useTopNavLinks()
  return (
    <nav>
      {links.map((link) => (
        <a key={link.href} href={link.href}>
          {link.title}
        </a>
      ))}
    </nav>
  )
}

afterEach(() => {
  cleanup()
})

describe('useTopNavLinks console target', () => {
  test('points Console at the dashboard overview section instead of the redirecting index', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <NavLinksProbe />
      </I18nextProvider>
    )

    expect(screen.getByRole('link', { name: 'Console' })).toHaveAttribute(
      'href',
      '/dashboard/overview'
    )
  })
})
