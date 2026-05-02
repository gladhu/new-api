import type { AxiosRequestConfig } from 'axios'
import { api } from '@/lib/api'
import type {
  User,
  GetUsersParams,
  GetUsersResponse,
  SearchUsersParams,
  UserFormData,
  ManageUserAction,
  ManageUserQuotaPayload,
  ApiResponse,
} from './types'

// ============================================================================
// User Management APIs
// ============================================================================

/**
 * Get paginated users list
 */
export async function getUsers(
  params: GetUsersParams = {}
): Promise<GetUsersResponse> {
  const { p = 1, page_size = 10 } = params
  const res = await api.get(`/api/user/?p=${p}&page_size=${page_size}`)
  return res.data
}

/**
 * Search users by keyword or group
 */
export async function searchUsers(
  params: SearchUsersParams
): Promise<GetUsersResponse> {
  const { keyword = '', group = '', p = 1, page_size = 10 } = params
  const res = await api.get(
    `/api/user/search?keyword=${keyword}&group=${group}&p=${p}&page_size=${page_size}`
  )
  return res.data
}

/**
 * Get single user by ID
 */
export async function getUser(id: number): Promise<ApiResponse<User>> {
  const res = await api.get(`/api/user/${id}`)
  return res.data
}

/**
 * Create a new user
 */
export async function createUser(
  data: UserFormData
): Promise<ApiResponse<User>> {
  const res = await api.post('/api/user/', data)
  return res.data
}

/**
 * Update an existing user
 */
export async function updateUser(
  data: UserFormData & { id: number }
): Promise<ApiResponse<Partial<User>>> {
  const res = await api.put('/api/user/', data)
  return res.data
}

/**
 * Delete a single user (hard delete)
 */
export async function deleteUser(id: number): Promise<ApiResponse> {
  const res = await api.delete(`/api/user/${id}/`)
  return res.data
}

/**
 * Manage user (promote, demote, enable, disable, delete)
 */
export async function manageUser(
  id: number,
  action: ManageUserAction
): Promise<ApiResponse<Partial<User>>> {
  const res = await api.post('/api/user/manage', { id, action })
  return res.data
}

/**
 * Adjust user quota atomically (add/subtract/override)
 */
export async function adjustUserQuota(
  payload: ManageUserQuotaPayload
): Promise<ApiResponse<Partial<User>>> {
  const res = await api.post('/api/user/manage', payload)
  return res.data
}

/**
 * Reset user's Passkey registration
 */
export async function resetUserPasskey(id: number): Promise<ApiResponse> {
  const res = await api.delete(`/api/user/${id}/reset_passkey`)
  return res.data
}

/**
 * Reset user's Two-Factor Authentication setup
 */
export async function resetUserTwoFA(id: number): Promise<ApiResponse> {
  const res = await api.delete(`/api/user/${id}/2fa`)
  return res.data
}

/**
 * Get all available groups
 */
export async function getGroups(): Promise<ApiResponse<string[]>> {
  const res = await api.get('/api/group/')
  return res.data
}

// ============================================================================
// Admin Binding Management APIs
// ============================================================================

export interface OAuthBinding {
  provider_id: string
  provider_name: string
  user_id?: number
  external_id?: string
}

/**
 * Get user's custom OAuth bindings (admin)
 */
export async function getUserOAuthBindings(
  userId: number
): Promise<ApiResponse<OAuthBinding[]>> {
  const res = await api.get(`/api/user/${userId}/oauth/bindings`)
  return res.data
}

/**
 * Clear a user's built-in binding (admin)
 */
export async function adminClearUserBinding(
  userId: number,
  bindingType: string
): Promise<ApiResponse> {
  const res = await api.delete(`/api/user/${userId}/bindings/${bindingType}`)
  return res.data
}

/**
 * Unbind custom OAuth for a user (admin)
 */
export async function adminUnbindCustomOAuth(
  userId: number,
  providerId: string
): Promise<ApiResponse> {
  const res = await api.delete(
    `/api/user/${userId}/oauth/bindings/${providerId}`
  )
  return res.data
}

// ============================================================================
// Admin: user usage CSV export (logs DB)
// ============================================================================

export type AdminUserLogExportKind =
  | 'monthly_bill'
  | 'consumption_details'
  | 'monthly_bill_and_consumption_details'

async function parseBlobErrorMessage(blob: Blob): Promise<string> {
  const text = await blob.text()
  try {
    const j = JSON.parse(text) as { message?: string }
    return j.message?.trim() || text || 'Request failed'
  } catch {
    return text || 'Request failed'
  }
}

/**
 * Download CSV (monthly bill summary or consumption line items) for a user.
 */
export async function downloadAdminUserLogExport(
  kind: AdminUserLogExportKind,
  params: { userId: number; year: number; month: number; timezone?: string }
): Promise<void> {
  const pathMap: Record<AdminUserLogExportKind, string> = {
    monthly_bill: '/api/log/admin/export/monthly_bill',
    consumption_details: '/api/log/admin/export/consumption_details',
    monthly_bill_and_consumption_details:
      '/api/log/admin/export/monthly_bill_and_consumption_details',
  }
  const path = pathMap[kind]
  const config = {
    params: {
      user_id: params.userId,
      year: params.year,
      month: params.month,
      ...(params.timezone ? { timezone: params.timezone } : {}),
    },
    responseType: 'blob' as const,
    skipBusinessError: true,
    skipErrorHandler: true,
  } satisfies AxiosRequestConfig & {
    skipBusinessError?: boolean
    skipErrorHandler?: boolean
  }
  const res = await api.get(path, config)

  const ctype = String(res.headers['content-type'] || '')
  if (ctype.includes('application/json')) {
    const msg = await parseBlobErrorMessage(res.data as Blob)
    throw new Error(msg)
  }
  if (res.status >= 400) {
    throw new Error(`HTTP ${res.status}`)
  }

  const blob = res.data as Blob
  const dispo = String(res.headers['content-disposition'] || '')
  const utf8Match = /filename\*=UTF-8''([^;]+)/i.exec(dispo)
  const fallbackMatch = /filename="([^"]+)"/i.exec(dispo)
  const filename =
    (utf8Match?.[1] ? decodeURIComponent(utf8Match[1]) : undefined) ||
    fallbackMatch?.[1] ||
    `export-${params.userId}-${params.year}-${String(params.month).padStart(2, '0')}.csv`

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
