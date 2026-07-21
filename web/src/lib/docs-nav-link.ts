/**
 * 将后台配置的文档链接解析为顶部导航使用的 href，并判断是否在站内打开。
 * 与当前浏览器 origin 或 {@link serverAddress} 同源时用 SPA 路由，否则新开标签页。
 */
export function resolveDocsNavLink(
  docsLink: string,
  serverAddress?: string
): { href: string; external: boolean } {
  const trimmed = docsLink.trim()
  if (!trimmed) {
    return { href: '/docs', external: false }
  }

  if (trimmed.startsWith('/') && !trimmed.startsWith('//')) {
    return { href: trimmed, external: false }
  }

  let absolute: URL
  try {
    if (trimmed.startsWith('//')) {
      absolute = new URL(`https:${trimmed}`)
    } else if (!/^https?:\/\//i.test(trimmed)) {
      const base =
        (typeof window !== 'undefined' && window.location.origin) ||
        originFromServerAddress(serverAddress) ||
        'http://localhost'
      const normalizedBase = base.endsWith('/') ? base : `${base}/`
      absolute = new URL(trimmed, normalizedBase)
    } else {
      absolute = new URL(trimmed)
    }
  } catch {
    return { href: trimmed, external: true }
  }

  const browserOrigin =
    typeof window !== 'undefined' ? window.location.origin : undefined
  const sameAsBrowser = Boolean(
    browserOrigin && browserOrigin === absolute.origin
  )

  const srvOrigin = originFromServerAddress(serverAddress)
  const sameAsServer = Boolean(srvOrigin && srvOrigin === absolute.origin)

  if (sameAsBrowser || sameAsServer) {
    return {
      href: `${absolute.pathname}${absolute.search}${absolute.hash}`,
      external: false,
    }
  }

  return { href: absolute.href, external: true }
}

function originFromServerAddress(serverAddress?: string): string | undefined {
  if (!serverAddress?.trim()) return undefined
  try {
    return new URL(serverAddress.trim()).origin
  } catch {
    return undefined
  }
}
