import type { Direction, RepositorySummary, RepositorySlot } from "../types";
import { ArrowSplitIcon, BranchIcon, RefreshIcon, RepoMirrorLogo, SettingsIcon, SwapIcon } from "./Icons";
import { RepoField } from "./RepoField";
import { repoStateLabel, repoStateTone } from "./ui";

interface AppHeaderProps {
  busy: boolean;
  direction: Direction;
  sourceSlot: RepositorySlot;
  targetSlot: RepositorySlot;
  repositoryA: RepositorySummary;
  repositoryB: RepositorySummary;
  sourceRepo: RepositorySummary;
  targetRepo: RepositorySummary;
  onRefresh: () => void;
  onSave: () => void;
  onSwap: () => void;
  onToggleDirection: () => void;
  onSelectRepo: (slot: RepositorySlot) => void;
}

export function AppHeader(props: AppHeaderProps) {
  return (
    <header className="app-header">
      <HeaderTopBar {...props} />
      <RepositoryBars {...props} />
    </header>
  );
}

function HeaderTopBar({
  busy,
  sourceSlot,
  targetSlot,
  sourceRepo,
  targetRepo,
  onRefresh,
  onSave,
}: AppHeaderProps) {
  return (
    <div className="top-bar">
      <div className="brand-block">
        <RepoMirrorLogo className="brand-logo" />
        <span className="brand-name">RepoMirror</span>
      </div>
      <div className="top-divider" />
      <div className="direction-block">
        <span className="direction-pill">
          {sourceSlot} → {targetSlot}
        </span>
        <span className="top-repo muted">{sourceRepo.name || "源仓库"}</span>
        <ArrowSplitIcon className="inline-icon subtle-icon" />
        <span className="top-repo strong">{targetRepo.name || "目标仓库"}</span>
      </div>
      <div className="top-spacer" />
      <BranchStatus targetRepo={targetRepo} />
      <TopActions busy={busy} onRefresh={onRefresh} onSave={onSave} />
    </div>
  );
}

function RepositoryBars({
  busy,
  direction,
  repositoryA,
  repositoryB,
  targetSlot,
  onSelectRepo,
  onSwap,
  onToggleDirection,
}: AppHeaderProps) {
  return (
    <div className="repo-bars">
      <RepoField label="A" repository={repositoryA} isTarget={targetSlot === "A"} onSelect={onSelectRepo} />
      <div className="repo-col-divider" />
      <RepoField label="B" repository={repositoryB} isTarget={targetSlot === "B"} onSelect={onSelectRepo} />
      <RepositoryActions busy={busy} direction={direction} onSwap={onSwap} onToggleDirection={onToggleDirection} />
    </div>
  );
}

function BranchStatus({ targetRepo }: { targetRepo: RepositorySummary }) {
  const branch = targetRepo.isGitRepo ? targetRepo.branch || "HEAD" : "—";
  return (
    <div className="branch-block">
      <BranchIcon className="inline-icon subtle-icon" />
      <span className="branch-name">{branch}</span>
      <span className="branch-separator">·</span>
      <span className={`repo-state-text ${repoStateTone(targetRepo)}`}>{repoStateLabel(targetRepo)}</span>
    </div>
  );
}

function TopActions({
  busy,
  onRefresh,
  onSave,
}: {
  busy: boolean;
  onRefresh: () => void;
  onSave: () => void;
}) {
  return (
    <div className="top-actions">
      <button className="icon-button" disabled={busy} onClick={onRefresh} title="刷新状态" type="button">
        <RefreshIcon className="action-icon" />
      </button>
      <button className="icon-button" disabled={busy} onClick={onSave} title="保存配置" type="button">
        <SettingsIcon className="action-icon" />
      </button>
    </div>
  );
}

function RepositoryActions({
  busy,
  direction,
  onSwap,
  onToggleDirection,
}: {
  busy: boolean;
  direction: Direction;
  onSwap: () => void;
  onToggleDirection: () => void;
}) {
  const title = direction === "A_TO_B" ? "切换到 B → A" : "切换到 A → B";

  return (
    <div className="repo-inline-actions">
      <button className="icon-button" disabled={busy} onClick={onSwap} title="交换仓库" type="button">
        <SwapIcon className="action-icon" />
      </button>
      <button
        className={`icon-button ${direction === "A_TO_B" ? "active" : ""}`}
        disabled={busy}
        onClick={onToggleDirection}
        title={title}
        type="button"
      >
        <ArrowSplitIcon className="action-icon" />
      </button>
    </div>
  );
}
