import { getConfiguredTableDefaultPageSize, normalizeTablePageSize } from '@/utils/tablePreferences'

const STORAGE_KEY = 'table-page-size'
const DEFAULT_STORAGE_KEY = 'table-page-size-default'

export function getPersistedPageSize(fallback = getConfiguredTableDefaultPageSize()): number {
  const configuredDefault = getConfiguredTableDefaultPageSize()
  if (typeof window !== 'undefined') {
    try {
      const storedDefault = window.localStorage.getItem(DEFAULT_STORAGE_KEY)
      if (storedDefault !== String(configuredDefault)) {
        window.localStorage.removeItem(STORAGE_KEY)
        window.localStorage.setItem(DEFAULT_STORAGE_KEY, String(configuredDefault))
        return normalizeTablePageSize(configuredDefault || fallback)
      }

      const stored = window.localStorage.getItem(STORAGE_KEY)
      if (stored !== null) {
        const parsed = Number(stored)
        if (Number.isFinite(parsed)) {
          return normalizeTablePageSize(parsed)
        }
      }
    } catch (error) {
      console.warn('Failed to read persisted page size:', error)
    }
  }
  return normalizeTablePageSize(configuredDefault || fallback)
}

export function setPersistedPageSize(size: number): void {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(STORAGE_KEY, String(size))
    window.localStorage.setItem(DEFAULT_STORAGE_KEY, String(getConfiguredTableDefaultPageSize()))
  } catch (error) {
    console.warn('Failed to persist page size:', error)
  }
}
