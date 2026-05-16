/*
Copyright (C) 2025 QuantumNous

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
 * @param {boolean} isAdminUser
 * @param {{
 *   logType?: number,
 *   username?: string,
 *   token_name?: string,
 *   model_name?: string,
 *   start_timestamp: number,
 *   end_timestamp: number,
 *   channel?: string|number,
 *   group?: string,
 *   request_id?: string,
 * }} filters
 */
export async function downloadUsageLogsExport(isAdminUser, filters) {
  const path = isAdminUser ? '/api/log/export' : '/api/log/self/export';
  const query = {
    type: filters.logType ?? 0,
    start_timestamp: filters.start_timestamp,
    end_timestamp: filters.end_timestamp,
    model_name: filters.model_name || '',
    token_name: filters.token_name || '',
    group: filters.group || '',
    request_id: filters.request_id || '',
  };
  if (isAdminUser) {
    if (filters.username) query.username = filters.username;
    if (filters.channel) query.channel = filters.channel;
  }
  try {
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
    if (tz) query.timezone = tz;
  } catch {
    /* ignore */
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
    `usage-logs-${Date.now()}.csv`;

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
