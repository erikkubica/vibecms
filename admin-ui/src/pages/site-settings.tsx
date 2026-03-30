import { useEffect, useState, useCallback } from "react";
import { Save, Loader2, RefreshCw, Globe, Home, FileText } from "lucide-react";
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
import { Separator } from "@/components/ui/separator";
import { toast } from "sonner";
import { getSiteSettings, updateSiteSettings, clearCache } from "@/api/client";

interface SettingsState {
  site_name: string;
  site_url: string;
  site_description: string;
  homepage_node_id: string;
  default_language: string;
  analytics_code: string;
  custom_head_code: string;
}

const DEFAULT_SETTINGS: SettingsState = {
  site_name: "",
  site_url: "",
  site_description: "",
  homepage_node_id: "",
  default_language: "en",
  analytics_code: "",
  custom_head_code: "",
};

export default function SiteSettingsPage() {
  const [settings, setSettings] = useState<SettingsState>(DEFAULT_SETTINGS);
  const [original, setOriginal] = useState<SettingsState>(DEFAULT_SETTINGS);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [clearing, setClearing] = useState(false);

  const fetchSettings = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getSiteSettings();
      const loaded: SettingsState = {
        site_name: data.site_name || "",
        site_url: data.site_url || "",
        site_description: data.site_description || "",
        homepage_node_id: data.homepage_node_id || "",
        default_language: data.default_language || "en",
        analytics_code: data.analytics_code || "",
        custom_head_code: data.custom_head_code || "",
      };
      setSettings(loaded);
      setOriginal(loaded);
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
      // Only send changed values
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
          <CardHeader className="pb-3">
            <div className="flex items-center gap-2">
              <Globe className="h-5 w-5 text-indigo-500" />
              <div>
                <CardTitle className="text-base font-semibold text-slate-900">
                  General
                </CardTitle>
                <CardDescription className="text-xs text-slate-500">
                  Basic site identity and configuration
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

        {/* Content */}
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <CardHeader className="pb-3">
            <div className="flex items-center gap-2">
              <Home className="h-5 w-5 text-emerald-500" />
              <div>
                <CardTitle className="text-base font-semibold text-slate-900">
                  Content
                </CardTitle>
                <CardDescription className="text-xs text-slate-500">
                  Homepage and language defaults
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4 pt-0">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-slate-700">
                Homepage Node ID
              </Label>
              <Input
                type="number"
                placeholder="e.g. 6"
                value={settings.homepage_node_id}
                onChange={(e) => update("homepage_node_id", e.target.value)}
                className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
              />
              <p className="text-[11px] text-slate-400">
                The content node to use as your site's homepage
              </p>
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-slate-700">
                Default Language
              </Label>
              <Input
                placeholder="en"
                value={settings.default_language}
                onChange={(e) => update("default_language", e.target.value)}
                className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
              />
              <p className="text-[11px] text-slate-400">
                Language code used as fallback (e.g. en, de, fr)
              </p>
            </div>
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
                  Add custom code to your site's head section
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
                  placeholder="<!-- Google Analytics, Plausible, etc. -->"
                  value={settings.analytics_code}
                  onChange={(e) => update("analytics_code", e.target.value)}
                  rows={4}
                  className="rounded-lg border-slate-300 font-mono text-xs focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 resize-none"
                />
              </div>
              <div className="space-y-1.5">
                <Label className="text-sm font-medium text-slate-700">
                  Custom Head Code
                </Label>
                <Textarea
                  placeholder="<meta>, <link>, <script> tags..."
                  value={settings.custom_head_code}
                  onChange={(e) => update("custom_head_code", e.target.value)}
                  rows={4}
                  className="rounded-lg border-slate-300 font-mono text-xs focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 resize-none"
                />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
