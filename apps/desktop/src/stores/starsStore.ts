import { defineStore } from "pinia";
import { apiClient } from "../services/apiClient";
import type {
  ExportPayload,
  GovernanceMetrics,
  ImportPayload,
  ImportResult,
  ReadmePreview,
  SmartRule,
  StarRecord,
  SyncJob,
  SyncSettings,
  Tag,
} from "../types/models";

interface StarsFilters {
  query: string;
  language: string;
  tagId: number | null;
  hasNote: boolean | null;
  sortBy: "starred_at" | "pushed_at" | "stargazers_count" | "updated_at";
  sortOrder: "asc" | "desc";
  page: number;
  limit: number;
}

type CachedResource = {
  updatedAt: number;
};

interface StarsState {
  items: StarRecord[];
  total: number;
  tags: Tag[];
  syncJob: SyncJob | null;
  syncSettings: SyncSettings | null;
  smartRules: SmartRule[];
  governanceMetrics: GovernanceMetrics | null;
  readmePreview: ReadmePreview | null;
  filters: StarsFilters;
  loading: boolean;
  syncLoading: boolean;
  readmeLoading: boolean;
  readmeError: string;
  error: string;
  tagsCache: CachedResource | null;
  syncStatusCache: CachedResource | null;
  governanceCache: CachedResource | null;
}

