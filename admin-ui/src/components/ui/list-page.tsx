import type { ReactNode, MouseEvent as ReactMouseEvent } from "react";
import { Link } from "react-router-dom";
import { Plus, Pencil, Trash2, Eye, ExternalLink } from "lucide-react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

export function ListPageShell({ children }: { children: ReactNode }) {
  return <div className="w-full" style={{ paddingBottom: 32 }}>{children}</div>;
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
  leading?: ReactNode;
}

export function ListHeader({ count, tabs, activeTab, onTabChange, newLabel, newHref, onNew, extra, leading }: ListHeaderProps) {
  return (
    <div
      className="flex items-center"
      style={{
        gap: 0,
        borderBottom: "1px solid var(--divider)",
        marginBottom: 12,
      }}
    >
      {leading && <div className="flex items-center" style={{ paddingBottom: 6, paddingRight: 6 }}>{leading}</div>}
      {tabs && tabs.length > 0 ? (
        <nav className="flex-1 flex items-center" style={{ gap: 0, marginBottom: -1 }}>
          {tabs.map((t) => {
            const active = t.value === activeTab;
            return (
              <button
                key={t.value}
                type="button"
                onClick={() => onTabChange?.(t.value)}
                className="inline-flex items-center"
                style={{
                  padding: "12px 14px",
                  fontSize: 12.5,
                  fontWeight: active ? 600 : 500,
                  color: active ? "var(--fg)" : "var(--fg-muted)",
                  borderBottom: `1.5px solid ${active ? "var(--fg)" : "transparent"}`,
                  background: "transparent",
                  cursor: "pointer",
                  transition: "color 0.12s, border-color 0.12s",
                  letterSpacing: "-0.005em",
                  gap: 6,
                  marginBottom: -1,
                }}
              >
                {t.label}
                {t.count !== undefined && (
                  <span
                    style={{
                      fontFamily: "var(--font-mono)",
                      fontSize: 10,
                      fontWeight: 500,
                      padding: "1px 5px",
                      borderRadius: 8,
                      background: active ? "var(--accent-mid)" : "var(--sub-bg)",
                      color: active ? "var(--accent-strong)" : "var(--fg-muted)",
                    }}
                  >
                    {t.count}
                  </span>
                )}
              </button>
            );
          })}
        </nav>
      ) : (
        <div className="flex-1" style={{ paddingBottom: 10 }}>
          {count !== undefined && (
            <span style={{ fontFamily: "var(--font-mono)", fontSize: 11.5, fontWeight: 500, color: "var(--fg-muted)" }}>
              {count} items
            </span>
          )}
        </div>
      )}
      <div className="flex" style={{ gap: 6, paddingBottom: 6 }}>
        {extra}
        {(newHref || onNew) && (
          newHref ? (
            <Link
              to={newHref}
              className="inline-flex items-center"
              style={{
                height: 28,
                padding: "0 10px",
                fontSize: 12,
                fontWeight: 500,
                color: "var(--accent-fg)",
                background: "var(--accent)",
                borderRadius: "var(--radius-md)",
                gap: 6,
                letterSpacing: "-0.005em",
                boxShadow: "0 1px 0 rgba(255,255,255,0.18) inset, 0 1px 2px rgba(20,18,15,0.18)",
              }}
            >
              <Plus className="w-3 h-3" />
              {newLabel ?? "New"}
            </Link>
          ) : (
            <button
              type="button"
              onClick={onNew}
              className="inline-flex items-center cursor-pointer"
              style={{
                height: 28,
                padding: "0 10px",
                fontSize: 12,
                fontWeight: 500,
                color: "var(--accent-fg)",
                background: "var(--accent)",
                borderRadius: "var(--radius-md)",
                gap: 6,
                letterSpacing: "-0.005em",
                border: "none",
                boxShadow: "0 1px 0 rgba(255,255,255,0.18) inset, 0 1px 2px rgba(20,18,15,0.18)",
              }}
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
  return <div className="flex items-center flex-wrap" style={{ gap: 8, marginBottom: 10 }}>{children}</div>;
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
    <div className="flex-1 relative" style={{ maxWidth: 440 }}>
      <svg
        className="absolute"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
        style={{
          left: 10,
          top: "50%",
          transform: "translateY(-50%)",
          width: 14,
          height: 14,
          color: "var(--fg-subtle)",
          pointerEvents: "none",
        }}
      >
        <circle cx="11" cy="11" r="8" />
        <path d="m21 21-4.35-4.35" />
      </svg>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="w-full"
        style={{
          height: 30,
          padding: "0 11px 0 30px",
          background: "var(--card-bg)",
          border: "1px solid var(--border-input)",
          borderRadius: "var(--radius-md)",
          fontSize: 13,
          color: "var(--fg)",
          outline: "none",
          letterSpacing: "-0.005em",
          boxShadow: "0 1px 1px rgba(20,18,15,0.02) inset",
        }}
      />
    </div>
  );
}

export function ListCard({ children }: { children: ReactNode }) {
  return (
    <div
      data-slot="list-card"
      style={{
        background: "var(--card-bg)",
        borderRadius: "var(--radius-lg)",
        boxShadow: "var(--shadow-card)",
        overflow: "hidden",
      }}
    >
      {children}
    </div>
  );
}

export function ListTable({ children, minWidth = 880 }: { children: ReactNode; minWidth?: number }) {
  return (
    <div style={{ overflowX: "auto" }}>
      <table
        className="w-full"
        style={{ borderCollapse: "separate", borderSpacing: 0, minWidth: `${minWidth}px` }}
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
  style: extraStyle,
}: {
  children?: ReactNode;
  width?: string | number;
  align?: "left" | "right" | "center";
  className?: string;
  style?: React.CSSProperties;
}) {
  const style: React.CSSProperties = {
    textAlign: align,
    padding: "9px 14px",
    background: "var(--sub-bg)",
    borderBottom: "1px solid var(--divider)",
    fontFamily: "var(--font-mono)",
    fontSize: 10.5,
    fontWeight: 500,
    textTransform: "uppercase",
    letterSpacing: "0.07em",
    color: "var(--fg-subtle)",
    whiteSpace: "nowrap",
    userSelect: "none",
    ...extraStyle,
  };
  if (width) style.width = typeof width === "number" ? `${width}px` : width;
  return (
    <th style={style} className={className}>
      {children}
    </th>
  );
}

export function Tr({
  children,
  className = "",
  onClick,
  style: extraStyle,
}: {
  children: ReactNode;
  className?: string;
  onClick?: (e: ReactMouseEvent<HTMLTableRowElement>) => void;
  style?: React.CSSProperties;
}) {
  return (
    <tr
      className={`group ${className}`}
      onClick={onClick}
      style={{ background: "var(--card-bg)", cursor: onClick ? "pointer" : undefined, transition: "background 80ms ease", ...extraStyle }}
      onMouseEnter={(e) => (e.currentTarget.style.background = "var(--sub-bg)")}
      onMouseLeave={(e) => (e.currentTarget.style.background = (extraStyle?.background as string) || "var(--card-bg)")}
    >
      {children}
    </tr>
  );
}

export function Td({
  children,
  className = "",
  align = "left",
  onClick,
  style: extraStyle,
  colSpan,
}: {
  children?: ReactNode;
  className?: string;
  align?: "left" | "right" | "center";
  onClick?: (e: ReactMouseEvent<HTMLTableCellElement>) => void;
  style?: React.CSSProperties;
  colSpan?: number;
}) {
  return (
    <td
      onClick={onClick}
      colSpan={colSpan}
      className={`group-last:border-0 ${className}`}
      style={{
        padding: "11px 14px",
        borderBottom: "1px solid var(--divider)",
        fontSize: 13,
        color: "var(--fg-2)",
        textAlign: align,
        verticalAlign: "middle",
        ...extraStyle,
      }}
    >
      {children}
    </td>
  );
}

type StatusKind = "published" | "draft" | "archived" | "active" | "inactive" | "neutral" | "success" | "warning" | "danger";

export function StatusPill({ status, label }: { status: StatusKind | string; label?: string }) {
  type Palette = { color: string; bg: string; dot: string };
  const palette: Record<string, Palette> = {
    published: { color: "var(--success)", bg: "var(--success-bg)", dot: "var(--success)" },
    active:    { color: "var(--success)", bg: "var(--success-bg)", dot: "var(--success)" },
    success:   { color: "var(--success)", bg: "var(--success-bg)", dot: "var(--success)" },
    draft:     { color: "var(--fg-muted)", bg: "var(--sub-bg)", dot: "var(--fg-subtle)" },
    inactive:  { color: "var(--fg-muted)", bg: "var(--sub-bg)", dot: "var(--fg-subtle)" },
    neutral:   { color: "var(--fg-muted)", bg: "var(--sub-bg)", dot: "var(--fg-subtle)" },
    archived:  { color: "var(--warning)", bg: "var(--warning-bg)", dot: "var(--warning)" },
    warning:   { color: "var(--warning)", bg: "var(--warning-bg)", dot: "var(--warning)" },
    danger:    { color: "var(--danger)", bg: "var(--danger-bg)", dot: "var(--danger)" },
  };
  const p = palette[status] ?? palette.neutral;
  return (
    <span
      className="inline-flex items-center"
      style={{
        gap: 5,
        padding: "2.5px 8px",
        borderRadius: 11,
        fontSize: 11,
        fontWeight: 500,
        letterSpacing: "-0.003em",
        color: p.color,
        background: p.bg,
      }}
    >
      <span
        style={{
          width: 6,
          height: 6,
          borderRadius: "50%",
          background: p.dot,
          boxShadow:
            p.dot === "var(--success)"
              ? `0 0 0 2px color-mix(in oklab, ${p.dot} 22%, transparent)`
              : undefined,
        }}
      />
      {label ?? status}
    </span>
  );
}

export function Chip({ children }: { children: ReactNode }) {
  return (
    <span
      className="inline-flex items-center"
      style={{
        padding: "1.5px 6px",
        fontSize: 10,
        fontWeight: 500,
        fontFamily: "var(--font-mono)",
        color: "var(--fg-muted)",
        background: "var(--sub-bg)",
        borderRadius: 3,
        textTransform: "lowercase",
        letterSpacing: "0.02em",
        whiteSpace: "nowrap",
      }}
    >
      {children}
    </span>
  );
}

export function SlugLink({ slug, href }: { slug: string; href?: string }) {
  const content = (
    <>
      /{slug}
      {href && <ExternalLink style={{ width: 10, height: 10, opacity: 0.7 }} aria-hidden />}
    </>
  );
  const style: React.CSSProperties = {
    fontFamily: "var(--font-mono)",
    fontSize: 11,
    color: href ? "var(--accent-strong)" : "var(--fg-subtle)",
    display: "inline-flex",
    alignItems: "center",
    gap: 4,
    marginTop: 2,
  };
  if (href) {
    return (
      <a href={href} target="_blank" rel="noreferrer" style={style} className="hover:underline">
        {content}
      </a>
    );
  }
  return <span style={style}>{content}</span>;
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
      <div className="flex items-center" style={{ gap: 6 }}>
        {to ? (
          <Link
            to={to}
            className="truncate"
            style={{
              fontSize: 13,
              fontWeight: 500,
              color: "var(--fg)",
              letterSpacing: "-0.005em",
              transition: "color 100ms",
            }}
            onMouseEnter={(e) => (e.currentTarget.style.color = "var(--accent-strong)")}
            onMouseLeave={(e) => (e.currentTarget.style.color = "var(--fg)")}
          >
            {title}
          </Link>
        ) : (
          <span className="truncate" style={{ fontSize: 13, fontWeight: 500, color: "var(--fg)" }}>
            {title}
          </span>
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
  const iconBtnStyle: React.CSSProperties = {
    width: 26,
    height: 26,
    display: "grid",
    placeItems: "center",
    color: "var(--fg-subtle)",
    background: "transparent",
    border: "none",
    borderRadius: 5,
    cursor: "pointer",
    transition: "background 0.1s, color 0.1s",
  };
  const onHover = (e: React.MouseEvent<HTMLElement>) => {
    e.currentTarget.style.background = "var(--hover-bg)";
    e.currentTarget.style.color = "var(--fg)";
  };
  const onLeave = (e: React.MouseEvent<HTMLElement>) => {
    e.currentTarget.style.background = "transparent";
    e.currentTarget.style.color = "var(--fg-subtle)";
  };
  const onHoverDel = (e: React.MouseEvent<HTMLElement>) => {
    e.currentTarget.style.background = "var(--danger-bg)";
    e.currentTarget.style.color = "var(--danger)";
  };
  const onLeaveDel = (e: React.MouseEvent<HTMLElement>) => {
    e.currentTarget.style.background = "transparent";
    e.currentTarget.style.color = "var(--fg-subtle)";
  };
  return (
    <div className="inline-flex group-hover:opacity-100 transition-opacity" style={{ gap: 1, opacity: 0.55 }}>
      {previewHref && (
        <a title="Preview" href={previewHref} target="_blank" rel="noreferrer" style={iconBtnStyle} onMouseEnter={onHover} onMouseLeave={onLeave}>
          <Eye style={{ width: 12, height: 12 }} />
        </a>
      )}
      {extra}
      {editTo && (
        <Link title="Edit" to={editTo} style={iconBtnStyle} onMouseEnter={onHover} onMouseLeave={onLeave}>
          <Pencil style={{ width: 12, height: 12 }} />
        </Link>
      )}
      {onEdit && !editTo && (
        <button title="Edit" type="button" onClick={onEdit} style={iconBtnStyle} onMouseEnter={onHover} onMouseLeave={onLeave}>
          <Pencil style={{ width: 12, height: 12 }} />
        </button>
      )}
      {onDelete && (
        <button
          title={deleteTitle ?? "Delete"}
          type="button"
          onClick={onDelete}
          disabled={disableDelete}
          style={{ ...iconBtnStyle, opacity: disableDelete ? 0.4 : 1, cursor: disableDelete ? "not-allowed" : "pointer" }}
          onMouseEnter={(e) => !disableDelete && onHoverDel(e)}
          onMouseLeave={(e) => !disableDelete && onLeaveDel(e)}
        >
          <Trash2 style={{ width: 12, height: 12 }} />
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
    <div
      className="flex items-center justify-between"
      style={{
        padding: "10px 14px",
        borderTop: "1px solid var(--divider)",
        background: "transparent",
      }}
    >
      <div className="flex items-center" style={{ gap: 12, fontSize: 12, color: "var(--fg-muted)" }}>
        <span>
          Showing <span style={{ color: "var(--fg)", fontWeight: 500 }}>{start}–{end}</span> of{" "}
          <span style={{ color: "var(--fg)", fontWeight: 500 }}>{total}</span> {label}
        </span>
        {onPerPage && (
          <div className="flex items-center" style={{ gap: 6 }}>
            <span style={{ color: "var(--fg-subtle)" }}>Per page</span>
            <Select value={String(perPage)} onValueChange={(v) => onPerPage(Number(v))}>
              <SelectTrigger size="sm" className="w-[68px]" style={{ height: 26, fontSize: 12 }}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PER_PAGE_OPTIONS.map((n) => (
                  <SelectItem key={n} value={String(n)}>{n}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
      </div>
      <div className="flex items-center" style={{ gap: 6 }}>
        <button
          type="button"
          disabled={page <= 1}
          onClick={() => onPage(page - 1)}
          className="inline-flex items-center cursor-pointer"
          style={{
            height: 26,
            padding: "0 8px",
            fontSize: 12,
            color: "var(--fg-2)",
            background: "transparent",
            border: "none",
            borderRadius: 5,
            opacity: page <= 1 ? 0.45 : 1,
            cursor: page <= 1 ? "not-allowed" : "pointer",
          }}
        >
          Prev
        </button>
        <div
          className="inline-flex items-center"
          style={{
            gap: 2,
            padding: 2,
            background: "var(--card-bg)",
            border: "1px solid var(--border)",
            borderRadius: 5,
          }}
        >
          {pages.map((p) => (
            <button
              key={p}
              type="button"
              onClick={() => onPage(p)}
              style={{
                minWidth: 22,
                height: 22,
                padding: "0 6px",
                fontSize: 12,
                fontFamily: "var(--font-mono)",
                borderRadius: 3,
                border: "none",
                cursor: "pointer",
                fontWeight: p === page ? 600 : 500,
                color: p === page ? "var(--accent-fg)" : "var(--fg-2)",
                background: p === page ? "var(--accent)" : "transparent",
              }}
              onMouseEnter={(e) => {
                if (p !== page) e.currentTarget.style.background = "var(--hover-bg)";
              }}
              onMouseLeave={(e) => {
                if (p !== page) e.currentTarget.style.background = "transparent";
              }}
            >
              {p}
            </button>
          ))}
        </div>
        <button
          type="button"
          disabled={page >= totalPages}
          onClick={() => onPage(page + 1)}
          className="inline-flex items-center cursor-pointer"
          style={{
            height: 26,
            padding: "0 8px",
            fontSize: 12,
            color: "var(--fg-2)",
            background: "transparent",
            border: "none",
            borderRadius: 5,
            opacity: page >= totalPages ? 0.45 : 1,
            cursor: page >= totalPages ? "not-allowed" : "pointer",
          }}
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
  icon: React.ComponentType<{ className?: string; style?: React.CSSProperties }>;
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div
      className="flex flex-col items-center justify-center"
      style={{
        gap: 8,
        padding: "60px 0",
        background: "transparent",
        color: "var(--fg-subtle)",
      }}
    >
      <Icon style={{ width: 28, height: 28, color: "var(--fg-subtle)" }} />
      <p style={{ fontSize: 13, fontWeight: 500, color: "var(--fg-muted)", margin: 0 }}>{title}</p>
      {description && (
        <p style={{ fontSize: 12, color: "var(--fg-subtle)", margin: 0 }}>{description}</p>
      )}
      {action && <div style={{ marginTop: 8 }}>{action}</div>}
    </div>
  );
}

export function LoadingRow() {
  return (
    <div className="flex items-center justify-center" style={{ height: 256 }}>
      <svg
        className="animate-spin"
        viewBox="0 0 24 24"
        fill="none"
        style={{ width: 24, height: 24, color: "var(--accent)" }}
      >
        <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="3" strokeOpacity="0.2" />
        <path d="M12 2a10 10 0 0 1 10 10" stroke="currentColor" strokeWidth="3" strokeLinecap="round" />
      </svg>
    </div>
  );
}
