import React from "react";
import { Trash2, Archive, CheckCircle, Circle } from "@vibecms/icons";

const { Button } = (window as any).__VIBECMS_SHARED__.ui;
const { toast } = (window as any).__VIBECMS_SHARED__.Sonner;

type BulkAction = "mark_read" | "mark_unread" | "archive" | "delete";

interface BulkActionsBarProps {
  selectedIds: Set<number>;
  onClearSelection: () => void;
  onBulkComplete: () => void;
}

export default function BulkActionsBar({
  selectedIds,
  onClearSelection,
  onBulkComplete,
}: BulkActionsBarProps) {
  const count = selectedIds.size;

  async function executeBulk(action: BulkAction) {
    if (action === "delete") {
      if (!confirm(`Delete ${count} submission${count !== 1 ? "s" : ""}? This cannot be undone.`)) {
        return;
      }
    }
    try {
      const res = await fetch("/admin/api/ext/forms/submissions/bulk", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action, ids: Array.from(selectedIds) }),
        credentials: "include",
      });
      if (!res.ok) {
        toast.error("Bulk action failed");
        return;
      }
      const data = await res.json();
      toast.success(`${data.count} submission${data.count !== 1 ? "s" : ""} updated`);
      onClearSelection();
      onBulkComplete();
    } catch {
      toast.error("Bulk action failed");
    }
  }

  return (
    <div className="flex items-center gap-2 mb-2.5 flex-wrap px-2.5 h-[30px] rounded border border-indigo-200 bg-indigo-50/70">
      <span className="text-[12px] font-medium text-indigo-700">
        {count} selected
      </span>
      <div className="h-3 w-px bg-indigo-200" />
      <Button
        variant="ghost"
        size="sm"
        className="text-indigo-700 hover:bg-indigo-100 text-xs gap-1.5"
        onClick={() => executeBulk("mark_read")}
      >
        <CheckCircle className="h-3.5 w-3.5" />
        Mark Read
      </Button>
      <Button
        variant="ghost"
        size="sm"
        className="text-indigo-700 hover:bg-indigo-100 text-xs gap-1.5"
        onClick={() => executeBulk("mark_unread")}
      >
        <Circle className="h-3.5 w-3.5" />
        Mark Unread
      </Button>
      <Button
        variant="ghost"
        size="sm"
        className="text-indigo-700 hover:bg-indigo-100 text-xs gap-1.5"
        onClick={() => executeBulk("archive")}
      >
        <Archive className="h-3.5 w-3.5" />
        Archive
      </Button>
      <Button
        variant="ghost"
        size="sm"
        className="text-red-600 hover:bg-red-50 text-xs gap-1.5"
        onClick={() => executeBulk("delete")}
      >
        <Trash2 className="h-3.5 w-3.5" />
        Delete
      </Button>
      <div className="ml-auto">
        <Button
          variant="ghost"
          size="sm"
          className="text-slate-500 text-xs"
          onClick={onClearSelection}
        >
          Clear selection
        </Button>
      </div>
    </div>
  );
}
