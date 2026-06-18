import type { DiffSummary, TargetRepositoryStatus } from "../types";
import { BranchIcon, CommitIcon, PushIcon, SyncIcon, WarningIcon } from "./Icons";
import { commitHelperText, repoStateLabel, targetNotice } from "./ui";

interface TargetStatusPanelProps {
  status: TargetRepositoryStatus;
  summary: DiffSummary;
  targetSlot: "A" | "B";
  canSync: boolean;
  busy: boolean;
  commitMessage: string;
  error: string;
  onCommitMessageChange: (value: string) => void;
  onSync: () => void;
  onCommit: () => void;
  onPush: () => void;
  disableActions: boolean;
}

export function TargetStatusPanel(props: TargetStatusPanelProps) {
  return (
    <section className="target-panel">
      <div className="target-header">
        <span className="panel-title">Target</span>
        <span className="target-slot">{props.targetSlot}</span>
      </div>
      <TargetBody props={props} />
    </section>
  );
}

function TargetBody({ props }: { props: TargetStatusPanelProps }) {
  const actionState = buildActionState(props);

  return (
    <div className="target-body">
      <TargetRepoSummary status={props.status} />
      <div className="target-divider" />
      <SyncSummarySection summary={props.summary} />
      <div className="target-divider" />
      <TargetWarning status={props.status} />
      <CommitSection
        commitMessage={props.commitMessage}
        helperText={actionState.helperText}
        onCommitMessageChange={props.onCommitMessageChange}
      />
      <TargetActions
        commitDisabled={actionState.commitDisabled}
        disableActions={props.disableActions}
        syncDisabled={actionState.syncDisabled}
        onCommit={props.onCommit}
        onPush={props.onPush}
        onSync={props.onSync}
      />
    </div>
  );
}

function TargetRepoSummary({ status }: { status: TargetRepositoryStatus }) {
  const branch = status.isGitRepo ? status.branch || "HEAD" : "—";
  const changeSummary = status.isGitRepo
    ? `${status.modifiedCount} unstaged · ${status.untrackedCount} untracked`
    : "status unavailable";

  return (
    <div className="target-summary-block">
      <div className="target-repo-line">
        <span className="target-name">{status.name || "target-repo"}</span>
        <span className={`target-dirty ${status.isClean ? "ok" : "warn"}`}>{repoStateLabel(status)}</span>
      </div>
      <div className="target-meta-line">
        <span className="target-branch">
          <BranchIcon className="tiny-icon muted-icon" />
          {branch}
        </span>
        <span className="target-changes">{changeSummary}</span>
      </div>
    </div>
  );
}

function SyncSummarySection({ summary }: { summary: DiffSummary }) {
  return (
    <div className="summary-section">
      <div className="section-label">Sync Summary</div>
      <SummaryRow symbol="+" count={summary.added} label="added" tone="added" />
      <SummaryRow symbol="~" count={summary.modified} label="modified" tone="modified" />
      <SummaryRow symbol="−" count={summary.deleted} label="deleted" tone="deleted" />
      <SummaryRow symbol="" count={summary.protected} label="protected" tone="protected" />
    </div>
  );
}

function TargetWarning({ status }: { status: TargetRepositoryStatus }) {
  return (
    <div className="warning-strip">
      <WarningIcon className="warning-icon" />
      <span>{targetNotice(status)}</span>
    </div>
  );
}

function CommitSection({
  commitMessage,
  helperText,
  onCommitMessageChange,
}: {
  commitMessage: string;
  helperText: string;
  onCommitMessageChange: (value: string) => void;
}) {
  return (
    <div className="commit-section">
      <div className="section-label">Commit Message</div>
      <textarea
        className="commit-textarea"
        placeholder="chore: mirror sync from source"
        value={commitMessage}
        onChange={(event) => onCommitMessageChange(event.target.value)}
      />
      <div className="commit-helper">{helperText}</div>
    </div>
  );
}

function TargetActions({
  commitDisabled,
  disableActions,
  syncDisabled,
  onCommit,
  onPush,
  onSync,
}: {
  commitDisabled: boolean;
  disableActions: boolean;
  syncDisabled: boolean;
  onCommit: () => void;
  onPush: () => void;
  onSync: () => void;
}) {
  return (
    <div className="target-actions">
      <button className="sync-button" disabled={syncDisabled} onClick={onSync} type="button">
        <SyncIcon className="button-icon" />
        <span>Sync</span>
      </button>
      <div className="secondary-actions">
        <button className="secondary-button" disabled={commitDisabled} onClick={onCommit} type="button">
          <CommitIcon className="button-icon" />
          <span>Commit</span>
        </button>
        <button className="secondary-button" disabled={disableActions} onClick={onPush} type="button">
          <PushIcon className="button-icon" />
          <span>Push</span>
        </button>
      </div>
    </div>
  );
}

interface ActionState {
  helperText: string;
  commitDisabled: boolean;
  syncDisabled: boolean;
}

function buildActionState(props: TargetStatusPanelProps): ActionState {
  const helperText = props.error || commitHelperText(props.status, props.commitMessage, props.busy);
  const hasPendingChanges = props.status.isGitRepo && !props.status.isClean;

  return {
    helperText,
    commitDisabled: props.disableActions || !hasPendingChanges || !props.commitMessage.trim(),
    syncDisabled: props.busy || !props.canSync,
  };
}

function SummaryRow({
  symbol,
  count,
  label,
  tone,
}: {
  symbol: string;
  count: number;
  label: string;
  tone: "added" | "modified" | "deleted" | "protected";
}) {
  return (
    <div className="summary-row">
      <span className={`summary-symbol ${tone}`}>{symbol}</span>
      <span className={`summary-count ${tone}`}>{count}</span>
      <span className="summary-label">{label}</span>
    </div>
  );
}
