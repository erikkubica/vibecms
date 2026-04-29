import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import {
  ArrowLeft,
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

        if (isEdit && id) {
          const term = await getTerm(Number(id));
          setName(term.name);
          setSlug(term.slug);
          setDescription(term.description || "");
          setFieldsData(term.fields_data || {});
          setLanguageCode(term.language_code);
          setParentId(term.parent_id ?? null);
          setAutoSlug(false);
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
        // only). Always loaded so toggling the taxonomy's hierarchical
        // flag in another tab doesn't require reloading this page.
        if (tax.hierarchical) {
          listTerms(nodeType!, taxSlug, { language_code: "all" })
            .then((all) => {
              const lc = isEdit && id ? undefined : currentCode;
              const filtered = all.filter((t) => {
                if (lc && t.language_code !== lc) return false;
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
        <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
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
        {/* Compact pill header */}
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
            <Link to={`/admin/content/${nodeType}/taxonomies/${taxSlug}`} title={`Back to ${taxLabel}`}>
              <ArrowLeft className="h-3.5 w-3.5" style={{ color: "var(--fg-muted)" }} />
            </Link>
          </Button>

          <div className="flex items-center gap-1.5 flex-[1_1_60%] min-w-0 px-1">
            <span
              className="shrink-0 uppercase"
              style={{ fontSize: 10.5, fontWeight: 600, color: "var(--fg-muted)", letterSpacing: "0.06em" }}
            >
              Name
            </span>
            <input
              placeholder={`Enter ${taxLabel?.toLowerCase()} name`}
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              required
              className="flex-1 min-w-0 bg-transparent outline-none"
              style={{ border: "none", padding: "6px 4px", fontSize: 14, fontWeight: 500, color: "var(--fg)" }}
            />
          </div>

          <div className="w-px h-5 shrink-0" style={{ background: "var(--border)" }} />

          <div className="flex items-center gap-1 flex-[1_1_40%] min-w-0 px-1">
            <span className="shrink-0" style={{ fontSize: 11, color: "var(--fg-subtle)", fontFamily: "var(--font-mono)" }}>/</span>
            <input
              placeholder="auto-generated"
              value={slug}
              onChange={(e) => { setAutoSlug(false); setSlug(slugify(e.target.value)); }}
              disabled={autoSlug}
              required
              className="flex-1 min-w-0 bg-transparent outline-none disabled:opacity-60"
              style={{ border: "none", padding: "6px 0", fontSize: 12.5, color: "var(--fg)", fontFamily: "var(--font-mono)" }}
            />
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
              title={autoSlug ? "Click to edit slug manually" : "Click to auto-generate from name"}
            >
              {autoSlug ? "Auto" : "Edit"}
            </button>
          </div>
        </div>

        {/* Description */}
        <Card>
          <SectionHeader title="Description" />
          <CardContent className="space-y-2">
            <Textarea
              placeholder="Optional description for this term..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={4}
              className="rounded-lg border-slate-300 text-sm focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 resize-none"
            />
            <p className="text-[11px] text-slate-400">
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
                  <Label className="text-sm font-medium text-slate-700">
                    {field.label}
                    {field.required && <span className="text-red-500 ml-1">*</span>}
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
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-50 text-indigo-500">
              <Tag className="h-4 w-4" />
            </div>
            <div>
              <p className="font-medium text-slate-800">{taxLabel}</p>
              <p className="text-[11px] text-slate-400">{nodeType} taxonomy</p>
            </div>
          </div>

          {/* Language is editable in both create and edit modes — slug
              uniqueness is per-language so a relabel only fails if another
              term in the target language already owns this slug. To clone
              into an additional language without losing the original, use
              the Translations card below instead. */}
          {languages.length > 0 && (
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-slate-500">Language</Label>
              <Select value={languageCode} onValueChange={setLanguageCode}>
                <SelectTrigger className="h-9 rounded-lg border-slate-300 text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {languages.map((lang) => (
                    <SelectItem key={lang.code} value={lang.code}>
                      {lang.name || lang.code}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          {taxonomy?.hierarchical && (
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-slate-500">Parent</Label>
              <Select
                value={parentId == null ? "__none__" : String(parentId)}
                onValueChange={(v) => setParentId(v === "__none__" ? null : Number(v))}
              >
                <SelectTrigger className="h-9 rounded-lg border-slate-300 text-sm">
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
              <p className="text-[11px] text-slate-400">
                Nests this term under another. Leave empty for a top-level term.
              </p>
            </div>
          )}

          <Button
            type="submit"
            className="w-full bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg shadow-sm h-9 text-sm"
            disabled={saving}
          >
            {saving ? <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" /> : <Save className="mr-1.5 h-3.5 w-3.5" />}
            {saving ? "Saving..." : "Save Term"}
          </Button>
          {isEdit && (
            <Button
              type="button"
              variant="outline"
              className="w-full bg-red-50 text-red-700 border-red-200 hover:bg-red-100 rounded-lg font-medium h-8 text-xs"
              onClick={() => setShowDeleteDialog(true)}
            >
              <Trash2 className="mr-1.5 h-3.5 w-3.5" />
              Delete
            </Button>
          )}
        </SidebarCard>

        {/* Translations — visible only after the term is persisted. Each
            row links to the sibling's edit page. The trailing select offers
            languages that don't yet have a translation. */}
        {isEdit && languages.length > 1 && (
          <SidebarCard title="Translations">
            <div className="space-y-1.5">
              <div className="flex items-center gap-2 rounded-md bg-indigo-50 border border-indigo-100 px-3 py-2">
                <span className="text-xs font-medium text-indigo-700 flex-1 truncate">
                  {languages.find((l) => l.code === languageCode)?.name || languageCode}
                </span>
                <span className="rounded bg-indigo-100 px-1.5 py-0.5 text-[10px] font-medium text-indigo-600">
                  Current
                </span>
              </div>
              {translations.map((t) => (
                <Link
                  key={t.id}
                  to={`/admin/content/${nodeType}/taxonomies/${taxSlug}/${t.id}/edit`}
                  className="flex items-center gap-2 rounded-md border border-slate-200 px-3 py-2 hover:bg-slate-50 transition-colors"
                >
                  <span className="text-xs font-medium text-slate-700 flex-1 truncate">
                    {languages.find((l) => l.code === t.language_code)?.name || t.language_code}
                  </span>
                </Link>
              ))}
            </div>
            {(() => {
              const taken = new Set([languageCode, ...translations.map((t) => t.language_code)]);
              const remaining = languages.filter((l) => !taken.has(l.code));
              if (remaining.length === 0) return null;
              return (
                <Select
                  value=""
                  onValueChange={(v) => v && handleCreateTranslation(v)}
                  disabled={creatingTranslation}
                >
                  <SelectTrigger className="h-9 rounded-lg border-slate-300 text-sm">
                    <SelectValue
                      placeholder={
                        creatingTranslation ? "Creating…" : "+ Add translation"
                      }
                    />
                  </SelectTrigger>
                  <SelectContent>
                    {remaining.map((lang) => (
                      <SelectItem key={lang.code} value={lang.code}>
                        {lang.name || lang.code}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              );
            })()}
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
