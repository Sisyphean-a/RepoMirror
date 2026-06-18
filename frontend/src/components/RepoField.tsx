import type { RepositorySummary, RepositorySlot } from "../types";
import { BranchIcon, FolderIcon } from "./Icons";
import { repoStateLabel, repoStateTone } from "./ui";

interface RepoFieldProps {
  label: string;
  repository: RepositorySummary;
  isTarget: boolean;
  onSelect: (slot: RepositorySlot) => void;
}

export function RepoField({ label, repository, isTarget, onSelect }: RepoFieldProps) {
  const branch = repository.isGitRepo ? repository.branch || "HEAD" : "—";
  const tone = repoStateTone(repository);
  const path = repository.path || "Select git repository";

  return (
    <div className="repo-row">
      <div className={`repo-slot ${label === "B" ? "secondary" : ""}`}>{label}</div>
      <div className="repo-row-main">
        <div className="repo-row-path" title={path}>
          {path}
        </div>
      </div>
      <div className="repo-inline-meta">
        <BranchIcon className="tiny-icon muted-icon" />
        <span className="repo-branch">{branch}</span>
      </div>
      <div className={`repo-inline-meta ${tone}`}>
        <span className="repo-state-dot" />
        <span className="repo-state-label">{repoStateLabel(repository)}</span>
      </div>
      {isTarget ? <div className="repo-target-tag">target</div> : null}
      <button className="repo-select-button" onClick={() => onSelect(repository.slot)} type="button">
        <FolderIcon className="tiny-icon" />
      </button>
    </div>
  );
}
