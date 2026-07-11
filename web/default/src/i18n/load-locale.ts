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
import i18n from 'i18next'
import { normalizeInterfaceLanguage } from './languages'

type LocaleModule = {
  translation: Record<string, string>
}

const localeLoaders: Record<
  string,
  () => Promise<{ default: LocaleModule }>
> = {
  en: () => import('./locales/en.json'),
  fr: () => import('./locales/fr.json'),
  ja: () => import('./locales/ja.json'),
  ru: () => import('./locales/ru.json'),
  vi: () => import('./locales/vi.json'),
}

export const DEFAULT_INTERFACE_LANGUAGE = 'zh'

export async function ensureLanguageLoaded(language: string): Promise<void> {
  const code = normalizeInterfaceLanguage(language)
  if (code === DEFAULT_INTERFACE_LANGUAGE) return
  if (i18n.hasResourceBundle(code, 'translation')) return

  const loader = localeLoaders[code]
  if (!loader) return

  const mod = await loader()
  i18n.addResourceBundle(
    code,
    'translation',
    mod.default.translation,
    true,
    true
  )
}

export async function changeAppLanguage(language: string): Promise<void> {
  const code = normalizeInterfaceLanguage(language)
  await ensureLanguageLoaded(code)
  await i18n.changeLanguage(code)
}

export async function applyStoredInterfaceLanguage(): Promise<void> {
  const code = normalizeInterfaceLanguage(
    i18n.resolvedLanguage || i18n.language
  )
  if (code === DEFAULT_INTERFACE_LANGUAGE) return
  await changeAppLanguage(code)
}
