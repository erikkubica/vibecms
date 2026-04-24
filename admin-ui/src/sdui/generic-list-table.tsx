import React from "react";
import { Link } from "react-router-dom";
import { useSearchParams } from "react-router-dom";
import { Unplug, ArrowUp, ArrowDown, ArrowUpDown } from "lucide-react";
import {
  ListCard,
  ListTable,
  ListFooter,
  Th,
  Tr,
  Td,
  Chip,
  TitleCell,
  RowActions,
  EmptyState,
  LoadingRow,
} from "@/components/ui/list-page";
import { iconMap } from "./sdui-components";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function toPascalCase(s: string): string {
  return s
    .split(/[-_]/)
    .map((p) => p.charAt(0).toUpperCase() + p.slice(1).toLowerCase())
    .join("");
}

interface Column {
  key: string;
  label: string;
  width?: number;
  align?: "left" | "right" | "center";
  sortable?: boolean;
}

// ---------------------------------------------------------------------------
// GenericListTable — SDUI table driven entirely by backend layout data
// ---------------------------------------------------------------------------

export function GenericListTable({
  columns,
  rows,
  emptyIcon,
  emptyTitle,
  emptyDesc,
  newPath,
  newLabel,
  pagination,
  label,
  hasFilters,
  sortBy,
  sortOrder,
  onRowDelete,
  onRowDetach,
}: {
  columns: Column[];
  rows?: Array<Record<string, any>>;
  emptyIcon?: string;
  emptyTitle?: string;
  emptyDesc?: string;
  newPath?: string;
  newLabel?: string;
  pagination?: {
    page: number;
    perPage: number;
    total: number;
    totalPages: number;
  };
  label?: string;
  hasFilters?: boolean;
  sortBy?: string;
  sortOrder?: string;
  onRowDelete?: (row: Record<string, any>) => void;
  onRowDetach?: (row: Record<string, any>) => void;
}) {
  const [, setSearchParams] = useSearchParams();

  const IconComp = emptyIcon ? iconMap[toPascalCase(emptyIcon)] : null;

  const handleSort = (colKey: string) => {
    setSearchParams((prev) => {
      const currentSort = prev.get("sort");
      const currentOrder = prev.get("order") || "desc";
      if (currentSort === colKey) {
        prev.set("order", currentOrder === "asc" ? "desc" : "asc");
      } else {
        prev.set("sort", colKey);
        // Date fields default desc (newest first); text fields default asc
        prev.set("order", colKey === "updated_at" || colKey === "created_at" ? "desc" : "asc");
      }
      prev.delete("page");
      return prev;
    });
  };

  // --- Loading state ---
  if (!rows) {
    return (
      <ListCard>
        <LoadingRow />
      </ListCard>
    );
  }

  // --- Empty state ---
  if (rows.length === 0) {
    return (
      <ListCard>
        <EmptyState
          icon={IconComp ?? (() => null)}
          title={
            hasFilters
              ? `No ${label ?? "items"} match your filters`
              : (emptyTitle ?? "No items found")
          }
          description={
            hasFilters
              ? "Try adjusting your filters"
              : (emptyDesc ?? "Create your first item to get started")
          }
          action={
            !hasFilters && newPath ? (
              <Link
                to={newPath}
                className="h-[30px] px-3 inline-flex items-center gap-1.5 text-[13px] font-medium text-white bg-indigo-600 rounded hover:bg-indigo-700"
              >
                + {newLabel ?? "New"}
              </Link>
            ) : undefined
          }
        />
      </ListCard>
    );
  }

  // --- Cell renderer ---
  function renderCell(
    row: Record<string, any>,
    col: Column,
  ) {
    const { key } = col;
    const val = row[key];

    switch (key) {
      case "label":
      case "name": {
        const title = row.label || row.name;
        return (
          <Td>
            <TitleCell to={row.editPath} title={title} slug={row.slug} />
          </Td>
        );
      }

      case "slug":
        return (
          <Td className="font-mono text-[12px] text-slate-500">{val}</Td>
        );

      case "blockCount":
      case "fieldCount":
      case "itemCount":
        return (
          <Td className="text-slate-500 tabular-nums" align="center">
            {val ?? 0}
          </Td>
        );

      case "sourceLabel":
      case "source":
        return (
          <Td>
            <Chip>{val || "Custom"}</Chip>
          </Td>
        );

      case "description":
        return (
          <Td className="text-slate-500">
            <span
              className="block max-w-xs truncate"
              title={val || ""}
            >
              {val || "—"}
            </span>
          </Td>
        );

      case "isDefault":
        return (
          <Td>
            {val ? (
              <span className="inline-flex items-center gap-1 px-1.5 py-px text-[11px] font-medium text-indigo-700 bg-indigo-50 border border-indigo-200 rounded-[2px]">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="10"
                  height="10"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="3"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <polyline points="20 6 9 17 4 12" />
                </svg>
                Default
              </span>
            ) : (
              <span className="text-slate-400 text-[12px]">—</span>
            )}
          </Td>
        );

      case "version":
        return (
          <Td>
            <Chip>v{val}</Chip>
          </Td>
        );

      case "langDisplay": {
        const flag = row.langFlag;
        if (flag) {
          return (
            <Td>
              <span className="inline-flex items-center gap-1.5 text-[12px] text-slate-700">
                <span>{flag}</span>
                {val}
              </span>
            </Td>
          );
        }
        return <Td className="text-slate-600">{val}</Td>;
      }

      case "updated_at":
        return (
          <Td className="font-mono text-[12px] text-slate-500 tabular-nums">
            {val || "—"}
          </Td>
        );

      case "actions":
        return (
          <Td align="right" className="whitespace-nowrap">
            <RowActions
              editTo={row.editPath}
              onDelete={
                row.isCustom !== false ? () => onRowDelete?.(row) : undefined
              }
              disableDelete={row.isCustom === false}
              deleteTitle={
                row.isCustom === false
                  ? "Built-in, cannot delete"
                  : "Delete"
              }
              extra={
                row.isCustom === false ? (
                  <button
                    type="button"
                    title="Detach from source"
                    onClick={() => onRowDetach?.(row)}
                    className="w-[26px] h-[26px] grid place-items-center text-amber-600 hover:bg-amber-50 hover:border-amber-200 border border-transparent rounded-[2px] cursor-pointer bg-transparent disabled:opacity-40"
                  >
                    <Unplug className="w-3 h-3" />
                  </button>
                ) : undefined
              }
            />
          </Td>
        );

      default:
        return (
          <Td className="text-slate-500">
            {val != null ? String(val) : "—"}
          </Td>
        );
    }
  }

  // --- Sortable header cell ---
  function SortableHeader({ col }: { col: Column }) {
    if (!col.sortable) {
      return <Th key={col.key} width={col.width} align={col.align as any}>{col.label}</Th>;
    }
    const isActive = sortBy === col.key;
    const Icon = isActive
      ? sortOrder === "asc" ? ArrowUp : ArrowDown
      : ArrowUpDown;
    return (
      <Th key={col.key} width={col.width} align={col.align as any}>
        <button
          type="button"
          onClick={() => handleSort(col.key)}
          className={`inline-flex items-center gap-1 cursor-pointer bg-transparent border-0 p-0 font-[inherit] text-[inherit] ${isActive ? "text-slate-900" : "text-slate-500 hover:text-slate-700"}`}
        >
          {col.label}
          <Icon className={`w-3 h-3 ${isActive ? "text-indigo-600" : "text-slate-400"}`} />
        </button>
      </Th>
    );
  }

  // --- Main table ---
  return (
    <ListCard>
      <ListTable>
        <thead>
          <tr>
            {columns.map((col) => (
              <SortableHeader key={col.key} col={col} />
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <Tr key={row.id}>
              {columns.map((col) => (
                <React.Fragment key={col.key}>
                  {renderCell(row, col)}
                </React.Fragment>
              ))}
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
          onPage={(p: number) => {
            setSearchParams((prev) => {
              prev.set("page", String(p));
              return prev;
            });
          }}
          onPerPage={(n: number) => {
            setSearchParams((prev) => {
              prev.set("per_page", String(n));
              prev.delete("page");
              return prev;
            });
          }}
          label={label ?? "items"}
        />
      )}
    </ListCard>
  );
}
