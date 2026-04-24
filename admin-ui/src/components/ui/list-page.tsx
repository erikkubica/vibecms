import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { Plus, Pencil, Trash2, Eye, ExternalLink } from "lucide-react";

export function ListPageShell({ children }: { children: ReactNode }) {
  return <div className="w-full pb-8">{children}</div>;
}

interface ListHeaderProps {
  title: string;
  count?: number;
  tabs?: { value: string; label: string; count?: number }[];
  activeTab?: string;
  onTabChange?: (v: string) => void;
  newLabel?: string;
  newHref?: string;
  onNew?: () => void;
  extra?: ReactNode;
}

export function ListHeader({ count, tabs, activeTab, onTabChange, newLabel, newHref, onNew, extra }: ListHeaderProps) {
  return (
    <div className="flex items-center gap-0 border-b border-slate-200 mb-3">
      {tabs && tabs.length > 0 ? (
        <nav className="flex-1 flex items-center gap-0.5 -mb-px">
          {tabs.map((t) => {
            const active = t.value === activeTab;
            return (
              <button
                key={t.value}
                type="button"
                onClick={() => onTabChange?.(t.value)}
                className={`px-2.5 pt-[7px] pb-[9px] inline-flex items-center gap-1.5 text-[12.5px] cursor-pointer border-b-2 bg-transparent ${
                  active
                    ? "font-semibold text-slate-900 border-indigo-600"
                    : "font-medium text-slate-500 border-transparent hover:text-slate-700"
                }`}
              >
                {t.label}
                {t.count !== undefined && (
                  <span
                    className={`font-mono text-[10.5px] px-1.5 py-px rounded-full border ${
                      active
                        ? "border-slate-200 bg-indigo-50 text-indigo-600"
                        : "border-slate-200 bg-slate-100 text-slate-500"
                    }`}
                  >
                    {t.count}
                  </span>
                )}
              </button>
            );
          })}
        </nav>
      ) : (
        <div className="flex-1 pb-[10px]">
          {count !== undefined && (
            <span className="font-mono text-[11.5px] font-medium text-slate-500">{count} items</span>
          )}
        </div>
      )}
      <div className="flex gap-1.5 pb-1.5">
        {extra}
        {(newHref || onNew) && (
          newHref ? (
            <Link
              to={newHref}
              className="h-[26px] px-2.5 inline-flex items-center gap-1.5 text-[12px] font-medium text-white bg-indigo-600 border border-indigo-600 rounded hover:bg-indigo-700"
            >
              <Plus className="w-3 h-3" />
              {newLabel ?? "New"}
            </Link>
          ) : (
            <button
              type="button"
              onClick={onNew}
              className="h-[26px] px-2.5 inline-flex items-center gap-1.5 text-[12px] font-medium text-white bg-indigo-600 border border-indigo-600 rounded hover:bg-indigo-700 cursor-pointer"
            >
              <Plus className="w-3 h-3" />
              {newLabel ?? "New"}
            </button>
          )
        )}
      </div>
    </div>
  );
}

export function ListToolbar({ children }: { children: ReactNode }) {
  return <div className="flex items-center gap-2 mb-2.5 flex-wrap">{children}</div>;
}

export function ListSearch({
  value,
  onChange,
  placeholder = "Search…",
}: {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
}) {
  return (
    <div className="flex-1 max-w-[440px] relative">
      <svg
        className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-slate-400"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="11" cy="11" r="8" />
        <path d="m21 21-4.35-4.35" />
      </svg>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="h-[30px] w-full pl-8 pr-3 bg-white border border-slate-300 rounded text-[13px] placeholder:text-slate-400 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/30 outline-none"
      />
    </div>
  );
}

export function ListCard({ children }: { children: ReactNode }) {
  return (
    <div className="bg-white border border-slate-200 rounded-lg shadow-sm overflow-hidden">
      {children}
    </div>
  );
}

export function ListTable({ children, minWidth = 960 }: { children: ReactNode; minWidth?: number }) {
  return (
    <div className="overflow-x-auto">
      <table
        className="w-full border-separate border-spacing-0"
        style={{ minWidth: `${minWidth}px` }}
      >
        {children}
      </table>
    </div>
  );
}

export function Th({
  children,
  width,
  align = "left",
  className = "",
}: {
  children?: ReactNode;
  width?: string | number;
  align?: "left" | "right" | "center";
  className?: string;
}) {
  const alignCls = align === "right" ? "text-right" : align === "center" ? "text-center" : "text-left";
  const style = width ? { width: typeof width === "number" ? `${width}px` : width } : undefined;
  return (
    <th
      style={style}
      className={`${alignCls} px-3 py-2.5 bg-slate-50 border-b border-slate-200 text-[10.5px] font-semibold uppercase tracking-[0.06em] text-slate-500 whitespace-nowrap ${className}`}
    >
      {children}
    </th>
  );
}

export function Tr({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <tr className={`group bg-white hover:bg-slate-50 ${className}`}>{children}</tr>
  );
}

