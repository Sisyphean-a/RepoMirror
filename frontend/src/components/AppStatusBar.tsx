import { ClockIcon, SaveIcon } from "./Icons";
import { formatRelativeTime } from "./ui";

interface AppStatusBarProps {
  busyMessage: string;
  error: string;
  notice: string;
  lastUpdatedAt: number;
}

export function AppStatusBar({ busyMessage, error, notice, lastUpdatedAt }: AppStatusBarProps) {
  const tone = busyMessage ? "busy" : error ? "error" : notice ? "success" : "";
  const message = busyMessage || error || notice || "就绪";

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
      <div className={`status-meta ${tone}`}>
        {busyMessage ? <span className="status-spinner" aria-hidden="true" /> : <SaveIcon className="status-icon" />}
        <span>{message}</span>
      </div>
    </footer>
  );
}
