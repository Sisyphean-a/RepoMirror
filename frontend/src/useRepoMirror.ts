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
  notice: string;
  state: DashboardState | null;
  visibleEntries: DiffEntry[];
  setFilter: (filter: DiffFilter) => void;
  setCommitMessage: (value: string) => void;
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
  const [notice, setNotice] = useState("");
  const [commitMessage, setCommitMessage] = useState("");

  useEffect(() => {
    void executeAction(setters(setState, setBusy, setError, setNotice), loadState);
  }, []);

  const visibleEntries = filterEntries(state?.differences ?? [], filter);
  const action = actionFactory({ commitMessage, setBusy, setCommitMessage, setError, setNotice, setState });

  return { busy, commitMessage, error, filter, notice, state, visibleEntries, setFilter, setCommitMessage, ...action };
}

function setters(
  setState: (value: DashboardState | null | ((prev: DashboardState | null) => DashboardState | null)) => void,
  setBusy: (value: boolean) => void,
  setError: (value: string) => void,
  setNotice: (value: string) => void,
) {
  return { setState, setBusy, setError, setNotice };
}

function actionFactory(context: {
  commitMessage: string;
  setBusy: (value: boolean) => void;
  setCommitMessage: (value: string) => void;
  setError: (value: string) => void;
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
    if (successNotice) {
      context.setNotice(successNotice);
    }
  } catch (error) {
    context.setError(error instanceof Error ? error.message : String(error));
  } finally {
    context.setBusy(false);
  }
}

function filterEntries(entries: DiffEntry[], filter: DiffFilter) {
  if (filter === "all") {
    return entries;
  }
  return entries.filter((entry) => entry.kind === filter);
}
