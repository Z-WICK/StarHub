<template>
  <section class="panel ui-glass-panel">
    <h2>GitHub 登录</h2>
    <p class="hint">粘贴你的 GitHub Personal Access Token（仅用于首次绑定）。</p>

    <form class="form" @submit.prevent="onSubmit">
      <label>
        <span>GitHub Token</span>
        <input class="ui-input"
          v-model="githubToken"
          type="password"
          placeholder="ghp_xxx"
          autocomplete="off"
          required
        />
      </label>
      <button class="ui-btn" :disabled="authStore.loading || !githubToken.trim()" type="submit">
        {{ authStore.loading ? '登录中...' : '登录' }}
      </button>
      <p v-if="authStore.error" class="error ui-error">{{ authStore.error }}</p>
    </form>

    <div v-if="authStore.isAuthenticated" class="profile">
      <h3>当前会话</h3>
      <p><strong>用户：</strong>{{ authStore.user?.displayName }}</p>
      <p><strong>GitHub：</strong>{{ authStore.user?.githubLogin }}</p>
      <button class="ui-btn" @click="goStars">进入 Stars 管理</button>
      <button class="ui-btn ui-btn--ghost" @click="logout">退出</button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/authStore'

const router = useRouter()
const authStore = useAuthStore()
const githubToken = ref('')

onMounted(async () => {
  await authStore.initializeSession()
})

const onSubmit = async () => {
  try {
    await authStore.loginWithGitHubToken(githubToken.value)
    githubToken.value = ''
    await router.push('/stars')
  } catch {
    // state handled in store
  }
}

const goStars = async () => {
  await router.push('/stars')
}

const logout = async () => {
  await authStore.logout()
}
</script>

<style scoped>
.panel {
  max-width: 680px;
  padding: 22px;
  background: var(--surface-elevated);
}

h2,
h3 {
  margin: 0;
}

.hint {
  margin: 0;
  color: var(--text-secondary);
  line-height: 1.6;
}

.form {
  display: grid;
  gap: 12px;
  margin-top: 16px;
}

.profile {
  display: grid;
  gap: 8px;
}

label {
  display: grid;
  gap: 6px;
  color: var(--text-secondary);
}

.form :deep(.ui-input) {
  padding: 10px 12px;
}

.form :deep(.ui-btn) {
  justify-self: start;
  padding: 10px 14px;
  min-width: 100px;
}

.error {
  margin: 0;
}

.profile {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid rgba(186, 199, 220, 0.34);
}

.profile :deep(.ui-btn + .ui-btn) {
  margin-left: 8px;
}

@media (max-width: 760px) {
  .panel {
    padding: 16px;
  }

  .profile :deep(.ui-btn + .ui-btn) {
    margin-left: 0;
  }

  .profile :deep(.ui-btn) {
    width: 100%;
  }
}
</style>
