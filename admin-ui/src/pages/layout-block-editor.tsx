import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Save,
  Trash2,
  Loader2,
  ArrowLeft,
  Unlink,
  Info,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import { Link } from "react-router-dom";
import {
  getLayoutBlock,
  createLayoutBlock,
  updateLayoutBlock,
  deleteLayoutBlock,
  detachLayoutBlock,
  getLanguages,
  type LayoutBlock,
  type Language,
} from "@/api/client";

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, "")
    .replace(/[\s_]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export default function LayoutBlockEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isNew = !id;

  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [languages, setLanguages] = useState<Language[]>([]);
  const [showDelete, setShowDelete] = useState(false);
  const [showDetach, setShowDetach] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [detaching, setDetaching] = useState(false);
  const [slugManual, setSlugManual] = useState(false);

  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [languageCode, setLanguageCode] = useState("");
  const [templateCode, setTemplateCode] = useState("");
  const [source, setSource] = useState("custom");

  const isTheme = source === "theme";

  const fetchLayoutBlock = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const data = await getLayoutBlock(id);
      setName(data.name);
      setSlug(data.slug);
      setDescription(data.description || "");
      setLanguageCode(data.language_code || "");
      setTemplateCode(data.template_code || "");
      setSource(data.source || "custom");
      setSlugManual(true);
    } catch {
      toast.error("Failed to load layout block");
      navigate("/admin/layout-blocks");
    } finally {
      setLoading(false);
    }
  }, [id, navigate]);

  const fetchLanguages = useCallback(async () => {
    try {
      const data = await getLanguages(true);
      setLanguages(data);
      // Default to the default language if creating new
      if (!id && !languageCode) {
        const defaultLang = data.find((l: Language) => l.is_default);
        if (defaultLang) setLanguageCode(defaultLang.code);
        else if (data.length > 0) setLanguageCode(data[0].code);
      }
    } catch {
      // silent
    }
  }, []);

  useEffect(() => {
    fetchLanguages();
  }, [fetchLanguages]);

  useEffect(() => {
    fetchLayoutBlock();
  }, [fetchLayoutBlock]);

  function handleNameChange(value: string) {
    setName(value);
    if (!slugManual) {
      setSlug(slugify(value));
    }
  }

  async function handleSave() {
    if (!name.trim()) {
      toast.error("Name is required");
      return;
    }
    if (!slug.trim()) {
      toast.error("Slug is required");
      return;
    }

    setSaving(true);
    try {
      const payload: Partial<LayoutBlock> = {
        name: name.trim(),
        slug: slug.trim(),
        description: description.trim(),
        language_code: languageCode,
        template_code: templateCode,
      };

      if (isNew) {
        const created = await createLayoutBlock(payload);
        toast.success("Layout block created successfully");
        navigate(`/admin/layout-blocks/${created.id}`);
      } else {
        await updateLayoutBlock(id!, payload);
        toast.success("Layout block updated successfully");
      }
    } catch {
      toast.error(isNew ? "Failed to create layout block" : "Failed to update layout block");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!id) return;
    setDeleting(true);
    try {
      await deleteLayoutBlock(id);
      toast.success("Layout block deleted successfully");
      navigate("/admin/layout-blocks");
    } catch {
      toast.error("Failed to delete layout block");
    } finally {
      setDeleting(false);
    }
  }

  async function handleDetach() {
    if (!id) return;
    setDetaching(true);
    try {
      const detached = await detachLayoutBlock(id);
      toast.success("Layout block detached from theme");
      setSource(detached.source);
      setShowDetach(false);
    } catch {
      toast.error("Failed to detach layout block");
    } finally {
      setDetaching(false);
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
            <Link to="/admin/layout-blocks">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <h1 className="text-2xl font-bold text-slate-900">
            {isNew ? "New Layout Block" : name || "Edit Layout Block"}
          </h1>
          {isTheme && (
            <Badge className="bg-amber-100 text-amber-700 hover:bg-amber-100 border-0 text-xs">Theme</Badge>
          )}
        </div>
        <div className="flex items-center gap-2">
          {!isNew && isTheme && (
            <Button
              variant="outline"
              onClick={() => setShowDetach(true)}
              className="text-amber-600 border-amber-300 hover:bg-amber-50"
            >
              <Unlink className="mr-2 h-4 w-4" />
              Detach
            </Button>
          )}
          {!isNew && !isTheme && (
            <Button
              variant="outline"
              className="text-red-500 border-red-300 hover:bg-red-50"
              onClick={() => setShowDelete(true)}
            >
              <Trash2 className="mr-2 h-4 w-4" />
              Delete
            </Button>
          )}
          <Button
            onClick={handleSave}
            disabled={saving || isTheme}
            className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium"
          >
            {saving ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Save className="mr-2 h-4 w-4" />
            )}
            {saving ? "Saving..." : "Save"}
          </Button>
        </div>
      </div>

      {isTheme && (
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700 flex items-start gap-2">
          <Info className="h-4 w-4 mt-0.5 shrink-0" />
          <p>
            This layout block is managed by the active theme and is read-only. To customize it, click
            &quot;Detach&quot; to create an editable copy.
          </p>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main form */}
        <div className="lg:col-span-2 space-y-6">
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-base font-semibold text-slate-900">Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="name">Name</Label>
                <Input
                  id="name"
                  value={name}
                  onChange={(e) => handleNameChange(e.target.value)}
                  placeholder="e.g. Site Header"
                  disabled={isTheme}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="slug">Slug</Label>
                <Input
                  id="slug"
                  value={slug}
                  onChange={(e) => {
                    setSlug(e.target.value);
                    setSlugManual(true);
                  }}
                  placeholder="e.g. site-header"
                  disabled={isTheme}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Optional description of this layout block"
                  rows={2}
                  disabled={isTheme}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="language">Language</Label>
                <Select
                  value={languageCode || ""}
                  onValueChange={(v) => setLanguageCode(v)}
                  disabled={isTheme}
                >
                  <SelectTrigger id="language">
                    <SelectValue placeholder="Select language..." />
                  </SelectTrigger>
                  <SelectContent>
                    {languages.map((lang) => (
                      <SelectItem key={lang.code} value={lang.code}>
                        {lang.flag} {lang.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </CardContent>
          </Card>

          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-base font-semibold text-slate-900">Template Code</CardTitle>
            </CardHeader>
            <CardContent>
              <textarea
                value={templateCode}
                onChange={(e) => setTemplateCode(e.target.value)}
                disabled={isTheme}
                className="w-full h-96 font-mono text-sm bg-slate-950 text-slate-100 rounded-lg p-4 border-0 focus:ring-2 focus:ring-indigo-500 focus:outline-none resize-y disabled:opacity-60"
                placeholder={`{{/* Layout block template */}}\n<header class="site-header">\n  <nav>\n    {{ yield nav() }}\n  </nav>\n</header>`}
                spellCheck={false}
              />
            </CardContent>
          </Card>
        </div>

        {/* Reference panel */}
        <div className="space-y-6">
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-base font-semibold text-slate-900">Template Reference</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm text-slate-600">
              <p>Layout blocks are reusable partials included in layouts using the Jet template engine.</p>
              <div>
                <p className="font-medium text-slate-700 mb-1">Include in a layout:</p>
                <code className="block bg-slate-100 rounded p-2 text-xs font-mono text-slate-700">
                  {`{{ include "blocks/<slug>" }}`}
                </code>
              </div>
              <div>
                <p className="font-medium text-slate-700 mb-1">Variables:</p>
                <ul className="list-disc list-inside space-y-1 text-xs">
                  <li><code className="bg-slate-100 px-1 rounded">.Site</code> - Site settings</li>
                  <li><code className="bg-slate-100 px-1 rounded">.Node</code> - Current content node</li>
                  <li><code className="bg-slate-100 px-1 rounded">.Menus</code> - Navigation menus</li>
                  <li><code className="bg-slate-100 px-1 rounded">.Language</code> - Current language</li>
                </ul>
              </div>
              <div>
                <p className="font-medium text-slate-700 mb-1">Common patterns:</p>
                <ul className="list-disc list-inside space-y-1 text-xs">
                  <li>Site header / navigation</li>
                  <li>Footer with links</li>
                  <li>Sidebar widgets</li>
                  <li>Breadcrumb trails</li>
                </ul>
              </div>
              <div>
                <p className="font-medium text-slate-700 mb-1">Jet syntax:</p>
                <ul className="list-disc list-inside space-y-1 text-xs">
                  <li><code className="bg-slate-100 px-1 rounded">{`{{ .Variable }}`}</code> - Output</li>
                  <li><code className="bg-slate-100 px-1 rounded">{`{{ if .Cond }}...{{ end }}`}</code> - Conditional</li>
                  <li><code className="bg-slate-100 px-1 rounded">{`{{ range .Items }}...{{ end }}`}</code> - Loop</li>
                  <li><code className="bg-slate-100 px-1 rounded">{`{{ yield content() }}`}</code> - Block yield</li>
                </ul>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Delete dialog */}
      <Dialog open={showDelete} onOpenChange={setShowDelete}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Layout Block</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{name}&quot;? This action cannot be undone.
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

      {/* Detach dialog */}
      <Dialog open={showDetach} onOpenChange={setShowDetach}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Detach from Theme</DialogTitle>
            <DialogDescription>
              This will create an editable copy of this layout block. The theme version will no longer
              be used. You can always re-sync from the theme later.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDetach(false)} disabled={detaching}>
              Cancel
            </Button>
            <Button
              onClick={handleDetach}
              disabled={detaching}
              className="bg-amber-600 hover:bg-amber-700 text-white"
            >
              {detaching ? "Detaching..." : "Detach"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
