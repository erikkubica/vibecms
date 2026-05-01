import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Save,
  Trash2,
  Loader2,
  Boxes,
  ListTree,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent } from "@/components/ui/card";
import { SectionHeader } from "@/components/ui/section-header";
import { Separator } from "@/components/ui/separator";
import { Titlebar } from "@/components/ui/titlebar";
import { MetaRow, MetaList } from "@/components/ui/meta-row";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Switch } from "@/components/ui/switch";
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
  const [hierarchical, setHierarchical] = useState(false);

  const [availableNodeTypes, setAvailableNodeTypes] = useState<NodeType[]>([]);
  const [autoSlug, setAutoSlug] = useState(!isEdit);
  const [originalTaxonomy, setOriginalTaxonomy] = useState<Taxonomy | null>(null);

  usePageMeta([
    "Taxonomies",
    isEdit ? (label ? `Edit "${label}"` : "Edit") : "New Taxonomy",
  ]);

  useEffect(() => {
    getNodeTypes().then(setAvailableNodeTypes).catch(console.error);

    if (isEdit && urlSlug) {
      getTaxonomy(urlSlug)
        .then((t) => {
          setOriginalTaxonomy(t);
          setLabel(t.label);
          setLabelPlural(t.label_plural || "");
          setSlug(t.slug);
          setDescription(t.description || "");
          setNodeTypes(t.node_types || []);
          setFields(t.field_schema || []);
          setHierarchical(!!t.hierarchical);
          setAutoSlug(false);
        })
        .catch(() => {
          toast.error("Failed to load taxonomy");
          navigate("/admin/taxonomies");
        })
        .finally(() => setLoading(false));
    }
  }, [isEdit, urlSlug, navigate]);

  useEffect(() => {
    if (autoSlug) {
      setSlug(slugify(label));
    }
  }, [label, autoSlug]);

  const handleSave = async (e?: FormEvent) => {
    e?.preventDefault();
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
      hierarchical,
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
        <Loader2 className="h-8 w-8 animate-spin" style={{color: "var(--accent-strong)"}} />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <form onSubmit={handleSave} className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        {/* Main content */}
        <div className="space-y-4 min-w-0">
          <Titlebar
            title={label}
            onTitleChange={setLabel}
            titleLabel="Label"
            titlePlaceholder="e.g. Category, Tag"
            slug={slug}
            onSlugChange={isEdit ? undefined : (v) => { setAutoSlug(false); setSlug(v); }}
            slugPrefix=""
            autoSlug={autoSlug}
            onAutoSlugToggle={isEdit ? undefined : () => setAutoSlug(!autoSlug)}
            id={isEdit && originalTaxonomy ? originalTaxonomy.id : undefined}
            onBack={() => navigate("/admin/taxonomies")}
          />

          {/* Tabs */}
          <Tabs defaultValue="fields" className="w-full">
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="fields" className="">
                <Boxes className="mr-2 h-4 w-4" /> Term Fields
              </TabsTrigger>
              <TabsTrigger value="content-types" className="">
                <ListTree className="mr-2 h-4 w-4" /> Content Types
              </TabsTrigger>
            </TabsList>

            <TabsContent value="fields" className="mt-4 ring-offset-white focus-visible:outline-none">
              <Card className="rounded-xl border border-border shadow-sm">
                <SectionHeader title="Term Custom Fields" />
                <CardContent>
                  <p className="text-xs text-muted-foreground mb-4">
                    Fields that appear when editing terms in this taxonomy.
                  </p>
                  <FieldSchemaEditor fields={fields} onChange={setFields} />
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="content-types" className="mt-4 ring-offset-white focus-visible:outline-none">
              <Card className="rounded-xl border border-border shadow-sm">
                <SectionHeader title="Assigned Content Types" />
                <CardContent>
                  <p className="text-xs text-muted-foreground mb-4">Select which content types can use this taxonomy.</p>
                  {availableNodeTypes.length === 0 ? (
                    <p className="text-sm italic text-center py-4" style={{color: "var(--fg-subtle)"}}>No content types available.</p>
                  ) : (
                    <div className="grid gap-2 sm:grid-cols-2">
                      {availableNodeTypes.map((type) => (
                        <label
                          key={type.slug}
                          className="flex items-center justify-between gap-3 p-2.5 rounded-lg border border-border bg-muted/50 cursor-pointer hover:bg-muted transition-colors"
                        >
                          <div className="flex items-center gap-2 min-w-0 flex-1">
                            <span className="text-sm font-medium text-foreground truncate">{type.label}</span>
                            <span className="text-[10px] font-mono shrink-0" style={{color: "var(--fg-subtle)"}}>{type.slug}</span>
                          </div>
                          <Switch
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
                        </label>
                      ))}
                    </div>
                  )}
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
              <Button
                type="submit"
                className="w-full"
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
                    variant="ghost"
                    className="w-full"
                    style={{ color: "var(--danger)" }}
                    onClick={() => setShowDeleteDialog(true)}
                  >
                    <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                    Delete
                  </Button>
                </>
              )}
              {isEdit && originalTaxonomy && (
                <>
                  <div style={{ height: 1, background: "var(--divider)", margin: "4px 0" }} />
                  <MetaList>
                    {originalTaxonomy.created_at && <MetaRow label="Created" value={new Date(originalTaxonomy.created_at).toLocaleDateString("en-GB")} />}
                    {originalTaxonomy.updated_at && <MetaRow label="Updated" value={new Date(originalTaxonomy.updated_at).toLocaleDateString("en-GB")} />}
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
                <Label htmlFor="label_plural" className="text-xs font-medium text-muted-foreground">Label (plural)</Label>
                <Input
                  id="label_plural"
                  placeholder="e.g. Categories, Tags"
                  value={labelPlural}
                  onChange={(e) => setLabelPlural(e.target.value)}
                />
                <p className="text-[11px]" style={{color: "var(--fg-subtle)"}}>Used in list headings.</p>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="description" className="text-xs font-medium text-muted-foreground">Description</Label>
                <Textarea
                  id="description"
                  placeholder="What is this taxonomy used for?"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={2}
                />
              </div>
              <Separator />
              <div className="flex items-center justify-between gap-3">
                <div className="space-y-0.5 min-w-0 flex-1">
                  <Label htmlFor="hierarchical" className="text-xs font-medium text-foreground">Hierarchical</Label>
                  <p className="text-[11px]" style={{color: "var(--fg-subtle)"}}>
                    Terms can have a parent term (categories).
                    Disable for flat taxonomies (tags).
                  </p>
                </div>
                <Switch
                  id="hierarchical"
                  checked={hierarchical}
                  onCheckedChange={setHierarchical}
                />
              </div>
            </CardContent>
          </Card>
        </div>
      </form>

      {/* Delete dialog */}
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
