import React from "react";
import { Eye, Calendar } from "@vibecms/icons";

const { Tr, Td, StatusPill, Chip, Checkbox, Button } =
  (window as any).__VIBECMS_SHARED__.ui;

export type SubmissionStatus = "unread" | "read" | "archived";

export interface Submission {
  id: number;
  form_id: number;
  data: Record<string, unknown>;
  metadata: Record<string, unknown>;
  created_at: string;
  form_name?: string;
  status?: SubmissionStatus;
}

const STATUS_MAP: Record<SubmissionStatus, { status: string; label: string }> = {
  unread: { status: "active", label: "Unread" },
  read: { status: "published", label: "Read" },
  archived: { status: "inactive", label: "Archived" },
};

interface SubmissionRowProps {
  sub: Submission;
  index: number;
  formId: string | null;
  selected: boolean;
  onToggle: (id: number) => void;
  onViewDetails: (sub: Submission) => void;
}

export default function SubmissionRow({
  sub,
  index: _index,
  formId,
  selected,
  onToggle,
  onViewDetails,
}: SubmissionRowProps) {
  const isUnread = sub.status === "unread";

  return (
    <Tr
      className={`${selected ? "bg-indigo-50/40 " : ""}cursor-pointer hover:bg-slate-50`}
      onClick={() => onViewDetails(sub)}
    >
      {/* Checkbox */}
      <Td onClick={(e: React.MouseEvent) => e.stopPropagation()}>
        <Checkbox
          checked={selected}
          onCheckedChange={() => onToggle(sub.id)}
          aria-label={`Select submission ${sub.id}`}
        />
      </Td>

      {/* Date */}
      <Td className="whitespace-nowrap">
        <div className="flex items-center gap-1.5 text-[12px] text-slate-500">
          {isUnread ? (
            <span
              className="inline-block h-2 w-2 rounded-full bg-indigo-500 shrink-0"
              title="Unread"
            />
          ) : (
            <Calendar className="h-3 w-3 opacity-40 shrink-0" />
          )}
          <span className={isUnread ? "font-semibold text-slate-700" : ""}>
            {new Date(sub.created_at).toLocaleString()}
          </span>
        </div>
      </Td>

      {/* Form name — only when not filtered */}
      {!formId && (
        <Td>
          <Chip>{sub.form_name}</Chip>
        </Td>
      )}

      {/* Summary */}
      <Td className="max-w-[400px]">
        <div className="text-[12px] text-slate-500 truncate">
          {Object.entries(sub.data)
            .slice(0, 3)
            .map(([k, v]) => (
              <span key={k} className="mr-3">
                <strong className="text-slate-700">{k}:</strong>{" "}
                {typeof v === "object" ? JSON.stringify(v) : String(v)}
              </span>
            ))}
          {Object.keys(sub.data).length > 3 && (
            <span className="text-slate-400">…</span>
          )}
        </div>
      </Td>

      {/* Actions */}
      <Td align="right" onClick={(e: React.MouseEvent) => e.stopPropagation()}>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onViewDetails(sub)}
          className="text-indigo-600 hover:text-indigo-700 hover:bg-indigo-50 text-xs"
        >
          <Eye className="mr-1.5 h-3.5 w-3.5" /> View
        </Button>
      </Td>
    </Tr>
  );
}

// Kept for legacy imports (tests + SubmissionDetailDialog badge styling)
export const STATUS_BADGE_VARIANTS: Record<
  SubmissionStatus,
  { className: string; label: string }
> = {
  unread: {
    label: "Unread",
    className: "bg-blue-100 text-blue-700 border-blue-200 hover:bg-blue-100",
  },
  read: {
    label: "Read",
    className: "bg-green-100 text-green-700 border-green-200 hover:bg-green-100",
  },
  archived: {
    label: "Archived",
    className: "bg-slate-100 text-slate-500 border-slate-200 hover:bg-slate-100",
  },
};
