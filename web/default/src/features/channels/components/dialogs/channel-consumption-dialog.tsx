import { useCallback, useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { BarChart3, Loader2, RefreshCw, User } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import dayjs from '@/lib/dayjs'
import { formatLogQuota } from '@/lib/format'
import { computeTimeRange } from '@/lib/time'
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
import { CompactDateTimeRangePicker } from '@/features/usage-logs/components/compact-date-time-range-picker'
import { getChannelConsumption } from '../../api'
import { useChannels } from '../channels-provider'

type ChannelConsumptionDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

function defaultMonthRange(): { start: Date; end: Date } {
  const now = dayjs()
  return {
    start: now.startOf('month').toDate(),
    end: now.endOf('day').toDate(),
  }
}

export function ChannelConsumptionDialog({
  open,
  onOpenChange,
}: ChannelConsumptionDialogProps) {
  const { t } = useTranslation()
  const { currentRow } = useChannels()
  const [range, setRange] = useState(defaultMonthRange)
  const [userIdInput, setUserIdInput] = useState('')
  const [usernameInput, setUsernameInput] = useState('')
  const [appliedUserId, setAppliedUserId] = useState<number | undefined>()
  const [appliedUsername, setAppliedUsername] = useState<string | undefined>()

  useEffect(() => {
    if (!open) return
    setRange(defaultMonthRange())
    setUserIdInput('')
    setUsernameInput('')
    setAppliedUserId(undefined)
    setAppliedUsername(undefined)
  }, [open, currentRow?.id])

  const timeParams = useMemo(() => {
    const { start_timestamp, end_timestamp } = computeTimeRange(
      30,
      range.start,
      range.end,
      false
    )
    return { start_timestamp, end_timestamp }
  }, [range.end, range.start])

  const {
    data: consumption,
    isLoading,
    isFetching,
    refetch,
    error,
  } = useQuery({
    queryKey: [
      'channel-consumption',
      currentRow?.id,
      timeParams.start_timestamp,
      timeParams.end_timestamp,
      appliedUserId,
      appliedUsername,
    ],
    queryFn: async () => {
      if (!currentRow) throw new Error(t('No channel selected'))
      const res = await getChannelConsumption(currentRow.id, {
        ...timeParams,
        user_id: appliedUserId,
        username: appliedUsername,
      })
      if (!res.success || !res.data) {
        throw new Error(res.message || t('Failed to load consumption'))
      }
      return res.data
    },
    enabled: open && !!currentRow?.id,
  })

  const applyUserFilter = useCallback(() => {
    const trimmedUsername = usernameInput.trim()
    const parsedUserId = Number.parseInt(userIdInput.trim(), 10)
    if (userIdInput.trim()) {
      if (!Number.isFinite(parsedUserId) || parsedUserId <= 0) {
        setAppliedUserId(undefined)
        setAppliedUsername(undefined)
        return
      }
      setAppliedUserId(parsedUserId)
      setAppliedUsername(undefined)
      return
    }
    setAppliedUserId(undefined)
    setAppliedUsername(trimmedUsername || undefined)
  }, [userIdInput, usernameInput])

  const clearUserFilter = () => {
    setUserIdInput('')
    setUsernameInput('')
    setAppliedUserId(undefined)
    setAppliedUsername(undefined)
  }

  const totalTokens = useMemo(() => {
    if (!consumption) return 0
    return (
      Number(consumption.prompt_tokens || 0) +
      Number(consumption.completion_tokens || 0)
    )
  }, [consumption])

  if (!currentRow) return null

  const userFilterActive = appliedUserId != null || !!appliedUsername

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-w-lg'>
        <DialogHeader>
          <DialogTitle className='flex items-center gap-2'>
            <BarChart3 className='size-5' />
            {t('Channel Consumption')}
          </DialogTitle>
          <DialogDescription>
            {currentRow.name} (#{currentRow.id})
          </DialogDescription>
        </DialogHeader>

        <div className='space-y-4'>
          <div className='space-y-2'>
            <Label>{t('Date Range')}</Label>
            <CompactDateTimeRangePicker
              start={range.start}
              end={range.end}
              onChange={(next) =>
                setRange({
                  start: next.start ?? range.start,
                  end: next.end ?? range.end,
                })
              }
            />
          </div>

          <div className='space-y-2 rounded-lg border border-border/60 p-3'>
            <div className='flex items-center gap-2 text-sm font-medium'>
              <User className='size-4' />
              {t('Filter by user (optional)')}
            </div>
            <div className='grid gap-2 sm:grid-cols-2'>
              <div className='space-y-1'>
                <Label className='text-muted-foreground text-xs'>
                  {t('User ID')}
                </Label>
                <Input
                  value={userIdInput}
                  onChange={(e) => setUserIdInput(e.target.value)}
                  placeholder='123'
                  className='h-8 font-mono text-xs'
                />
              </div>
              <div className='space-y-1'>
                <Label className='text-muted-foreground text-xs'>
                  {t('Username')}
                </Label>
                <Input
                  value={usernameInput}
                  onChange={(e) => setUsernameInput(e.target.value)}
                  placeholder='alice'
                  className='h-8 text-xs'
                />
              </div>
            </div>
            <div className='flex flex-wrap gap-2'>
              <Button
                type='button'
                size='sm'
                variant='secondary'
                onClick={applyUserFilter}
              >
                {t('Apply user filter')}
              </Button>
              {userFilterActive && (
                <Button
                  type='button'
                  size='sm'
                  variant='ghost'
                  onClick={clearUserFilter}
                >
                  {t('Clear user filter')}
                </Button>
              )}
            </div>
            {userFilterActive && (
              <p className='text-muted-foreground text-xs'>
                {appliedUserId != null
                  ? `${t('User ID')}: ${appliedUserId}`
                  : `${t('Username')}: ${appliedUsername}`}
              </p>
            )}
          </div>

          <div className='rounded-lg border border-border/60 bg-muted/20 p-4'>
            {isLoading ? (
              <div className='text-muted-foreground flex items-center justify-center gap-2 py-6 text-sm'>
                <Loader2 className='size-4 animate-spin' />
                {t('Loading...')}
              </div>
            ) : error ? (
              <p className='text-destructive text-sm'>
                {error instanceof Error ? error.message : t('Failed to load')}
              </p>
            ) : (
              <dl className='grid gap-3 text-sm'>
                <div className='flex items-center justify-between gap-4'>
                  <dt className='text-muted-foreground'>{t('Usage')}</dt>
                  <dd className='font-mono font-semibold tabular-nums'>
                    {formatLogQuota(Number(consumption?.quota || 0))}
                  </dd>
                </div>
                <div className='flex items-center justify-between gap-4'>
                  <dt className='text-muted-foreground'>
                    {t('Total requests')}
                  </dt>
                  <dd className='font-mono tabular-nums'>
                    {consumption?.request_count ?? 0}
                  </dd>
                </div>
                <div className='flex items-center justify-between gap-4'>
                  <dt className='text-muted-foreground'>{t('Total tokens')}</dt>
                  <dd className='font-mono tabular-nums'>{totalTokens}</dd>
                </div>
                {!userFilterActive && (
                  <div className='flex items-center justify-between gap-4 border-t border-border/50 pt-3'>
                    <dt className='text-muted-foreground'>
                      {t('Lifetime channel usage')}
                    </dt>
                    <dd className='font-mono tabular-nums text-xs'>
                      {formatLogQuota(
                        Number(consumption?.lifetime_used_quota || 0)
                      )}
                    </dd>
                  </div>
                )}
              </dl>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => refetch()}
            disabled={isFetching}
          >
            {isFetching ? (
              <Loader2 className='mr-2 size-4 animate-spin' />
            ) : (
              <RefreshCw className='mr-2 size-4' />
            )}
            {t('Refresh')}
          </Button>
          <Button type='button' onClick={() => onOpenChange(false)}>
            {t('Close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}