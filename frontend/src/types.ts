export type Direction = "A_TO_B" | "B_TO_A";
export type RepositorySlot = "A" | "B";
export type DiffKind = "added" | "modified" | "deleted" | "protected";
export type DiffFilter = "all" | DiffKind;

export interface AppConfig {
  projectA: string;
  projectB: string;
  direction: Direction;
  windowWidth: number;
  windowHeight: number;
}

export interface RepositorySummary {
  slot: RepositorySlot;
  path: string;
  name: string;
  isConfigured: boolean;
  isGitRepo: boolean;
  validationError: string;
  branch: string;
  isClean: boolean;
  modifiedCount: number;
  untrackedCount: number;
}

export interface TargetRepositoryStatus {
  path: string;
  name: string;
  branch: string;
  isGitRepo: boolean;
  error: string;
  isClean: boolean;
  modifiedCount: number;
  untrackedCount: number;
}

export interface DiffEntry {
  path: string;
  kind: DiffKind;
  rule: string;
  sizeBytes: number;
}

export interface DiffSummary {
  total: number;
  added: number;
  modified: number;
  deleted: number;
  protected: number;
}

export interface DashboardState {
  config: AppConfig;
  repositoryA: RepositorySummary;
  repositoryB: RepositorySummary;
  sourceSlot: RepositorySlot;
  targetSlot: RepositorySlot;
  differences: DiffEntry[];
  summary: DiffSummary;
  targetStatus: TargetRepositoryStatus;
  canSync: boolean;
}

export const diffKindLabel: Record<DiffKind, string> = {
  added: "新增",
  modified: "修改",
  deleted: "删除",
  protected: "受保护",
};

export const diffKindCode: Record<DiffKind, string> = {
  added: "A",
  modified: "M",
  deleted: "D",
  protected: "P",
};
