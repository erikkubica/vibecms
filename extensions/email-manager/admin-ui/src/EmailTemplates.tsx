import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Pencil, Trash2, Loader2, Mail } from "@vibecms/icons";
import {
  ListHeader,
  ListToolbar,
  ListSearch,
  Button,
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
} from "@vibecms/ui";
import { toast } from "sonner";
import { getEmailTemplates, deleteEmailTemplate } from "@vibecms/api";

interface EmailTemplate {
  id: number;
  slug: string;
  name: string;
  language_id: number | null;
  subject_template: string;
  body_template: string;
  test_data: Record<string, unknown>;
}

export default function EmailTemplates() {
  const [templates, setTemplates] = useState<EmailTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");

  const [showDelete, setShowDelete] = useState(false);
  const [deletingTemplate, setDeletingTemplate] = useState<EmailTemplate | null>(null);
  const [deleting, setDeleting] = useState(false);

  async function fetchTemplates() {
    try {
      const data = await getEmailTemplates();
      setTemplates(data);
    } catch {
      toast.error("Failed to load email templates");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    fetchTemplates();
  }, []);

  function openDeleteDialog(tpl: EmailTemplate) {
    setDeletingTemplate(tpl);
    setShowDelete(true);
  }

  async function handleDelete() {
    if (!deletingTemplate) return;
    setDeleting(true);
    try {
      await deleteEmailTemplate(deletingTemplate.id);
      toast.success("Email template deleted successfully");
      setShowDelete(false);
      setDeletingTemplate(null);
      await fetchTemplates();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to delete email template";
      toast.error(message);
    } finally {
      setDeleting(false);
    }
  }

  const q = search.toLowerCase();
  const filtered = q
    ? templates.filter(
        (t) =>
          t.name.toLowerCase().includes(q) ||
          t.slug.toLowerCase().includes(q) ||
          t.subject_template.toLowerCase().includes(q),
      )
    : templates;

  return (
    <div className="w-full pb-8">
      <ListHeader
        title="Email Templates"
        tabs={[{ value: "all", label: "All", count: templates.length }]}
        activeTab="all"
        newLabel="Add Template"
        newHref="/admin/ext/email-manager/templates/new"
      />

      <ListToolbar>
        <ListSearch value={search} onChange={setSearch} placeholder="Search templates…" />
      </ListToolbar>

      <div className="bg-white border border-slate-200 rounded-lg shadow-sm overflow-hidden">
        {loading ? (
          <div className="flex h-64 items-center justify-center">
            <Loader2 className="h-6 w-6 animate-spin text-indigo-500" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex h-64 flex-col items-center justify-center gap-3 text-slate-400">
            <Mail className="h-12 w-12" />
            <p className="text-[15px] font-medium text-slate-600">No email templates found</p>
            {templates.length === 0 && (
              <p className="text-[13px] text-slate-400">Click &ldquo;Add Template&rdquo; to get started.</p>
            )}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <Table className="w-full border-separate border-spacing-0">
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="px-3 py-2.5 bg-slate-50 border-b border-slate-200 text-[10.5px] font-semibold uppercase tracking-[0.06em] text-slate-500 whitespace-nowrap">Name</TableHead>
                  <TableHead className="px-3 py-2.5 bg-slate-50 border-b border-slate-200 text-[10.5px] font-semibold uppercase tracking-[0.06em] text-slate-500 whitespace-nowrap">Slug</TableHead>
                  <TableHead className="px-3 py-2.5 bg-slate-50 border-b border-slate-200 text-[10.5px] font-semibold uppercase tracking-[0.06em] text-slate-500 whitespace-nowrap">Subject</TableHead>
                  <TableHead className="px-3 py-2.5 bg-slate-50 border-b border-slate-200 text-[10.5px] font-semibold uppercase tracking-[0.06em] text-slate-500 whitespace-nowrap text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filtered.map((tpl) => (
                  <TableRow key={tpl.id} className="group bg-white hover:bg-slate-50">
                    <TableCell className="px-3 py-2.5 border-b border-slate-100 text-[13px] font-medium text-slate-900">
                      {tpl.name}
                    </TableCell>
                    <TableCell className="px-3 py-2.5 border-b border-slate-100">
                      <span className="font-mono text-[11px] text-indigo-600">{tpl.slug}</span>
                    </TableCell>
                    <TableCell className="px-3 py-2.5 border-b border-slate-100 text-[13px] text-slate-600 max-w-xs truncate">
                      {tpl.subject_template}
                    </TableCell>
                    <TableCell className="px-3 py-2.5 border-b border-slate-100 text-right whitespace-nowrap">
                      <div className="inline-flex gap-0.5 opacity-55 group-hover:opacity-100 transition-opacity">
                        <Link
                          to={`/admin/ext/email-manager/templates/${tpl.id}`}
                          title="Edit"
                          className="w-[26px] h-[26px] grid place-items-center text-slate-500 hover:bg-slate-100 hover:border-slate-200 border border-transparent rounded-[2px]"
                        >
                          <Pencil className="w-3 h-3" />
                        </Link>
                        <button
                          type="button"
                          title="Delete"
                          onClick={() => openDeleteDialog(tpl)}
                          className="w-[26px] h-[26px] grid place-items-center text-red-500/80 hover:text-red-600 hover:bg-red-50 hover:border-red-200 border border-transparent rounded-[2px] cursor-pointer bg-transparent"
                        >
                          <Trash2 className="w-3 h-3" />
                        </button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>

      <Dialog open={showDelete} onOpenChange={setShowDelete}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Email Template</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deletingTemplate?.name}&quot;? This action cannot be undone.
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
