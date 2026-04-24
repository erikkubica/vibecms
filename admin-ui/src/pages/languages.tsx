import { useEffect, useState, type FormEvent } from "react";
import { Globe, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { toast } from "sonner";
import {
  getLanguages,
  createLanguage,
  updateLanguage,
  deleteLanguage,
  type Language,
} from "@/api/client";
import {
  ListPageShell,
  ListHeader,
  ListToolbar,
  ListSearch,
  ListCard,
  ListTable,
  Th,
  Tr,
  Td,
  StatusPill,
  Chip,
  RowActions,
  EmptyState,
  LoadingRow,
} from "@/components/ui/list-page";

export default function LanguagesPage() {
  const [languages, setLanguages] = useState<Language[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [search, setSearch] = useState("");

  const [showEditor, setShowEditor] = useState(false);
  const [editingLanguage, setEditingLanguage] = useState<Language | null>(null);

  const [showDelete, setShowDelete] = useState(false);
  const [deletingLanguage, setDeletingLanguage] = useState<Language | null>(null);
  const [deleting, setDeleting] = useState(false);

  const [formCode, setFormCode] = useState("");
  const [formSlug, setFormSlug] = useState("");
  const [formName, setFormName] = useState("");
  const [formNativeName, setFormNativeName] = useState("");
  const [formFlag, setFormFlag] = useState("");
  const [formIsDefault, setFormIsDefault] = useState(false);
  const [formIsActive, setFormIsActive] = useState(true);
  const [formHidePrefix, setFormHidePrefix] = useState(false);
  const [autoSlug, setAutoSlug] = useState(true);

  async function fetchLanguages() {
    try {
      const data = await getLanguages();
      setLanguages(data);
    } catch {
      toast.error("Failed to load languages");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    fetchLanguages();
  }, []);

  function openAddDialog() {
    setEditingLanguage(null);
    setFormCode("");
    setFormSlug("");
    setFormName("");
    setFormNativeName("");
    setFormFlag("");
    setFormIsDefault(false);
    setFormIsActive(true);
    setFormHidePrefix(false);
    setAutoSlug(true);
    setShowEditor(true);
  }

  function openEditDialog(lang: Language) {
    setEditingLanguage(lang);
    setFormCode(lang.code);
    setFormSlug(lang.slug);
    setFormName(lang.name);
    setFormNativeName(lang.native_name);
    setFormFlag(lang.flag);
    setFormIsDefault(lang.is_default);
    setFormIsActive(lang.is_active);
    setFormHidePrefix(lang.hide_prefix);
    setAutoSlug(false);
    setShowEditor(true);
  }

  async function handleSave(e: FormEvent) {
    e.preventDefault();

    if (!formCode.trim() || !formName.trim()) {
      toast.error("Code and name are required");
      return;
    }

    const data: Partial<Language> = {
      code: formCode.trim().toLowerCase(),
      slug: (formSlug.trim() || formCode.trim()).toLowerCase(),
      name: formName.trim(),
      native_name: formNativeName.trim(),
      flag: formFlag.trim(),
      is_default: formIsDefault,
      is_active: formIsActive,
      hide_prefix: formHidePrefix,
    };

    setSaving(true);
    try {
      if (editingLanguage) {
        await updateLanguage(editingLanguage.id, data);
        toast.success("Language updated successfully");
      } else {
        await createLanguage(data);
        toast.success("Language created successfully");
      }
      setShowEditor(false);
      await fetchLanguages();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to save language";
      toast.error(message);
    } finally {
      setSaving(false);
    }
  }

  function openDeleteDialog(lang: Language) {
    setDeletingLanguage(lang);
    setShowDelete(true);
  }

  async function handleDelete() {
    if (!deletingLanguage) return;
    setDeleting(true);
    try {
      await deleteLanguage(deletingLanguage.id);
      toast.success("Language deleted successfully");
      setShowDelete(false);
      setDeletingLanguage(null);
      await fetchLanguages();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to delete language";
      toast.error(message);
    } finally {
      setDeleting(false);
    }
  }

  const q = search.toLowerCase();
  const filteredLanguages = q
    ? languages.filter(
        (l) =>
          l.name.toLowerCase().includes(q) ||
          l.code.toLowerCase().includes(q) ||
          l.native_name.toLowerCase().includes(q),
      )
    : languages;

  return (
    <ListPageShell>
      <ListHeader
        title="Languages"
        tabs={[{ value: "all", label: "All", count: languages.length }]}
        activeTab="all"
        newLabel="Add Language"
        onNew={openAddDialog}
      />

      <ListToolbar>
        <ListSearch value={search} onChange={setSearch} placeholder="Search languages…" />
      </ListToolbar>

      <ListCard>
        {loading ? (
          <LoadingRow />
        ) : languages.length === 0 ? (
          <EmptyState
            icon={Globe}
            title="No languages configured yet"
            description='Click "Add Language" to get started.'
          />
        ) : (
          <ListTable>
            <thead>
              <tr>
                <Th width={60}>Flag</Th>
                <Th width={90}>Code</Th>
                <Th width={110}>URL Slug</Th>
                <Th>Name</Th>
                <Th>Native Name</Th>
                <Th width={120}>Status</Th>
                <Th width={110} align="right">Actions</Th>
              </tr>
            </thead>
            <tbody>
              {filteredLanguages.map((lang) => (
                <Tr key={lang.id}>
                  <Td className="text-xl leading-none">{lang.flag}</Td>
                  <Td className="font-mono text-[12px] text-slate-700">{lang.code}</Td>
                  <Td className="font-mono text-[12px] text-indigo-600">/{lang.slug}/</Td>
                  <Td>
                    <div className="flex items-center gap-1.5">
                      <span className="text-[13px] font-medium text-slate-900">{lang.name}</span>
                      {lang.is_default && <Chip>Default</Chip>}
                      {lang.hide_prefix && <Chip>No prefix</Chip>}
                    </div>
                  </Td>
                  <Td className="text-slate-600">{lang.native_name}</Td>
                  <Td>
                    {lang.is_default ? (
                      <StatusPill status="success" label="default" />
                    ) : lang.is_active ? (
                      <StatusPill status="active" />
                    ) : (
                      <StatusPill status="inactive" />
                    )}
                  </Td>
                  <Td align="right" className="whitespace-nowrap">
                    <RowActions
                      onEdit={() => openEditDialog(lang)}
                      onDelete={lang.is_default ? undefined : () => openDeleteDialog(lang)}
                    />
                  </Td>
                </Tr>
              ))}
            </tbody>
          </ListTable>
        )}
      </ListCard>

      <Dialog open={showEditor} onOpenChange={setShowEditor}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingLanguage ? "Edit Language" : "Add Language"}</DialogTitle>
            <DialogDescription>
              {editingLanguage
                ? "Update the language details below."
                : "Fill in the details for the new language."}
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSave} className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-3">
              <div className="space-y-2">
                <Label htmlFor="lang-code" className="text-sm font-medium text-slate-700">
                  Code (ISO)
                </Label>
                <Input
                  id="lang-code"
                  placeholder="e.g. en, es, fr"
                  value={formCode}
                  onChange={(e) => {
                    setFormCode(e.target.value);
                    if (autoSlug) setFormSlug(e.target.value.toLowerCase());
                  }}
                  required
                  disabled={!!editingLanguage}
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
                {editingLanguage && <p className="text-xs text-slate-400">Code cannot be changed</p>}
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label htmlFor="lang-slug" className="text-sm font-medium text-slate-700">
                    URL Slug
                  </Label>
                  {!editingLanguage && (
                    <button
                      type="button"
                      className="text-xs text-indigo-600 hover:underline"
                      onClick={() => setAutoSlug(!autoSlug)}
                    >
                      {autoSlug ? "Edit" : "Auto"}
                    </button>
                  )}
                </div>
                <Input
                  id="lang-slug"
                  placeholder="e.g. en, english, pt-br"
                  value={formSlug}
                  onChange={(e) => {
                    setAutoSlug(false);
                    setFormSlug(e.target.value);
                  }}
                  disabled={autoSlug && !editingLanguage}
                  required
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
                <p className="text-xs text-slate-400">
                  Used in URLs: /{formSlug || formCode}/page-slug
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="lang-flag" className="text-sm font-medium text-slate-700">
                  Flag
                </Label>
                <Input
                  id="lang-flag"
                  placeholder="e.g. 🇺🇸, 🇪🇸"
                  value={formFlag}
                  onChange={(e) => setFormFlag(e.target.value)}
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="lang-name" className="text-sm font-medium text-slate-700">
                  Name
                </Label>
                <Input
                  id="lang-name"
                  placeholder="e.g. English"
                  value={formName}
                  onChange={(e) => setFormName(e.target.value)}
                  required
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="lang-native-name" className="text-sm font-medium text-slate-700">
                  Native Name
                </Label>
                <Input
                  id="lang-native-name"
                  placeholder="e.g. English"
                  value={formNativeName}
                  onChange={(e) => setFormNativeName(e.target.value)}
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>
            </div>

            <div className="flex items-center gap-6">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={formIsDefault}
                  onChange={(e) => setFormIsDefault(e.target.checked)}
                  className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                />
                <span className="text-sm font-medium text-slate-700">Default language</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={formIsActive}
                  onChange={(e) => setFormIsActive(e.target.checked)}
                  className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                />
                <span className="text-sm font-medium text-slate-700">Active</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={formHidePrefix}
                  onChange={(e) => setFormHidePrefix(e.target.checked)}
                  className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                />
                <span className="text-sm font-medium text-slate-700">Hide URL prefix</span>
              </label>
            </div>
            {formHidePrefix && (
              <p className="text-xs text-amber-600 bg-amber-50 rounded-lg px-3 py-2">
                URLs for this language won't have a prefix:{" "}
                <span className="font-mono">/page-slug</span> instead of{" "}
                <span className="font-mono">/{formSlug || formCode}/page-slug</span>. Typically used
                for the default language.
              </p>
            )}

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setShowEditor(false)}
                disabled={saving}
                className="rounded-lg border-slate-300"
              >
                Cancel
              </Button>
              <Button
                type="submit"
                className="bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg"
                disabled={saving}
              >
                {saving ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Saving...
                  </>
                ) : editingLanguage ? (
                  "Update Language"
                ) : (
                  "Create Language"
                )}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={showDelete} onOpenChange={setShowDelete}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Language</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deletingLanguage?.name}&quot; (
              {deletingLanguage?.code})? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDelete(false)} disabled={deleting}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDelete} disabled={deleting}>
              {deleting ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </ListPageShell>
  );
}
