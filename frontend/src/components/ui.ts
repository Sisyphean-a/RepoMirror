import type { DiffEntry, DiffKind, RepositorySummary, TargetRepositoryStatus } from "../types";

export function repoStateLabel(repository: RepositorySummary | TargetRepositoryStatus) {
  const repoError = "error" in repository ? repository.error : repository.validationError;
  if (!repository.isGitRepo) {
    return repoError || "未配置";
  }
  return repository.isClean ? "干净" : "有改动";
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
    return "刚刚扫描";
  }
  const elapsedSeconds = Math.max(0, Math.round((Date.now() - lastUpdatedAt) / 1000));
  if (elapsedSeconds < 5) {
    return "刚刚扫描";
  }
  return `${elapsedSeconds} 秒前扫描`;
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

export function commitHelperText(
  status: TargetRepositoryStatus,
  commitMessage: string,
  busy: boolean,
) {
  if (!status.isGitRepo) {
    return "目标仓库不可用";
  }
  if (busy) {
    return "当前操作仍在执行";
  }
  if (status.isClean) {
    return "目标仓库没有可提交内容";
  }
  if (!commitMessage.trim()) {
    return "请输入提交信息后再执行提交";
  }
  return "可以创建目标仓库提交";
}
