import { defineStore } from 'pinia'
import type { UserProfile } from '../types/models'
import { apiClient } from '../services/apiClient'
import { clearSessionToken, getSessionToken, saveSessionToken } from '../services/sessionStorage'

interface AuthState {
  token: string
  user: UserProfile | null
  loading: boolean
  error: string
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    token: '',
    user: null,
    loading: false,
    error: '',
  }),

  getters: {
    isAuthenticated: (state) => !!state.token,
  },

  actions: {
    async initializeSession() {
      this.loading = true
      this.error = ''
      try {
        const token = await getSessionToken()
        if (!token) {
          this.token = ''
          this.user = null
          return
        }
        this.token = token
        const result = await apiClient.session()
        this.user = {
          id: result.data?.userId ?? 0,
          displayName: 'Authenticated User',
          githubLogin: 'unknown',
          avatarUrl: '',
        }
      } catch (error) {
        this.token = ''
        this.user = null
        this.error = error instanceof Error ? error.message : '会话已失效'
        await clearSessionToken()
      } finally {
        this.loading = false
      }
    },

    async loginWithGitHubToken(githubToken: string) {
      this.loading = true
      this.error = ''
      try {
        const result = await apiClient.loginWithGitHubToken(githubToken)
        const token = result.data?.token ?? ''
        const profile = result.data?.profile ?? null
        if (!token || !profile) {
          throw new Error('登录响应无效')
        }
        await saveSessionToken(token)
        this.token = token
        this.user = profile
      } catch (error) {
        this.error = error instanceof Error ? error.message : '登录失败'
        throw error
      } finally {
        this.loading = false
      }
    },

    async logout() {
      this.token = ''
      this.user = null
      this.error = ''
      await clearSessionToken()
    },
  },
})
