import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import {
  ArrowLeft,
  Save,
  Trash2,
  Loader2,
  Plus,
  X,
  FileText,
  Newspaper,
  ShoppingBag,
  Calendar,
  Users,
  Folder,
  Bookmark,
  Tag,
  Star,
  Heart,
  Image,
  Boxes,
  Link2,
  ListTree,
  type LucideIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent } from "@/components/ui/card";
import { SectionHeader } from "@/components/ui/section-header";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import FieldSchemaEditor from "@/components/ui/field-schema-editor";
import { toast } from "sonner";
import { usePageMeta } from "@/components/layout/page-meta";
import {
  getNodeType,
  createNodeType,
  updateNodeType,
  deleteNodeType,
  getLanguages,
  type NodeType,
  type NodeTypeField,
  type Language,
} from "@/api/client";

const ICON_OPTIONS: { value: string; label: string; icon: LucideIcon }[] = [
  { value: "file-text", label: "File", icon: FileText },
  { value: "newspaper", label: "News", icon: Newspaper },
  { value: "shopping-bag", label: "Shop", icon: ShoppingBag },
  { value: "calendar", label: "Calendar", icon: Calendar },
  { value: "users", label: "Users", icon: Users },
  { value: "folder", label: "Folder", icon: Folder },
  { value: "bookmark", label: "Bookmark", icon: Bookmark },
  { value: "tag", label: "Tag", icon: Tag },
  { value: "star", label: "Star", icon: Star },
  { value: "heart", label: "Heart", icon: Heart },
  { value: "image", label: "Image", icon: Image },
  { value: "boxes", label: "Boxes", icon: Boxes },
];

function slugify(text: string): string {
  return text
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^\w\s-]/g, "")
    .replace(/[\s_]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function keyify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s]/g, "")
    .replace(/[\s]+/g, "_")
    .replace(/^_+|_+$/g, "");
}

