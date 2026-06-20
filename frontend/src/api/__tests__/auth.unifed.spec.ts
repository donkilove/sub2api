import { beforeEach, describe, expect, it, vi } from 'vitest'

const post = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    post
  }
}))

describe('UniFed auth API', () => {
  beforeEach(() => {
    post.mockReset()
    post.mockResolvedValue({ data: {} })
    localStorage.clear()
  })

  it('posts pending UniFed account creation to the complete-registration endpoint', async () => {
    const { createPendingUniFedOAuthAccount } = await import('@/api/auth')

    await createPendingUniFedOAuthAccount(
      'invite-1',
      {
        adoptDisplayName: true,
        adoptAvatar: false
      },
      ' AFF-1 '
    )

    expect(post).toHaveBeenCalledWith('/auth/oauth/unifed/complete-registration', {
      invitation_code: 'invite-1',
      aff_code: 'AFF-1',
      adopt_display_name: true,
      adopt_avatar: false
    })
  })
})
