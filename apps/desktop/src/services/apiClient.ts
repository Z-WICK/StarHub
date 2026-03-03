import type {
  ApiEnvelope,
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
  UserProfile,
} from "../types/models";
import { getSessionToken } from "./sessionStorage";

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

type StarsResponse = {
  items: StarRecord[];
};

type LoginResponse = {
  token: string;
  profile: UserProfile;
};

type SessionResponse = {
  userId: number;
};

type SyncResponse = {
  processed: number;
  cursor: string;
};

type ApplyRulesResponse = {
  applied: number;
};

type RequestOptions = {
  dedupe?: boolean;
  cacheTtlMs?: number;
};

const inFlightRequests = new Map<string, Promise<ApiEnvelope<unknown>>>();
const responseCache = new Map<
  string,
  { expiresAt: number; value: ApiEnvelope<unknown> }
>();

const cleanupExpiredCache = () => {
  const now = Date.now();
  for (const [key, entry] of responseCache.entries()) {
    if (entry.expiresAt <= now) {
      responseCache.delete(key);
    }
  }
};

const request = async <T>(
  path: string,
  init: RequestInit = {},
  options: RequestOptions = {},
): Promise<ApiEnvelope<T>> => {
  cleanupExpiredCache();
  const sessionToken = await getSessionToken();
  const method = (init.method ?? "GET").toUpperCase();
  const body = typeof init.body === "string" ? init.body : "";
  const cacheKey = `${method}:${path}:${sessionToken}:${body}`;

  if (options.cacheTtlMs && options.cacheTtlMs > 0) {
    const cached = responseCache.get(cacheKey);
    if (cached && cached.expiresAt > Date.now()) {
      return cached.value as ApiEnvelope<T>;
    }
  }

  if (options.dedupe) {
    const existing = inFlightRequests.get(cacheKey);
    if (existing) {
      return (await existing) as ApiEnvelope<T>;
    }
  }

  const requestPromise = (async () => {
    const headers = new Headers(init.headers);
    headers.set("Content-Type", "application/json");
    if (sessionToken) {
      headers.set("Authorization", `Bearer ${sessionToken}`);
    }

    const response = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers,
    });

    const data = (await response.json()) as ApiEnvelope<T>;
    if (!response.ok || !data.success) {
      throw new Error(data.error ?? "请求失败");
    }

    if (options.cacheTtlMs && options.cacheTtlMs > 0) {
      responseCache.set(cacheKey, {
        value: data as ApiEnvelope<unknown>,
        expiresAt: Date.now() + options.cacheTtlMs,
      });
    }

    return data;
  })();

  if (options.dedupe) {
    inFlightRequests.set(
      cacheKey,
      requestPromise as Promise<ApiEnvelope<unknown>>,
    );
  }

  try {
    return await requestPromise;
  } finally {
    if (options.dedupe) {
      inFlightRequests.delete(cacheKey);
    }
  }
};

export const apiClient = {
  loginWithGitHubToken: (githubToken: string) =>
    request<LoginResponse>("/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ githubToken }),
    }),

  session: () =>
    request<SessionResponse>(
      "/v1/auth/session",
      {},
      { dedupe: true, cacheTtlMs: 1000 },
    ),

  listStars: (params: {
    page?: number;
    limit?: number;
    q?: string;
    query?: string;
    language?: string;
    tagId?: number;
    hasNote?: boolean;
    sortBy?: "starred_at" | "pushed_at" | "stargazers_count" | "updated_at";
    sortOrder?: "asc" | "desc";
  }) => {
    const search = new URLSearchParams();
    if (params.page) search.set("page", String(params.page));
    if (params.limit) search.set("limit", String(params.limit));
    if (params.q) search.set("q", params.q);
    if (params.query) search.set("query", params.query);
    if (params.language) search.set("language", params.language);
    if (params.tagId) search.set("tagId", String(params.tagId));
    if (typeof params.hasNote === "boolean")
      search.set("hasNote", String(params.hasNote));
    if (params.sortBy) search.set("sortBy", params.sortBy);
    if (params.sortOrder) search.set("sortOrder", params.sortOrder);
    return request<StarsResponse>(`/v1/stars?${search.toString()}`);
  },

  listTags: () =>
    request<Tag[]>("/v1/tags", {}, { dedupe: true, cacheTtlMs: 30000 }),

  createTag: (name: string, color: string) =>
    request<Tag>("/v1/tags", {
      method: "POST",
      body: JSON.stringify({ name, color }),
    }),

  assignTag: (repositoryId: number, tagId: number) =>
    request<{ status: string }>("/v1/tags/assign", {
      method: "POST",
      body: JSON.stringify({ repositoryId, tagId }),
    }),

  unassignTag: (repositoryId: number, tagId: number) =>
    request<{ status: string }>("/v1/tags/unassign", {
      method: "POST",
      body: JSON.stringify({ repositoryId, tagId }),
    }),

  batchAssignTag: (repositoryIds: number[], tagId: number) =>
    request<{ status: string }>("/v1/tags/batch/assign", {
      method: "POST",
      body: JSON.stringify({ repositoryIds, tagId }),
    }),

  batchUnassignTag: (repositoryIds: number[], tagId: number) =>
    request<{ status: string }>("/v1/tags/batch/unassign", {
      method: "POST",
      body: JSON.stringify({ repositoryIds, tagId }),
    }),

  saveNote: (repositoryId: number, content: string) =>
    request<{ status: string }>("/v1/notes", {
      method: "POST",
      body: JSON.stringify({ repositoryId, content }),
    }),

  readme: (repositoryId: number) =>
    request<ReadmePreview>("/v1/readme", {
      method: "POST",
      body: JSON.stringify({ repositoryId }),
    }),

  sync: () =>
    request<SyncResponse>("/v1/sync", {
      method: "POST",
    }),

  syncStatus: () =>
    request<SyncJob>("/v1/sync/status", {}, { dedupe: true, cacheTtlMs: 5000 }),

  getSyncSettings: () => request<SyncSettings>("/v1/sync/settings"),

  updateSyncSettings: (
    enabled: boolean,
    intervalHours: number,
    retryMax: number,
  ) =>
    request<SyncSettings>("/v1/sync/settings", {
      method: "POST",
      body: JSON.stringify({ enabled, intervalHours, retryMax }),
    }),

  listSmartRules: () => request<SmartRule[]>("/v1/sync/rules"),

  createSmartRule: (payload: {
    name: string;
    enabled: boolean;
    languageEquals: string;
    ownerContains: string;
    nameContains: string;
    descriptionContains: string;
    tagId: number;
  }) =>
    request<SmartRule>("/v1/sync/rules", {
      method: "POST",
      body: JSON.stringify(payload),
    }),

  deleteSmartRule: (ruleId: number) =>
    request<{ status: string }>("/v1/sync/rules/delete", {
      method: "POST",
      body: JSON.stringify({ ruleId }),
    }),

  applySmartRules: () =>
    request<ApplyRulesResponse>("/v1/sync/rules/apply", {
      method: "POST",
    }),

  governanceMetrics: () =>
    request<GovernanceMetrics>(
      "/v1/governance/metrics",
      {},
      { dedupe: true, cacheTtlMs: 15000 },
    ),

  exportData: () => request<ExportPayload>("/v1/io/export"),

  importData: (payload: ImportPayload) =>
    request<ImportResult>("/v1/io/import", {
      method: "POST",
      body: JSON.stringify(payload),
    }),
};
