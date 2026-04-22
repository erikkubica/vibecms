import React, { useEffect, useState } from "react";
import {
  Search,
  Eye,
  Download,
  Calendar,
  ArrowLeft,
  Filter,
} from "@vibecms/icons";

const {
  Button,
  Card,
  CardContent,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Input,
  Badge,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} = (window as any).__VIBECMS_SHARED__.ui;
const { useSearchParams, useNavigate } = (window as any).__VIBECMS_SHARED__
  .ReactRouterDOM;
const { toast } = (window as any).__VIBECMS_SHARED__.Sonner;

type SubmissionStatus = "unread" | "read" | "archived";

interface Submission {
  id: number;
  form_id: number;
  data: Record<string, any>;
  metadata: Record<string, any>;
  created_at: string;
  form_name?: string;
  status?: SubmissionStatus;
}

const STATUS_BADGE_VARIANTS: Record<
  SubmissionStatus,
  { className: string; label: string }
> = {
  unread: {
    label: "Unread",
    className: "bg-blue-100 text-blue-700 border-blue-200 hover:bg-blue-100",
  },
  read: {
    label: "Read",
    className:
      "bg-green-100 text-green-700 border-green-200 hover:bg-green-100",
  },
  archived: {
    label: "Archived",
    className:
      "bg-slate-100 text-slate-500 border-slate-200 hover:bg-slate-100",
  },
};

