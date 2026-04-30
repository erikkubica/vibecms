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
import { LanguageSelect } from "@/components/ui/language-select";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "sonner";
import {
  clearCache,
  getNodes,
  getRoles,
  getSettingsSchema,
  saveSettingsSchema,
  type ContentNode,
  type Role,
  type SettingsSchema,
  type SettingsSchemaField,
} from "@/api/client";
import { iconMap } from "./sdui-components";

// SchemaSettings is the schema-driven counterpart of SettingsForm. The
// React component is purely a renderer — the schema (sections, fields,
// per-field translatable flag) comes from the server. One mixed page can
// have translatable fields (per-language) alongside global fields
// (language_code='') with the framework routing storage transparently.
//
// SDUI emits { type: "SchemaSettings", props: { schema_id, show_clear_cache } }.
// The component fetches the schema + current values for the admin's
// locale and posts diffs back to /admin/api/settings/schemas/<id>.

const ICON_COLORS: Record<string, string> = {
  Globe: "text-indigo-500",
  Home: "text-emerald-500",
  FileText: "text-amber-500",
  Code: "text-amber-500",
  Settings: "text-slate-500",
  Shield: "text-rose-500",
};

function renderIcon(name: string | undefined) {
  if (!name) return null;
  const Icon = iconMap[name];
  if (!Icon) return null;
  const color = ICON_COLORS[name] || "text-indigo-500";
  return <Icon className={`h-4 w-4 ${color}`} />;
}

export interface SchemaSettingsProps {
  schema_id: string;
  show_clear_cache?: boolean;
}

