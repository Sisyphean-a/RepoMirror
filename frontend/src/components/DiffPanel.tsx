import type { DiffEntry, DiffFilter, DiffSummary } from "../types";
import { diffKindLabel } from "../types";

interface DiffPanelProps {
  filter: DiffFilter;
  summary: DiffSummary;
  entries: DiffEntry[];
  onFilterChange: (filter: DiffFilter) => void;
}

export function DiffPanel({ filter, summary, entries, onFilterChange }: DiffPanelProps) {
  return (
    <section className="card diff-card">
      <div className="panel-header">
        <div>
          <h2>差异文件</h2>
          <p>源端输出 + 目标端保护规则共同决定最终同步结果。</p>
        </div>
        <div className="filter-row">
          {renderFilterButton("all", "全部", filter, onFilterChange)}
          {renderFilterButton("added", "新增", filter, onFilterChange, summary.added)}
          {renderFilterButton("modified", "修改", filter, onFilterChange, summary.modified)}
          {renderFilterButton("deleted", "删除", filter, onFilterChange, summary.deleted)}
        </div>
      </div>
      <div className="diff-list">
        {entries.length === 0 ? (
          <div className="empty-state">当前方向下没有需要同步的差异。</div>
        ) : (
          entries.map((entry) => (
            <div className={`diff-item ${entry.kind}`} key={`${entry.kind}-${entry.path}`}>
              <span className="diff-badge">{diffKindLabel[entry.kind]}</span>
              <span className="diff-path">{entry.path}</span>
            </div>
          ))
        )}
      </div>
      <div className="panel-footer">共 {summary.total} 项差异</div>
    </section>
  );
}

function renderFilterButton(
  value: DiffFilter,
  label: string,
  active: DiffFilter,
  onChange: (filter: DiffFilter) => void,
  count?: number,
) {
  return (
    <button
      className={`filter-button ${active === value ? "active" : ""}`}
      onClick={() => onChange(value)}
      type="button"
    >
      {label}
      {count !== undefined ? ` ${count}` : ""}
    </button>
  );
}
