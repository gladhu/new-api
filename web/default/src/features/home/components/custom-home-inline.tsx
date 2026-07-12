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
import DOMPurify from 'dompurify'
import { useMemo, useRef } from 'react'

import { useStatus } from '@/hooks/use-status'
import { cn } from '@/lib/utils'

import { useCustomHomeInteractions } from '../hooks/use-custom-home-interactions'
import { scopeCustomHomeStyles } from '../lib/custom-home-html'

interface CustomHomeInlineProps {
  html: string
  className?: string
}

function splitCustomHomeHtml(html: string): { styles: string; body: string } {
  const styleMatch = html.match(/^<style>([\s\S]*?)<\/style>([\s\S]*)$/i)
  if (!styleMatch) {
    return { styles: '', body: html }
  }
  return { styles: styleMatch[1] ?? '', body: styleMatch[2] ?? '' }
}

export function CustomHomeInline(props: CustomHomeInlineProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const { status } = useStatus()
  useCustomHomeInteractions(
    containerRef,
    status?.server_address as string | undefined
  )

  const { styles, body } = useMemo(() => {
    const split = splitCustomHomeHtml(props.html)
    return {
      styles: split.styles ? scopeCustomHomeStyles(split.styles) : '',
      body: split.body,
    }
  }, [props.html])

  const sanitizedBody = useMemo(() => DOMPurify.sanitize(body), [body])

  return (
    <div className={cn('custom-home-content', props.className)}>
      {styles ? (
        <style
          // Trusted same-origin admin home markup; scoped under .custom-home-content usage.
          // eslint-disable-next-line react/no-danger
          dangerouslySetInnerHTML={{ __html: styles }}
        />
      ) : null}
      <div
        ref={containerRef}
        // eslint-disable-next-line react/no-danger
        dangerouslySetInnerHTML={{ __html: sanitizedBody }}
      />
    </div>
  )
}
