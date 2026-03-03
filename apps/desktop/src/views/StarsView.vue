<template>
  <section class="panel">
    <header class="top">
      <h2>Stars 列表</h2>
      <div class="actions">
        <button class="ui-btn" :disabled="starsStore.syncLoading" @click="syncNow">
          {{ starsStore.syncLoading ? '同步中...' : '同步' }}
        </button>
      </div>
    </header>

    <div class="filters">
      <input class="ui-input filter-input" v-model="query" placeholder="搜索 full_name / description / note" @keyup.enter="applyFilters" />
      <input class="ui-input filter-input" v-model="language" placeholder="按语言过滤，如 TypeScript" @keyup.enter="applyFilters" />
      <select class="ui-select" v-model="selectedTagId" @change="applyFilters">
        <option :value="0">全部标签</option>
        <option v-for="tag in starsStore.tags" :key="tag.id" :value="tag.id">{{ tag.name }}</option>
      </select>
      <select class="ui-select" v-model="hasNote" @change="applyFilters">
        <option value="all">备注：全部</option>
        <option value="true">仅有备注</option>
        <option value="false">仅无备注</option>
      </select>
      <select class="ui-select" v-model="sortBy" @change="applyFilters">
        <option value="starred_at">按 Star 时间</option>
        <option value="pushed_at">按 Push 时间</option>
        <option value="stargazers_count">按 Star 数</option>
        <option value="updated_at">按更新时间</option>
      </select>
      <select class="ui-select" v-model="sortOrder" @change="applyFilters">
        <option value="desc">降序</option>
        <option value="asc">升序</option>
      </select>
      <button class="ui-btn" @click="applyFilters">查询</button>
    </div>

    <div v-if="starsStore.items.length" class="batch-panel ui-glass-soft">
      <label class="batch-select-all">
        <input type="checkbox" :checked="isAllSelectedOnPage" @change="toggleSelectAllOnPage" />
        当前页全选
      </label>
      <span>已选 {{ selectedRepositoryIds.length }} 项</span>
      <select class="ui-select" v-model="batchTagId">
        <option :value="0">选择标签</option>
        <option v-for="tag in starsStore.tags" :key="`batch-${tag.id}`" :value="tag.id">{{ tag.name }}</option>
      </select>
      <button class="ui-btn" :disabled="!canRunBatchAction" @click="batchAssign">批量绑定标签</button>
      <button class="ui-btn ui-btn--ghost" :disabled="!canRunBatchAction" @click="batchUnassign">批量解绑标签</button>
    </div>

    <p v-if="starsStore.error" class="error ui-error">{{ starsStore.error }}</p>
    <p v-if="starsStore.loading" class="hint ui-hint">加载中...</p>
    <p v-else-if="!starsStore.items.length" class="hint ui-hint">暂无数据，先去设置页触发一次同步。</p>

    <section v-if="starsStore.readmePreview || starsStore.readmeLoading || starsStore.readmeError" class="readme-panel ui-glass-soft">
      <header class="readme-header">
        <h3>README 预览</h3>
        <button class="ui-btn ui-btn--ghost ui-btn--mini" @click="closeReadme">关闭</button>
      </header>
      <p v-if="starsStore.readmeLoading" class="hint ui-hint">README 加载中...</p>
      <p v-else-if="starsStore.readmeError" class="error ui-error">{{ starsStore.readmeError }}</p>
      <article v-else-if="starsStore.readmePreview" class="readme-content">
        <h4>{{ starsStore.readmePreview.repository.fullName }}</h4>
        <pre>{{ starsStore.readmePreview.content }}</pre>
      </article>
    </section>

    <div class="list" v-else>
      <div ref="listViewportRef" class="list-viewport" tabindex="0" role="region" aria-label="Stars 列表滚动区域" @scroll="onListScroll">
        <div class="list-inner" :style="{ height: `${virtualTotalHeight}px` }">
          <div class="list-window">
            <div class="spacer" :style="{ height: `${topSpacerHeight}px` }"></div>
            <div
              class="row"
              v-for="star in visibleItems"
              :key="star.repositoryId"
              :ref="(el) => setRowRef(star.repositoryId, el as HTMLElement | null)"
            >
              <label class="row-select">
                <input
                  type="checkbox"
                  :value="star.repositoryId"
                  :checked="selectedRepositoryIds.includes(star.repositoryId)"
                  @change="toggleSelect(star.repositoryId)"
                />
              </label>
              <StarListItem
                :star="star"
                @save-note="saveNote"
                @preview-readme="previewReadme"
                @open-tag-picker="openTagPicker"
              />
            </div>
            <div class="spacer" :style="{ height: `${bottomSpacerHeight}px` }"></div>
          </div>
        </div>
      </div>
    </div>

    <TagPicker
      :open="tagPickerOpen"
      :tags="starsStore.tags"
      :repository-id="activeTagRepositoryId"
      @close="closeTagPicker"
      @create="createTag"
      @assign="onAssignTag"
      @unassign="onUnassignTag"
    />

    <footer class="pager">
      <button class="ui-btn" :disabled="starsStore.filters.page <= 1" @click="prevPage">上一页</button>
      <span>第 {{ starsStore.filters.page }} 页 · 共 {{ starsStore.total }} 条</span>
      <button class="ui-btn" :disabled="isLastPage" @click="nextPage">下一页</button>
    </footer>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import StarListItem from '../components/StarListItem.vue'
