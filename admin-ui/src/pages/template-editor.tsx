import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import {
  ArrowLeft,
  Save,
  Trash2,
  Loader2,
  Plus,
  ChevronUp,
  ChevronDown,
  X,
  Square,
  Unlink,
  Info,
  type LucideIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Card,
  CardContent,
} from "@/components/ui/card";
import { SectionHeader } from "@/components/ui/section-header";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { toast } from "sonner";
import BlockPicker, { BLOCK_ICON_MAP } from "@/components/ui/block-picker";
import { usePageMeta } from "@/components/layout/page-meta";
import {
  getTemplate,
  createTemplate,
  updateTemplate,
  deleteTemplate,
  detachTemplate,
  getBlockTypes,
  type Template,
  type TemplateBlockConfig,
  type BlockType,
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

export default function TemplateEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEdit = !!id;

  const [loading, setLoading] = useState(isEdit);
  const [saving, setSaving] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [showDetach, setShowDetach] = useState(false);
  const [detaching, setDetaching] = useState(false);
  const [autoSlug, setAutoSlug] = useState(!isEdit);
  const [showAddBlock, setShowAddBlock] = useState(false);

  // Form state
  const [label, setLabel] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [blockConfig, setBlockConfig] = useState<TemplateBlockConfig[]>([]);
  const [originalTemplate, setOriginalTemplate] = useState<Template | null>(null);
  const [source, setSource] = useState("custom");
  const [themeName, setThemeName] = useState<string | null>(null);
  
  const isManaged = source !== "custom";

  usePageMeta([
    "Templates",
    isEdit ? (label ? `Edit "${label}"` : "Edit") : "New Template",
  ]);

  // Block types
  const [blockTypes, setBlockTypes] = useState<BlockType[]>([]);

  useEffect(() => {
    getBlockTypes().then(setBlockTypes).catch(() => {});
  }, []);

  useEffect(() => {
    if (!isEdit) return;
    let cancelled = false;
    setLoading(true);
    getTemplate(id)
      .then((tpl) => {
        if (cancelled) return;
        setOriginalTemplate(tpl);
        setLabel(tpl.label);
        setSlug(tpl.slug);
        setDescription(tpl.description || "");
        setBlockConfig(tpl.block_config || []);
        setSource(tpl.source || "custom");
        setThemeName(tpl.theme_name || null);
        setAutoSlug(false);
      })
      .catch(() => {
        toast.error("Failed to load template");
        navigate("/admin/templates", { replace: true });
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [id, isEdit, navigate]);

  // Auto-generate slug from label
  useEffect(() => {
    if (autoSlug) {
      setSlug(slugify(label));
    }
  }, [label, autoSlug]);

  function getBlockTypeBySlug(btSlug: string): BlockType | undefined {
    return blockTypes.find((bt) => bt.slug === btSlug);
  }

  function getBlockIcon(btSlug: string): LucideIcon {
    const bt = getBlockTypeBySlug(btSlug);
    if (bt?.icon && BLOCK_ICON_MAP[bt.icon]) return BLOCK_ICON_MAP[bt.icon];
    return Square;
  }

  function getBlockLabel(btSlug: string): string {
    const bt = getBlockTypeBySlug(btSlug);
    return bt?.label || btSlug;
  }

  function handleAddBlock(btSlug: string) {
    setBlockConfig([...blockConfig, { block_type_slug: btSlug, default_values: {} }]);
    setShowAddBlock(false);
  }

  function handleRemoveBlock(index: number) {
    setBlockConfig(blockConfig.filter((_, i) => i !== index));
  }

  function handleMoveBlock(index: number, direction: "up" | "down") {
    const newConfig = [...blockConfig];
    const targetIndex = direction === "up" ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= newConfig.length) return;
    [newConfig[index], newConfig[targetIndex]] = [newConfig[targetIndex], newConfig[index]];
    setBlockConfig(newConfig);
  }

  async function handleSave(e: FormEvent) {
    e.preventDefault();

    if (!label.trim() || !slug.trim()) {
      toast.error("Label and slug are required");
      return;
    }

    const data: Partial<Template> = {
      label,
      slug,
      description,
      block_config: blockConfig,
    };

    setSaving(true);
    try {
      if (isEdit) {
        const updated = await updateTemplate(id, data);
        setOriginalTemplate(updated);
        toast.success("Template updated successfully");
      } else {
        const created = await createTemplate(data);
        toast.success("Template created successfully");
        navigate(`/admin/templates/${created.id}/edit`, { replace: true });
      }
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to save template";
      toast.error(message);
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!id) return;
    setDeleting(true);
    try {
      await deleteTemplate(id);
      toast.success("Template deleted successfully");
      navigate("/admin/templates", { replace: true });
    } catch {
      toast.error("Failed to delete template");
    } finally {
      setDeleting(false);
    }
  }

  async function handleDetach() {
    if (!id) return;
    setDetaching(true);
    try {
      const detached = await detachTemplate(id);
      setOriginalTemplate(detached);
      setSource(detached.source);
      toast.success("Template detached — now editable");
      setShowDetach(false);
    } catch {
      toast.error("Failed to detach template");
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
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild className="rounded-lg hover:bg-slate-200">
            <Link to="/admin/templates">
              <ArrowLeft className="h-5 w-5 text-slate-600" />
            </Link>
          </Button>
          <h1 className="text-2xl font-bold text-slate-900">
            {isEdit ? "Edit Template" : "New Template"}
          </h1>
          {isEdit && isManaged && (
            <Badge className="bg-amber-100 text-amber-700 hover:bg-amber-100 border-0 text-xs">
              {source === "theme" ? (themeName || "Theme") : "Extension"}
            </Badge>
          )}
        </div>
        {isEdit && isManaged && (
          <Button
            variant="outline"
            onClick={() => setShowDetach(true)}
            className="text-amber-600 border-amber-300 hover:bg-amber-50"
          >
            <Unlink className="mr-2 h-4 w-4" />
            Detach
          </Button>
        )}
      </div>

      {isEdit && isManaged && (
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700 flex items-start gap-2">
          <Info className="h-4 w-4 mt-0.5 shrink-0" />
          <p>
            This template is managed by the active {source} and is read-only. To customize it, click
            &quot;Detach&quot; to create an editable copy.
          </p>
        </div>
      )}

      <form onSubmit={handleSave} className="grid gap-6 lg:grid-cols-3">
        {/* Main content */}
        <div className="space-y-6 lg:col-span-2">
          {/* Basic info */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="Basic Info" />
            <CardContent className="space-y-4 p-6">
              <div className="space-y-2">
                <Label htmlFor="label" className="text-sm font-medium text-slate-700">Label</Label>
                <Input
                  id="label"
                  placeholder="e.g. Landing Page, Blog Post"
                  value={label}
                  onChange={(e) => setLabel(e.target.value)}
                  required
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>

              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label htmlFor="slug" className="text-sm font-medium text-slate-700">Slug</Label>
                  <button
                    type="button"
                    className="text-xs text-indigo-600 hover:underline"
                    onClick={() => setAutoSlug(!autoSlug)}
                  >
                    {autoSlug ? "Edit manually" : "Auto-generate"}
                  </button>
                </div>
                <Input
                  id="slug"
                  placeholder="template-slug"
                  value={slug}
                  onChange={(e) => {
                    setAutoSlug(false);
                    setSlug(e.target.value);
                  }}
                  disabled={autoSlug}
                  required
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="description" className="text-sm font-medium text-slate-700">Description</Label>
                <Textarea
                  id="description"
                  placeholder="A brief description of this template"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={3}
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>
            </CardContent>
          </Card>

            {/* Blocks */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <h2 className="font-semibold" style={{ fontSize: 14, color: "var(--fg)" }}>Blocks</h2>
              </div>
              <div className="space-y-2">
                {blockConfig.length === 0 && (
                  <div className="flex flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed border-slate-200 py-12 text-slate-400">
                    <span className="text-sm font-medium">No blocks yet</span>
                    <span className="text-xs">Add blocks or insert a template to get started</span>
                  </div>
                )}

                {blockConfig.length > 0 && (
                  <div className="flex flex-col gap-2">
                    {blockConfig.map((block, index) => {
                      const IconComp = getBlockIcon(block.block_type_slug);
                      const typeCategory = block.block_type_slug.split("-")[0];

                      return (
                        <div
                          key={index}
                          className="overflow-hidden"
                          style={{
                            border: "1px solid var(--border)",
                            borderRadius: "var(--radius-lg)",
                          }}
                        >
                          <div
                            className="flex items-center gap-2 select-none"
                            style={{ padding: "8px 10px", background: "var(--sub-bg)" }}
                          >
                            <IconComp size={14} className="shrink-0" style={{ color: "var(--fg-muted)" }} />
                            <span
                              className="font-semibold"
                              style={{ fontSize: 12.5, color: "var(--fg)" }}
                            >
                              {getBlockLabel(block.block_type_slug)}
                            </span>
                            <span
                              className="font-mono"
                              style={{ fontSize: 11, color: "var(--fg-muted)" }}
                            >
                              {block.block_type_slug}
                            </span>
                            {typeCategory && typeCategory !== block.block_type_slug && (
                              <Badge
                                variant="secondary"
                                style={{
                                  fontSize: 10,
                                  background: "color-mix(in oklab, var(--accent) 10%, transparent)",
                                  color: "var(--accent-strong)",
                                  border: "1px solid color-mix(in oklab, var(--accent) 20%, transparent)",
                                }}
                              >
                                {typeCategory}
                              </Badge>
                            )}
                            <div className="flex-1" />
                            <div className="flex items-center gap-0.5">
                              <button
                                type="button"
                                onClick={() => handleMoveBlock(index, "up")}
                                disabled={index === 0}
                                className="p-1 rounded disabled:opacity-30 disabled:cursor-not-allowed hover:bg-black/5"
                                style={{ color: "var(--fg-muted)" }}
                                title="Move up"
                              >
                                <ChevronUp className="h-3.5 w-3.5" />
                              </button>
                              <button
                                type="button"
                                onClick={() => handleMoveBlock(index, "down")}
                                disabled={index === blockConfig.length - 1}
                                className="p-1 rounded disabled:opacity-30 disabled:cursor-not-allowed hover:bg-black/5"
                                style={{ color: "var(--fg-muted)" }}
                                title="Move down"
                              >
                                <ChevronDown className="h-3.5 w-3.5" />
                              </button>
                              <button
                                type="button"
                                onClick={() => handleRemoveBlock(index)}
                                className="p-1 rounded hover:bg-red-50"
                                style={{ color: "var(--danger)" }}
                                title="Delete block"
                              >
                                <X className="h-3.5 w-3.5" />
                              </button>
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
              <div className="mt-2">
                <Button
                  type="button"
                  variant="outline"
                  className="w-full rounded-lg border-dashed border-slate-300 text-slate-500 hover:border-indigo-400 hover:text-indigo-600 py-2"
                  onClick={() => setShowAddBlock(true)}
                >
                  <Plus className="mr-2 h-4 w-4" />
                  Add Block
                </Button>
              </div>
            </div>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardContent className="space-y-4 p-5">
              {/* Blocks count */}
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-slate-500">Blocks</Label>
                <p className="text-sm text-slate-700">{blockConfig.length} block{blockConfig.length !== 1 ? "s" : ""}</p>
              </div>

              {/* Dates (edit mode) */}
              {isEdit && originalTemplate && (
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1.5">
                    <Label className="text-xs font-medium text-slate-500">Created</Label>
                    <p className="text-sm text-slate-700">{new Date(originalTemplate.created_at).toLocaleDateString()}</p>
                  </div>
                  <div className="space-y-1.5">
                    <Label className="text-xs font-medium text-slate-500">Updated</Label>
                    <p className="text-sm text-slate-700">{new Date(originalTemplate.updated_at).toLocaleDateString()}</p>
                  </div>
                </div>
              )}

              {/* Save button */}
              <Button
                type="submit"
                className="w-full bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg shadow-sm h-9 text-sm"
                disabled={saving || isManaged}
              >
                <Save className="mr-1.5 h-3.5 w-3.5" />
                {saving ? "Saving..." : "Save"}
              </Button>

              {/* Actions (edit mode) */}
              {isEdit && !isManaged && (
                <>
                  <Separator />
                  <Button
                    type="button"
                    variant="outline"
                    className="w-full text-red-600 border-red-200 hover:bg-red-50 rounded-lg h-9 text-sm"
                    onClick={() => setShowDelete(true)}
                  >
                    <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                    Delete
                  </Button>
                </>
              )}
            </CardContent>
          </Card>
        </div>
      </form>

      {/* Add Block picker */}
      <BlockPicker
        open={showAddBlock}
        onClose={() => setShowAddBlock(false)}
        onSelect={(item) => handleAddBlock(item.slug)}
        items={blockTypes.map((bt) => ({
          id: bt.id,
          slug: bt.slug,
          label: bt.label,
          description: bt.description,
          icon: bt.icon,
        }))}
        title="Add Block"
        description="Select a block type to add to this template."
        emptyMessage="No block types available. Create block types first."
      />

      {/* Delete dialog */}
      <Dialog open={showDelete} onOpenChange={setShowDelete}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Template</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{originalTemplate?.label}&quot;?
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

      {/* Detach dialog */}
      <Dialog open={showDetach} onOpenChange={setShowDetach}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Detach from {source === "theme" ? "Theme" : "Extension"}</DialogTitle>
            <DialogDescription>
              This will create an editable copy of this template. The {source} version will no longer
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