export function Td({
  children,
  className = "",
  align = "left",
}: {
  children?: ReactNode;
  className?: string;
  align?: "left" | "right" | "center";
}) {
  const alignCls = align === "right" ? "text-right" : align === "center" ? "text-center" : "";
  return (
    <td className={`px-3 py-2.5 border-b border-slate-100 text-[13px] text-slate-800 group-last:border-0 ${alignCls} ${className}`}>
      {children}
    </td>
  );
}

type StatusKind = "published" | "draft" | "archived" | "active" | "inactive" | "neutral" | "success" | "warning" | "danger";

export function StatusPill({ status, label }: { status: StatusKind | string; label?: string }) {
  const palette: Record<string, { text: string; bg: string; border: string; dot: string; ring: string }> = {
    published: { text: "text-emerald-700", bg: "bg-emerald-50", border: "border-emerald-200", dot: "bg-emerald-500", ring: "ring-emerald-200" },
    active: { text: "text-emerald-700", bg: "bg-emerald-50", border: "border-emerald-200", dot: "bg-emerald-500", ring: "ring-emerald-200" },
    success: { text: "text-emerald-700", bg: "bg-emerald-50", border: "border-emerald-200", dot: "bg-emerald-500", ring: "ring-emerald-200" },
    draft: { text: "text-slate-600", bg: "bg-slate-50", border: "border-slate-200", dot: "bg-slate-400", ring: "ring-slate-200" },
    inactive: { text: "text-slate-600", bg: "bg-slate-50", border: "border-slate-200", dot: "bg-slate-400", ring: "ring-slate-200" },
    neutral: { text: "text-slate-600", bg: "bg-slate-50", border: "border-slate-200", dot: "bg-slate-400", ring: "ring-slate-200" },
    archived: { text: "text-amber-700", bg: "bg-amber-50", border: "border-amber-200", dot: "bg-amber-500", ring: "ring-amber-200" },
    warning: { text: "text-amber-700", bg: "bg-amber-50", border: "border-amber-200", dot: "bg-amber-500", ring: "ring-amber-200" },
    danger: { text: "text-red-700", bg: "bg-red-50", border: "border-red-200", dot: "bg-red-500", ring: "ring-red-200" },
  };
  const p = palette[status] ?? palette.neutral;
  return (
    <span className={`font-mono inline-flex items-center gap-1.5 pl-1.5 pr-2 py-px text-[11px] font-medium rounded-full border ${p.text} ${p.bg} ${p.border}`}>
      <span className={`w-[5px] h-[5px] rounded-full ring-2 ${p.dot} ${p.ring}`} />
      {label ?? status}
    </span>
  );
}

export function Chip({ children }: { children: ReactNode }) {
  return (
    <span className="inline-flex items-center px-1.5 py-px text-[11px] font-medium text-slate-700 bg-slate-50 border border-slate-200 rounded-[2px] whitespace-nowrap">
      {children}
    </span>
  );
}

export function SlugLink({ slug, href }: { slug: string; href?: string }) {
  const content = (
    <>
      /{slug}
      {href && (
        <ExternalLink className="w-2.5 h-2.5 opacity-70" aria-hidden />
      )}
    </>
  );
  if (href) {
    return (
      <a
        href={href}
        target="_blank"
        rel="noreferrer"
        className="font-mono text-[11px] text-indigo-600 inline-flex items-center gap-1 mt-0.5 hover:underline"
      >
        {content}
      </a>
    );
  }
  return (
    <span className="font-mono text-[11px] text-slate-500 inline-flex items-center gap-1 mt-0.5">
      {content}
    </span>
  );
}

export function TitleCell({
  to,
  title,
  slug,
  href,
  extra,
}: {
  to?: string;
  title: string;
  slug?: string;
  href?: string;
  extra?: ReactNode;
}) {
  return (
    <div className="min-w-0">
      <div className="flex items-center gap-1.5">
        {to ? (
          <Link to={to} className="text-[13px] font-medium text-slate-900 hover:text-indigo-600 truncate">
            {title}
          </Link>
        ) : (
          <span className="text-[13px] font-medium text-slate-900 truncate">{title}</span>
        )}
        {extra}
      </div>
      {slug && <SlugLink slug={slug} href={href} />}
    </div>
  );
}

