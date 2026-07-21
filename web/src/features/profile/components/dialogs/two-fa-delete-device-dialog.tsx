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
import { useState } from 'react'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { delete2FADevice } from '@/lib/api'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { TwoFADevice } from '../../types'

interface TwoFADeleteDeviceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  device: TwoFADevice | null
  onSuccess: () => void
}

export function TwoFADeleteDeviceDialog({
  open,
  onOpenChange,
  device,
  onSuccess,
}: TwoFADeleteDeviceDialogProps) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [code, setCode] = useState('')

  const handleDelete = async () => {
    if (!device || !code) {
      toast.error(t('Please enter the verification code'))
      return
    }

    try {
      setLoading(true)
      const response = await delete2FADevice(device.id, code)

      if (response.success) {
        toast.success(t('Authenticator removed successfully'))
        onOpenChange(false)
        onSuccess()
        setCode('')
      } else {
        toast.error(response.message || t('Failed to remove authenticator'))
      }
    } catch (_error) {
      toast.error(t('Failed to remove authenticator'))
    } finally {
      setLoading(false)
    }
  }

  const handleOpenChange = (nextOpen: boolean) => {
    if (!loading) {
      if (!nextOpen) {
        setCode('')
      }
      onOpenChange(nextOpen)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>{t('Remove Authenticator')}</DialogTitle>
          <DialogDescription>
            {t('Enter a verification code from any remaining authenticator to remove {{label}}.', {
              label: device?.label ?? t('this device'),
            })}
          </DialogDescription>
        </DialogHeader>

        <div className='space-y-2 py-2'>
          <Label htmlFor='delete-device-code'>{t('Verification Code')}</Label>
          <Input
            id='delete-device-code'
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder={t('Enter 6-digit code or backup code')}
            disabled={loading}
          />
        </div>

        <DialogFooter>
          <Button
            variant='outline'
            onClick={() => handleOpenChange(false)}
            disabled={loading}
          >
            {t('Cancel')}
          </Button>
          <Button
            variant='destructive'
            onClick={handleDelete}
            disabled={loading || !code}
          >
            {loading && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
            {t('Remove')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