export default function SubmissionsList() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const formId = searchParams.get("form_id");

  const [submissions, setSubmissions] = useState<Submission[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedSubmission, setSelectedSubmission] =
    useState<Submission | null>(null);
  const [search, setSearch] = useState("");
  const [exporting, setExporting] = useState(false);

  const fetchSubmissions = async () => {
    try {
      const url = formId
        ? `/admin/api/ext/forms/submissions?form_id=${formId}`
        : "/admin/api/ext/forms/submissions";
      const res = await fetch(url, { credentials: "include" });
      const body = await res.json();
      setSubmissions(body.rows || []);
    } catch (err) {
      toast.error("Failed to load submissions");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSubmissions();
  }, [formId]);

  const exportCSV = async () => {
    if (!formId) {
      toast.error("Please filter by a specific form to export");
      return;
    }
    setExporting(true);
    try {
      const res = await fetch(
        `/admin/api/ext/forms/submissions/export?form_id=${formId}`,
        { credentials: "include" },
      );
      if (res.ok) {
        const blob = await res.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = "form-submissions.csv";
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);
        toast.success("CSV exported successfully");
      } else {
        toast.error("Failed to export CSV");
      }
    } catch {
      toast.error("Export failed");
    } finally {
      setExporting(false);
    }
  };

  const handleViewDetails = (sub: Submission) => {
    setSelectedSubmission(sub);

    // TODO: mark as read when backend endpoint exists
    // PUT /admin/api/ext/forms/submissions/{id}/status
    // Body: { status: "read" }

    if (sub.status === "unread") {
      setSubmissions((prev) =>
        prev.map((s) =>
          s.id === sub.id ? { ...s, status: "read" as SubmissionStatus } : s,
        ),
      );
      setSelectedSubmission({ ...sub, status: "read" });
    }
  };

  const filteredSubmissions = submissions.filter((s) => {
    const dataStr = JSON.stringify(s.data).toLowerCase();
    return dataStr.includes(search.toLowerCase());
  });

  const unreadCount = submissions.filter((s) => s.status === "unread").length;

  const headerSubtitle = () => {
    if (!formId) {
      return "All form submissions across the site.";
    }
    const formName = submissions[0]?.form_name || "Form #" + formId;
    const unreadPart = unreadCount > 0 ? ` (${unreadCount} unread)` : "";
    return `Viewing entries for ${formName}${unreadPart}`;
  };

  const colSpan = formId ? 5 : 6;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          {formId && (
            <Button
              variant="ghost"
              size="icon"
              onClick={() => navigate("/admin/ext/forms")}
              className="rounded-full"
            >
              <ArrowLeft className="h-4 w-4" />
            </Button>
          )}
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-slate-900">
              Submissions
            </h1>
            <p className="text-sm text-slate-500">{headerSubtitle()}</p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            size="sm"
            onClick={exportCSV}
            disabled={!formId || exporting}
            title={
              formId
                ? "Export submissions as CSV"
                : "Select a specific form to enable CSV export"
            }
          >
            <Download className="mr-2 h-4 w-4" />
            {exporting ? "Exporting..." : "Export CSV"}
          </Button>
        </div>
      </div>

      <Card className="rounded-xl border border-slate-200 shadow-sm overflow-hidden">
        <CardContent className="p-0">
          <div className="flex items-center gap-4 p-4 border-b border-slate-100 bg-slate-50/50">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
              <Input
                placeholder="Search in submissions..."
                className="pl-9 bg-white border-slate-200 focus:border-indigo-500"
                value={search}
                onChange={(e: any) => setSearch(e.target.value)}
              />
            </div>
          </div>

          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent border-slate-100">
                <TableHead className="text-slate-500 font-medium w-8 px-3" />
                <TableHead className="text-slate-500 font-medium text-xs">
                  Date
                </TableHead>
                {!formId && (
                  <TableHead className="text-slate-500 font-medium text-xs">
                    Form
                  </TableHead>
                )}
                <TableHead className="text-slate-500 font-medium text-xs">
                  Status
                </TableHead>
                <TableHead className="text-slate-500 font-medium text-xs">
                  Summary
                </TableHead>
                <TableHead className="text-right text-slate-500 font-medium text-xs">
                  Actions
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell
                    colSpan={colSpan}
                    className="h-32 text-center text-slate-400 text-sm"
                  >
                    Loading entries...
                  </TableCell>
                </TableRow>
              ) : filteredSubmissions.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={colSpan}
                    className="h-32 text-center text-slate-400 text-sm"
                  >
                    No submissions found.
                  </TableCell>
                </TableRow>
              ) : (
                filteredSubmissions.map((sub, index) => {
                  const isArchived = sub.status === "archived";
                  const isUnread = sub.status === "unread";
                  const isEven = index % 2 === 1;

                  return (
                    <TableRow
                      key={sub.id}
                      className={`border-slate-100 transition-colors ${
                        isArchived ? "opacity-50" : "hover:bg-slate-50/50"
                      } ${isEven ? "bg-slate-50/30" : ""}`}
                    >
                      <TableCell className="w-8 px-3 py-2.5">
                        {isUnread && (
                          <span
                            className="inline-block h-2.5 w-2.5 rounded-full bg-blue-500"
                            title="Unread"
                          />
                        )}
                        {isArchived && (
                          <span
                            className="inline-block h-2.5 w-2.5 rounded-full bg-slate-300"
                            title="Archived"
                          />
                        )}
                      </TableCell>
                      <TableCell className="text-slate-600 text-xs whitespace-nowrap py-2.5">
                        <div className="flex items-center gap-1.5">
                          <Calendar className="h-3 w-3 opacity-40 flex-shrink-0" />
                          {new Date(sub.created_at).toLocaleString()}
                        </div>
                      </TableCell>
                      {!formId && (
                        <TableCell className="py-2.5">
                          <Badge
                            variant="outline"
                            className="bg-slate-100 text-slate-600 border-slate-200 text-xs"
                          >
                            {sub.form_name || "Unknown"}
                          </Badge>
                        </TableCell>
                      )}
                      <TableCell className="py-2.5">
                        {sub.status && sub.status !== "read" && (
                          <Badge
                            variant="outline"
                            className={`text-[10px] ${
                              STATUS_BADGE_VARIANTS[sub.status].className
                            }`}
                          >
                            {STATUS_BADGE_VARIANTS[sub.status].label}
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell className="max-w-[400px] py-2.5">
                        <div className="text-xs text-slate-500 truncate">
                          {Object.entries(sub.data)
                            .slice(0, 3)
                            .map(([k, v]) => (
                              <span key={k} className="mr-3">
                                <strong className="text-slate-700">{k}:</strong>{" "}
                                {typeof v === "object"
                                  ? JSON.stringify(v)
                                  : String(v)}
                              </span>
                            ))}
                          {Object.keys(sub.data).length > 3 && (
                            <span className="text-slate-400">...</span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-right py-2.5">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleViewDetails(sub)}
                          className="text-indigo-600 hover:text-indigo-700 hover:bg-indigo-50 text-xs"
                        >
                          <Eye className="mr-1.5 h-3.5 w-3.5" /> View Details
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Dialog
        open={!!selectedSubmission}
        onOpenChange={(open) => !open && setSelectedSubmission(null)}
      >
        <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center justify-between">
              <span className="flex items-center gap-3">
                Submission Details
                {selectedSubmission?.status && (
                  <Badge
                    variant="outline"
                    className={`font-normal text-xs ${
                      STATUS_BADGE_VARIANTS[selectedSubmission.status].className
                    }`}
                  >
                    {STATUS_BADGE_VARIANTS[selectedSubmission.status].label}
                  </Badge>
                )}
              </span>
              <Badge
                variant="outline"
                className="ml-4 font-normal text-xs text-slate-400"
              >
                ID: #{selectedSubmission?.id}
              </Badge>
            </DialogTitle>
          </DialogHeader>
          {selectedSubmission && (
            <div className="space-y-6 py-4">
              <div className="grid grid-cols-2 gap-4 p-4 bg-slate-50 rounded-lg border border-slate-100 text-sm">
                <div>
                  <p className="text-slate-400 uppercase text-[10px] tracking-wider mb-1">
                    Date Submitted
                  </p>
                  <p className="font-medium text-slate-900">
                    {new Date(selectedSubmission.created_at).toLocaleString()}
                  </p>
                </div>
                <div>
                  <p className="text-slate-400 uppercase text-[10px] tracking-wider mb-1">
                    Form Name
                  </p>
                  <p className="font-medium text-slate-900">
                    {selectedSubmission.form_name || "N/A"}
                  </p>
                </div>
              </div>

              <div className="space-y-4">
                <h4 className="font-semibold text-slate-900 flex items-center gap-2 text-sm">
                  <Filter className="h-4 w-4 text-indigo-500" />
                  Submitted Data
                </h4>
                <div className="divide-y divide-slate-100 border border-slate-100 rounded-lg overflow-hidden">
                  {Object.entries(selectedSubmission.data).map(
                    ([key, value]) => (
                      <div key={key} className="grid grid-cols-3 p-3 text-sm">
                        <div className="font-medium text-slate-600 capitalize">
                          {key.replace(/_/g, " ")}
                        </div>
                        <div className="col-span-2 text-slate-900 bg-white p-1 rounded min-h-[1.5rem] break-words">
                          {typeof value === "object"
                            ? JSON.stringify(value)
                            : String(value)}
                        </div>
                      </div>
                    ),
                  )}
                </div>
              </div>

              {Object.keys(selectedSubmission.metadata || {}).length > 0 && (
                <div className="space-y-2">
                  <h4 className="font-semibold text-slate-900 text-sm">
                    Technical Metadata
                  </h4>
                  <pre className="p-3 bg-slate-900 text-indigo-300 rounded-lg text-[11px] font-mono overflow-x-auto">
                    {JSON.stringify(selectedSubmission.metadata, null, 2)}
                  </pre>
                </div>
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
