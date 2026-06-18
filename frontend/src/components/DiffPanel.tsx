import type { DiffEntry, DiffFilter, DiffSummary } from "../types";
import { diffKindCode, diffKindLabel } from "../types";
import { SearchIcon } from "./Icons";
import { diffKindTone, formatSize, isDisabledAction } from "./ui";

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
        <span className="panel-title">Diff Plan</span>
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
          placeholder="path / type / rule..."
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
      {renderFilterButton("all", "All", summary.total, filter, onFilterChange)}
      {renderFilterButton("added", "Added", summary.added, filter, onFilterChange)}
      {renderFilterButton("modified", "Modified", summary.modified, filter, onFilterChange)}
      {renderFilterButton("deleted", "Deleted", summary.deleted, filter, onFilterChange)}
      {renderFilterButton("protected", "Protected", summary.protected, filter, onFilterChange)}
    </div>
  );
}

function DiffTable({ rows }: { rows: DiffEntry[] }) {
  if (rows.length === 0) {
    return (
      <div className="table-wrap">
        <TableHeader />
        <div className="table-body">
          <div className="empty-table">No files match the current filter.</div>
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
      <span className="col-type">Type</span>
      <span className="col-path">Path</span>
      <span className="col-rule">Rule</span>
      <span className="col-size">Size</span>
    </div>
  );
}

function DiffRow({ entry, alt }: { entry: DiffEntry; alt: boolean }) {
  const pathClassName = [
    "path-cell",
    entry.kind === "deleted" ? "deleted" : "",
    isDisabledAction(entry.kind) ? "protected" : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={`table-row ${alt ? "alt" : ""}`}>
      <span className={`type-badge ${diffKindTone(entry.kind)}`}>{diffKindCode[entry.kind]}</span>
      <span className={pathClassName} title={entry.path}>
        {entry.path}
      </span>
      <span className={`rule-cell ${entry.rule ? "visible" : ""}`}>{entry.rule || "—"}</span>
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
