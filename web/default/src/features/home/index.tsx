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
import { lazy, Suspense } from 'react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { Skeleton } from '@/components/ui/skeleton'
import { useAuthStore } from '@/stores/auth-store'

import { CustomHomeInline } from './components/custom-home-inline'
import { useHomePageContent } from './hooks'

const DefaultHomeContent = lazy(
  () => import('./components/default-home-content').then((m) => ({
    default: m.DefaultHomeContent,
  }))
)

const MarkdownHomeContent = lazy(() =>
  import('./components/markdown-home-content').then((m) => ({
    default: m.MarkdownHomeContent,
  }))
)

function HomeContentSkeleton() {
  return (
    <main className='mx-auto flex min-h-[60vh] max-w-6xl flex-col gap-4 px-4 pt-24 pb-16'>
      <Skeleton className='h-8 w-48' />
      <Skeleton className='h-12 w-full max-w-2xl' />
      <Skeleton className='h-24 w-full max-w-xl' />
      <Skeleton className='mt-4 h-40 w-full' />
    </main>
  )
}

export function Home() {
  const { t } = useTranslation()
  const { auth } = useAuthStore()
  const isAuthenticated = !!auth.user
  const { content, mode, isRefreshing } = useHomePageContent()

  if (mode === 'pending' || (mode === 'inline-html' && !content)) {
    return (
      <PublicLayout showMainContainer={false}>
        <HomeContentSkeleton />
      </PublicLayout>
    )
  }

  if (mode === 'inline-html') {
    return (
      <PublicLayout showMainContainer={false}>
        <CustomHomeInline html={content} />
        {isRefreshing ? (
          <span className='sr-only' aria-live='polite'>
            {t('Loading...')}
          </span>
        ) : null}
      </PublicLayout>
    )
  }

  if (mode === 'markdown' && content) {
    return (
      <PublicLayout>
        <Suspense fallback={<HomeContentSkeleton />}>
          <MarkdownHomeContent content={content} />
        </Suspense>
      </PublicLayout>
    )
  }

  return (
    <PublicLayout showMainContainer={false}>
      <Suspense fallback={<HomeContentSkeleton />}>
        <DefaultHomeContent isAuthenticated={isAuthenticated} />
      </Suspense>
    </PublicLayout>
  )
}
