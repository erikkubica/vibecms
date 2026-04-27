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
  type LucideIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent } from "@/components/ui/card";
import { SectionHeader } from "@/components/ui/section-header";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Separator } from "@/components/ui/separator";
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

  // Form state
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

  usePageMeta([
    "Content Types",
    isEdit ? (label ? `Edit "${label}"` : "Edit") : "New Content Type",
  ]);

  // Languages
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

  // Auto-generate slug from label
  useEffect(() => {
    if (autoSlug) {
      setSlug(slugify(label));
    }
  }, [label, autoSlug]);

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

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild className="rounded-lg hover:bg-slate-200">
          <Link to="/admin/content-types">
            <ArrowLeft className="h-5 w-5 text-slate-600" />
          </Link>
        </Button>
        <h1 className="text-2xl font-bold text-slate-900">
          {isEdit ? `Edit Content Type` : `New Content Type`}
        </h1>
      </div>

      <form onSubmit={handleSave} className="grid gap-6 lg:grid-cols-3">
        {/* Main content */}
        <div className="space-y-6 lg:col-span-2">
          {/* Basic info */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="Basic Info" />
            <CardContent className="space-y-4 p-6">
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="label" className="text-sm font-medium text-slate-700">Label (singular)</Label>
                  <Input
                    id="label"
                    placeholder="e.g. Product, Event, Testimonial"
                    value={label}
                    onChange={(e) => setLabel(e.target.value)}
                    required
                    className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="label_plural" className="text-sm font-medium text-slate-700">Label (plural)</Label>
                  <Input
                    id="label_plural"
                    placeholder="e.g. Products, Events, Testimonials"
                    value={labelPlural}
                    onChange={(e) => setLabelPlural(e.target.value)}
                    className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                  />
                  <p className="text-xs text-slate-500">Used in menus and list headings. Falls back to singular if blank.</p>
                </div>
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
                  placeholder="url-slug"
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
                  placeholder="A brief description of this content type"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={3}
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>

              <div className="space-y-1 pt-1">
                <label className="flex items-center gap-2 text-sm font-medium text-slate-700 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={supportsBlocks}
                    onChange={(e) => setSupportsBlocks(e.target.checked)}
                    className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                  />
                  Supports blocks
                </label>
                <p className="text-xs text-slate-500 pl-6">
                  Allow block-based composition on this content type. Disable when content
                  is rendered entirely from fields (e.g. custom post types with fixed layouts).
                </p>
              </div>

              <div className="space-y-2">
                <Label className="text-sm font-medium text-slate-700">Icon</Label>
                <div className="grid grid-cols-6 gap-2">
                  {ICON_OPTIONS.map((opt) => {
                    const IconComp = opt.icon;
                    const isSelected = icon === opt.value;
                    return (
                      <button
                        key={opt.value}
                        type="button"
                        onClick={() => setIcon(opt.value)}
                        title={opt.label}
                        className={`flex flex-col items-center gap-1 rounded-lg border-2 p-2.5 transition-all ${
                          isSelected
                            ? "border-indigo-500 bg-indigo-50 text-indigo-700 shadow-sm"
                            : "border-slate-200 bg-white text-slate-500 hover:border-slate-300 hover:bg-slate-50"
                        }`}
                      >
                        <IconComp className="h-5 w-5" />
                        <span className="text-[10px] font-medium leading-none">{opt.label}</span>
                      </button>
                    );
                  })}
                </div>
              </div>
            </CardContent>
          </Card>

          {/* URL Prefixes */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="URL Prefixes" />
            <CardContent className="space-y-4 p-6">
              <p className="text-sm text-slate-500">
                Set the URL prefix per language. For example, a "Team Member" type could use
                <span className="font-mono text-indigo-600"> /en/team/</span> in English and
                <span className="font-mono text-indigo-600"> /es/equipo/</span> in Spanish.
                Leave empty to use the type slug as prefix.
              </p>
              <div className="grid gap-3 sm:grid-cols-2">
                {languages.map((lang) => (
                  <div key={lang.code} className="space-y-1">
                    <Label className="text-sm font-medium text-slate-700">{lang.name} ({lang.code})</Label>
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

          {/* Taxonomies */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="Taxonomies" />
            <CardContent className="space-y-4 p-6">
              <p className="text-sm text-slate-500">
                Register custom taxonomies (e.g. Categories, Tags, Genres) to classify your content.
              </p>

              {Array.isArray(taxonomies) && taxonomies.length > 0 ? (
                <div className="space-y-2">
                  {taxonomies.map((tax, index) => (
                    <div key={tax.slug} className="flex items-center justify-between p-3 rounded-lg border border-slate-200 bg-slate-50">
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium text-slate-800">{tax.label}</span>
                          <span className="text-xs text-slate-400 font-mono">{tax.slug}</span>
                        </div>
                        <span className="text-xs text-slate-500">{tax.multiple ? "Multiple terms" : "Single term"}</span>
                      </div>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-red-500 hover:text-red-600"
                        onClick={() => setTaxonomies(taxonomies.filter((_, i) => i !== index))}
                      >
                        <X className="h-4 w-4" />
                      </Button>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-slate-400 italic text-center py-2">No taxonomies registered.</p>
              )}

              <Separator />

              <div className="space-y-4 pt-2">
                <p className="text-xs font-semibold text-slate-700 uppercase tracking-wider">Add Taxonomy</p>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-2">
                    <Label className="text-xs font-medium text-slate-700">Label</Label>
                    <Input
                      id="new-tax-label"
                      placeholder="e.g. Category, Tag"
                      className="h-9 text-sm rounded-lg"
                      onKeyDown={(e) => {
                        if (e.key === "Enter") {
                          e.preventDefault();
                          const labelInput = e.currentTarget;
                          const keyInput = document.getElementById("new-tax-key") as HTMLInputElement;
                          const multipleInput = document.getElementById("new-tax-multiple") as HTMLInputElement;
                          
                          const labelVal = labelInput.value.trim();
                          const keyVal = keyInput.value.trim() || keyify(labelVal);
                          
                          if (labelVal && keyVal) {
                            const currentTaxes = Array.isArray(taxonomies) ? taxonomies : [];
                            if (currentTaxes.some(t => t.slug === keyVal)) {
                              toast.error("Taxonomy with this key already exists");
                              return;
                            }
                            setTaxonomies([...currentTaxes, {
                              label: labelVal,
                              slug: keyVal,
                              multiple: multipleInput.checked
                            }]);
                            labelInput.value = "";
                            keyInput.value = "";
                            multipleInput.checked = true;
                          }
                        }
                      }}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label className="text-xs font-medium text-slate-700">Key (slug)</Label>
                    <Input
                      id="new-tax-key"
                      placeholder="category, tag"
                      className="h-9 text-sm font-mono rounded-lg"
                    />
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="new-tax-multiple"
                    defaultChecked
                    className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                  />
                  <Label htmlFor="new-tax-multiple" className="text-sm text-slate-700 cursor-pointer">
                    Allow multiple terms per node
                  </Label>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="rounded-lg"
                  onClick={() => {
                    const labelInput = document.getElementById("new-tax-label") as HTMLInputElement;
                    const keyInput = document.getElementById("new-tax-key") as HTMLInputElement;
                    const multipleInput = document.getElementById("new-tax-multiple") as HTMLInputElement;
                    
                    const labelVal = labelInput.value.trim();
                    const keyVal = keyInput.value.trim() || keyify(labelVal);
                    
                    if (labelVal && keyVal) {
                      const currentTaxes = Array.isArray(taxonomies) ? taxonomies : [];
                      if (currentTaxes.some(t => t.slug === keyVal)) {
                        toast.error("Taxonomy with this key already exists");
                        return;
                      }
                      setTaxonomies([...currentTaxes, {
                        label: labelVal,
                        slug: keyVal,
                        multiple: multipleInput.checked
                      }]);
                      labelInput.value = "";
                      keyInput.value = "";
                      multipleInput.checked = true;
                    } else {
                      toast.error("Label and key are required");
                    }
                  }}
                >
                  <Plus className="mr-2 h-4 w-4" />
                  Add Taxonomy
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* Fields */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="Fields" />
            <CardContent className="space-y-4 p-6">
              <FieldSchemaEditor fields={fields} onChange={setFields} />
            </CardContent>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="Save" />
            <CardContent className="space-y-4 p-6">
              <Button
                type="submit"
                className="w-full bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg shadow-sm"
                disabled={saving}
              >
                <Save className="mr-2 h-4 w-4" />
                {saving ? "Saving..." : isEdit ? "Update Content Type" : "Create Content Type"}
              </Button>
            </CardContent>
          </Card>

          {/* Actions (edit mode only) */}
          {isEdit && (
            <Card className="rounded-xl border border-slate-200 shadow-sm">
              <SectionHeader title="Actions" />
              <CardContent className="space-y-2 p-6">
                <Button
                  type="button"
                  variant="outline"
                  className="w-full bg-red-50 text-red-700 border-red-200 hover:bg-red-100 rounded-lg font-medium"
                  onClick={() => setShowDelete(true)}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete Content Type
                </Button>
              </CardContent>
            </Card>
          )}

          {/* Info (edit mode) */}
          {isEdit && originalNodeType && (
            <Card className="rounded-xl border border-slate-200 shadow-sm">
              <CardContent className="space-y-2 p-6 text-sm text-slate-500">
                <div className="flex justify-between">
                  <span>Created</span>
                  <span>
                    {new Date(originalNodeType.created_at).toLocaleDateString()}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span>Updated</span>
                  <span>
                    {new Date(originalNodeType.updated_at).toLocaleDateString()}
                  </span>
                </div>
              </CardContent>
            </Card>
          )}
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
