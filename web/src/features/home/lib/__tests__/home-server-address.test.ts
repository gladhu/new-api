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
import { describe, expect, test } from 'vitest'

import {
  applyHomeServerAddress,
  readStatusServerAddress,
  resolveDisplayServerAddress,
} from '../custom-home-html'

const HERO_INPUT = `
<input type="text" class="hero-api-input" id="hero-api-url" readonly value="" aria-label="API 地址" />
`

describe('readStatusServerAddress', () => {
  test('reads nested data.server_address from cached status payloads', () => {
    expect(
      readStatusServerAddress({
        data: { server_address: 'https://nested.example.com' },
      })
    ).toBe('https://nested.example.com')
  })
})

describe('applyHomeServerAddress', () => {
  test('writes the base URL into the hero input markup', () => {
    const html = applyHomeServerAddress(HERO_INPUT, 'https://api.example.com')
    const container = document.createElement('div')
    container.innerHTML = html
    const input = container.querySelector('#hero-api-url') as HTMLInputElement

    expect(input.value).toBe('https://api.example.com')
    expect(input.getAttribute('value')).toBe('https://api.example.com')
  })
})

describe('resolveDisplayServerAddress', () => {
  test('falls back to the current origin when no address is configured', () => {
    expect(resolveDisplayServerAddress('')).toBe(
      window.location.origin.replace(/\/$/, '')
    )
  })
})
