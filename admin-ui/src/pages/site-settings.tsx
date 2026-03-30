import { useEffect, useState, useCallback } from "react";
import { Save, Loader2, RefreshCw, Globe, Home, FileText, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { toast } from "sonner";
import {
  getSiteSettings,
  updateSiteSettings,
  clearCache,
  getNodes,
  type ContentNode,
} from "@/api/client";

interface SettingsState {
  site_name: string;
  site_url: string;
  site_description: string;
  homepage_node_id: string;
  analytics_code: string;
  custom_head_code: string;
}

const DEFAULT_SETTINGS: SettingsState = {
  site_name: "",
  site_url: "",
  site_description: "",
  homepage_node_id: "",
  analytics_code: "",
  custom_head_code: "",
};

export default function SiteSettingsPage() {
  const [settings, setSettings] = useState<SettingsState>(DEFAULT_SETTINGS);
  const [original, setOriginal] = useState<SettingsState>(DEFAULT_SETTINGS);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [clearing, setClearing] = useState(false);

  // Homepage picker
  const [pages, setPages] = useState<ContentNode[]>([]);
  const [pageSearch, setPageSearch] = useState("");
  const [showPagePicker, setShowPagePicker] = useState(false);
  const [selectedPage, setSelectedPage] = useState<ContentNode | null>(null);

  const fetchSettings = useCallback(async () => {
    setLoading(true);
    try {
      const [data, pagesRes] = await Promise.all([
        getSiteSettings(),
        getNodes({ page: 1, per_page: 200, status: "published" }),
      ]);
      const loaded: SettingsState = {
        site_name: data.site_name || "",
        site_url: data.site_url || "",
        site_description: data.site_description || "",
        homepage_node_id: data.homepage_node_id || "",
        analytics_code: data.analytics_code || "",
        custom_head_code: data.custom_head_code || "",
      };
      setSettings(loaded);
      setOriginal(loaded);
      setPages(pagesRes.data);

      // Find selected homepage
      if (data.homepage_node_id) {
        const hp = pagesRes.data.find(
          (p) => String(p.id) === data.homepage_node_id
        );
        if (hp) setSelectedPage(hp);
      }
    } catch {
      toast.error("Failed to load settings");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSettings();
  }, [fetchSettings]);

  async function handleSave() {
    setSaving(true);
    try {
      const updates: Record<string, string> = {};
      for (const key of Object.keys(settings) as (keyof SettingsState)[]) {
        if (settings[key] !== original[key]) {
          updates[key] = settings[key];
        }
      }
      if (Object.keys(updates).length === 0) {
        toast.info("No changes to save");
        setSaving(false);
        return;
      }
      await updateSiteSettings(updates);
      setOriginal({ ...settings });
      toast.success("Settings saved successfully");
    } catch {
      toast.error("Failed to save settings");
    } finally {
      setSaving(false);
    }
  }

  async function handleClearCache() {
    setClearing(true);
    try {
      await clearCache();
      toast.success("All caches cleared");
    } catch {
      toast.error("Failed to clear caches");
    } finally {
      setClearing(false);
    }
  }

  function update(key: keyof SettingsState, value: string) {
    setSettings((prev) => ({ ...prev, [key]: value }));
  }

  function selectHomepage(page: ContentNode) {
    setSelectedPage(page);
    update("homepage_node_id", String(page.id));
    setShowPagePicker(false);
    setPageSearch("");
  }

  const filteredPages = pageSearch
    ? pages.filter(
        (p) =>
          p.title.toLowerCase().includes(pageSearch.toLowerCase()) ||
          p.full_url.toLowerCase().includes(pageSearch.toLowerCase())
      )
    : pages;

  const hasChanges = JSON.stringify(settings) !== JSON.stringify(original);

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
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Site Settings</h1>
          <p className="text-sm text-slate-500 mt-0.5">
            Configure your site's core settings
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={handleClearCache}
            disabled={clearing}
            className="rounded-lg font-medium"
          >
            <RefreshCw
              className={`mr-2 h-4 w-4 ${clearing ? "animate-spin" : ""}`}
            />
            {clearing ? "Clearing..." : "Clear Cache"}
          </Button>
          <Button
            onClick={handleSave}
            disabled={saving || !hasChanges}
            className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium"
          >
            <Save className="mr-2 h-4 w-4" />
            {saving ? "Saving..." : "Save Changes"}
          </Button>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* General */}
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <CardHeader className="pb-3">
            <div className="flex items-center gap-2">
              <Globe className="h-5 w-5 text-indigo-500" />
              <div>
                <CardTitle className="text-base font-semibold text-slate-900">
                  General
                </CardTitle>
                <CardDescription className="text-xs text-slate-500">
                  Basic site identity
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4 pt-0">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-slate-700">
                Site Name
              </Label>
              <Input
                placeholder="My Website"
                value={settings.site_name}
                onChange={(e) => update("site_name", e.target.value)}
                className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-slate-700">
                Site URL
              </Label>
              <Input
                placeholder="https://example.com"
                value={settings.site_url}
                onChange={(e) => update("site_url", e.target.value)}
                className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
              />
              <p className="text-[11px] text-slate-400">
                Used for sitemaps, canonical URLs, and absolute links
              </p>
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-slate-700">
                Site Description
              </Label>
              <Textarea
                placeholder="A short description of your website..."
                value={settings.site_description}
                onChange={(e) => update("site_description", e.target.value)}
                rows={2}
                className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 resize-none"
              />
            </div>
          </CardContent>
        </Card>

        {/* Homepage */}
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <CardHeader className="pb-3">
            <div className="flex items-center gap-2">
              <Home className="h-5 w-5 text-emerald-500" />
              <div>
                <CardTitle className="text-base font-semibold text-slate-900">
                  Homepage
                </CardTitle>
                <CardDescription className="text-xs text-slate-500">
                  Choose which page visitors see first
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-3 pt-0">
            {/* Selected page display */}
            {selectedPage ? (
              <div className="flex items-center justify-between rounded-lg border border-emerald-200 bg-emerald-50 p-3">
                <div>
                  <p className="text-sm font-medium text-slate-800">
                    {selectedPage.title}
                  </p>
                  <p className="text-xs text-slate-500">
                    {selectedPage.full_url} &middot;{" "}
                    {selectedPage.language_code.toUpperCase()}
                  </p>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  className="text-xs rounded-lg"
                  onClick={() => setShowPagePicker(!showPagePicker)}
                >
                  Change
                </Button>
              </div>
            ) : (
              <Button
                variant="outline"
                className="w-full rounded-lg border-dashed border-slate-300 text-slate-500 hover:border-indigo-300 hover:text-indigo-600"
                onClick={() => setShowPagePicker(true)}
              >
                <Home className="mr-2 h-4 w-4" />
                Select a homepage
              </Button>
            )}

            {/* Page picker */}
            {showPagePicker && (
              <div className="rounded-lg border border-slate-200 bg-white shadow-sm">
                <div className="relative p-2">
                  <Search className="absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                  <Input
                    placeholder="Search pages..."
                    value={pageSearch}
                    onChange={(e) => setPageSearch(e.target.value)}
                    className="pl-8 h-8 rounded-md border-slate-200 text-sm"
                    autoFocus
                  />
                </div>
                <div className="max-h-48 overflow-y-auto border-t border-slate-100">
                  {filteredPages.length === 0 ? (
                    <p className="py-4 text-center text-sm text-slate-400">
                      No pages found
                    </p>
                  ) : (
                    filteredPages.map((page) => (
                      <button
                        key={page.id}
                        onClick={() => selectHomepage(page)}
                        className={`w-full flex items-center justify-between px-3 py-2 text-left hover:bg-slate-50 transition-colors ${
                          String(page.id) === settings.homepage_node_id
                            ? "bg-indigo-50"
                            : ""
                        }`}
                      >
                        <div>
                          <p className="text-sm font-medium text-slate-800">
                            {page.title}
                          </p>
                          <p className="text-xs text-slate-400">
                            {page.full_url}
                          </p>
                        </div>
                        <span className="text-[10px] font-medium text-slate-400 bg-slate-100 px-1.5 py-0.5 rounded">
                          {page.language_code.toUpperCase()}
                        </span>
                      </button>
                    ))
                  )}
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Code Injection */}
        <Card className="rounded-xl border border-slate-200 shadow-sm lg:col-span-2">
          <CardHeader className="pb-3">
            <div className="flex items-center gap-2">
              <FileText className="h-5 w-5 text-amber-500" />
              <div>
                <CardTitle className="text-base font-semibold text-slate-900">
                  Code Injection
                </CardTitle>
                <CardDescription className="text-xs text-slate-500">
                  Add custom code to your site's &lt;head&gt; section
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4 pt-0">
            <div className="grid gap-4 lg:grid-cols-2">
              <div className="space-y-1.5">
                <Label className="text-sm font-medium text-slate-700">
                  Analytics Code
                </Label>
                <Textarea
                  placeholder={"<!-- Google Analytics, Plausible, etc. -->\n<script async src=\"...\"></script>"}
                  value={settings.analytics_code}
                  onChange={(e) => update("analytics_code", e.target.value)}
                  rows={5}
                  className="rounded-lg border-slate-300 font-mono text-xs focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 resize-none"
                />
                <p className="text-[11px] text-slate-400">
                  Injected into &lt;head&gt; on every public page
                </p>
              </div>
              <div className="space-y-1.5">
                <Label className="text-sm font-medium text-slate-700">
                  Custom Head Code
                </Label>
                <Textarea
                  placeholder={"<!-- Custom meta tags, fonts, etc. -->\n<link rel=\"preconnect\" href=\"...\">"}
                  value={settings.custom_head_code}
                  onChange={(e) => update("custom_head_code", e.target.value)}
                  rows={5}
                  className="rounded-lg border-slate-300 font-mono text-xs focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 resize-none"
                />
                <p className="text-[11px] text-slate-400">
                  Injected into &lt;head&gt; on every public page
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
