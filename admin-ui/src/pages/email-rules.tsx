import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Mail, Plus, Pencil, Trash2, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { toast } from "sonner";
import {
  getEmailRules,
  updateEmailRule,
  deleteEmailRule,
  getEmailTemplates,
  getSystemActions,
  type EmailRule,
  type EmailTemplate,
  type SystemAction,
} from "@/api/client";

export default function EmailRulesPage() {
  const [rules, setRules] = useState<EmailRule[]>([]);
  const [loading, setLoading] = useState(true);

  // Lookup data
  const [templates, setTemplates] = useState<EmailTemplate[]>([]);
  const [actions, setActions] = useState<SystemAction[]>([]);

  // Delete confirmation
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
      const [tpls, acts] = await Promise.all([
        getEmailTemplates(),
        getSystemActions(),
      ]);
      setTemplates(tpls);
      setActions(acts);
    } catch {
      // Non-fatal: lookups may partially fail
    }
  }

  useEffect(() => {
    fetchRules();
    fetchLookups();
  }, []);

  async function handleToggleEnabled(rule: EmailRule) {
    try {
      await updateEmailRule(rule.id, { enabled: !rule.enabled });
      setRules((prev) =>
        prev.map((r) =>
          r.id === rule.id ? { ...r, enabled: !r.enabled } : r
        )
      );
      toast.success(
        `Rule ${!rule.enabled ? "enabled" : "disabled"}`
      );
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
      const message =
        err instanceof Error ? err.message : "Failed to delete email rule";
      toast.error(message);
    } finally {
      setDeleting(false);
    }
  }

  function getTemplateName(templateId: number): string {
    const tpl = templates.find((t) => t.id === templateId);
    return tpl?.name || `Template #${templateId}`;
  }

  function getActionLabel(actionSlug: string): string {
    const act = actions.find((a) => a.slug === actionSlug);
    return act?.label || actionSlug;
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Mail className="h-7 w-7 text-indigo-600" />
          <h1 className="text-2xl font-bold text-slate-900">Email Rules</h1>
        </div>
        <Button
          className="bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg shadow-sm"
          asChild
        >
          <Link to="/admin/email-rules/new">
            <Plus className="mr-2 h-4 w-4" />
            Add Rule
          </Link>
        </Button>
      </div>

      {/* Table */}
      <Card className="rounded-xl border border-slate-200 shadow-sm">
        <CardHeader>
          <CardTitle className="text-lg font-semibold text-slate-900">
            All Email Rules
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow className="border-slate-200 hover:bg-transparent">
                <TableHead className="text-slate-500 font-medium">Action</TableHead>
                <TableHead className="text-slate-500 font-medium">Node Type</TableHead>
                <TableHead className="text-slate-500 font-medium">Template</TableHead>
                <TableHead className="text-slate-500 font-medium">Recipient</TableHead>
                <TableHead className="text-slate-500 font-medium">Enabled</TableHead>
                <TableHead className="text-slate-500 font-medium text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rules.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-12 text-slate-400">
                    No email rules configured yet. Click &quot;Add Rule&quot; to get started.
                  </TableCell>
                </TableRow>
              )}
              {rules.map((rule) => (
                <TableRow key={rule.id} className="border-slate-100">
                  <TableCell className="font-medium text-slate-800">
                    {getActionLabel(rule.action)}
                  </TableCell>
                  <TableCell>
                    {rule.node_type ? (
                      <Badge variant="outline" className="text-xs">
                        {rule.node_type}
                      </Badge>
                    ) : (
                      <span className="text-slate-400 text-sm">All</span>
                    )}
                  </TableCell>
                  <TableCell className="text-slate-600">
                    {getTemplateName(rule.template_id)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Badge className="bg-slate-100 text-slate-700 hover:bg-slate-100 border-0 text-xs">
                        {rule.recipient_type}
                      </Badge>
                      {rule.recipient_value && (
                        <span className="text-sm text-slate-500">{rule.recipient_value}</span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <button
                      type="button"
                      onClick={() => handleToggleEnabled(rule)}
                      className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-500/20 ${
                        rule.enabled ? "bg-indigo-600" : "bg-slate-300"
                      }`}
                    >
                      <span
                        className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                          rule.enabled ? "translate-x-6" : "translate-x-1"
                        }`}
                      />
                    </button>
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-slate-500 hover:text-indigo-600"
                        asChild
                      >
                        <Link to={`/admin/email-rules/${rule.id}/edit`}>
                          <Pencil className="h-4 w-4" />
                        </Link>
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-slate-500 hover:text-red-600"
                        onClick={() => openDeleteDialog(rule)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Delete confirmation dialog */}
      <Dialog open={showDelete} onOpenChange={setShowDelete}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Email Rule</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete this email rule? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDelete(false)}
              disabled={deleting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleting}
            >
              {deleting ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
