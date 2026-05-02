import { useEffect, useState } from 'react'
import { FileSpreadsheet, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { downloadAdminUserLogExport } from '../../api'

interface UserBillExportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  userId: number
  username: string
}

export function UserBillExportDialog({
  open,
  onOpenChange,
  userId,
  username,
}: UserBillExportDialogProps) {
  const { t } = useTranslation()
  const now = new Date()
  const [year, setYear] = useState(now.getFullYear())
  const [month, setMonth] = useState(now.getMonth() + 1)
  const [timezone, setTimezone] = useState('')
  const [loadingBill, setLoadingBill] = useState(false)
  const [loadingDetails, setLoadingDetails] = useState(false)
  const [loadingAll, setLoadingAll] = useState(false)

  useEffect(() => {
    if (!open) return
    const d = new Date()
    setYear(d.getFullYear())
    setMonth(d.getMonth() + 1)
    setTimezone('')
  }, [open])

  const runExport = async (
    kind:
      | 'monthly_bill'
      | 'consumption_details'
      | 'monthly_bill_and_consumption_details',
    setBusy: (v: boolean) => void
  ) => {
    if (!Number.isFinite(year) || year < 1970 || year > 9999) {
      toast.error(t('Invalid year'))
      return
    }
    if (month < 1 || month > 12) {
      toast.error(t('Invalid month'))
      return
    }
    setBusy(true)
    try {
      await downloadAdminUserLogExport(kind, {
        userId,
        year,
        month,
        timezone: timezone.trim() || undefined,
      })
      toast.success(t('Download started'))
      onOpenChange(false)
    } catch (e) {
      const msg = e instanceof Error ? e.message : t('Export failed')
      toast.error(msg)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>{t('Export usage CSV')}</DialogTitle>
          <DialogDescription>
            {t('Export usage CSV for user {{name}} (ID {{id}})', {
              name: username,
              id: userId,
            })}
          </DialogDescription>
        </DialogHeader>
        <div className='grid gap-4 py-2'>
          <div className='grid gap-2'>
            <Label htmlFor='export-year'>{t('Year')}</Label>
            <Input
              id='export-year'
              type='number'
              min={1970}
              max={9999}
              value={year}
              onChange={(e) => setYear(Number(e.target.value))}
            />
          </div>
          <div className='grid gap-2'>
            <Label>{t('Month')}</Label>
            <Select
              value={String(month)}
              onValueChange={(v) => setMonth(Number(v))}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {Array.from({ length: 12 }, (_, i) => i + 1).map((m) => (
                  <SelectItem key={m} value={String(m)}>
                    {m}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='export-tz'>{t('Timezone (IANA, optional)')}</Label>
            <Input
              id='export-tz'
              placeholder={t('e.g. Asia/Shanghai')}
              value={timezone}
              onChange={(e) => setTimezone(e.target.value)}
            />
            <p className='text-muted-foreground text-xs'>
              {t('Month range uses server local time unless a timezone is set.')}
            </p>
          </div>
        </div>
        <DialogFooter className='flex-col gap-2 sm:flex-col'>
          <Button
            type='button'
            variant='secondary'
            disabled={loadingBill || loadingDetails || loadingAll}
            onClick={() => runExport('monthly_bill', setLoadingBill)}
          >
            {loadingBill ? (
              <Loader2 className='mr-2 h-4 w-4 animate-spin' />
            ) : (
              <FileSpreadsheet className='mr-2 h-4 w-4' />
            )}
            {t('Export monthly bill')}
          </Button>
          <Button
            type='button'
            variant='secondary'
            disabled={loadingBill || loadingDetails || loadingAll}
            onClick={() => runExport('consumption_details', setLoadingDetails)}
          >
            {loadingDetails ? (
              <Loader2 className='mr-2 h-4 w-4 animate-spin' />
            ) : (
              <FileSpreadsheet className='mr-2 h-4 w-4' />
            )}
            {t('Export consumption details')}
          </Button>
          <Button
            type='button'
            disabled={loadingBill || loadingDetails || loadingAll}
            onClick={() =>
              runExport(
                'monthly_bill_and_consumption_details',
                setLoadingAll
              )
            }
          >
            {loadingAll ? (
              <Loader2 className='mr-2 h-4 w-4 animate-spin' />
            ) : (
              <FileSpreadsheet className='mr-2 h-4 w-4' />
            )}
            {t('Export monthly bill and consumption details')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
