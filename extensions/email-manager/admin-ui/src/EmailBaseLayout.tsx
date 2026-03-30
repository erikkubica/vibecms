import { useState, useEffect } from "react";
import { ArrowLeft, Save, Loader2, Eye, Plus, Trash2, Globe, RotateCcw, Pencil } from "@vibecms/icons";
import {
  Button,
  Input,
  Label,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Textarea,
  Badge,
} from "@vibecms/ui";
import { toast } from "sonner";
import { getLanguages } from "@vibecms/api";

interface EmailLayout {
  id: number;
  name: string;
  language_id: number | null;
  body_template: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

interface Language {
  id: number;
  code: string;
  name: string;
  flag: string;
}

const API_BASE = "/admin/api/ext/email-manager";

async function fetchLayouts(): Promise<EmailLayout[]> {
  const res = await fetch(`${API_BASE}/layouts`, { credentials: "include" });
  if (!res.ok) throw new Error("Failed to fetch layouts");
  const json = await res.json();
  return json.data || [];
}

async function fetchLayout(id: number): Promise<EmailLayout> {
  const res = await fetch(`${API_BASE}/layouts/${id}`, { credentials: "include" });
  if (!res.ok) throw new Error("Failed to fetch layout");
  const json = await res.json();
  return json.data;
}

async function saveLayout(id: number | null, data: Partial<EmailLayout>): Promise<EmailLayout> {
  const url = id ? `${API_BASE}/layouts/${id}` : `${API_BASE}/layouts`;
  const method = id ? "PUT" : "POST";
  const res = await fetch(url, {
    method,
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err?.error?.message || "Failed to save layout");
  }
  const json = await res.json();
  return json.data;
}

async function deleteLayout(id: number): Promise<void> {
  const res = await fetch(`${API_BASE}/layouts/${id}`, {
    method: "DELETE",
    credentials: "include",
  });
  if (!res.ok) throw new Error("Failed to delete layout");
}

const DEFAULT_LAYOUT = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=0" />
</head>
<body style="margin:0; padding:0; background-color:#f1f5f9; font-family:-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f1f5f9; padding:32px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width:600px; width:100%; border-radius:12px; overflow:hidden; box-shadow:0 1px 3px rgba(0,0,0,0.1);">
          <!-- Header -->
          <tr>
            <td style="background-color:#2563eb; padding:24px 32px; text-align:center;">
              <h1 style="margin:0; color:#ffffff; font-size:22px; font-weight:600; letter-spacing:-0.02em;">{{.site.site_name}}</h1>
            </td>
          </tr>
          <!-- Content -->
          <tr>
            <td style="background-color:#ffffff; padding:32px;">
              {{.email_body}}
            </td>
          </tr>
          <!-- Footer -->
          <tr>
            <td style="background-color:#f8fafc; padding:20px 32px; text-align:center; border-top:1px solid #e2e8f0;">
              <p style="margin:0; color:#94a3b8; font-size:13px;">&copy; {{.site.site_name}}</p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`;

const SAMPLE_EMAIL_BODY = `<h2 style="margin:0 0 16px; color:#1e293b; font-size:20px;">Welcome!</h2>
<p style="margin:0 0 12px; color:#475569; font-size:15px; line-height:1.6;">
  This is a sample email body that will replace <code>{{.email_body}}</code> in the layout.
</p>
<p style="margin:0; color:#475569; font-size:15px; line-height:1.6;">
  Use base layouts to wrap all outgoing emails with consistent branding.
</p>`;

export default function EmailBaseLayout() {
  const [layouts, setLayouts] = useState<EmailLayout[]>([]);
  const [languages, setLanguages] = useState<Language[]>([]);
  const [loading, setLoading] = useState(true);

  // Editor state
  const [editing, setEditing] = useState(false);
  const [editId, setEditId] = useState<number | null>(null);
  const [saving, setSaving] = useState(false);

  // Form state
  const [formName, setFormName] = useState("");
  const [formLanguageId, setFormLanguageId] = useState<string>("__universal__");
  const [formBody, setFormBody] = useState("");

  async function loadLayouts() {
    try {
      const data = await fetchLayouts();
      setLayouts(data);
    } catch {
      toast.error("Failed to load base layouts");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadLayouts();
    getLanguages()
      .then((langs: Language[]) => setLanguages(langs))
      .catch(() => {});
  }, []);

  function openEditor(layout?: EmailLayout) {
    if (layout) {
      setEditId(layout.id);
      setFormName(layout.name);
      setFormLanguageId(layout.language_id ? String(layout.language_id) : "__universal__");
      setFormBody(layout.body_template);
    } else {
      setEditId(null);
      setFormName("");
      setFormLanguageId("__universal__");
      setFormBody(DEFAULT_LAYOUT);
    }
    setEditing(true);
  }

  function closeEditor() {
    setEditing(false);
    setEditId(null);
    setFormName("");
    setFormLanguageId("__universal__");
    setFormBody("");
  }

  async function handleSave() {
    if (!formName.trim()) {
      toast.error("Name is required");
      return;
    }

    setSaving(true);
    try {
      const data: Partial<EmailLayout> = {
        name: formName.trim(),
        language_id: formLanguageId === "__universal__" ? null : Number(formLanguageId),
        body_template: formBody,
      };
      await saveLayout(editId, data);
      toast.success(editId ? "Layout updated successfully" : "Layout created successfully");
      closeEditor();
      await loadLayouts();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to save layout";
      toast.error(message);
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(layout: EmailLayout) {
    if (layout.is_default) return;
    if (!confirm(`Delete layout "${layout.name}"? This action cannot be undone.`)) return;
    try {
      await deleteLayout(layout.id);
      toast.success("Layout deleted successfully");
      if (editing && editId === layout.id) {
        closeEditor();
      }
      await loadLayouts();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to delete layout";
      toast.error(message);
    }
  }

  function getPreviewHtml(): string {
    let html = formBody;
    html = html.replace(/\{\{\s*\.email_body\s*\}\}/g, SAMPLE_EMAIL_BODY);
    html = html.replace(/\{\{\s*\.site\.site_name\s*\}\}/g, "My Site");
    return html;
  }

  function getLanguageLabel(languageId: number | null): React.ReactNode {
    if (!languageId) {
      return (
        <span className="flex items-center gap-1.5 text-slate-500">
          <Globe className="h-3.5 w-3.5" />
          Universal
        </span>
      );
    }
    const lang = languages.find((l) => l.id === languageId);
    if (!lang) return <span className="text-slate-400">Unknown</span>;
    return (
      <span>
        {lang.flag} {lang.name}
      </span>
    );
  }

  function formatDate(dateStr: string): string {
    try {
      return new Date(dateStr).toLocaleDateString(undefined, {
        year: "numeric",
        month: "short",
        day: "numeric",
      });
    } catch {
      return dateStr;
    }
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
      </div>
    );
  }

  // --- Editor View ---
  if (editing) {
    const currentLayout = editId ? layouts.find((l) => l.id === editId) : null;

    return (
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={closeEditor}
            >
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <h1 className="text-2xl font-bold text-slate-900">
              {editId ? formName || "Edit Layout" : "New Layout"}
            </h1>
          </div>
          <div className="flex items-center gap-2">
            {!editId || !formBody.trim() ? (
              <Button
                variant="outline"
                className="rounded-lg"
                onClick={() => setFormBody(DEFAULT_LAYOUT)}
              >
                <RotateCcw className="mr-2 h-4 w-4" />
                Reset to Default
              </Button>
            ) : null}
            {editId && currentLayout && !currentLayout.is_default && (
              <Button
                variant="outline"
                className="rounded-lg text-red-600 hover:text-red-700 hover:bg-red-50"
                onClick={() => handleDelete(currentLayout)}
              >
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </Button>
            )}
            <Button
              onClick={handleSave}
              disabled={saving}
              className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium"
            >
              <Save className="mr-2 h-4 w-4" />
              {saving ? "Saving..." : "Save"}
            </Button>
          </div>
        </div>

        {/* Form Fields */}
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <CardHeader>
            <CardTitle className="text-base font-semibold text-slate-800">Layout Details</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="layout-name" className="text-sm font-medium text-slate-700">
                  Name
                </Label>
                <Input
                  id="layout-name"
                  placeholder="e.g. Default Layout"
                  value={formName}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setFormName(e.target.value)}
                  required
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>
              <div className="space-y-2">
                <Label className="text-sm font-medium text-slate-700">Language</Label>
                <Select value={formLanguageId} onValueChange={setFormLanguageId}>
                  <SelectTrigger className="rounded-lg border-slate-300">
                    <SelectValue placeholder="Universal" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__universal__">Universal (fallback)</SelectItem>
                    {languages.map((lang) => (
                      <SelectItem key={lang.id} value={String(lang.id)}>
                        {lang.flag} {lang.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Split pane: Body Template + Preview */}
        <div className="grid grid-cols-2 gap-4">
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-base font-semibold text-slate-800">Body Template</CardTitle>
            </CardHeader>
            <CardContent>
              <Textarea
                value={formBody}
                onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setFormBody(e.target.value)}
                className="min-h-[500px] font-mono text-sm rounded-lg border-slate-300"
                placeholder="<!DOCTYPE html>..."
              />
            </CardContent>
          </Card>
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <div className="flex items-center gap-2">
                <Eye className="h-4 w-4 text-slate-500" />
                <CardTitle className="text-base font-semibold text-slate-800">Preview</CardTitle>
              </div>
            </CardHeader>
            <CardContent>
              <div className="rounded-lg border border-slate-300 bg-white overflow-auto" style={{ height: "500px" }}>
                <iframe
                  srcDoc={getPreviewHtml()}
                  title="Layout Preview"
                  className="w-full h-full border-0"
                  sandbox=""
                />
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    );
  }

  // --- List View ---
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Base Layouts</h1>
          <p className="mt-1 text-sm text-slate-500">
            HTML wrappers applied to outgoing emails. The universal layout is used as fallback when no language-specific layout exists.
          </p>
        </div>
        <Button
          className="bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg shadow-sm"
          onClick={() => openEditor()}
        >
          <Plus className="mr-2 h-4 w-4" />
          New Layout
        </Button>
      </div>

      {/* Layout Cards */}
      {layouts.length === 0 ? (
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <CardContent className="py-12 text-center text-slate-400">
            No base layouts found. Click "New Layout" to get started.
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {layouts.map((layout) => (
            <Card key={layout.id} className="rounded-xl border border-slate-200 shadow-sm">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base font-semibold text-slate-800">
                    {layout.name}
                  </CardTitle>
                  {layout.is_default && (
                    <Badge variant="secondary" className="bg-indigo-50 text-indigo-700 border-indigo-200">
                      Default
                    </Badge>
                  )}
                </div>
                <CardDescription className="text-sm text-slate-500">
                  {getLanguageLabel(layout.language_id)}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="flex items-center justify-between">
                  <div className="text-xs text-slate-400 space-y-0.5">
                    <div>Created: {formatDate(layout.created_at)}</div>
                    <div>Updated: {formatDate(layout.updated_at)}</div>
                  </div>
                  <div className="flex items-center gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-slate-500 hover:text-indigo-600"
                      onClick={() => openEditor(layout)}
                    >
                      <Pencil className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-slate-500 hover:text-red-600"
                      disabled={layout.is_default}
                      onClick={() => handleDelete(layout)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
