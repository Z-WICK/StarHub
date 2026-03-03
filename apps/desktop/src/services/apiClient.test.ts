import { afterEach, describe, expect, it, vi } from 'vitest'
import { apiClient } from './apiClient'

vi.mock('./sessionStorage', () => ({
  getSessionToken: vi.fn(async () => 'session-token'),
}))

describe('apiClient', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('throws when api returns success=false', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        ok: false,
        json: async () => ({ success: false, error: 'bad request' }),
      })),
    )

    await expect(apiClient.listTags()).rejects.toThrow('bad request')
  })
})