export const useStarsStore = defineStore("stars", {
  state: (): StarsState => ({
    items: [],
    total: 0,
    tags: [],
    syncJob: null,
    syncSettings: null,
    smartRules: [],
    governanceMetrics: null,
    readmePreview: null,
    filters: {
      query: "",
      language: "",
      tagId: null,
      hasNote: null,
      sortBy: "starred_at",
      sortOrder: "desc",
      page: 1,
      limit: 20,
    },
    loading: false,
    syncLoading: false,
    readmeLoading: false,
    readmeError: "",
    error: "",
    tagsCache: null,
    syncStatusCache: null,
    governanceCache: null,
  }),

  actions: {
    async fetchStars() {
      this.loading = true;
      this.error = "";
      try {
        const result = await apiClient.listStars({
          page: this.filters.page,
          limit: this.filters.limit,
          q: this.filters.query,
          language: this.filters.language,
          tagId: this.filters.tagId ?? undefined,
          hasNote: this.filters.hasNote ?? undefined,
          sortBy: this.filters.sortBy,
          sortOrder: this.filters.sortOrder,
        });
        this.items = result.data?.items ?? [];
        this.total = result.meta?.total ?? 0;
      } catch (error) {
        this.error = error instanceof Error ? error.message : "加载 stars 失败";
      } finally {
        this.loading = false;
      }
    },

    async fetchTags(options?: { force?: boolean }) {
      const now = Date.now();
      if (
        !options?.force &&
        this.tagsCache &&
        now - this.tagsCache.updatedAt < 30000
      ) {
        return;
      }
      try {
        const result = await apiClient.listTags();
        this.tags = result.data ?? [];
        this.tagsCache = { updatedAt: now };
      } catch (error) {
        this.error = error instanceof Error ? error.message : "加载 tags 失败";
      }
    },

    async createTag(name: string, color: string) {
      await apiClient.createTag(name, color);
      await this.fetchTags();
    },

    async assignTag(repositoryId: number, tagId: number) {
      const selectedTag = this.tags.find((tag) => tag.id === tagId);
      if (!selectedTag) {
        throw new Error("标签不存在");
      }
      const index = this.items.findIndex(
        (item) => item.repositoryId === repositoryId,
      );
      if (index >= 0) {
        const target = this.items[index];
        if (!target.tags.some((tag) => tag.id === tagId)) {
          const nextItems = [...this.items];
          nextItems[index] = {
            ...target,
            tags: [...target.tags, selectedTag],
          };
          this.items = nextItems;
        }
      }
      try {
        await apiClient.assignTag(repositoryId, tagId);
      } catch (error) {
        await this.fetchStars();
        throw error;
      }
    },

    async unassignTag(repositoryId: number, tagId: number) {
      const index = this.items.findIndex(
        (item) => item.repositoryId === repositoryId,
      );
      if (index >= 0) {
        const target = this.items[index];
        const nextItems = [...this.items];
        nextItems[index] = {
          ...target,
          tags: target.tags.filter((tag) => tag.id !== tagId),
        };
        this.items = nextItems;
      }
      try {
        await apiClient.unassignTag(repositoryId, tagId);
      } catch (error) {
        await this.fetchStars();
        throw error;
      }
    },

    async batchAssignTag(repositoryIds: number[], tagId: number) {
      const selectedTag = this.tags.find((tag) => tag.id === tagId);
      if (!selectedTag) {
        throw new Error("标签不存在");
      }
      const repositoryIdSet = new Set(repositoryIds);
      this.items = this.items.map((item) => {
        if (
          !repositoryIdSet.has(item.repositoryId) ||
          item.tags.some((tag) => tag.id === tagId)
        ) {
          return item;
        }
        return {
          ...item,
          tags: [...item.tags, selectedTag],
        };
      });
      try {
        await apiClient.batchAssignTag(repositoryIds, tagId);
      } catch (error) {
        await this.fetchStars();
        throw error;
      }
    },

    async batchUnassignTag(repositoryIds: number[], tagId: number) {
      const repositoryIdSet = new Set(repositoryIds);
      this.items = this.items.map((item) => {
        if (!repositoryIdSet.has(item.repositoryId)) {
          return item;
        }
        return {
          ...item,
          tags: item.tags.filter((tag) => tag.id !== tagId),
        };
      });
      try {
        await apiClient.batchUnassignTag(repositoryIds, tagId);
      } catch (error) {
        await this.fetchStars();
        throw error;
      }
    },

    async saveNote(repositoryId: number, content: string) {
      await apiClient.saveNote(repositoryId, content);
      const index = this.items.findIndex(
        (item) => item.repositoryId === repositoryId,
      );
      if (index < 0) {
        return;
      }
      const nextItems = [...this.items];
      nextItems[index] = { ...nextItems[index], note: content };
      this.items = nextItems;
    },

    async fetchReadme(repositoryId: number) {
      this.readmeLoading = true;
      this.readmeError = "";
      try {
        const result = await apiClient.readme(repositoryId);
        this.readmePreview = result.data;
      } catch (error) {
        this.readmePreview = null;
        this.readmeError =
          error instanceof Error ? error.message : "加载 README 失败";
      } finally {
        this.readmeLoading = false;
      }
    },

    clearReadmePreview() {
      this.readmePreview = null;
      this.readmeError = "";
    },

    setFilters(partial: Partial<StarsFilters>) {
      this.filters = {
        ...this.filters,
        ...partial,
      };
    },

    async triggerSync() {
      this.syncLoading = true;
      this.error = "";
      try {
        await apiClient.sync();
        await Promise.all([this.fetchStars(), this.fetchSyncStatus()]);
      } catch (error) {
        this.error = error instanceof Error ? error.message : "同步失败";
      } finally {
        this.syncLoading = false;
      }
    },

    async fetchSyncStatus(options?: { force?: boolean }) {
      const now = Date.now();
      if (
        !options?.force &&
        this.syncStatusCache &&
        now - this.syncStatusCache.updatedAt < 5000
      ) {
        return;
      }
      try {
        const result = await apiClient.syncStatus();
        this.syncJob = result.data;
        this.syncStatusCache = { updatedAt: now };
      } catch {
        this.syncJob = null;
      }
    },

    async fetchSyncSettings() {
      try {
        const result = await apiClient.getSyncSettings();
        this.syncSettings = result.data;
      } catch (error) {
        this.error =
          error instanceof Error ? error.message : "加载同步设置失败";
      }
    },

    async updateSyncSettings(
      enabled: boolean,
      intervalHours: number,
      retryMax: number,
    ) {
      const result = await apiClient.updateSyncSettings(
        enabled,
        intervalHours,
        retryMax,
      );
      this.syncSettings = result.data;
    },

    async fetchSmartRules() {
      try {
        const result = await apiClient.listSmartRules();
        this.smartRules = result.data ?? [];
      } catch (error) {
        this.error = error instanceof Error ? error.message : "加载规则失败";
      }
    },

    async createSmartRule(payload: {
      name: string;
      enabled: boolean;
      languageEquals: string;
      ownerContains: string;
      nameContains: string;
      descriptionContains: string;
      tagId: number;
    }) {
      await apiClient.createSmartRule(payload);
      await this.fetchSmartRules();
    },

    async deleteSmartRule(ruleId: number) {
      await apiClient.deleteSmartRule(ruleId);
      this.smartRules = this.smartRules.filter((rule) => rule.id !== ruleId);
    },

    async applySmartRules() {
      const result = await apiClient.applySmartRules();
      return result.data?.applied ?? 0;
    },

    async fetchGovernanceMetrics(options?: { force?: boolean }) {
      const now = Date.now();
      if (
        !options?.force &&
        this.governanceCache &&
        now - this.governanceCache.updatedAt < 15000
      ) {
        return;
      }
      try {
        const result = await apiClient.governanceMetrics();
        this.governanceMetrics = result.data;
        this.governanceCache = { updatedAt: now };
      } catch (error) {
        this.error =
          error instanceof Error ? error.message : "加载治理指标失败";
      }
    },

    async exportData() {
      const result = await apiClient.exportData();
      return result.data as ExportPayload;
    },

    async importData(payload: ImportPayload) {
      const result = await apiClient.importData(payload);
      return result.data as ImportResult;
    },
  },
});
