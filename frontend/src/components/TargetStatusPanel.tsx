import type { TargetRepositoryStatus } from "../types";

interface TargetStatusPanelProps {
  status: TargetRepositoryStatus;
  commitMessage: string;
  onCommitMessageChange: (value: string) => void;
  onCommit: () => void;
  onPush: () => void;
  disableActions: boolean;
}

export function TargetStatusPanel(props: TargetStatusPanelProps) {
  const { status, commitMessage, onCommitMessageChange, onCommit, onPush, disableActions } = props;
  return (
    <section className="card status-card">
      <div className="panel-header compact">
        <div>
          <h2>目标仓库状态</h2>
          <p>同步、提交、推送都只针对当前目标仓库。</p>
        </div>
      </div>
      <div className="status-grid">
        <StatusRow label="仓库" value={status.name || "-"} />
        <StatusRow label="分支" value={status.branch || "-"} />
        <StatusRow label="状态" value={status.isClean ? "干净" : "有未提交改动"} />
        <StatusRow label="未提交" value={String(status.modifiedCount)} />
        <StatusRow label="未跟踪" value={String(status.untrackedCount)} />
      </div>
      <div className={`status-note ${status.isClean ? "ok" : "warn"}`}>
        {status.error || (status.isClean ? "目标仓库可直接提交或推送。" : "同步前请先判断现有改动是否需要保留。")}
      </div>
      <textarea
        className="commit-input"
        placeholder="输入提交信息"
        value={commitMessage}
        onChange={(event) => onCommitMessageChange(event.target.value)}
      />
      <div className="action-row">
        <button className="primary-button slim" onClick={onCommit} disabled={disableActions}>
          提交
        </button>
        <button className="ghost-button slim" onClick={onPush} disabled={disableActions}>
          推送
        </button>
      </div>
    </section>
  );
}

function StatusRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="status-row">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}
