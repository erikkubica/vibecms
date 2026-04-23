import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import {
  ArrowLeft,
  Save,
  Trash2,
  Loader2,
  Unplug,
  Info,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import CodeEditor from "@/components/ui/code-editor";
import { toast } from "sonner";
import { usePageMeta } from "@/components/layout/page-meta";
import {
  getLayout,
  createLayout,
  updateLayout,
  deleteLayout,
  detachLayout,
  getLanguages,
  type Layout,
  type Language,
} from "@/api/client";

function slugify(text: string): string {
  return text
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^\w\s-]/g, "")
    .replace(/[\s_]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

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

export default function LayoutEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEdit = !!id;

  const [loading, setLoading] = useState(isEdit);
  const [saving, setSaving] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [detaching, setDetaching] = useState(false);
  const [autoSlug, setAutoSlug] = useState(!isEdit);

  // Form state
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [languageId, setLanguageId] = useState<number | null>(null);
  const [templateCode, setTemplateCode] = useState("");
  const [isDefault, setIsDefault] = useState(false);
  const [supportsBlocks, setSupportsBlocks] = useState(true);
  const [source, setSource] = useState("custom");
  const [themeName, setThemeName] = useState<string | null>(null);
  const [originalLayout, setOriginalLayout] = useState<Layout | null>(null);

  usePageMeta([
    "Layouts",
    isEdit ? (name ? `Edit "${name}"` : "Edit") : "New Layout",
  ]);

  // Languages
  const [languages, setLanguages] = useState<Language[]>([]);

  const isManaged = source !== "custom";

  useEffect(() => {
    getLanguages(true)
      .then((langs) => setLanguages(langs))
      .catch(() => {});
  }, []);

  useEffect(() => {
    if (!isEdit) return;
    let cancelled = false;
    setLoading(true);
    getLayout(id)
      .then((layout) => {
        if (cancelled) return;
        setOriginalLayout(layout);
        setName(layout.name);
        setSlug(layout.slug);
        setDescription(layout.description || "");
        setLanguageId(layout.language_id);
        setTemplateCode(layout.template_code || "");
        setIsDefault(layout.is_default);
        setSupportsBlocks(layout.supports_blocks !== false);
        setSource(layout.source || "custom");
        setThemeName(layout.theme_name || null);
        setAutoSlug(false);
      })
      .catch(() => {
        toast.error("Failed to load layout");
        navigate("/admin/layouts", { replace: true });
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [id, isEdit, navigate]);

  // Auto-generate slug from name
  useEffect(() => {
    if (autoSlug) {
      setSlug(slugify(name));
    }
  }, [name, autoSlug]);

  async function handleSave(e: FormEvent) {
    e.preventDefault();

    if (!name.trim() || !slug.trim()) {
      toast.error("Name and slug are required");
      return;
    }

    const data: Partial<Layout> = {
      name,
      slug,
      description,
      language_id: languageId,
      template_code: templateCode,
      is_default: isDefault,
      supports_blocks: supportsBlocks,
    };

    setSaving(true);
    try {
      if (isEdit) {
        const updated = await updateLayout(id, data);
        setOriginalLayout(updated);
        toast.success("Layout updated successfully");
      } else {
        const created = await createLayout(data);
        toast.success("Layout created successfully");
        navigate(`/admin/layouts/${created.id}`, { replace: true });
      }
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to save layout";
      toast.error(message);
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!id) return;
    setDeleting(true);
    try {
      await deleteLayout(id);
      toast.success("Layout deleted successfully");
      navigate("/admin/layouts", { replace: true });
    } catch {
      toast.error("Failed to delete layout");
    } finally {
      setDeleting(false);
    }
  }

  async function handleDetach() {
    if (!id) return;
    setDetaching(true);
    try {
      const detached = await detachLayout(id);
      setOriginalLayout(detached);
      setSource(detached.source);
      toast.success("Layout detached from theme — now editable");
    } catch {
      toast.error("Failed to detach layout");
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
            <Link to="/admin/layouts">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <h1 className="text-2xl font-bold text-slate-900">
            {isEdit ? "Edit Layout" : "New Layout"}
          </h1>
          {isManaged && (
            <Badge className="bg-amber-100 text-amber-700 hover:bg-amber-100 border-0 text-xs">
              {source === "theme" ? (themeName || "Theme") : "Extension"}
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-2">
          {isEdit && isManaged && (
            <Button
              variant="outline"
              onClick={handleDetach}
              disabled={detaching}
              className="text-amber-600 border-amber-300 hover:bg-amber-50"
            >
              <Unplug className="mr-2 h-4 w-4" />
              {detaching ? "Detaching..." : "Detach"}
            </Button>
          )}
          {isEdit && !isManaged && (
            <Button
              variant="outline"
              className="text-red-600 border-red-300 hover:bg-red-50"
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
            <Save className="mr-2 h-4 w-4" />
            {saving ? "Saving..." : "Save"}
          </Button>
        </div>
      </div>

      {isManaged && (
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700 flex items-start gap-2">
          <Info className="h-4 w-4 mt-0.5 shrink-0" />
          <p>
            This layout is managed by the active {source} and is read-only. To customize it, click
            &quot;Detach&quot; to create an editable copy.
          </p>
        </div>
      )}

      <form onSubmit={handleSave} className="grid gap-6 lg:grid-cols-3">
        {/* Main content */}
        <div className="space-y-6 lg:col-span-2">
          {/* Code Editor */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-base font-semibold text-slate-800">Template Code</CardTitle>
            </CardHeader>
            <CardContent>
              <CodeEditor
                value={templateCode}
                onChange={setTemplateCode}
                disabled={isManaged}
                height="500px"
                placeholder="Enter your Go html/template code here..."
                variables={TEMPLATE_VARIABLES}
              />
            </CardContent>
          </Card>

          {/* Reference Panel */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-base font-semibold text-slate-800">
                Template Reference
              </CardTitle>
            </CardHeader>
            <CardContent>
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
                    </div>
                    <h3 className="mb-3 mt-4 text-sm font-semibold text-slate-700">Menus</h3>
                    <div className="space-y-2 text-sm">
                      <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{'{{$menu := index .app.menus "main-nav"}}'}</code> <span className="text-slate-500">get menu by slug</span></div>
                      <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{range $menu.items}}"}</code> <span className="text-slate-500">loop menu items</span></div>
                      <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.title}} {{.url}} {{.target}}"}</code> <span className="text-slate-500">item fields</span></div>
                      <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.css_class}} {{.item_type}}"}</code> <span className="text-slate-500">more item fields</span></div>
                      <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{range .children}}...{{end}}"}</code> <span className="text-slate-500">nested submenu items</span></div>
                    </div>
                    <h3 className="mb-3 mt-4 text-sm font-semibold text-slate-700">Language</h3>
                    <div className="space-y-2 text-sm">
                      <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.app.current_lang.code}}"}</code> <span className="text-slate-500">e.g. "en"</span></div>
                      <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.app.current_lang.name}}"}</code> <span className="text-slate-500">e.g. "English"</span></div>
                      <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.app.current_lang.flag}}"}</code> <span className="text-slate-500">e.g. emoji flag</span></div>
                      <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{range .app.languages}}{{.code}}{{end}}"}</code> <span className="text-slate-500">all languages</span></div>
                    </div>
                    <h3 className="mb-3 mt-4 text-sm font-semibold text-slate-700">User / Auth</h3>
                    <div className="space-y-2 text-sm">
                      <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{if .user.logged_in}}...{{end}}"}</code> <span className="text-slate-500">check if logged in</span></div>
                      <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.user.email}}"}</code> <span className="text-slate-500">user email</span></div>
                      <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.user.role}}"}</code> <span className="text-slate-500">user role (admin, editor...)</span></div>
                      <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.user.full_name}}"}</code> <span className="text-slate-500">display name</span></div>
                      <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.user.id}}"}</code> <span className="text-slate-500">user ID</span></div>
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
              <CardTitle className="text-base font-semibold text-slate-800">Layout Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="name">Name</Label>
                <Input
                  id="name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Main Layout"
                  disabled={isManaged}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="slug">Slug</Label>
                <Input
                  id="slug"
                  value={slug}
                  onChange={(e) => {
                    setAutoSlug(false);
                    setSlug(e.target.value);
                  }}
                  placeholder="main-layout"
                  disabled={isManaged}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="A brief description of this layout..."
                  rows={2}
                  disabled={isManaged}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="language">Language</Label>
                <select
                  id="language"
                  className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-700 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 disabled:opacity-50"
                  value={languageId === null ? "" : String(languageId)}
                  onChange={(e) => setLanguageId(e.target.value === "" ? null : Number(e.target.value))}
                  disabled={isManaged}
                >
                  <option value="">All Languages</option>
                  {languages.map((lang) => (
                    <option key={lang.id} value={String(lang.id)}>
                      {lang.flag} {lang.name}
                    </option>
                  ))}
                </select>
              </div>
              <div className="flex items-center space-x-3 pt-1">
                <label className="flex items-center gap-2 text-sm text-slate-700 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={isDefault}
                    onChange={(e) => setIsDefault(e.target.checked)}
                    disabled={isManaged}
                    className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                  />
                  Set as default layout
                </label>
              </div>
              <div className="space-y-1 pt-1">
                <label className="flex items-center gap-2 text-sm text-slate-700 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={supportsBlocks}
                    onChange={(e) => setSupportsBlocks(e.target.checked)}
                    disabled={isManaged}
                    className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                  />
                  Supports blocks
                </label>
                <p className="text-xs text-slate-500 pl-6">
                  Allow block-based composition on nodes using this layout. Disable for
                  layouts that render content entirely from node fields.
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </form>

      {/* Delete dialog */}
      <Dialog
        open={showDelete}
        onOpenChange={(open) => !open && setShowDelete(false)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Layout</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{originalLayout?.name}&quot;?
              This action cannot be undone.
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
