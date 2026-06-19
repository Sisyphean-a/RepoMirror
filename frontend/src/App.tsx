import { useCallback, useMemo } from "react";
import { AppHeader } from "./components/AppHeader";
import { AppStatusBar } from "./components/AppStatusBar";
import { DiffPanel } from "./components/DiffPanel";
import { TargetStatusPanel } from "./components/TargetStatusPanel";
import "./App.css";
import { useRepoMirror } from "./useRepoMirror";

function App() {
  const viewModel = useRepoMirror();
  return viewModel.state ? <Dashboard {...viewModel} /> : <LoadingScreen error={viewModel.error} message={viewModel.busyMessage} />;
}

function Dashboard(viewModel: ReturnType<typeof useRepoMirror>) {
  const {
    busy,
    busyMessage,
    changeDirection,
    commit,
    error,
    generateCommit,
    lastUpdatedAt,
    notice,
    push,
    refresh,
    save,
    saveAICommitAPIKey,
    selectRepo,
    swap,
    sync,
  } = viewModel;
  const state = viewModel.state!;
  const sourceRepo = useMemo(
    () => (state.sourceSlot === "A" ? state.repositoryA : state.repositoryB),
    [state.repositoryA, state.repositoryB, state.sourceSlot],
  );
  const targetRepo = useMemo(
    () => (state.targetSlot === "A" ? state.repositoryA : state.repositoryB),
    [state.repositoryA, state.repositoryB, state.targetSlot],
  );
  const toggleDirection = state.config.direction === "A_TO_B" ? "B_TO_A" : "A_TO_B";
  const onToggleDirection = useCallback(() => void changeDirection(toggleDirection), [changeDirection, toggleDirection]);

  return (
    <div className="app-shell">
      <div className="app-frame">
        <AppHeader
          busy={busy}
          direction={state.config.direction}
          sourceSlot={state.sourceSlot}
          targetSlot={state.targetSlot}
          repositoryA={state.repositoryA}
          repositoryB={state.repositoryB}
          sourceRepo={sourceRepo}
          targetRepo={targetRepo}
          onRefresh={refresh}
          onSave={save}
          onSwap={swap}
          onToggleDirection={onToggleDirection}
          onSelectRepo={selectRepo}
        />

        <main className="workspace">
          <DiffPanel summary={state.summary} entries={state.differences} />
          <TargetStatusPanel
            status={state.targetStatus}
            summary={state.summary}
            targetSlot={state.targetSlot}
            canSync={state.canSync}
            busy={busy}
            error={error}
            onSync={sync}
            onCommit={commit}
            onGenerateCommit={generateCommit}
            onPush={push}
            onSaveAICommitAPIKey={saveAICommitAPIKey}
            aiCommitConfigured={state.aiCommitConfigured}
            disableActions={busy || !state.targetStatus.isGitRepo}
          />
        </main>
        <AppStatusBar busyMessage={busyMessage} error={error} notice={notice} lastUpdatedAt={lastUpdatedAt} />
        {busy ? <BusyOverlay message={busyMessage} /> : null}
      </div>
    </div>
  );
}

function LoadingScreen({ error, message }: { error: string; message: string }) {
  return (
    <div className="loading-screen">
      <div className={`loading-card ${error ? "error" : ""}`}>
        {error ? null : <span className="loading-spinner" aria-hidden="true" />}
        <span>{error || message || "正在扫描仓库状态..."}</span>
      </div>
    </div>
  );
}

function BusyOverlay({ message }: { message: string }) {
  return (
    <div className="busy-overlay" aria-busy="true" aria-live="polite" role="status">
      <div className="busy-card">
        <span className="loading-spinner" aria-hidden="true" />
        <span>{message || "正在处理..."}</span>
      </div>
    </div>
  );
}

export default App;
