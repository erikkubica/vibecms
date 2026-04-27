import React, { useCallback, useEffect, useState } from "react";
import { ArrowLeft, Filter, Inbox } from "@vibecms/icons";
import {
  Submission,
  STATUS_BADGE_VARIANTS,
  SubmissionStatus,
} from "./submissions/SubmissionRow";

const {
  Badge,
  Button,
  ListPageShell,
  ListHeader,
  ListCard,
  EmptyState,
  LoadingRow,
} = (window as any).__VIBECMS_SHARED__.ui;
const { useParams, useNavigate } = (window as any).__VIBECMS_SHARED__.ReactRouterDOM;
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
    return (
      <pre className="text-xs whitespace-pre-wrap">
        {JSON.stringify(value, null, 2)}
      </pre>
    );
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
  const str = String(value ?? "");
  return str === "" ? <span className="text-slate-300">—</span> : str;
}

export default function SubmissionDetail() {
  const { id: idParam } = useParams();
  const navigate = useNavigate();
  const id = Number(idParam);

  const [submission, setSubmission] = useState<EnrichedSubmission | null>(null);
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState(false);

  const fetchSubmission = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const res = await fetch(`/admin/api/ext/forms/submissions/${id}`, {
        credentials: "include",
      });
      if (res.ok) {
        const body = await res.json();
        setSubmission(body);
        if (body.status === "unread") {
          fetch(`/admin/api/ext/forms/submissions/${id}`, {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ status: "read" }),
            credentials: "include",
          }).catch(() => {});
        }
      } else if (res.status === 404) {
        setSubmission(null);
      } else {
        toast.error("Failed to load submission");
      }
    } catch {
      toast.error("Failed to load submission");
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    fetchSubmission();
  }, [fetchSubmission]);

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
        setSubmission({ ...submission, status: newStatus });
        toast.success(`Marked as ${newStatus}`);
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
        toast.success("Submission deleted");
        navigate("/admin/ext/forms/submissions");
      } else {
        toast.error("Failed to delete submission");
      }
    } catch {
      toast.error("Failed to delete submission");
    } finally {
      setActing(false);
    }
  }

  const labelMap: Record<string, string> = {};
  if (submission?.form_fields) {
    for (const field of submission.form_fields) {
      const fid = field["id"] as string;
      const label = (field["label"] as string) || fid;
      if (fid) labelMap[fid] = label;
    }
  }

  const headerTitle = submission
    ? `Submission #${submission.id}`
    : "Submission";
  const status = submission?.status as SubmissionStatus | undefined;

  return (
    <ListPageShell>
      <ListHeader
        title={headerTitle}
        extra={
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => navigate(-1)}
              className="w-[26px] h-[26px] grid place-items-center text-slate-500 hover:bg-slate-100 rounded border-0 bg-transparent cursor-pointer"
              aria-label="Back"
            >
              <ArrowLeft className="h-4 w-4" />
            </button>
            {status && (
              <Badge
                variant="outline"
                className={`font-normal text-xs ${STATUS_BADGE_VARIANTS[status].className}`}
              >
                {STATUS_BADGE_VARIANTS[status].label}
              </Badge>
            )}
            {submission && status === "read" && (
              <Button
                variant="outline"
                size="sm"
                disabled={acting}
                onClick={() => patchStatus("unread")}
              >
                Mark Unread
              </Button>
            )}
            {submission && status !== "archived" && (
              <Button
                variant="outline"
                size="sm"
                disabled={acting}
                onClick={() => patchStatus("archived")}
              >
                Archive
              </Button>
            )}
            {submission && status === "archived" && (
              <Button
                variant="outline"
                size="sm"
                disabled={acting}
                onClick={() => patchStatus("read")}
              >
                Unarchive
              </Button>
            )}
            {submission && (
              <Button
                variant="destructive"
                size="sm"
                disabled={acting}
                onClick={handleDelete}
              >
                Delete
              </Button>
            )}
          </div>
        }
      />

      <ListCard>
        {loading ? (
          <LoadingRow />
        ) : !submission ? (
          <EmptyState
            icon={Inbox}
            title="Submission not found"
            description="This submission may have been deleted."
          />
        ) : (
          <div className="space-y-6 p-6">
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 p-4 bg-slate-50 rounded-lg border border-slate-100 text-sm">
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
                  Form
                </p>
                <p className="font-medium text-slate-900">
                  {submission.form_name || "N/A"}
                </p>
              </div>
              <div>
                <p className="text-slate-400 uppercase text-[10px] tracking-wider mb-1">
                  Submission ID
                </p>
                <p className="font-medium text-slate-900">#{submission.id}</p>
              </div>
            </div>

            <div className="space-y-3">
              <h4 className="font-semibold text-slate-900 flex items-center gap-2 text-sm">
                <Filter className="h-4 w-4 text-indigo-500" />
                Submitted Data
              </h4>
              <div className="divide-y divide-slate-100 border border-slate-100 rounded-lg overflow-hidden">
                {Object.entries(submission.data || {}).map(([key, value]) => (
                  <div
                    key={key}
                    className="grid grid-cols-1 sm:grid-cols-4 gap-2 p-3 text-sm"
                  >
                    <div className="font-medium text-slate-600 capitalize sm:col-span-1">
                      {labelMap[key] || key.replace(/_/g, " ")}
                    </div>
                    <div className="sm:col-span-3 text-slate-900 break-words">
                      {renderFieldValue(value)}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {submission.metadata &&
              typeof submission.metadata === "object" &&
              Object.keys(submission.metadata).length > 0 && (
                <div className="space-y-2">
                  <h4 className="font-semibold text-slate-900 text-sm">
                    Technical Metadata
                  </h4>
                  <div className="divide-y divide-slate-100 border border-slate-100 rounded-lg overflow-hidden">
                    {Object.entries(submission.metadata as Record<string, unknown>).map(
                      ([key, value]) => (
                        <div
                          key={key}
                          className="grid grid-cols-1 sm:grid-cols-4 gap-2 p-3 text-sm"
                        >
                          <div className="font-medium text-slate-600 capitalize sm:col-span-1">
                            {key.replace(/_/g, " ")}
                          </div>
                          <div className="sm:col-span-3 text-slate-900 break-all font-mono text-xs">
                            {renderFieldValue(value)}
                          </div>
                        </div>
                      ),
                    )}
                  </div>
                </div>
              )}
          </div>
        )}
      </ListCard>
    </ListPageShell>
  );
}
