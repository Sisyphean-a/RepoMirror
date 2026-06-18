import type { DiffEntry, DiffFilter, DiffSummary } from "../types";
import { diffKindCode, diffKindLabel } from "../types";
import { SearchIcon } from "./Icons";
import { diffKindTone, formatSize } from "./ui";

interface DiffPanelProps {
  filter: DiffFilter;
  summary: DiffSummary;
  entries: DiffEntry[];
  searchTerm: string;
  onFilterChange: (filter: DiffFilter) => void;
  onSearchTermChange: (value: string) => void;
}

export function DiffPanel(props: DiffPanelProps) {
  return (
    <section className="diff-panel">
      <DiffPanelTopbar
        totalVisible={props.entries.length}
        total={props.summary.total}
        searchTerm={props.searchTerm}
        onSearchTermChange={props.onSearchTermChange}
      />
      <DiffFilterBar filter={props.filter} summary={props.summary} onFilterChange={props.onFilterChange} />
      <DiffTable rows={props.entries} />
    </section>
  );
}

function DiffPanelTopbar({
  totalVisible,
  total,
  searchTerm,
  onSearchTermChange,
}: {
  totalVisible: number;
  total: number;
  searchTerm: string;
  onSearchTermChange: (value: string) => void;
}) {
  return (
    <div className="panel-topbar">
      <div className="panel-title-wrap">
        <span className="panel-title">差异列表</span>
        <span className="panel-count">
          {totalVisible} / {total}
        </span>
      </div>
      <label className="search-box">
        <SearchIcon className="search-icon" />
        <input
          className="search-input"
          value={searchTerm}
          onChange={(event) => onSearchTermChange(event.target.value)}
          placeholder="搜索路径或类型"
          type="text"
        />
      </label>
    </div>
  );
}

function DiffFilterBar({
  filter,
  summary,
  onFilterChange,
}: {
  filter: DiffFilter;
  summary: DiffSummary;
  onFilterChange: (filter: DiffFilter) => void;
}) {
  return (
    <div className="filter-bar">
      {renderFilterButton("all", "全部", summary.total, filter, onFilterChange)}
      {renderFilterButton("added", "新增", summary.added, filter, onFilterChange)}
      {renderFilterButton("modified", "修改", summary.modified, filter, onFilterChange)}
      {renderFilterButton("deleted", "删除", summary.deleted, filter, onFilterChange)}
    </div>
  );
}

function DiffTable({ rows }: { rows: DiffEntry[] }) {
  if (rows.length === 0) {
    return (
      <div className="table-wrap">
        <TableHeader />
        <div className="table-body">
          <div className="empty-table">当前筛选下没有差异文件</div>
        </div>
      </div>
    );
  }

  return (
    <div className="table-wrap">
      <TableHeader />
      <div className="table-body">
        {rows.map((entry, index) => <DiffRow entry={entry} alt={index % 2 === 0} key={`${entry.kind}-${entry.path}`} />)}
      </div>
    </div>
  );
}

function TableHeader() {
  return (
    <div className="table-header">
      <span className="col-type">类型</span>
      <span className="col-path">路径</span>
      <span className="col-size">大小</span>
    </div>
  );
}

function DiffRow({ entry, alt }: { entry: DiffEntry; alt: boolean }) {
  const pathClassName = ["path-cell", entry.kind === "deleted" ? "deleted" : ""].filter(Boolean).join(" ");

  return (
    <div className={`table-row ${alt ? "alt" : ""}`}>
      <span className={`type-badge ${diffKindTone(entry.kind)}`}>{diffKindCode[entry.kind]}</span>
      <span className={pathClassName} title={entry.path}>
        {entry.path}
      </span>
      <span className="size-cell">{formatSize(entry.sizeBytes)}</span>
    </div>
  );
}

function renderFilterButton(
  value: DiffFilter,
  label: string,
  count: number,
  active: DiffFilter,
  onChange: (filter: DiffFilter) => void,
) {
  return (
    <button
      className={`filter-pill ${active === value ? "active" : ""}`}
      onClick={() => onChange(value)}
      type="button"
      title={`${diffKindLabel[value as keyof typeof diffKindLabel] ?? label}: ${count}`}
    >
      <span>{label}</span>
      <span className="filter-count">{count}</span>
    </button>
  );
}