export default function NodeTypeEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEdit = !!id;

  const [loading, setLoading] = useState(isEdit);
  const [saving, setSaving] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [autoSlug, setAutoSlug] = useState(!isEdit);

  const [label, setLabel] = useState("");
  const [labelPlural, setLabelPlural] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [icon, setIcon] = useState("file-text");
  const [taxonomies, setTaxonomies] = useState<NodeType["taxonomies"]>([]);
  const [fields, setFields] = useState<NodeTypeField[]>([]);
  const [urlPrefixes, setUrlPrefixes] = useState<Record<string, string>>({});
  const [supportsBlocks, setSupportsBlocks] = useState(true);
  const [originalNodeType, setOriginalNodeType] = useState<NodeType | null>(null);

  // Add taxonomy state
  const [newTaxLabel, setNewTaxLabel] = useState("");
  const [newTaxKey, setNewTaxKey] = useState("");
  const [newTaxMultiple, setNewTaxMultiple] = useState(true);
  const [autoTaxKey, setAutoTaxKey] = useState(true);

  usePageMeta([
    "Content Types",
    isEdit ? (label ? `Edit "${label}"` : "Edit") : "New Content Type",
  ]);

  const [languages, setLanguages] = useState<Language[]>([]);

  useEffect(() => {
    getLanguages(true).then(setLanguages).catch(() => {});
  }, []);

  useEffect(() => {
    if (!isEdit) return;
    let cancelled = false;
    setLoading(true);
    getNodeType(id)
      .then((nt) => {
        if (cancelled) return;
        setOriginalNodeType(nt);
        setLabel(nt.label);
        setLabelPlural(nt.label_plural || "");
        setSlug(nt.slug);
        setDescription(nt.description || "");
        setIcon(nt.icon || "file-text");
        const taxes = nt.taxonomies || [];
        setTaxonomies(Array.isArray(taxes) ? taxes : []);
        setFields(nt.field_schema || []);
        setUrlPrefixes(nt.url_prefixes || {});
        setSupportsBlocks(nt.supports_blocks !== false);
        setAutoSlug(false);
      })
      .catch(() => {
        toast.error("Failed to load content type");
        navigate("/admin/content-types", { replace: true });
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
      setSlug(slugify(label));
    }
  }, [label, autoSlug]);

  useEffect(() => {
    if (autoTaxKey) {
      setNewTaxKey(keyify(newTaxLabel));
    }
  }, [newTaxLabel, autoTaxKey]);

  function addTaxonomy() {
    const labelVal = newTaxLabel.trim();
    const keyVal = newTaxKey.trim() || keyify(labelVal);
    if (!labelVal || !keyVal) {
      toast.error("Label and key are required");
      return;
    }
    const currentTaxes = Array.isArray(taxonomies) ? taxonomies : [];
    if (currentTaxes.some(t => t.slug === keyVal)) {
      toast.error("Taxonomy with this key already exists");
      return;
    }
    setTaxonomies([...currentTaxes, { label: labelVal, slug: keyVal, multiple: newTaxMultiple }]);
    setNewTaxLabel("");
    setNewTaxKey("");
    setNewTaxMultiple(true);
    setAutoTaxKey(true);
  }

  async function handleSave(e: FormEvent) {
    e.preventDefault();

    if (!label.trim() || !slug.trim()) {
      toast.error("Label and slug are required");
      return;
    }

    const data: Partial<NodeType> = {
      label,
      label_plural: labelPlural,
      slug,
      description,
      icon,
      taxonomies,
      field_schema: fields,
      url_prefixes: urlPrefixes,
      supports_blocks: supportsBlocks,
    };

    setSaving(true);
    try {
      if (isEdit) {
        const updated = await updateNodeType(id, data);
        setOriginalNodeType(updated);
        toast.success("Content type updated successfully");
      } else {
        const created = await createNodeType(data);
        toast.success("Content type created successfully");
        navigate(`/admin/content-types/${created.id}/edit`, { replace: true });
      }
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to save content type";
      toast.error(message);
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!id) return;
    setDeleting(true);
    try {
      await deleteNodeType(id);
      toast.success("Content type deleted successfully");
      navigate("/admin/content-types", { replace: true });
    } catch {
      toast.error("Failed to delete content type");
    } finally {
      setDeleting(false);
    }
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
      </div>
    );
  }

  const taxArray = Array.isArray(taxonomies) ? taxonomies : [];

  return (
    <div className="space-y-4">
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
              <Link to="/admin/content-types" title="Back to Content Types">
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
                Label
              </span>
              <input
                placeholder="e.g. Product, Event"
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                required
                className="flex-1 min-w-0 bg-transparent outline-none"
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
                placeholder="product"
                value={slug}
                onChange={(e) => {
                  setAutoSlug(false);
                  setSlug(e.target.value);
                }}
                disabled={isEdit || autoSlug}
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
              {!isEdit && (
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
                  title={autoSlug ? "Click to edit slug manually" : "Click to auto-generate slug from label"}
                >
                  {autoSlug ? "Auto" : "Edit"}
                </button>
              )}
            </div>
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
          <Tabs defaultValue="fields" className="w-full">
            <TabsList className="grid w-full grid-cols-3">
              <TabsTrigger value="fields" className="">
                <Boxes className="mr-2 h-4 w-4" /> Fields
              </TabsTrigger>
              <TabsTrigger value="taxonomies" className="">
                <ListTree className="mr-2 h-4 w-4" /> Taxonomies
              </TabsTrigger>
              <TabsTrigger value="urls" className="">
                <Link2 className="mr-2 h-4 w-4" /> URLs
              </TabsTrigger>
            </TabsList>

            <TabsContent value="fields" className="mt-4 ring-offset-white focus-visible:outline-none">
              <Card className="rounded-xl border border-slate-200 shadow-sm">
                <SectionHeader title="Custom Fields" />
                <CardContent>
                  <p className="text-xs text-slate-500 mb-4">
                    Define editable fields for this content type. They appear in the node editor sidebar.
                  </p>
                  <FieldSchemaEditor fields={fields} onChange={setFields} />
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="taxonomies" className="mt-4 ring-offset-white focus-visible:outline-none">
              <Card className="rounded-xl border border-slate-200 shadow-sm">
                <SectionHeader title="Taxonomies" />
                <CardContent className="space-y-4">
                  <p className="text-xs text-slate-500">
                    Register taxonomies (e.g. Categories, Tags, Genres) to classify content of this type.
                  </p>

                  {taxArray.length > 0 ? (
                    <div className="space-y-2">
                      {taxArray.map((tax, index) => (
                        <div
                          key={tax.slug}
                          className="overflow-hidden"
                          style={{
                            border: "1px solid var(--border)",
                            borderRadius: "var(--radius-lg)",
                            background: "var(--card-bg)",
                          }}
                        >
                          <div
                            className="flex items-center gap-2 select-none"
                            style={{ padding: "8px 10px", background: "var(--sub-bg)" }}
                          >
                            <span className="font-semibold" style={{ fontSize: 12.5, color: "var(--fg)" }}>
                              {tax.label}
                            </span>
                            <span className="font-mono" style={{ fontSize: 11, color: "var(--fg-muted)" }}>
                              {tax.slug}
                            </span>
                            <Badge className={`border-0 text-[10px] ${tax.multiple ? "bg-violet-100 text-violet-700 hover:bg-violet-100" : "bg-slate-100 text-slate-600 hover:bg-slate-100"}`}>
                              {tax.multiple ? "Multiple" : "Single"}
                            </Badge>
                            <div className="flex-1" />
                            <button
                              type="button"
                              onClick={() => setTaxonomies(taxArray.filter((_, i) => i !== index))}
                              className="p-1 rounded hover:bg-red-50"
                              style={{ color: "var(--danger)" }}
                              title="Remove"
                            >
                              <X className="h-3.5 w-3.5" />
                            </button>
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-sm text-slate-400 italic text-center py-4">No taxonomies registered.</p>
                  )}

                  <Separator />

                  <div className="space-y-3 rounded-lg border border-indigo-200 bg-indigo-50/50 p-4">
                    <p className="text-sm font-semibold text-slate-700">Add Taxonomy</p>
                    <div className="grid gap-3 sm:grid-cols-2">
                      <div className="space-y-1.5">
                        <Label className="text-xs font-medium text-slate-700">Label</Label>
                        <Input
                          placeholder="e.g. Category"
                          value={newTaxLabel}
                          onChange={(e) => setNewTaxLabel(e.target.value)}
                          className="h-9 text-sm"
                        />
                      </div>
                      <div className="space-y-1.5">
                        <div className="flex items-center justify-between">
                          <Label className="text-xs font-medium text-slate-700">Key (slug)</Label>
                          <button
                            type="button"
                            className="text-[10px] text-indigo-600 hover:underline"
                            onClick={() => setAutoTaxKey(!autoTaxKey)}
                          >
                            {autoTaxKey ? "Edit manually" : "Auto"}
                          </button>
                        </div>
                        <Input
                          placeholder="category"
                          value={newTaxKey}
                          onChange={(e) => {
                            setAutoTaxKey(false);
                            setNewTaxKey(e.target.value);
                          }}
                          disabled={autoTaxKey}
                          className="h-9 text-sm font-mono"
                        />
                      </div>
                    </div>
                    <label htmlFor="new-tax-multiple" className="flex items-center gap-2 cursor-pointer">
                      <Switch
                        id="new-tax-multiple"
                        checked={newTaxMultiple}
                        onCheckedChange={setNewTaxMultiple}
                      />
                      <span className="text-sm text-slate-700">Allow multiple terms per node</span>
                    </label>
                    <Button
                      type="button"
                      size="sm"
                      className="bg-indigo-600 hover:bg-indigo-700 text-white"
                      onClick={addTaxonomy}
                    >
                      <Plus className="mr-1.5 h-4 w-4" /> Add Taxonomy
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="urls" className="mt-4 ring-offset-white focus-visible:outline-none">
              <Card className="rounded-xl border border-slate-200 shadow-sm">
                <SectionHeader title="URL Prefixes" />
                <CardContent className="space-y-4">
                  <p className="text-sm text-slate-500">
                    Set the URL prefix per language. Leave empty to use the type slug as prefix.
                  </p>
                  <div className="grid gap-3 sm:grid-cols-2">
                    {languages.map((lang) => (
                      <div key={lang.code} className="space-y-1.5">
                        <Label className="text-xs font-medium text-slate-600">{lang.name} ({lang.code})</Label>
                        <div className="flex items-center rounded-lg border border-slate-300 focus-within:border-indigo-500 focus-within:ring-2 focus-within:ring-indigo-500/20 overflow-hidden">
                          <span className="shrink-0 bg-slate-100 px-2 py-2 text-sm text-slate-500 border-r border-slate-300">
                            /{lang.code}/
                          </span>
                          <input
                            placeholder={slug || "prefix"}
                            value={urlPrefixes[lang.code] || ""}
                            onChange={(e) =>
                              setUrlPrefixes((prev) => ({
                                ...prev,
                                [lang.code]: e.target.value,
                              }))
                            }
                            className="flex-1 bg-transparent px-2 py-2 text-sm outline-none"
                          />
                          <span className="shrink-0 text-sm text-slate-400 pr-2">/slug</span>
                        </div>
                      </div>
                    ))}
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
              <Button
                type="submit"
                className="w-full bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg shadow-sm h-9 text-sm"
                disabled={saving}
              >
                <Save className="mr-1.5 h-3.5 w-3.5" />
                {saving ? "Saving..." : "Save"}
              </Button>

              {isEdit && (
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

              {isEdit && originalNodeType && (
                <>
                  <Separator />
                  <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-slate-400">
                    <div className="flex justify-between">
                      <span>Created</span>
                      <span className="text-slate-600">{new Date(originalNodeType.created_at).toLocaleDateString()}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Updated</span>
                      <span className="text-slate-600">{new Date(originalNodeType.updated_at).toLocaleDateString()}</span>
                    </div>
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
                <Label htmlFor="label_plural" className="text-xs font-medium text-slate-500">Label (plural)</Label>
                <Input
                  id="label_plural"
                  placeholder="e.g. Products, Events"
                  value={labelPlural}
                  onChange={(e) => setLabelPlural(e.target.value)}
                />
                <p className="text-[11px] text-slate-400">Used in menus and list headings.</p>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="description" className="text-xs font-medium text-slate-500">Description</Label>
                <Textarea
                  id="description"
                  placeholder="What is this content type for?"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={2}
                />
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-slate-500">Icon</Label>
                <div className="grid grid-cols-4 gap-1.5">
                  {ICON_OPTIONS.map((opt) => {
                    const IconComp = opt.icon;
                    const isSelected = icon === opt.value;
                    return (
                      <button
                        key={opt.value}
                        type="button"
                        onClick={() => setIcon(opt.value)}
                        title={opt.label}
                        className={`flex items-center justify-center rounded-lg border-2 p-2 transition-all ${
                          isSelected
                            ? "border-indigo-500 bg-indigo-50 text-indigo-700"
                            : "border-slate-200 bg-white text-slate-500 hover:border-slate-300 hover:bg-slate-50"
                        }`}
                      >
                        <IconComp className="h-4 w-4" />
                      </button>
                    );
                  })}
                </div>
              </div>
              <label className="flex items-center justify-between gap-3 pt-1 cursor-pointer">
                <div className="space-y-0.5 min-w-0">
                  <span className="block text-sm text-slate-700">Supports blocks</span>
                  <span className="block text-[11px] text-slate-400">Block-based composition on nodes.</span>
                </div>
                <Switch
                  checked={supportsBlocks}
                  onCheckedChange={setSupportsBlocks}
                />
              </label>
            </CardContent>
          </Card>
        </div>
      </form>

      {/* Delete dialog */}
      <Dialog open={showDelete} onOpenChange={setShowDelete}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Content Type</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{originalNodeType?.label}&quot;?
              This action cannot be undone. All content using this type may become inaccessible.
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
