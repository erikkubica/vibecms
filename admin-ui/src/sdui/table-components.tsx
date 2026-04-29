import { useState, useEffect, memo } from "react";
import { useNavigate, useSearchParams, Link } from "react-router-dom";
import {
  ArrowLeft,
  ArrowDown,
  ArrowUp,
  ArrowUpDown,
  FileText,
  Globe,
  Home,
  Plus,
  Tag,
  X,
} from "lucide-react";
import {
  ListCard,
  ListTable,
  ListFooter,
  ListSearch,
  Th,
  Tr,
  Td,
  StatusPill,
  Chip,
  TitleCell,
  RowActions,
  EmptyState,
  LoadingRow,
} from "@/components/ui/list-page";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** A single content node row passed from the Go backend via SDUI layout tree. */
interface ContentNodeRow {
  id: number;
  title: string;
  slug: string;
  status: string;
  language_code?: string;
  taxonomies?: Record<string, string[]>;
  updated_at: string;
  editPath: string;
  is_homepage?: boolean;
}

/** A single taxonomy term row passed from the Go backend via SDUI layout tree. */
interface TaxonomyTermRow {
  id: number;
  name: string;
  slug: string;
  description?: string;
  count: number;
  language_code?: string;
  editPath: string;
}

/** Simple flag map for common language codes. */
const LANG_FLAGS: Record<string, string> = {
  en: "🇺🇸",
  de: "🇩🇪",
  fr: "🇫🇷",
  es: "🇪🇸",
  it: "🇮🇹",
  pt: "🇵🇹",
  nl: "🇳🇱",
  pl: "🇵🇱",
  sv: "🇸🇪",
  no: "🇳🇴",
  da: "🇩🇰",
  fi: "🇫🇮",
  cs: "🇨🇿",
  sk: "🇸🇰",
  hu: "🇭🇺",
  ro: "🇷🇴",
  bg: "🇧🇬",
  hr: "🇭🇷",
  sl: "🇸🇮",
  et: "🇪🇪",
  lv: "🇱🇻",
  lt: "🇱🇹",
  uk: "🇺🇦",
  ru: "🇷🇺",
  ja: "🇯🇵",
  ko: "🇰🇷",
  zh: "🇨🇳",
  ar: "🇸🇦",
  he: "🇮🇱",
  th: "🇹🇭",
  vi: "🇻🇳",
  id: "🇮🇩",
  ms: "🇲🇾",
  tr: "🇹🇷",
  el: "🇬🇷",
};

// ---------------------------------------------------------------------------
// PageHeader
// Page header with title, count badge, optional back button, and "New" button.
// Matches the existing ListHeader border/spacing/typography exactly.
// ---------------------------------------------------------------------------

