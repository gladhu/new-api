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
import { Download, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import type { UsageLogExportDetail } from '../lib/export'

interface UsageLogExportButtonProps {
  isAdmin: boolean
  disabled: boolean
  exporting: boolean
  onExport: (detail: UsageLogExportDetail) => void
}

export function UsageLogExportButton(props: UsageLogExportButtonProps) {
  const { t } = useTranslation()
  const busy = props.disabled || props.exporting
  const icon = props.exporting ? (
    <Loader2 className='animate-spin' />
  ) : (
    <Download className='size-3.5' />
  )

  if (!props.isAdmin) {
    return (
      <Button
        type='button'
        variant='outline'
        onClick={() => props.onExport('simple')}
        disabled={busy}
      >
        {icon}
        {t('Export logs')}
      </Button>
    )
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button type='button' variant='outline' disabled={busy} />
        }
      >
        {icon}
        {t('Export logs')}
      </DropdownMenuTrigger>
      <DropdownMenuContent align='end' className='w-72'>
        <DropdownMenuItem
          disabled={busy}
          onClick={() => props.onExport('simple')}
        >
          <span className='flex min-w-0 flex-col items-start gap-0.5 py-0.5 whitespace-normal'>
            <span>{t('Simple logs')}</span>
            <span className='text-muted-foreground text-xs font-normal'>
              {t('Billing columns only')}
            </span>
          </span>
        </DropdownMenuItem>
        <DropdownMenuItem
          disabled={busy}
          onClick={() => props.onExport('full')}
        >
          <span className='flex min-w-0 flex-col items-start gap-0.5 py-0.5 whitespace-normal'>
            <span>{t('Detailed logs')}</span>
            <span className='text-muted-foreground text-xs font-normal'>
              {t(
                'Include user, channel, request content, and raw details'
              )}
            </span>
          </span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
