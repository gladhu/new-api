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
import { lazy, Suspense } from 'react'
import { HelpCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Skeleton } from '@/components/ui/skeleton'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useFAQ } from '@/features/dashboard/hooks/use-status-data'
import type { FAQItem } from '@/features/dashboard/types'
import { PanelWrapper } from '../ui/panel-wrapper'

const Markdown = lazy(() =>
  import('@/components/ui/markdown').then((m) => ({ default: m.Markdown }))
)

function FaqMarkdown(props: { className?: string; children: string }) {
  return (
    <Suspense fallback={<Skeleton className='h-4 w-full' />}>
      <Markdown className={props.className}>{props.children}</Markdown>
    </Suspense>
  )
}

export function FAQPanel() {
  const { t } = useTranslation()
  const { items: list, loading } = useFAQ()

  return (
    <PanelWrapper
      title={
        <span className='flex items-center gap-2'>
          <HelpCircle className='text-muted-foreground/60 size-4' />
          {t('FAQ')}
        </span>
      }
      description={t('Answers for common access and billing questions')}
      loading={loading}
      empty={!list.length}
      emptyMessage={t('No FAQ entries available')}
      height='h-80'
      contentClassName='p-0'
    >
      <ScrollArea className='h-80'>
        <Accordion className='w-full px-4 sm:px-5'>
          {list.map((item: FAQItem, idx: number) => {
            const key = item.id ?? `faq-${idx}`
            const value = `item-${key}`
            return (
              <AccordionItem
                key={key}
                value={value}
                className='border-border/60'
              >
                <AccordionTrigger className='text-start hover:no-underline'>
                  <FaqMarkdown className='text-sm leading-relaxed font-semibold'>
                    {item.question}
                  </FaqMarkdown>
                </AccordionTrigger>
                <AccordionContent>
                  <FaqMarkdown className='text-muted-foreground/60 text-sm'>
                    {item.answer}
                  </FaqMarkdown>
                </AccordionContent>
              </AccordionItem>
            )
          })}
        </Accordion>
      </ScrollArea>
    </PanelWrapper>
  )
}
