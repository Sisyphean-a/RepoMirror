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
        <span>skip .git</span>
        <span className="status-dot">·</span>
        <span>skip .gitignore</span>
        <span className="status-dot">·</span>
        <span>respect target ignore</span>
      </div>
      <div className="status-meta">
        <ClockIcon className="status-icon" />
        <span>{formatRelativeTime(lastUpdatedAt)}</span>
      </div>
      <div className={`status-meta ${error ? "error" : notice ? "success" : ""}`}>
        <SaveIcon className="status-icon" />
        <span>{error || notice || "ready"}</span>
      </div>
    </footer>
  );
}
