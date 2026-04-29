import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { Save, Loader2, AlertCircle, Palette } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Card, CardContent } from "@/components/ui/card";
import { SectionHeader } from "@/components/ui/section-header";
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
  // Re-fetch when the admin's selected language changes — translatable
  // fields resolve per-locale on the backend, so the form must reload to
  // show the right values for the current language.
  const { currentCode } = useAdminLanguage();
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
    getThemeSettingsPage(pageSlug)
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
  }, [pageSlug, currentCode]);

  const handleSave = async () => {
    if (!pageSlug || !data) return;
    setSaving(true);
    try {
      await saveThemeSettingsPage(pageSlug, values);
      toast.success("Theme settings saved");
      const fresh = await getThemeSettingsPage(pageSlug);
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

  if (loading) {
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
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-slate-900">
              {data.page.name}
            </h1>
            {data.page.description && (
              <p className="text-sm text-slate-500 mt-0.5">
                {data.page.description}
              </p>
            )}
          </div>
          <Button
            onClick={handleSave}
            disabled={saving || !hasChanges}
            className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium"
          >
            {saving ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Save className="mr-2 h-4 w-4" />
            )}
            {saving ? "Saving..." : "Save Changes"}
          </Button>
        </div>

        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <SectionHeader
            title={data.page.name}
            icon={<Palette className="h-4 w-4 text-indigo-500" />}
          />
          <CardContent className="space-y-4">
            {adaptedFields.map((field, idx) => {
              const originalField = data.page.fields[idx];
              const v = data.values[originalField.key];
              const incompatible =
                v && v.compatible === false && v.raw !== "";
              return (
                <div key={originalField.key} className="space-y-2">
                  <Label htmlFor={`tf-${originalField.key}`}>
                    {originalField.label}
                  </Label>
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
    </SduiAdminShell>
  );
}

export default ThemeSettingsPage;
