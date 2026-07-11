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
import { HtmlContent } from '@/components/html-content'
import { lazy, Suspense } from 'react'
import { Skeleton } from '@/components/ui/skeleton'

const Markdown = lazy(() =>
  import('@/components/ui/markdown').then((m) => ({ default: m.Markdown }))
)

type RichContentMode = 'markdown' | 'html'

interface RichContentProps {
  content: string
  mode?: RichContentMode
  breaks?: boolean
  className?: string
}

function MarkdownFallback(props: { className?: string }) {
  return (
    <div className={props.className}>
      <Skeleton className='mb-3 h-4 w-full' />
      <Skeleton className='mb-3 h-4 w-5/6' />
      <Skeleton className='h-4 w-2/3' />
    </div>
  )
}

export function RichContent(props: RichContentProps) {
  if (props.mode === 'html') {
    return <HtmlContent content={props.content} className={props.className} />
  }

  return (
    <Suspense fallback={<MarkdownFallback className={props.className} />}>
      <Markdown breaks={props.breaks} className={props.className}>
        {props.content}
      </Markdown>
    </Suspense>
  )
}
