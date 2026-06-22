import { memo, useState } from "react";
import type { DiffSummary, TargetRepositoryStatus } from "../types";
import { BranchIcon, CommitIcon, PushIcon, SyncIcon } from "./Icons";
import { commitHelperText, repoStateLabel } from "./ui";

interface TargetStatusPanelProps {
  status: TargetRepositoryStatus;
  summary: DiffSummary;
  targetSlot: "A" | "B";
  canSync: boolean;
  busy: boolean;
  error: string;
  onSync: () => void;
  onCommit: (message: string) => Promise<boolean>;
  onGenerateCommit: () => Promise<string>;
  onPush: () => void;
  onSaveAICommitAPIKey: (apiKey: string) => Promise<boolean>;
  aiCommitConfigured: boolean;
  disableActions: boolean;
}

export const TargetStatusPanel = memo(function TargetStatusPanel(props: TargetStatusPanelProps) {
  const [commitMessage, setCommitMessage] = useState("");

  return (
    <section className="target-panel">
      <div className="target-header">
        <span className="panel-title">目标仓库</span>
        <span className="target-slot">{props.targetSlot}</span>
      </div>
      <TargetBody props={props} commitMessage={commitMessage} onCommitMessageChange={setCommitMessage} />
    </section>
  );
});

function TargetBody({
  props,
  commitMessage,
  onCommitMessageChange,
}: {
  props: TargetStatusPanelProps;
  commitMessage: string;
  onCommitMessageChange: (value: string) => void;
}) {
  const actionState = buildActionState(props, commitMessage);
  const handleCommit = async () => {
    const committed = await props.onCommit(commitMessage);
    if (committed) {
      onCommitMessageChange("");
    }
  };

  return (
    <div className="target-body">
      <TargetRepoSummary status={props.status} />
      <div className="target-divider" />
      <SyncSummarySection summary={props.summary} />
      <CommitSection
        aiCommitConfigured={props.aiCommitConfigured}
        busy={props.busy}
        commitMessage={commitMessage}
        generateDisabled={actionState.generateDisabled}
        helperText={actionState.helperText}
        onGenerateCommit={props.onGenerateCommit}
        onSaveAICommitAPIKey={props.onSaveAICommitAPIKey}
        onCommitMessageChange={onCommitMessageChange}
      />
      <TargetActions
        commitDisabled={actionState.commitDisabled}
        pushDisabled={actionState.pushDisabled}
        syncDisabled={actionState.syncDisabled}
        onCommit={handleCommit}
        onPush={props.onPush}
        onSync={props.onSync}
      />
    </div>
  );
}

