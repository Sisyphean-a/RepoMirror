import { startTransition, useEffect, useState } from "react";
import {
  commitTarget,
  loadState,
  pushTarget,
  refreshState,
  saveConfig,
  selectRepository,
  setDirection,
  swapRepositories,
  syncRepositories,
} from "./api";
import type { DashboardState, DiffEntry, DiffFilter, Direction, RepositorySlot } from "./types";

interface ViewModel {
  busy: boolean;
  commitMessage: string;
  error: string;
  filter: DiffFilter;
  lastUpdatedAt: number;
  notice: string;
  searchTerm: string;
  state: DashboardState | null;
  visibleEntries: DiffEntry[];
  setFilter: (filter: DiffFilter) => void;
  setCommitMessage: (value: string) => void;
  setSearchTerm: (value: string) => void;
  selectRepo: (slot: RepositorySlot) => Promise<void>;
  swap: () => Promise<void>;
  changeDirection: (direction: Direction) => Promise<void>;
  save: () => Promise<void>;
  refresh: () => Promise<void>;
  sync: () => Promise<void>;
  commit: () => Promise<void>;
  push: () => Promise<void>;
}

export function useRepoMirror(): ViewModel {
  const [state, setState] = useState<DashboardState | null>(null);
  const [filter, setFilter] = useState<DiffFilter>("all");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [lastUpdatedAt, setLastUpdatedAt] = useState(0);
  const [notice, setNotice] = useState("");
  const [commitMessage, setCommitMessage] = useState("");
  const [searchTerm, setSearchTerm] = useState("");

  useEffect(() => {
    void executeAction(setters(setState, setBusy, setError, setLastUpdatedAt, setNotice), loadState);
  }, []);

  const visibleEntries = filterEntries(state?.differences ?? [], filter, searchTerm);
  const action = actionFactory({
    commitMessage,
    setBusy,
    setCommitMessage,
    setError,
    setLastUpdatedAt,
    setNotice,
    setState,
  });

  return {
    busy,
    commitMessage,
    error,
    filter,
    lastUpdatedAt,
    notice,
    searchTerm,
    state,
    visibleEntries,
    setFilter,
    setCommitMessage,
    setSearchTerm,
    ...action,
  };
}

function setters(
  setState: (value: DashboardState | null | ((prev: DashboardState | null) => DashboardState | null)) => void,
  setBusy: (value: boolean) => void,
  setError: (value: string) => void,
  setLastUpdatedAt: (value: number) => void,
  setNotice: (value: string) => void,
) {
  return { setState, setBusy, setError, setLastUpdatedAt, setNotice };
}

function actionFactory(context: {
  commitMessage: string;
  setBusy: (value: boolean) => void;
  setCommitMessage: (value: string) => void;
  setError: (value: string) => void;
  setLastUpdatedAt: (value: number) => void;
  setNotice: (value: string) => void;
  setState: (value: DashboardState | null | ((prev: DashboardState | null) => DashboardState | null)) => void;
}) {
  return {
    selectRepo: (slot: RepositorySlot) => executeAction(context, () => selectRepository(slot)),
    swap: () => executeAction(context, swapRepositories, "已交换仓库路径"),
    changeDirection: (direction: Direction) => executeAction(context, () => setDirection(direction)),
    save: () => executeAction(context, saveConfig, "配置已保存"),
    refresh: () => executeAction(context, refreshState, "状态已刷新"),
    sync: () => executeAction(context, syncRepositories, "同步完成"),
    commit: () => commitAction(context),
    push: () => executeAction(context, pushTarget, "推送完成"),
  };
}

async function commitAction(context: {
  commitMessage: string;
  setBusy: (value: boolean) => void;
  setCommitMessage: (value: string) => void;
  setError: (value: string) => void;
  setLastUpdatedAt: (value: number) => void;
  setNotice: (value: string) => void;
  setState: (value: DashboardState | null | ((prev: DashboardState | null) => DashboardState | null)) => void;
}) {
  await executeAction(context, () => commitTarget(context.commitMessage), "提交完成");
  context.setCommitMessage("");
}

async function executeAction(
  context: {
    setBusy: (value: boolean) => void;
    setError: (value: string) => void;
    setLastUpdatedAt: (value: number) => void;
    setNotice: (value: string) => void;
    setState: (value: DashboardState | null | ((prev: DashboardState | null) => DashboardState | null)) => void;
  },
  action: () => Promise<DashboardState>,
  successNotice = "",
) {
  context.setBusy(true);
  context.setError("");
  if (successNotice) {
    context.setNotice("");
  }
  try {
    const nextState = await action();
    startTransition(() => context.setState(nextState));
    context.setLastUpdatedAt(Date.now());
    if (successNotice) {
      context.setNotice(successNotice);
    }
  } catch (error) {
    context.setError(error instanceof Error ? error.message : String(error));
  } finally {
    context.setBusy(false);
  }
}

function filterEntries(entries: DiffEntry[], filter: DiffFilter, searchTerm: string) {
  const normalizedQuery = searchTerm.trim().toLowerCase();
  return entries.filter((entry) => {
    if (filter !== "all" && entry.kind !== filter) {
      return false;
    }
    if (!normalizedQuery) {
      return true;
    }
    return [entry.path, entry.kind, entry.rule].some((value) => value.toLowerCase().includes(normalizedQuery));
  });
}