export function PageHeader({
  newLabel,
  newPath,
  backPath,
  onNew,
  onBack,
  tabs,
  activeTab,
  tabParam = "status",
  languages,
  activeLanguage,
}: {
  title?: string;
  newLabel?: string;
  newPath?: string;
  backPath?: string;
  onNew?: () => void;
  onBack?: () => void;
  tabs?: Array<{ value: string; label: string; count?: number }>;
  activeTab?: string;
  tabParam?: string;
  languages?: Array<{ id?: number; code: string; name: string; flag: string }>;
  activeLanguage?: string;
}) {
  const navigate = useNavigate();
  const [, setSearchParams] = useSearchParams();

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
                onClick={() => {
                  setSearchParams((prev) => {
                    if (t.value === "all") prev.delete(tabParam);
                    else prev.set(tabParam, t.value);
                    prev.delete("page");
                    return prev;
                  });
                }}
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
        <div className="flex-1" />
      )}
      <div className="flex gap-1.5 pb-1.5">
        {(backPath || onBack) &&
          (backPath ? (
            <button
              type="button"
              onClick={() => (onBack ? onBack() : navigate(backPath))}
              className="h-[26px] px-2 inline-flex items-center gap-1.5 text-[12px] font-medium text-slate-700 bg-white border border-slate-300 rounded hover:bg-slate-50 cursor-pointer"
            >
              <ArrowLeft className="w-3 h-3" />
              Back
            </button>
          ) : (
            <button
              type="button"
              onClick={onBack}
              className="h-[26px] px-2 inline-flex items-center gap-1.5 text-[12px] font-medium text-slate-700 bg-white border border-slate-300 rounded hover:bg-slate-50 cursor-pointer"
            >
              <ArrowLeft className="w-3 h-3" />
              Back
            </button>
          ))}
        {(newPath || onNew) &&
          (newPath ? (
            <button
              type="button"
              onClick={() => navigate(newPath)}
              className="h-[26px] px-2.5 inline-flex items-center gap-1.5 text-[12px] font-medium text-white bg-indigo-600 border border-indigo-600 rounded hover:bg-indigo-700 cursor-pointer"
            >
              <Plus className="w-3 h-3" />
              {newLabel ?? "New"}
            </button>
          ) : (
            <button
              type="button"
              onClick={onNew}
              className="h-[26px] px-2.5 inline-flex items-center gap-1.5 text-[12px] font-medium text-white bg-indigo-600 border border-indigo-600 rounded hover:bg-indigo-700 cursor-pointer"
            >
              <Plus className="w-3 h-3" />
              {newLabel ?? "New"}
            </button>
          ))}
        {languages && languages.length > 0 && (
          <Select
            value={activeLanguage || "all"}
            onValueChange={(val) => {
              setSearchParams((prev) => {
                if (val === "all") prev.delete("language");
                else prev.set("language", val);
                prev.delete("page");
                return prev;
              });
            }}
          >
            <SelectTrigger size="sm" className="w-[160px]">
              <Globe className="w-3.5 h-3.5 text-slate-400 mr-1" />
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All languages</SelectItem>
              {languages.map((lang) => (
                <SelectItem
                  key={lang.code}
                  value={lang.id != null ? String(lang.id) : lang.code}
                >
                  {lang.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// SearchToolbar
// Search input matching the existing ListSearch style.
// ---------------------------------------------------------------------------

export const SearchToolbar = memo(
  function SearchToolbar({
    searchPlaceholder,
    value,
    onChange,
    languages,
    activeLanguage,
  }: {
    searchPlaceholder?: string;
    value?: string;
    onChange?: (v: string) => void;
    languages?: Array<{ id?: number; code: string; name: string; flag: string }>;
    activeLanguage?: string;
  }) {
    const [searchParams, setSearchParams] = useSearchParams();

    // Read initial state from URL params (source of truth) so remounts don't lose state
    const [localSearch, setLocalSearch] = useState(
      () => searchParams.get("search") || value || "",
    );

    useEffect(() => {
      const timer = setTimeout(() => {
        setSearchParams((prev) => {
          // Read current URL value inside setter to avoid stale closure
          const current = prev.get("search") || "";
          if (localSearch === current) return prev; // no-op if already in sync
          if (localSearch) prev.set("search", localSearch);
          else prev.delete("search");
          prev.delete("page");
          return prev;
        });
      }, 300);
      return () => clearTimeout(timer);
    }, [localSearch]);

    return (
      <div className="flex items-center gap-2 mb-2.5 flex-wrap">
        <ListSearch
          value={localSearch}
          onChange={(v) => {
            setLocalSearch(v);
            onChange?.(v);
          }}
          placeholder={searchPlaceholder ?? "Search…"}
        />
        {languages && languages.length > 0 && (
          <Select
            value={activeLanguage || "all"}
            onValueChange={(val) => {
              setSearchParams((prev) => {
                if (val === "all") prev.delete("language");
                else prev.set("language", val);
                prev.delete("page");
                return prev;
              });
            }}
          >
            <SelectTrigger className="w-[160px]">
              <Globe className="w-3.5 h-3.5 text-slate-400 mr-1" />
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All languages</SelectItem>
              {languages.map((lang) => (
                <SelectItem
                  key={lang.code}
                  value={lang.id != null ? String(lang.id) : lang.code}
                >
                  {lang.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
      </div>
    );
  },
  function areEqual(prev, next) {
    // Only re-render when data props actually changed.
    // Ignore callback refs (onChange) that get new identity every render.
    return (
      prev.searchPlaceholder === next.searchPlaceholder &&
      prev.value === next.value &&
      prev.activeLanguage === next.activeLanguage &&
      prev.languages?.length === next.languages?.length &&
      (!prev.languages ||
        !next.languages ||
        prev.languages.every((l, i) => l.code === next.languages![i].code && l.id === next.languages![i].id))
    );
  },
);

// ---------------------------------------------------------------------------
// TaxonomyFilterChips
// Renders active taxonomy filter chips matching the legacy design.
// When content is filtered by taxonomy (via URL params like ?category=Vietnam),
// show removable chips.
// ---------------------------------------------------------------------------

export function TaxonomyFilterChips({
  filters,
}: {
  filters: Array<{ taxonomy: string; term: string; label: string }>;
}) {
  const [, setSearchParams] = useSearchParams();

  if (!filters || filters.length === 0) return null;

  const removeFilter = (taxonomy: string) => {
    setSearchParams((prev) => {
      prev.delete(taxonomy);
      return prev;
    });
  };

  return (
    <div className="flex flex-wrap gap-1.5 mb-2.5">
      {filters.map((f) => (
        <span
          key={f.taxonomy}
          className="inline-flex items-center gap-1.5 px-2 py-0.5 text-[11px] font-medium text-indigo-700 bg-indigo-50 border border-indigo-200 rounded"
        >
          <Tag className="w-3 h-3" />
          {f.label}: <strong>{f.term}</strong>
          <button
            type="button"
            onClick={() => removeFilter(f.taxonomy)}
            className="hover:text-red-500 cursor-pointer bg-transparent border-0"
          >
            <X className="w-3 h-3" />
          </button>
        </span>
      ))}
      <button
        type="button"
        onClick={() => setSearchParams(new URLSearchParams())}
        className="text-[11px] text-slate-500 hover:text-slate-700 cursor-pointer bg-transparent border-0"
      >
        Clear all
      </button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// ContentNodeTable
// Table displaying content nodes with columns: Title, Status, Taxonomies,
// Language, Updated, Actions. Matches the existing NodesListPage exactly.
//
// Data comes from the Go backend via the SDUI layout tree — no client-side
// fetching. Rows and pagination are passed as props.
// ---------------------------------------------------------------------------

export function ContentNodeTable({
  nodeType,
  rows,
  pagination,
  taxonomyDefs: _taxonomyDefs,
  onRowDelete,
  hasActiveFilters,
  nodeTypeLabel,
  nodeTypeLabelPlural,
  basePath,
  sortBy,
  sortOrder,
}: {
  nodeType?: string;
  rows?: ContentNodeRow[];
  pagination?: {
    page: number;
    perPage: number;
    total: number;
    totalPages: number;
  };
  taxonomyDefs?: Array<{ slug: string; label: string }>;
  onRowDelete?: (row: ContentNodeRow) => void;
  hasActiveFilters?: boolean;
  nodeTypeLabel?: string;
  nodeTypeLabelPlural?: string;
  basePath?: string;
  sortBy?: string;
  sortOrder?: string;
}) {
  const [, setSearchParams] = useSearchParams();

  const handleSort = (colKey: string) => {
    setSearchParams((prev) => {
      const currentSort = prev.get("sort");
      const currentOrder = prev.get("order") || "desc";
      if (currentSort === colKey) {
        prev.set("order", currentOrder === "asc" ? "desc" : "asc");
      } else {
        prev.set("sort", colKey);
        prev.set("order", colKey === "title" ? "asc" : "desc");
      }
      prev.delete("page");
      return prev;
    });
  };

  if (!rows) {
    return (
      <ListCard>
        <LoadingRow />
      </ListCard>
    );
  }

  if (rows.length === 0) {
    const label = nodeTypeLabelPlural || nodeTypeLabel || nodeType || "items";
    const singular = nodeTypeLabel || "item";
    return (
      <ListCard>
        <EmptyState
          icon={FileText}
          title={
            hasActiveFilters
              ? `No ${label.toLowerCase()} match your filters`
              : `No ${label.toLowerCase()} yet`
          }
          description={
            hasActiveFilters
              ? "Try adjusting your filters"
              : `Create your first ${singular.toLowerCase()} to get started`
          }
          action={
            !hasActiveFilters && basePath ? (
              <Link
                to={`${basePath}/new`}
                className="h-[30px] px-3 inline-flex items-center gap-1.5 text-[13px] font-medium text-white bg-indigo-600 rounded hover:bg-indigo-700"
              >
                <Plus className="w-3.5 h-3.5" />
                New {singular}
              </Link>
            ) : undefined
          }
        />
      </ListCard>
    );
  }

  return (
    <ListCard>
      <ListTable>
        <thead>
          <tr>
            <Th>Title</Th>
            <Th width={120}>Status</Th>
            <Th width={240}>Taxonomies</Th>
            <Th width={80}>Lang</Th>
            <Th width={110}>
              <button
                type="button"
                onClick={() => handleSort("updated_at")}
                className={`inline-flex items-center gap-1 cursor-pointer bg-transparent border-0 p-0 font-[inherit] text-[inherit] ${sortBy === "updated_at" ? "text-slate-900" : "text-slate-500 hover:text-slate-700"}`}
              >
                Updated
                {sortBy === "updated_at" ? (
                  sortOrder === "asc" ? <ArrowUp className="w-2.5 h-2.5 text-indigo-600" /> : <ArrowDown className="w-2.5 h-2.5 text-indigo-600" />
                ) : (
                  <ArrowUpDown className="w-2.5 h-2.5 text-slate-400" />
                )}
              </button>
            </Th>
            <Th width={110} align="right">
              Actions
            </Th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => {
            const flag = row.language_code
              ? (LANG_FLAGS[row.language_code.toLowerCase()] ?? "")
              : "";
            return (
              <Tr key={row.id}>
                <Td>
                  <TitleCell
                    to={row.editPath}
                    title={row.title}
                    slug={row.slug}
                    extra={
                      row.is_homepage ? (
                        <span className="inline-flex items-center gap-1 px-1.5 py-px text-[10px] font-medium text-emerald-700 bg-emerald-50 border border-emerald-200 rounded-[2px]">
                          <Home className="w-2.5 h-2.5" />
                          Home
                        </span>
                      ) : undefined
                    }
                  />
                </Td>
                <Td>
                  <StatusPill status={row.status} />
                </Td>
                <Td>
                  {Object.entries(row.taxonomies || {}).length === 0 ? (
                    <span className="text-slate-400 text-[12px]">—</span>
                  ) : (
                    <div className="flex gap-1 flex-wrap">
                      {Object.entries(row.taxonomies || {}).flatMap(
                        ([tax, terms]) =>
                          (terms as string[]).map((term) => (
                            <Chip key={`${tax}-${term}`}>{term}</Chip>
                          )),
                      )}
                    </div>
                  )}
                </Td>
                <Td>
                  {row.language_code ? (
                    <span
                      className="inline-flex items-center gap-1.5 text-[12px] text-slate-700"
                      title={row.language_code}
                    >
                      {flag && <span>{flag}</span>}
                      {row.language_code.toUpperCase()}
                    </span>
                  ) : (
                    <span className="text-slate-400 text-[12px]">—</span>
                  )}
                </Td>
                <Td className="font-mono text-[12px] text-slate-500 tabular-nums">
                  {new Date(row.updated_at).toLocaleDateString()}
                </Td>
                <Td align="right" className="whitespace-nowrap">
                  <RowActions
                    editTo={row.editPath}
                    onDelete={onRowDelete ? () => onRowDelete(row) : undefined}
                  />
                </Td>
              </Tr>
            );
          })}
        </tbody>
      </ListTable>
      {pagination && (
        <ListFooter
          page={pagination.page}
          totalPages={pagination.totalPages}
          total={pagination.total}
          perPage={pagination.perPage}
          onPage={(p) => {
            setSearchParams((prev) => {
              prev.set("page", String(p));
              return prev;
            });
          }}
          onPerPage={(n) => {
            setSearchParams((prev) => {
              prev.set("per_page", String(n));
              prev.delete("page");
              return prev;
            });
          }}
          label={(nodeTypeLabelPlural || nodeType || "items").toLowerCase()}
        />
      )}
    </ListCard>
  );
}

// ---------------------------------------------------------------------------
// TaxonomyTermsTable
// Table displaying taxonomy terms with columns: Name, Slug, Count, Actions.
// Matches the existing TaxonomyTermsPage exactly.
//
// Data comes from the Go backend via the SDUI layout tree — no client-side
// fetching. Rows are passed as props.
// ---------------------------------------------------------------------------

export function TaxonomyTermsTable({
  taxonomy,
  nodeType,
  rows,
  onRowDelete,
  hasActiveFilters,
  taxonomyLabel,
  taxonomyLabelPlural,
  basePath: basePathProp,
  sortBy,
  sortOrder,
  pagination,
}: {
  taxonomy?: string;
  nodeType?: string;
  rows?: TaxonomyTermRow[];
  onRowDelete?: (row: TaxonomyTermRow) => void;
  hasActiveFilters?: boolean;
  taxonomyLabel?: string;
  taxonomyLabelPlural?: string;
  basePath?: string;
  sortBy?: string;
  sortOrder?: string;
  pagination?: { page: number; perPage: number; total: number; totalPages: number };
}) {
  const [, setSearchParams] = useSearchParams();

  const handleSort = (colKey: string) => {
    setSearchParams((prev) => {
      const currentSort = prev.get("sort");
      const currentOrder = prev.get("order") || "asc";
      if (currentSort === colKey) {
        prev.set("order", currentOrder === "asc" ? "desc" : "asc");
      } else {
        prev.set("sort", colKey);
        prev.set("order", colKey === "count" ? "desc" : "asc");
      }
      prev.delete("page");
      return prev;
    });
  };
  if (!rows) {
    return (
      <ListCard>
        <LoadingRow />
      </ListCard>
    );
  }

  if (rows.length === 0) {
    const label = taxonomyLabelPlural || taxonomyLabel || taxonomy || "terms";
    const singular = taxonomyLabel || "term";
    return (
      <ListCard>
        <EmptyState
          icon={Tag}
          title={
            hasActiveFilters
              ? `No ${label.toLowerCase()} match your filters`
              : `No ${label.toLowerCase()} yet`
          }
          description={
            hasActiveFilters
              ? "Try adjusting your filters"
              : `Create your first ${singular.toLowerCase()} to get started`
          }
          action={
            !hasActiveFilters && basePathProp ? (
              <Link
                to={`${basePathProp}/new`}
                className="h-[30px] px-3 inline-flex items-center gap-1.5 text-[13px] font-medium text-white bg-indigo-600 rounded hover:bg-indigo-700"
              >
                <Plus className="w-3.5 h-3.5" />
                New {singular}
              </Link>
            ) : undefined
          }
        />
      </ListCard>
    );
  }

  const contentListPath = nodeType
    ? nodeType === "page"
      ? "/admin/pages"
      : nodeType === "post"
        ? "/admin/posts"
        : `/admin/content/${nodeType}`
    : "/admin/content/page";

  const nameActive = sortBy === "name";
  const countActive = sortBy === "count";

  return (
    <ListCard>
      <ListTable minWidth={640}>
        <thead>
          <tr>
            <Th>
              <button
                type="button"
                onClick={() => handleSort("name")}
                className={`inline-flex items-center gap-1 cursor-pointer bg-transparent border-0 p-0 font-[inherit] text-[inherit] ${nameActive ? "text-slate-900" : "text-slate-500 hover:text-slate-700"}`}
              >
                Name
                {nameActive ? (
                  sortOrder === "asc" ? <ArrowUp className="w-2.5 h-2.5 text-indigo-600" /> : <ArrowDown className="w-2.5 h-2.5 text-indigo-600" />
                ) : (
                  <ArrowUpDown className="w-2.5 h-2.5 text-slate-400" />
                )}
              </button>
            </Th>
            <Th width={200}>Slug</Th>
            <Th width={70}>Lang</Th>
            <Th width={80} align="center">
              <button
                type="button"
                onClick={() => handleSort("count")}
                className={`inline-flex items-center gap-1 cursor-pointer bg-transparent border-0 p-0 font-[inherit] text-[inherit] ${countActive ? "text-slate-900" : "text-slate-500 hover:text-slate-700"}`}
              >
                Count
                {countActive ? (
                  sortOrder === "asc" ? <ArrowUp className="w-2.5 h-2.5 text-indigo-600" /> : <ArrowDown className="w-2.5 h-2.5 text-indigo-600" />
                ) : (
                  <ArrowUpDown className="w-2.5 h-2.5 text-slate-400" />
                )}
              </button>
            </Th>
            <Th width={110} align="right">
              Actions
            </Th>
          </tr>
        </thead>
        <tbody>
          {rows.map((term) => (
            <Tr key={term.id}>
              <Td>
                <TitleCell to={term.editPath} title={term.name} />
                {term.description && (
                  <p className="text-[11px] text-slate-500 line-clamp-1 mt-0.5">
                    {term.description}
                  </p>
                )}
              </Td>
              <Td className="font-mono text-[12px] text-slate-500">
                {term.slug}
              </Td>
              <Td className="text-[12px]">
                {term.language_code ? (
                  <span className="inline-flex items-center gap-1">
                    {LANG_FLAGS[term.language_code] && (
                      <span aria-hidden>{LANG_FLAGS[term.language_code]}</span>
                    )}
                    <span className="font-medium uppercase text-slate-600">
                      {term.language_code}
                    </span>
                  </span>
                ) : (
                  <span className="text-slate-400">—</span>
                )}
              </Td>
              <Td align="center">
                <a
                  href={`${contentListPath}?${taxonomy ?? "term"}=${encodeURIComponent(term.name)}`}
                  className="inline-flex h-[22px] min-w-[24px] items-center justify-center rounded-full bg-slate-100 px-2 text-[11px] font-medium text-slate-600 hover:bg-indigo-100 hover:text-indigo-700 transition-colors"
                >
                  {term.count}
                </a>
              </Td>
              <Td align="right" className="whitespace-nowrap">
                <RowActions
                  editTo={term.editPath}
                  onDelete={onRowDelete ? () => onRowDelete(term) : undefined}
                />
              </Td>
            </Tr>
          ))}
        </tbody>
      </ListTable>
      {pagination && (
        <ListFooter
          page={pagination.page}
          totalPages={pagination.totalPages}
          total={pagination.total}
          perPage={pagination.perPage}
          onPage={(p) => {
            setSearchParams((prev) => {
              prev.set("page", String(p));
              return prev;
            });
          }}
          onPerPage={(n) => {
            setSearchParams((prev) => {
              prev.set("per_page", String(n));
              prev.delete("page");
              return prev;
            });
          }}
          label="terms"
        />
      )}
    </ListCard>
  );
}
