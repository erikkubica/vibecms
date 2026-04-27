import React, { useState } from "react";
import { Filter } from "@vibecms/icons";
import { Submission, STATUS_BADGE_VARIANTS, SubmissionStatus } from "./SubmissionRow";

const {
  Badge,
  Button,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  StatusPill,
} = (window as any).__VIBECMS_SHARED__.ui;
const { toast } = (window as any).__VIBECMS_SHARED__.Sonner;

interface EnrichedSubmission extends Submission {
  form_fields?: Array<Record<string, unknown>>;
}

function renderFieldValue(value: unknown): React.ReactNode {
  if (value === true) return <span className="text-green-600 font-medium">✓</span>;
  if (value === false) return <span className="text-red-500 font-medium">✗</span>;
  if (value && typeof value === "object" && !Array.isArray(value)) {
    const obj = value as Record<string, unknown>;
    if ("label" in obj && "value" in obj && Object.keys(obj).length === 2) {
      return <span>{obj.label as string}</span>;
    }
    if ("url" in obj && "name" in obj) {
      const isImage =
        typeof obj.mime_type === "string" && obj.mime_type.startsWith("image/");
      const sizeKB =
        typeof obj.size === "number" ? (obj.size / 1024).toFixed(1) + " KB" : "";
      return (
        <div className="flex items-center gap-3">
          {isImage && (
            <img
              src={obj.url as string}
              alt={obj.name as string}
              className="h-12 w-12 object-cover rounded border"
            />
          )}
          <a
            href={obj.url as string}
            download={obj.name as string}
            className="text-indigo-600 hover:underline text-sm"
          >
            {obj.name as string}
          </a>
          {sizeKB && <span className="text-xs text-slate-400">{sizeKB}</span>}
        </div>
      );
    }
    return <pre className="text-xs">{JSON.stringify(value, null, 2)}</pre>;
  }
  if (Array.isArray(value)) {
    return (
      <div className="space-y-1">
        {value.map((v, i) => (
          <div key={i}>{renderFieldValue(v)}</div>
        ))}
      </div>
    );
  }
  return String(value ?? "");
}

interface SubmissionDetailDialogProps {
  submission: EnrichedSubmission | null;
  onClose: () => void;
  onStatusChange?: (id: number, newStatus: SubmissionStatus) => void;
  onDeleted?: (id: number) => void;
}

export default function SubmissionDetailDialog({
  submission,
  onClose,
  onStatusChange,
  onDeleted,
}: SubmissionDetailDialogProps) {
  const [acting, setActing] = useState(false);

  const labelMap: Record<string, string> = {};
  if (submission?.form_fields) {
    for (const field of submission.form_fields) {
      const id = field["id"] as string;
      const label = (field["label"] as string) || id;
      if (id) labelMap[id] = label;
    }
  }

  async function patchStatus(newStatus: SubmissionStatus) {
    if (!submission) return;
    setActing(true);
    try {
      const res = await fetch(`/admin/api/ext/forms/submissions/${submission.id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status: newStatus }),
        credentials: "include",
      });
      if (res.ok) {
        onStatusChange?.(submission.id, newStatus);
        toast.success(`Marked as ${newStatus}`);
        onClose();
      } else {
        toast.error("Failed to update status");
      }
    } catch {
      toast.error("Failed to update status");
    } finally {
      setActing(false);
    }
  }

  async function handleDelete() {
    if (!submission) return;
    if (!confirm("Delete this submission? This cannot be undone.")) return;
    setActing(true);
    try {
      const res = await fetch(`/admin/api/ext/forms/submissions/${submission.id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (res.ok) {
        onDeleted?.(submission.id);
        toast.success("Submission deleted");
        onClose();
      } else {
        toast.error("Failed to delete submission");
      }
    } catch {
      toast.error("Failed to delete submission");
    } finally {
      setActing(false);
    }
  }

  return (
    <Dialog open={!!submission} onOpenChange={(open: boolean) => !open && onClose()}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center justify-between">
            <span className="flex items-center gap-3">
              Submission Details
              {submission?.status && (
                <Badge
                  variant="outline"
                  className={`font-normal text-xs ${
                    STATUS_BADGE_VARIANTS[submission.status as SubmissionStatus].className
                  }`}
                >
                  {STATUS_BADGE_VARIANTS[submission.status as SubmissionStatus].label}
                </Badge>
              )}
            </span>
            <Badge variant="outline" className="ml-4 font-normal text-xs text-slate-400">
              ID: #{submission?.id}
            </Badge>
          </DialogTitle>
        </DialogHeader>

        {submission && (
          <div className="space-y-5 py-3">
            {/* Meta grid */}
            <div className="grid grid-cols-2 gap-4 p-4 bg-slate-50 rounded-lg border border-slate-100 text-sm">
              <div>
                <p className="text-slate-400 uppercase text-[10px] tracking-wider mb-1">
                  Date Submitted
                </p>
                <p className="font-medium text-slate-900">
                  {new Date(submission.created_at).toLocaleString()}
                </p>
              </div>
              <div>
                <p className="text-slate-400 uppercase text-[10px] tracking-wider mb-1">
                  Form Name
                </p>
                <p className="font-medium text-slate-900">
                  {submission.form_name || "N/A"}
                </p>
              </div>
            </div>

            {/* Submitted data */}
            <div className="space-y-3">
              <h4 className="font-semibold text-slate-900 flex items-center gap-2 text-sm">
                <Filter className="h-4 w-4 text-indigo-500" />
                Submitted Data
              </h4>
              <div className="divide-y divide-slate-100 border border-slate-100 rounded-lg overflow-hidden">
                {Object.entries(submission.data).map(([key, value]) => (
                  <div key={key} className="grid grid-cols-3 p-3 text-sm">
                    <div className="font-medium text-slate-600 capitalize">
                      {labelMap[key] || key.replace(/_/g, " ")}
                    </div>
                    <div className="col-span-2 text-slate-900 bg-white p-1 rounded min-h-[1.5rem] break-words">
                      {renderFieldValue(value)}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {Object.keys(submission.metadata || {}).length > 0 && (
              <div className="space-y-2">
                <h4 className="font-semibold text-slate-900 text-sm">
                  Technical Metadata
                </h4>
                <pre className="p-3 bg-slate-900 text-indigo-300 rounded-lg text-[11px] font-mono overflow-x-auto">
                  {JSON.stringify(submission.metadata, null, 2)}
                </pre>
              </div>
            )}
          </div>
        )}

        <DialogFooter className="gap-2 flex-wrap">
          {submission?.status === "read" && (
            <Button variant="outline" size="sm" disabled={acting} onClick={() => patchStatus("unread")}>
              Mark Unread
            </Button>
          )}
          {submission?.status !== "archived" && (
            <Button variant="outline" size="sm" disabled={acting} onClick={() => patchStatus("archived")}>
              Archive
            </Button>
          )}
          {submission?.status === "archived" && (
            <Button variant="outline" size="sm" disabled={acting} onClick={() => patchStatus("read")}>
              Unarchive
            </Button>
          )}
          <Button variant="destructive" size="sm" disabled={acting} onClick={handleDelete}>
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
