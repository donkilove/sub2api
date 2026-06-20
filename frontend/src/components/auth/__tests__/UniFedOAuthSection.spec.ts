import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import UniFedOAuthSection from '@/components/auth/UniFedOAuthSection.vue'

const routeState = vi.hoisted(() => ({
  query: {} as Record<string, unknown>
}))

const locationState = vi.hoisted(() => ({
  current: { href: 'http://localhost/login' } as { href: string }
}))

const resolveAffiliateReferralCode = vi.hoisted(() => vi.fn(() => 'aff-1'))
const storeOAuthAffiliateCode = vi.hoisted(() => vi.fn())

vi.mock('vue-router', () => ({
  useRoute: () => routeState
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/utils/oauthAffiliate', () => ({
  resolveAffiliateReferralCode,
  storeOAuthAffiliateCode
}))

describe('UniFedOAuthSection', () => {
  beforeEach(() => {
    routeState.query = { redirect: '/settings/profile' }
    locationState.current = { href: 'http://localhost/login' }
    resolveAffiliateReferralCode.mockClear()
    storeOAuthAffiliateCode.mockClear()
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: locationState.current
    })
  })

  it('starts UniFed OAuth with the current redirect path', async () => {
    const wrapper = mount(UniFedOAuthSection, {
      props: { showDivider: false }
    })

    await wrapper.get('button').trigger('click')

    expect(locationState.current.href).toBe('/api/v1/auth/oauth/unifed/start?redirect=%2Fsettings%2Fprofile')
    expect(storeOAuthAffiliateCode).toHaveBeenCalledWith('aff-1')
  })
})
