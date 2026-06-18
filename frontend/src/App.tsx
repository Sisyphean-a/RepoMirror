import { DiffPanel } from "./components/DiffPanel";
import { RepoField } from "./components/RepoField";
import { TargetStatusPanel } from "./components/TargetStatusPanel";
import "./App.css";
import { useRepoMirror } from "./useRepoMirror";

function App() {
  const viewModel = useRepoMirror();
  return viewModel.state ? <Dashboard {...viewModel} /> : <LoadingScreen error={viewModel.error} />;
}

function Dashboard(viewModel: ReturnType<typeof useRepoMirror>) {
  const { busy, commitMessage, error, filter, notice, visibleEntries } = viewModel;
  const state = viewModel.state!;
  return (
    <div className="app-shell">
      <header className="hero">
        <div>
          <div className="eyebrow">RepoMirror</div>
          <h1>双仓库镜像同步台</h1>
          <p>主视角永远是差异文件列表，其次才是同步动作与目标仓库状态。</p>
        </div>
        <div className="hero-actions">
          <div className="direction-chip">方向: {state.sourceSlot} → {state.targetSlot}</div>
          <button className="ghost-button" disabled={busy} onClick={() => void viewModel.refresh()}>
            刷新
          </button>
        </div>
      </header>

      <section className="card controls-card">
        <RepoField label="仓库 A" repository={state.repositoryA} onSelect={(slot) => void viewModel.selectRepo(slot)} />
        <RepoField label="仓库 B" repository={state.repositoryB} onSelect={(slot) => void viewModel.selectRepo(slot)} />
        <div className="toolbar">
          <button className="ghost-button" disabled={busy} onClick={() => void viewModel.swap()}>
            交换
          </button>
          <button className="ghost-button" disabled={busy} onClick={() => void viewModel.save()}>
            保存
          </button>
          <button className={`direction-button ${state.config.direction === "A_TO_B" ? "active" : ""}`} disabled={busy} onClick={() => void viewModel.changeDirection("A_TO_B")}>
            A → B
          </button>
          <button className={`direction-button ${state.config.direction === "B_TO_A" ? "active" : ""}`} disabled={busy} onClick={() => void viewModel.changeDirection("B_TO_A")}>
            B → A
          </button>
          <button className="primary-button" disabled={busy || !state.canSync} onClick={() => void viewModel.sync()}>
            执行同步
          </button>
        </div>
        <Banner error={error} notice={notice} />
      </section>

      <main className="workspace">
        <DiffPanel filter={filter} summary={state.summary} entries={visibleEntries} onFilterChange={viewModel.setFilter} />
        <TargetStatusPanel
          status={state.targetStatus}
          commitMessage={commitMessage}
          onCommitMessageChange={viewModel.setCommitMessage}
          onCommit={() => void viewModel.commit()}
          onPush={() => void viewModel.push()}
          disableActions={busy || !state.targetStatus.isGitRepo}
        />
      </main>

      <footer className="rules-note">不同步 .gitignore ｜ 不修改 .git ｜ 目标端 ignore 内容受保护</footer>
    </div>
  );
}

function Banner({ error, notice }: { error: string; notice: string }) {
  if (!error && !notice) {
    return null;
  }
  return <div className={`banner ${error ? "error" : "success"}`}>{error || notice}</div>;
}

function LoadingScreen({ error }: { error: string }) {
  return <div className="loading-screen">{error || "正在加载仓库状态..."}</div>;
}

export default App;
