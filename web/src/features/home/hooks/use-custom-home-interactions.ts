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
import { type RefObject, useLayoutEffect } from 'react'
import { getCachedStatus } from '@/hooks/use-status'

const API_ENDPOINTS = [
  '/v1/chat/completions',
  '/v1/responses',
  '/v1/responses/compact',
  '/v1/messages',
  '/v1beta/models',
  '/v1/embeddings',
  '/v1/rerank',
  '/v1/images/generations',
  '/v1/images/edits',
  '/v1/images/variations',
  '/v1/audio/speech',
  '/v1/audio/transcriptions',
  '/v1/audio/translations',
]

const ROTATE_INTERVAL_MS = 3000

export function useCustomHomeInteractions(
  containerRef: RefObject<HTMLElement | null>,
  html: string,
  serverAddress?: string | null
) {
  useLayoutEffect(() => {
    const root = containerRef.current
    if (!root || !html.trim()) return

    const apiUrlInput = root.querySelector(
      '#hero-api-url'
    ) as HTMLInputElement | null
    const apiEndpointEl = root.querySelector(
      '#hero-api-endpoint'
    ) as HTMLElement | null
    const apiCopyBtn = root.querySelector(
      '#hero-api-copy'
    ) as HTMLButtonElement | null

    let endpointIndex = 0
    let copyResetTimer: ReturnType<typeof setTimeout> | undefined
    let rotateTimer: ReturnType<typeof setInterval> | undefined

    const defaultServerAddress = () =>
      window.location.origin.replace(/\/$/, '')

    const setServerAddress = (addr?: string | null) => {
      if (!apiUrlInput) return
      apiUrlInput.value =
        addr && String(addr).trim()
          ? String(addr).trim().replace(/\/$/, '')
          : defaultServerAddress()
    }

    const showEndpoint = (index: number) => {
      if (!apiEndpointEl || API_ENDPOINTS.length === 0) return
      endpointIndex =
        ((index % API_ENDPOINTS.length) + API_ENDPOINTS.length) %
        API_ENDPOINTS.length
      apiEndpointEl.textContent = API_ENDPOINTS[endpointIndex]
    }

    const restartRotateTimer = () => {
      if (rotateTimer) clearInterval(rotateTimer)
      if (!apiEndpointEl || API_ENDPOINTS.length === 0) return
      rotateTimer = setInterval(() => {
        showEndpoint(endpointIndex + 1)
      }, ROTATE_INTERVAL_MS)
    }

    const stepEndpoint = (delta: number) => {
      showEndpoint(endpointIndex + delta)
      restartRotateTimer()
    }

    const fallbackCopy = (text: string, done?: () => void) => {
      const ta = document.createElement('textarea')
      ta.value = text
      ta.setAttribute('readonly', '')
      ta.style.position = 'fixed'
      ta.style.left = '-9999px'
      document.body.appendChild(ta)
      ta.select()
      try {
        document.execCommand('copy')
        done?.()
      } catch {
        /* ignore */
      }
      document.body.removeChild(ta)
    }

    const onCopied = () => {
      if (!apiCopyBtn) return
      apiCopyBtn.textContent = '已复制'
      apiCopyBtn.classList.add('is-copied')
      if (copyResetTimer) clearTimeout(copyResetTimer)
      copyResetTimer = setTimeout(() => {
        apiCopyBtn.textContent = '复制'
        apiCopyBtn.classList.remove('is-copied')
      }, 1600)
    }

    const copyServerAddress = () => {
      if (!apiUrlInput) return
      const text = apiUrlInput.value || defaultServerAddress()
      if (navigator.clipboard?.writeText) {
        navigator.clipboard.writeText(text).then(onCopied).catch(() => {
          fallbackCopy(text, onCopied)
        })
      } else {
        fallbackCopy(text, onCopied)
      }
    }

    const onEndpointClick = () => {
      stepEndpoint(1)
    }

    const onEndpointWheel = (event: WheelEvent) => {
      event.preventDefault()
      stepEndpoint(event.deltaY > 0 ? 1 : -1)
    }

    const onEndpointKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault()
        stepEndpoint(1)
        return
      }
      if (event.key === 'ArrowDown') {
        event.preventDefault()
        stepEndpoint(1)
        return
      }
      if (event.key === 'ArrowUp') {
        event.preventDefault()
        stepEndpoint(-1)
      }
    }

    if (apiUrlInput) {
      const cachedAddress = getCachedStatus()?.server_address
      const resolvedAddress =
        serverAddress ??
        (typeof cachedAddress === 'string' ? cachedAddress : undefined)
      setServerAddress(resolvedAddress)
    }

    if (apiEndpointEl && API_ENDPOINTS.length > 0) {
      apiEndpointEl.setAttribute('role', 'button')
      apiEndpointEl.setAttribute('tabindex', '0')
      apiEndpointEl.setAttribute(
        'aria-label',
        'API 端点，点击或滚动切换'
      )
      showEndpoint(0)
      restartRotateTimer()
      apiEndpointEl.addEventListener('click', onEndpointClick)
      apiEndpointEl.addEventListener('wheel', onEndpointWheel, {
        passive: false,
      })
      apiEndpointEl.addEventListener('keydown', onEndpointKeyDown)
    }

    apiCopyBtn?.addEventListener('click', copyServerAddress)

    return () => {
      if (copyResetTimer) clearTimeout(copyResetTimer)
      if (rotateTimer) clearInterval(rotateTimer)
      apiCopyBtn?.removeEventListener('click', copyServerAddress)
      apiEndpointEl?.removeEventListener('click', onEndpointClick)
      apiEndpointEl?.removeEventListener('wheel', onEndpointWheel)
      apiEndpointEl?.removeEventListener('keydown', onEndpointKeyDown)
    }
  }, [containerRef, html, serverAddress])
}