import TagPicker from '../components/TagPicker.vue'
import { useStarsStore } from '../stores/starsStore'

const starsStore = useStarsStore()

const query = ref('')
const language = ref('')
const selectedTagId = ref(0)
const hasNote = ref('all')
const sortBy = ref<'starred_at' | 'pushed_at' | 'stargazers_count' | 'updated_at'>('starred_at')
const sortOrder = ref<'asc' | 'desc'>('desc')
const selectedRepositoryIds = ref<number[]>([])
const batchTagId = ref(0)
const tagPickerOpen = ref(false)
const activeTagRepositoryId = ref<number | null>(null)

const defaultRowHeight = 280
const rowGap = 0
const overscan = 4
const listViewportRef = ref<HTMLElement | null>(null)
const scrollTop = ref(0)
const rowHeights = ref<Record<number, number>>({})
const rowElements = new Map<number, HTMLElement>()
let resizeObserver: ResizeObserver | null = null

const isLastPage = computed(() => {
  return starsStore.filters.page * starsStore.filters.limit >= starsStore.total
})

const isAllSelectedOnPage = computed(() => {
  if (!starsStore.items.length) {
    return false
  }
  return starsStore.items.every((item) => selectedRepositoryIds.value.includes(item.repositoryId))
})

const canRunBatchAction = computed(() => {
  return selectedRepositoryIds.value.length > 0 && batchTagId.value > 0
})

const measuredAverageHeight = computed(() => {
  const values = Object.values(rowHeights.value)
  if (!values.length) {
    return defaultRowHeight
  }
  const total = values.reduce((sum, value) => sum + value, 0)
  return Math.max(defaultRowHeight, Math.ceil(total / values.length))
})

const itemSizeAt = (index: number) => {
  const item = starsStore.items[index]
  if (!item) {
    return measuredAverageHeight.value + rowGap
  }
  const measured = rowHeights.value[item.repositoryId]
  return (measured ?? measuredAverageHeight.value) + rowGap
}

const prefixHeights = computed(() => {
  const prefix = new Array<number>(starsStore.items.length + 1)
  prefix[0] = 0
  for (let i = 0; i < starsStore.items.length; i += 1) {
    prefix[i + 1] = prefix[i] + itemSizeAt(i)
  }
  return prefix
})

const totalHeightRaw = computed(() => {
  const prefix = prefixHeights.value
  return prefix[prefix.length - 1] ?? 0
})

const virtualTotalHeight = computed(() => {
  return Math.max(0, totalHeightRaw.value - rowGap)
})

const findStartIndexByScrollTop = (value: number) => {
  const prefix = prefixHeights.value
  let low = 0
  let high = Math.max(0, starsStore.items.length - 1)
  while (low <= high) {
    const mid = Math.floor((low + high) / 2)
    const start = prefix[mid]
    const end = prefix[mid + 1]
    if (value < start) {
      high = mid - 1
      continue
    }
    if (value >= end) {
      low = mid + 1
      continue
    }
    return mid
  }
  return Math.min(starsStore.items.length, low)
}

const startIndex = computed(() => {
  const found = findStartIndexByScrollTop(scrollTop.value)
  return Math.max(0, found - overscan)
})

const visibleCount = computed(() => {
  const viewportHeight = listViewportRef.value?.clientHeight ?? 0
  return Math.max(1, Math.ceil(viewportHeight / (measuredAverageHeight.value + rowGap)))
})

const endIndex = computed(() => {
  return Math.min(starsStore.items.length, startIndex.value + visibleCount.value + overscan * 2)
})

const visibleItems = computed(() => {
  return starsStore.items.slice(startIndex.value, endIndex.value)
})

const topSpacerHeight = computed(() => {
  return prefixHeights.value[startIndex.value] ?? 0
})

