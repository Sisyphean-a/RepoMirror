import { startTransition, useDeferredValue, useEffect, useState } from "react";
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
  busyMessage: string;
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
  const [busyMessage, setBusyMessage] = useState("");
  const [error, setError] = useState("");
  const [lastUpdatedAt, setLastUpdatedAt] = useState(0);
  const [notice, setNotice] = useState("");
  const [commitMessage, setCommitMessage] = useState("");
  const [searchTerm, setSearchTerm] = useState("");
  const deferredSearchTerm = useDeferredValue(searchTerm);

  useEffect(() => {
    void executeAction(
      setters(setState, setBusy, setBusyMessage, setError, setLastUpdatedAt, setNotice),
      loadState,
      { pendingNotice: "正在扫描仓库状态..." },
    );
  }, []);

  const visibleEntries = filterEntries(state?.differences ?? [], filter, deferredSearchTerm);
  const action = actionFactory({
    commitMessage,
    setBusy,
    setBusyMessage,
    setCommitMessage,
    setError,
    setLastUpdatedAt,
    setNotice,
    setState,
  });

  return {
    busy,
    busyMessage,
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
  setBusyMessage: (value: string) => void,
  setError: (value: string) => void,
  setLastUpdatedAt: (value: number) => void,
  setNotice: (value: string) => void,
) {
  return { setState, setBusy, setBusyMessage, setError, setLastUpdatedAt, setNotice };
}

function actionFactory(context: {
  commitMessage: string;
  setBusy: (value: boolean) => void;
  setBusyMessage: (value: string) => void;
  setCommitMessage: (value: string) => void;
  setError: (value: string) => void;
  setLastUpdatedAt: (value: number) => void;
  setNotice: (value: string) => void;
  setState: (value: DashboardState | null | ((prev: DashboardState | null) => DashboardState | null)) => void;
}) {
  return {
    selectRepo: (slot: RepositorySlot) =>
      executeAction(context, () => selectRepository(slot), { pendingNotice: `正在选择仓库 ${slot}...` }),
    swap: () => executeAction(context, swapRepositories, { pendingNotice: "正在交换仓库...", successNotice: "已交换仓库路径" }),
    changeDirection: (direction: Direction) =>
      executeAction(context, () => setDirection(direction), { pendingNotice: "正在切换同步方向..." }),
    save: () => executeAction(context, saveConfig, { pendingNotice: "正在保存配置...", successNotice: "配置已保存" }),
    refresh: () => executeAction(context, refreshState, { pendingNotice: "正在刷新状态...", successNotice: "状态已刷新" }),
    sync: () => executeAction(context, syncRepositories, { pendingNotice: "正在同步仓库...", successNotice: "同步完成" }),
    commit: () => commitAction(context),
    push: () => executeAction(context, pushTarget, { pendingNotice: "正在推送目标仓库...", successNotice: "推送完成" }),
  };
}

async function commitAction(context: {
  commitMessage: string;
  setBusy: (value: boolean) => void;
  setBusyMessage: (value: string) => void;
  setCommitMessage: (value: string) => void;
  setError: (value: string) => void;
  setLastUpdatedAt: (value: number) => void;
  setNotice: (value: string) => void;
  setState: (value: DashboardState | null | ((prev: DashboardState | null) => DashboardState | null)) => void;
}) {
  await executeAction(context, () => commitTarget(context.commitMessage), {
    pendingNotice: "正在创建提交...",
    successNotice: "提交完成",
  });
  context.setCommitMessage("");
}

async function executeAction(
  context: {
    setBusy: (value: boolean) => void;
    setBusyMessage: (value: string) => void;
    setError: (value: string) => void;
    setLastUpdatedAt: (value: number) => void;
    setNotice: (value: string) => void;
    setState: (value: DashboardState | null | ((prev: DashboardState | null) => DashboardState | null)) => void;
  },
  action: () => Promise<DashboardState>,
  status: { pendingNotice: string; successNotice?: string },
) {
  context.setBusy(true);
  context.setBusyMessage(status.pendingNotice);
  context.setError("");
  context.setNotice("");
  try {
    const nextState = await action();
    startTransition(() => context.setState(nextState));
    context.setLastUpdatedAt(Date.now());
    if (status.successNotice) {
      context.setNotice(status.successNotice);
    }
  } catch (error) {
    context.setError(error instanceof Error ? error.message : String(error));
  } finally {
    context.setBusy(false);
    context.setBusyMessage("");
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
    return [entry.path, entry.kind].some((value) => value.toLowerCase().includes(normalizedQuery));
  });
}
