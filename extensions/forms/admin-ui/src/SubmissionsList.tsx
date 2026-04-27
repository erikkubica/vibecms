import React, { useCallback, useEffect, useState } from "react";
import { Download, ArrowLeft, Inbox } from "@vibecms/icons";
import { Submission } from "./submissions/SubmissionRow";
import SubmissionRow from "./submissions/SubmissionRow";
import SubmissionsToolbar, { ToolbarFilters } from "./submissions/SubmissionsToolbar";
import BulkActionsBar from "./submissions/BulkActionsBar";

const {
  Button,
  ListPageShell,
  ListHeader,
  ListCard,
  ListTable,
  ListFooter,
  Th,
  EmptyState,
  LoadingRow,
  Checkbox,
} = (window as any).__VIBECMS_SHARED__.ui;
const { useSearchParams, useNavigate } = (window as any).__VIBECMS_SHARED__
  .ReactRouterDOM;
const { toast } = (window as any).__VIBECMS_SHARED__.Sonner;

interface PaginatedResult {
  rows: Submission[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
  status_counts?: { all: number; unread: number; read: number; archived: number };
}

export default function SubmissionsList() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const formId = searchParams.get("form_id");

  const [submissions, setSubmissions] = useState<Submission[]>([]);
  const [loading, setLoading] = useState(true);
  const [exporting, setExporting] = useState(false);

  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [total, setTotal] = useState(0);
  const perPage = 25;

  const [filters, setFilters] = useState<ToolbarFilters>({
    search: "",
    status: "",
    dateFrom: "",
    dateTo: "",
  });
  const [searchInput, setSearchInput] = useState("");
  const [statusCounts, setStatusCounts] = useState({ all: 0, unread: 0, read: 0, archived: 0 });

  useEffect(() => {
    const timer = setTimeout(() => {
      setFilters((prev) =>
        prev.search === searchInput ? prev : { ...prev, search: searchInput },
      );
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchInput]);

  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());

  const fetchSubmissions = useCallback(
    async (currentPage = page, currentFilters = filters) => {
      setLoading(true);
      try {
        const params = new URLSearchParams();
        params.set("page", String(currentPage));
        params.set("per_page", String(perPage));
        if (formId) params.set("form_id", formId);
        if (currentFilters.status) params.set("status", currentFilters.status);
        if (currentFilters.search) params.set("search", currentFilters.search);
        if (currentFilters.dateFrom) params.set("date_from", currentFilters.dateFrom);
        if (currentFilters.dateTo) params.set("date_to", currentFilters.dateTo);

        const res = await fetch(
          `/admin/api/ext/forms/submissions?${params.toString()}`,
          { credentials: "include" },
        );
        const body: PaginatedResult = await res.json();
        setSubmissions(body.rows || []);
        setTotal(body.total ?? 0);
        setTotalPages(body.total_pages ?? 1);
        if (body.status_counts) setStatusCounts(body.status_counts);
        setSelectedIds(new Set());
      } catch {
        toast.error("Failed to load submissions");
      } finally {
        setLoading(false);
      }
    },
    [formId, page, filters],
  );

  useEffect(() => {
    fetchSubmissions(page, filters);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [formId, page, filters]);

  const handleFilterChange = useCallback((next: Partial<ToolbarFilters>) => {
    setFilters((prev) => ({ ...prev, ...next }));
    setPage(1);
  }, []);

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
    navigate(`/admin/ext/forms/submissions/${sub.id}`);
  };

  const toggleSelect = useCallback((id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const toggleSelectAll = useCallback(() => {
    if (selectedIds.size === submissions.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(submissions.map((s) => s.id)));
    }
  }, [selectedIds.size, submissions]);

  const unreadCount = submissions.filter((s) => s.status === "unread").length;
  const allSelected = submissions.length > 0 && selectedIds.size === submissions.length;
  const someSelected = selectedIds.size > 0;
  const colCount = formId ? 5 : 6;

  const headerTitle = formId
    ? `${submissions[0]?.form_name || "Form #" + formId} Submissions`
    : "Submissions";

  const extraButton = (
    <button
      type="button"
      onClick={exportCSV}
      disabled={!formId || exporting}
      title={
        formId
          ? "Export submissions as CSV"
          : "Select a specific form to enable CSV export"
      }
      className="h-[26px] px-2.5 inline-flex items-center gap-1.5 text-[12px] font-medium text-white bg-indigo-600 border border-indigo-600 rounded hover:bg-indigo-700 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
    >
      <Download className="w-3 h-3" />
      {exporting ? "Exporting…" : "Export CSV"}
    </button>
  );

  return (
    <ListPageShell>
      <ListHeader
        title={headerTitle}
        leading={
          formId ? (
            <button
              type="button"
              onClick={() => navigate("/admin/ext/forms")}
              className="w-[26px] h-[26px] grid place-items-center text-slate-500 hover:bg-slate-100 rounded border-0 bg-transparent cursor-pointer"
              aria-label="Back to forms"
            >
              <ArrowLeft className="h-4 w-4" />
            </button>
          ) : undefined
        }
        tabs={[
          { value: "", label: "All", count: statusCounts.all },
          { value: "unread", label: "Unread", count: statusCounts.unread },
          { value: "read", label: "Read", count: statusCounts.read },
          { value: "archived", label: "Archived", count: statusCounts.archived },
        ]}
        activeTab={filters.status}
        onTabChange={(v: string) => handleFilterChange({ status: v })}
        extra={extraButton}
      />

      {someSelected ? (
        <BulkActionsBar
          selectedIds={selectedIds}
          onClearSelection={() => setSelectedIds(new Set())}
          onBulkComplete={() => fetchSubmissions(page, filters)}
        />
      ) : (
        <SubmissionsToolbar
          filters={filters}
          searchValue={searchInput}
          onSearchChange={setSearchInput}
          onChange={handleFilterChange}
        />
      )}

      <ListCard>
        {loading ? (
          <LoadingRow />
        ) : submissions.length === 0 ? (
          <EmptyState
            icon={Inbox}
            title="No submissions found"
            description={
              filters.search || filters.status || filters.dateFrom || filters.dateTo
                ? "Try adjusting your filters"
                : "No submissions have been received yet"
            }
          />
        ) : (
          <ListTable minWidth={760}>
            <thead>
              <tr>
                <Th width={40}>
                  <Checkbox
                    checked={allSelected}
                    onCheckedChange={toggleSelectAll}
                    aria-label="Select all"
                  />
                </Th>
                <Th width={180}>Date</Th>
                {!formId && <Th width={160}>Form</Th>}
                <Th>Summary</Th>
                <Th width={100} align="right">Actions</Th>
              </tr>
            </thead>
            <tbody>
              {submissions.map((sub, index) => (
                <SubmissionRow
                  key={sub.id}
                  sub={sub}
                  index={index}
                  formId={formId}
                  selected={selectedIds.has(sub.id)}
                  onToggle={toggleSelect}
                  onViewDetails={handleViewDetails}
                />
              ))}
            </tbody>
          </ListTable>
        )}

        {!loading && totalPages > 1 && (
          <ListFooter
            page={page}
            totalPages={totalPages}
            total={total}
            perPage={perPage}
            onPage={setPage}
            label="submissions"
          />
        )}
      </ListCard>
    </ListPageShell>
  );
}
