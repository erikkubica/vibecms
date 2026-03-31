import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Save,
  Loader2,
  ArrowLeft,
  Globe,
  Link as LinkIcon,
  Plus,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
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
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
}

function newMenuItem(type: MenuItem["item_type"]): MenuItem {
  const uid = generateTempId();
  const base: MenuItem = {
    title: "",
    item_type: type,
    target: "_self",
    css_class: "",
    children: [],
  };
  (base as Record<string, unknown>)._uid = uid;
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
    const uid = (item as Record<string, unknown>)._uid as string;
    setMenuItems((prev) => [...prev, item]);
    setLastAddedId(uid);
  }

  function stripUids(items: MenuItem[]): MenuItem[] {
    return items.map((item) => {
      const clean = { ...item };
      delete (clean as Record<string, unknown>)._uid;
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
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" asChild className="h-9 w-9">
            <Link to="/admin/menus">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold text-slate-900">
              {isNew ? "New Menu" : "Edit Menu"}
            </h1>
            {!isNew && (
              <div className="flex items-center gap-2 mt-0.5">
                <Badge className="bg-slate-100 text-slate-500 hover:bg-slate-100 border-0 text-xs">
                  v{version}
                </Badge>
              </div>
            )}
          </div>
        </div>
        <Button
          onClick={handleSave}
          disabled={saving}
          className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium"
        >
          {saving ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Save className="mr-2 h-4 w-4" />
          )}
          {saving ? "Saving..." : "Save Menu"}
        </Button>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Main content */}
        <div className="space-y-6 lg:col-span-2">
          {/* Add item buttons */}
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-slate-600">Add:</span>
            <Button
              variant="outline"
              size="sm"
              className="gap-1.5"
              onClick={() => addItem("node")}
            >
              <LinkIcon className="h-3.5 w-3.5" />
              Page Link
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="gap-1.5"
              onClick={() => addItem("custom")}
            >
              <Globe className="h-3.5 w-3.5" />
              Custom URL
            </Button>
          </div>

          {/* Menu tree */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader className="pb-3">
              <CardTitle className="text-base font-semibold text-slate-800">
                Menu Items
              </CardTitle>
            </CardHeader>
            <CardContent>
              <MenuTree items={menuItems} onChange={setMenuItems} autoEditId={lastAddedId} />
            </CardContent>
          </Card>

          {/* Bottom add buttons */}
          {menuItems.length > 0 && (
            <div className="flex items-center justify-center gap-2 rounded-lg border border-dashed border-slate-300 p-4">
              <Plus className="h-4 w-4 text-slate-400" />
              <Button
                variant="ghost"
                size="sm"
                className="text-xs"
                onClick={() => addItem("node")}
              >
                Add Page
              </Button>
              <Button
                variant="ghost"
                size="sm"
                className="text-xs"
                onClick={() => addItem("custom")}
              >
                Add Custom URL
              </Button>
            </div>
          )}
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader className="pb-4">
              <CardTitle className="text-base font-semibold text-slate-800">
                Menu Details
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <label className="mb-1.5 block text-sm font-medium text-slate-700">
                  Name
                </label>
                <Input
                  value={name}
                  onChange={(e) => handleNameChange(e.target.value)}
                  placeholder="Main Navigation"
                />
              </div>
              <div>
                <label className="mb-1.5 block text-sm font-medium text-slate-700">
                  Slug
                </label>
                <Input
                  value={slug}
                  onChange={(e) => {
                    setSlug(e.target.value);
                    setSlugTouched(true);
                  }}
                  placeholder="main-navigation"
                />
              </div>
              <div>
                <label className="mb-1.5 block text-sm font-medium text-slate-700">
                  Language
                </label>
                <select
                  className="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20"
                  value={languageId === null ? "" : String(languageId)}
                  onChange={(e) => setLanguageId(e.target.value === "" ? null : Number(e.target.value))}
                >
                  <option value="">All Languages</option>
                  {languages.map((lang) => (
                    <option key={lang.id} value={String(lang.id)}>
                      {lang.flag} {lang.name}
                    </option>
                  ))}
                </select>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
