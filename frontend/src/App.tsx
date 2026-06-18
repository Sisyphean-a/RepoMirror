import { AppHeader } from "./components/AppHeader";
import { AppStatusBar } from "./components/AppStatusBar";
import { DiffPanel } from "./components/DiffPanel";
import { TargetStatusPanel } from "./components/TargetStatusPanel";
import "./App.css";
import { useRepoMirror } from "./useRepoMirror";

function App() {
  const viewModel = useRepoMirror();
  return viewModel.state ? <Dashboard {...viewModel} /> : <LoadingScreen error={viewModel.error} />;
}

function Dashboard(viewModel: ReturnType<typeof useRepoMirror>) {
  const { busy, commitMessage, error, filter, lastUpdatedAt, notice, searchTerm, visibleEntries } = viewModel;
  const state = viewModel.state!;
  const sourceRepo = state.sourceSlot === "A" ? state.repositoryA : state.repositoryB;
  const targetRepo = state.targetSlot === "A" ? state.repositoryA : state.repositoryB;

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
          onRefresh={() => void viewModel.refresh()}
          onSave={() => void viewModel.save()}
          onSwap={() => void viewModel.swap()}
          onToggleDirection={() =>
            void viewModel.changeDirection(state.config.direction === "A_TO_B" ? "B_TO_A" : "A_TO_B")
          }
          onSelectRepo={(slot) => void viewModel.selectRepo(slot)}
        />

        <main className="workspace">
          <DiffPanel
            filter={filter}
            summary={state.summary}
            entries={visibleEntries}
            searchTerm={searchTerm}
            onFilterChange={viewModel.setFilter}
            onSearchTermChange={viewModel.setSearchTerm}
          />
          <TargetStatusPanel
            status={state.targetStatus}
            summary={state.summary}
            targetSlot={state.targetSlot}
            canSync={state.canSync}
            busy={busy}
            commitMessage={commitMessage}
            error={error}
            onCommitMessageChange={viewModel.setCommitMessage}
            onSync={() => void viewModel.sync()}
            onCommit={() => void viewModel.commit()}
            onPush={() => void viewModel.push()}
            disableActions={busy || !state.targetStatus.isGitRepo}
          />
        </main>
        <AppStatusBar error={error} notice={notice} lastUpdatedAt={lastUpdatedAt} />
      </div>
    </div>
  );
}

function LoadingScreen({ error }: { error: string }) {
  return <div className="loading-screen">{error || "正在扫描仓库状态..."}</div>;
}

export default App;