export function SchemaSettings({
  schema_id,
  show_clear_cache = false,
}: SchemaSettingsProps) {
  const { languages, currentCode } = useAdminLanguage();
  const [schema, setSchema] = useState<SettingsSchema | null>(null);
  const [values, setValues] = useState<Record<string, string>>({});
  const [original, setOriginal] = useState<Record<string, string>>({});
  const [pages, setPages] = useState<ContentNode[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [clearing, setClearing] = useState(false);
  // pageLocale only matters when at least one field is translatable.
  // For all-global schemas the picker is hidden and saves omit the
  // locale header (server falls back to '' for non-translatable rows
  // regardless).
  const [pageLocale, setPageLocale] = useState<string>(currentCode);
  useEffect(() => {
    setPageLocale(currentCode);
  }, [currentCode]);

  const fetchAll = useCallback(async () => {
    setLoading(true);
    try {
      const env = await getSettingsSchema(schema_id, pageLocale);
      const needsPages = env.schema.sections.some((s) =>
        s.fields.some((f) => f.type === "node_select"),
      );
      const needsRoles = env.schema.sections.some((s) =>
        s.fields.some((f) => f.type === "role_select"),
      );
      const sidePromises: Promise<unknown>[] = [];
      if (needsPages) {
        sidePromises.push(getNodes({ page: 1, per_page: 200, status: "published" }));
      } else {
        sidePromises.push(Promise.resolve(undefined));
      }
      if (needsRoles) sidePromises.push(getRoles());
      else sidePromises.push(Promise.resolve(undefined));

      const [pagesRes, rolesRes] = (await Promise.all(sidePromises)) as [
        { data: ContentNode[] } | undefined,
        Role[] | undefined,
      ];

      // Initialise every key the schema declares so onChange handlers
      // never produce undefined → controlled-input warnings.
      const initial: Record<string, string> = {};
      for (const sec of env.schema.sections) {
        for (const f of sec.fields) {
          initial[f.key] = env.values[f.key] ?? f.default ?? "";
        }
      }
      setSchema(env.schema);
      setValues(initial);
      setOriginal(initial);
      if (pagesRes) setPages(pagesRes.data);
      if (rolesRes) setRoles(rolesRes);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to load settings";
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  }, [schema_id, pageLocale]);

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  const hasTranslatable = schema?.sections.some((s) =>
    s.fields.some((f) => f.translatable),
  ) ?? false;

  async function handleSave() {
    if (!schema) return;
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
      // Only forward the locale when the schema actually has
      // translatable fields. For all-global schemas the server doesn't
      // need it and forwarding it would tie an unrelated header to
      // the request.
      const localeArg = hasTranslatable ? pageLocale : undefined;
      await saveSettingsSchema(schema.id, diff, localeArg);
      setOriginal({ ...values });
      toast.success("Settings saved");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to save settings";
      toast.error(msg);
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

  if (loading || !schema) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
      </div>
    );
  }

  const hasChanges = JSON.stringify(values) !== JSON.stringify(original);

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">{schema.title}</h1>
        {schema.description && (
          <p className="text-sm text-slate-500 mt-0.5">{schema.description}</p>
        )}
      </div>

      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        <div className="space-y-4 min-w-0">
          {schema.sections.map((section, idx) => (
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
                    <SchemaField
                      key={field.key}
                      field={field}
                      value={values[field.key] ?? ""}
                      pages={pages}
                      roles={roles}
                      onChange={(v) =>
                        setValues((prev) => ({ ...prev, [field.key]: v }))
                      }
                    />
                  ))}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>

        <aside className="space-y-4 lg:sticky lg:top-4 lg:self-start">
          <SidebarCard title="Publish">
            {hasTranslatable && languages.length > 0 && (
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-slate-500">
                  Language
                </Label>
                <LanguageSelect
                  languages={languages}
                  value={pageLocale}
                  onChange={setPageLocale}
                />
                <p className="text-[11px] leading-snug text-slate-500">
                  Translatable fields store a separate value per language.
                  Fields marked “Global” apply to every language.
                </p>
              </div>
            )}
            {!hasTranslatable && (
              <p className="text-[11px] leading-snug text-slate-500">
                These settings apply to every language.
              </p>
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

function SchemaField({
  field,
  value,
  pages,
  roles,
  onChange,
}: {
  field: SettingsSchemaField;
  value: string;
  pages: ContentNode[];
  roles: Role[];
  onChange: (v: string) => void;
}) {
  const inputClasses =
    "rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20";

  // Always render translatability — operators repeatedly asked "did
  // it actually take effect?" and an absent badge isn't proof. On
  // all-global schemas the sidebar still says so up top; the badge
  // is redundant there but cheap, and it removes any "did the flag
  // wire through?" doubt.
  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-2">
        <Label className="text-sm font-medium text-slate-700">{field.label}</Label>
        {field.translatable ? (
          <span
            className="rounded-full bg-indigo-50 px-2 py-0.5 text-[10px] font-medium text-indigo-700 ring-1 ring-inset ring-indigo-200"
            title="This field stores a separate value per language"
          >
            Translatable
          </span>
        ) : (
          <span
            className="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-medium text-slate-600 ring-1 ring-inset ring-slate-200"
            title="This field applies to every language"
          >
            Global
          </span>
        )}
      </div>
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
      {field.type === "toggle" && (() => {
        const trueVal = field.true_value ?? "true";
        const falseVal = field.false_value ?? "false";
        const effective = value === "" ? (field.default ?? falseVal) : value;
        const checked = effective === trueVal;
        return (
          <div className="flex items-center gap-3 pt-1">
            <Switch
              checked={checked}
              onCheckedChange={(v: boolean) => onChange(v ? trueVal : falseVal)}
            />
            <span className="text-xs text-slate-500">
              {checked ? "On" : "Off"}
            </span>
          </div>
        );
      })()}
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
      {field.type === "role_select" && (
        <Select value={value || ""} onValueChange={onChange}>
          <SelectTrigger className={inputClasses}>
            <SelectValue placeholder={field.placeholder ?? "Select a role..."} />
          </SelectTrigger>
          <SelectContent>
            {roles.map((r) => (
              <SelectItem key={r.id} value={r.slug}>
                {r.name} ({r.slug})
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
      {field.type === "select" && (
        <Select value={value || ""} onValueChange={onChange}>
          <SelectTrigger className={inputClasses}>
            <SelectValue placeholder={field.placeholder ?? "Select..."} />
          </SelectTrigger>
          <SelectContent>
            {(field.options ?? []).map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
      {field.help && <p className="text-[11px] text-slate-400">{field.help}</p>}
      {field.warning && (
        <div className="flex gap-2 rounded-md border border-amber-200 bg-amber-50 px-2.5 py-1.5">
          <span className="text-amber-600 text-[11px] leading-tight" aria-hidden="true">⚠</span>
          <p className="text-[11px] leading-snug text-amber-800">{field.warning}</p>
        </div>
      )}
    </div>
  );
}
