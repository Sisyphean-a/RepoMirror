import type { DashboardState, Direction, RepositorySlot } from "./types";

interface RepoMirrorBackend {
  LoadState(): Promise<DashboardState>;
  Refresh(): Promise<DashboardState>;
  SelectRepository(slot: RepositorySlot): Promise<DashboardState>;
  SwapRepositories(): Promise<DashboardState>;
  SetDirection(direction: Direction): Promise<DashboardState>;
  SaveConfig(): Promise<DashboardState>;
  SyncRepositories(): Promise<DashboardState>;
  CommitTarget(message: string): Promise<DashboardState>;
  GenerateCommitMessage(): Promise<string>;
  SetAICommitAPIKey(apiKey: string): Promise<DashboardState>;
  PushTarget(): Promise<DashboardState>;
}

declare global {
  interface Window {
    go: {
      main: {
        App: RepoMirrorBackend;
      };
    };
  }
}

export {};
