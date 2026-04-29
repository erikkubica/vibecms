import { useEffect, useState, useCallback } from "react";
import { Save, Loader2, RefreshCw, Globe, Home, FileText, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent } from "@/components/ui/card";
import { SectionHeader } from "@/components/ui/section-header";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
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
  custom_footer_code: string;
  // SEO defaults — site-wide fallbacks emitted in the head when a node
  // doesn't override them. Keys map 1:1 to settings rows.
  seo_default_meta_title: string;
  seo_default_meta_description: string;
  seo_default_og_image: string;
  seo_og_site_name: string;
  seo_twitter_handle: string;
  seo_robots_index: string;
}

const DEFAULT_SETTINGS: SettingsState = {
  site_name: "",
  site_url: "",
  site_description: "",
  homepage_node_id: "",
  analytics_code: "",
  custom_head_code: "",
  custom_footer_code: "",
  seo_default_meta_title: "",
  seo_default_meta_description: "",
  seo_default_og_image: "",
  seo_og_site_name: "",
  seo_twitter_handle: "",
  seo_robots_index: "true",
};

export default function SiteSettingsPage() {
  const [settings, setSettings] = useState<SettingsState>(DEFAULT_SETTINGS);
  const [original, setOriginal] = useState<SettingsState>(DEFAULT_SETTINGS);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [clearing, setClearing] = useState(false);
  const [pages, setPages] = useState<ContentNode[]>([]);

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
        custom_footer_code: data.custom_footer_code || "",
        seo_default_meta_title: data.seo_default_meta_title || "",
        seo_default_meta_description: data.seo_default_meta_description || "",
        seo_default_og_image: data.seo_default_og_image || "",
        seo_og_site_name: data.seo_og_site_name || "",
        seo_twitter_handle: data.seo_twitter_handle || "",
        // Robots default = "true" (allow indexing) when unset, so a brand
        // new install doesn't accidentally noindex itself before any value
        // is saved.
        seo_robots_index: data.seo_robots_index ?? "true",
      };
      setSettings(loaded);
      setOriginal(loaded);
      setPages(pagesRes.data);
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
          <SectionHeader title="General" icon={<Globe className="h-4 w-4 text-indigo-500" />} />
          <CardContent className="space-y-4">
            <p className="text-xs text-slate-500 -mt-1">Basic site identity</p>
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
          <SectionHeader title="Homepage" icon={<Home className="h-4 w-4 text-emerald-500" />} />
          <CardContent className="space-y-3">
            <p className="text-xs text-slate-500 -mt-1">Choose which page visitors see first</p>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-slate-700">
                Homepage
              </Label>
              <Select
                value={settings.homepage_node_id || "none"}
                onValueChange={(v) =>
                  update("homepage_node_id", v === "none" ? "" : v)
                }
              >
                <SelectTrigger className="rounded-lg border-slate-300">
                  <SelectValue placeholder="Select a page..." />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">No homepage set</SelectItem>
                  {pages.map((page) => (
                    <SelectItem key={page.id} value={String(page.id)}>
                      {page.title} ({page.full_url}) [{page.language_code.toUpperCase()}]
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-[11px] text-slate-400">
                This page will be displayed when visitors access your site root
              </p>
            </div>
          </CardContent>
        </Card>

        {/* SEO defaults — emitted in the rendered <head> when a node
            doesn't override them. Per-node SEO (Meta Title / Description
            on the node edit screen) always wins. */}
        <Card className="rounded-xl border border-slate-200 shadow-sm lg:col-span-2">
          <SectionHeader title="SEO" icon={<Search className="h-4 w-4 text-sky-500" />} />
          <CardContent className="space-y-4">
            <p className="text-xs text-slate-500 -mt-1">
              Site-wide defaults. Per-node SEO settings always take precedence.
              Themes read these as <code className="text-[11px] font-mono">{`{{ index $s "seo_default_og_image" }}`}</code> etc.
            </p>

            <div className="grid gap-4 lg:grid-cols-2">
              <div className="space-y-1.5">
                <Label className="text-sm font-medium text-slate-700">Default Meta Title</Label>
                <Input
                  placeholder={settings.site_name || "Site title fallback"}
                  value={settings.seo_default_meta_title}
                  onChange={(e) => update("seo_default_meta_title", e.target.value)}
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
                <p className="text-[11px] text-slate-400">
                  Used when a page has no Meta Title. {settings.seo_default_meta_title.length || 0}/60.
                </p>
              </div>
              <div className="space-y-1.5">
                <Label className="text-sm font-medium text-slate-700">Default Meta Description</Label>
                <Textarea
                  placeholder={settings.site_description || "Brief site description"}
                  value={settings.seo_default_meta_description}
                  onChange={(e) => update("seo_default_meta_description", e.target.value)}
                  rows={2}
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 resize-none"
                />
                <p className="text-[11px] text-slate-400">
                  {settings.seo_default_meta_description.length || 0}/160 recommended.
                </p>
              </div>
            </div>

            <div className="grid gap-4 lg:grid-cols-2">
              <div className="space-y-1.5">
                <Label className="text-sm font-medium text-slate-700">Default OG Image</Label>
                <Input
                  placeholder="https://example.com/og.png"
                  value={settings.seo_default_og_image}
                  onChange={(e) => update("seo_default_og_image", e.target.value)}
                  className="rounded-lg border-slate-300 font-mono text-xs focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
                <p className="text-[11px] text-slate-400">
                  Fallback for og:image / twitter:image when a page has no featured image.
                  1200×630 recommended.
                </p>
              </div>
              <div className="space-y-1.5">
                <Label className="text-sm font-medium text-slate-700">OG Site Name</Label>
                <Input
                  placeholder={settings.site_name}
                  value={settings.seo_og_site_name}
                  onChange={(e) => update("seo_og_site_name", e.target.value)}
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
                <p className="text-[11px] text-slate-400">
                  Emitted as og:site_name. Defaults to Site Name when blank.
                </p>
              </div>
            </div>

            <div className="grid gap-4 lg:grid-cols-2">
              <div className="space-y-1.5">
                <Label className="text-sm font-medium text-slate-700">Twitter Handle</Label>
                <Input
                  placeholder="@yoursite"
                  value={settings.seo_twitter_handle}
                  onChange={(e) => update("seo_twitter_handle", e.target.value)}
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
                <p className="text-[11px] text-slate-400">Emitted as twitter:site for cards.</p>
              </div>
              <div className="space-y-1.5 flex flex-col justify-between">
                <div>
                  <Label className="text-sm font-medium text-slate-700">Search Engines</Label>
                  <p className="text-[11px] text-slate-400">
                    When off, every page emits <code className="font-mono">noindex,nofollow</code>.
                    Use during staging or to take a site offline from search.
                  </p>
                </div>
                <div className="flex items-center gap-3 pt-2">
                  <Switch
                    id="seo_robots_index"
                    checked={settings.seo_robots_index === "true"}
                    onCheckedChange={(v: boolean) => update("seo_robots_index", v ? "true" : "false")}
                  />
                  <Label htmlFor="seo_robots_index" className="text-sm">
                    {settings.seo_robots_index === "true" ? "Indexing allowed" : "Site hidden from search"}
                  </Label>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Code Injection */}
        <Card className="rounded-xl border border-slate-200 shadow-sm lg:col-span-2">
          <SectionHeader title="Code Injection" icon={<FileText className="h-4 w-4 text-amber-500" />} />
          <CardContent className="space-y-4">
            <p className="text-xs text-slate-500 -mt-1">Add custom code to your site's &lt;head&gt; section</p>
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
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-slate-700">
                Footer Code
              </Label>
              <Textarea
                placeholder={"<!-- Chat widgets, tracking pixels, etc. -->\n<script src=\"...\"></script>"}
                value={settings.custom_footer_code}
                onChange={(e) => update("custom_footer_code", e.target.value)}
                rows={5}
                className="rounded-lg border-slate-300 font-mono text-xs focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 resize-none"
              />
              <p className="text-[11px] text-slate-400">
                Injected before &lt;/body&gt; on every public page
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