const bottomSpacerHeight = computed(() => {
  const prefix = prefixHeights.value
  const bottomStart = prefix[endIndex.value] ?? 0
  return Math.max(0, totalHeightRaw.value - bottomStart)
})

const getIndexByRepositoryId = (repositoryId: number) => {
  return starsStore.items.findIndex((item) => item.repositoryId === repositoryId)
}

const updateRowHeight = (repositoryId: number, element: HTMLElement) => {
  const measured = Math.ceil(element.getBoundingClientRect().height)
  if (measured <= 0) {
    return
  }
  const previous = rowHeights.value[repositoryId]
  if (previous === measured) {
    return
  }
  const changedIndex = getIndexByRepositoryId(repositoryId)
  const currentStart = startIndex.value
  if (changedIndex >= 0 && changedIndex < currentStart && listViewportRef.value) {
    const delta = measured - (previous ?? measuredAverageHeight.value)
    listViewportRef.value.scrollTop += delta
    scrollTop.value = listViewportRef.value.scrollTop
  }
  rowHeights.value = {
    ...rowHeights.value,
    [repositoryId]: measured,
  }
}

const setRowRef = (repositoryId: number, element: HTMLElement | null) => {
  const existing = rowElements.get(repositoryId)
  if (existing && resizeObserver) {
    resizeObserver.unobserve(existing)
  }
  if (!element) {
    rowElements.delete(repositoryId)
    return
  }
  rowElements.set(repositoryId, element)
  updateRowHeight(repositoryId, element)
  if (resizeObserver) {
    resizeObserver.observe(element)
  }
}

onMounted(async () => {
  await Promise.all([starsStore.fetchTags(), starsStore.fetchStars(), starsStore.fetchSyncStatus()])
  await nextTick()
  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const target = entry.target as HTMLElement
        for (const [repositoryId, element] of rowElements.entries()) {
          if (element === target) {
            updateRowHeight(repositoryId, element)
            break
          }
        }
      }
    })
    for (const [repositoryId, element] of rowElements.entries()) {
      updateRowHeight(repositoryId, element)
      resizeObserver.observe(element)
    }
  }
})

watch(
  () => starsStore.items.map((item) => item.repositoryId),
  async (nextIDs) => {
    const nextIDSet = new Set(nextIDs)
    const nextHeights: Record<number, number> = {}
    for (const id of nextIDs) {
      const measured = rowHeights.value[id]
      if (typeof measured === 'number') {
        nextHeights[id] = measured
      }
    }
    rowHeights.value = nextHeights
    for (const [id, element] of rowElements.entries()) {
      if (!nextIDSet.has(id)) {
        if (resizeObserver) {
          resizeObserver.unobserve(element)
        }
        rowElements.delete(id)
      }
    }
    await nextTick()
  },
)

onBeforeUnmount(() => {
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
  rowElements.clear()
})

const onListScroll = () => {
  scrollTop.value = listViewportRef.value?.scrollTop ?? 0
}

const applyFilters = async () => {
  starsStore.setFilters({
    query: query.value.trim(),
    language: language.value.trim(),
    tagId: selectedTagId.value > 0 ? selectedTagId.value : null,
    hasNote: hasNote.value === 'all' ? null : hasNote.value === 'true',
    sortBy: sortBy.value,
    sortOrder: sortOrder.value,
    page: 1,
  })
  selectedRepositoryIds.value = []
  scrollTop.value = 0
  if (listViewportRef.value) {
    listViewportRef.value.scrollTop = 0
  }
  await starsStore.fetchStars()
}

const prevPage = async () => {
  starsStore.setFilters({ page: starsStore.filters.page - 1 })
  selectedRepositoryIds.value = []
  scrollTop.value = 0
  if (listViewportRef.value) {
    listViewportRef.value.scrollTop = 0
  }
  await starsStore.fetchStars()
}

const nextPage = async () => {
  starsStore.setFilters({ page: starsStore.filters.page + 1 })
  selectedRepositoryIds.value = []
  scrollTop.value = 0
  if (listViewportRef.value) {
    listViewportRef.value.scrollTop = 0
  }
  await starsStore.fetchStars()
}

const saveNote = async (payload: { repositoryId: number; content: string }) => {
  await starsStore.saveNote(payload.repositoryId, payload.content)
}

const createTag = async (payload: { name: string; color: string }) => {
  await starsStore.createTag(payload.name, payload.color)
}

const onAssignTag = async (payload: { repositoryId: number; tagId: number }) => {
  await starsStore.assignTag(payload.repositoryId, payload.tagId)
}

const onUnassignTag = async (payload: { repositoryId: number; tagId: number }) => {
  await starsStore.unassignTag(payload.repositoryId, payload.tagId)
}

