import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Pencil, Trash2, Loader2, Settings } from "@vibecms/icons";
import {
  ListHeader,
  ListToolbar,
  ListSearch,
  Button,
  Badge,
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
import {
  getEmailRules,
  updateEmailRule,
  deleteEmailRule,
  getEmailTemplates,
  getSystemActions,
} from "@vibecms/api";

interface EmailTemplate {
  id: number;
  name: string;
}

interface SystemAction {
  slug: string;
  label: string;
}

interface EmailRule {
  id: number;
  action: string;
  node_type: string | null;
  template_id: number;
  recipient_type: string;
  recipient_value: string;
  enabled: boolean;
}

export default function EmailRules() {
  const [rules, setRules] = useState<EmailRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");

  const [templates, setTemplates] = useState<EmailTemplate[]>([]);
  const [actions, setActions] = useState<SystemAction[]>([]);

  const [showDelete, setShowDelete] = useState(false);
  const [deletingRule, setDeletingRule] = useState<EmailRule | null>(null);
  const [deleting, setDeleting] = useState(false);

  async function fetchRules() {
    try {
      const data = await getEmailRules();
      setRules(data);
    } catch {
      toast.error("Failed to load email rules");
    } finally {
      setLoading(false);
    }
  }

  async function fetchLookups() {
    try {
      const [tpls, acts] = await Promise.all([getEmailTemplates(), getSystemActions()]);
      setTemplates(tpls);
      setActions(acts);
    } catch {
      // Non-fatal
    }
  }

  useEffect(() => {
    fetchRules();
    fetchLookups();
  }, []);

  async function handleToggleEnabled(rule: EmailRule) {
    try {
      await updateEmailRule(rule.id, { enabled: !rule.enabled });
      setRules((prev) => prev.map((r) => r.id === rule.id ? { ...r, enabled: !r.enabled } : r));
      toast.success(`Rule ${!rule.enabled ? "enabled" : "disabled"}`);
    } catch {
      toast.error("Failed to update rule");
    }
  }

  function openDeleteDialog(rule: EmailRule) {
    setDeletingRule(rule);
    setShowDelete(true);
  }

  async function handleDelete() {
    if (!deletingRule) return;
    setDeleting(true);
    try {
      await deleteEmailRule(deletingRule.id);
      toast.success("Email rule deleted successfully");
      setShowDelete(false);
      setDeletingRule(null);
      await fetchRules();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to delete email rule";
      toast.error(message);
    } finally {
      setDeleting(false);
    }
  }

  function getTemplateName(id: number): string {
    return templates.find((t) => t.id === id)?.name || `Template #${id}`;
  }

  function getActionLabel(slug: string): string {
    return actions.find((a) => a.slug === slug)?.label || slug;
  }

  const q = search.toLowerCase();
  const filtered = q
    ? rules.filter(
        (r) =>
          getActionLabel(r.action).toLowerCase().includes(q) ||
          (r.node_type || "").toLowerCase().includes(q) ||
          getTemplateName(r.template_id).toLowerCase().includes(q) ||
          r.recipient_value.toLowerCase().includes(q),
      )
    : rules;

  return (
    <div className="w-full pb-8">
      <ListHeader
        title="Email Rules"
        tabs={[{ value: "all", label: "All", count: rules.length }]}
        activeTab="all"
        newLabel="Add Rule"
        newHref="/admin/ext/email-manager/rules/new"
      />

      <ListToolbar>
        <ListSearch value={search} onChange={setSearch} placeholder="Search rules…" />
      </ListToolbar>

      <div className="bg-white border border-slate-200 rounded-lg shadow-sm overflow-hidden">
        {loading ? (
          <div className="flex h-64 items-center justify-center">
            <Loader2 className="h-6 w-6 animate-spin text-indigo-500" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex h-64 flex-col items-center justify-center gap-3 text-slate-400">
            <Settings className="h-12 w-12" />
            <p className="text-[15px] font-medium text-slate-600">No email rules found</p>
            {rules.length === 0 && (
              <p className="text-[13px] text-slate-400">Click &ldquo;Add Rule&rdquo; to get started.</p>
            )}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <Table className="w-full border-separate border-spacing-0">
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="px-3 py-2.5 bg-slate-50 border-b border-slate-200 text-[10.5px] font-semibold uppercase tracking-[0.06em] text-slate-500 whitespace-nowrap">Action</TableHead>
                  <TableHead className="px-3 py-2.5 bg-slate-50 border-b border-slate-200 text-[10.5px] font-semibold uppercase tracking-[0.06em] text-slate-500 whitespace-nowrap">Node Type</TableHead>
                  <TableHead className="px-3 py-2.5 bg-slate-50 border-b border-slate-200 text-[10.5px] font-semibold uppercase tracking-[0.06em] text-slate-500 whitespace-nowrap">Template</TableHead>
                  <TableHead className="px-3 py-2.5 bg-slate-50 border-b border-slate-200 text-[10.5px] font-semibold uppercase tracking-[0.06em] text-slate-500 whitespace-nowrap">Recipient</TableHead>
                  <TableHead className="px-3 py-2.5 bg-slate-50 border-b border-slate-200 text-[10.5px] font-semibold uppercase tracking-[0.06em] text-slate-500 whitespace-nowrap">Enabled</TableHead>
                  <TableHead className="px-3 py-2.5 bg-slate-50 border-b border-slate-200 text-[10.5px] font-semibold uppercase tracking-[0.06em] text-slate-500 whitespace-nowrap text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filtered.map((rule) => (
                  <TableRow key={rule.id} className="group bg-white hover:bg-slate-50">
                    <TableCell className="px-3 py-2.5 border-b border-slate-100 text-[13px] font-medium text-slate-800">
                      {getActionLabel(rule.action)}
                    </TableCell>
                    <TableCell className="px-3 py-2.5 border-b border-slate-100">
                      {rule.node_type ? (
                        <Badge variant="outline" className="text-xs">{rule.node_type}</Badge>
                      ) : (
                        <span className="text-slate-400 text-[13px]">—</span>
                      )}
                    </TableCell>
                    <TableCell className="px-3 py-2.5 border-b border-slate-100 text-[13px] text-slate-600">
                      {getTemplateName(rule.template_id)}
                    </TableCell>
                    <TableCell className="px-3 py-2.5 border-b border-slate-100">
                      <div className="flex items-center gap-2">
                        <Badge className="bg-slate-100 text-slate-700 hover:bg-slate-100 border-0 text-xs">
                          {rule.recipient_type}
                        </Badge>
                        {rule.recipient_value && (
                          <span className="text-[13px] text-slate-500">{rule.recipient_value}</span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="px-3 py-2.5 border-b border-slate-100">
                      <button
                        type="button"
                        onClick={() => handleToggleEnabled(rule)}
                        className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-500/20 ${rule.enabled ? "bg-indigo-600" : "bg-slate-300"}`}
                      >
                        <span className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${rule.enabled ? "translate-x-[18px]" : "translate-x-[2px]"}`} />
                      </button>
                    </TableCell>
                    <TableCell className="px-3 py-2.5 border-b border-slate-100 text-right whitespace-nowrap">
                      <div className="inline-flex gap-0.5 opacity-55 group-hover:opacity-100 transition-opacity">
                        <Link
                          to={`/admin/ext/email-manager/rules/${rule.id}`}
                          title="Edit"
                          className="w-[26px] h-[26px] grid place-items-center text-slate-500 hover:bg-slate-100 hover:border-slate-200 border border-transparent rounded-[2px]"
                        >
                          <Pencil className="w-3 h-3" />
                        </Link>
                        <button
                          type="button"
                          title="Delete"
                          onClick={() => openDeleteDialog(rule)}
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
            <DialogTitle>Delete Email Rule</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete this email rule? This action cannot be undone.
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