export function RowActions({
  editTo,
  onEdit,
  onDelete,
  previewHref,
  disableDelete,
  deleteTitle,
  extra,
}: {
  editTo?: string;
  onEdit?: () => void;
  onDelete?: () => void;
  previewHref?: string;
  disableDelete?: boolean;
  deleteTitle?: string;
  extra?: ReactNode;
}) {
  const iconBtn = "w-[26px] h-[26px] grid place-items-center text-slate-500 hover:bg-slate-100 hover:border-slate-200 border border-transparent rounded-[2px] cursor-pointer bg-transparent";
  const delBtn = "w-[26px] h-[26px] grid place-items-center text-red-500/80 hover:text-red-600 hover:bg-red-50 hover:border-red-200 border border-transparent rounded-[2px] cursor-pointer bg-transparent disabled:opacity-40 disabled:cursor-not-allowed";
  return (
    <div className="inline-flex gap-0.5 opacity-55 group-hover:opacity-100 transition-opacity">
      {previewHref && (
        <a title="Preview" href={previewHref} target="_blank" rel="noreferrer" className={iconBtn}>
          <Eye className="w-3 h-3" />
        </a>
      )}
      {extra}
      {editTo && (
        <Link title="Edit" to={editTo} className={iconBtn}>
          <Pencil className="w-3 h-3" />
        </Link>
      )}
      {onEdit && !editTo && (
        <button title="Edit" type="button" onClick={onEdit} className={iconBtn}>
          <Pencil className="w-3 h-3" />
        </button>
      )}
      {onDelete && (
        <button
          title={deleteTitle ?? "Delete"}
          type="button"
          onClick={onDelete}
          disabled={disableDelete}
          className={delBtn}
        >
          <Trash2 className="w-3 h-3" />
        </button>
      )}
    </div>
  );
}

const PER_PAGE_OPTIONS = [10, 25, 50, 100];

export function ListFooter({
  page,
  totalPages,
  total,
  perPage,
  onPage,
  onPerPage,
  label = "items",
}: {
  page: number;
  totalPages: number;
  total: number;
  perPage: number;
  onPage: (p: number) => void;
  onPerPage?: (n: number) => void;
  label?: string;
}) {
  if (totalPages <= 1 && total <= perPage && !onPerPage) return null;
  const start = total === 0 ? 0 : (page - 1) * perPage + 1;
  const end = Math.min(page * perPage, total);
  const pages: number[] = [];
  const max = 5;
  let lo = Math.max(1, page - 2);
  const hi = Math.min(totalPages, lo + max - 1);
  lo = Math.max(1, hi - max + 1);
  for (let i = lo; i <= hi; i++) pages.push(i);
  return (
    <div className="flex items-center justify-between px-3.5 py-2.5 border-t border-slate-200 bg-slate-50">
      <div className="flex items-center gap-3 text-[12px] text-slate-500">
        <span>
          Showing <span className="text-slate-900 font-medium">{start}–{end}</span> of{" "}
          <span className="text-slate-900 font-medium">{total}</span> {label}
        </span>
        {onPerPage && (
          <label className="flex items-center gap-1.5">
            <span className="text-slate-400">Per page</span>
            <select
              value={perPage}
              onChange={(e) => onPerPage(Number(e.target.value))}
              className="h-[24px] pl-1.5 pr-5 text-[12px] text-slate-700 bg-white border border-slate-200 rounded appearance-none cursor-pointer focus:outline-none focus:border-indigo-400"
            >
              {PER_PAGE_OPTIONS.map((n) => (
                <option key={n} value={n}>{n}</option>
              ))}
            </select>
          </label>
        )}
      </div>
      <div className="flex items-center gap-1.5">
        <button
          type="button"
          disabled={page <= 1}
          onClick={() => onPage(page - 1)}
          className="h-[26px] px-2 text-[12px] text-slate-700 inline-flex items-center gap-1 hover:bg-slate-100 rounded disabled:opacity-45 disabled:cursor-not-allowed cursor-pointer bg-transparent border-0"
        >
          Prev
        </button>
        <div className="inline-flex items-center gap-0.5 p-0.5 bg-white border border-slate-200 rounded">
          {pages.map((p) => (
            <button
              key={p}
              type="button"
              onClick={() => onPage(p)}
              className={`min-w-[22px] h-[22px] px-1.5 text-[12px] rounded-[2px] font-mono cursor-pointer border-0 ${
                p === page
                  ? "font-semibold text-white bg-indigo-600"
                  : "font-medium text-slate-700 bg-transparent hover:bg-slate-100"
              }`}
            >
              {p}
            </button>
          ))}
        </div>
        <button
          type="button"
          disabled={page >= totalPages}
          onClick={() => onPage(page + 1)}
          className="h-[26px] px-2 text-[12px] text-slate-700 inline-flex items-center gap-1 hover:bg-slate-100 rounded disabled:opacity-45 disabled:cursor-not-allowed cursor-pointer bg-transparent border-0"
        >
          Next
        </button>
      </div>
    </div>
  );
}

export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
}: {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex h-64 flex-col items-center justify-center gap-3 text-slate-400">
      <Icon className="h-12 w-12" />
      <p className="text-[15px] font-medium text-slate-600">{title}</p>
      {description && <p className="text-[13px] text-slate-400">{description}</p>}
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
}

export function LoadingRow() {
  return (
    <div className="flex h-64 items-center justify-center">
      <svg className="h-6 w-6 animate-spin text-indigo-500" viewBox="0 0 24 24" fill="none">
        <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="3" strokeOpacity="0.2" />
        <path d="M12 2a10 10 0 0 1 10 10" stroke="currentColor" strokeWidth="3" strokeLinecap="round" />
      </svg>
    </div>
  );
}
