import React, { useEffect, useRef, useState } from "react";
import { FileText, Copy, ExternalLink, Trash2, Edit2, Upload } from "@vibecms/icons";

const {
  ListPageShell,
  ListHeader,
  ListToolbar,
  ListSearch,
  ListCard,
  ListTable,
  Th,
  Tr,
  Td,
  TitleCell,
  Chip,
  RowActions,
  EmptyState,
  LoadingRow,
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} = (window as any).__VIBECMS_SHARED__.ui;
const { useNavigate } = (window as any).__VIBECMS_SHARED__.ReactRouterDOM;
const { toast } = (window as any).__VIBECMS_SHARED__.Sonner;

interface Form {
  id: number;
  name: string;
  slug: string;
  created_at: string;
  submission_count?: number;
  last_submission_at?: string | null;
}

function relativeTime(dateStr: string | null | undefined): string {
  if (!dateStr) return "—";
  const date = new Date(dateStr);
  const diffMs = Date.now() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 60) return "just now";
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDays = Math.floor(diffHr / 24);
  if (diffDays < 30) return `${diffDays}d ago`;
  const diffMon = Math.floor(diffDays / 30);
  if (diffMon < 12) return `${diffMon}mo ago`;
  return `${Math.floor(diffMon / 12)}y ago`;
}

