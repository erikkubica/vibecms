import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import {
  ArrowLeft,
  Save,
  Trash2,
  Loader2,
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
import { Checkbox } from "@/components/ui/checkbox";
import FieldSchemaEditor from "@/components/ui/field-schema-editor";
import {
  getTaxonomy,
  createTaxonomy,
  updateTaxonomy,
  deleteTaxonomy,
  getNodeTypes,
  type Taxonomy,
  type NodeType,
  type NodeTypeField,
} from "@/api/client";
import { toast } from "sonner";
import { usePageMeta } from "@/components/layout/page-meta";

function slugify(text: string) {
  return text
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^\w\s-]/g, "")
    .replace(/[\s_]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export default function TaxonomyEditorPage() {
  const { slug: urlSlug } = useParams<{ slug: string }>();
  const navigate = useNavigate();
  const isEdit = !!urlSlug;

  const [loading, setLoading] = useState(isEdit);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  const [label, setLabel] = useState("");
  const [labelPlural, setLabelPlural] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [nodeTypes, setNodeTypes] = useState<string[]>([]);
  const [fields, setFields] = useState<NodeTypeField[]>([]);

  const [availableNodeTypes, setAvailableNodeTypes] = useState<NodeType[]>([]);
  const [autoSlug, setAutoSlug] = useState(!isEdit);

  usePageMeta([
    "Taxonomies",
    isEdit ? (label ? `Edit "${label}"` : "Edit") : "New Taxonomy",
  ]);

  useEffect(() => {
    getNodeTypes().then(setAvailableNodeTypes).catch(console.error);

    if (isEdit && urlSlug) {
      getTaxonomy(urlSlug)
        .then((t) => {
          setLabel(t.label);
          setLabelPlural(t.label_plural || "");
          setSlug(t.slug);
          setDescription(t.description || "");
          setNodeTypes(t.node_types || []);
          setFields(t.field_schema || []);
          setAutoSlug(false);
        })
        .catch(() => {
          toast.error("Failed to load taxonomy");
          navigate("/admin/taxonomies");
        })
        .finally(() => setLoading(false));
    }
  }, [isEdit, urlSlug, navigate]);

  const handleLabelChange = (val: string) => {
    setLabel(val);
    if (autoSlug) {
      setSlug(slugify(val));
    }
  };

  const handleSave = async (e: FormEvent) => {
    e.preventDefault();
    if (!label || !slug) {
      toast.error("Label and slug are required");
      return;
    }

    const data: Partial<Taxonomy> = {
      label,
      label_plural: labelPlural,
      slug,
      description,
      node_types: nodeTypes,
      field_schema: fields,
    };

    setSaving(true);
    try {
      if (isEdit && urlSlug) {
        await updateTaxonomy(urlSlug, data);
        toast.success("Taxonomy updated");
      } else {
        await createTaxonomy(data);
        toast.success("Taxonomy created");
        navigate("/admin/taxonomies");
      }
    } catch (err: any) {
      toast.error(err.message || "Failed to save taxonomy");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!urlSlug) return;
    setDeleting(true);
    try {
      await deleteTaxonomy(urlSlug);
      toast.success("Taxonomy deleted");
      navigate("/admin/taxonomies");
    } catch (err: any) {
      toast.error(err.message || "Failed to delete taxonomy");
    } finally {
      setDeleting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild className="rounded-lg">
            <Link to="/admin/taxonomies">
              <ArrowLeft className="h-5 w-5" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold text-slate-900">
              {isEdit ? `Edit Taxonomy: ${label}` : "Create Taxonomy"}
            </h1>
            <p className="text-sm text-slate-500">
              Define how your content is classified.
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          {isEdit && (
            <Button
              variant="outline"
              className="text-red-600 hover:bg-red-50 hover:text-red-700"
              onClick={() => setShowDeleteDialog(true)}
            >
              <Trash2 className="mr-2 h-4 w-4" />
              Delete
            </Button>
          )}
          <Button onClick={handleSave} disabled={saving} className="bg-indigo-600 hover:bg-indigo-700">
            {saving ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Save className="mr-2 h-4 w-4" />
            )}
            Save Taxonomy
          </Button>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-6">
          {/* General Info */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="General Information" />
            <CardContent className="space-y-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="label">Display Label (singular)</Label>
                  <Input
                    id="label"
                    value={label}
                    onChange={(e) => handleLabelChange(e.target.value)}
                    placeholder="e.g. Category, Tag, Genre"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="label_plural">Display Label (plural)</Label>
                  <Input
                    id="label_plural"
                    value={labelPlural}
                    onChange={(e) => setLabelPlural(e.target.value)}
                    placeholder="e.g. Categories, Tags, Genres"
                  />
                  <p className="text-xs text-slate-500">Used in list headings. Falls back to singular if blank.</p>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="slug">Slug (Unique Key)</Label>
                  <Input
                    id="slug"
                    value={slug}
                    onChange={(e) => {
                      setSlug(slugify(e.target.value));
                      setAutoSlug(false);
                    }}
                    disabled={isEdit}
                    placeholder="category"
                    className="font-mono text-sm"
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="What is this taxonomy used for?"
                  rows={2}
                />
              </div>
            </CardContent>
          </Card>

          {/* Assigned Node Types */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="Assigned Content Types" />
            <CardContent>
              <p className="text-xs text-slate-500 mb-4">Select which content types can use this taxonomy.</p>
              <div className="grid gap-4 sm:grid-cols-2 md:grid-cols-3">
                {availableNodeTypes.map((type) => (
                  <div key={type.slug} className="flex items-center space-x-2 p-2 rounded-lg border border-slate-100 bg-slate-50/50">
                    <Checkbox
                      id={`type-${type.slug}`}
                      checked={nodeTypes.includes(type.slug)}
                      onCheckedChange={(checked: boolean) => {
                        if (checked) {
                          setNodeTypes([...nodeTypes, type.slug]);
                        } else {
                          setNodeTypes(nodeTypes.filter(s => s !== type.slug));
                        }
                      }}
                    />
                    <Label htmlFor={`type-${type.slug}`} className="text-sm font-medium cursor-pointer flex-1">
                      {type.label}
                    </Label>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* Custom Fields */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="Term Custom Fields" />
            <CardContent className="space-y-4">
              <FieldSchemaEditor fields={fields} onChange={setFields} title="Term Custom Fields" description="Fields that appear when editing terms in this taxonomy." />
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="Settings" />
            <CardContent className="space-y-4">
              <div className="p-3 rounded-lg bg-indigo-50 border border-indigo-100 text-xs text-indigo-700 leading-relaxed">
                Assigning a taxonomy to a content type adds a term picker to that type's editor.
                Custom fields added here will appear when you edit individual terms in this taxonomy.
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Delete Confirmation */}
      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Taxonomy?</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete <strong>{label}</strong>?
              This will remove the taxonomy definition and all its terms. Existing nodes will lose their assignments.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setShowDeleteDialog(false)} disabled={deleting}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDelete} disabled={deleting}>
              {deleting ? "Deleting..." : "Delete Permanently"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