const TargetRepoSummary = memo(function TargetRepoSummary({ status }: { status: TargetRepositoryStatus }) {
  const branch = status.isGitRepo ? status.branch || "HEAD" : "—";
  const changeSummary = status.isGitRepo
    ? `${status.modifiedCount} 个未暂存 · ${status.untrackedCount} 个未跟踪`
    : "状态不可用";

  return (
    <div className="target-summary-block">
      <div className="target-repo-line">
        <span className="target-name">{status.name || "目标仓库"}</span>
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
});

const SyncSummarySection = memo(function SyncSummarySection({ summary }: { summary: DiffSummary }) {
  return (
    <div className="summary-section">
      <div className="section-label">同步摘要</div>
      <SummaryRow symbol="+" count={summary.added} label="新增" tone="added" />
      <SummaryRow symbol="~" count={summary.modified} label="修改" tone="modified" />
      <SummaryRow symbol="−" count={summary.deleted} label="删除" tone="deleted" />
    </div>
  );
});

function CommitSection({
  aiCommitConfigured,
  busy,
  commitMessage,
  generateDisabled,
  helperText,
  onGenerateCommit,
  onSaveAICommitAPIKey,
  onCommitMessageChange,
}: {
  aiCommitConfigured: boolean;
  busy: boolean;
  commitMessage: string;
  generateDisabled: boolean;
  helperText: string;
  onGenerateCommit: () => Promise<string>;
  onSaveAICommitAPIKey: (apiKey: string) => Promise<boolean>;
  onCommitMessageChange: (value: string) => void;
}) {
  const [apiKeyDraft, setAPIKeyDraft] = useState("");

  const handleSaveKey = async () => {
    const trimmed = apiKeyDraft.trim();
    if (!trimmed) {
      return;
    }
    const saved = await onSaveAICommitAPIKey(trimmed);
    if (saved) {
      setAPIKeyDraft("");
    }
  };

  const handleGenerateCommit = async () => {
    const trimmed = apiKeyDraft.trim();
    if (trimmed) {
      const saved = await onSaveAICommitAPIKey(trimmed);
      if (!saved) {
        return;
      }
      setAPIKeyDraft("");
    }
    const message = await onGenerateCommit();
    if (message) {
      onCommitMessageChange(message);
    }
  };

  return (
    <div className="commit-section">
      <div className="section-label">提交信息</div>
      <div className="ai-key-row">
        <input
          className="ai-key-input"
          placeholder={aiCommitConfigured ? "已保存 DeepSeek Key，输入可覆盖" : "输入 DeepSeek API Key"}
          type="password"
          value={apiKeyDraft}
          onChange={(event) => setAPIKeyDraft(event.target.value)}
        />
        <button className="secondary-button ai-key-button" disabled={busy || !apiKeyDraft.trim()} onClick={handleSaveKey} type="button">
          保存 Key
        </button>
      </div>
      <div className="ai-key-status">{aiCommitConfigured ? "已配置 DeepSeek Key" : "未配置 DeepSeek Key"}</div>
      <textarea
        className="commit-textarea"
        placeholder="chore: 从源仓库同步"
        value={commitMessage}
        onChange={(event) => onCommitMessageChange(event.target.value)}
      />
      <button className="secondary-button ai-generate-button" disabled={generateDisabled} onClick={handleGenerateCommit} type="button">
        AI 生成
      </button>
      <div className="commit-helper">{helperText}</div>
    </div>
  );
}

const TargetActions = memo(function TargetActions({
  commitDisabled,
  pushDisabled,
  syncDisabled,
  onCommit,
  onPush,
  onSync,
}: {
  commitDisabled: boolean;
  pushDisabled: boolean;
  syncDisabled: boolean;
  onCommit: () => void;
  onPush: () => void;
  onSync: () => void;
}) {
  return (
    <div className="target-actions">
      <button className="sync-button" disabled={syncDisabled} onClick={onSync} type="button">
        <SyncIcon className="button-icon" />
        <span>同步</span>
      </button>
      <div className="secondary-actions">
        <button className="secondary-button" disabled={commitDisabled} onClick={onCommit} type="button">
          <CommitIcon className="button-icon" />
          <span>提交</span>
        </button>
        <button className="secondary-button" disabled={pushDisabled} onClick={onPush} type="button">
          <PushIcon className="button-icon" />
          <span>推送</span>
        </button>
      </div>
    </div>
  );
});

interface ActionState {
  helperText: string;
  commitDisabled: boolean;
  generateDisabled: boolean;
  pushDisabled: boolean;
  syncDisabled: boolean;
}

function buildActionState(props: TargetStatusPanelProps, commitMessage: string): ActionState {
  const helperText = props.error || commitHelperText(props.status, commitMessage, props.busy);
  const hasPendingChanges = props.status.isGitRepo && !props.status.isClean;

  return {
    helperText,
    commitDisabled: props.disableActions || !hasPendingChanges || !commitMessage.trim(),
    generateDisabled: props.disableActions || !hasPendingChanges,
    pushDisabled: props.disableActions || !props.status.canPush,
    syncDisabled: props.busy || !props.canSync,
  };
}

const SummaryRow = memo(function SummaryRow({
  symbol,
  count,
  label,
  tone,
}: {
  symbol: string;
  count: number;
  label: string;
  tone: "added" | "modified" | "deleted";
}) {
  return (
    <div className="summary-row">
      <span className={`summary-symbol ${tone}`}>{symbol}</span>
      <span className={`summary-count ${tone}`}>{count}</span>
      <span className="summary-label">{label}</span>
    </div>
  );
});
