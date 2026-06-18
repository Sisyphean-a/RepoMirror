import type { RepositorySummary, RepositorySlot } from "../types";

interface RepoFieldProps {
  label: string;
  repository: RepositorySummary;
  onSelect: (slot: RepositorySlot) => void;
}

export function RepoField({ label, repository, onSelect }: RepoFieldProps) {
  const status = repository.isConfigured
    ? repository.isGitRepo
      ? "Git 仓库"
      : repository.validationError
    : "未选择";

  return (
    <div className="repo-field">
      <div className="repo-label">{label}</div>
      <div className="repo-path-panel">
        <div className="repo-path">{repository.path || "请选择 Git 仓库目录"}</div>
        <button className="ghost-button" onClick={() => onSelect(repository.slot)}>
          选目录
        </button>
      </div>
      <div className={`repo-status ${repository.isGitRepo ? "ok" : "warn"}`}>{status}</div>
    </div>
  );
}
