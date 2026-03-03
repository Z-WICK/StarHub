<template>
  <article class="item">
    <header>
      <a href="#" @click.prevent="openRepository">
        {{ star.fullName }}
      </a>
      <small>{{ starredAtText }}</small>
    </header>

    <p class="desc">{{ star.description || '暂无描述' }}</p>

    <div class="meta">
      <span>Language: {{ star.language || 'Unknown' }}</span>
      <span>⭐ {{ star.stargazersCount }}</span>
      <button class="mini ui-btn ui-btn--ghost ui-btn--mini" @click="$emit('open-tag-picker', star.repositoryId)">管理标签</button>
    </div>

    <div class="tags">
      <span v-for="tag in star.tags" :key="tag.id" class="tag" :style="{ borderColor: tag.color }">
        {{ tag.name }}
      </span>
    </div>

    <textarea class="ui-textarea"
      :value="star.note"
      placeholder="写一点备注..."
      @change="onNoteChange($event)"
    />
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { StarRecord } from '../types/models'

const props = defineProps<{
  star: StarRecord
}>()

const emit = defineEmits<{
  (event: 'save-note', payload: { repositoryId: number; content: string }): void
  (event: 'preview-readme', repositoryId: number): void
  (event: 'open-tag-picker', repositoryId: number): void
}>()

const starredAtText = computed(() => new Date(props.star.starredAt).toLocaleString())

const onNoteChange = (event: Event) => {
  const target = event.target as HTMLTextAreaElement
  emit('save-note', {
    repositoryId: props.star.repositoryId,
    content: target.value,
  })
}

const openRepository = async () => {
  emit('preview-readme', props.star.repositoryId)
}
</script>

<style scoped>
.item {
  padding: 14px;
  min-width: 0;
  transition: transform var(--motion-micro), border-color var(--transition-base), background var(--transition-base), box-shadow var(--transition-base);
}

header {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  align-items: baseline;
}

header a {
  color: color-mix(in srgb, var(--accent-strong) 70%, var(--text-primary));
  font-weight: 600;
  transition: color var(--transition-base), opacity var(--transition-fast);
}

header a:hover {
  color: color-mix(in srgb, var(--accent-strong) 86%, var(--text-primary));
}

.desc {
  color: var(--text-secondary);
  line-height: 1.58;
}

.meta {
  display: flex;
  gap: 14px;
  color: var(--text-secondary);
  align-items: center;
  flex-wrap: wrap;
}

.meta :deep(.ui-btn) {
  margin-left: auto;
}

.tags {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 8px;
}

.tag {
  border: 1px solid var(--border-soft);
  border-radius: 999px;
  padding: 2px 8px;
  font-size: 12px;
  color: var(--text-secondary);
  background: color-mix(in srgb, var(--surface-overlay) 72%, transparent);
}

.item :deep(.ui-textarea) {
  margin-top: 10px;
  min-height: 72px;
}

.item :deep(.ui-btn--mini) {
  line-height: 1.2;
}

header small {
  color: var(--text-muted);
}

.item :deep(.ui-textarea) {
  font-size: 13px;
}

@media (max-width: 880px) {
  header {
    flex-direction: column;
    align-items: flex-start;
  }

  .meta :deep(.ui-btn) {
    margin-left: 0;
  }
}
</style>
