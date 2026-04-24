import React, { useEffect, useState } from "react";
import { Search, Edit2, Trash2, FileText, ExternalLink } from "@vibecms/icons";

const {
  ListHeader,
  ListToolbar,
  ListSearch,
  Button,
  Card,
  CardContent,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
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
}

export default function FormsList() {
  const [forms, setForms] = useState<Form[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const navigate = useNavigate();

  const [showDelete, setShowDelete] = useState(false);
  const [deletingForm, setDeletingForm] = useState<Form | null>(null);
  const [deleting, setDeleting] = useState(false);

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

  return (
    <div className="w-full pb-8">
      <ListHeader
        title="Forms"
        tabs={[{ value: "all", label: "All", count: forms.length }]}
        activeTab="all"
        newLabel="Add Form"
        onNew={() => navigate("/admin/ext/forms/new")}
      />

      <ListToolbar>
        <ListSearch value={search} onChange={setSearch} placeholder="Search forms…" />
      </ListToolbar>

      <Card className="rounded-lg border border-slate-200 shadow-sm overflow-hidden">
        <CardContent className="p-0">
          <Table className="w-full border-separate border-spacing-0">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="px-3 py-2.5 bg-slate-50 border-b border-slate-200 text-[10.5px] font-semibold uppercase tracking-[0.06em] text-slate-500 whitespace-nowrap w-[300px]">
                  Name
                </TableHead>
                <TableHead className="px-3 py-2.5 bg-slate-50 border-b border-slate-200 text-[10.5px] font-semibold uppercase tracking-[0.06em] text-slate-500 whitespace-nowrap">
                  Shortcode / Slug
                </TableHead>
                <TableHead className="px-3 py-2.5 bg-slate-50 border-b border-slate-200 text-[10.5px] font-semibold uppercase tracking-[0.06em] text-slate-500 whitespace-nowrap">
                  Created
                </TableHead>
                <TableHead className="px-3 py-2.5 bg-slate-50 border-b border-slate-200 text-[10.5px] font-semibold uppercase tracking-[0.06em] text-slate-500 whitespace-nowrap text-right">
                  Actions
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={4} className="h-32 text-center text-[13px] text-slate-400">
                    Loading forms...
                  </TableCell>
                </TableRow>
              ) : filteredForms.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} className="h-32 text-center text-[13px] text-slate-400">
                    {forms.length === 0
                      ? 'No forms yet. Click "Add Form" to get started.'
                      : "No forms match your search."}
                  </TableCell>
                </TableRow>
              ) : (
                filteredForms.map((form) => (
                  <TableRow key={form.id} className="group border-slate-100 bg-white hover:bg-slate-50">
                    <TableCell className="px-3 py-2.5 border-b border-slate-100 text-[13px] font-medium text-slate-900">
                      <div className="flex items-center gap-2.5">
                        <div className="h-7 w-7 rounded bg-indigo-50 flex items-center justify-center text-indigo-600 group-hover:bg-indigo-100 transition-colors flex-shrink-0">
                          <FileText className="h-3.5 w-3.5" />
                        </div>
                        <button
                          type="button"
                          onClick={() => navigate(`/admin/ext/forms/edit/${form.id}`)}
                          className="hover:text-indigo-600 transition-colors bg-transparent border-0 p-0 cursor-pointer text-[13px] font-medium text-slate-900"
                        >
                          {form.name}
                        </button>
                      </div>
                    </TableCell>
                    <TableCell className="px-3 py-2.5 border-b border-slate-100">
                      <code className="px-1.5 py-0.5 bg-slate-100 rounded text-[11px] font-mono text-slate-600">
                        [form slug=&quot;{form.slug}&quot;]
                      </code>
                    </TableCell>
                    <TableCell className="px-3 py-2.5 border-b border-slate-100 font-mono text-[12px] text-slate-500">
                      {new Date(form.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="px-3 py-2.5 border-b border-slate-100 text-right whitespace-nowrap">
                      <div className="inline-flex gap-0.5 opacity-55 group-hover:opacity-100 transition-opacity">
                        <button
                          type="button"
                          onClick={() => navigate(`/admin/ext/forms/submissions?form_id=${form.id}`)}
                          title="View Submissions"
                          className="w-[26px] h-[26px] grid place-items-center text-slate-500 hover:bg-slate-100 hover:border-slate-200 border border-transparent rounded-[2px] cursor-pointer bg-transparent"
                        >
                          <ExternalLink className="w-3 h-3" />
                        </button>
                        <button
                          type="button"
                          onClick={() => navigate(`/admin/ext/forms/edit/${form.id}`)}
                          title="Edit Form"
                          className="w-[26px] h-[26px] grid place-items-center text-slate-500 hover:bg-slate-100 hover:border-slate-200 border border-transparent rounded-[2px] cursor-pointer bg-transparent"
                        >
                          <Edit2 className="w-3 h-3" />
                        </button>
                        <button
                          type="button"
                          onClick={() => openDeleteDialog(form)}
                          title="Delete Form"
                          className="w-[26px] h-[26px] grid place-items-center text-red-500/80 hover:text-red-600 hover:bg-red-50 hover:border-red-200 border border-transparent rounded-[2px] cursor-pointer bg-transparent"
                        >
                          <Trash2 className="w-3 h-3" />
                        </button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Dialog open={showDelete} onOpenChange={setShowDelete}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Form</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deletingForm?.name}&quot;? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDelete(false)} disabled={deleting}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDelete} disabled={deleting}>
              {deleting ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
