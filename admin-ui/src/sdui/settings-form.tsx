import { useEffect, useState, useCallback } from "react";
import { Save, Loader2, RefreshCw } from "lucide-react";
import { useAdminLanguage } from "@/hooks/use-admin-language";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent } from "@/components/ui/card";
import { SectionHeader } from "@/components/ui/section-header";
import { SidebarCard } from "@/components/ui/sidebar-card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "sonner";
import {
  getSiteSettings,
  updateSiteSettings,
  clearCache,
  getNodes,
  type ContentNode,
} from "@/api/client";
import { iconMap } from "./sdui-components";

// SettingsForm renders a server-described settings page. The Go kernel emits
// a schema (sections + fields) and this component handles the load/save loop.
// The same component backs site settings, extension settings, theme settings —
// everything that's "load some settings, render a form, save the diff."

export interface SettingsFieldDef {
  key: string;
  label: string;
  type: "text" | "textarea" | "node_select";
  placeholder?: string;
  help?: string;
  rows?: number;
  font_mono?: boolean;
  // node_select-specific
  node_type?: string;
  empty_label?: string;
}

export interface SettingsSectionDef {
  title: string;
  icon?: string;
  description?: string;
  full_width?: boolean;
  fields: SettingsFieldDef[];
}

export interface SettingsFormProps {
  title?: string;
  description?: string;
  schema: SettingsSectionDef[];
  show_clear_cache?: boolean;
}

const ICON_COLORS: Record<string, string> = {
  Globe: "text-indigo-500",
  Home: "text-emerald-500",
  FileText: "text-amber-500",
  Code: "text-amber-500",
  Settings: "text-slate-500",
};

function renderIcon(name: string | undefined) {
  if (!name) return null;
  const Icon = iconMap[name];
  if (!Icon) return null;
  const color = ICON_COLORS[name] || "text-indigo-500";
  return <Icon className={`h-4 w-4 ${color}`} />;
}

