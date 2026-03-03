import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  clearSessionToken,
  getSessionToken,
  saveSessionToken,
} from './sessionStorage'

describe('sessionStorage', () => {
  beforeEach(async () => {
    delete (window as Window & { sessionToken?: unknown }).sessionToken
    await clearSessionToken()
    vi.restoreAllMocks()
  })

  it('stores token in memory when electron bridge is unavailable', async () => {
    await saveSessionToken('memory-token')
    await expect(getSessionToken()).resolves.toBe('memory-token')

    await clearSessionToken()
    await expect(getSessionToken()).resolves.toBe('')
  })

  it('uses electron bridge when available', async () => {
    const bridge = {
      set: vi.fn(async () => undefined),
      get: vi.fn(async () => 'bridge-token'),
      clear: vi.fn(async () => undefined),
    }
    ;(window as Window & { sessionToken?: typeof bridge }).sessionToken = bridge

    await saveSessionToken('bridge-token')
    await expect(getSessionToken()).resolves.toBe('bridge-token')
    await clearSessionToken()

    expect(bridge.set).toHaveBeenCalledWith('bridge-token')
    expect(bridge.clear).toHaveBeenCalledTimes(1)
  })
})
