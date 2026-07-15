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
import { Shield, AlertTriangle, RefreshCw, Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { IconBadge } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'
import { useDialogs } from '@/hooks/use-dialog'

import { useTwoFA } from '../hooks'
import type { TwoFADevice } from '../types'
import { TwoFABackupDialog } from './dialogs/two-fa-backup-dialog'
import { TwoFADeleteDeviceDialog } from './dialogs/two-fa-delete-device-dialog'
import { TwoFADisableDialog } from './dialogs/two-fa-disable-dialog'
import { TwoFASetupDialog } from './dialogs/two-fa-setup-dialog'

interface TwoFACardProps {
  loading: boolean
}

type DialogKey = 'setup' | 'addDevice' | 'disable' | 'backup' | 'deleteDevice'

export function TwoFACard({ loading: pageLoading }: TwoFACardProps) {
  const { t } = useTranslation()
  const { status, loading, refetch } = useTwoFA(!pageLoading)
  const dialogs = useDialogs<DialogKey>()
  const [deviceToDelete, setDeviceToDelete] = useState<TwoFADevice | null>(null)

  const canAddDevice =
    status.enabled && status.device_count < (status.max_devices || 3)

  const handleDeleteDevice = (device: TwoFADevice) => {
    setDeviceToDelete(device)
    dialogs.open('deleteDevice')
  }

  if (pageLoading || loading) {
    return (
      <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
        <CardHeader className='p-3 sm:p-5'>
          <Skeleton className='h-6 w-48' />
          <Skeleton className='mt-2 h-4 w-64' />
        </CardHeader>
        <CardContent className='p-3 sm:p-5'>
          <Skeleton className='h-20 w-full' />
        </CardContent>
      </Card>
    )
  }

  return (
    <>
      <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
        <CardHeader className='p-3 sm:p-5'>
          <CardTitle className='text-lg tracking-tight sm:text-xl'>
            {t('Two-Factor Authentication')}
          </CardTitle>
          <CardDescription className='text-xs sm:text-sm'>
            {t('Add an extra layer of security to your account')}
          </CardDescription>
        </CardHeader>

        <CardContent className='p-3 sm:p-5'>
          <div className='space-y-6'>
            <div className='flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between xl:flex-col 2xl:flex-row'>
              <div className='flex items-start gap-4'>
                <IconBadge tone='success' size='sm'>
                  <Shield />
                </IconBadge>
                <div className='space-y-1'>
                  <div className='flex items-center gap-2'>
                    <p className='font-medium'>{t('Two-Step Verification')}</p>
                    {status.enabled ? (
                      <StatusBadge
                        label={t('Enabled')}
                        variant='success'
                        showDot
                        copyable={false}
                      />
                    ) : (
                      <StatusBadge
                        label={t('Disabled')}
                        variant='neutral'
                        showDot
                        copyable={false}
                      />
                    )}
                    {status.locked && (
                      <StatusBadge
                        label={t('Locked')}
                        variant='danger'
                        showDot
                        copyable={false}
                      />
                    )}
                  </div>
                  <p className='text-muted-foreground text-sm'>
                    {status.enabled
                      ? t('Backup codes remaining: {{count}}', {
                          count: status.backup_codes_remaining,
                        })
                      : t('Add an extra layer of security to your account')}
                  </p>
                  {status.enabled && (
                    <p className='text-muted-foreground text-sm'>
                      {t('Authenticators: {{count}}/{{max}}', {
                        count: status.device_count,
                        max: status.max_devices || 3,
                      })}
                    </p>
                  )}
                </div>
              </div>

              {!status.enabled && (
                <Button
                  className='w-full sm:w-auto xl:w-full 2xl:w-auto'
                  onClick={() => dialogs.open('setup')}
                >
                  {t('Enable')}
                </Button>
              )}
            </div>

            {status.enabled && status.devices.length > 0 && (
              <div className='space-y-3 border-t pt-6'>
                <p className='text-sm font-medium'>{t('Registered Authenticators')}</p>
                <div className='space-y-2'>
                  {status.devices.map((device) => (
                    <div
                      key={device.id}
                      className='flex items-center justify-between rounded-lg border px-3 py-2'
                    >
                      <div>
                        <p className='text-sm font-medium'>{device.label}</p>
                        {device.last_used_at ? (
                          <p className='text-muted-foreground text-xs'>
                            {t('Last used: {{date}}', {
                              date: new Date(device.last_used_at * 1000).toLocaleString(),
                            })}
                          </p>
                        ) : null}
                      </div>
                      {status.device_count > 1 && !device.is_primary && device.id !== 0 ? (
                        <Button
                          variant='ghost'
                          size='icon-sm'
                          onClick={() => handleDeleteDevice(device)}
                          aria-label={t('Remove authenticator')}
                        >
                          <Trash2 className='h-4 w-4' />
                        </Button>
                      ) : null}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {status.enabled && (
              <div className='flex flex-col gap-3 border-t pt-6 sm:flex-row xl:flex-col 2xl:flex-row'>
                {canAddDevice ? (
                  <Button
                    variant='outline'
                    className='flex-1'
                    onClick={() => dialogs.open('addDevice')}
                  >
                    <Plus className='mr-2 h-4 w-4' />
                    {t('Add Authenticator')}
                  </Button>
                ) : null}
                <Button
                  variant='outline'
                  className='flex-1'
                  onClick={() => dialogs.open('backup')}
                >
                  <RefreshCw className='mr-2 h-4 w-4' />
                  {t('Regenerate Backup Codes')}
                </Button>
                <Button
                  variant='destructive'
                  className='flex-1'
                  onClick={() => dialogs.open('disable')}
                >
                  <AlertTriangle className='mr-2 h-4 w-4' />
                  {t('Disable 2FA')}
                </Button>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      <TwoFASetupDialog
        open={dialogs.isOpen('setup')}
        onOpenChange={(open) =>
          open ? dialogs.open('setup') : dialogs.close('setup')
        }
        onSuccess={refetch}
        mode='initial'
      />

      <TwoFASetupDialog
        open={dialogs.isOpen('addDevice')}
        onOpenChange={(open) =>
          open ? dialogs.open('addDevice') : dialogs.close('addDevice')
        }
        onSuccess={refetch}
        mode='additional'
      />

      <TwoFADisableDialog
        open={dialogs.isOpen('disable')}
        onOpenChange={(open) =>
          open ? dialogs.open('disable') : dialogs.close('disable')
        }
        onSuccess={refetch}
      />

      <TwoFABackupDialog
        open={dialogs.isOpen('backup')}
        onOpenChange={(open) =>
          open ? dialogs.open('backup') : dialogs.close('backup')
        }
        onSuccess={refetch}
      />

      <TwoFADeleteDeviceDialog
        open={dialogs.isOpen('deleteDevice')}
        onOpenChange={(open) => {
          if (!open) {
            dialogs.close('deleteDevice')
            setDeviceToDelete(null)
          }
        }}
        device={deviceToDelete}
        onSuccess={refetch}
      />
    </>
  )
}
