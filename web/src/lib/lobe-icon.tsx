/*
Copyright (C) 2023-2026 QuantumNous

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
/**
 * LobeHub Icon Loader
 * Dynamically load and render icons from @lobehub/icons
 *
 * Supports:
 * - Basic: "OpenAI", "OpenAI.Color"
 * - Chained properties: "OpenAI.Avatar.type={'platform'}"
 * - Size parameter: getLobeIcon("OpenAI", 20)
 */
/* eslint-disable react/only-export-components */
import { useEffect, useState, type ComponentType, type ReactNode } from 'react'

type LobeIconComponent = ComponentType<Record<string, unknown>> &
  Record<string, unknown>

// Icon modules live in one directory per brand, e.g. @lobehub/icons/es/OpenAI.
// Vendor icon keys come from admin-editable data, so reject anything that cannot
// be a directory name before it reaches the bundler's dynamic import.
const ICON_MODULE_NAME = /^[A-Za-z][A-Za-z0-9]*$/

const lobeIconPromises = new Map<string, Promise<LobeIconComponent | null>>()

function loadLobeIcon(baseKey: string): Promise<LobeIconComponent | null> {
  if (!ICON_MODULE_NAME.test(baseKey)) return Promise.resolve(null)

  const existing = lobeIconPromises.get(baseKey)
  if (existing) return existing

  // The static prefix and suffix keep this a per-brand lazy chunk instead of
  // pulling the whole icon library into the importing route. webpackInclude is
  // matched against absolute resource paths and restricts the generated context
  // to brand directories, since es/components/* holds demo widgets that depend
  // on packages this app does not install.
  const pending = import(
    /* webpackInclude: /icons[\\/]es[\\/][A-Za-z0-9]+[\\/]index\.js$/ */
    `@lobehub/icons/es/${baseKey}/index.js`
  )
    .then((mod) => mod.default as LobeIconComponent)
    .catch(() => {
      lobeIconPromises.delete(baseKey)
      return null
    })
  lobeIconPromises.set(baseKey, pending)
  return pending
}

function parseValue(raw: string | undefined | null): string | number | boolean {
  if (raw == null) return true

  let v = String(raw).trim()

  if (v.startsWith('{') && v.endsWith('}')) {
    v = v.slice(1, -1).trim()
  }

  if (
    (v.startsWith('"') && v.endsWith('"')) ||
    (v.startsWith("'") && v.endsWith("'"))
  ) {
    return v.slice(1, -1)
  }

  if (v === 'true') return true
  if (v === 'false') return false

  if (/^-?\d+(?:\.\d+)?$/.test(v)) return Number(v)

  return v
}

function renderFallback(iconName: string | undefined | null, size: number) {
  const label =
    iconName && typeof iconName === 'string' && iconName.trim()
      ? iconName.trim().charAt(0).toUpperCase()
      : '?'

  return (
    <div
      className='bg-muted text-muted-foreground flex items-center justify-center rounded-full text-xs font-medium'
      style={{ width: size, height: size }}
    >
      {label}
    </div>
  )
}

function renderLobeIcon(
  BaseIcon: LobeIconComponent,
  iconName: string,
  size: number
): ReactNode {
  const segments = iconName.split('.')

  let IconComponent: LobeIconComponent | undefined
  let propStartIndex: number

  if (segments.length > 1 && BaseIcon[segments[1]]) {
    IconComponent = BaseIcon[segments[1]] as LobeIconComponent
    propStartIndex = 2
  } else {
    IconComponent = BaseIcon
    propStartIndex = segments.length > 1 && /^[A-Z]/.test(segments[1]) ? 2 : 1
  }

  if (
    !IconComponent ||
    (typeof IconComponent !== 'function' && typeof IconComponent !== 'object')
  ) {
    return renderFallback(iconName, size)
  }

  const props: Record<string, string | number | boolean> = {}

  for (let i = propStartIndex; i < segments.length; i++) {
    const seg = segments[i]
    if (!seg) continue

    const eqIdx = seg.indexOf('=')
    if (eqIdx === -1) {
      props[seg.trim()] = true
      continue
    }

    const key = seg.slice(0, eqIdx).trim()
    const valRaw = seg.slice(eqIdx + 1).trim()
    props[key] = parseValue(valRaw)
  }

  if (props.size == null && size != null) {
    props.size = size
  }

  return <IconComponent {...props} />
}

function LobeIcon(props: {
  iconName: string | undefined | null
  size?: number
}) {
  const size = props.size ?? 20
  const [icon, setIcon] = useState<ReactNode>(() =>
    renderFallback(props.iconName, size)
  )

  useEffect(() => {
    if (!props.iconName || typeof props.iconName !== 'string') {
      setIcon(renderFallback(props.iconName, size))
      return
    }

    const trimmedName = props.iconName.trim()
    if (!trimmedName) {
      setIcon(renderFallback(props.iconName, size))
      return
    }

    let cancelled = false
    setIcon(renderFallback(trimmedName, size))

    const baseKey = trimmedName.split('.')[0]
    void loadLobeIcon(baseKey).then((BaseIcon) => {
      if (cancelled) return
      if (!BaseIcon) {
        setIcon(renderFallback(trimmedName, size))
        return
      }
      setIcon(renderLobeIcon(BaseIcon, trimmedName, size))
    })

    return () => {
      cancelled = true
    }
  }, [props.iconName, size])

  return icon
}

export function getLobeIcon(
  iconName: string | undefined | null,
  size: number = 20
): ReactNode {
  return <LobeIcon iconName={iconName} size={size} />
}
