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
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { describe, expect, test, vi } from 'vitest'

vi.mock('@tanstack/react-router', () => ({
  Link: (props: { to: string; children: ReactNode; className?: string }) => (
    <a href={props.to} className={props.className}>
      {props.children}
    </a>
  ),
  Outlet: () => <div data-testid='integration-outlet' />,
  useRouterState: () => ({
    location: { pathname: '/docs/integration' },
  }),
}))

vi.mock('@/components/layout', () => ({
  PublicLayout: (props: { children: ReactNode }) => <div>{props.children}</div>,
}))

const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { IntegrationDocsShell } = await import('../integration-layout')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Claude Code': 'Claude Code',
        'Code Buddy': 'Code Buddy',
        Codex: 'Codex',
        'Gemini CLI': 'Gemini CLI',
        Integration: 'Integration',
        'Integration guides': 'Integration guides',
        OpenCode: 'OpenCode',
        'Open menu': 'Open menu',
        Overview: 'Overview',
        'Trace (Trae IDE)': 'Trace (Trae IDE)',
      },
    },
  },
})

function renderShell() {
  return render(
    <I18nextProvider i18n={i18n}>
      <IntegrationDocsShell />
    </I18nextProvider>
  )
}

describe('Integration docs mobile navigation', () => {
  test('opens the guide sheet when the menu button is clicked', async () => {
    const user = userEvent.setup()
    renderShell()

    const menuButton = screen.getByRole('button', { name: 'Open menu' })
    expect(menuButton).toBeEnabled()
    expect(
      screen.queryByRole('heading', { name: 'Integration guides' })
    ).not.toBeInTheDocument()

    await user.click(menuButton)

    expect(
      await screen.findByRole('heading', { name: 'Integration guides' })
    ).toBeVisible()
  })
})
