import type { AxiosRequestConfig } from 'axios'
import { api } from '@/lib/api'
import type { GetLogsParams } from '../types'

async function parseBlobErrorMessage(blob: Blob): Promise<string> {
  const text = await blob.text()
  try {
    const j = JSON.parse(text) as { message?: string }
    return j.message?.trim() || text || 'Request failed'
  } catch {
    return text || 'Request failed'
  }
}

function triggerBlobDownload(blob: Blob, contentDisposition: string) {
  const dispo = contentDisposition || ''
  const utf8Match = /filename\*=UTF-8''([^;]+)/i.exec(dispo)
  const fallbackMatch = /filename="([^"]+)"/i.exec(dispo)
  const filename =
    (utf8Match?.[1] ? decodeURIComponent(utf8Match[1]) : undefined) ||
    fallbackMatch?.[1] ||
    `usage-logs-${Date.now()}.csv`

  const url = URL.createObjectURL(blob)
  try {
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.rel = 'noopener'
    document.body.appendChild(a)
    a.click()
    a.remove()
  } finally {
    URL.revokeObjectURL(url)
  }
}

/**
 * Download usage logs CSV using the same filters as the list API.
 */
export async function downloadUsageLogsExport(
  params: Omit<GetLogsParams, 'p' | 'page_size'>,
  isAdmin: boolean
): Promise<void> {
  const path = isAdmin ? '/api/log/export' : '/api/log/self/export'
  const query: Record<string, string | number> = {}
  if (params.type != null) query.type = params.type
  if (params.start_timestamp != null) query.start_timestamp = params.start_timestamp
  if (params.end_timestamp != null) query.end_timestamp = params.end_timestamp
  if (params.model_name) query.model_name = params.model_name
  if (params.token_name) query.token_name = params.token_name
  if (params.group) query.group = params.group
  if (params.request_id) query.request_id = params.request_id
  if (isAdmin && params.username) query.username = params.username
  if (isAdmin && params.channel) query.channel = params.channel

  try {
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone
    if (tz) query.timezone = tz
  } catch {
    /* ignore */
  }

  const config = {
    params: query,
    responseType: 'blob' as const,
    skipBusinessError: true,
    skipErrorHandler: true,
    disableDuplicate: true,
  } satisfies AxiosRequestConfig & {
    skipBusinessError?: boolean
    skipErrorHandler?: boolean
    disableDuplicate?: boolean
  }

  const res = await api.get(path, config)
  const ctype = String(res.headers['content-type'] || '')
  if (ctype.includes('application/json')) {
    throw new Error(await parseBlobErrorMessage(res.data as Blob))
  }
  if (res.status >= 400) {
    throw new Error(`HTTP ${res.status}`)
  }

  triggerBlobDownload(
    res.data as Blob,
    String(res.headers['content-disposition'] || '')
  )
}