const openTagPicker = (repositoryId: number) => {
  activeTagRepositoryId.value = repositoryId
  tagPickerOpen.value = true
}

const closeTagPicker = () => {
  tagPickerOpen.value = false
  activeTagRepositoryId.value = null
}

const toggleSelect = (repositoryId: number) => {
  if (selectedRepositoryIds.value.includes(repositoryId)) {
    selectedRepositoryIds.value = selectedRepositoryIds.value.filter((id) => id !== repositoryId)
    return
  }
  selectedRepositoryIds.value = [...selectedRepositoryIds.value, repositoryId]
}

const toggleSelectAllOnPage = () => {
  if (isAllSelectedOnPage.value) {
    const pageIds = new Set(starsStore.items.map((item) => item.repositoryId))
    selectedRepositoryIds.value = selectedRepositoryIds.value.filter((id) => !pageIds.has(id))
    return
  }
  const next = new Set(selectedRepositoryIds.value)
  for (const item of starsStore.items) {
    next.add(item.repositoryId)
  }
  selectedRepositoryIds.value = Array.from(next)
}

const batchAssign = async () => {
  if (!canRunBatchAction.value) {
    return
  }
  await starsStore.batchAssignTag(selectedRepositoryIds.value, batchTagId.value)
}

const batchUnassign = async () => {
  if (!canRunBatchAction.value) {
    return
  }
  await starsStore.batchUnassignTag(selectedRepositoryIds.value, batchTagId.value)
}

const syncNow = async () => {
  await starsStore.triggerSync()
}

const previewReadme = async (repositoryId: number) => {
  await starsStore.fetchReadme(repositoryId)
}

const closeReadme = () => {
  starsStore.clearReadmePreview()
}
</script>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 14px;
  min-height: calc(100dvh - 68px);
}

.top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.filters {
  display: grid;
  grid-template-columns: minmax(260px, 2fr) minmax(220px, 1.5fr) repeat(4, minmax(120px, 160px)) auto;
  gap: 8px;
}

.filter-input {
  min-width: 0;
}

.batch-panel {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  padding: 10px;
}

.batch-select-all {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding-right: 4px;
}

.batch-panel span {
  color: var(--text-secondary);
}

.batch-select-all input,
.row-select input {
  accent-color: var(--accent-strong);
}

.list {
  display: grid;
}

.list-viewport {
  height: 65vh;
  overflow: auto;
  border: 1px solid var(--border-glass);
  border-radius: 14px;
  padding: 8px;
  background: var(--surface-overlay);
  backdrop-filter: blur(14px) saturate(148%);
  -webkit-backdrop-filter: blur(14px) saturate(148%);
  scrollbar-gutter: stable;
}

.list-viewport:focus-visible {
  box-shadow: var(--focus-ring);
}

.list-inner {
  position: relative;
}

.list-window {
  position: absolute;
  left: 0;
  right: 0;
  top: 0;
  display: grid;
  gap: 0;
}

.row {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 8px;
}

.row-select {
  padding-top: 12px;
}

.pager {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-top: auto;
}

.pager :deep(.ui-btn) {
  min-width: 80px;
}

.readme-panel {
  padding: 12px;
  display: grid;
  gap: 10px;
}

.readme-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.readme-content h4 {
  margin: 0;
  color: var(--text-primary);
}

.readme-content pre {
  white-space: pre-wrap;
  word-break: break-word;
  margin: 0;
  color: var(--text-primary);
  background: color-mix(in srgb, var(--surface-overlay) 86%, transparent);
  border: 1px solid rgba(186, 199, 220, 0.32);
  border-radius: 10px;
  padding: 12px;
  max-height: 420px;
  overflow: auto;
  line-height: 1.52;
  font-size: 12.5px;
}

.readme-header h3,
.top h2,
.hint,
.error {
  margin: 0;
}

.readme-content h4 {
  margin-bottom: 8px;
}

.batch-panel,
.readme-panel {
  border-color: rgba(191, 219, 254, 0.3);
}

@media (max-width: 1400px) {
  .filters {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (max-width: 1200px) {
  .filters {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .list-viewport {
    height: 62vh;
  }
}

@media (max-width: 980px) {
  .filters {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .top {
    flex-wrap: wrap;
  }

  .batch-panel {
    gap: 8px;
  }
}

@media (max-width: 720px) {
  .filters {
    grid-template-columns: 1fr;
  }

  .list-viewport {
    height: 58vh;
  }

  .pager {
    justify-content: stretch;
  }

  .pager :deep(.ui-btn) {
    flex: 1;
  }
}
</style>
