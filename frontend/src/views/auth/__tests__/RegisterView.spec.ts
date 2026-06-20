import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RegisterView from '@/views/auth/RegisterView.vue'
import type { PublicSettings } from '@/types'

const {
  getPublicSettingsMock,
  showErrorMock,
  showWarningMock,
  registerMock
} = vi.hoisted(() => ({
  getPublicSettingsMock: vi.fn(),
  showErrorMock: vi.fn(),
  showWarningMock: vi.fn(),
  registerMock: vi.fn()
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn()
  }),
  useRoute: () => ({
    query: {}
  })
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      t: (key: string) => key
    }
  }),
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) => {
      if (key === 'auth.signUpToStart') {
        return `注册以开始使用 ${params?.siteName ?? 'Sub2API'}`
      }
      return key
    },
    locale: { value: 'zh-CN' }
  })
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    register: (...args: any[]) => registerMock(...args)
  }),
  useAppStore: () => ({
    showError: (...args: any[]) => showErrorMock(...args),
    showWarning: (...args: any[]) => showWarningMock(...args)
  })
}))

vi.mock('@/api/auth', async () => {
  const actual = await vi.importActual<typeof import('@/api/auth')>('@/api/auth')
  return {
    ...actual,
    getPublicSettings: (...args: any[]) => getPublicSettingsMock(...args),
    validatePromoCode: vi.fn(),
    validateInvitationCode: vi.fn()
  }
})

const basePublicSettings: PublicSettings = {
  registration_enabled: true,
  email_verify_enabled: false,
  force_email_on_third_party_signup: false,
  registration_email_suffix_whitelist: [],
  promo_code_enabled: true,
  password_reset_enabled: true,
  invitation_code_enabled: false,
  turnstile_enabled: false,
  turnstile_site_key: '',
  site_name: 'Sub2API',
  site_logo: '',
  site_subtitle: '',
  api_base_url: '',
  contact_info: '',
  doc_url: '',
  home_content: '',
  hide_ccs_import_button: false,
  payment_enabled: false,
  risk_control_enabled: false,
  table_default_page_size: 20,
  table_page_size_options: [10, 20, 50, 100],
  custom_menu_items: [],
  custom_endpoints: [],
  linuxdo_oauth_enabled: false,
  dingtalk_oauth_enabled: false,
  wechat_oauth_enabled: false,
  wechat_oauth_open_enabled: false,
  wechat_oauth_mp_enabled: false,
  wechat_oauth_mobile_enabled: false,
  oidc_oauth_enabled: false,
  oidc_oauth_provider_name: 'OIDC',
  github_oauth_enabled: false,
  google_oauth_enabled: false,
  unifed_oauth_enabled: false,
  unifed_hide_email_register_ui: false,
  backend_mode_enabled: false,
  version: 'test',
  balance_low_notify_enabled: false,
  account_quota_notify_enabled: false,
  balance_low_notify_threshold: 0,
  channel_monitor_enabled: false,
  channel_monitor_default_interval_seconds: 60,
  available_channels_enabled: false,
  service_quota_enabled: false,
  affiliate_enabled: false
}

function mountRegisterView(overrides: Partial<PublicSettings> = {}) {
  getPublicSettingsMock.mockResolvedValue({
    ...basePublicSettings,
    ...overrides
  })

  const wrapper = mount(RegisterView, {
    global: {
      stubs: {
        AuthLayout: { template: '<div><slot /><slot name="footer" /></div>' },
        Icon: true,
        TurnstileWidget: true,
        LoginAgreementPrompt: true,
        EmailOAuthButtons: { template: '<div data-testid="email-oauth-buttons" />' },
        LinuxDoOAuthSection: { template: '<div data-testid="linuxdo-oauth-section" />' },
        WechatOAuthSection: { template: '<div data-testid="wechat-oauth-section" />' },
        OidcOAuthSection: { template: '<div data-testid="oidc-oauth-section" />' },
        UniFedOAuthSection: { template: '<button data-testid="unifed-oauth-section">Universe Federation</button>' },
        RouterLink: { template: '<a><slot /></a>' }
      }
    }
  })

  return wrapper
}

describe('RegisterView', () => {
  beforeEach(() => {
    getPublicSettingsMock.mockReset()
    showErrorMock.mockReset()
    showWarningMock.mockReset()
    registerMock.mockReset()
    localStorage.clear()
  })

  it('hides the email registration form when UniFed requests a third-party-only register page', async () => {
    const wrapper = mountRegisterView({
      unifed_oauth_enabled: true,
      unifed_hide_email_register_ui: true
    })

    await flushPromises()

    expect(wrapper.find('[data-testid="email-register-form"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="unifed-oauth-section"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('auth.createAccount')
    expect(wrapper.text()).not.toContain('auth.oauthOrContinue')
  })

  it('keeps the email registration form when the hide flag is enabled but UniFed login is unavailable', async () => {
    const wrapper = mountRegisterView({
      unifed_oauth_enabled: false,
      unifed_hide_email_register_ui: true
    })

    await flushPromises()

    expect(wrapper.find('[data-testid="email-register-form"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="unifed-oauth-section"]').exists()).toBe(false)
  })
})
