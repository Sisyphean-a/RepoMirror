import { startTransition, useCallback, useEffect, useMemo, useState } from "react";
import {
  commitTarget,
  generateCommitMessage,
  loadState,
  pushTarget,
  refreshState,
  saveConfig,
  selectRepository,
  setAICommitAPIKey,
  setDirection,
  swapRepositories,
  syncRepositories,
} from "./api";
import type { DashboardState, Direction, RepositorySlot } from "./types";

interface ViewModel {
  busy: boolean;
  busyMessage: string;
  error: string;
  lastUpdatedAt: number;
  notice: string;
  state: DashboardState | null;
  selectRepo: (slot: RepositorySlot) => Promise<void>;
  swap: () => Promise<void>;
  changeDirection: (direction: Direction) => Promise<void>;
  save: () => Promise<void>;
  refresh: () => Promise<void>;
  sync: () => Promise<void>;
  commit: (message: string) => Promise<boolean>;
  generateCommit: () => Promise<string>;
  saveAICommitAPIKey: (apiKey: string) => Promise<boolean>;
  push: () => Promise<void>;
}

interface ActionContext {
  setBusy: (value: boolean) => void;
  setBusyMessage: (value: string) => void;
  setError: (value: string) => void;
  setLastUpdatedAt: (value: number) => void;
  setNotice: (value: string) => void;
  setState: (value: DashboardState | null | ((prev: DashboardState | null) => DashboardState | null)) => void;
}

export function useRepoMirror(): ViewModel {
  const [state, setState] = useState<DashboardState | null>(null);
  const [busy, setBusy] = useState(false);
  const [busyMessage, setBusyMessage] = useState("");
  const [error, setError] = useState("");
  const [lastUpdatedAt, setLastUpdatedAt] = useState(0);
  const [notice, setNotice] = useState("");

  useEffect(() => {
    void executeStateAction(
      setters(setState, setBusy, setBusyMessage, setError, setLastUpdatedAt, setNotice),
      loadState,
      { pendingNotice: "正在扫描仓库状态..." },
    );
  }, []);

  const actionContext = useMemo(
    () => setters(setState, setBusy, setBusyMessage, setError, setLastUpdatedAt, setNotice),
    [setState, setBusy, setBusyMessage, setError, setLastUpdatedAt, setNotice],
  );
  const action = useMemo(() => actionFactory(actionContext), [actionContext]);
  const commit = useCallback((message: string) => commitAction(actionContext, message), [actionContext]);
  const generateCommit = useCallback(() => generateCommitAction(actionContext), [actionContext]);
  const saveAICommitAPIKey = useCallback((apiKey: string) => saveAICommitAPIKeyAction(actionContext, apiKey), [actionContext]);

  return {
    busy,
    busyMessage,
    error,
    lastUpdatedAt,
    notice,
    state,
    commit,
    generateCommit,
    saveAICommitAPIKey,
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

function actionFactory(context: ActionContext) {
  return {
    selectRepo: async (slot: RepositorySlot) => {
      await executeStateAction(context, () => selectRepository(slot), { pendingNotice: `正在选择仓库 ${slot}...` });
    },
    swap: async () => {
      await executeStateAction(context, swapRepositories, { pendingNotice: "正在交换仓库...", successNotice: "已交换仓库路径" });
    },
    changeDirection: async (direction: Direction) => {
      await executeStateAction(context, () => setDirection(direction), { pendingNotice: "正在切换同步方向..." });
    },
    save: async () => {
      await executeStateAction(context, saveConfig, { pendingNotice: "正在保存配置...", successNotice: "配置已保存" });
    },
    refresh: async () => {
      await executeStateAction(context, refreshState, { pendingNotice: "正在刷新状态...", successNotice: "状态已刷新" });
    },
    sync: async () => {
      await executeStateAction(context, syncRepositories, { pendingNotice: "正在同步仓库...", successNotice: "同步完成" });
    },
    push: async () => {
      await executeStateAction(context, pushTarget, { pendingNotice: "正在推送目标仓库...", successNotice: "推送完成" });
    },
  };
}

async function commitAction(
  context: ActionContext,
  commitMessage: string,
) {
  return executeStateAction(context, () => commitTarget(commitMessage), {
    pendingNotice: "正在创建提交...",
    successNotice: "提交完成",
  });
}

async function generateCommitAction(context: ActionContext) {
  const message = await executeValueAction(context, generateCommitMessage, {
    pendingNotice: "正在生成提交信息...",
    successNotice: "提交信息已生成",
  });
  return message ?? "";
}

async function saveAICommitAPIKeyAction(context: ActionContext, apiKey: string) {
  return executeStateAction(context, () => setAICommitAPIKey(apiKey), {
    pendingNotice: "正在保存 DeepSeek Key...",
    successNotice: "DeepSeek Key 已保存",
  });
}

async function executeStateAction(
  context: ActionContext,
  action: () => Promise<DashboardState>,
  status: { pendingNotice: string; successNotice?: string },
) {
  const nextState = await executeTask(context, action, status);
  if (!nextState) {
    return false;
  }
  startTransition(() => context.setState(nextState));
  return true;
}

async function executeValueAction<T>(
  context: ActionContext,
  action: () => Promise<T>,
  status: { pendingNotice: string; successNotice?: string },
) {
  const value = await executeTask(context, action, status);
  return value;
}

async function executeTask<T>(
  context: ActionContext,
  action: () => Promise<T>,
  status: { pendingNotice: string; successNotice?: string },
) {
  context.setBusy(true);
  context.setBusyMessage(status.pendingNotice);
  context.setError("");
  context.setNotice("");
  try {
    const value = await action();
    context.setLastUpdatedAt(Date.now());
    if (status.successNotice) {
      context.setNotice(status.successNotice);
    }
    return value;
  } catch (error) {
    context.setError(error instanceof Error ? error.message : String(error));
    return null;
  } finally {
    context.setBusy(false);
    context.setBusyMessage("");
  }
}
