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
import { useState, useEffect, useCallback, useMemo } from 'react'
import { Loader2 } from 'lucide-react'
import { QRCodeSVG } from 'qrcode.react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { setup2FA, enable2FA } from '@/lib/api'
import { Alert, AlertDescription } from '@/components/ui/alert'
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
import { CopyButton } from '@/components/copy-button'
import type { TwoFASetupData } from '../../types'

interface TwoFASetupDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
  mode?: 'initial' | 'additional'
}

export function TwoFASetupDialog({
  open,
  onOpenChange,
  onSuccess,
  mode = 'initial',
}: TwoFASetupDialogProps) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [initializing, setInitializing] = useState(false)
  const [step, setStep] = useState(0)
  const [setupData, setSetupData] = useState<TwoFASetupData | null>(null)
  const [code, setCode] = useState('')

  const isAdditional = mode === 'additional' || setupData?.is_additional === true
  const stepLabels = useMemo(
    () =>
      isAdditional
        ? [t('Scan QR Code'), t('Verify Setup')]
        : [t('Scan QR Code'), t('Save Backup Codes'), t('Verify Setup')],
    [isAdditional, t]
  )
  const verifyStep = isAdditional ? 1 : 2

  const handleSetup = useCallback(async () => {
    try {
      setInitializing(true)
      const response = await setup2FA()

      if (response.success && response.data) {
        setSetupData(response.data)
        setStep(0)
      } else {
        toast.error(response.message || t('Failed to setup 2FA'))
        onOpenChange(false)
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Setup 2FA error:', error)
      toast.error(t('Failed to setup 2FA'))
      onOpenChange(false)
    } finally {
      setInitializing(false)
    }
  }, [onOpenChange, t])

  const handleEnable = async () => {
    if (!code || !setupData) {
      toast.error(t('Please enter the verification code'))
      return
    }

    try {
      setLoading(true)
      const response = await enable2FA(code, setupData.device_id)

      if (response.success) {
        toast.success(
          isAdditional
            ? t('Authenticator added successfully')
            : t('Two-factor authentication enabled successfully!')
        )
        onOpenChange(false)
        onSuccess()
        setStep(0)
        setCode('')
        setSetupData(null)
      } else {
        toast.error(response.message || t('Failed to enable 2FA'))
      }
    } catch (_error) {
      toast.error(t('Failed to enable 2FA'))
    } finally {
      setLoading(false)
    }
  }

  const handleOpenChange = (nextOpen: boolean) => {
    if (!loading && !initializing) {
      if (nextOpen && !setupData) {
        handleSetup()
      }
      if (!nextOpen) {
        setStep(0)
        setCode('')
        setSetupData(null)
      }
      onOpenChange(nextOpen)
    }
  }

  useEffect(() => {
    if (open && !setupData && !initializing) {
      handleSetup()
    }
  }, [open, setupData, initializing, handleSetup])

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>
            {isAdditional
              ? t('Add Authenticator')
              : t('Setup Two-Factor Authentication')}
          </DialogTitle>
          <DialogDescription>
            {t('Step')} {step + 1} {t('of {{total}}:', { total: stepLabels.length })}{' '}
            {stepLabels[step]}
          </DialogDescription>
        </DialogHeader>

        <div className='space-y-4 py-4'>
          {initializing ? (
            <div className='flex flex-col items-center justify-center gap-3 py-8'>
              <div className='border-primary h-8 w-8 animate-spin rounded-full border-4 border-t-transparent' />
              <div className='text-muted-foreground text-sm'>
                {t('Setting up 2FA...')}
              </div>
            </div>
          ) : !setupData ? (
            <div className='flex justify-center py-8'>
              <div className='text-muted-foreground'>
                {t('Failed to load setup data')}
              </div>
            </div>
          ) : (
            <>
              {step === 0 && (
                <div className='space-y-4'>
                  <p className='text-muted-foreground text-sm'>
                    {t(
                      'Scan this QR code with your authenticator app (Google Authenticator, Microsoft Authenticator, etc.)'
                    )}
                  </p>
                  <div className='flex justify-center rounded-lg bg-white p-4'>
                    <QRCodeSVG value={setupData.qr_code_data} size={200} />
                  </div>
                  <div className='bg-muted rounded-lg p-3'>
                    <div className='flex items-center justify-between'>
                      <div>
                        <p className='text-muted-foreground text-xs'>
                          {t('Or enter this key manually:')}
                        </p>
                        <code className='font-mono text-sm'>
                          {setupData.secret}
                        </code>
                      </div>
                      <CopyButton
                        value={setupData.secret}
                        variant='ghost'
                        tooltip={t('Copy secret key')}
                        aria-label={t('Copy secret key')}
                      />
                    </div>
                  </div>
                </div>
              )}

              {!isAdditional && step === 1 && (
                <div className='space-y-4'>
                  <Alert>
                    <AlertDescription>
                      {t(
                        'Save these backup codes in a safe place. Each code can only be used once.'
                      )}
                    </AlertDescription>
                  </Alert>
                  <div className='rounded-lg border p-4'>
                    <div className='grid grid-cols-2 gap-2'>
                      {setupData.backup_codes.map((backupCode, index) => (
                        <div
                          key={index}
                          className='bg-muted rounded-md p-2 text-center font-mono text-sm'
                        >
                          {backupCode}
                        </div>
                      ))}
                    </div>
                  </div>
                  <CopyButton
                    value={setupData.backup_codes.join('\n')}
                    variant='outline'
                    size='default'
                    className='w-full'
                    iconClassName='mr-2 size-4'
                    tooltip={t('Copy all backup codes')}
                    aria-label={t('Copy all backup codes')}
                  >
                    {t('Copy All Codes')}
                  </CopyButton>
                </div>
              )}

              {step === verifyStep && (
                <div className='space-y-4'>
                  <div className='space-y-2'>
                    <Label htmlFor='code'>{t('Verification Code')}</Label>
                    <Input
                      id='code'
                      value={code}
                      onChange={(e) => setCode(e.target.value)}
                      placeholder={t('Enter 6-digit code')}
                      maxLength={6}
                      disabled={loading}
                    />
                    <p className='text-muted-foreground text-xs'>
                      {t('Enter the 6-digit code from your authenticator app')}
                    </p>
                  </div>
                </div>
              )}
            </>
          )}
        </div>

        <DialogFooter>
          {step > 0 && (
            <Button
              variant='outline'
              onClick={() => setStep(step - 1)}
              disabled={initializing || loading}
            >
              {t('Back')}
            </Button>
          )}
          {step < verifyStep ? (
            <Button
              onClick={() => setStep(step + 1)}
              disabled={initializing || !setupData}
            >
              {t('Next')}
            </Button>
          ) : (
            <Button
              onClick={handleEnable}
              disabled={initializing || loading || !code}
            >
              {loading && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
              {loading
                ? isAdditional
                  ? t('Adding...')
                  : t('Enabling...')
                : isAdditional
                  ? t('Add Authenticator')
                  : t('Enable 2FA')}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
