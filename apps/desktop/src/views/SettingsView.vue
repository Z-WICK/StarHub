<template>
  <section class="panel ui-glass-panel">
    <h2>同步设置</h2>
    <p class="hint">查看同步状态，配置定时同步与 Smart Rules。</p>

    <div class="actions">
      <button class="ui-btn" :disabled="starsStore.syncLoading" @click="syncNow">
        {{ starsStore.syncLoading ? '同步中...' : '立即同步' }}
      </button>
      <button class="ui-btn ui-btn--ghost" @click="refreshStatus">刷新状态</button>
    </div>

    <div class="status">
      <p><strong>状态：</strong>{{ starsStore.syncJob?.status || 'unknown' }}</p>
      <p><strong>开始：</strong>{{ formatTime(starsStore.syncJob?.startedAt) }}</p>
      <p><strong>结束：</strong>{{ formatTime(starsStore.syncJob?.finishedAt) }}</p>
      <p><strong>Cursor：</strong>{{ starsStore.syncJob?.cursor || '-' }}</p>
      <p><strong>Error：</strong>{{ starsStore.syncJob?.errorMessage || '-' }}</p>
    </div>

    <section class="box ui-glass-soft" v-if="starsStore.syncSettings">
      <h3>定时同步</h3>
      <label>
        <input type="checkbox" v-model="syncEnabled" />
        启用自动同步
      </label>
      <div class="grid">
        <label>
          频率
          <select class="ui-select" v-model.number="intervalHours">
            <option :value="6">每 6 小时</option>
            <option :value="12">每 12 小时</option>
            <option :value="24">每 24 小时</option>
          </select>
        </label>
        <label>
          重试次数
          <select class="ui-select" v-model.number="retryMax">
            <option :value="0">0</option>
            <option :value="1">1</option>
            <option :value="2">2</option>
            <option :value="3">3</option>
            <option :value="4">4</option>
            <option :value="5">5</option>
          </select>
        </label>
      </div>
      <button class="ui-btn" @click="saveSyncSettings">保存设置</button>
    </section>

    <section class="box ui-glass-soft">
      <h3>Smart Rules</h3>
      <div class="grid rule-form">
        <input class="ui-input" v-model="ruleForm.name" placeholder="规则名称" />
        <select class="ui-select" v-model.number="ruleForm.tagId">
          <option :value="0">选择标签</option>
          <option v-for="tag in starsStore.tags" :key="tag.id" :value="tag.id">{{ tag.name }}</option>
        </select>
        <input class="ui-input" v-model="ruleForm.languageEquals" placeholder="语言等于（可选）" />
        <input class="ui-input" v-model="ruleForm.ownerContains" placeholder="Owner 包含（可选）" />
        <input class="ui-input" v-model="ruleForm.nameContains" placeholder="仓库名包含（可选）" />
        <input class="ui-input" v-model="ruleForm.descriptionContains" placeholder="描述包含（可选）" />
      </div>
      <button class="ui-btn" @click="createRule">新增规则</button>
      <button class="ui-btn ui-btn--ghost" @click="applyRules">手动重跑规则</button>
      <p class="hint ui-hint" v-if="rulesAppliedTip">{{ rulesAppliedTip }}</p>

      <ul class="rule-list">
        <li v-for="rule in starsStore.smartRules" :key="rule.id">
          <span>{{ rule.name }} → #{{ rule.tagId }}</span>
          <small>
            {{ formatRule(rule) }}
          </small>
          <button class="ui-btn ui-btn--ghost ui-btn--mini" @click="deleteRule(rule.id)">删除</button>
        </li>
      </ul>
    </section>

    <section class="box ui-glass-soft" v-if="starsStore.governanceMetrics">
      <h3>治理看板</h3>
      <div class="grid metrics">
        <p><strong>Star 总数：</strong>{{ starsStore.governanceMetrics.totalStars }}</p>
        <p><strong>未分类：</strong>{{ starsStore.governanceMetrics.untaggedStars }}</p>
        <p><strong>未分类占比：</strong>{{ formatPercent(starsStore.governanceMetrics.untaggedRatio) }}</p>
        <p><strong>7天同步成功率：</strong>{{ formatPercent(starsStore.governanceMetrics.syncSuccessRate7d) }}</p>
        <p><strong>7天同步次数：</strong>{{ starsStore.governanceMetrics.syncJobs7d }}</p>
        <p><strong>长期未更新：</strong>{{ starsStore.governanceMetrics.staleStars }}</p>
      </div>
      <button class="ui-btn ui-btn--ghost" @click="refreshGovernance">刷新治理指标</button>
    </section>

    <section class="box ui-glass-soft">
      <h3>JSON 导入导出</h3>
      <div class="actions">
        <button class="ui-btn ui-btn--ghost" @click="exportJson">导出配置</button>
      </div>
      <textarea class="ui-textarea" v-model="importText" rows="10" placeholder="将导出的 JSON 粘贴到这里"></textarea>
      <button class="ui-btn" @click="importJson">导入配置</button>
      <p class="hint ui-hint" v-if="importTip">{{ importTip }}</p>
    </section>

    <p v-if="starsStore.error" class="error ui-error">{{ starsStore.error }}</p>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useStarsStore } from '../stores/starsStore'
import { jsonWorkerClient } from '../services/jsonWorkerClient'
import type { ImportPayload, SmartRule } from '../types/models'

const starsStore = useStarsStore()

const syncEnabled = ref(true)
const intervalHours = ref(12)
const retryMax = ref(2)
const rulesAppliedTip = ref('')
const importText = ref('')
const importTip = ref('')

