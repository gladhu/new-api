/*
Copyright (C) 2025

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

import { API } from './api';

/**
 * @param {'monthly_bill'|'consumption_details'|'monthly_bill_and_consumption_details'} kind
 * @param {{ userId: number, year: number, month: number, timezone?: string }} params
 */
export async function downloadAdminUserLogExport(kind, params) {
  const pathMap = {
    monthly_bill: '/api/log/admin/export/monthly_bill',
    consumption_details: '/api/log/admin/export/consumption_details',
    monthly_bill_and_consumption_details:
      '/api/log/admin/export/monthly_bill_and_consumption_details',
  };
  const path = pathMap[kind];
  const query = {
    user_id: params.userId,
    year: params.year,
    month: params.month,
  };
  if (params.timezone && String(params.timezone).trim()) {
    query.timezone = String(params.timezone).trim();
  }
  const res = await API.get(path, {
    params: query,
    responseType: 'blob',
    skipErrorHandler: true,
  });
  const ctype = String(res.headers['content-type'] || '');
  if (ctype.includes('application/json')) {
    const text = await res.data.text();
    let msg = text;
    try {
      const j = JSON.parse(text);
      if (j.message) msg = j.message;
    } catch {
      /* keep text */
    }
    throw new Error(msg);
  }
  const dispo = String(res.headers['content-disposition'] || '');
  const utf8Match = /filename\*=UTF-8''([^;]+)/i.exec(dispo);
  const fallbackMatch = /filename="([^"]+)"/i.exec(dispo);
  const filename =
    (utf8Match?.[1] ? decodeURIComponent(utf8Match[1]) : undefined) ||
    fallbackMatch?.[1] ||
    `export-${params.userId}-${params.year}-${String(params.month).padStart(2, '0')}.csv`;
  const url = URL.createObjectURL(res.data);
  try {
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.rel = 'noopener';
    document.body.appendChild(a);
    a.click();
    a.remove();
  } finally {
    URL.revokeObjectURL(url);
  }
}
