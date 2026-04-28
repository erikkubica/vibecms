import { useEffect, useRef, useState } from "react";
import { Eye, RefreshCw, Loader2 } from "@vibecms/icons";
import {
  ListHeader,
  Button,
  Input,
  Label,
  Badge,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  Card,
  CardContent,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@vibecms/ui";
import { toast } from "sonner";
import { getEmailLogs, resendEmail, getSystemActions } from "@vibecms/api";

interface EmailLog {
  id: number;
  action: string;
  recipient_email: string;
  subject: string;
  rendered_body: string;
  status: string;
  error_message: string;
  created_at: string;
}

interface SystemAction {
  slug: string;
  label: string;
}

const PER_PAGE = 20;

// Bypass the Chromium srcDoc + sandbox="" rendering glitch by writing the
// HTML through contentWindow.document instead.
function PreviewIframe({ html, title }: { html: string; title: string }) {
  const ref = useRef<HTMLIFrameElement>(null);
  useEffect(() => {
    const win = ref.current?.contentWindow;
    if (!win) return;
    win.document.open();
    win.document.write(html);
    win.document.close();
  }, [html]);
  return <iframe ref={ref} title={title} className="w-full h-full border-0" />;
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleString();
}

function statusBadge(status: string) {
  switch (status) {
    case "sent":
      return <Badge className="bg-green-100 text-green-700 hover:bg-green-100 border-0 text-xs">Sent</Badge>;
    case "failed":
      return <Badge className="bg-red-100 text-red-700 hover:bg-red-100 border-0 text-xs">Failed</Badge>;
    case "pending":
      return <Badge className="bg-yellow-100 text-yellow-700 hover:bg-yellow-100 border-0 text-xs">Pending</Badge>;
    default:
      return <Badge variant="outline" className="text-xs">{status}</Badge>;
  }
}

export default function EmailLogs() {
  const [logs, setLogs] = useState<EmailLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);

  // Filters
  const [filterStatus, setFilterStatus] = useState("");
  const [filterAction, setFilterAction] = useState("");
  const [filterRecipient, setFilterRecipient] = useState("");
  const [filterDateFrom, setFilterDateFrom] = useState("");
  const [filterDateTo, setFilterDateTo] = useState("");

  // Lookups
  const [actions, setActions] = useState<SystemAction[]>([]);

  // View dialog
  const [showView, setShowView] = useState(false);
  const [viewingLog, setViewingLog] = useState<EmailLog | null>(null);

  // Resend loading
  const [resending, setResending] = useState<number | null>(null);

  async function fetchLogs(p?: number) {
    setLoading(true);
    try {
      const currentPage = p ?? page;
      const res = await getEmailLogs({
        status: filterStatus || undefined,
        action: filterAction || undefined,
        recipient: filterRecipient || undefined,
        date_from: filterDateFrom || undefined,
        date_to: filterDateTo || undefined,
        page: currentPage,
        per_page: PER_PAGE,
      });
      setLogs(res.data);
      setTotal(res.total);
      setPage(currentPage);
    } catch {
      toast.error("Failed to load email logs");
    } finally {
      setLoading(false);
    }
  }

  async function fetchActions() {
    try {
      const data = await getSystemActions();
      setActions(data);
    } catch {
      // Non-fatal
    }
  }

  useEffect(() => {
    fetchLogs(1);
    fetchActions();
  }, []);

  function handleFilter() {
    fetchLogs(1);
  }

  function openViewDialog(log: EmailLog) {
    setViewingLog(log);
    setShowView(true);
  }

  async function handleResend(id: number) {
    setResending(id);
    try {
      await resendEmail(id);
      toast.success("Email resent successfully");
      await fetchLogs();
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to resend email";
      toast.error(message);
    } finally {
      setResending(null);
    }
  }

  const totalPages = Math.ceil(total / PER_PAGE);

  return (
    <div className="w-full pb-8 space-y-4">
      <ListHeader
        title="Email Logs"
        tabs={[{ value: "all", label: "All", count: total }]}
        activeTab="all"
      />

      {/* Filters */}
      <Card className="rounded-xl border border-slate-200 shadow-sm">
        <CardContent className="p-4">
          <div className="flex flex-wrap items-end gap-4">
            <div className="space-y-1">
              <Label className="text-xs font-medium text-slate-500">Status</Label>
              <Select value={filterStatus || "__all__"} onValueChange={(v) => setFilterStatus(v === "__all__" ? "" : v)}>
                <SelectTrigger className="w-32 rounded-lg border-slate-300">
                  <SelectValue placeholder="All" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__all__">All</SelectItem>
                  <SelectItem value="sent">Sent</SelectItem>
                  <SelectItem value="failed">Failed</SelectItem>
                  <SelectItem value="pending">Pending</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1">
              <Label className="text-xs font-medium text-slate-500">Action</Label>
              <Select value={filterAction || "__all__"} onValueChange={(v) => setFilterAction(v === "__all__" ? "" : v)}>
                <SelectTrigger className="w-40 rounded-lg border-slate-300">
                  <SelectValue placeholder="All" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__all__">All</SelectItem>
                  {actions.map((act) => (
                    <SelectItem key={act.slug} value={act.slug}>
                      {act.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1">
              <Label className="text-xs font-medium text-slate-500">Recipient</Label>
              <Input
                placeholder="Search recipient..."
                value={filterRecipient}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setFilterRecipient(e.target.value)}
                className="w-48 rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs font-medium text-slate-500">From</Label>
              <Input
                type="date"
                value={filterDateFrom}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setFilterDateFrom(e.target.value)}
                className="w-40 rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs font-medium text-slate-500">To</Label>
              <Input
                type="date"
                value={filterDateTo}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setFilterDateTo(e.target.value)}
                className="w-40 rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs font-medium text-transparent select-none">&nbsp;</Label>
              <Button
                className="bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg"
                onClick={handleFilter}
              >
                Filter
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Table */}
      <Card className="rounded-xl border border-slate-200 shadow-sm">
        <CardContent className="p-0">
          {loading ? (
            <div className="flex h-48 items-center justify-center">
              <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className="border-slate-200 hover:bg-transparent">
                  <TableHead className="text-slate-500 font-medium">Date</TableHead>
                  <TableHead className="text-slate-500 font-medium">Action</TableHead>
                  <TableHead className="text-slate-500 font-medium">Recipient</TableHead>
                  <TableHead className="text-slate-500 font-medium">Subject</TableHead>
                  <TableHead className="text-slate-500 font-medium">Status</TableHead>
                  <TableHead className="text-slate-500 font-medium text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {logs.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center py-12 text-slate-400">
                      No email logs found.
                    </TableCell>
                  </TableRow>
                )}
                {logs.map((log) => (
                  <TableRow key={log.id} className="border-slate-100">
                    <TableCell className="text-sm text-slate-600 whitespace-nowrap">
                      {formatDate(log.created_at)}
                    </TableCell>
                    <TableCell className="text-sm text-slate-700">{log.action}</TableCell>
                    <TableCell className="text-sm text-slate-700">{log.recipient_email}</TableCell>
                    <TableCell className="text-sm text-slate-600 max-w-xs truncate">
                      {log.subject}
                    </TableCell>
                    <TableCell>{statusBadge(log.status)}</TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-slate-500 hover:text-indigo-600"
                          onClick={() => openViewDialog(log)}
                          title="View"
                        >
                          <Eye className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-slate-500 hover:text-indigo-600"
                          onClick={() => handleResend(log.id)}
                          disabled={resending === log.id}
                          title="Resend"
                        >
                          {resending === log.id ? (
                            <Loader2 className="h-4 w-4 animate-spin" />
                          ) : (
                            <RefreshCw className="h-4 w-4" />
                          )}
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-slate-500">
            Page {page} of {totalPages} ({total} total)
          </p>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => fetchLogs(page - 1)}
              disabled={page <= 1 || loading}
              className="rounded-lg border-slate-300"
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => fetchLogs(page + 1)}
              disabled={page >= totalPages || loading}
              className="rounded-lg border-slate-300"
            >
              Next
            </Button>
          </div>
        </div>
      )}

      {/* View Dialog */}
      <Dialog open={showView} onOpenChange={setShowView}>
        <DialogContent className="max-w-3xl max-h-[80vh]">
          <DialogHeader>
            <DialogTitle>Email Details</DialogTitle>
          </DialogHeader>
          {viewingLog && (
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="font-medium text-slate-500">Recipient:</span>{" "}
                  <span className="text-slate-800">{viewingLog.recipient_email}</span>
                </div>
                <div>
                  <span className="font-medium text-slate-500">Status:</span>{" "}
                  {statusBadge(viewingLog.status)}
                </div>
                <div>
                  <span className="font-medium text-slate-500">Subject:</span>{" "}
                  <span className="text-slate-800">{viewingLog.subject}</span>
                </div>
                <div>
                  <span className="font-medium text-slate-500">Date:</span>{" "}
                  <span className="text-slate-800">{formatDate(viewingLog.created_at)}</span>
                </div>
                {viewingLog.error_message && (
                  <div className="col-span-2">
                    <span className="font-medium text-red-500">Error:</span>{" "}
                    <span className="text-red-700">{viewingLog.error_message}</span>
                  </div>
                )}
              </div>
              <div className="rounded-lg border border-slate-200 bg-white overflow-hidden" style={{ height: "400px" }}>
                <PreviewIframe html={viewingLog.rendered_body} title="Rendered Email" />
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
