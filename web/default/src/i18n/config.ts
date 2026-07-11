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
import LanguageDetector from 'i18next-browser-languagedetector'
import { initReactI18next } from 'react-i18next'
import {
  applyStoredInterfaceLanguage,
  DEFAULT_INTERFACE_LANGUAGE,
} from './load-locale'

void i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {},
    lng: DEFAULT_INTERFACE_LANGUAGE,
    fallbackLng: [DEFAULT_INTERFACE_LANGUAGE],
    supportedLngs: ['en', 'zh', 'fr', 'ru', 'ja', 'vi'],
    partialBundledLanguages: true,
    load: 'languageOnly',
    nsSeparator: false,
    debug: import.meta.env.DEV,
    interpolation: {
      escapeValue: false,
    },
    detection: {
      order: ['localStorage', 'navigator'],
      caches: ['localStorage'],
    },
  })

export async function bootstrapI18n(): Promise<void> {
  await applyStoredInterfaceLanguage()
}

export default i18n
