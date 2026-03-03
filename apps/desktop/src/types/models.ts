export interface UserProfile {
  id: number;
  displayName: string;
  githubLogin: string;
  avatarUrl: string;
}

export interface Tag {
  id: number;
  name: string;
  color: string;
}

export interface StarRecord {
  repositoryId: number;
  githubRepoId: number;
  ownerLogin: string;
  name: string;
  fullName: string;
  private: boolean;
  htmlUrl: string;
  description: string;
  language: string;
  stargazersCount: number;
  starredAt: string;
  lastSeenAt: string;
  note: string;
  tags: Tag[];
}

export interface SyncJob {
  id: number;
  status: string;
  startedAt: string;
  finishedAt: string | null;
  cursor: string;
  errorMessage: string;
}

export interface ReadmePreview {
  repository: {
    repositoryId: number;
    ownerLogin: string;
    name: string;
    fullName: string;
  };
  content: string;
}

export interface SyncSettings {
  enabled: boolean;
  intervalHours: number;
  retryMax: number;
  updatedAt: string;
}

export interface SmartRule {
  id: number;
  name: string;
  enabled: boolean;
  languageEquals: string;
  ownerContains: string;
  nameContains: string;
  descriptionContains: string;
  tagId: number;
  createdAt: string;
}

export interface GovernanceMetrics {
  totalStars: number;
  untaggedStars: number;
  untaggedRatio: number;
  syncJobs7d: number;
  syncSuccess7d: number;
  syncSuccessRate7d: number;
  staleStars: number;
}

export interface ExportRule {
  name: string;
  enabled: boolean;
  languageEquals: string;
  ownerContains: string;
  nameContains: string;
  descriptionContains: string;
  tagName: string;
}

export interface ExportNote {
  githubRepoId: number;
  content: string;
}

export interface ExportTagBinding {
  githubRepoId: number;
  tagName: string;
}

export interface ExportPayload {
  version: string;
  exportedAt: string;
  syncSettings: SyncSettings;
  tags: Tag[];
  smartRules: ExportRule[];
  notes: ExportNote[];
  tagBindings: ExportTagBinding[];
}

export interface ImportPayload {
  syncSettings?: SyncSettings;
  tags: Tag[];
  smartRules: ExportRule[];
  notes: ExportNote[];
  tagBindings: ExportTagBinding[];
}

export interface ImportResult {
  tagsUpserted: number;
  rulesUpserted: number;
  notesUpserted: number;
  tagBindingsLinked: number;
}

export interface JsonWorkerSuccess<T> {
  success: true;
  data: T;
  error: null;
}

export interface JsonWorkerFailure {
  success: false;
  data: null;
  error: string;
}

export type JsonWorkerResult<T> = JsonWorkerSuccess<T> | JsonWorkerFailure;

export interface ApiEnvelope<T> {
  success: boolean;
  data: T | null;
  error: string | null;
  meta: {
    page: number;
    limit: number;
    total: number;
  } | null;
}
