import { ClockIcon, SaveIcon } from "./Icons";
import { formatRelativeTime } from "./ui";

interface AppStatusBarProps {
  error: string;
  notice: string;
  lastUpdatedAt: number;
}

export function AppStatusBar({ error, notice, lastUpdatedAt }: AppStatusBarProps) {
  return (
    <footer className="status-bar">
      <div className="status-meta">
        <span>跳过 .git</span>
        <span className="status-dot">·</span>
        <span>跳过 .gitignore</span>
        <span className="status-dot">·</span>
        <span>遵守目标仓库忽略规则</span>
      </div>
      <div className="status-meta">
        <ClockIcon className="status-icon" />
        <span>{formatRelativeTime(lastUpdatedAt)}</span>
      </div>
      <div className={`status-meta ${error ? "error" : notice ? "success" : ""}`}>
        <SaveIcon className="status-icon" />
        <span>{error || notice || "就绪"}</span>
      </div>
    </footer>
  );
}
