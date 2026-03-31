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
import CodeEditor from "@/components/ui/code-editor";
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

const TEMPLATE_VARIABLES = [
  "app.menus",
  "app.settings",
  "app.languages",
  "app.current_lang",
  "app.head_styles",
  "app.head_scripts",
  "app.foot_scripts",
  "app.block_styles",
  "app.block_scripts",
  "node.title",
  "node.slug",
  "node.full_url",
  "node.blocks_html",
  "node.fields",
  "node.seo",
  "node.node_type",
  "node.language_code",
];

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
  const [languageId, setLanguageId] = useState<number | null>(null);
  const [templateCode, setTemplateCode] = useState("");
  const [source, setSource] = useState("custom");

  const isManaged = source !== "custom";

  const fetchLayoutBlock = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const data = await getLayoutBlock(id);
      setName(data.name);
      setSlug(data.slug);
      setDescription(data.description || "");
      setLanguageId(data.language_id);
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
        language_id: languageId,
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
          {isManaged && (
            <Badge className="bg-amber-100 text-amber-700 hover:bg-amber-100 border-0 text-xs">{source === "theme" ? "Theme" : "Extension"}</Badge>
          )}
        </div>
        <div className="flex items-center gap-2">
          {!isNew && isManaged && (
            <Button
              variant="outline"
              onClick={() => setShowDetach(true)}
              className="text-amber-600 border-amber-300 hover:bg-amber-50"
            >
              <Unlink className="mr-2 h-4 w-4" />
              Detach
            </Button>
          )}
          {!isNew && !isManaged && (
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
            disabled={saving || isManaged}
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

      {isManaged && (
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700 flex items-start gap-2">
          <Info className="h-4 w-4 mt-0.5 shrink-0" />
          <p>
            This layout block is managed by the active {source} and is read-only. To customize it, click
            &quot;Detach&quot; to create an editable copy.
          </p>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Main content */}
        <div className="space-y-6 lg:col-span-2">
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-base font-semibold text-slate-900">Template Code</CardTitle>
            </CardHeader>
            <CardContent>
              <CodeEditor
                value={templateCode}
                onChange={setTemplateCode}
                disabled={isManaged}
                height="400px"
                placeholder="Enter your Go html/template code here..."
                variables={TEMPLATE_VARIABLES}
              />
            </CardContent>
          </Card>

          {/* Template Reference */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-base font-semibold text-slate-900">Template Reference</CardTitle>
            </CardHeader>
            <CardContent className="border-t border-slate-100 pt-4">
              <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
                <div>
                  <h3 className="mb-3 text-sm font-semibold text-slate-700">App Variables</h3>
                  <div className="space-y-2 text-sm">
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.app.settings.site_name}}"}</code> <span className="text-slate-500">site setting by key</span></div>
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.app.current_lang.code}}"}</code> <span className="text-slate-500">current language code</span></div>
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.app.block_styles}}"}</code> <span className="text-slate-500">inline block CSS (HTML)</span></div>
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.app.block_scripts}}"}</code> <span className="text-slate-500">inline block JS (HTML)</span></div>
                  </div>
                  <h3 className="mb-3 mt-4 text-sm font-semibold text-slate-700">Loops (use range)</h3>
                  <div className="space-y-2 text-sm">
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{'{{range .app.head_styles}}<link rel="stylesheet" href="{{.}}">{{end}}'}</code></div>
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{'{{range .app.head_scripts}}<script src="{{.}}"></script>{{end}}'}</code></div>
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{'{{range .app.foot_scripts}}<script src="{{.}}" defer></script>{{end}}'}</code></div>
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{'{{range .app.languages}}{{.code}}{{end}}'}</code></div>
                  </div>
                </div>
                <div>
                  <h3 className="mb-3 text-sm font-semibold text-slate-700">Node Variables</h3>
                  <div className="space-y-2 text-sm">
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.node.title}}"}</code> <span className="text-slate-500">page title</span></div>
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.node.slug}}"}</code> <span className="text-slate-500">page slug</span></div>
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.node.full_url}}"}</code> <span className="text-slate-500">full URL path</span></div>
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.node.blocks_html}}"}</code> <span className="text-slate-500">rendered content blocks</span></div>
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.node.node_type}}"}</code> <span className="text-slate-500">page, post, etc.</span></div>
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.node.language_code}}"}</code> <span className="text-slate-500">language code</span></div>
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.node.seo.title}}"}</code> <span className="text-slate-500">SEO title</span></div>
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.node.fields}}"}</code> <span className="text-slate-500">custom fields map</span></div>
                  </div>
                  <h3 className="mb-3 mt-4 text-sm font-semibold text-slate-700">Functions</h3>
                  <div className="space-y-2 text-sm">
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{renderLayoutBlock \"slug\"}}"}</code> <span className="text-slate-500">render a partial/layout block</span></div>
                    <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{'{{$menu := index .app.menus "main-nav"}}'}</code> <span className="text-slate-500">get menu by slug</span></div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
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
                  disabled={isManaged}
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
                  disabled={isManaged}
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
                  disabled={isManaged}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="language">Language</Label>
                <Select
                  value={languageId === null ? "all" : String(languageId)}
                  onValueChange={(v) => setLanguageId(v === "all" ? null : Number(v))}
                  disabled={isManaged}
                >
                  <SelectTrigger id="language">
                    <SelectValue placeholder="All Languages" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Languages</SelectItem>
                    {languages.map((lang) => (
                      <SelectItem key={lang.id} value={String(lang.id)}>
                        {lang.flag} {lang.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
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
            <DialogTitle>Detach from {source === "theme" ? "Theme" : "Extension"}</DialogTitle>
            <DialogDescription>
              This will create an editable copy of this layout block. The {source} version will no longer
              be used. You can always re-sync from the {source} later.
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