export function SettingsForm({
  title = "Settings",
  description,
  schema,
  show_clear_cache = false,
}: SettingsFormProps) {
  const [values, setValues] = useState<Record<string, string>>({});
  const [original, setOriginal] = useState<Record<string, string>>({});
  const [pages, setPages] = useState<ContentNode[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [clearing, setClearing] = useState(false);
  const { languages, currentCode } = useAdminLanguage();
  // Per-form language override. Defaults to and follows the admin header
  // language, but the in-form selector below can pin a different value.
  // Currently a placeholder for site/extension settings — fields don't yet
  // carry a translatable flag at this layer, so changing it has no visible
  // effect until those forms opt into per-locale storage.
  const [pageLocale, setPageLocale] = useState<string>(currentCode);
  useEffect(() => {
    setPageLocale(currentCode);
  }, [currentCode]);

  const needsPages = schema.some((s) => s.fields.some((f) => f.type === "node_select"));

  const fetchAll = useCallback(async () => {
    setLoading(true);
    try {
      const promises: Promise<unknown>[] = [getSiteSettings()];
      if (needsPages) {
        promises.push(getNodes({ page: 1, per_page: 200, status: "published" }));
      }
      const [settings, pagesRes] = await Promise.all(promises) as [
        Record<string, string>,
        { data: ContentNode[] } | undefined,
      ];

      const initial: Record<string, string> = {};
      for (const section of schema) {
        for (const field of section.fields) {
          initial[field.key] = settings[field.key] ?? "";
        }
      }
      setValues(initial);
      setOriginal(initial);
      if (pagesRes) setPages(pagesRes.data);
    } catch {
      toast.error("Failed to load settings");
    } finally {
      setLoading(false);
    }
  }, [schema, needsPages]);

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  async function handleSave() {
    setSaving(true);
    try {
      const diff: Record<string, string> = {};
      for (const key of Object.keys(values)) {
        if (values[key] !== original[key]) diff[key] = values[key];
      }
      if (Object.keys(diff).length === 0) {
        toast.info("No changes to save");
        return;
      }
      await updateSiteSettings(diff);
      setOriginal({ ...values });
      toast.success("Settings saved");
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

  const hasChanges = JSON.stringify(values) !== JSON.stringify(original);

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        {/* Main content — section cards */}
        <div className="space-y-4 min-w-0">
          <div>
            <h1 className="text-2xl font-bold text-slate-900">{title}</h1>
            {description && (
              <p className="text-sm text-slate-500 mt-0.5">{description}</p>
            )}
          </div>

          {schema.map((section, idx) => (
            <Card
              key={idx}
              className="rounded-xl border border-slate-200 shadow-sm"
            >
              <SectionHeader title={section.title} icon={renderIcon(section.icon)} />
              <CardContent className="space-y-4">
                {section.description && (
                  <p className="text-xs text-slate-500 -mt-1">{section.description}</p>
                )}
                <div className="space-y-4">
                  {section.fields.map((field) => (
                    <SettingsField
                      key={field.key}
                      field={field}
                      value={values[field.key] || ""}
                      pages={pages}
                      onChange={(v) => setValues((prev) => ({ ...prev, [field.key]: v }))}
                    />
                  ))}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>

        {/* Sidebar — Publish-style card matching the node editor */}
        <aside className="space-y-4 lg:sticky lg:top-4 lg:self-start">
          <SidebarCard title="Publish">
            {languages.length > 0 && (
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-slate-500">
                  Language
                </Label>
                <Select value={pageLocale} onValueChange={setPageLocale}>
                  <SelectTrigger className="h-9 rounded-lg border-slate-300 text-sm">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All languages (shared)</SelectItem>
                    {languages.map((lang) => (
                      <SelectItem key={lang.code} value={lang.code}>
                        {lang.flag ? `${lang.flag} ` : ""}
                        {lang.name || lang.code}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-[11px] leading-snug text-slate-500">
                  Defaults to the admin header language. Site settings don't
                  yet support per-locale values — picking a language here is
                  a no-op until those fields opt in.
                </p>
              </div>
            )}

            <Button
              onClick={handleSave}
              disabled={saving || !hasChanges}
              className="w-full bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium"
            >
              <Save className="mr-2 h-4 w-4" />
              {saving ? "Saving..." : "Save Changes"}
            </Button>

            {show_clear_cache && (
              <Button
                variant="outline"
                onClick={handleClearCache}
                disabled={clearing}
                className="w-full rounded-lg font-medium"
              >
                <RefreshCw className={`mr-2 h-4 w-4 ${clearing ? "animate-spin" : ""}`} />
                {clearing ? "Clearing..." : "Clear Cache"}
              </Button>
            )}
          </SidebarCard>
        </aside>
      </div>
    </div>
  );
}

function SettingsField({
  field,
  value,
  pages,
  onChange,
}: {
  field: SettingsFieldDef;
  value: string;
  pages: ContentNode[];
  onChange: (v: string) => void;
}) {
  const inputClasses =
    "rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20";

  return (
    <div className="space-y-1.5">
      <Label className="text-sm font-medium text-slate-700">{field.label}</Label>
      {field.type === "text" && (
        <Input
          placeholder={field.placeholder}
          value={value}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => onChange(e.target.value)}
          className={inputClasses}
        />
      )}
      {field.type === "textarea" && (
        <Textarea
          placeholder={field.placeholder}
          value={value}
          onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => onChange(e.target.value)}
          rows={field.rows ?? 4}
          className={`${inputClasses} resize-none ${field.font_mono ? "font-mono text-xs" : ""}`}
        />
      )}
      {field.type === "node_select" && (
        <Select
          value={value || "__none__"}
          onValueChange={(v) => onChange(v === "__none__" ? "" : v)}
        >
          <SelectTrigger className={inputClasses}>
            <SelectValue placeholder={field.placeholder ?? "Select..."} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__none__">{field.empty_label ?? "None"}</SelectItem>
            {pages.map((p) => (
              <SelectItem key={p.id} value={String(p.id)}>
                {p.title} ({p.full_url}) [{p.language_code.toUpperCase()}]
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
      {field.help && <p className="text-[11px] text-slate-400">{field.help}</p>}
    </div>
  );
}
