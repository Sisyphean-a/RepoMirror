import { memo, useEffect, useState } from "react";
import { ClockIcon, SaveIcon } from "./Icons";
import { formatRelativeTimeAt } from "./ui";

interface AppStatusBarProps {
  busyMessage: string;
  error: string;
  notice: string;
  lastUpdatedAt: number;
}

export const AppStatusBar = memo(function AppStatusBar({ busyMessage, error, notice, lastUpdatedAt }: AppStatusBarProps) {
  const [timeLabel, setTimeLabel] = useState(() => formatRelativeTimeAt(lastUpdatedAt, Date.now()));

  useEffect(() => {
    let timerId = 0;

    const updateTimeLabel = () => {
      const now = Date.now();
      setTimeLabel(formatRelativeTimeAt(lastUpdatedAt, now));
      const nextDelay = nextTimeLabelDelay(lastUpdatedAt, now);
      if (nextDelay > 0) {
        timerId = window.setTimeout(updateTimeLabel, nextDelay);
      }
    };

    updateTimeLabel();
    if (!lastUpdatedAt) {
      return;
    }

    return () => window.clearTimeout(timerId);
  }, [lastUpdatedAt]);

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
        <span>{timeLabel}</span>
      </div>
      <div className={`status-meta ${tone}`}>
        {busyMessage ? <span className="status-spinner" aria-hidden="true" /> : <SaveIcon className="status-icon" />}
        <span>{message}</span>
      </div>
    </footer>
  );
});

function nextTimeLabelDelay(lastUpdatedAt: number, now: number) {
  if (!lastUpdatedAt) {
    return 0;
  }

  const elapsedMs = Math.max(0, now - lastUpdatedAt);
  if (elapsedMs < 5_000) {
    return 5_000 - elapsedMs;
  }
  if (elapsedMs < 60_000) {
    return 1_000 - (elapsedMs % 1_000) || 1_000;
  }
  if (elapsedMs < 3_600_000) {
    return 60_000 - (elapsedMs % 60_000) || 60_000;
  }
  return 3_600_000 - (elapsedMs % 3_600_000) || 3_600_000;
}
