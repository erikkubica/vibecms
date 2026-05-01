import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
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
import { Titlebar } from "@/components/ui/titlebar";
import { MetaRow, MetaList } from "@/components/ui/meta-row";
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
        <Loader2 className="h-8 w-8 animate-spin" style={{color: "var(--accent-strong)"}} />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {isManaged && (
        <div className="rounded-lg border p-3 text-xs flex items-start gap-2" style={{background: "var(--warning-bg)", borderColor: "var(--border)", color: "var(--warning)"}}>
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
          <Titlebar
            title={name}
            onTitleChange={setName}
            titleLabel="Name"
            titlePlaceholder="Main Layout"
            slug={slug}
            onSlugChange={isManaged ? undefined : (v) => { setAutoSlug(false); setSlug(v); }}
            slugPrefix=""
            autoSlug={autoSlug}
            onAutoSlugToggle={isManaged ? undefined : () => setAutoSlug(!autoSlug)}
            id={isEdit && id ? Number(id) : undefined}
            onBack={() => navigate("/admin/layouts")}
            actions={isManaged ? (
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
            ) : undefined}
          />

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
              <Card className="rounded-xl border border-border shadow-sm">
                <SectionHeader title="Template Reference" />
                <CardContent>
                  <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
                    <div>
                      <h3 className="mb-3 text-sm font-semibold text-foreground">App Variables</h3>
                      <div className="space-y-2 text-sm">
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.app.settings.site_name}}"}</code> <span className="text-muted-foreground">site setting by key</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.app.current_lang.code}}"}</code> <span className="text-muted-foreground">current language code</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.app.block_styles}}"}</code> <span className="text-muted-foreground">inline block CSS (HTML)</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.app.block_scripts}}"}</code> <span className="text-muted-foreground">inline block JS (HTML)</span></div>
                      </div>
                      <h3 className="mb-3 mt-4 text-sm font-semibold text-foreground">Loops (use range)</h3>
                      <div className="space-y-2 text-sm">
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{'{{range .app.head_styles}}<link rel="stylesheet" href="{{.}}">{{end}}'}</code></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{'{{range .app.head_scripts}}<script src="{{.}}"></script>{{end}}'}</code></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{'{{range .app.foot_scripts}}<script src="{{.}}" defer></script>{{end}}'}</code></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{'{{range .app.languages}}{{.code}}{{end}}'}</code></div>
                      </div>
                    </div>
                    <div>
                      <h3 className="mb-3 text-sm font-semibold text-foreground">Node Variables</h3>
                      <div className="space-y-2 text-sm">
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.node.title}}"}</code> <span className="text-muted-foreground">page title</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.node.slug}}"}</code> <span className="text-muted-foreground">page slug</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.node.full_url}}"}</code> <span className="text-muted-foreground">full URL path</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.node.blocks_html}}"}</code> <span className="text-muted-foreground">rendered content blocks</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.node.node_type}}"}</code> <span className="text-muted-foreground">page, post, etc.</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.node.language_code}}"}</code> <span className="text-muted-foreground">language code</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.node.seo.title}}"}</code> <span className="text-muted-foreground">SEO title</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.node.fields}}"}</code> <span className="text-muted-foreground">custom fields map</span></div>
                      </div>
                      <h3 className="mb-3 mt-4 text-sm font-semibold text-foreground">Functions</h3>
                      <div className="space-y-2 text-sm">
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{renderLayoutBlock \"slug\"}}"}</code> <span className="text-muted-foreground">render a partial/layout block</span></div>
                      </div>
                      <h3 className="mb-3 mt-4 text-sm font-semibold text-foreground">Menus</h3>
                      <div className="space-y-2 text-sm">
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{'{{$menu := index .app.menus "main-nav"}}'}</code> <span className="text-muted-foreground">get menu by slug</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{range $menu.items}}"}</code> <span className="text-muted-foreground">loop menu items</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.title}} {{.url}} {{.target}}"}</code> <span className="text-muted-foreground">item fields</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.css_class}} {{.item_type}}"}</code> <span className="text-muted-foreground">more item fields</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{range .children}}...{{end}}"}</code> <span className="text-muted-foreground">nested submenu items</span></div>
                      </div>
                      <h3 className="mb-3 mt-4 text-sm font-semibold text-foreground">Language</h3>
                      <div className="space-y-2 text-sm">
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.app.current_lang.code}}"}</code> <span className="text-muted-foreground">e.g. "en"</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.app.current_lang.name}}"}</code> <span className="text-muted-foreground">e.g. "English"</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.app.current_lang.flag}}"}</code> <span className="text-muted-foreground">e.g. emoji flag</span></div>
                      </div>
                      <h3 className="mb-3 mt-4 text-sm font-semibold text-foreground">User / Auth</h3>
                      <div className="space-y-2 text-sm">
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{if .user.logged_in}}...{{end}}"}</code> <span className="text-muted-foreground">check if logged in</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.user.email}}"}</code> <span className="text-muted-foreground">user email</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.user.role}}"}</code> <span className="text-muted-foreground">user role</span></div>
                        <div><code className="rounded bg-muted px-2 py-0.5 text-xs font-mono" style={{color: "var(--accent-strong)"}}>{"{{.user.full_name}}"}</code> <span className="text-muted-foreground">display name</span></div>
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
          <Card className="rounded-xl border border-border shadow-sm">
            <SectionHeader title="Publish" />
            <CardContent className="space-y-4">
              {isManaged ? (
                <Button
                  type="button"
                  className="w-full font-medium rounded-lg shadow-sm h-9 text-sm" style={{background: "var(--warning)", color: "#fff"}}
                  onClick={handleDetach}
                  disabled={detaching}
                >
                  <Unplug className="mr-1.5 h-3.5 w-3.5" />
                  {detaching ? "Detaching..." : "Detach"}
                </Button>
              ) : (
                <Button
                  type="submit"
                  className="w-full"
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
                    variant="ghost"
                    className="w-full"
                    style={{ color: "var(--danger)" }}
                    onClick={() => setShowDelete(true)}
                  >
                    <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                    Delete
                  </Button>
                </>
              )}

              {isEdit && originalLayout && (
                <>
                  <div style={{ height: 1, background: "var(--divider)", margin: "4px 0" }} />
                  <MetaList>
                    <MetaRow label="Source" value={<span className="capitalize">{source}</span>} />
                    {originalLayout.created_at && <MetaRow label="Created" value={new Date(originalLayout.created_at).toLocaleDateString("en-GB")} />}
                    {originalLayout.updated_at && <MetaRow label="Updated" value={new Date(originalLayout.updated_at).toLocaleDateString("en-GB")} />}
                  </MetaList>
                </>
              )}
            </CardContent>
          </Card>

          {/* Settings card */}
          <Card className="rounded-xl border border-border shadow-sm">
            <SectionHeader title="Settings" />
            <CardContent className="space-y-4">
              <div className="space-y-1.5">
                <Label htmlFor="description" className="text-xs font-medium text-muted-foreground">Description</Label>
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
                <Label htmlFor="language" className="text-xs font-medium text-muted-foreground">Language</Label>
                <Select
                  value={languageId === null ? "all" : String(languageId)}
                  onValueChange={(v) => setLanguageId(v === "all" ? null : Number(v))}
                  disabled={isManaged}
                >
                  <SelectTrigger id="language" className="h-9 rounded-lg border-border text-sm">
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
                  <span className="text-sm text-foreground">Set as default layout</span>
                  <Switch
                    checked={isDefault}
                    onCheckedChange={setIsDefault}
                    disabled={isManaged}
                  />
                </label>
                <label className={`flex items-center justify-between gap-3 ${isManaged ? "cursor-not-allowed opacity-60" : "cursor-pointer"}`}>
                  <div className="space-y-0.5 min-w-0">
                    <span className="block text-sm text-foreground">Supports blocks</span>
                    <span className="block text-[11px]" style={{color: "var(--fg-subtle)"}}>Enable block-based composition.</span>
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
