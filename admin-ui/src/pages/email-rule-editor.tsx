import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { ArrowLeft, Save, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "sonner";
import {
  getEmailRule,
  createEmailRule,
  updateEmailRule,
  getEmailTemplates,
  getSystemActions,
  getNodeTypes,
  getRoles,
  type EmailRule,
  type EmailTemplate,
  type SystemAction,
  type NodeType,
  type Role,
} from "@/api/client";

export default function EmailRuleEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEdit = !!id;

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  // Lookup data
  const [templates, setTemplates] = useState<EmailTemplate[]>([]);
  const [actions, setActions] = useState<SystemAction[]>([]);
  const [nodeTypes, setNodeTypes] = useState<NodeType[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);

  // Form state
  const [formAction, setFormAction] = useState("");
  const [formNodeType, setFormNodeType] = useState("");
  const [formTemplateId, setFormTemplateId] = useState("");
  const [formRecipientType, setFormRecipientType] = useState("actor");
  const [formRecipientValue, setFormRecipientValue] = useState("");
  const [formEnabled, setFormEnabled] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const [tpls, acts, nts, rls] = await Promise.all([
          getEmailTemplates(),
          getSystemActions(),
          getNodeTypes(),
          getRoles(),
        ]);
        if (cancelled) return;
        setTemplates(tpls);
        setActions(acts);
        setNodeTypes(nts);
        setRoles(rls);

        if (isEdit) {
          const rule = await getEmailRule(Number(id));
          if (cancelled) return;
          setFormAction(rule.action);
          setFormNodeType(rule.node_type || "");
          setFormTemplateId(String(rule.template_id));
          setFormRecipientType(rule.recipient_type);
          setFormRecipientValue(rule.recipient_value);
          setFormEnabled(rule.enabled);
        }
      } catch {
        if (!cancelled) {
          toast.error("Failed to load data");
          navigate("/admin/email-rules", { replace: true });
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, [id, isEdit, navigate]);

  async function handleSave(e: FormEvent) {
    e.preventDefault();

    if (!formAction || !formTemplateId) {
      toast.error("Action and template are required");
      return;
    }

    const data: Partial<EmailRule> = {
      action: formAction,
      node_type: formNodeType || null,
      template_id: Number(formTemplateId),
      recipient_type: formRecipientType,
      recipient_value: formRecipientValue,
      enabled: formEnabled,
    };

    setSaving(true);
    try {
      if (isEdit) {
        await updateEmailRule(Number(id), data);
        toast.success("Email rule updated successfully");
      } else {
        const created = await createEmailRule(data);
        toast.success("Email rule created successfully");
        navigate(`/admin/email-rules/${created.id}/edit`, { replace: true });
      }
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to save email rule";
      toast.error(message);
    } finally {
      setSaving(false);
    }
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
          <Button variant="ghost" size="icon" asChild className="h-8 w-8">
            <Link to="/admin/email-rules">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <h1 className="text-2xl font-bold text-slate-900">
            {isEdit ? "Edit Email Rule" : "New Email Rule"}
          </h1>
        </div>
        <Button
          onClick={handleSave}
          disabled={saving}
          className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium"
        >
          <Save className="mr-2 h-4 w-4" />
          {saving ? "Saving..." : "Save"}
        </Button>
      </div>

      <form onSubmit={handleSave} className="space-y-6">
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <CardHeader>
            <CardTitle className="text-base font-semibold text-slate-800">Rule Configuration</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label className="text-sm font-medium text-slate-700">Action</Label>
              <Select value={formAction} onValueChange={setFormAction}>
                <SelectTrigger className="rounded-lg border-slate-300">
                  <SelectValue placeholder="Select an action..." />
                </SelectTrigger>
                <SelectContent>
                  {actions.map((act) => (
                    <SelectItem key={act.slug} value={act.slug}>
                      {act.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label className="text-sm font-medium text-slate-700">Node Type (optional)</Label>
              <Select value={formNodeType || "__all__"} onValueChange={(v) => setFormNodeType(v === "__all__" ? "" : v)}>
                <SelectTrigger className="rounded-lg border-slate-300">
                  <SelectValue placeholder="All types" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__all__">All types</SelectItem>
                  {nodeTypes.map((nt) => (
                    <SelectItem key={nt.slug} value={nt.slug}>
                      {nt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label className="text-sm font-medium text-slate-700">Template</Label>
              <Select value={formTemplateId} onValueChange={setFormTemplateId}>
                <SelectTrigger className="rounded-lg border-slate-300">
                  <SelectValue placeholder="Select a template..." />
                </SelectTrigger>
                <SelectContent>
                  {templates.map((tpl) => (
                    <SelectItem key={tpl.id} value={String(tpl.id)}>
                      {tpl.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label className="text-sm font-medium text-slate-700">Recipient Type</Label>
                <Select value={formRecipientType} onValueChange={(v) => {
                  setFormRecipientType(v);
                  setFormRecipientValue("");
                }}>
                  <SelectTrigger className="rounded-lg border-slate-300">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="actor">Actor (triggering user)</SelectItem>
                    <SelectItem value="node_author">Node Author</SelectItem>
                    <SelectItem value="role">Role</SelectItem>
                    <SelectItem value="fixed">Fixed Email(s)</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {formRecipientType === "role" && (
                <div className="space-y-2">
                  <Label className="text-sm font-medium text-slate-700">Role</Label>
                  <Select value={formRecipientValue} onValueChange={setFormRecipientValue}>
                    <SelectTrigger className="rounded-lg border-slate-300">
                      <SelectValue placeholder="Select a role..." />
                    </SelectTrigger>
                    <SelectContent>
                      {roles.map((role) => (
                        <SelectItem key={role.slug} value={role.slug}>
                          {role.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}

              {formRecipientType === "fixed" && (
                <div className="space-y-2">
                  <Label className="text-sm font-medium text-slate-700">Email Address(es)</Label>
                  <Input
                    placeholder="email1@example.com, email2@example.com"
                    value={formRecipientValue}
                    onChange={(e) => setFormRecipientValue(e.target.value)}
                    className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                  />
                </div>
              )}
            </div>

            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={formEnabled}
                onChange={(e) => setFormEnabled(e.target.checked)}
                className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
              />
              <span className="text-sm font-medium text-slate-700">Enabled</span>
            </label>
          </CardContent>
        </Card>
      </form>
    </div>
  );
}
