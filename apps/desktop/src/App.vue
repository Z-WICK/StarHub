<template>
  <div class="layout">
    <aside class="sidebar ui-glass-panel">
      <RouterLink class="brand-link" to="/stars">Star Manager</RouterLink>

      <nav class="sidebar-nav" aria-label="主导航">
        <RouterLink v-if="showLoginEntry" to="/">登录</RouterLink>
      </nav>

      <nav class="sidebar-footer" aria-label="底部导航">
        <RouterLink to="/settings">设置</RouterLink>
      </nav>
    </aside>

    <main class="content ui-glass-panel">
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";

const route = useRoute();
const showLoginEntry = computed(() => route.name === "login");
</script>

<style scoped>
.layout {
  display: grid;
  grid-template-columns: 248px 1fr;
  height: 100dvh;
  gap: 14px;
  padding: 14px;
}

.sidebar {
  padding: 20px;
  min-width: 0;
  display: grid;
  grid-template-rows: auto 1fr auto;
  gap: 10px;
  height: 100%;
  overflow: auto;
}

.brand-link {
  margin: 0 0 16px;
  font-size: 20px;
  font-weight: 700;
  letter-spacing: 0.01em;
  color: var(--text-primary);
}

.sidebar-nav,
.sidebar-footer {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.sidebar-nav {
  align-content: start;
}

.sidebar-footer {
  margin-top: auto;
  padding-top: 12px;
  border-top: 1px solid color-mix(in srgb, var(--border-strong) 40%, transparent);
}

.sidebar-nav a,
.sidebar-footer a {
  padding: 10px 12px;
  border-radius: 12px;
  color: var(--text-secondary);
  border: 1px solid transparent;
  transition: background var(--transition-base), border-color var(--transition-base), color var(--transition-base), transform var(--motion-micro);
}

.sidebar-nav a:hover,
.sidebar-footer a:hover {
  background: var(--glass-hover);
  border-color: rgba(226, 232, 240, 0.28);
  color: var(--text-primary);
  transform: translateY(-1px);
}

.sidebar-nav a:focus-visible,
.sidebar-footer a:focus-visible {
  border-color: var(--border-strong);
  background: color-mix(in srgb, var(--glass-hover) 78%, transparent);
  color: var(--text-primary);
}

.sidebar-nav a.router-link-exact-active,
.sidebar-footer a.router-link-exact-active {
  background: color-mix(in srgb, var(--glass-hover) 82%, rgba(59, 130, 246, 0.18));
  color: var(--text-primary);
  border-color: color-mix(in srgb, var(--border-strong) 68%, rgba(59, 130, 246, 0.22));
}

.content {
  padding: 20px;
  background: var(--surface-base);
  min-width: 0;
  height: 100%;
  overflow: auto;
  display: flex;
}

@media (max-width: 1400px) {
  .layout {
    grid-template-columns: 228px 1fr;
    gap: 12px;
    padding: 12px;
  }

  .sidebar,
  .content {
    padding: 18px;
  }
}

@media (max-width: 1200px) {
  .layout {
    grid-template-columns: 208px 1fr;
  }

  .sidebar,
  .content {
    padding: 16px;
  }
}

@media (max-width: 960px) {
  .layout {
    grid-template-columns: 1fr;
    gap: 10px;
    padding: 10px;
  }

  .sidebar {
    position: sticky;
    top: 10px;
    z-index: 10;
    padding: 14px;
    grid-template-rows: auto auto auto;
    gap: 8px;
    max-height: none;
    overflow: visible;
  }

  .sidebar-nav,
  .sidebar-footer {
    flex-direction: row;
    flex-wrap: nowrap;
    overflow-x: auto;
    scrollbar-width: thin;
    gap: 8px;
  }

  .sidebar-footer {
    margin-left: 0;
    max-width: none;
    justify-content: flex-start;
    padding-top: 0;
    border-top: 0;
  }

  .content {
    padding: 14px;
  }
}

@media (max-width: 768px) {
  .sidebar-nav a,
  .sidebar-footer a {
    padding: 8px 10px;
  }
}
</style>
