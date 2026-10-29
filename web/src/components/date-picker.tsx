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
import { Calendar as CalendarIcon } from 'lucide-react'
import { useState } from 'react'
import { enUS, fr, ja, ru, vi, zhCN } from 'react-day-picker/locale'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

const calendarLocales = {
  en: enUS,
  zh: zhCN,
  fr,
  ru,
  ja,
  vi,
} as const

type DatePickerProps = {
  selected: Date | undefined
  onSelect: (date: Date | undefined) => void
  placeholder?: string
  ariaLabel?: string
  disabled?: (date: Date) => boolean
  className?: string
  defaultMonth?: Date
  startMonth?: Date
  endMonth?: Date
}

export function DatePicker({
  selected,
  onSelect,
  placeholder,
  ariaLabel,
  disabled,
  className,
  defaultMonth,
  startMonth,
  endMonth,
}: DatePickerProps) {
  const { t, i18n } = useTranslation()
  const [open, setOpen] = useState(false)
  const placeholderText = placeholder ?? t('Pick a date')
  const calendarLocale =
    calendarLocales[i18n.language as keyof typeof calendarLocales] ?? enUS
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            variant='outline'
            aria-label={ariaLabel}
            data-empty={!selected}
            className={cn(
              'data-[empty=true]:text-muted-foreground justify-start text-start font-normal',
              className ?? 'w-[240px]'
            )}
          />
        }
      >
        {selected ? (
          dayjs(selected).format('YYYY-MM-DD')
        ) : (
          <span>{placeholderText}</span>
        )}
        <CalendarIcon className='ms-auto h-4 w-4 opacity-50' />
      </PopoverTrigger>
      <PopoverContent className='w-auto p-0'>
        <Calendar
          mode='single'
          captionLayout='dropdown'
          selected={selected}
          defaultMonth={defaultMonth ?? selected}
          startMonth={startMonth}
          endMonth={endMonth}
          onSelect={(date) => {
            onSelect(date)
            if (date) setOpen(false)
          }}
          locale={calendarLocale}
          disabled={
            disabled ??
            ((date: Date) => date > new Date() || date < new Date('1900-01-01'))
          }
        />
      </PopoverContent>
    </Popover>
  )
}