const ruleForm = ref({
  name: '',
  tagId: 0,
  languageEquals: '',
  ownerContains: '',
  nameContains: '',
  descriptionContains: '',
})

onMounted(async () => {
  await Promise.all([
    starsStore.fetchTags(),
    starsStore.fetchSyncStatus(),
    starsStore.fetchSyncSettings(),
    starsStore.fetchSmartRules(),
    starsStore.fetchGovernanceMetrics(),
  ])
  if (starsStore.syncSettings) {
    syncEnabled.value = starsStore.syncSettings.enabled
    intervalHours.value = starsStore.syncSettings.intervalHours
    retryMax.value = starsStore.syncSettings.retryMax
  }
})

const syncNow = async () => {
  await starsStore.triggerSync()
}

const refreshStatus = async () => {
  await Promise.all([
    starsStore.fetchSyncStatus({ force: true }),
    starsStore.fetchSmartRules(),
  ])
}

const saveSyncSettings = async () => {
  await starsStore.updateSyncSettings(syncEnabled.value, intervalHours.value, retryMax.value)
}

const createRule = async () => {
  await starsStore.createSmartRule({
    name: ruleForm.value.name,
    enabled: true,
    languageEquals: ruleForm.value.languageEquals,
    ownerContains: ruleForm.value.ownerContains,
    nameContains: ruleForm.value.nameContains,
    descriptionContains: ruleForm.value.descriptionContains,
    tagId: ruleForm.value.tagId,
  })
  ruleForm.value = {
    name: '',
    tagId: 0,
    languageEquals: '',
    ownerContains: '',
    nameContains: '',
    descriptionContains: '',
  }
}

const deleteRule = async (ruleId: number) => {
  await starsStore.deleteSmartRule(ruleId)
}

const applyRules = async () => {
  const affected = await starsStore.applySmartRules()
  rulesAppliedTip.value = `本次新增匹配 ${affected} 条标签绑定`
  await starsStore.fetchStars()
}

const refreshGovernance = async () => {
  await starsStore.fetchGovernanceMetrics({ force: true })
}

const exportJson = async () => {
  const payload = await starsStore.exportData()
  const stringifyResult = await jsonWorkerClient.stringify(payload)
  if (!stringifyResult.success) {
    importTip.value = stringifyResult.error
    return
  }
  importText.value = stringifyResult.data
  importTip.value = '已生成导出 JSON，可复制保存。'
}

const importJson = async () => {
  const parseResult = await jsonWorkerClient.parse<ImportPayload>(importText.value)
  if (!parseResult.success) {
    importTip.value = parseResult.error
    return
  }
  const result = await starsStore.importData(parseResult.data)
  importTip.value = `导入完成：标签 ${result.tagsUpserted}、规则 ${result.rulesUpserted}、备注 ${result.notesUpserted}、绑定 ${result.tagBindingsLinked}`
  await Promise.all([
    starsStore.fetchTags({ force: true }),
    starsStore.fetchSmartRules(),
    starsStore.fetchStars(),
    starsStore.fetchGovernanceMetrics({ force: true }),
  ])
}

const formatPercent = (value: number) => {
  return `${(value * 100).toFixed(1)}%`
}

const formatTime = (value?: string | null) => {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

const formatRule = (rule: SmartRule) => {
  const parts = [
    rule.languageEquals ? `lang=${rule.languageEquals}` : '',
    rule.ownerContains ? `owner~${rule.ownerContains}` : '',
    rule.nameContains ? `name~${rule.nameContains}` : '',
    rule.descriptionContains ? `desc~${rule.descriptionContains}` : '',
  ].filter(Boolean)
  return parts.length ? parts.join(' · ') : '无条件'
}
</script>

<style scoped>
.panel {
  max-width: 920px;
  padding: 20px;
  background: var(--surface-elevated);
  display: grid;
  gap: 14px;
}

.actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.panel :deep(.ui-btn) {
  letter-spacing: 0.01em;
}

.panel :deep(.ui-input),
.panel :deep(.ui-select) {
  font-size: 13px;
}

.panel :deep(.ui-select) {
  padding-right: 28px;
}

.panel :deep(.ui-textarea) {
  min-height: 180px;
}

.hint,
.error {
  margin: 0;
}

.status {
  display: grid;
  gap: 6px;
  border: 1px solid rgba(186, 199, 220, 0.34);
  border-radius: var(--radius-control);
  padding: 10px;
  background: color-mix(in srgb, var(--surface-overlay) 82%, transparent);
}

.box {
  padding: 12px;
  display: grid;
  gap: 10px;
}

.grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.rule-form {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.rule-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 8px;
}

.rule-list li {
  display: grid;
  grid-template-columns: 1fr auto auto;
  gap: 8px;
  align-items: center;
}

.rule-list small {
  color: var(--text-secondary);
}

.hint,
.error,
.panel :deep(p) {
  margin: 0;
}

@media (max-width: 1200px) {
  .panel {
    padding: 18px;
  }

  .rule-form {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 980px) {
  .panel {
    padding: 16px;
  }

  .grid,
  .rule-form {
    grid-template-columns: 1fr;
  }

  .rule-list li {
    grid-template-columns: 1fr auto;
  }
}

@media (max-width: 760px) {
  .panel {
    padding: 14px;
  }

  .actions {
    flex-direction: column;
    align-items: stretch;
  }

  .rule-list li {
    grid-template-columns: 1fr;
  }
}
</style>
