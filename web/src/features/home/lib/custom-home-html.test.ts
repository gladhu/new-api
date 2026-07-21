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
import { describe, expect, it } from 'vitest'

import { normalizeHomeContentSource, scopeCustomHomeStyles } from './custom-home-html'

describe('scopeCustomHomeStyles', () => {
  it('scopes global selectors to the custom home container', () => {
    const scoped = scopeCustomHomeStyles(`
      a { color: inherit; }
      .hero { padding: 1rem; }
    `)

    expect(scoped).toContain('.custom-home-content a { color: inherit; }')
    expect(scoped).toContain('.custom-home-content .hero { padding: 1rem; }')
  })

  it('maps standalone theme hooks to the app shell theme', () => {
    const scoped = scopeCustomHomeStyles('html.light .btn { color: #111; }')

    expect(scoped).toContain(
      ':root:not(.dark) .custom-home-content .btn { color: #111; }'
    )
  })
})

describe('normalizeHomeContentSource', () => {
  it('rewrites primary-domain absolute URLs to relative paths', () => {
    expect(
      normalizeHomeContentSource(
        'https://www.faceapi.ai/home-custom.html',
        'https://www.faceapi.ai'
      )
    ).toBe('/home-custom.html')
  })

  it('keeps relative paths unchanged', () => {
    expect(
      normalizeHomeContentSource('/home-custom.html', 'https://www.faceapi.ai')
    ).toBe('/home-custom.html')
  })

  it('leaves external iframe URLs unchanged', () => {
    expect(
      normalizeHomeContentSource(
        'https://example.com/page',
        'https://www.faceapi.ai'
      )
    ).toBe('https://example.com/page')
  })
})
