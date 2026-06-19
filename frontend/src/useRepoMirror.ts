import { startTransition, useCallback, useEffect, useMemo, useState } from "react";
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
  commit: (message: string) => Promise<void>;
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
    void executeAction(
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

  return {
    busy,
    busyMessage,
    error,
    lastUpdatedAt,
    notice,
    state,
    commit,
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
    selectRepo: (slot: RepositorySlot) =>
      executeAction(context, () => selectRepository(slot), { pendingNotice: `正在选择仓库 ${slot}...` }),
    swap: () => executeAction(context, swapRepositories, { pendingNotice: "正在交换仓库...", successNotice: "已交换仓库路径" }),
    changeDirection: (direction: Direction) =>
      executeAction(context, () => setDirection(direction), { pendingNotice: "正在切换同步方向..." }),
    save: () => executeAction(context, saveConfig, { pendingNotice: "正在保存配置...", successNotice: "配置已保存" }),
    refresh: () => executeAction(context, refreshState, { pendingNotice: "正在刷新状态...", successNotice: "状态已刷新" }),
    sync: () => executeAction(context, syncRepositories, { pendingNotice: "正在同步仓库...", successNotice: "同步完成" }),
    push: () => executeAction(context, pushTarget, { pendingNotice: "正在推送目标仓库...", successNotice: "推送完成" }),
  };
}

async function commitAction(
  context: ActionContext,
  commitMessage: string,
) {
  await executeAction(context, () => commitTarget(commitMessage), {
    pendingNotice: "正在创建提交...",
    successNotice: "提交完成",
  });
}

async function executeAction(
  context: ActionContext,
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
