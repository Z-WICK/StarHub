import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useStarsStore } from './starsStore'

const listStarsMock = vi.fn()

vi.mock('../services/apiClient', () => ({
  apiClient: {
    listStars: (...args: unknown[]) => listStarsMock(...args),
    listTags: vi.fn(),
    createTag: vi.fn(),
    assignTag: vi.fn(),
    unassignTag: vi.fn(),
    saveNote: vi.fn(),
    sync: vi.fn(),
    syncStatus: vi.fn(),
  },
}))

describe('stars store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    listStarsMock.mockReset()
  })

  it('fetchStars should update items and total', async () => {
    listStarsMock.mockResolvedValue({
      data: {
        items: [
          {
            repositoryId: 1,
            githubRepoId: 1,
            ownerLogin: 'wick',
            name: 'repo',
            fullName: 'wick/repo',
            private: false,
            htmlUrl: 'https://github.com/wick/repo',
            description: 'demo',
            language: 'Go',
            stargazersCount: 10,
            starredAt: new Date().toISOString(),
            lastSeenAt: new Date().toISOString(),
            note: '',
            tags: [],
          },
        ],
      },
      meta: { total: 1 },
    })

    const store = useStarsStore()
    await store.fetchStars()

    expect(store.items).toHaveLength(1)
    expect(store.total).toBe(1)
    expect(store.error).toBe('')
  })

  it('fetchStars should set error on failure', async () => {
    listStarsMock.mockRejectedValue(new Error('network failed'))

    const store = useStarsStore()
    await store.fetchStars()

    expect(store.error).toContain('network failed')
  })
})
