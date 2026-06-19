<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Default Home Page -->
  <div
    v-else
    class="relative flex min-h-screen flex-col overflow-hidden bg-gradient-to-br from-gray-50 via-primary-50/30 to-gray-100 dark:from-dark-950 dark:via-dark-900 dark:to-dark-950"
  >
    <!-- Background Decorations -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div
        class="absolute -right-40 -top-40 h-96 w-96 rounded-full bg-primary-400/20 blur-3xl"
      ></div>
      <div
        class="absolute -bottom-40 -left-40 h-96 w-96 rounded-full bg-primary-500/15 blur-3xl"
      ></div>
      <div
        class="absolute inset-0 bg-[linear-gradient(rgba(20,184,166,0.03)_1px,transparent_1px),linear-gradient(90deg,rgba(20,184,166,0.03)_1px,transparent_1px)] bg-[size:64px_64px]"
      ></div>
    </div>

    <!-- Header -->
    <header class="relative z-20 px-6 py-4">
      <nav class="mx-auto flex max-w-6xl items-center justify-between">
        <!-- Logo -->
        <div class="flex items-center">
          <div class="h-10 w-10 overflow-hidden rounded-xl shadow-md">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
        </div>

        <!-- Nav Actions -->
        <div class="flex items-center gap-3">
          <!-- Language Switcher -->
          <LocaleSwitcher />

          <!-- Doc Link -->
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>

          <!-- Theme Toggle -->
          <button
            @click="toggleTheme"
            class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>

          <!-- Login / Dashboard Button -->
          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="inline-flex items-center gap-1.5 rounded-full bg-gray-900 py-1 pl-1 pr-2.5 transition-colors hover:bg-gray-800 dark:bg-gray-800 dark:hover:bg-gray-700"
          >
            <span
              class="flex h-5 w-5 items-center justify-center rounded-full bg-gradient-to-br from-primary-400 to-primary-600 text-[10px] font-semibold text-white"
            >
              {{ userInitial }}
            </span>
            <span class="text-xs font-medium text-white">{{ t('home.dashboard') }}</span>
            <svg
              class="h-3 w-3 text-gray-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M4.5 19.5l15-15m0 0H8.25m11.25 0v11.25"
              />
            </svg>
          </router-link>
          <router-link
            v-else
            to="/login"
            class="inline-flex items-center rounded-full bg-gray-900 px-3 py-1 text-xs font-medium text-white transition-colors hover:bg-gray-800 dark:bg-gray-800 dark:hover:bg-gray-700"
          >
            {{ t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <!-- Main Content -->
    <main class="relative z-10 flex-1 px-6">
      <div class="mx-auto max-w-6xl">
        <!-- Hero Section - Left/Right Layout -->
        <div class="mb-12 flex flex-col items-center justify-between gap-8 pt-16 lg:flex-row lg:gap-16 lg:pt-20">
          <!-- Left: Text Content -->
          <div class="flex-1 text-center lg:text-left">
            <h1
              class="mb-4 text-4xl font-bold text-gray-900 dark:text-white md:text-5xl lg:text-6xl"
            >
              {{ siteName }}
            </h1>
            <p class="mb-8 text-lg text-gray-600 dark:text-dark-300 md:text-xl">
              {{ siteSubtitle }}
            </p>

            <!-- CTA Button -->
            <div>
              <router-link
                :to="isAuthenticated ? dashboardPath : '/login'"
                class="btn btn-primary px-8 py-3 text-base shadow-lg shadow-primary-500/30"
              >
                {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
                <Icon name="arrowRight" size="md" class="ml-2" :stroke-width="2" />
              </router-link>
            </div>
          </div>

          <!-- Right: Pinned Announcement Card -->
          <div class="flex flex-1 justify-center lg:justify-end">
            <div v-if="pinnedAnnouncement" class="w-full max-w-lg">
              <div
                class="rounded-2xl border border-primary-200/60 bg-white/70 p-6 shadow-lg shadow-primary-500/10 backdrop-blur-sm dark:border-primary-800/40 dark:bg-dark-800/70"
              >
                <div class="mb-3 flex items-center gap-2">
                  <span class="inline-flex items-center rounded-md bg-primary-100 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-400">
                    📌 {{ t('home.pinned') }}
                  </span>
                  <span class="text-xs text-gray-400 dark:text-dark-500">{{ formatDate(pinnedAnnouncement.created_at) }}</span>
                </div>
                <h3 class="mb-3 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ pinnedAnnouncement.title }}
                </h3>
                <div
                  class="prose prose-sm prose-gray max-w-none line-clamp-6 dark:prose-invert"
                  v-html="renderMarkdown(pinnedAnnouncement.content)"
                ></div>
              </div>
            </div>
            <div v-else class="w-full max-w-md">
              <div
                class="rounded-2xl border border-dashed border-gray-300/60 bg-white/40 p-6 text-center dark:border-dark-700/40 dark:bg-dark-800/40"
              >
                <div class="text-4xl mb-3">📢</div>
                <p class="text-sm text-gray-400 dark:text-dark-500">{{ t('home.noPinnedAnnouncement') }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Announcements Section -->
        <div class="mb-12">
          <div class="flex flex-col gap-8 lg:flex-row">
            <!-- Latest Announcement (2/3) -->
            <div class="flex-[2] flex flex-col">
              <h2 class="mb-4 text-xl font-bold text-gray-900 dark:text-white">
                {{ t('home.latestAnnouncement') }}
              </h2>

              <div v-if="selectedAnnouncement" class="flex-1">
                <div
                  class="h-full rounded-2xl border border-gray-200/60 bg-white/70 p-6 shadow-sm backdrop-blur-sm transition-all dark:border-dark-700/50 dark:bg-dark-800/70"
                >
                  <div class="mb-4 flex items-center justify-between">
                    <span class="text-xs text-gray-400 dark:text-dark-500">{{ formatDate(selectedAnnouncement.created_at) }}</span>
                    <span
                      v-if="selectedAnnouncement.is_pinned"
                      class="inline-flex items-center rounded-md bg-primary-100 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-400"
                    >📌 {{ t('home.pinned') }}</span>
                  </div>
                  <h3 class="mb-4 text-xl font-semibold text-gray-900 dark:text-white">
                    {{ selectedAnnouncement.title }}
                  </h3>
                  <div
                    class="prose prose-gray max-w-none dark:prose-invert prose-headings:text-gray-900 dark:prose-headings:text-white prose-a:text-primary-600 dark:prose-a:text-primary-400 prose-code:bg-gray-100 dark:prose-code:bg-dark-800 prose-code:rounded prose-code:px-1 prose-code:text-sm prose-pre:bg-gray-900 dark:prose-pre:bg-dark-950"
                    v-html="renderMarkdown(selectedAnnouncement.content)"
                  ></div>
                </div>
              </div>

              <div v-else class="flex h-48 items-center justify-center">
                <div class="text-center">
                  <div class="text-4xl mb-3">📭</div>
                  <p class="text-sm text-gray-400 dark:text-dark-500">{{ t('home.noAnnouncements') }}</p>
                </div>
              </div>
            </div>

            <!-- Timeline (1/3) -->
            <div class="flex-1 flex flex-col">
              <h2 class="mb-4 text-xl font-bold text-gray-900 dark:text-white">
                {{ t('home.announcementTimeline') }}
              </h2>

              <div v-if="timelineAnnouncements.length > 0" class="flex flex-1 flex-col justify-between gap-3">
                <div
                  v-for="ann in timelineAnnouncements"
                  :key="ann.id"
                  @click="selectAnnouncement(ann)"
                  :class="[
                    'flex-1 cursor-pointer rounded-xl border p-4 transition-all hover:shadow-md',
                    selectedAnnouncement?.id === ann.id
                      ? 'border-primary-300 bg-primary-50/50 shadow-md dark:border-primary-700 dark:bg-primary-900/10'
                      : 'border-gray-200/60 bg-white/70 hover:bg-gray-50 dark:border-dark-700/50 dark:bg-dark-800/70 dark:hover:bg-dark-800'
                  ]"
                >
                  <div class="mb-1 flex items-center gap-2">
                    <span
                      v-if="ann.is_pinned"
                      class="text-xs text-primary-600 dark:text-primary-400"
                      :title="t('home.pinned')"
                    >📌</span>
                    <span class="text-xs text-gray-400 dark:text-dark-500">{{ formatDate(ann.created_at) }}</span>
                  </div>
                  <h4 class="text-sm font-medium text-gray-800 dark:text-dark-200 line-clamp-2">
                    {{ ann.title }}
                  </h4>
                  <div
                    class="mt-2 text-xs text-gray-500 dark:text-dark-400 line-clamp-2"
                    v-html="renderMarkdown(stripMarkdown(ann.content))"
                  ></div>
                </div>
              </div>

              <div v-else class="flex h-32 items-center justify-center">
                <p class="text-sm text-gray-400 dark:text-dark-500">{{ t('home.noAnnouncements') }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="relative z-10 border-t border-gray-200/50 px-6 py-8 dark:border-dark-800/50">
      <div
        class="mx-auto flex max-w-6xl flex-col items-center justify-center gap-4 text-center sm:flex-row sm:text-left"
      >
        <p class="text-sm text-gray-500 dark:text-dark-400">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
        <div class="flex items-center gap-4">
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-sm text-gray-500 transition-colors hover:text-gray-700 dark:text-dark-400 dark:hover:text-white"
          >
            {{ t('home.docs') }}
          </a>
          <a
            :href="githubUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-sm text-gray-500 transition-colors hover:text-gray-700 dark:text-dark-400 dark:hover:text-white"
          >
            GitHub
          </a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore, useAnnouncementStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import type { Announcement } from '@/types'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()
const announcementStore = useAnnouncementStore()

// Site settings
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

// Theme
const isDark = ref(document.documentElement.classList.contains('dark'))

// GitHub URL
const githubUrl = 'https://github.com/donkilove/sub2api'

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

// Announcement state
const pinnedAnnouncement = computed(() => announcementStore.homepagePinned)
// Timeline (right column): show only 2nd and 3rd newest, max 2 cards
const timelineAnnouncements = computed<Announcement[]>(() => {
  return announcementStore.homepageRecent.slice(1, 3)
})
const selectedAnnouncement = ref<Announcement | null>(null)

function selectAnnouncement(ann: Announcement) {
  selectedAnnouncement.value = ann
}

// Markdown rendering
function renderMarkdown(content: string): string {
  if (!content) return ''
  const raw = marked.parse(content, { breaks: true, gfm: true }) as string
  return DOMPurify.sanitize(raw)
}

function stripMarkdown(content: string): string {
  // Simple strip: remove markdown syntax for timeline preview
  return content
    .replace(/^#{1,6}\s+/gm, '')
    .replace(/\*\*(.+?)\*\*/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/`{1,3}[^`]*`{1,3}/g, '')
    .replace(/^\s*[-*+]\s+/gm, '')
    .replace(/^\s*\d+\.\s+/gm, '')
    .substring(0, 200)
}

function formatDate(dateStr: string | undefined): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleDateString('zh-CN', { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

// Toggle theme
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

// Initialize theme
function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(async () => {
  initTheme()

  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }

  // Fetch homepage announcements
  await announcementStore.fetchHomepageAnnouncements()

  // Auto-select the most recent non-pinned announcement (latest first)
  if (announcementStore.homepageRecent.length > 0) {
    selectedAnnouncement.value = announcementStore.homepageRecent[0]
  } else if (announcementStore.homepagePinned) {
    selectedAnnouncement.value = announcementStore.homepagePinned
  }
})
</script>

<style scoped>
/* Minimal styles - most styling handled by Tailwind */
</style>