export default function FormsList() {
  const [forms, setForms] = useState<Form[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const navigate = useNavigate();

  const [showDelete, setShowDelete] = useState(false);
  const [deletingForm, setDeletingForm] = useState<Form | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [duplicating, setDuplicating] = useState<number | null>(null);
  const importInputRef = useRef<HTMLInputElement>(null);

  const fetchForms = async () => {
    try {
      const res = await fetch("/admin/api/ext/forms/", { credentials: "include" });
      const body = await res.json();
      setForms(body.rows || []);
    } catch {
      toast.error("Failed to load forms");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchForms();
  }, []);

  const filteredForms = forms.filter(
    (f) =>
      f.name.toLowerCase().includes(search.toLowerCase()) ||
      f.slug.toLowerCase().includes(search.toLowerCase()),
  );

  function openDeleteDialog(form: Form) {
    setDeletingForm(form);
    setShowDelete(true);
  }

  const handleDelete = async () => {
    if (!deletingForm) return;
    setDeleting(true);
    try {
      const res = await fetch(`/admin/api/ext/forms/${deletingForm.id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (res.ok) {
        toast.success("Form deleted");
        setShowDelete(false);
        setDeletingForm(null);
        fetchForms();
      } else {
        toast.error("Failed to delete form");
      }
    } catch {
      toast.error("Failed to delete form");
    } finally {
      setDeleting(false);
    }
  };

  const handleDuplicate = async (form: Form) => {
    setDuplicating(form.id);
    try {
      const res = await fetch(`/admin/api/ext/forms/${form.id}/duplicate`, {
        method: "POST",
        credentials: "include",
      });
      if (res.ok) {
        const data = await res.json();
        toast.success(`"${data.name}" created`);
        fetchForms();
      } else {
        toast.error("Failed to duplicate form");
      }
    } catch {
      toast.error("Failed to duplicate form");
    } finally {
      setDuplicating(null);
    }
  };

  const handleImportFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      const text = await file.text();
      const res = await fetch("/admin/api/ext/forms/import", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: text,
      });
      if (res.ok) {
        const data = await res.json();
        toast.success(`"${data.name}" imported`);
        fetchForms();
      } else {
        const err = await res.json();
        toast.error(err.message || "Import failed");
      }
    } catch {
      toast.error("Import failed");
    } finally {
      e.target.value = "";
    }
  };

  return (
    <ListPageShell>
      <input
        ref={importInputRef}
        type="file"
        accept="application/json"
        className="hidden"
        onChange={handleImportFile}
      />
      <ListHeader
        title="Forms"
        count={forms.length}
        tabs={[{ value: "all", label: "All", count: forms.length }]}
        activeTab="all"
        onTabChange={() => {}}
        newLabel="New Form"
        onNew={() => navigate("/admin/ext/forms/new")}
        extra={
          <Button
            variant="outline"
            size="sm"
            onClick={() => importInputRef.current?.click()}
            className="h-[26px] px-2.5 text-[12px]"
          >
            <Upload className="h-3 w-3 mr-1.5" /> Import
          </Button>
        }
      />

      <ListToolbar>
        <ListSearch value={search} onChange={setSearch} placeholder="Search forms…" />
      </ListToolbar>

      <ListCard>
        {loading ? (
          <LoadingRow />
        ) : filteredForms.length === 0 ? (
          <EmptyState
            icon={FileText}
            title={
              forms.length === 0 ? "No forms yet" : "No forms match your search"
            }
            description={
              forms.length === 0
                ? 'Click "New Form" to create your first form'
                : "Try adjusting your search"
            }
            action={
              forms.length === 0 ? (
                <Button size="sm" onClick={() => navigate("/admin/ext/forms/new")}>
                  New Form
                </Button>
              ) : undefined
            }
          />
        ) : (
          <ListTable minWidth={800}>
            <thead>
              <tr>
                <Th width={280}>Name</Th>
                <Th>Shortcode</Th>
                <Th width={120}>Created</Th>
                <Th width={100} align="center">Submissions</Th>
                <Th width={130}>Last Submission</Th>
                <Th width={100} align="right">Actions</Th>
              </tr>
            </thead>
            <tbody>
              {filteredForms.map((form) => (
                <Tr key={form.id}>
                  <Td>
                    <TitleCell
                      title={form.name}
                      to={`/admin/ext/forms/edit/${form.id}`}
                      slug={form.slug}
                    />
                  </Td>
                  <Td>
                    <Chip>[form slug=&quot;{form.slug}&quot;]</Chip>
                  </Td>
                  <Td className="font-mono text-[12px] text-slate-500">
                    {new Date(form.created_at).toLocaleDateString()}
                  </Td>
                  <Td align="center">
                    {form.submission_count != null && form.submission_count > 0 ? (
                      <button
                        type="button"
                        onClick={() =>
                          navigate(`/admin/ext/forms/submissions?form_id=${form.id}`)
                        }
                        className="inline-flex items-center justify-center h-5 min-w-[20px] px-1.5 rounded-full bg-indigo-100 text-indigo-700 text-[11px] font-semibold hover:bg-indigo-200 transition-colors cursor-pointer border-0"
                      >
                        {form.submission_count}
                      </button>
                    ) : (
                      <span className="text-slate-300 text-[12px]">—</span>
                    )}
                  </Td>
                  <Td className="text-[12px] text-slate-500">
                    {relativeTime(form.last_submission_at)}
                  </Td>
                  <Td align="right">
                    <RowActions
                      editTo={`/admin/ext/forms/edit/${form.id}`}
                      onDelete={() => openDeleteDialog(form)}
                      extra={
                        <>
                          <button
                            type="button"
                            onClick={() =>
                              navigate(
                                `/admin/ext/forms/submissions?form_id=${form.id}`,
                              )
                            }
                            title="View Submissions"
                            className="w-[26px] h-[26px] grid place-items-center text-slate-500 hover:bg-slate-100 hover:border-slate-200 border border-transparent rounded-[2px] cursor-pointer bg-transparent"
                          >
                            <ExternalLink className="w-3 h-3" />
                          </button>
                          <button
                            type="button"
                            onClick={() => handleDuplicate(form)}
                            disabled={duplicating === form.id}
                            title="Duplicate Form"
                            className="w-[26px] h-[26px] grid place-items-center text-slate-500 hover:bg-slate-100 hover:border-slate-200 border border-transparent rounded-[2px] cursor-pointer bg-transparent disabled:opacity-40"
                          >
                            <Copy className="w-3 h-3" />
                          </button>
                        </>
                      }
                    />
                  </Td>
                </Tr>
              ))}
            </tbody>
          </ListTable>
        )}
      </ListCard>

      <Dialog open={showDelete} onOpenChange={setShowDelete}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Form</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deletingForm?.name}&quot;? This action
              cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDelete(false)} disabled={deleting}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDelete} disabled={deleting}>
              {deleting ? "Deleting…" : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </ListPageShell>
  );
}
