import type { DashboardState, Direction, RepositorySlot } from "./types";

function backend() {
  return window.go.main.App;
}

export function loadState(): Promise<DashboardState> {
  return backend().LoadState();
}

export function refreshState(): Promise<DashboardState> {
  return backend().Refresh();
}

export function selectRepository(slot: RepositorySlot): Promise<DashboardState> {
  return backend().SelectRepository(slot);
}

export function swapRepositories(): Promise<DashboardState> {
  return backend().SwapRepositories();
}

export function setDirection(direction: Direction): Promise<DashboardState> {
  return backend().SetDirection(direction);
}

export function saveConfig(): Promise<DashboardState> {
  return backend().SaveConfig();
}

export function syncRepositories(): Promise<DashboardState> {
  return backend().SyncRepositories();
}

export function commitTarget(message: string): Promise<DashboardState> {
  return backend().CommitTarget(message);
}

export function generateCommitMessage(): Promise<string> {
  return backend().GenerateCommitMessage();
}

export function setAICommitAPIKey(apiKey: string): Promise<DashboardState> {
  return backend().SetAICommitAPIKey(apiKey);
}

export function pushTarget(): Promise<DashboardState> {
  return backend().PushTarget();
}
