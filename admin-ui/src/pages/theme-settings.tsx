import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { Save, Loader2, AlertCircle, Palette, Globe } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
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
import CustomFieldInput from "@/components/ui/custom-field-input";
import { toast } from "sonner";
import {
  getThemeSettingsPage,
  saveThemeSettingsPage,
  type NodeTypeField,
  type ThemeSettingsField,
  type ThemeSettingsPageResponse,
} from "@/api/client";
import { SduiAdminShell } from "@/sdui/admin-shell";
import { useAdminLanguage } from "@/hooks/use-admin-language";

// Adapt the theme-settings schema (key/label/type/default/config) to the
// NodeTypeField shape that CustomFieldInput already understands. Config keys
// (options, min, max, sub_fields, placeholder, ...) are spread directly into
// the field record so CustomFieldInput's switch on `type` picks them up.
function toNodeTypeField(f: ThemeSettingsField): NodeTypeField {
  const config = (f.config || {}) as Record<string, unknown>;
  return {
    name: f.key,
    key: f.key,
    label: f.label,
    type: f.type,
    default_value: f.default,
    ...config,
  } as NodeTypeField;
}

export function ThemeSettingsPage() {
  const { page: pageSlug } = useParams<{ page: string }>();
  // Header language is the default; per-page selector below can override it.
  const { languages, currentCode } = useAdminLanguage();
  const [pageLocale, setPageLocale] = useState<string>(currentCode);
  // When the header default changes (e.g. user picks a different language
  // globally) and they haven't pinned a per-page override yet, follow it.
  useEffect(() => {
    setPageLocale(currentCode);
  }, [currentCode]);

  const [data, setData] = useState<ThemeSettingsPageResponse | null>(null);
  const [values, setValues] = useState<Record<string, unknown>>({});
  const [original, setOriginal] = useState<Record<string, unknown>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!pageSlug) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    getThemeSettingsPage(pageSlug, pageLocale)
      .then((resp) => {
        if (cancelled) return;
        setData(resp);
        const initial: Record<string, unknown> = {};
        for (const f of resp.page.fields) {
          initial[f.key] = resp.values[f.key]?.value ?? null;
        }
        setValues(initial);
        setOriginal(initial);
      })
      .catch((e: Error) => {
        if (!cancelled) setError(e.message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [pageSlug, pageLocale]);

  const handleSave = async () => {
    if (!pageSlug || !data) return;
    setSaving(true);
    try {
      await saveThemeSettingsPage(pageSlug, values, pageLocale);
      toast.success("Theme settings saved");
      const fresh = await getThemeSettingsPage(pageSlug, pageLocale);
      setData(fresh);
      const refreshed: Record<string, unknown> = {};
      for (const f of fresh.page.fields) {
        refreshed[f.key] = fresh.values[f.key]?.value ?? null;
      }
      setValues(refreshed);
      setOriginal(refreshed);
    } catch (e) {
      toast.error(`Save failed: ${(e as Error).message}`);
    } finally {
      setSaving(false);
    }
  };

  const adaptedFields = useMemo(
    () => (data ? data.page.fields.map(toNodeTypeField) : []),
    [data],
  );

  const hasChanges = JSON.stringify(values) !== JSON.stringify(original);
  const hasTranslatable = useMemo(
    () => (data ? data.page.fields.some((f) => f.translatable) : false),
    [data],
  );

  if (loading && !data) {
    return (
      <SduiAdminShell>
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
        </div>
      </SduiAdminShell>
    );
  }

  if (error || !data) {
    return (
      <SduiAdminShell>
        <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
          <p className="font-medium">Failed to load theme settings</p>
          {error && <p className="mt-1 text-red-600">{error}</p>}
        </div>
      </SduiAdminShell>
    );
  }

  return (
    <SduiAdminShell>
      <div className="space-y-4">
        <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
          {/* Main content — settings card */}
          <div className="space-y-4 min-w-0">
            <Card className="rounded-xl border border-slate-200 shadow-sm">
              <SectionHeader
                title={data.page.name}
                icon={<Palette className="h-4 w-4 text-indigo-500" />}
              />
              <CardContent className="space-y-4">
                {data.page.description && (
                  <p className="text-xs text-slate-500 -mt-1">
                    {data.page.description}
                  </p>
                )}
                {adaptedFields.map((field, idx) => {
                  const originalField = data.page.fields[idx];
                  const v = data.values[originalField.key];
                  const incompatible =
                    v && v.compatible === false && v.raw !== "";
                  return (
                    <div key={originalField.key} className="space-y-2">
                      <div className="flex items-center gap-2">
                        <Label htmlFor={`tf-${originalField.key}`}>
                          {originalField.label}
                        </Label>
                        {originalField.translatable && (
                          <span
                            className="inline-flex items-center gap-1 rounded bg-indigo-50 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-indigo-700"
                            title="This field stores a separate value per language."
                          >
                            <Globe className="h-2.5 w-2.5" />
                            translatable
                          </span>
                        )}
                      </div>
                      <CustomFieldInput
                        field={field}
                        value={values[originalField.key]}
                        onChange={(val) =>
                          setValues((prev) => ({
                            ...prev,
                            [originalField.key]: val,
                          }))
                        }
                      />
                      {incompatible && (
                        <div className="flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 p-2 text-xs text-amber-900">
                          <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                          <div>
                            Previous value was incompatible with the new field
                            type and will be replaced when you save:&nbsp;
                            <code className="rounded bg-amber-100 px-1 py-0.5 font-mono">
                              {v.raw}
                            </code>
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}
              </CardContent>
            </Card>
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
                    {hasTranslatable
                      ? "Translatable fields store a separate value per language. Defaults to the admin header language."
                      : "No fields here are marked translatable, so the language pick is a no-op."}
                  </p>
                </div>
              )}

              <Button
                onClick={handleSave}
                disabled={saving || !hasChanges}
                className="w-full bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium"
              >
                {saving ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Save className="mr-2 h-4 w-4" />
                )}
                {saving ? "Saving..." : "Save Changes"}
              </Button>
            </SidebarCard>
          </aside>
        </div>
      </div>
    </SduiAdminShell>
  );
}

export default ThemeSettingsPage;
