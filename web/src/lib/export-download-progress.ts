import type { AxiosProgressEvent } from 'axios'
import { t } from 'i18next'

export function exportDownloadProgressMessage(
  event: AxiosProgressEvent
): string {
  if (event.total && event.total > 0) {
    const percent = Math.min(100, Math.round((event.loaded / event.total) * 100))
    return t('Downloading export {{percent}}%', { percent })
  }
  if (event.loaded > 0) {
    return t('Downloading export...')
  }
  return t('Preparing export...')
}
