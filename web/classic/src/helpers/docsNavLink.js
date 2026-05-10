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

function originFromServerAddress(serverAddress) {
  if (!serverAddress || !String(serverAddress).trim()) return undefined;
  try {
    return new URL(String(serverAddress).trim()).origin;
  } catch {
    return undefined;
  }
}

/**
 * @param {string} docsLink
 * @param {string} [serverAddress]
 * @returns {{ href: string, external: boolean }}
 */
export function resolveDocsNavLink(docsLink, serverAddress) {
  const trimmed = String(docsLink).trim();
  if (!trimmed) {
    return { href: '/docs', external: false };
  }

  if (trimmed.startsWith('/') && !trimmed.startsWith('//')) {
    return { href: trimmed, external: false };
  }

  let absolute;
  try {
    if (trimmed.startsWith('//')) {
      absolute = new URL(`https:${trimmed}`);
    } else if (!/^https?:\/\//i.test(trimmed)) {
      const base =
        (typeof window !== 'undefined' && window.location.origin) ||
        originFromServerAddress(serverAddress) ||
        'http://localhost';
      const normalizedBase = base.endsWith('/') ? base : `${base}/`;
      absolute = new URL(trimmed, normalizedBase);
    } else {
      absolute = new URL(trimmed);
    }
  } catch {
    return { href: trimmed, external: true };
  }

  const browserOrigin = typeof window !== 'undefined' ? window.location.origin : undefined;
  const sameAsBrowser = Boolean(browserOrigin && browserOrigin === absolute.origin);

  const srvOrigin = originFromServerAddress(serverAddress);
  const sameAsServer = Boolean(srvOrigin && srvOrigin === absolute.origin);

  if (sameAsBrowser || sameAsServer) {
    return {
      href: `${absolute.pathname}${absolute.search}${absolute.hash}`,
      external: false,
    };
  }

  return { href: absolute.href, external: true };
}
