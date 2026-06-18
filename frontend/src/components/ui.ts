import type { DiffEntry, DiffKind, RepositorySummary, TargetRepositoryStatus } from "../types";

export function repoStateLabel(repository: RepositorySummary | TargetRepositoryStatus) {
  const repoError = "error" in repository ? repository.error : repository.validationError;
  if (!repository.isGitRepo) {
    return repoError || "unconfigured";
  }
  return repository.isClean ? "clean" : "dirty";
}

export function repoStateTone(repository: RepositorySummary | TargetRepositoryStatus) {
  if (!repository.isGitRepo) {
    return "warn";
  }
  return repository.isClean ? "ok" : "dirty";
}

export function formatSize(sizeBytes: number) {
  if (sizeBytes <= 0) {
    return "—";
  }
  if (sizeBytes < 1024) {
    return `${sizeBytes} B`;
  }
  const kiloBytes = sizeBytes / 1024;
  return `${kiloBytes.toFixed(kiloBytes >= 10 ? 0 : 1)} KB`;
}

export function formatRelativeTime(lastUpdatedAt: number) {
  if (!lastUpdatedAt) {
    return "scanned just now";
  }
  const elapsedSeconds = Math.max(0, Math.round((Date.now() - lastUpdatedAt) / 1000));
  if (elapsedSeconds < 5) {
    return "scanned just now";
  }
  return `scanned ${elapsedSeconds}s ago`;
}

export function diffKindTone(kind: DiffKind) {
  switch (kind) {
    case "added":
      return "added";
    case "modified":
      return "modified";
    case "deleted":
      return "deleted";
    case "protected":
      return "protected";
  }
}

export function isDisabledAction(kind: DiffKind) {
  return kind === "protected";
}

export function commitHelperText(
  status: TargetRepositoryStatus,
  commitMessage: string,
  busy: boolean,
) {
  if (!status.isGitRepo) {
    return "Target repository is unavailable";
  }
  if (busy) {
    return "Current action is still running";
  }
  if (status.isClean) {
    return "Nothing to commit in target repository";
  }
  if (!commitMessage.trim()) {
    return "Enter a commit message to enable Commit";
  }
  return "Ready to create target commit";
}

export function targetNotice(status: TargetRepositoryStatus) {
  if (!status.isGitRepo) {
    return status.error || "Target repository is unavailable.";
  }
  if (status.error) {
    return status.error;
  }
  if (status.isClean) {
    return "Target is clean. Sync can proceed directly.";
  }
  return "Target has local changes. Review before sync.";
}
