import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import {
  ArrowLeft,
  Save,
  Trash2,
  Loader2,
  Unplug,
  Info,
  FileCode,
  BookOpen,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent } from "@/components/ui/card";
import { SectionHeader } from "@/components/ui/section-header";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { CodeWindow } from "@/components/ui/code-window";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
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

  useEffect(() => {
    if (autoSlug) {
      setSlug(slugify(name));
    }
  }, [name, autoSlug]);

  async function handleSave(e?: FormEvent) {
    e?.preventDefault();

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
    <div className="space-y-4">
      {isManaged && (
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-700 flex items-start gap-2">
          <Info className="h-4 w-4 mt-0.5 shrink-0" />
          <p>
            This layout is managed by the active {source} and is read-only. To customize it, click
            &quot;Detach&quot; in the sidebar to create an editable copy.
          </p>
        </div>
      )}

      <form onSubmit={handleSave} className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        {/* Main content */}
        <div className="space-y-4 min-w-0">
          {/* Title + Slug pill */}
          <div
            className="flex items-center gap-1.5"
            style={{
              padding: 6,
              background: "var(--card-bg)",
              border: "1px solid var(--border)",
              borderRadius: "var(--radius-lg)",
              boxShadow: "var(--shadow-sm)",
            }}
          >
            <Button variant="ghost" size="icon" asChild className="h-7 w-7 shrink-0">
              <Link to="/admin/layouts" title="Back to Layouts">
                <ArrowLeft className="h-3.5 w-3.5" style={{ color: "var(--fg-muted)" }} />
              </Link>
            </Button>
            <div className="flex items-center gap-1.5 flex-[1_1_60%] min-w-0 px-1">
              <span
                className="shrink-0 uppercase"
                style={{
                  fontSize: 10.5,
                  fontWeight: 600,
                  color: "var(--fg-muted)",
                  letterSpacing: "0.06em",
                }}
              >
                Name
              </span>
              <input
                placeholder="Main Layout"
                value={name}
                onChange={(e) => setName(e.target.value)}
                disabled={isManaged}
                required
                className="flex-1 min-w-0 bg-transparent outline-none disabled:opacity-60"
                style={{
                  border: "none",
                  padding: "6px 4px",
                  fontSize: 14,
                  fontWeight: 500,
                  color: "var(--fg)",
                }}
              />
            </div>
            <div className="w-px h-5 shrink-0" style={{ background: "var(--border)" }} />
            <div className="flex items-center gap-1 flex-[1_1_40%] min-w-0 px-1">
              <span
                className="shrink-0"
                style={{
                  fontSize: 11,
                  color: "var(--fg-subtle)",
                  fontFamily: "var(--font-mono)",
                }}
              >
                slug:
              </span>
              <input
                placeholder="main-layout"
                value={slug}
                onChange={(e) => {
                  setAutoSlug(false);
                  setSlug(e.target.value);
                }}
                disabled={isManaged || autoSlug}
                required
                className="flex-1 min-w-0 bg-transparent outline-none disabled:opacity-60"
                style={{
                  border: "none",
                  padding: "6px 0",
                  fontSize: 12.5,
                  color: "var(--fg)",
                  fontFamily: "var(--font-mono)",
                }}
              />
              {!isManaged && (
                <button
                  type="button"
                  className="shrink-0 px-1.5 py-0.5 rounded text-[10.5px] font-medium uppercase"
                  style={{
                    color: autoSlug ? "var(--accent)" : "var(--fg-muted)",
                    background: autoSlug ? "color-mix(in oklab, var(--accent) 12%, transparent)" : "var(--sub-bg)",
                    border: "1px solid var(--border)",
                    letterSpacing: "0.04em",
                  }}
                  onClick={() => setAutoSlug(!autoSlug)}
                  title={autoSlug ? "Click to edit slug manually" : "Click to auto-generate slug from name"}
                >
                  {autoSlug ? "Auto" : "Edit"}
                </button>
              )}
            </div>
            {isManaged && (
              <Badge
                className="shrink-0"
                style={{
                  fontSize: 10.5,
                  background: "color-mix(in oklab, #f59e0b 14%, transparent)",
                  color: "#a16207",
                  border: "1px solid color-mix(in oklab, #f59e0b 30%, transparent)",
                }}
              >
                {source === "theme" ? (themeName || "Theme") : "Extension"}
              </Badge>
            )}
            {isEdit && (
              <Badge
                variant="secondary"
                className="shrink-0 font-mono"
                style={{ fontSize: 10.5, background: "var(--sub-bg)", color: "var(--fg-muted)", border: "1px solid var(--border)" }}
              >
                ID {id}
              </Badge>
            )}
          </div>

          {/* Tabs */}
          <Tabs defaultValue="template" className="w-full">
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="template" className="">
                <FileCode className="mr-2 h-4 w-4" /> Template
              </TabsTrigger>
              <TabsTrigger value="reference" className="">
                <BookOpen className="mr-2 h-4 w-4" /> Reference
              </TabsTrigger>
            </TabsList>

            <TabsContent value="template" className="mt-4 ring-offset-white focus-visible:outline-none">
              <CodeWindow
                title="Template Code — Go html/template"
                value={templateCode}
                onChange={setTemplateCode}
                disabled={isManaged}
                height="600px"
                placeholder="Enter your Go html/template code here..."
                variables={TEMPLATE_VARIABLES}
              />
            </TabsContent>

            <TabsContent value="reference" className="mt-4 ring-offset-white focus-visible:outline-none">
              <Card className="rounded-xl border border-slate-200 shadow-sm">
                <SectionHeader title="Template Reference" />
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
                      </div>
                      <h3 className="mb-3 mt-4 text-sm font-semibold text-slate-700">User / Auth</h3>
                      <div className="space-y-2 text-sm">
                        <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{if .user.logged_in}}...{{end}}"}</code> <span className="text-slate-500">check if logged in</span></div>
                        <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.user.email}}"}</code> <span className="text-slate-500">user email</span></div>
                        <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.user.role}}"}</code> <span className="text-slate-500">user role</span></div>
                        <div><code className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-700">{"{{.user.full_name}}"}</code> <span className="text-slate-500">display name</span></div>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </div>

        {/* Sidebar */}
        <div className="space-y-4">
          {/* Publish card */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="Publish" />
            <CardContent className="space-y-4">
              {isManaged ? (
                <Button
                  type="button"
                  className="w-full bg-amber-600 hover:bg-amber-700 text-white font-medium rounded-lg shadow-sm h-9 text-sm"
                  onClick={handleDetach}
                  disabled={detaching}
                >
                  <Unplug className="mr-1.5 h-3.5 w-3.5" />
                  {detaching ? "Detaching..." : "Detach"}
                </Button>
              ) : (
                <Button
                  type="submit"
                  className="w-full bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg shadow-sm h-9 text-sm"
                  disabled={saving}
                >
                  <Save className="mr-1.5 h-3.5 w-3.5" />
                  {saving ? "Saving..." : "Save"}
                </Button>
              )}

              {isEdit && !isManaged && (
                <>
                  <Separator />
                  <Button
                    type="button"
                    variant="outline"
                    className="w-full bg-red-50 text-red-700 border-red-200 hover:bg-red-100 rounded-lg font-medium h-8 text-xs"
                    onClick={() => setShowDelete(true)}
                  >
                    <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                    Delete
                  </Button>
                </>
              )}

              {isEdit && originalLayout && (
                <>
                  <Separator />
                  <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-slate-400">
                    <div className="flex justify-between">
                      <span>Source</span>
                      <span className="text-slate-600 capitalize">{source}</span>
                    </div>
                    {originalLayout.created_at && (
                      <div className="flex justify-between">
                        <span>Created</span>
                        <span className="text-slate-600">{new Date(originalLayout.created_at).toLocaleDateString()}</span>
                      </div>
                    )}
                    {originalLayout.updated_at && (
                      <div className="flex justify-between">
                        <span>Updated</span>
                        <span className="text-slate-600">{new Date(originalLayout.updated_at).toLocaleDateString()}</span>
                      </div>
                    )}
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          {/* Settings card */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="Settings" />
            <CardContent className="space-y-4">
              <div className="space-y-1.5">
                <Label htmlFor="description" className="text-xs font-medium text-slate-500">Description</Label>
                <Textarea
                  id="description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="A brief description of this layout..."
                  rows={2}
                  disabled={isManaged}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="language" className="text-xs font-medium text-slate-500">Language</Label>
                <Select
                  value={languageId === null ? "all" : String(languageId)}
                  onValueChange={(v) => setLanguageId(v === "all" ? null : Number(v))}
                  disabled={isManaged}
                >
                  <SelectTrigger id="language" className="h-9 rounded-lg border-slate-300 text-sm">
                    <SelectValue placeholder="All Languages" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Languages</SelectItem>
                    {languages.map((lang) => (
                      <SelectItem key={lang.id} value={String(lang.id)}>
                        {lang.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-3 pt-1">
                <label className={`flex items-center justify-between gap-3 ${isManaged ? "cursor-not-allowed opacity-60" : "cursor-pointer"}`}>
                  <span className="text-sm text-slate-700">Set as default layout</span>
                  <Switch
                    checked={isDefault}
                    onCheckedChange={setIsDefault}
                    disabled={isManaged}
                  />
                </label>
                <label className={`flex items-center justify-between gap-3 ${isManaged ? "cursor-not-allowed opacity-60" : "cursor-pointer"}`}>
                  <div className="space-y-0.5 min-w-0">
                    <span className="block text-sm text-slate-700">Supports blocks</span>
                    <span className="block text-[11px] text-slate-400">Enable block-based composition.</span>
                  </div>
                  <Switch
                    checked={supportsBlocks}
                    onCheckedChange={setSupportsBlocks}
                    disabled={isManaged}
                  />
                </label>
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
