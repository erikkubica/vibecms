import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import {
  Save,
  Trash2,
  Loader2,
  Tag,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent } from "@/components/ui/card";
import { SectionHeader } from "@/components/ui/section-header";
import { SidebarCard } from "@/components/ui/sidebar-card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  getTerm,
  createTerm,
  updateTerm,
  deleteTerm,
  getTaxonomy,
  getTermTranslations,
  createTermTranslation,
  listTerms,
  type TaxonomyTerm,
  type Taxonomy,
} from "@/api/client";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { LanguageSelect, LanguageLabel } from "@/components/ui/language-select";
import { Titlebar } from "@/components/ui/titlebar";
import { MetaRow, MetaList } from "@/components/ui/meta-row";
import { toast } from "sonner";
import CustomFieldInput from "@/components/ui/custom-field-input";
import { usePageMeta } from "@/components/layout/page-meta";
import { useAdminLanguage } from "@/hooks/use-admin-language";

function slugify(text: string) {
  return text
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^\w\s-]/g, "")
    .replace(/[\s_]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export default function TermEditorPage() {
  const {
    nodeType,
    taxonomy: taxSlug,
    id,
  } = useParams<{ nodeType: string; taxonomy: string; id: string }>();
  const navigate = useNavigate();
  const isEdit = !!id;

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [fieldsData, setFieldsData] = useState<Record<string, any>>({});
  const [taxonomy, setTaxonomy] = useState<Taxonomy | null>(null);
  const [languageCode, setLanguageCode] = useState<string>("");
  const [translations, setTranslations] = useState<TaxonomyTerm[]>([]);
  const [creatingTranslation, setCreatingTranslation] = useState(false);
  const [parentId, setParentId] = useState<number | null>(null);
  const [siblingTerms, setSiblingTerms] = useState<TaxonomyTerm[]>([]);
  const [originalTerm, setOriginalTerm] = useState<TaxonomyTerm | null>(null);

  const [autoSlug, setAutoSlug] = useState(!isEdit);

  const { languages, currentCode } = useAdminLanguage();

  usePageMeta([
    "Taxonomies",
    taxonomy?.label || taxSlug || "",
    isEdit ? (name ? `Edit "${name}"` : "Edit") : "New Term",
  ].filter(Boolean) as string[]);

  useEffect(() => {
    if (!taxSlug) return;

    const loadData = async () => {
      try {
        const tax = await getTaxonomy(taxSlug);
        setTaxonomy(tax);

        // Resolve the term's effective language up-front so the parent
        // picker can scope siblings correctly. Edit mode uses the term's
        // own language_code; create mode uses the admin's current locale.
        let effectiveLang = currentCode;
        if (isEdit && id) {
          const term = await getTerm(Number(id));
          setOriginalTerm(term);
          setName(term.name);
          setSlug(term.slug);
          setDescription(term.description || "");
          setFieldsData(term.fields_data || {});
          setLanguageCode(term.language_code);
          setParentId(term.parent_id ?? null);
          setAutoSlug(false);
          effectiveLang = term.language_code;
          // Translations are best-effort: if the endpoint fails (or no
          // siblings exist) we just show an empty panel.
          getTermTranslations(Number(id))
            .then(setTranslations)
            .catch(() => setTranslations([]));
        } else {
          // Create flow: default to the admin's current language so the
          // term lands in whatever locale the operator is editing in.
          setLanguageCode(currentCode);
        }

        // Sibling terms power the parent picker (hierarchical taxonomies
        // only). Always scoped to the term's own language — a Spanish
        // term must only see Spanish parents, otherwise hierarchies
        // cross-pollinate across translations and break breadcrumbs.
        if (tax.hierarchical && effectiveLang) {
          listTerms(nodeType!, taxSlug, { language_code: effectiveLang })
            .then((all) => {
              const filtered = all.filter((t) => {
                if (isEdit && id && t.id === Number(id)) return false;
                return true;
              });
              setSiblingTerms(filtered);
            })
            .catch(() => setSiblingTerms([]));
        }
      } catch {
        toast.error("Failed to load data");
        navigate(-1);
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, [isEdit, id, taxSlug, navigate, currentCode]);

  // Reload sibling parent options whenever the term's language changes
  // (operators can flip the language in the sidebar before saving). The
  // parent picker only shows same-language candidates, so a stale list
  // would surface English parents on a Spanish term.
  useEffect(() => {
    if (!taxonomy?.hierarchical || !languageCode || !taxSlug || !nodeType) return;
    listTerms(nodeType, taxSlug, { language_code: languageCode })
      .then((all) => {
        const filtered = all.filter((t) => !(isEdit && id && t.id === Number(id)));
        setSiblingTerms(filtered);
      })
      .catch(() => setSiblingTerms([]));
  }, [languageCode, taxonomy?.hierarchical, taxSlug, nodeType, isEdit, id]);

  const handleNameChange = (val: string) => {
    setName(val);
    if (autoSlug) {
      setSlug(slugify(val));
    }
  };

  const updateFieldValue = (key: string, value: any) => {
    setFieldsData((prev) => ({ ...prev, [key]: value }));
  };

  const handleSave = async (e: FormEvent) => {
    e.preventDefault();
    if (!name || !slug) {
      toast.error("Name and slug are required");
      return;
    }

    const data: Partial<TaxonomyTerm> = {
      name,
      slug,
      description,
      fields_data: fieldsData,
      language_code: languageCode || currentCode,
      parent_id: parentId ?? undefined,
    };

    setSaving(true);
    try {
      if (isEdit && id) {
        await updateTerm(Number(id), data);
        toast.success("Term updated");
      } else {
        await createTerm(nodeType!, taxSlug!, data);
        toast.success("Term created");
      }
      navigate(`/admin/content/${nodeType}/taxonomies/${taxSlug}`);
    } catch (err: any) {
      toast.error(err.message || "Failed to save term");
    } finally {
      setSaving(false);
    }
  };

  const handleCreateTranslation = async (langCode: string) => {
    if (!id) return;
    setCreatingTranslation(true);
    try {
      const created = await createTermTranslation(Number(id), { language_code: langCode });
      toast.success(`Translation created in ${langCode}`);
      navigate(`/admin/content/${nodeType}/taxonomies/${taxSlug}/${created.id}/edit`);
    } catch (err: any) {
      toast.error(err.message || "Failed to create translation");
    } finally {
      setCreatingTranslation(false);
    }
  };

  const handleDelete = async () => {
    if (!id) return;
    setDeleting(true);
    try {
      await deleteTerm(Number(id));
      toast.success("Term deleted");
      navigate(`/admin/content/${nodeType}/taxonomies/${taxSlug}`);
    } catch (err: any) {
      toast.error(err.message || "Failed to delete term");
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

  const taxLabel = taxonomy?.label || taxSlug;
  const customFields = (taxonomy?.field_schema || []).map((f: any) => ({ ...f, key: f.key || f.name }));

  return (
    <>
    <form onSubmit={handleSave} className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
      {/* Main content */}
      <div className="space-y-4 min-w-0">
        <Titlebar
          title={name}
          onTitleChange={handleNameChange}
          titleLabel="Name"
          titlePlaceholder={`Enter ${taxLabel?.toLowerCase()} name`}
          slug={slug}
          onSlugChange={(v) => { setAutoSlug(false); setSlug(slugify(v)); }}
          slugPrefix="/"
          autoSlug={autoSlug}
          onAutoSlugToggle={() => setAutoSlug(!autoSlug)}
          id={isEdit && id ? Number(id) : undefined}
          onBack={() => navigate(`/admin/content/${nodeType}/taxonomies/${taxSlug}`)}
        />

        {/* Description */}
        <Card>
          <SectionHeader title="Description" />
          <CardContent className="space-y-2">
            <Textarea
              placeholder="Optional description for this term..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={4}
              className="rounded-lg border-border text-sm focus:ring-2 resize-none"
            />
            <p className="text-[11px]" style={{color: "var(--fg-subtle)"}}>
              Some themes display term descriptions on archive pages.
            </p>
          </CardContent>
        </Card>

        {/* Custom Fields */}
        {customFields.length > 0 && (
          <Card>
            <SectionHeader title="Custom Fields" />
            <CardContent className="space-y-4">
              {customFields.map((field: any) => (
                <div key={field.name} className="space-y-1.5">
                  <Label className="text-sm font-medium text-foreground">
                    {field.label}
                    {field.required && <span className="ml-1" style={{color: "var(--danger)"}}>*</span>}
                  </Label>
                  <CustomFieldInput
                    field={field}
                    value={fieldsData[field.name]}
                    onChange={(val) => updateFieldValue(field.name, val)}
                  />
                </div>
              ))}
            </CardContent>
          </Card>
        )}
      </div>

      {/* Sidebar */}
      <aside className="space-y-4 lg:sticky lg:top-4 lg:self-start">
        <SidebarCard title="Publish">
          <div className="flex items-center gap-2 text-sm">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg" style={{background: "var(--accent-weak)", color: "var(--accent-strong)"}}>
              <Tag className="h-4 w-4" />
            </div>
            <div>
              <p className="font-medium text-foreground">{taxLabel}</p>
              <p className="text-[11px]" style={{color: "var(--fg-subtle)"}}>{nodeType} taxonomy</p>
            </div>
          </div>

          {/* Language is editable in both create and edit modes — slug
              uniqueness is per-language so a relabel only fails if another
              term in the target language already owns this slug. To clone
              into an additional language without losing the original, use
              the Translations card below instead. */}
          {languages.length > 0 && (
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-muted-foreground">Language</Label>
              <LanguageSelect
                languages={languages}
                value={languageCode}
                onChange={setLanguageCode}
              />
            </div>
          )}

          {taxonomy?.hierarchical && (
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-muted-foreground">Parent</Label>
              <Select
                value={parentId == null ? "__none__" : String(parentId)}
                onValueChange={(v) => setParentId(v === "__none__" ? null : Number(v))}
              >
                <SelectTrigger className="h-9 rounded-lg border-border text-sm">
                  <SelectValue placeholder="No parent (top-level)" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">— No parent —</SelectItem>
                  {siblingTerms.map((t) => (
                    <SelectItem key={t.id} value={String(t.id)}>
                      {t.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-[11px]" style={{color: "var(--fg-subtle)"}}>
                Nests this term under another. Leave empty for a top-level term.
              </p>
            </div>
          )}

          <Button
            type="submit"
            className="w-full"
            disabled={saving}
          >
            {saving ? <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" /> : <Save className="mr-1.5 h-3.5 w-3.5" />}
            {saving ? "Saving..." : "Save Term"}
          </Button>
          {isEdit && (
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
          )}
          {isEdit && originalTerm && (
            <>
              <div style={{ height: 1, background: "var(--divider)", margin: "4px 0" }} />
              <MetaList>
                {originalTerm.created_at && <MetaRow label="Created" value={new Date(originalTerm.created_at).toLocaleDateString("en-GB")} />}
                {originalTerm.updated_at && <MetaRow label="Updated" value={new Date(originalTerm.updated_at).toLocaleDateString("en-GB")} />}
              </MetaList>
            </>
          )}
        </SidebarCard>

        {/* Translations — visible only after the term is persisted. Each
            row links to the sibling's edit page. The trailing select offers
            languages that don't yet have a translation. */}
        {isEdit && languages.length > 1 && (
          <SidebarCard title="Translations">
            <div className="space-y-1.5">
              <div className="flex items-center gap-2 rounded-md border px-3 py-2" style={{background: "var(--accent-weak)", borderColor: "var(--accent-mid)"}}>
                <span className="text-xs font-medium flex-1 truncate" style={{color: "var(--accent-strong)"}}>
                  <LanguageLabel languages={languages} code={languageCode} />
                </span>
                <span className="rounded px-1.5 py-0.5 text-[10px] font-medium" style={{background: "var(--accent-weak)", color: "var(--accent-strong)"}}>
                  Current
                </span>
              </div>
              {translations.map((t) => (
                <Link
                  key={t.id}
                  to={`/admin/content/${nodeType}/taxonomies/${taxSlug}/${t.id}/edit`}
                  className="flex items-center gap-2 rounded-md border border-border px-3 py-2 hover:bg-muted transition-colors"
                >
                  <span className="text-xs font-medium text-foreground flex-1 truncate">
                    <LanguageLabel languages={languages} code={t.language_code} />
                  </span>
                </Link>
              ))}
            </div>
            <LanguageSelect
              mode="add"
              languages={languages}
              existing={[languageCode, ...translations.map((t) => t.language_code)]}
              onAdd={handleCreateTranslation}
              disabled={creatingTranslation}
              placeholder={creatingTranslation ? "Creating…" : "+ Add translation"}
            />
          </SidebarCard>
        )}
      </aside>
    </form>

      {/* Delete Confirmation */}
      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete {taxLabel}?</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete <strong>{name}</strong>? This
              action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDeleteDialog(false)}
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
    </>
  );
}
