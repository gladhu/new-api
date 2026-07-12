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

function toRelativePath(url) {
  return `${url.pathname}${url.search}${url.hash}`;
}

export function resolveServerAddressForHome(serverAddress) {
  const trimmed = String(serverAddress || '').trim().replace(/\/$/, '');
  if (trimmed && trimmed !== window.location.origin) {
    return trimmed;
  }

  try {
    const raw = localStorage.getItem('status');
    if (raw) {
      const status = JSON.parse(raw);
      const cached = status?.server_address;
      if (cached) {
        return String(cached).trim().replace(/\/$/, '');
      }
    }
  } catch {
    /* empty */
  }

  return window.location.origin;
}

/** Rewrite primary-domain absolute URLs to site-relative paths for CDN mirrors. */
export function normalizeHomeContentSource(source, serverAddress) {
  const trimmed = String(source || '').trim();
  if (!trimmed || trimmed.startsWith('/')) {
    return trimmed;
  }
  if (!trimmed.startsWith('http://') && !trimmed.startsWith('https://')) {
    return trimmed;
  }

  const effectiveServerAddress = resolveServerAddressForHome(serverAddress);

  try {
    const url = new URL(trimmed);
    if (url.origin === window.location.origin) {
      return toRelativePath(url);
    }

    if (effectiveServerAddress) {
      const serverOrigin = new URL(effectiveServerAddress).origin;
      if (url.origin === serverOrigin) {
        return toRelativePath(url);
      }
    }
  } catch {
    return trimmed;
  }

  return trimmed;
}

export function isHomePageEmbedSource(value) {
  const trimmed = String(value || '').trim();
  return (
    trimmed.startsWith('http://') ||
    trimmed.startsWith('https://') ||
    trimmed.startsWith('/')
  );
}

export function resolveHomePageEmbedSrc(content, serverAddress) {
  const source = normalizeHomeContentSource(content, serverAddress);
  const trimmed = String(source || '').trim();
  if (!trimmed) return '';

  if (trimmed.startsWith('/')) {
    return `${window.location.origin}${trimmed}`;
  }

  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
    return trimmed;
  }

  return '';
}
