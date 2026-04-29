import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Save,
  Loader2,
  ArrowLeft,
  Globe,
  Link as LinkIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { SectionHeader } from "@/components/ui/section-header";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { toast } from "sonner";
import { usePageMeta } from "@/components/layout/page-meta";
import {
  getMenu,
  createMenu,
  updateMenu,
  replaceMenuItems,
  getLanguages,
  type MenuItem,
  type Language,
} from "@/api/client";
import MenuTree, { generateTempId } from "@/components/menu-tree";
import { Link } from "react-router-dom";

function slugify(text: string): string {
  return text
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
}

function newMenuItem(type: MenuItem["item_type"]): MenuItem {
  const base: MenuItem = {
    _uid: generateTempId(),
    title: "",
    item_type: type,
    target: "_self",
    css_class: "",
    children: [],
  };
  if (type === "custom") base.url = "";
  if (type === "node") base.node_id = null;
  return base;
}

export default function MenuEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isNew = !id;

  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [languages, setLanguages] = useState<Language[]>([]);

  // Form state
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [slugTouched, setSlugTouched] = useState(false);
  const [languageId, setLanguageId] = useState<number | null>(null);
  const [version, setVersion] = useState(1);
  const [menuItems, setMenuItems] = useState<MenuItem[]>([]);
  const [lastAddedId, setLastAddedId] = useState<string | null>(null);

  usePageMeta([
    "Menus",
    isNew ? "New Menu" : (name ? `Edit "${name}"` : "Edit"),
  ]);

  const fetchMenu = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const menu = await getMenu(id);
      setName(menu.name);
      setSlug(menu.slug);
      setSlugTouched(true);
      setLanguageId(menu.language_id);
      setVersion(menu.version);
      setMenuItems(menu.items || []);
    } catch {
      toast.error("Failed to load menu");
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    fetchMenu();
  }, [fetchMenu]);

  useEffect(() => {
    getLanguages(true)
      .then((langs) => {
        setLanguages(langs);
        if (!id && langs.length > 0) {
          const def = langs.find((l) => l.is_default);
          if (def) setLanguageId(def.id);
        }
      })
      .catch(() => {});
  }, [id]);

  function handleNameChange(val: string) {
    setName(val);
    if (!slugTouched) {
      setSlug(slugify(val));
    }
  }

  function addItem(type: MenuItem["item_type"]) {
    const item = newMenuItem(type);
    const uid = item._uid!;
    setMenuItems((prev) => [...prev, item]);
    setLastAddedId(uid);
  }

  function stripUids(items: MenuItem[]): MenuItem[] {
    return items.map((item) => {
      const clean = { ...item };
      delete clean._uid;
      if (clean.children && clean.children.length > 0) {
        clean.children = stripUids(clean.children);
      }
      return clean;
    });
  }

  async function handleSave() {
    if (!name.trim()) {
      toast.error("Menu name is required");
      return;
    }
    if (!slug.trim()) {
      toast.error("Menu slug is required");
      return;
    }

    const cleanItems = stripUids(menuItems);

    setSaving(true);
    try {
      if (isNew) {
        const menu = await createMenu({
          name,
          slug,
          language_id: languageId,
          items: cleanItems,
        });
        toast.success("Menu created successfully");
        navigate(`/admin/menus/${menu.id}`, { replace: true });
      } else {
        // Update menu metadata
        await updateMenu(id!, {
          name,
          slug,
          language_id: languageId,
        });
        // Replace menu items with version check
        try {
          const updated = await replaceMenuItems(id!, version, cleanItems);
          setVersion(updated.version);
          toast.success("Menu saved successfully");
        } catch (err: unknown) {
          if (
            err &&
            typeof err === "object" &&
            "code" in err &&
            (err as { code: string }).code === "VERSION_CONFLICT"
          ) {
            toast.error("Menu was modified by another user. Refreshing...");
            await fetchMenu();
            return;
          }
          throw err;
        }
      }
    } catch {
      toast.error("Failed to save menu");
    } finally {
      setSaving(false);
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
    {/* Match the editor template used by node / taxonomy / term editors:
        a fluid main column plus a fixed 320px sidebar. Was lg:grid-cols-3
        with col-span-2, which gave the sidebar a third of the viewport
        on wide screens — much wider than every other edit page. */}
    <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
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
            <Link to="/admin/menus" title="Back to Menus">
              <ArrowLeft className="h-3.5 w-3.5" style={{ color: "var(--fg-muted)" }} />
            </Link>
          </Button>

          {/* Name field */}
          <div className="flex items-center gap-1.5 flex-[1_1_60%] min-w-0 px-1">
            <span
              className="shrink-0 uppercase"
              style={{ fontSize: 10.5, fontWeight: 600, color: "var(--fg-muted)", letterSpacing: "0.06em" }}
            >
              Name
            </span>
            <input
              placeholder="Menu name"
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              required
              className="flex-1 min-w-0 bg-transparent outline-none"
              style={{ border: "none", padding: "6px 4px", fontSize: 14, fontWeight: 500, color: "var(--fg)" }}
            />
          </div>

          <div className="w-px h-5 shrink-0" style={{ background: "var(--border)" }} />

          {/* Slug field */}
          <div className="flex items-center gap-1 flex-[1_1_40%] min-w-0 px-1">
            <span
              className="shrink-0"
              style={{ fontSize: 11, color: "var(--fg-subtle)", fontFamily: "var(--font-mono)" }}
            >
              /
            </span>
            <input
              placeholder="menu-slug"
              value={slug}
              onChange={(e) => { setSlugTouched(true); setSlug(e.target.value); }}
              disabled={!slugTouched}
              required
              className="flex-1 min-w-0 bg-transparent outline-none disabled:opacity-60"
              style={{ border: "none", padding: "6px 0", fontSize: 12.5, color: "var(--fg)", fontFamily: "var(--font-mono)" }}
            />
            <button
              type="button"
              className="shrink-0 px-1.5 py-0.5 rounded text-[10.5px] font-medium uppercase"
              style={{
                color: !slugTouched ? "var(--accent)" : "var(--fg-muted)",
                background: !slugTouched ? "color-mix(in oklab, var(--accent) 12%, transparent)" : "var(--sub-bg)",
                border: "1px solid var(--border)",
                letterSpacing: "0.04em",
              }}
              onClick={() => {
                if (slugTouched) setSlug(slugify(name));
                setSlugTouched(!slugTouched);
              }}
              title={!slugTouched ? "Click to edit slug manually" : "Click to auto-generate from name"}
            >
              {!slugTouched ? "Auto" : "Edit"}
            </button>
          </div>

          {/* Version badge */}
          {!isNew && (
            <Badge
              variant="secondary"
              className="shrink-0 font-mono"
              style={{ fontSize: 10.5, background: "var(--sub-bg)", color: "var(--fg-muted)", border: "1px solid var(--border)" }}
            >
              v{version}
            </Badge>
          )}
        </div>

        <MenuTree items={menuItems} onChange={setMenuItems} autoEditId={lastAddedId} />

          {/* Add buttons — always visible */}
          <div className="flex gap-2">
            <Button
              variant="outline"
              className="flex-1 rounded-lg border-dashed border-slate-300 text-slate-500 hover:border-indigo-400 hover:text-indigo-600"
              onClick={() => addItem("node")}
            >
              <LinkIcon className="mr-2 h-4 w-4" />
              Add Page Link
            </Button>
            <Button
              variant="outline"
              className="flex-1 rounded-lg border-dashed border-slate-300 text-slate-500 hover:border-indigo-400 hover:text-indigo-600"
              onClick={() => addItem("custom")}
            >
              <Globe className="mr-2 h-4 w-4" />
              Add Custom URL
            </Button>
          </div>
        </div>

      {/* Sidebar */}
      <div className="space-y-6">
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <SectionHeader title="Menu Details" />
          <CardContent className="space-y-4">
            <div>
              <label className="mb-1.5 block text-sm font-medium text-slate-700">Language</label>
              <Select
                value={languageId === null ? "all" : String(languageId)}
                onValueChange={(v) => setLanguageId(v === "all" ? null : Number(v))}
              >
                <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
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
            <Button
              onClick={handleSave}
              disabled={saving}
              className="w-full bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg shadow-sm h-9 text-sm"
            >
              {saving ? <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" /> : <Save className="mr-1.5 h-3.5 w-3.5" />}
              {saving ? "Saving..." : "Save Menu"}
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
