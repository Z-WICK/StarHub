<template>
  <div v-if="open" class="overlay ui-overlay" @click.self="$emit('close')">
    <div
      ref="panelRef"
      class="panel ui-glass-panel"
      role="dialog"
      aria-modal="true"
      aria-labelledby="tag-picker-title"
      tabindex="-1"
      @keydown="onPanelKeydown"
    >
      <header class="head">
        <h4 id="tag-picker-title">管理标签</h4>
        <button class="mini ghost ui-btn ui-btn--ghost ui-btn--mini" @click="$emit('close')">关闭</button>
      </header>

      <form class="inline" @submit.prevent="createTag">
        <input class="ui-input" v-model="name" placeholder="标签名" required />
        <input class="ui-input color-input" v-model="color" type="color" />
        <button class="ui-btn" type="submit">添加</button>
      </form>

      <ul>
        <li v-for="tag in tags" :key="tag.id">
          <span class="dot" :style="{ backgroundColor: tag.color }"></span>
          {{ tag.name }}
          <button class="mini ui-btn ui-btn--mini" :disabled="repositoryId == null" @click="onAssign(tag.id)">绑定</button>
          <button class="mini ghost ui-btn ui-btn--ghost ui-btn--mini" :disabled="repositoryId == null" @click="onUnassign(tag.id)">解绑</button>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import type { Tag } from '../types/models'

const props = defineProps<{
  open: boolean
  tags: Tag[]
  repositoryId: number | null
}>()

const emit = defineEmits<{
  (event: 'close'): void
  (event: 'create', payload: { name: string; color: string }): void
  (event: 'assign', payload: { repositoryId: number; tagId: number }): void
  (event: 'unassign', payload: { repositoryId: number; tagId: number }): void
}>()

const name = ref('')
const color = ref('#4f46e5')
const panelRef = ref<HTMLElement | null>(null)
const previousActiveElement = ref<HTMLElement | null>(null)

const focusFirstInteractive = async () => {
  await nextTick()
  if (!panelRef.value) {
    return
  }
  const focusables = panelRef.value.querySelectorAll<HTMLElement>(
    'button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
  )
  if (focusables.length) {
    focusables[0].focus()
    return
  }
  panelRef.value.focus()
}

const getFocusables = () => {
  if (!panelRef.value) {
    return [] as HTMLElement[]
  }
  return Array.from(
    panelRef.value.querySelectorAll<HTMLElement>(
      'button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
    ),
  )
}

const onPanelKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape') {
    event.preventDefault()
    emit('close')
    return
  }
  if (event.key !== 'Tab') {
    return
  }
  const focusables = getFocusables()
  if (!focusables.length) {
    event.preventDefault()
    panelRef.value?.focus()
    return
  }
  const first = focusables[0]
  const last = focusables[focusables.length - 1]
  const active = document.activeElement as HTMLElement | null
  if (event.shiftKey && active === first) {
    event.preventDefault()
    last.focus()
    return
  }
  if (!event.shiftKey && active === last) {
    event.preventDefault()
    first.focus()
  }
}

watch(
  () => props.open,
  async (isOpen) => {
    if (isOpen) {
      previousActiveElement.value = document.activeElement as HTMLElement | null
      await focusFirstInteractive()
      return
    }
    await nextTick()
    previousActiveElement.value?.focus()
  },
)

onBeforeUnmount(() => {
  previousActiveElement.value = null
})

const createTag = () => {
  emit('create', { name: name.value, color: color.value })
  name.value = ''
}

const onAssign = (tagId: number) => {
  if (props.repositoryId == null) {
    return
  }
  emit('assign', { repositoryId: props.repositoryId, tagId })
}

const onUnassign = (tagId: number) => {
  if (props.repositoryId == null) {
    return
  }
  emit('unassign', { repositoryId: props.repositoryId, tagId })
}
</script>

<style scoped>
.panel {
  padding: 14px;
  min-width: 360px;
  max-height: 70vh;
  overflow: auto;
  background: var(--surface-overlay);
}

.panel :deep(.ui-btn) {
  letter-spacing: 0.01em;
}

.head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  gap: 8px;
}

.head h4 {
  margin: 0;
}

.inline {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}

.color-input {
  max-width: 64px;
  padding: 4px;
}

ul {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 6px;
}

li {
  display: grid;
  grid-template-columns: 12px 1fr auto auto;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  border-bottom: 1px solid rgba(186, 199, 220, 0.18);
}

li :deep(.ui-btn) {
  min-width: 56px;
}

.dot {
  width: 12px;
  height: 12px;
  border-radius: 999px;
  box-shadow: 0 0 0 1px rgba(226, 232, 240, 0.4);
}

@media (max-width: 760px) {
  .panel {
    min-width: min(92vw, 420px);
    padding: 12px;
  }

  .inline,
  li {
    grid-template-columns: 1fr;
    display: grid;
  }

  .inline :deep(.ui-btn) {
    width: 100%;
  }
}
</style>
